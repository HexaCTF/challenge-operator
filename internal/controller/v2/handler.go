package controller

import (
	"context"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ChallengeReconciler) handleDeletion(ctx context.Context, challenge *hexactfproj.Challenge) (ctrl.Result, error) {
	log.Info("Processing deletion", "challenge", challenge.Name)

	// 1. Finalizer가 남아있는지 확인
	if controllerutil.ContainsFinalizer(challenge, "challenge.hexactf.io/finalizer") {

		// 3. 파이널라이저 제거
		controllerutil.RemoveFinalizer(challenge, "challenge.hexactf.io/finalizer")

		// 4. 메타데이터 업데이트
		if err := r.Update(ctx, challenge); err != nil {
			log.Error(err, "Failed to remove finalizer")
			// 재시도 위해 Requeue
			return ctrl.Result{RequeueAfter: time.Second * 5}, err
		}

	}

	log.Info("Successfully completed deletion process")
	// 이 시점에서 finalizers가 비어 있으므로, K8s가 오브젝트를 실제 삭제함
	return ctrl.Result{}, nil
}
