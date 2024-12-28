/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1alpha1 "github.com/hexactf/challenge-operator/api/v1alpha1"
)

const (
	finalizerName     = "finalizer.hexactf.io/challenge"
	challengeDuration = 2 * time.Minute
	requeueInterval   = 30 * time.Second
	warningThreshold  = 25 * time.Minute // Time to start warning about impending timeout
)

// ChallengeReconciler reconciles a Challenge object
type ChallengeReconciler struct {
	client.Client
	Kafka  *KafkaProducer
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Challenge object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile

func (r *ChallengeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	logger := log.FromContext(ctx)
	logger.Info("Starting reconciliation", "request", req)

	var challenge appsv1alpha1.Challenge
	if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	userLabel := fmt.Sprintf("%s-%s",
		challenge.Labels["hexactf.io/user"],
		challenge.Labels["hexactf.io/problemId"])

	// Check deletion first
	if !challenge.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &challenge, userLabel)
	}

	// Add finalizer if not present
	if !containsString(challenge.Finalizers, finalizerName) {
		return r.addFinalizer(ctx, &challenge)
	}

	template, err := r.loadChallengeTemplate(ctx, &challenge)
	if err != nil {
		return r.handleError(ctx, &challenge, err)
	}

	// Initialize if needed
	if challenge.Status.StartedAt == nil {
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		challenge.Status.CurrentStatus = *appsv1alpha1.NewCurrentStatus()
		if err := r.Status().Update(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to initialize status")
			return r.handleError(ctx, &challenge, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Handle state transitions
	switch {
	case challenge.Status.CurrentStatus.IsNone():
		if err := r.reconcileResources(ctx, &challenge, template, userLabel); err != nil {
			return r.handleError(ctx, &challenge, err)
		}
		return ctrl.Result{Requeue: true}, nil

	case challenge.Status.CurrentStatus.IsCreated():
		// Check if Pod is running
		var pod v1.Pod
		err := r.Get(ctx, client.ObjectKey{
			Namespace: challenge.Spec.Namespace,
			Name:      fmt.Sprintf("pod-%s", userLabel),
		}, &pod)
		if err == nil && pod.Status.Phase == v1.PodRunning {
			challenge.Status.CurrentStatus.SetRunning()
			if err := r.Status().Update(ctx, &challenge); err != nil {
				logger.Error(err, "Failed to update status to Running")
				return r.handleError(ctx, &challenge, err)
			}
			// Send Running message
			if err := r.Kafka.SendStatusChange(
				challenge.Labels["hexactf.io/user"],
				challenge.Labels["hexactf.io/problemId"],
				"Running",
			); err != nil {
				logger.Error(err, "Failed to send Running status")
			}
		}

		var service v1.Service
		err = r.Get(ctx, client.ObjectKey{
			Namespace: challenge.Spec.Namespace,
			Name:      fmt.Sprintf("svc-%s", userLabel),
		}, &service)
		if err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		// 서비스가 있고 NodePort가 할당되었다면
		if service.Spec.Type == v1.ServiceTypeNodePort {
			for _, port := range service.Spec.Ports {
				// NodePort가 할당된 것을 확인
				if port.NodePort != 0 {
					// Status에 NodePort 업데이트
					challenge.Status.AllocatedNodePort = port.NodePort
					if err := r.Status().Update(ctx, &challenge); err != nil {
						return ctrl.Result{}, err
					}
					// NodePort를 찾았으니 loop 종료
					break
				}
			}
		}

	}

	if !challenge.DeletionTimestamp.IsZero() || time.Since(challenge.Status.StartedAt.Time) > challengeDuration {
		if err := r.Delete(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to request deletion")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}

		// // Terminating 상태로 변경
		// challenge.Status.CurrentStatus.SetTerminating()
		// if err := r.Status().Update(ctx, &challenge); err != nil {
		//     logger.Error(err, "Failed to update status to Terminating")
		//     return ctrl.Result{RequeueAfter: time.Second * 5}, err
		// }

		return r.handleDeletion(ctx, &challenge, userLabel)

	}

	// Check for challenge timeout
	timeoutReached, err := r.checkTimeout(ctx, &challenge)
	if err != nil {
		logger.Error(err, "Failed to check timeout status")
		return r.handleError(ctx, &challenge, err)
	}

	if timeoutReached {
		logger.Info("Challenge timeout reached (some minutes), initiating cleanup",
			"challenge", challenge.Name,
			"elapsed", time.Since(challenge.Status.StartedAt.Time))

		if err := r.Kafka.SendStatusChange(
			challenge.Labels["hexactf.io/user"],
			challenge.Labels["hexactf.io/problemId"],
			"TimedOut",
		); err != nil {
			logger.Error(err, "Kafka TimeOut")
		}

		if err := r.Delete(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to delete timed-out challenge")
			return r.handleError(ctx, &challenge, err)
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *ChallengeReconciler) handleError(ctx context.Context, challenge *appsv1alpha1.Challenge, err error) (ctrl.Result, error) {
	if err := r.Kafka.SendStatusChange(
		challenge.Labels["hexactf.io/user"],
		challenge.Labels["hexactf.io/problemId"],
		"Error",
	); err != nil {
		log.FromContext(ctx).Error(err, "Failed to send status change to Kafka")
	}

	challenge.Status.CurrentStatus.SetError(err.Error())

	return ctrl.Result{}, err
}

// handleDeletion 관련 리소스 삭제 후 Finalizer 제거
func (r *ChallengeReconciler) handleDeletion(ctx context.Context, challenge *appsv1alpha1.Challenge, userLabel string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing deletion", "challenge", challenge.Name)

	if containsString(challenge.Finalizers, finalizerName) {
		if err := r.removeFinalizer(ctx, challenge); err != nil {
			logger.Error(err, "Failed to remove finalizer")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}

		if err := r.Kafka.SendStatusChange(
			challenge.Labels["hexactf.io/user"],
			challenge.Labels["hexactf.io/problemId"],
			"Deleted",
		); err != nil {
			logger.Error(err, "Failed to send status to Kafka")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}

	}

	logger.Info("Successfully completed deletion process")
	return ctrl.Result{}, nil
}

// removeFinalizer handles the finalizer removal with retries
func (r *ChallengeReconciler) removeFinalizer(ctx context.Context, challenge *appsv1alpha1.Challenge) error {
	retries := 3

	for i := 0; i < retries; i++ {
		// Get the latest version of the resource
		var latestChallenge appsv1alpha1.Challenge
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: challenge.Namespace,
			Name:      challenge.Name,
		}, &latestChallenge); err != nil {
			if apierrors.IsNotFound(err) {
				// Resource is already gone, nothing to do
				return nil
			}
			return err
		}

		// Remove the finalizer
		latestChallenge.Finalizers = removeString(latestChallenge.Finalizers, finalizerName)

		// Try to update
		if err := r.Update(ctx, &latestChallenge); err != nil {
			if apierrors.IsConflict(err) {
				// If there's a conflict, wait a bit and retry
				time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
				continue
			}
			return err
		}

		// Update was successful
		return nil
	}

	return fmt.Errorf("failed to remove finalizer after %d attempts", retries)
}

func (r *ChallengeReconciler) loadChallengeTemplate(ctx context.Context, challenge *appsv1alpha1.Challenge) (*appsv1alpha1.ChallengeTemplate, error) {
	var template appsv1alpha1.ChallengeTemplate
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      challenge.Spec.CTemplate,
	}, &template); err != nil {
		return nil, fmt.Errorf("failed to load template %s: %w", challenge.Spec.CTemplate, err)
	}
	return &template, nil
}

// reconcileResources Pod 서비스를 생성해준다.
// 생성이 모두 성공되면 Kafka로 생성 상태를 전송한다.
func (r *ChallengeReconciler) reconcileResources(ctx context.Context, challenge *appsv1alpha1.Challenge, template *appsv1alpha1.ChallengeTemplate, userLabel string) error {
	logger := log.FromContext(ctx)

	// Reconcile Pod
	if err := r.reconcilePod(ctx, challenge, template, userLabel); err != nil {
		return err
	}

	// Reconcile Service
	if err := r.reconcileService(ctx, challenge, template, userLabel); err != nil {
		return err
	}

	if challenge.Status.CurrentStatus.IsNone() {
		challenge.Status.CurrentStatus.SetCreated()
		if err := r.Status().Update(ctx, challenge); err != nil {
			logger.Error(err, "Failed to update status to Created")
			return err
		}
		if err := r.Kafka.SendStatusChange(
			challenge.Labels["hexactf.io/user"],
			challenge.Labels["hexactf.io/problemId"],
			"Created",
		); err != nil {
			logger.Error(err, "Failed to send Created status to Kafka")
		} else {
			logger.Info("Successfully sent Created status to Kafka",
				"user", challenge.Labels["hexactf.io/user"],
				"problemId", challenge.Labels["hexactf.io/problemId"])
		}
	}
	return nil
}

func (r *ChallengeReconciler) reconcilePod(ctx context.Context, challenge *appsv1alpha1.Challenge, template *appsv1alpha1.ChallengeTemplate, userLabel string) error {
	podName := fmt.Sprintf("pod-%s", userLabel)
	var existingPod v1.Pod

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Spec.Namespace,
		Name:      podName,
	}, &existingPod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to check existing pod: %w", err)
		}

		// Create new Pod
		pod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: challenge.Spec.Namespace,
				Labels: map[string]string{
					"hexactf.io/challenge": userLabel,
				},
			},
			Spec: v1.PodSpec{
				Containers: template.Spec.Resources.Pod.Containers,
			},
		}
		if err := controllerutil.SetControllerReference(challenge, &pod, r.Scheme); err != nil {
			return fmt.Errorf("failed to set owner reference: %w", err)
		}

		if err := r.Create(ctx, &pod); err != nil {
			return fmt.Errorf("failed to create pod: %w", err)
		}
	}

	return nil
}

