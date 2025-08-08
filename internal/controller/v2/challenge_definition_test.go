package controller

import (
	"context"
	"fmt"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Challenge Definition", func() {
	var (
		client     client.Client
		reconciler *ChallengeReconciler
		ctx        context.Context
	)

	BeforeEach(func() {
		client = CreateFakeClient()
		reconciler = &ChallengeReconciler{
			Client: client,
			Scheme: client.Scheme(),
		}
		ctx = context.Background()
	})

	Describe("LoadChallengeDefinition", func() {
		Context("when challenge definition exists", func() {
			It("should load challenge definition successfully", func() {
				// Given
				challenge := NewChallengeBuilder().
					WithName("test-challenge").
					WithNamespace("default").
					WithPodName("test-pod").
					Build()

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

				// Create definition in client
				err := client.Create(ctx, definition)
				Expect(err).NotTo(HaveOccurred())

				// Create challenge in client
				err = client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// Get the challenge from client to ensure it exists
				err = client.Get(ctx, types.NamespacedName{
					Name:      challenge.Name,
					Namespace: challenge.Namespace,
				}, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = reconciler.loadChallengeDefinition(ctx, ctrl.Request{}, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify labels were set correctly
				expectedLabels := map[string]string{
					"apps.hexactf.io/challengeId":   "test-challenge-id",
					"apps.hexactf.io/userId":        "test-user-id",
					"apps.hexactf.io/createdBy":     "challenge-operator",
					"apps.hexactf.io/challengeName": "challenge-test-challenge-id-test-user-id",
					"apps.hexactf.io/podName":       "challenge-test-challenge-id-test-user-id-pod",
					"apps.hexactf.io/svcName":       "challenge-test-challenge-id-test-user-id-svc",
				}

				for key, expectedValue := range expectedLabels {
					actualValue, exists := challenge.Labels[key]
					Expect(exists).To(BeTrue(), "Expected label %s to exist", key)
					Expect(actualValue).To(Equal(expectedValue), "Expected label %s to be %s, but got %s", key, expectedValue, actualValue)
				}
			})
		})

		Context("when challenge definition does not exist", func() {
			It("should return an error", func() {
				// Given
				challenge := NewChallengeBuilder().
					WithName("test-challenge").
					WithNamespace("default").
					Build()

				// Create challenge in client but not definition
				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// When
				err = reconciler.loadChallengeDefinition(ctx, ctrl.Request{}, challenge)

				// Then
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("LoadPod", func() {
		It("should load pod successfully", func() {
			// Given
			challenge := NewChallengeBuilder().
				WithName("test-challenge").
				WithNamespace("default").
				WithPodName("test-pod").
				Build()

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:latest",
						},
					},
				},
			}

			// When
			err := reconciler.loadPod(ctx, challenge, pod)

			// Then
			Expect(err).NotTo(HaveOccurred())

			// Verify pod was created
			createdPod := &corev1.Pod{}
			err = client.Get(ctx, types.NamespacedName{
				Name:      "test-pod",
				Namespace: "default",
			}, createdPod)
			Expect(err).NotTo(HaveOccurred())

			// Verify status was set to pending
			Expect(challenge.Status.CurrentStatus.Status).To(Equal("Pending"))
		})
	})

	Describe("LoadService", func() {
		Context("with NodePort service", func() {
			It("should load service successfully", func() {
				// Given
				challenge := NewChallengeBuilder().
					WithName("test-challenge").
					WithNamespace("default").
					Build()

				// Set the required labels for service creation
				challenge.Labels["apps.hexactf.io/svcName"] = "test-service"

				// Create challenge in client
				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// Get the challenge from client to ensure it exists
				err = client.Get(ctx, types.NamespacedName{
					Name:      challenge.Name,
					Namespace: challenge.Namespace,
				}, challenge)
				Expect(err).NotTo(HaveOccurred())

				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "default",
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
				}

				// When
				err = reconciler.loadService(ctx, challenge, service)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify service was created
				createdService := &corev1.Service{}
				err = client.Get(ctx, types.NamespacedName{
					Name:      "test-service",
					Namespace: "default",
				}, createdService)
				Expect(err).NotTo(HaveOccurred())

				// Verify endpoint was set
				Expect(challenge.Status.Endpoint).To(Equal(30080))
			})
		})

		Context("with ClusterIP service", func() {
			It("should load service successfully without setting endpoint", func() {
				// Given
				challenge := NewChallengeBuilder().
					WithName("test-challenge").
					WithNamespace("default").
					Build()

				// Set the required labels for service creation
				challenge.Labels["apps.hexactf.io/svcName"] = "test-service"

				// Create challenge in client
				err := client.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())

				// Get the challenge from client to ensure it exists
				err = client.Get(ctx, types.NamespacedName{
					Name:      challenge.Name,
					Namespace: challenge.Namespace,
				}, challenge)
				Expect(err).NotTo(HaveOccurred())

				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service",
						Namespace: "default",
					},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{
							{
								Port: 80,
							},
						},
					},
				}

				// When
				err = reconciler.loadService(ctx, challenge, service)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Verify service was created
				createdService := &corev1.Service{}
				err = client.Get(ctx, types.NamespacedName{
					Name:      "test-service",
					Namespace: "default",
				}, createdService)
				Expect(err).NotTo(HaveOccurred())

				// Verify endpoint was not set for ClusterIP
				Expect(challenge.Status.Endpoint).To(Equal(0))
			})
		})
	})

	Describe("GetChallengeDefinition", func() {
		It("should get challenge definition successfully", func() {
			// Given
			challenge := NewChallengeBuilder().
				WithName("test-challenge").
				WithNamespace("default").
				Build()

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
					},
				},
			}

			// Create definition in client
			err := client.Create(ctx, definition)
			Expect(err).NotTo(HaveOccurred())

			// When
			retrievedDefinition, err := reconciler.getChallengeDefinition(ctx, challenge)

			// Then
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedDefinition).NotTo(BeNil())
			Expect(retrievedDefinition.Name).To(Equal("test-definition"))
		})
	})
})

