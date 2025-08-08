package controller

import (
	"context"
	"testing"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func BenchmarkChallengeReconciliation(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	_ = hexactfproj.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &ChallengeReconciler{
		Client: client,
		Scheme: scheme,
	}
	ctx := context.Background()

	// Create definition
	definition := &hexactfproj.ChallengeDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-definition",
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
	_ = client.Create(ctx, definition)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create challenge
		challenge := NewChallengeBuilder().
			WithName("benchmark-challenge").
			WithNamespace("default").
			WithFinalizer("challenge.hexactf.io/finalizer").
			WithPodName("benchmark-pod").
			Build()
		challenge.Spec.Definition = "benchmark-definition"

		_ = client.Create(ctx, challenge)

		// Create pod
		pod := NewPodBuilder().
			WithName("benchmark-pod").
			WithNamespace("default").
			WithPhase(corev1.PodRunning).
			Build()
		_ = client.Create(ctx, pod)

		// Reconcile
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "benchmark-challenge",
				Namespace: "default",
			},
		}
		_, _ = reconciler.Reconcile(ctx, req)

		// Cleanup
		_ = client.Delete(ctx, challenge)
		_ = client.Delete(ctx, pod)
	}
}

func BenchmarkLoadChallengeDefinition(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	_ = hexactfproj.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &ChallengeReconciler{
		Client: client,
		Scheme: scheme,
	}
	ctx := context.Background()

	// Create definition
	definition := &hexactfproj.ChallengeDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-definition",
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
	_ = client.Create(ctx, definition)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		challenge := NewChallengeBuilder().
			WithName("benchmark-challenge").
			WithNamespace("default").
			Build()
		challenge.Spec.Definition = "benchmark-definition"

		_ = client.Create(ctx, challenge)

		// Get the challenge from client to ensure it exists
		_ = client.Get(ctx, types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		}, challenge)

		_ = reconciler.loadChallengeDefinition(ctx, ctrl.Request{}, challenge)

		// Cleanup
		_ = client.Delete(ctx, challenge)
	}
}

func BenchmarkHandlePendingState(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	_ = hexactfproj.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &ChallengeReconciler{
		Client: client,
		Scheme: scheme,
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create challenge with pending status
		challenge := NewChallengeBuilder().
			WithName("benchmark-challenge").
			WithNamespace("default").
			WithPodName("benchmark-pod").
			Build()

		_ = client.Create(ctx, challenge)

		// Create pod
		pod := NewPodBuilder().
			WithName("benchmark-pod").
			WithNamespace("default").
			WithPhase(corev1.PodRunning).
			Build()
		_ = client.Create(ctx, pod)

		// Get the challenge from client to ensure it exists
		_ = client.Get(ctx, types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		}, challenge)

		_, _ = reconciler.handlePendingState(ctx, challenge)

		// Cleanup
		_ = client.Delete(ctx, challenge)
		_ = client.Delete(ctx, pod)
	}
}

func BenchmarkHandleRunningState(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	_ = hexactfproj.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &ChallengeReconciler{
		Client: client,
		Scheme: scheme,
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create challenge with running status
		status := hexactfproj.NewCurrentStatus()
		status.Running()

		challenge := NewChallengeBuilder().
			WithName("benchmark-challenge").
			WithNamespace("default").
			Build()
		challenge.Status.CurrentStatus = *status

		_ = client.Create(ctx, challenge)

		// Get the challenge from client to ensure it exists
		_ = client.Get(ctx, types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		}, challenge)

		_, _ = reconciler.handleRunningState(ctx, challenge)

		// Cleanup
		_ = client.Delete(ctx, challenge)
	}
}
