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

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	challengeDuration = 5 * time.Minute
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
	// KafkaClient *KafkaProducer
}

// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

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
	challenge := &hexactfproj.Challenge{}
	if err := r.Get(ctx, req.NamespacedName, challenge); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Challenge", "challenge", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !challenge.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, challenge)
	}

	// Add finalizer if not present
	if !containsString(challenge.Finalizers) {
		return r.addFinalizer(ctx, challenge)
	}

	// Initialize challenge if not started
	if challenge.Status.StartedAt == nil {
		if err := r.initializeChallenge(ctx, challenge); err != nil {
			return r.handleError(ctx, req, challenge, err)
		}
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// Handle different states
	switch {
	case challenge.Status.CurrentStatus.IsPending():
		return r.handlePendingState(ctx, challenge)
	case challenge.Status.CurrentStatus.IsRunning():
		return r.handleRunningState(ctx, challenge)
	default:
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChallengeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hexactfproj.Challenge{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}), builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			// Only process pods that have our owner reference
			ownerRefs := obj.GetOwnerReferences()
			for _, ref := range ownerRefs {
				if ref.Kind == "Challenge" && ref.APIVersion == "apps.hexactf.io/v2alpha1" {
					return true
				}
			}
			return false
		}))).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{}), builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			// Only process services that have our owner reference
			ownerRefs := obj.GetOwnerReferences()
			for _, ref := range ownerRefs {
				if ref.Kind == "Challenge" && ref.APIVersion == "apps.hexactf.io/v2alpha1" {
					return true
				}
			}
			return false
		}))).
		Named("challenge").
		Complete(r)
}
