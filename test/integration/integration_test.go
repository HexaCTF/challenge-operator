package integration

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var _ = Describe("Challenge Operator Integration", Ordered, func() {
	var (
		k8sClient client.Client
		ctx       context.Context
		namespace string
	)

	BeforeAll(func() {
		By("Setting up the test environment")

		// Get the kubeconfig
		cfg, err := config.GetConfig()
		Expect(err).NotTo(HaveOccurred())

		// Add our custom types to the scheme
		err = hexactfproj.AddToScheme(scheme.Scheme)
		Expect(err).NotTo(HaveOccurred())

		// Create the client
		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()
		namespace = "default"

		By("CRDs should be pre-installed")
	})

	AfterAll(func() {
		By("Cleaning up test namespace")
		// Cleanup will be handled by individual tests
	})

	Describe("Challenge Definition Lifecycle", func() {
		var definitionName string

		BeforeEach(func() {
			definitionName = fmt.Sprintf("test-definition-%d", GinkgoParallelProcess())
		})

		It("should create and manage challenge definitions", func() {
			By("Creating a challenge definition")
			definition := &hexactfproj.ChallengeDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      definitionName,
					Namespace: namespace,
				},
				Spec: hexactfproj.ChallengeDefinitionSpec{
					Resource: hexactfproj.Resource{
						Pod: &hexactfproj.PodConfig{
							Containers: []hexactfproj.ContainerConfig{
								{
									Name:  "test-container",
									Image: "nginx:latest",
									Ports: []hexactfproj.ContainerPort{
										{
											ContainerPort: 80,
											Protocol:      "TCP",
										},
									},
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

			err := k8sClient.Create(ctx, definition)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the definition was created")
			createdDefinition := &hexactfproj.ChallengeDefinition{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      definitionName,
				Namespace: namespace,
			}, createdDefinition)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDefinition.Name).To(Equal(definitionName))

			By("Cleaning up the definition")
			err = k8sClient.Delete(ctx, createdDefinition)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Challenge Lifecycle", func() {
		var (
			definitionName string
			challengeName  string
			definition     *hexactfproj.ChallengeDefinition
			challenge      *hexactfproj.Challenge
		)

		BeforeEach(func() {
			definitionName = fmt.Sprintf("test-definition-%d", GinkgoParallelProcess())
			challengeName = fmt.Sprintf("test-challenge-%d", GinkgoParallelProcess())

			// Create definition
			definition = &hexactfproj.ChallengeDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      definitionName,
					Namespace: namespace,
				},
				Spec: hexactfproj.ChallengeDefinitionSpec{
					Resource: hexactfproj.Resource{
						Pod: &hexactfproj.PodConfig{
							Containers: []hexactfproj.ContainerConfig{
								{
									Name:  "test-container",
									Image: "nginx:latest",
									Ports: []hexactfproj.ContainerPort{
										{
											ContainerPort: 80,
											Protocol:      "TCP",
										},
									},
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

			err := k8sClient.Create(ctx, definition)
			Expect(err).NotTo(HaveOccurred())

			// Create challenge
			challenge = &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      challengeName,
					Namespace: namespace,
					Labels: map[string]string{
						"apps.hexactf.io/challengeId": fmt.Sprintf("challenge-id-%d", GinkgoParallelProcess()),
						"apps.hexactf.io/userId":      fmt.Sprintf("user-id-%d", GinkgoParallelProcess()),
					},
				},
				Spec: hexactfproj.ChallengeSpec{
					Definition: definitionName,
				},
			}
		})

		AfterEach(func() {
			By("Cleaning up test resources")
			if challenge != nil {
				err := k8sClient.Delete(ctx, challenge)
				if err != nil {
					fmt.Printf("Failed to delete challenge: %v\n", err)
				}
			}
			if definition != nil {
				err := k8sClient.Delete(ctx, definition)
				if err != nil {
					fmt.Printf("Failed to delete definition: %v\n", err)
				}
			}
		})

		It("should create challenges successfully", func() {
			By("Creating a challenge")
			err := k8sClient.Create(ctx, challenge)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the challenge was created")
			createdChallenge := &hexactfproj.Challenge{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      challengeName,
				Namespace: namespace,
			}, createdChallenge)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdChallenge.Name).To(Equal(challengeName))

			By("Verifying challenge spec is correct")
			Expect(createdChallenge.Spec.Definition).To(Equal(definitionName))
			Expect(createdChallenge.Labels).To(HaveKey("apps.hexactf.io/challengeId"))
			Expect(createdChallenge.Labels).To(HaveKey("apps.hexactf.io/userId"))

			By("Deleting the challenge")
			err = k8sClient.Delete(ctx, createdChallenge)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Challenge Error Handling", func() {
		var challengeName string

		BeforeEach(func() {
			challengeName = fmt.Sprintf("error-challenge-%d-%d", GinkgoParallelProcess(), time.Now().Unix())
		})

		It("should create challenge with invalid definition", func() {
			By("Creating a challenge with non-existent definition")
			challenge := &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      challengeName,
					Namespace: namespace,
					Labels: map[string]string{
						"apps.hexactf.io/challengeId": fmt.Sprintf("error-challenge-id-%d", GinkgoParallelProcess()),
						"apps.hexactf.io/userId":      fmt.Sprintf("error-user-id-%d", GinkgoParallelProcess()),
					},
				},
				Spec: hexactfproj.ChallengeSpec{
					Definition: "non-existent-definition",
				},
			}

			err := k8sClient.Create(ctx, challenge)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying the challenge was created")
			createdChallenge := &hexactfproj.Challenge{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      challengeName,
				Namespace: namespace,
			}, createdChallenge)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdChallenge.Name).To(Equal(challengeName))

			By("Verifying challenge spec is correct")
			Expect(createdChallenge.Spec.Definition).To(Equal("non-existent-definition"))

			By("Cleaning up the challenge")
			err = k8sClient.Delete(ctx, challenge)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Multiple Challenges", func() {
		var (
			definitionName string
			challenges     []*hexactfproj.Challenge
		)

		BeforeEach(func() {
			definitionName = fmt.Sprintf("multi-definition-%d", GinkgoParallelProcess())
			challenges = make([]*hexactfproj.Challenge, 0)

			// Create definition
			definition := &hexactfproj.ChallengeDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      definitionName,
					Namespace: namespace,
				},
				Spec: hexactfproj.ChallengeDefinitionSpec{
					Resource: hexactfproj.Resource{
						Pod: &hexactfproj.PodConfig{
							Containers: []hexactfproj.ContainerConfig{
								{
									Name:  "multi-container",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, definition)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			By("Cleaning up multiple challenges")
			for _, challenge := range challenges {
				err := k8sClient.Delete(ctx, challenge)
				if err != nil {
					fmt.Printf("Failed to delete challenge %s: %v\n", challenge.Name, err)
				}
			}
		})

		It("should create multiple challenges successfully", func() {
			By("Creating multiple challenges")
			for i := 0; i < 3; i++ {
				challenge := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("multi-challenge-%d-%d", GinkgoParallelProcess(), i),
						Namespace: namespace,
						Labels: map[string]string{
							"apps.hexactf.io/challengeId": fmt.Sprintf("multi-challenge-id-%d-%d", GinkgoParallelProcess(), i),
							"apps.hexactf.io/userId":      fmt.Sprintf("multi-user-id-%d-%d", GinkgoParallelProcess(), i),
						},
					},
					Spec: hexactfproj.ChallengeSpec{
						Definition: definitionName,
					},
				}

				err := k8sClient.Create(ctx, challenge)
				Expect(err).NotTo(HaveOccurred())
				challenges = append(challenges, challenge)
			}

			By("Verifying all challenges are created")
			challengeList := &hexactfproj.ChallengeList{}
			err := k8sClient.List(ctx, challengeList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(challengeList.Items)).To(BeNumerically(">=", 3))

			By("Verifying challenge specs are correct")
			for _, challenge := range challenges {
				createdChallenge := &hexactfproj.Challenge{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      challenge.Name,
					Namespace: namespace,
				}, createdChallenge)
				Expect(err).NotTo(HaveOccurred())
				Expect(createdChallenge.Spec.Definition).To(Equal(definitionName))
			}
		})
	})
})

// Helper function to check if error is "already exists"
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "admission webhook \"vchallenge.kb.io\" denied the request: resource already exists" ||
		errStr == "resource already exists" ||
		errStr == "namespaces \"integration-test\" already exists"
}
