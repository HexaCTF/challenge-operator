package controller

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ChallengeReconciler) initializeChallenge(ctx context.Context, challenge *hexactfproj.Challenge) error {
	if err := r.loadChallengeDefinition(ctx, ctrl.Request{}, challenge); err != nil {
		return fmt.Errorf("failed to load challenge definition: %w", err)
	}

	now := metav1.Now()
	challenge.Status.StartedAt = &now
	challenge.Status.CurrentStatus = *hexactfproj.NewCurrentStatus()

	if err := r.Status().Update(ctx, challenge); err != nil {
		return fmt.Errorf("failed to initialize status: %w", err)
	}

	return nil
}

func (r *ChallengeReconciler) handlePendingState(ctx context.Context, challenge *hexactfproj.Challenge) (ctrl.Result, error) {
	pod := &corev1.Pod{}
	podName := challenge.Labels["apps.hexactf.io/podName"]
	if podName == "" {
		return r.handleError(ctx, ctrl.Request{}, challenge, fmt.Errorf("podName is not set"))
	}

	if err := r.Get(ctx, client.ObjectKey{
		Name:      podName,
		Namespace: challenge.Namespace,
	}, pod); err != nil {
		log.Error(err, "Failed to get Pod", "podName", podName)
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodRunning {
		challenge.Status.CurrentStatus.Running()
		if err := r.Status().Update(ctx, challenge); err != nil {
			log.Error(err, "Failed to update Challenge status", "challenge", challenge.Name)
		}

	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *ChallengeReconciler) handleRunningState(ctx context.Context, challenge *hexactfproj.Challenge) (ctrl.Result, error) {
	if err := r.Get(ctx, client.ObjectKey{
		Name:      challenge.Name,
		Namespace: challenge.Namespace,
	}, challenge); err != nil {
		return r.handleError(ctx, ctrl.Request{}, challenge, err)
	}

	if noTimeCondition {
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	if time.Since(challenge.Status.StartedAt.Time) > challengeDuration || noTimeCondition {
		challenge.Status.CurrentStatus.Terminating()
		if err := r.Status().Update(ctx, challenge); err != nil {
			log.Error(err, "Failed to update Challenge status", "challenge", challenge.Name)
		}

		if challenge.DeletionTimestamp.IsZero() {
			log.Info("Time exceeded; issuing a Delete request", "challenge", challenge.Name)
			if err := r.Delete(ctx, challenge); err != nil {
				log.Error(err, "Failed to delete challenge")
				return r.handleError(ctx, ctrl.Request{}, challenge, err)
			}
			return ctrl.Result{}, nil
		}
		return r.handleDeletion(ctx, challenge)
	}

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

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

// handleError: 상태를 Error로 변경하고 로그 & Kafka 메시지 전송 등
func (r *ChallengeReconciler) handleError(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge, err error) (ctrl.Result, error) {
	// 최신 상태로 갱신
	if err := r.Get(ctx, req.NamespacedName, challenge); err != nil {
		// NotFound 에러 등은 무시
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	challenge.Status.CurrentStatus.Error(err)

	if err := r.Status().Update(ctx, challenge); err != nil {
		return ctrl.Result{}, err
	}

	// 에러 발생 시 challenge 삭제
	if deleteErr := r.Delete(ctx, challenge); deleteErr != nil {
		log.Error(deleteErr, "Failed to delete Challenge")
		return ctrl.Result{}, deleteErr
	}

	return ctrl.Result{}, err
}
