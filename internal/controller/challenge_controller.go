/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logr "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	challengeDuration = 5 * time.Minute
	requeueInterval   = 30 * time.Second
	warningThreshold  = 2 * time.Minute // Time to start warning about impending timeout
)

var log = logr.Log.WithName("ChallengeController")

// ChallengeReconciler reconciles a Challenge object
type ChallengeReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	// KafkaClient is the Kafka producer client
	// 중요한 메세지를 Kafka를 통해 보낸다.
	KafkaClient *KafkaProducer
}

// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.hexactf.io,resources=challenges/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Challenge object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *ChallengeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	challenge := hexactfproj.Challenge{}
	if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
		// NotFound 에러 등은 무시
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 이미 삭제 진행중(DeletionTimestamp가 찍힘)이라면 handleDeletion 수행
	if !challenge.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &challenge)
	}

	// Finalizer가 없다면 추가
	if !containsString(challenge.Finalizers) {
		return r.addFinalizer(ctx, &challenge)
	}

	// 처음 생성 시 StartedAt 등 Status 초기화
	if challenge.Status.StartedAt == nil {
		crStatusMetric.WithLabelValues(challenge.Name, challenge.Namespace).Set(0)

		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		challenge.Status.CurrentStatus = *hexactfproj.NewCurrentStatus()
		if err := r.Status().Update(ctx, &challenge); err != nil {
			log.Error(err, "Failed to initialize status")
			return r.handleError(ctx, req, &challenge, err)
		}
		err := r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Running")
		if err != nil {
			log.Error(err, "Failed to send status change message")
			return r.handleError(ctx, req, &challenge, err)
		}
	}

	// 최신 상태로 갱신
	if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
		// NotFound 에러 등은 무시
		return r.handleError(ctx, req, &challenge, err)
	}

	// 현재 상태에 따라 분기
	switch {
	case challenge.Status.CurrentStatus.IsNone():
		// 상태를 Creating -> Running 으로 전환
		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}

		challenge.Status.CurrentStatus.SetCreating()
		if err := r.Status().Update(ctx, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}

		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}
		// 실제 Challenge에 필요한 리소스들(Deployment, Service 등) 생성 로직
		err := r.loadChallengeDefinition(ctx, &challenge)
		if err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}

		// 다시 한번 최신화
		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}

		challenge.Status.CurrentStatus.SetRunning()
		now := metav1.Now()
		challenge.Status.StartedAt = &now
		if err := r.Status().Update(ctx, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}
		// Metrics
		crStatusMetric.WithLabelValues(challenge.Labels["apps.hexactf.io/challengeId"], challenge.Name, challenge.Labels["apps.hexactf.io/user"], challenge.Namespace).Set(1)

		// 한 번 더 재큐(Requeue)하여 바로 다음 단계 확인
		return ctrl.Result{Requeue: true}, nil

	case challenge.Status.CurrentStatus.IsRunning():
		// 최신화
		if err := r.Get(ctx, req.NamespacedName, &challenge); err != nil {
			return r.handleError(ctx, req, &challenge, err)
		}

		// 5분 초과 시 Delete 요청
		if time.Since(challenge.Status.StartedAt.Time) > challengeDuration {
			// 아직 DeletionTimestamp가 없다면 Delete 요청
			if challenge.DeletionTimestamp.IsZero() {

				log.Info("Time exceeded; issuing a Delete request", "challenge", challenge.Name)
				if err := r.Delete(ctx, &challenge); err != nil {
					log.Error(err, "Failed to delete challenge")
					return r.handleError(ctx, req, &challenge, err)
				}
				// Delete 요청 후에는 Kubernetes가 DeletionTimestamp를 설정하고
				// 다시 Reconcile이 호출되면 handleDeletion()이 수행됨
				return ctrl.Result{}, nil
			} else {
				// 이미 Delete 진행중이면 handleDeletion으로
				return r.handleDeletion(ctx, &challenge)
			}
		}

		// 이미 삭제 요청(DeletionTimestamp가 존재) 중이라면 handleDeletion으로
		if !challenge.DeletionTimestamp.IsZero() {
			return r.handleDeletion(ctx, &challenge)
		}

		// 아직 삭제 대상이 아니라면 Running 상태 유지
		err := r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Running")
		if err != nil {
			log.Error(err, "Failed to send status change message")
			return r.handleError(ctx, req, &challenge, err)
		}

		// 주기적으로 다시 Reconcile
		return ctrl.Result{RequeueAfter: requeueInterval}, nil
	}

	// 그 외 상태
	return ctrl.Result{}, nil
}

func (r *ChallengeReconciler) handleDeletion(ctx context.Context, challenge *hexactfproj.Challenge) (ctrl.Result, error) {
	log.Info("Processing deletion", "challenge", challenge.Name)
	crStatusMetric.WithLabelValues(challenge.Labels["apps.hexactf.io/challengeId"], challenge.Name, challenge.Labels["apps.hexactf.io/user"], challenge.Namespace).Set(2)

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

		// 필요하다면 Deleted 이벤트 전송
		err := r.KafkaClient.SendStatusChange(
			challenge.Labels["apps.hexactf.io/user"],
			challenge.Labels["apps.hexactf.io/challengeId"],
			"Deleted",
		)
		if err != nil {
			log.Error(err, "Failed to send status change message")
			// 여기서도 에러 시 재시도
			return ctrl.Result{}, err
		}
	}

	go func() {
		time.Sleep(1 * time.Minute) // scrape_interval이 30초라면 1분 정도 기다리면 안전
		crStatusMetric.DeleteLabelValues(challenge.Labels["apps.hexactf.io/challengeId"], challenge.Name, challenge.Labels["apps.hexactf.io/user"], challenge.Namespace)
	}()
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
	challenge.Status.CurrentStatus.SetError(err)

	if err := r.Status().Update(ctx, challenge); err != nil {
		return ctrl.Result{}, err
	}

	crStatusMetric.WithLabelValues(challenge.Labels["apps.hexactf.io/challengeId"], challenge.Name, challenge.Labels["apps.hexactf.io/user"], challenge.Namespace).Set(3)
	// 상태를 Error로 전송
	sendErr := r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Error")
	if sendErr != nil {
		log.Error(sendErr, "Failed to send status change message")
		return ctrl.Result{}, sendErr
	}

	// 에러 발생 시 challenge 삭제
	if deleteErr := r.Delete(ctx, challenge); deleteErr != nil {
		log.Error(deleteErr, "Failed to delete Challenge")
		return ctrl.Result{}, deleteErr
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChallengeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hexactfproj.Challenge{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Named("challenge").
		Complete(r)
}
