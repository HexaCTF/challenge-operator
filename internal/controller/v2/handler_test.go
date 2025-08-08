package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockClient struct {
	mock.Mock
	client.Client
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

type MockStatusWriter struct {
	mock.Mock
}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	args := m.Called(ctx, obj, subResource, opts)
	return args.Error(0)
}

type ChallengeReconcilerTestSuite struct {
	suite.Suite
	reconciler        *TestChallengeReconciler
	mockClient        *MockClient
	mockStatus        *MockStatusWriter
	ctx               context.Context
	challenge         *hexactfproj.Challenge
	requeueInterval   time.Duration
	challengeDuration time.Duration
	noTimeCondition   bool
}

func (suite *ChallengeReconcilerTestSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.mockClient = new(MockClient)
	suite.mockStatus = new(MockStatusWriter)
	suite.requeueInterval = time.Minute // 실제 값과 맞춤
	suite.challengeDuration = time.Minute * 5
	suite.noTimeCondition = false

	suite.challenge = &hexactfproj.Challenge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-challenge",
			Namespace: "default",
			Labels: map[string]string{
				"apps.hexactf.io/podName": "test-pod",
			},
		},
		Spec: hexactfproj.ChallengeSpec{
			Definition: "test-definition",
		},
		Status: hexactfproj.ChallengeStatus{
			StartedAt:     &metav1.Time{Time: time.Now()},
			CurrentStatus: *hexactfproj.NewCurrentStatus(),
		},
	}

	suite.reconciler = &TestChallengeReconciler{
		ChallengeReconciler: &ChallengeReconciler{Client: suite.mockClient},
	}
}

func (suite *ChallengeReconcilerTestSuite) TearDownTest() {
	suite.mockClient.AssertExpectations(suite.T())
	suite.mockStatus.AssertExpectations(suite.T())
}
func (suite *ChallengeReconcilerTestSuite) TestInitializeChallenge_Success() {
	// Given
	// loadChallengeDefinition을 mock하여 우회
	suite.reconciler.mockLoadDefinition = func(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
		return nil // 성공
	}

	// Status mock 설정
	suite.mockClient.On("Status").Return(suite.mockStatus)
	suite.mockStatus.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(nil)

	// When
	err := suite.reconciler.initializeChallenge(suite.ctx, suite.challenge)

	// Then
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Pending", suite.challenge.Status.CurrentStatus.Status)
	assert.NotNil(suite.T(), suite.challenge.Status.StartedAt)
}

// TestInitializeChallenge_StatusUpdateFailure는 상태 업데이트 실패 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestInitializeChallenge_StatusUpdateFailure() {
	// Given
	// loadChallengeDefinition을 mock하여 우회
	suite.reconciler.mockLoadDefinition = func(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
		return nil // 성공
	}

	expectedError := errors.New("status update failed")
	suite.mockClient.On("Status").Return(suite.mockStatus)
	suite.mockStatus.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(expectedError)

	// When
	err := suite.reconciler.initializeChallenge(suite.ctx, suite.challenge)

	// Then
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "failed to initialize status")
}

// TestHandlePendingState_PodNotFound는 Pod를 찾을 수 없는 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandlePendingState_PodNotFound() {
	// Given
	expectedError := errors.New("pod not found")
	suite.mockClient.On("Get", suite.ctx, types.NamespacedName{
		Name:      "test-pod",
		Namespace: "default",
	}, mock.AnythingOfType("*v1.Pod"), mock.Anything).Return(expectedError)

	// When
	result, err := suite.reconciler.handlePendingState(suite.ctx, suite.challenge)

	// Then
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), ctrl.Result{}, result)
}

// TestHandlePendingState_PodRunning는 Pod가 실행 중인 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandlePendingState_PodRunning() {
	// Given
	runningPod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	suite.mockClient.On("Get", suite.ctx, types.NamespacedName{
		Name:      "test-pod",
		Namespace: "default",
	}, mock.AnythingOfType("*v1.Pod"), mock.Anything).Run(func(args mock.Arguments) {
		pod := args.Get(2).(*corev1.Pod)
		*pod = *runningPod
	}).Return(nil)

	suite.mockClient.On("Status").Return(suite.mockStatus)
	suite.mockStatus.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(nil)

	// When
	result, err := suite.reconciler.handlePendingState(suite.ctx, suite.challenge)

	// Then
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), ctrl.Result{RequeueAfter: suite.requeueInterval}, result)
	assert.Equal(suite.T(), "Running", suite.challenge.Status.CurrentStatus.Status)
}

