package controller

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestChallengeReconciler는 테스트용 ChallengeReconciler를 래핑하는 구조체
type TestChallengeReconciler struct {
	*ChallengeReconciler
	mockLoadDefinition func(context.Context, ctrl.Request, *hexactfproj.Challenge) error
}

// loadChallengeDefinition을 테스트용으로 오버라이드
func (r *TestChallengeReconciler) loadChallengeDefinition(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
	if r.mockLoadDefinition != nil {
		return r.mockLoadDefinition(ctx, req, challenge)
	}
	return nil // 기본적으로는 성공으로 처리
}

// initializeChallenge을 테스트용으로 오버라이드
func (r *TestChallengeReconciler) initializeChallenge(ctx context.Context, challenge *hexactfproj.Challenge) error {
	// loadChallengeDefinition을 mock으로 우회
	if r.mockLoadDefinition != nil {
		if err := r.mockLoadDefinition(ctx, ctrl.Request{}, challenge); err != nil {
			return fmt.Errorf("failed to load challenge definition: %w", err)
		}
	}

	// initialize status
	// default status is Pending
	now := metav1.Now()
	challenge.Status.StartedAt = &now
	challenge.Status.CurrentStatus = *hexactfproj.NewCurrentStatus()

	if err := r.Status().Update(ctx, challenge); err != nil {
		return fmt.Errorf("failed to initialize status: %w", err)
	}

	return nil
}

// NewTestChallengeReconciler는 테스트용 reconciler를 생성
func NewTestChallengeReconciler(client client.Client) *TestChallengeReconciler {
	return &TestChallengeReconciler{
		ChallengeReconciler: &ChallengeReconciler{
			Client: client,
		},
	}
}

// ChallengeBuilder는 테스트용 Challenge 객체를 빌드하기 위한 빌더 패턴
type ChallengeBuilder struct {
	challenge *hexactfproj.Challenge
}

// NewChallengeBuilder는 새로운 ChallengeBuilder를 생성
func NewChallengeBuilder() *ChallengeBuilder {
	return &ChallengeBuilder{
		challenge: &hexactfproj.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-challenge",
				Namespace: "default",
				Labels: map[string]string{
					"apps.hexactf.io/challengeId": "test-challenge-id",
					"apps.hexactf.io/userId":      "test-user-id",
				},
			},
			Spec: hexactfproj.ChallengeSpec{
				Definition: "test-definition",
			},
			Status: hexactfproj.ChallengeStatus{
				StartedAt:     &metav1.Time{Time: time.Now()},
				CurrentStatus: *hexactfproj.NewCurrentStatus(),
			},
		},
	}
}

// WithName은 Challenge의 이름을 설정
func (b *ChallengeBuilder) WithName(name string) *ChallengeBuilder {
	b.challenge.Name = name
	return b
}

// WithNamespace는 Challenge의 네임스페이스를 설정
func (b *ChallengeBuilder) WithNamespace(namespace string) *ChallengeBuilder {
	b.challenge.Namespace = namespace
	return b
}

// WithPodName은 Challenge의 podName 라벨을 설정
func (b *ChallengeBuilder) WithPodName(podName string) *ChallengeBuilder {
	b.challenge.Labels["apps.hexactf.io/podName"] = podName
	return b
}

// WithStatus는 Challenge의 상태를 설정
func (b *ChallengeBuilder) WithStatus(status string) *ChallengeBuilder {
	b.challenge.Status.CurrentStatus.Status = status
	return b
}

// WithStartedAt은 Challenge의 시작 시간을 설정
func (b *ChallengeBuilder) WithStartedAt(startedAt time.Time) *ChallengeBuilder {
	b.challenge.Status.StartedAt = &metav1.Time{Time: startedAt}
	return b
}

// WithDeletionTimestamp는 Challenge의 삭제 타임스탬프를 설정
func (b *ChallengeBuilder) WithDeletionTimestamp(timestamp time.Time) *ChallengeBuilder {
	metaTime := metav1.Time{Time: timestamp}
	b.challenge.DeletionTimestamp = &metaTime
	return b
}

