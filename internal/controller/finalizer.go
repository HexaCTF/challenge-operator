package controller

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	challengeFinalizer = "challenge.hexactf.dev/finalizer"
)

func (r *ChallengeReconciler) addFinalizer(ctx context.Context, challenge *hexactfproj.Challenge) (ctrl.Result, error) {
	challenge.Finalizers = append(challenge.Finalizers, challengeFinalizer)
	if err := r.Update(ctx, challenge); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}
	return ctrl.Result{}, nil
}

func (r *ChallengeReconciler) removeFinalizer(ctx context.Context, challenge *hexactfproj.Challenge) error {
	retries := 3

	for i := 0; i < retries; i++ {
		var latestChallenge hexactfproj.Challenge
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: challenge.Namespace,
			Name:      challenge.Name,
		}, &latestChallenge); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		latestChallenge.Finalizers = removeString(latestChallenge.Finalizers)

		if err := r.Update(ctx, &latestChallenge); err != nil {
			if apierrors.IsConflict(err) {
				time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
				continue
			}
			return err
		}

		return nil
	}

	return fmt.Errorf("failed to remove finalizer after %d attempts", retries)
}

func containsString(slice []string) bool {
	for _, item := range slice {
		if item == challengeFinalizer {
			return true
		}
	}
	return false
}

func removeString(slice []string) []string {
	if len(slice) == 0 {
		return nil
	}

	var result []string
	for _, item := range slice {
		if item != challengeFinalizer {
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
