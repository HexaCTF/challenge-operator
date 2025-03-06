package controller

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Resource Loader", func() {
	var (
		reconciler *ChallengeReconciler
		ctx        context.Context
		challenge  *hexactfproj.Challenge
		component  *hexactfproj.Component
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = appsv1.AddToScheme(scheme)
		_ = hexactfproj.AddToScheme(scheme)

		ctx = context.Background()

		challenge = &hexactfproj.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-challenge",
				Namespace: "default",
			},
			Status: hexactfproj.ChallengeStatus{
				Endpoint: 0,
			},
		}
		reconciler = &ChallengeReconciler{
			Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(challenge).Build(),
			Scheme: scheme,
		}
		err := k8sClient.Create(ctx, challenge)
		if apierrors.IsConflict(err) {
			err := k8sClient.Delete(ctx, challenge)
			if err != nil && !apierrors.IsNotFound(err) {
				Fail(fmt.Sprintf("Failed to delete challenge after test: %v", err))
			}
			time.Sleep(1 * time.Second)
			err = k8sClient.Create(ctx, challenge)
			if err != nil {
				Fail(fmt.Sprintf("Failed to create challenge after test: %v", err))
			}
		}

		component = &hexactfproj.Component{
			Name: "ubuntu-sample",
			Deployment: &hexactfproj.CustomDeployment{
				Spec: hexactfproj.CustomDeploymentSpec{
					Replicas: 1,
					Template: hexactfproj.CustomPodTemplateSpec{
						Spec: hexactfproj.CustomPodSpec{
							Containers: []corev1.Container{
								{
									Name:  "ubuntu-container",
									Image: "ubuntu:latest",
									Command: []string{
										"/bin/bash", "-c", "sleep 3600",
									},
								},
							},
						},
					},
				},
			},
			Service: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ubuntu-service",
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"app": "ubuntu-sample",
					},
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{
							Port:       80,
							TargetPort: intstr.FromInt(80),
							NodePort:   30080,
						},
					},
				},
			},
		}
		time.Sleep(500 * time.Millisecond) // Give fake client time to persist
	})

	AfterEach(func() {
		By("cleaning up test challenge")
		err := reconciler.Client.Delete(ctx, challenge)
		if err != nil && !apierrors.IsNotFound(err) {
			Fail(fmt.Sprintf("Failed to delete challenge after test: %v", err))
		}
		time.Sleep(1 * time.Second)
	})

	Describe("Load Deployment", func() {
		It("should successfully load a deployment", func() {
			identifier := NewChallengeIdentifier(challenge, *component)
			err := reconciler.loadDeployment(ctx, challenge, *component, identifier)
			Expect(err).NotTo(HaveOccurred())
		})
	})

})