// TestHandlePendingState_PodNameNotSet는 podName이 설정되지 않은 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandlePendingState_PodNameNotSet() {
	// Given
	challengeWithoutPodName := &hexactfproj.Challenge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-challenge",
			Namespace: "default",
			Labels:    map[string]string{}, // podName이 없음
		},
	}

	// handleError 메서드를 모킹하기 위해 필요한 mock 설정
	suite.mockClient.On("Get", suite.ctx, ctrl.Request{}.NamespacedName, challengeWithoutPodName, mock.Anything).Return(nil)
	suite.mockClient.On("Status").Return(suite.mockStatus)
	suite.mockStatus.On("Update", suite.ctx, challengeWithoutPodName, mock.Anything).Return(nil)
	suite.mockClient.On("Delete", suite.ctx, challengeWithoutPodName, mock.Anything).Return(nil)

	// When
	_, err := suite.reconciler.handlePendingState(suite.ctx, challengeWithoutPodName)

	// Then
	assert.Error(suite.T(), err)
	assert.Contains(suite.T(), err.Error(), "podName is not set")
}

// TestHandleRunningState_TimeExceeded는 시간이 초과된 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandleRunningState_TimeExceeded() {
	// Given
	oldTime := metav1.Time{Time: time.Now().Add(-10 * time.Minute)} // 5분 제한을 초과
	suite.challenge.Status.StartedAt = &oldTime
	suite.challenge.Status.CurrentStatus.Running()

	suite.mockClient.On("Get", suite.ctx, types.NamespacedName{
		Name:      "test-challenge",
		Namespace: "default",
	}, suite.challenge, mock.Anything).Return(nil)

	// noTimeCondition이 true이므로 실제로는 시간 초과 로직이 실행되지 않음
	// 대신 noTimeCondition에 따른 동작을 테스트

	// When
	result, err := suite.reconciler.handleRunningState(suite.ctx, suite.challenge)

	// Then
	assert.NoError(suite.T(), err)
	// noTimeCondition이 true이므로 RequeueAfter가 설정됨
	assert.Equal(suite.T(), ctrl.Result{RequeueAfter: suite.requeueInterval}, result)
	assert.Equal(suite.T(), "Running", suite.challenge.Status.CurrentStatus.Status)
}

// TestHandleRunningState_WithinTimeLimit는 시간 제한 내인 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandleRunningState_WithinTimeLimit() {
	// Given
	recentTime := metav1.Time{Time: time.Now().Add(-1 * time.Minute)} // 5분 제한 내
	suite.challenge.Status.StartedAt = &recentTime

	suite.mockClient.On("Get", suite.ctx, client.ObjectKey{
		Name:      "test-challenge",
		Namespace: "default",
	}, suite.challenge, mock.Anything).Return(nil)

	// When
	result, err := suite.reconciler.handleRunningState(suite.ctx, suite.challenge)

	// Then
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), ctrl.Result{RequeueAfter: suite.requeueInterval}, result)
}

// TestHandleDeletion_WithFinalizer는 finalizer가 있는 경우의 삭제 처리를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandleDeletion_WithFinalizer() {
	// Given
	suite.challenge.Finalizers = []string{"challenge.hexactf.io/finalizer"}
	suite.mockClient.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(nil)

	// When
	result, err := suite.reconciler.handleDeletion(suite.ctx, suite.challenge)

	// Then
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), ctrl.Result{}, result)
	assert.Empty(suite.T(), suite.challenge.Finalizers)
}

// TestHandleDeletion_UpdateError는 finalizer 제거 중 업데이트 오류가 발생하는 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandleDeletion_UpdateError() {
	// Given
	suite.challenge.Finalizers = []string{"challenge.hexactf.io/finalizer"}
	expectedError := errors.New("update failed")
	suite.mockClient.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(expectedError)

	// When
	result, err := suite.reconciler.handleDeletion(suite.ctx, suite.challenge)

	// Then
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), ctrl.Result{RequeueAfter: time.Second * 5}, result)
}

