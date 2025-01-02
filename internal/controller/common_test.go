package controller

import (
	"context"
	"errors"
	"testing"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestHandleError(t *testing.T) {
	// Setup scheme
	scheme := runtime.NewScheme()
	assert.NoError(t, hexactfproj.AddToScheme(scheme))

	// Create test challenge
	challenge := &hexactfproj.Challenge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-challenge",
			Namespace: "default",
		},
		Status: hexactfproj.ChallengeStatus{
			CurrentStatus: hexactfproj.CurrentStatus{
				State: "Running",
			},
		},
	}

	// Create client with scheme and initial objects
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(challenge).
		WithStatusSubresource(challenge). // This is important for status updates
		Build()

	// Create reconciler
	reconciler := &ChallengeReconciler{
		Client: client,
		Scheme: scheme,
	}

	// Create test error
	testErr := errors.New("test error")

	// Test handleError
	result, err := reconciler.handleError(context.Background(), challenge, testErr)

	// Verify the returned error
	assert.Equal(t, testErr, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify the challenge status was updated
	updatedChallenge := &hexactfproj.Challenge{}
	err = client.Get(context.Background(),
		types.NamespacedName{
			Name:      challenge.Name,
			Namespace: challenge.Namespace,
		},
		updatedChallenge)

	assert.NoError(t, err)
	assert.Equal(t, "Error", updatedChallenge.Status.CurrentStatus.State)
	assert.Equal(t, testErr.Error(), updatedChallenge.Status.CurrentStatus.ErrorMsg)
}