func (r *ChallengeReconciler) reconcileService(ctx context.Context, challenge *appsv1alpha1.Challenge, template *appsv1alpha1.ChallengeTemplate, userLabel string) error {
	serviceName := fmt.Sprintf("svc-%s", userLabel)
	var existingService v1.Service

	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Spec.Namespace,
		Name:      serviceName,
	}, &existingService); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to check existing service: %w", err)
		}

		// Service 생성 전에 template의 Service 스펙 검증
		serviceSpec := template.Spec.Resources.Service
		if serviceSpec.Type == v1.ServiceTypeNodePort {
			// NodePort 타입이면 Ports 확인
			for i := range serviceSpec.Ports {
				// NodePort 필드는 제거하고 자동 할당되도록
				serviceSpec.Ports[i].NodePort = 0
			}
		}

		// Create new Service
		service := v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: challenge.Spec.Namespace,
				Labels: map[string]string{
					"hexactf.io/challenge": userLabel,
				},
			},
			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"hexactf.io/challenge": userLabel,
				},
				Type:  serviceSpec.Type,
				Ports: serviceSpec.Ports,
			},
		}

		// OwnerReference 설정
		if err := controllerutil.SetControllerReference(challenge, &service, r.Scheme); err != nil {
			return fmt.Errorf("failed to set owner reference: %w", err)
		}

		if err := r.Create(ctx, &service); err != nil {
			return fmt.Errorf("failed to create service: %w", err)
		}
	}

	return nil
}

