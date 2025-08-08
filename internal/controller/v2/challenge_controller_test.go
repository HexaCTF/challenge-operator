package controller

import (
	"context"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Challenge Controller", func() {
	var (
		reconciler *ChallengeReconciler
		client     client.Client
		ctx        context.Context
		req        ctrl.Request
	)

	BeforeEach(func() {
		ctx = context.Background()
		client = CreateFakeClient()
		reconciler = &ChallengeReconciler{
			Client: client,
			Scheme: client.Scheme(),
		}
		req = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-challenge",
				Namespace: "default",
			},
		}
	})

	Describe("Reconcile", func() {
		Context("when challenge is not found", func() {
			It("should return empty result", func() {
				// When
				result, err := reconciler.Reconcile(ctx, req)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
			})
		})

		Context("when challenge exists", func() {
			It("should handle challenge with deletion timestamp", func() {
				// Given
				challenge := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "test-challenge",
						Namespace:         "default",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
				}

				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				result, err := reconciler.Reconcile(ctx, req)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
			})

			It("should add finalizer if not present", func() {
				// Given
				challenge := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-challenge",
						Namespace: "default",
					},
				}

				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				result, err := reconciler.Reconcile(ctx, req)

				// Then
				Expect(err).NotTo(HaveOccurred())
				// Should return empty result since initialization will fail without definition
				Expect(result).To(Equal(ctrl.Result{}))
			})

			It("should initialize challenge if not started", func() {
				// Given
				// Create ChallengeDefinition first
				definition := &hexactfproj.ChallengeDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-definition",
						Namespace: "default",
					},
					Spec: hexactfproj.ChallengeDefinitionSpec{
						Resource: hexactfproj.Resource{
							Pod: &hexactfproj.PodConfig{
								Containers: []hexactfproj.ContainerConfig{
									{
										Name:  "test-container",
										Image: "nginx:latest",
									},
								},
							},
							Service: &corev1.Service{
								ObjectMeta: metav1.ObjectMeta{
									Name: "test-service",
								},
								Spec: corev1.ServiceSpec{
									Type: corev1.ServiceTypeNodePort,
									Ports: []corev1.ServicePort{
										{
											Port:     80,
											NodePort: 30080,
										},
									},
								},
							},
						},
					},
				}

				err := client.Create(ctx, definition)
				Expect(err).NotTo(HaveOccurred())

				challenge := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-challenge",
						Namespace:  "default",
						Finalizers: []string{challengeFinalizer},
					},
					Spec: hexactfproj.ChallengeSpec{
						Definition: "test-definition",
					},
				}

				err = client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				result, err := reconciler.Reconcile(ctx, req)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{RequeueAfter: requeueInterval}))
			})

			It("should handle pending state", func() {
				// Given
				challenge := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-challenge",
						Namespace:  "default",
						Finalizers: []string{challengeFinalizer},
						Labels: map[string]string{
							"apps.hexactf.io/podName": "test-pod",
						},
					},
					Status: hexactfproj.ChallengeStatus{
						StartedAt:     &metav1.Time{Time: time.Now()},
						CurrentStatus: *hexactfproj.NewCurrentStatus(),
					},
				}

				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				result, err := reconciler.Reconcile(ctx, req)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
			})

			It("should handle running state", func() {
				// Given
				status := hexactfproj.NewCurrentStatus()
				status.Running()

				challenge := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-challenge",
						Namespace:  "default",
						Finalizers: []string{challengeFinalizer},
					},
					Status: hexactfproj.ChallengeStatus{
						StartedAt:     &metav1.Time{Time: time.Now()},
						CurrentStatus: *status,
					},
				}

				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				result, err := reconciler.Reconcile(ctx, req)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{RequeueAfter: requeueInterval}))
			})
		})
	})
})

var _ = Describe("Challenge Controller Performance", func() {
	It("should reconcile efficiently", func() {
		// Given
		client := CreateFakeClient()
		reconciler := &ChallengeReconciler{
			Client: client,
			Scheme: client.Scheme(),
		}
		ctx := context.Background()

		// Create ChallengeDefinition first
		definition := &hexactfproj.ChallengeDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "performance-definition",
				Namespace: "default",
			},
			Spec: hexactfproj.ChallengeDefinitionSpec{
				Resource: hexactfproj.Resource{
					Pod: &hexactfproj.PodConfig{
						Containers: []hexactfproj.ContainerConfig{
							{
								Name:  "performance-container",
								Image: "nginx:latest",
							},
						},
					},
				},
			},
		}

		err := client.Create(ctx, definition)
		Expect(err).NotTo(HaveOccurred())

		challenge := NewChallengeBuilder().
			WithName("performance-challenge").
			WithNamespace("default").
			WithFinalizer("challenge.hexactf.io/finalizer").
			WithPodName("performance-pod").
			Build()

		// Set the definition reference
		challenge.Spec.Definition = "performance-definition"

		err = client.Create(ctx, challenge)
		Expect(err).NotTo(HaveOccurred())

		// Create a pod that the challenge references
		pod := NewPodBuilder().
			WithName("performance-pod").
			WithNamespace("default").
			WithPhase(corev1.PodRunning).
			Build()

		err = client.Create(ctx, pod)
		Expect(err).NotTo(HaveOccurred())

		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "performance-challenge",
				Namespace: "default",
			},
		}

		// When
		result, err := reconciler.Reconcile(ctx, req)

		// Then
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(ctrl.Result{RequeueAfter: requeueInterval}))
	})
})