// WithFinalizer는 Challenge에 finalizer를 추가
func (b *ChallengeBuilder) WithFinalizer(finalizer string) *ChallengeBuilder {
	b.challenge.Finalizers = append(b.challenge.Finalizers, finalizer)
	return b
}

// Build는 구성된 Challenge 객체를 반환
func (b *ChallengeBuilder) Build() *hexactfproj.Challenge {
	return b.challenge.DeepCopy()
}

// PodBuilder는 테스트용 Pod 객체를 빌드하기 위한 빌더 패턴
type PodBuilder struct {
	pod *corev1.Pod
}

// NewPodBuilder는 새로운 PodBuilder를 생성
func NewPodBuilder() *PodBuilder {
	return &PodBuilder{
		pod: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},
	}
}

// WithName은 Pod의 이름을 설정
func (b *PodBuilder) WithName(name string) *PodBuilder {
	b.pod.Name = name
	return b
}

// WithNamespace는 Pod의 네임스페이스를 설정
func (b *PodBuilder) WithNamespace(namespace string) *PodBuilder {
	b.pod.Namespace = namespace
	return b
}

// WithPhase는 Pod의 단계를 설정
func (b *PodBuilder) WithPhase(phase corev1.PodPhase) *PodBuilder {
	b.pod.Status.Phase = phase
	return b
}

// Build는 구성된 Pod 객체를 반환
func (b *PodBuilder) Build() *corev1.Pod {
	return b.pod.DeepCopy()
}

// CreateFakeClient는 테스트용 fake client를 생성
func CreateFakeClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = hexactfproj.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&hexactfproj.Challenge{}).
		Build()
}

// AssertChallengeStatus는 Challenge의 상태를 검증하는 헬퍼 함수
func AssertChallengeStatus(t mock.TestingT, challenge *hexactfproj.Challenge, expectedStatus string) {
	if challenge.Status.CurrentStatus.Status != expectedStatus {
		t.Errorf("Expected status %s, but got %s", expectedStatus, challenge.Status.CurrentStatus.Status)
	}
}

// AssertPodExists는 클러스터에 Pod가 존재하는지 확인하는 헬퍼 함수
func AssertPodExists(t mock.TestingT, client client.Client, ctx context.Context, name, namespace string) {
	pod := &corev1.Pod{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := client.Get(ctx, key, pod)
	if err != nil {
		t.Errorf("Expected pod %s/%s to exist, but got error: %v", namespace, name, err)
	}
}

// AssertChallengeNotExists는 클러스터에 Challenge가 존재하지 않는지 확인하는 헬퍼 함수
func AssertChallengeNotExists(t mock.TestingT, client client.Client, ctx context.Context, name, namespace string) {
	challenge := &hexactfproj.Challenge{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := client.Get(ctx, key, challenge)
	if err == nil {
		t.Errorf("Expected challenge %s/%s to not exist, but it was found", namespace, name)
	}
}

// MockChallengeReconcilerOptions는 mock reconciler 생성 옵션
type MockChallengeReconcilerOptions struct {
	Client            client.Client
	RequeueInterval   time.Duration
	ChallengeDuration time.Duration
	NoTimeCondition   bool
}

// NewMockChallengeReconciler는 설정 가능한 mock reconciler를 생성
func NewMockChallengeReconciler(opts MockChallengeReconcilerOptions) *TestChallengeReconciler {
	if opts.Client == nil {
		opts.Client = CreateFakeClient()
	}
	if opts.RequeueInterval == 0 {
		opts.RequeueInterval = time.Second * 10
	}
	if opts.ChallengeDuration == 0 {
		opts.ChallengeDuration = time.Minute * 5
	}

	reconciler := NewTestChallengeReconciler(opts.Client)

	return reconciler
}
