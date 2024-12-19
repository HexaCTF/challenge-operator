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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	appsv1alpha1 "github.com/hexactf/challenge-operator/api/v1alpha1"
)

const (
	finalizerName     = "finalizer.hexactf.io/challenge"
	challengeDuration = 2 * time.Minute
	requeueInterval   = 1 * time.Minute
	warningThreshold  = 25 * time.Minute // Time to start warning about impending timeout
)

// ChallengeReconciler reconciles a Challenge object
type ChallengeReconciler struct {
	client.Client
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

	// Fetch the Challenge resource
	var challenge appsv1alpha1.Challenge
	if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Build the unique label for resources
	userLabel := fmt.Sprintf("%s-%s",
		challenge.Labels["hexactf.io/user"],
		challenge.Labels["hexactf.io/problemId"])

	// Handle deletion
	if !challenge.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &challenge, userLabel)
	}

	// Add finalizer if not present
	if !containsString(challenge.Finalizers, finalizerName) {
		return r.addFinalizer(ctx, &challenge)
	}

	// Load associated ChallengeTemplate
	template, err := r.loadChallengeTemplate(ctx, &challenge)
	if err != nil {
		return ctrl.Result{}, err
	}

	// Initialize challenge if needed
	if err := r.initializeChallenge(ctx, &challenge); err != nil {
		return ctrl.Result{}, err
	}

	// Create or verify Pod
	if err := r.reconcilePod(ctx, &challenge, template, userLabel); err != nil {
		return ctrl.Result{}, err
	}

	// Create or verify Service
	if err := r.reconcileService(ctx, &challenge, template, userLabel); err != nil {
		return ctrl.Result{}, err
	}

	// Check for challenge timeout
	timeoutReached, err := r.checkTimeout(ctx, &challenge)
	if err != nil {
		logger.Error(err, "Failed to check timeout status")
		return ctrl.Result{}, err
	}

	if timeoutReached {
		logger.Info("Challenge timeout reached (some minutes), initiating cleanup",
			"challenge", challenge.Name,
			"elapsed", time.Since(challenge.Status.StartedAt.Time))

		if err := r.Delete(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to delete timed-out challenge")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// handleDeletion 관련 리소스 삭제 후 Finalizer 제거
func (r *ChallengeReconciler) handleDeletion(ctx context.Context, challenge *appsv1alpha1.Challenge, userLabel string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Processing deletion", "challenge", challenge.Name)

	// Prepare labels for resource selection
	labels := map[string]string{
		"hexactf.io/challenge": userLabel,
		"hexactf.io/user":      challenge.Labels["hexactf.io/user"],
		"hexactf.io/problemId": challenge.Labels["hexactf.io/problemId"],
	}

	// Delete Pods with matching labels
	if err := r.DeleteAllOf(ctx, &v1.Pod{},
		client.InNamespace(challenge.Spec.Namespace),
		client.MatchingLabels(labels)); err != nil {
		logger.Error(err, "Failed to delete Pods", "labels", labels)
		return ctrl.Result{}, fmt.Errorf("failed to delete pods: %w", err)
	}

	// Delete Services with matching labels
	if err := r.DeleteAllOf(ctx, &v1.Service{},
		client.InNamespace(challenge.Spec.Namespace),
		client.MatchingLabels(labels)); err != nil {
		logger.Error(err, "Failed to delete Services", "labels", labels)
		return ctrl.Result{}, fmt.Errorf("failed to delete services: %w", err)
	}

	// Remove finalizer
	challenge.Finalizers = removeString(challenge.Finalizers, finalizerName)
	if err := r.Update(ctx, challenge); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
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

func (r *ChallengeReconciler) initializeChallenge(ctx context.Context, challenge *appsv1alpha1.Challenge) error {
	if challenge.Status.StartedAt == nil {
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		return r.Status().Update(ctx, challenge)
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
				Type:  v1.ServiceType(template.Spec.Resources.Service.Type),
				Ports: r.buildServicePorts(template),
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

func (r *ChallengeReconciler) buildServicePorts(template *appsv1alpha1.ChallengeTemplate) []v1.ServicePort {
	ports := make([]v1.ServicePort, 0, len(template.Spec.Resources.Service.Ports))
	for _, p := range template.Spec.Resources.Service.Ports {
		ports = append(ports, v1.ServicePort{
			Port:       p.Port,
			TargetPort: intstr.FromInt(int(p.TargetPort)),
			NodePort:   p.NodePort,
		})
	}
	return ports
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
		Owns(&v1.Pod{}). // Challenge가 소유한 Pod 감시
		Owns(&v1.Service{}).
		Named("challenge").
		Complete(r)
}
