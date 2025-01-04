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
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	challengeDuration = 2 * time.Minute
	requeueInterval   = 30 * time.Second
	warningThreshold  = 2 * time.Minute // Time to start warning about impending timeout
)

var log = logr.Log.WithName("ChallengeController")

// ChallengeReconciler reconciles a Challenge object
type ChallengeReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// KafkaClient is the Kafka producer client
	// 중요한 메세지를 Kafka를 통해 보낸다.
	KafkaClient *KafkaProducer
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

	challenge := hexactfproj.Challenge{}
	if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check deletion first
	if !challenge.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &challenge)
	}

	// Add finalizer if not present
	if !containsString(challenge.Finalizers) {
		return r.addFinalizer(ctx, &challenge)
	}

	// init
	if challenge.Status.StartedAt == nil {
		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, &challenge, err)
		}
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		challenge.Status.CurrentStatus = *hexactfproj.NewCurrentStatus()
		if err := r.Status().Update(ctx, &challenge); err != nil {
			log.Error(err, "Failed to initialize status")
			return r.handleError(ctx, &challenge, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	switch {
	case challenge.Status.CurrentStatus.IsNone():
		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		challenge.Status.CurrentStatus.SetCreating()
		if err := r.Status().Update(ctx, &challenge); err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		err := r.loadChallengeDefinition(ctx, &challenge)
		if err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		challenge.Status.CurrentStatus.SetRunning()
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		if err := r.Status().Update(ctx, &challenge); err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		err = r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Running")
		if err != nil {
			log.Error(err, "Failed to send status change message")
			return r.handleError(ctx, &challenge, err)
		}

		return ctrl.Result{Requeue: true}, nil
	case challenge.Status.CurrentStatus.IsRunning():
		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, &challenge, err)
		}

		// isOne이 false이면 일정 시간 내에만 작동
		if !challenge.Status.IsOne && time.Since(challenge.Status.StartedAt.Time) > challengeDuration {

			if err := r.Delete(ctx, &challenge); err != nil {
				return r.handleError(ctx, &challenge, err)
			}
			return ctrl.Result{Requeue: true}, nil
		}

		if !challenge.DeletionTimestamp.IsZero() {

			if err := r.Delete(ctx, &challenge); err != nil {
				return r.handleError(ctx, &challenge, err)
			}
			return ctrl.Result{Requeue: true}, nil
		}
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// handleError 에러 발생 시 로깅과 상태(CurrentStatus)를 업데이트한다.
func (r *ChallengeReconciler) handleError(ctx context.Context, challenge *hexactfproj.Challenge, err error) (ctrl.Result, error) {

	// 최신 상태 가져오기
	latest := &hexactfproj.Challenge{}
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      challenge.Name,
	}, latest); err != nil {
		return ctrl.Result{}, err
	}

	// 낙관적 동시성 제어
	latest.Status.CurrentStatus.SetError(err)
	patch := client.MergeFrom(latest.DeepCopy())
	if updateErr := r.Status().Patch(ctx, latest, patch); updateErr != nil {
		log.Error(updateErr, "failed to update Challenge status")
		return ctrl.Result{}, updateErr
	}

	err = r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Error")
	if err != nil {
		log.Error(err, "Failed to send status change message")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, err
}

func (r *ChallengeReconciler) handleDeletion(ctx context.Context, challenge *hexactfproj.Challenge) (ctrl.Result, error) {
	log.Info("Processing deletion", "challenge", challenge.Name)

	if containsString(challenge.Finalizers) {
		if err := r.removeFinalizer(ctx, challenge); err != nil {

			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}

		err := r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Deleted")
		if err != nil {
			log.Error(err, "Failed to send status change message")
			return ctrl.Result{}, err
		}

	}

	log.Info("Successfully completed deletion process")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChallengeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hexactfproj.Challenge{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("challenge").
		Complete(r)
}
