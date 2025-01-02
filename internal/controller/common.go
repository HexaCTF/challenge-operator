package controller

import (
	"context"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// handleError 에러 발생 시 로깅과 상태(CurrentStatus)를 업데이트한다.
func (r *ChallengeReconciler) handleError(ctx context.Context, challenge *hexactfproj.Challenge, err error) (ctrl.Result, error) {
	challenge.Status.CurrentStatus.SetError(err)
	if err := r.Status().Update(ctx, challenge); err != nil {
		log.Error(err, "failed to update Challenge status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, err
}