var _ = Describe("Challenge Definition Performance", func() {
	var (
		client     client.Client
		reconciler *ChallengeReconciler
		ctx        context.Context
	)

	BeforeEach(func() {
		client = CreateFakeClient()
		reconciler = &ChallengeReconciler{
			Client: client,
			Scheme: client.Scheme(),
		}
		ctx = context.Background()
	})

	It("should load challenge definition efficiently", func() {
		// Create a fresh challenge for each iteration with unique name
		challengeName := fmt.Sprintf("benchmark-challenge-%d", GinkgoParallelProcess())
		definitionName := fmt.Sprintf("benchmark-definition-%d", GinkgoParallelProcess())
		challengeId := fmt.Sprintf("benchmark-challenge-id-%d", GinkgoParallelProcess())
		userId := fmt.Sprintf("benchmark-user-id-%d", GinkgoParallelProcess())

		challenge := NewChallengeBuilder().
			WithName(challengeName).
			WithNamespace("default").
			Build()

		// Set unique challengeId and userId to ensure unique pod/service names
		challenge.Labels["apps.hexactf.io/challengeId"] = challengeId
		challenge.Labels["apps.hexactf.io/userId"] = userId

		// Create a fresh definition for each iteration
		definition := &hexactfproj.ChallengeDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      definitionName,
				Namespace: "default",
			},
			Spec: hexactfproj.ChallengeDefinitionSpec{
				Resource: hexactfproj.Resource{
					Pod: &hexactfproj.PodConfig{
						Containers: []hexactfproj.ContainerConfig{
							{
								Name:  "benchmark-container",
								Image: "nginx:latest",
							},
						},
					},
					Service: &corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name: "benchmark-service",
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

		// Create definition in client
		err := client.Create(ctx, definition)
		Expect(err).NotTo(HaveOccurred())

		// Update the challenge to reference the correct definition
		challenge.Spec.Definition = definitionName

		err = client.Create(ctx, challenge)
		Expect(err).NotTo(HaveOccurred())

		// Get the challenge from client to ensure it exists
		err = client.Get(ctx, types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		}, challenge)
		Expect(err).NotTo(HaveOccurred())

		err = reconciler.loadChallengeDefinition(ctx, ctrl.Request{}, challenge)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should load pod efficiently", func() {
		// Create a fresh challenge for each iteration with unique name
		challengeName := fmt.Sprintf("benchmark-challenge-%d", GinkgoParallelProcess())
		podName := fmt.Sprintf("benchmark-pod-%d", GinkgoParallelProcess())
		challenge := NewChallengeBuilder().
			WithName(challengeName).
			WithNamespace("default").
			WithPodName(podName).
			Build()

		// Create challenge in client
		err := client.Create(ctx, challenge)
		Expect(err).NotTo(HaveOccurred())

		// Get the challenge from client to ensure it exists
		err = client.Get(ctx, types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		}, challenge)
		Expect(err).NotTo(HaveOccurred())

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "default",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "benchmark-container",
						Image: "nginx:latest",
					},
				},
			},
		}

		err = reconciler.loadPod(ctx, challenge, pod)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should load service efficiently", func() {
		// Create a fresh challenge for each iteration with unique name
		challengeName := fmt.Sprintf("benchmark-challenge-%d", GinkgoParallelProcess())
		challenge := NewChallengeBuilder().
			WithName(challengeName).
			WithNamespace("default").
			Build()

		// Set the required labels for service creation
		serviceName := fmt.Sprintf("benchmark-service-%d", GinkgoParallelProcess())
		challenge.Labels["apps.hexactf.io/svcName"] = serviceName

		// Create challenge in client
		err := client.Create(ctx, challenge)
		Expect(err).NotTo(HaveOccurred())

		// Get the challenge from client to ensure it exists
		err = client.Get(ctx, types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		}, challenge)
		Expect(err).NotTo(HaveOccurred())

		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: "default",
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
		}

		err = reconciler.loadService(ctx, challenge, service)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should get challenge definition efficiently", func() {
		// Create a fresh challenge for each iteration with unique name
		challengeName := fmt.Sprintf("benchmark-challenge-%d", GinkgoParallelProcess())
		definitionName := fmt.Sprintf("benchmark-definition-%d", GinkgoParallelProcess())

		challenge := NewChallengeBuilder().
			WithName(challengeName).
			WithNamespace("default").
			Build()

		// Update the challenge to reference the correct definition
		challenge.Spec.Definition = definitionName

		// Create a fresh definition for each iteration
		definition := &hexactfproj.ChallengeDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      definitionName,
				Namespace: "default",
			},
			Spec: hexactfproj.ChallengeDefinitionSpec{
				Resource: hexactfproj.Resource{
					Pod: &hexactfproj.PodConfig{
						Containers: []hexactfproj.ContainerConfig{
							{
								Name:  "benchmark-container",
								Image: "nginx:latest",
							},
						},
					},
				},
			},
		}

		// Create definition in client
		err := client.Create(ctx, definition)
		Expect(err).NotTo(HaveOccurred())

		_, err = reconciler.getChallengeDefinition(ctx, challenge)
		Expect(err).NotTo(HaveOccurred())
	})
})