func (r *ChallengeReconciler) checkTimeout(ctx context.Context, challenge *appsv1alpha1.Challenge) (bool, error) {
	if challenge.Status.StartedAt == nil {
		return false, nil
	}

	elapsed := time.Since(challenge.Status.StartedAt.Time)

	// Check if timeout is reached
	if elapsed > challengeDuration {
		return true, nil
	}

	// Update status with remaining time if near timeout
	//if elapsed > warningThreshold && challenge.Status.TimeoutWarningIssued != true {
	//	remainingTime := challengeDuration - elapsed
	//	challenge.Status.TimeoutWarning = fmt.Sprintf("Challenge will timeout in %.0f minutes", remainingTime.Minutes())
	//	challenge.Status.TimeoutWarningIssued = true
	//
	//	if err := r.Status().Update(ctx, challenge); err != nil {
	//		return false, fmt.Errorf("failed to update timeout warning status: %w", err)
	//	}
	//}

	return false, nil
}

func (r *ChallengeReconciler) addFinalizer(ctx context.Context, challenge *appsv1alpha1.Challenge) (ctrl.Result, error) {
	challenge.Finalizers = append(challenge.Finalizers, finalizerName)
	if err := r.Update(ctx, challenge); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChallengeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.Challenge{}).
		Owns(&v1.Service{}).
		Owns(&v1.Pod{}). // Challenge가 소유한 Pod 감시
		Named("challenge").
		Complete(r)
}