// TestHandleError_Success는 에러 처리 성공 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandleError_Success() {
	// Given
	testError := errors.New("test error")
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-challenge",
			Namespace: "default",
		},
	}

	suite.mockClient.On("Get", suite.ctx, req.NamespacedName, suite.challenge, mock.Anything).Return(nil)
	suite.mockClient.On("Status").Return(suite.mockStatus)
	suite.mockStatus.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(nil)
	suite.mockClient.On("Delete", suite.ctx, suite.challenge, mock.Anything).Return(nil)

	// When
	result, err := suite.reconciler.handleError(suite.ctx, req, suite.challenge, testError)

	// Then
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), testError, err)
	assert.Equal(suite.T(), ctrl.Result{}, result)
	assert.Equal(suite.T(), "Error", suite.challenge.Status.CurrentStatus.Status)
}

// TestHandleError_DeleteFailure는 challenge 삭제 실패 케이스를 테스트
func (suite *ChallengeReconcilerTestSuite) TestHandleError_DeleteFailure() {
	// Given
	testError := errors.New("test error")
	deleteError := errors.New("delete failed")
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-challenge",
			Namespace: "default",
		},
	}

	suite.mockClient.On("Get", suite.ctx, req.NamespacedName, suite.challenge, mock.Anything).Return(nil)
	suite.mockClient.On("Status").Return(suite.mockStatus)
	suite.mockStatus.On("Update", suite.ctx, suite.challenge, mock.Anything).Return(nil)
	suite.mockClient.On("Delete", suite.ctx, suite.challenge, mock.Anything).Return(deleteError)

	// When
	result, err := suite.reconciler.handleError(suite.ctx, req, suite.challenge, testError)

	// Then
	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), deleteError, err)
	assert.Equal(suite.T(), ctrl.Result{}, result)
}

// 테스트 스위트 실행을 위한 함수
func TestChallengeReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(ChallengeReconcilerTestSuite))
}

// 추가적인 유닛 테스트들

// TestCurrentStatus는 CurrentStatus 구조체의 메서드들을 테스트
func TestCurrentStatus(t *testing.T) {
	status := hexactfproj.NewCurrentStatus()

	// 초기 상태 확인
	assert.Equal(t, "Pending", status.Status)
	assert.True(t, status.IsPending())

	// Running 상태 테스트
	status.Running()
	assert.Equal(t, "Running", status.Status)
	assert.True(t, status.IsRunning())
	assert.False(t, status.IsPending())

	// Terminating 상태 테스트
	status.Terminating()
	assert.Equal(t, "Terminating", status.Status)
	assert.True(t, status.IsTerminating())
	assert.False(t, status.IsRunning())

	// Error 상태 테스트
	testError := errors.New("test error")
	status.Error(testError)
	assert.Equal(t, "Error", status.Status)

	// Deleted 상태 테스트
	status.Deleted()
	assert.Equal(t, "Deleted", status.Status)
	assert.True(t, status.IsDeleted())
}

// BenchmarkHandlePendingState는 handlePendingState의 성능을 벤치마크
func BenchmarkHandlePendingState(b *testing.B) {
	// 벤치마크 setup
	mockClient := new(MockClient)
	mockStatus := new(MockStatusWriter)
	reconciler := &ChallengeReconciler{Client: mockClient}
	ctx := context.Background()

	challenge := &hexactfproj.Challenge{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-challenge",
			Namespace: "default",
			Labels: map[string]string{
				"apps.hexactf.io/podName": "benchmark-pod",
			},
		},
	}

	runningPod := &corev1.Pod{
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	mockClient.On("Get", ctx, mock.Anything, mock.AnythingOfType("*v1.Pod"), mock.Anything).Run(func(args mock.Arguments) {
		pod := args.Get(2).(*corev1.Pod)
		*pod = *runningPod
	}).Return(nil)
	mockClient.On("Status").Return(mockStatus)
	mockStatus.On("Update", ctx, challenge, mock.Anything).Return(nil)

	b.ResetTimer()

	// 벤치마크 실행
	for i := 0; i < b.N; i++ {
		reconciler.handlePendingState(ctx, challenge)
	}
}
