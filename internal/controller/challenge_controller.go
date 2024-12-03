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
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	appsv1alpha1 "github.com/hexactf/challenge-operator/api/v1alpha1"
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

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

func (r *ChallengeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Challenge resource
	var challenge appsv1alpha1.Challenge
	if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Fetch the associated ChallengeTemplate
	var template appsv1alpha1.ChallengeTemplate
	if err := r.Get(ctx, client.ObjectKey{Namespace: challenge.Namespace, Name: challenge.Spec.CTemplate}, &template); err != nil {
		logger.Error(err, "Failed to load ChallengeTemplate", "template", challenge.Spec.CTemplate)
		return ctrl.Result{}, err
	}

	// Add a finalizer if not present
	finalizerName := "finalizer.hexactf.io/challenge"
	if !containsString(challenge.Finalizers, finalizerName) {
		challenge.Finalizers = append(challenge.Finalizers, finalizerName)
		if err := r.Update(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Handle deletion logic
	if !challenge.DeletionTimestamp.IsZero() {
		logger.Info("Challenge is being deleted, cleaning up resources", "name", challenge.Name)

		// Define the unique label for resources
		userLabel := fmt.Sprintf("%s-%s", challenge.Labels["hexactf.io/user"], challenge.Labels["hexactf.io/problemId"])

		// Delete Pods
		if err := r.DeleteAllOf(ctx, &v1.Pod{}, client.InNamespace(challenge.Spec.Namespace), client.MatchingLabels{"hexactf.io/challenge": userLabel}); err != nil {
			logger.Error(err, "Failed to delete Pods")
		}

		// Delete Services
		if err := r.DeleteAllOf(ctx, &v1.Service{}, client.InNamespace(challenge.Spec.Namespace), client.MatchingLabels{"hexactf.io/challenge": userLabel}); err != nil {
			logger.Error(err, "Failed to delete Services")
		}

		// Remove the finalizer
		challenge.Finalizers = removeString(challenge.Finalizers, finalizerName)
		if err := r.Update(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Set StartedAt if not already set
	if challenge.Status.StartedAt == nil {
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		if err := r.Status().Update(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to update Challenge status with StartedAt")
			return ctrl.Result{}, err
		}
	}

	// Define the unique label for resources
	userLabel := fmt.Sprintf("%s-%s", challenge.Labels["hexactf.io/user"], challenge.Labels["hexactf.io/problemId"])

	// Create Pod if it does not exist
	podName := fmt.Sprintf("pod-%s", userLabel)
	var existingPod v1.Pod
	if err := r.Get(ctx, client.ObjectKey{Namespace: challenge.Spec.Namespace, Name: podName}, &existingPod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get existing Pod")
			return ctrl.Result{}, err
		}

		// Create Pod
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
		if err := r.Create(ctx, &pod); err != nil {
			logger.Error(err, "Failed to create Pod")
			return ctrl.Result{}, err
		}
	}

	// Create Service if it does not exist
	serviceName := fmt.Sprintf("svc-%s", userLabel)
	var existingService v1.Service
	if err := r.Get(ctx, client.ObjectKey{Namespace: challenge.Spec.Namespace, Name: serviceName}, &existingService); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "Failed to get existing Service")
			return ctrl.Result{}, err
		}

		// Create Service
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
				Type: v1.ServiceType(template.Spec.Resources.Service.Type),
				Ports: func() []v1.ServicePort {
					ports := []v1.ServicePort{}
					for _, p := range template.Spec.Resources.Service.Ports {
						ports = append(ports, v1.ServicePort{
							Port:       p.Port,
							TargetPort: intstr.FromInt(int(p.TargetPort)),
							NodePort:   p.NodePort,
						})
					}
					return ports
				}(),
			},
		}
		if err := r.Create(ctx, &service); err != nil {
			logger.Error(err, "Failed to create Service")
			return ctrl.Result{}, err
		}
	}

	// Check elapsed time for deletion
	elapsed := time.Since(challenge.Status.StartedAt.Time)
	if elapsed > 2*time.Minute {
		logger.Info("2 minutes elapsed, cleaning up resources", "challenge", challenge.Name)

		// Trigger deletion by setting DeletionTimestamp
		if err := r.Delete(ctx, &challenge); err != nil {
			logger.Error(err, "Failed to delete Challenge")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChallengeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.Challenge{}).
		Named("challenge").
		Complete(r)
}
