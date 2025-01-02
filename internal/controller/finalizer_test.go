package controller

import (
	"context"
	"testing"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleFinalizer(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.NoError(t, clientgoscheme.AddToScheme(scheme))
	assert.NoError(t, hexactfproj.AddToScheme(scheme))

	tests := []struct {
		name       string
		challenge  *hexactfproj.Challenge
		isDeleting bool
	}{
		{
			name: "add finalizer to new challenge",
			challenge: &hexactfproj.Challenge{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Challenge",
					APIVersion: hexactfproj.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-challenge",
					Namespace: "default",
				},
				Status: hexactfproj.ChallengeStatus{
					CurrentStatus: hexactfproj.CurrentStatus{
						State: "Created",
					},
				},
			},
			isDeleting: false,
		},
		{
			name: "handle deleting challenge with finalizer",
			challenge: &hexactfproj.Challenge{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Challenge",
					APIVersion: hexactfproj.GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-challenge",
					Namespace:         "default",
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{challengeFinalizer},
				},
				Status: hexactfproj.ChallengeStatus{
					CurrentStatus: hexactfproj.CurrentStatus{
						State: "Running",
					},
				},
			},
			isDeleting: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake client
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&hexactfproj.Challenge{}).
				Build()

			// Create the initial challenge
			err := client.Create(context.Background(), tt.challenge.DeepCopy())
			assert.NoError(t, err)

			reconciler := &ChallengeReconciler{
				Client: client,
				Scheme: scheme,
			}

			// Execute handleFinalizer
			err = reconciler.handleFinalizer(context.Background(), tt.challenge)
			assert.NoError(t, err)

			// Get the updated challenge
			updatedChallenge := &hexactfproj.Challenge{}
			err = client.Get(context.Background(),
				types.NamespacedName{
					Name:      tt.challenge.Name,
					Namespace: tt.challenge.Namespace,
				},
				updatedChallenge)
			assert.NoError(t, err)

			if tt.isDeleting {
				// 삭제 중인 경우, finalizer가 제거되어야 함
				assert.False(t, containsString(updatedChallenge.Finalizers, challengeFinalizer),
					"Finalizer should be removed for deleting challenge")
			} else {
				// 새로운 challenge의 경우, finalizer가 추가되어야 함
				assert.True(t, containsString(updatedChallenge.Finalizers, challengeFinalizer),
					"Finalizer should be added for new challenge")
			}
		})
	}
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("test containsString", func(t *testing.T) {
		tests := []struct {
			slice    []string
			str      string
			expected bool
		}{
			{
				slice:    []string{"a", "b", "c"},
				str:      "b",
				expected: true,
			},
			{
				slice:    []string{"a", "b", "c"},
				str:      "d",
				expected: false,
			},
			{
				slice:    []string{},
				str:      "a",
				expected: false,
			},
		}

		for _, tt := range tests {
			result := containsString(tt.slice, tt.str)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("test removeString", func(t *testing.T) {
		tests := []struct {
			name     string
			slice    []string
			str      string
			expected []string
		}{
			{
				name:     "remove existing element",
				slice:    []string{"a", "b", "c"},
				str:      "b",
				expected: []string{"a", "c"},
			},
			{
				name:     "remove non-existing element",
				slice:    []string{"a", "b", "c"},
				str:      "d",
				expected: []string{"a", "b", "c"},
			},
			{
				name:     "remove from empty slice",
				slice:    []string{},
				str:      "a",
				expected: nil, // 기대값을 nil로 변경
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := removeString(tt.slice, tt.str)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}
func TestCleanupChallenge(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	assert.NoError(t, hexactfproj.AddToScheme(scheme))

	tests := []struct {
		name      string
		challenge *hexactfproj.Challenge
		expectErr bool
	}{
		{
			name: "successful cleanup",
			challenge: &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-challenge",
					Namespace:  "default",
					Finalizers: []string{challengeFinalizer},
				},
			},
			expectErr: false,
		},
		{
			name: "cleanup nonexistent challenge",
			challenge: &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nonexistent-challenge",
					Namespace: "default",
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.challenge).
				Build()

			reconciler := &ChallengeReconciler{
				Client: client,
				Scheme: scheme,
			}

			err := reconciler.cleanupChallenge(context.Background(), tt.challenge)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
