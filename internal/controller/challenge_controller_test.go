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

	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1alpha1 "github.com/hexactf/challenge-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ChallengeReconciler", func() {
	var (
		reconciler *ChallengeReconciler
		ctx        context.Context
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		// Create a scheme and add necessary types
		scheme = runtime.NewScheme()
		Expect(appsv1alpha1.AddToScheme(scheme)).Should(Succeed())
		Expect(v1.AddToScheme(scheme)).Should(Succeed())

		// Create a fake client
		client := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		// Create the reconciler
		reconciler = &ChallengeReconciler{
			Client: client,
			Scheme: scheme,
		}

		ctx = context.Background()
	})

	Describe("HandleError Method", func() {
		It("should return the original error", func() {
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
				},
			}

			testError := fmt.Errorf("test reconciliation error")

			result, err := reconciler.handleError(ctx, challenge, testError)

			// Verify error is returned
			Expect(err).To(Equal(testError))

			// Verify result is empty
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())
		})
	})

	Describe("HandleDeletion Method", func() {
		It("should remove finalizer", func() {
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-challenge",
					Namespace:  "default",
					Finalizers: []string{finalizerName},
				},
			}

			// Add the challenge to the fake client
			Expect(reconciler.Client.Create(ctx, challenge)).Should(Succeed())

			// Call handleDeletion
			result, err := reconciler.handleDeletion(ctx, challenge, "test-label")

			// Verify no error occurred
			Expect(err).To(BeNil())

			// Verify result is empty
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())

			// Retrieve the updated challenge
			updatedChallenge := &appsv1alpha1.Challenge{}
			err = reconciler.Client.Get(ctx, types.NamespacedName{
				Name:      challenge.Name,
				Namespace: challenge.Namespace,
			}, updatedChallenge)

			Expect(err).To(BeNil())

			// Verify finalizer is removed
			Expect(updatedChallenge.Finalizers).To(BeEmpty())
		})
	})
	Describe("ReconcilePod Method", func() {
		It("should create a pod when it does not exist", func() {
			// Create a challenge
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
				},
				Spec: appsv1alpha1.ChallengeSpec{
					Namespace: "default",
				},
			}

			// Create a template with pod resources
			template := &appsv1alpha1.ChallengeTemplate{
				Spec: appsv1alpha1.ChallengeTemplateSpec{
					Resources: appsv1alpha1.ChallengeTemplateResources{
						Pod: appsv1alpha1.PodSpecTemplate{
							Containers: []v1.Container{
								{
									Name:  "test-container",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			// Reconcile pod
			err := reconciler.reconcilePod(ctx, challenge, template, "test-user-problem")

			// Verify no error occurred
			Expect(err).To(BeNil())

			// Verify pod was created
			pod := &v1.Pod{}
			err = reconciler.Client.Get(ctx, types.NamespacedName{
				Name:      "pod-test-user-problem",
				Namespace: "default",
			}, pod)

			Expect(err).To(BeNil())
			Expect(pod.Spec.Containers).To(HaveLen(1))
			Expect(pod.Spec.Containers[0].Name).To(Equal("test-container"))
			Expect(pod.Spec.Containers[0].Image).To(Equal("nginx:latest"))
		})

		It("should not create pod if it already exists", func() {
			// Create an existing pod
			existingPod := &v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-test-user-problem",
					Namespace: "default",
				},
			}
			Expect(reconciler.Client.Create(ctx, existingPod)).Should(Succeed())

			// Create a challenge
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
				},
				Spec: appsv1alpha1.ChallengeSpec{
					Namespace: "default",
				},
			}

			// Create a template with pod resources
			template := &appsv1alpha1.ChallengeTemplate{
				Spec: appsv1alpha1.ChallengeTemplateSpec{
					Resources: appsv1alpha1.ChallengeTemplateResources{
						Pod: appsv1alpha1.PodSpecTemplate{
							Containers: []v1.Container{
								{
									Name:  "test-container",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			// Reconcile pod
			err := reconciler.reconcilePod(ctx, challenge, template, "test-user-problem")

			// Verify no error occurred
			Expect(err).To(BeNil())

			// Verify only one pod exists
			podList := &v1.PodList{}
			err = reconciler.Client.List(ctx, podList,
				client.InNamespace("default"),
				client.MatchingLabels{"hexactf.io/challenge": "test-user-problem"})

			Expect(err).To(BeNil())
			Expect(podList.Items).To(HaveLen(1))
		})
	})

	Describe("ReconcileService Method", func() {
		It("should create a service when it does not exist", func() {
			// Create a challenge
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
				},
				Spec: appsv1alpha1.ChallengeSpec{
					Namespace: "default",
				},
			}

			// Create a template with service resources
			template := &appsv1alpha1.ChallengeTemplate{
				Spec: appsv1alpha1.ChallengeTemplateSpec{
					Resources: appsv1alpha1.ChallengeTemplateResources{
						Service: appsv1alpha1.ServiceSpecTemplate{
							Type: "NodePort",
							Ports: []v1.ServicePort{
								{
									Port:       80,
									TargetPort: 8080,
									NodePort:   30080,
								},
							},
						},
					},
				},
			}

			// Reconcile service
			err := reconciler.reconcileService(ctx, challenge, template, "test-user-problem")

			// Verify no error occurred
			Expect(err).To(BeNil())

			// Verify service was created
			service := &v1.Service{}
			err = reconciler.Client.Get(ctx, types.NamespacedName{
				Name:      "svc-test-user-problem",
				Namespace: "default",
			}, service)

			Expect(err).To(BeNil())
			Expect(service.Spec.Type).To(Equal(v1.ServiceTypeNodePort))
			Expect(service.Spec.Ports).To(HaveLen(1))
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(80)))
			Expect(service.Spec.Ports[0].TargetPort).To(Equal(intstr.FromInt(8080)))
			Expect(service.Spec.Ports[0].NodePort).To(Equal(int32(30080)))
		})

		It("should not create service if it already exists", func() {
			// Create an existing service
			existingService := &v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "svc-test-user-problem",
					Namespace: "default",
				},
			}
			Expect(reconciler.Client.Create(ctx, existingService)).Should(Succeed())

			// Create a challenge
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
				},
				Spec: appsv1alpha1.ChallengeSpec{
					Namespace: "default",
				},
			}

			// Create a template with service resources
			template := &appsv1alpha1.ChallengeTemplate{
				Spec: appsv1alpha1.ChallengeTemplateSpec{
					Resources: appsv1alpha1.ChallengeTemplateResources{
						Service: appsv1alpha1.ServiceSpec{
							Type: "NodePort",
							Ports: []appsv1alpha1.ServicePort{
								{
									Port:       80,
									TargetPort: 8080,
									NodePort:   30080,
								},
							},
						},
					},
				},
			}

			// Reconcile service
			err := reconciler.reconcileService(ctx, challenge, template, "test-user-problem")

			// Verify no error occurred
			Expect(err).To(BeNil())

			// Verify only one service exists
			serviceList := &v1.ServiceList{}
			err = reconciler.Client.List(ctx, serviceList,
				client.InNamespace("default"),
				client.MatchingLabels{"hexactf.io/challenge": "test-user-problem"})

			Expect(err).To(BeNil())
			Expect(serviceList.Items).To(HaveLen(1))
		})
	})

	Describe("ReconcileResources Method", func() {
		It("should successfully reconcile pod and service", func() {
			// Create a challenge
			challenge := &appsv1alpha1.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
					Labels: map[string]string{
						"hexactf.io/user":      "testuser",
						"hexactf.io/problemId": "testproblem",
					},
				},
				Spec: appsv1alpha1.ChallengeSpec{
					Namespace: "default",
				},
			}

			// Create a template with resources
			template := &appsv1alpha1.ChallengeTemplate{
				Spec: appsv1alpha1.ChallengeTemplateSpec{
					Resources: appsv1alpha1.ChallengeTemplateResources{
						Pod: appsv1alpha1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "test-container",
									Image: "nginx:latest",
								},
							},
						},
						Service: appsv1alpha1.ServiceSpec{
							Type: "NodePort",
							Ports: []appsv1alpha1.ServicePort{
								{
									Port:       80,
									TargetPort: 8080,
									NodePort:   30080,
								},
							},
						},
					},
				},
			}

			// Reconcile resources
			err := reconciler.reconcileResources(ctx, challenge, template, "test-user-problem")

			// Verify no error occurred
			Expect(err).To(BeNil())

			// Verify pod was created
			pod := &v1.Pod{}
			err = reconciler.Client.Get(ctx, types.NamespacedName{
				Name:      "pod-test-user-problem",
				Namespace: "default",
			}, pod)
			Expect(err).To(BeNil())

			// Verify service was created
			service := &v1.Service{}
			err = reconciler.Client.Get(ctx, types.NamespacedName{
				Name:      "svc-test-user-problem",
				Namespace: "default",
			}, service)
			Expect(err).To(BeNil())
		})
	})
})
