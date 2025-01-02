package controller

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	challengeFinalizer = "challenge.hexactf.dev/finalizer"
)

func (r *ChallengeReconciler) handleFinalizer(ctx context.Context, challenge *hexactfproj.Challenge) error {
	// 삭제 중인 Challenge 처리
	if !challenge.ObjectMeta.DeletionTimestamp.IsZero() {
		if containsString(challenge.ObjectMeta.Finalizers, challengeFinalizer) {
			if err := r.cleanupChallenge(ctx, challenge); err != nil {
				return err
			}
		}
		return nil
	}

	// Finalizer 추가
	if !containsString(challenge.ObjectMeta.Finalizers, challengeFinalizer) {
		// 최신 버전의 Challenge를 가져옴
		var latestChallenge hexactfproj.Challenge
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: challenge.Namespace,
			Name:      challenge.Name,
		}, &latestChallenge); err != nil {
			return err
		}

		// Finalizer 추가
		controllerutil.AddFinalizer(&latestChallenge, challengeFinalizer)
		if err := r.Update(ctx, &latestChallenge); err != nil {
			return err
		}
	}

	return nil
}

func (r *ChallengeReconciler) cleanupChallenge(ctx context.Context, challenge *hexactfproj.Challenge) error {
	retries := 3

	for i := 0; i < retries; i++ {
		// Get the latest version of the resource
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

		// Remove the finalizer
		latestChallenge.Finalizers = removeString(latestChallenge.Finalizers, challengeFinalizer)

		// Try to update
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

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	if len(slice) == 0 {
		return nil
	}

	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
