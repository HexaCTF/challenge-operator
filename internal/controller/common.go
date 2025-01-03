package controller

import (
	"context"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// handleError 에러 발생 시 로깅과 상태(CurrentStatus)를 업데이트한다.
func (r *ChallengeReconciler) handleError(ctx context.Context, challenge *hexactfproj.Challenge, err error) (ctrl.Result, error) {

	// 최신 상태 가져오기
	latest := &hexactfproj.Challenge{}
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      challenge.Name,
	}, latest); err != nil {
		return ctrl.Result{}, err
	}

	latest.Status.CurrentStatus.SetError(err)
	patch := client.MergeFrom(latest.DeepCopy())
	if updateErr := r.Status().Patch(ctx, latest, patch); updateErr != nil {
		log.Error(updateErr, "failed to update Challenge status")
		return ctrl.Result{}, updateErr
	}
	return ctrl.Result{}, err
}
