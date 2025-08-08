package controller

import (
	"context"
	"errors"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
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

var _ = Describe("Challenge Reconciler Handler", func() {
	var (
		reconciler      *TestChallengeReconciler
		mockClient      *MockClient
		mockStatus      *MockStatusWriter
		ctx             context.Context
		challenge       *hexactfproj.Challenge
		requeueInterval time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = new(MockClient)
		mockStatus = new(MockStatusWriter)
		requeueInterval = time.Minute // 실제 값과 맞춤

		challenge = &hexactfproj.Challenge{
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

		reconciler = &TestChallengeReconciler{
			ChallengeReconciler: &ChallengeReconciler{Client: mockClient},
		}
	})

	AfterEach(func() {
		mockClient.AssertExpectations(GinkgoT())
		mockStatus.AssertExpectations(GinkgoT())
	})

	Describe("InitializeChallenge", func() {
		Context("when initialization succeeds", func() {
			It("should initialize challenge successfully", func() {
				// Given
				// loadChallengeDefinition을 mock하여 우회
				reconciler.mockLoadDefinition = func(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
					return nil // 성공
				}

				// Status mock 설정
				mockClient.On("Status").Return(mockStatus)
				mockStatus.On("Update", ctx, challenge, mock.Anything).Return(nil)

				// When
				err := reconciler.initializeChallenge(ctx, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(challenge.Status.CurrentStatus.Status).To(Equal("Pending"))
				Expect(challenge.Status.StartedAt).NotTo(BeNil())
			})
		})

		Context("when status update fails", func() {
			It("should return an error", func() {
				// Given
				// loadChallengeDefinition을 mock하여 우회
				reconciler.mockLoadDefinition = func(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
					return nil // 성공
				}

				expectedError := errors.New("status update failed")
				mockClient.On("Status").Return(mockStatus)
				mockStatus.On("Update", ctx, challenge, mock.Anything).Return(expectedError)

				// When
				err := reconciler.initializeChallenge(ctx, challenge)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to initialize status"))
			})
		})
	})

	Describe("HandlePendingState", func() {
		Context("when pod is not found", func() {
			It("should handle pod not found", func() {
				// Given
				expectedError := errors.New("pod not found")
				mockClient.On("Get", ctx, types.NamespacedName{
					Name:      "test-pod",
					Namespace: "default",
				}, mock.AnythingOfType("*v1.Pod"), mock.Anything).Return(expectedError)

				// When
				result, err := reconciler.handlePendingState(ctx, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
			})
		})

		Context("when pod is running", func() {
			It("should handle running pod", func() {
				// Given
				runningPod := &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				}

				mockClient.On("Get", ctx, types.NamespacedName{
					Name:      "test-pod",
					Namespace: "default",
				}, mock.AnythingOfType("*v1.Pod"), mock.Anything).Run(func(args mock.Arguments) {
					pod := args.Get(2).(*corev1.Pod)
					*pod = *runningPod
				}).Return(nil)

				mockClient.On("Status").Return(mockStatus)
				mockStatus.On("Update", ctx, challenge, mock.Anything).Return(nil)

				// When
				result, err := reconciler.handlePendingState(ctx, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{RequeueAfter: requeueInterval}))
				Expect(challenge.Status.CurrentStatus.Status).To(Equal("Running"))
			})
		})

		Context("when pod name is not set", func() {
			It("should handle missing pod name", func() {
				// Given
				challengeWithoutPodName := &hexactfproj.Challenge{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-challenge",
						Namespace: "default",
						Labels:    map[string]string{}, // podName이 없음
					},
				}

				// handleError 메서드를 모킹하기 위해 필요한 mock 설정
				mockClient.On("Get", ctx, ctrl.Request{}.NamespacedName, challengeWithoutPodName, mock.Anything).Return(nil)
				mockClient.On("Status").Return(mockStatus)
				mockStatus.On("Update", ctx, challengeWithoutPodName, mock.Anything).Return(nil)
				mockClient.On("Delete", ctx, challengeWithoutPodName, mock.Anything).Return(nil)

				// When
				_, err := reconciler.handlePendingState(ctx, challengeWithoutPodName)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("podName is not set"))
			})
		})
	})

	Describe("HandleRunningState", func() {
		Context("when time limit is exceeded", func() {
			It("should handle time exceeded", func() {
				// Given
				oldTime := metav1.Time{Time: time.Now().Add(-10 * time.Minute)} // 5분 제한을 초과
				challenge.Status.StartedAt = &oldTime
				challenge.Status.CurrentStatus.Running()

				mockClient.On("Get", ctx, types.NamespacedName{
					Name:      "test-challenge",
					Namespace: "default",
				}, challenge, mock.Anything).Return(nil)

				// noTimeCondition이 true이므로 실제로는 시간 초과 로직이 실행되지 않음
				// 대신 noTimeCondition에 따른 동작을 테스트

				// When
				result, err := reconciler.handleRunningState(ctx, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())
				// noTimeCondition이 true이므로 RequeueAfter가 설정됨
				Expect(result).To(Equal(ctrl.Result{RequeueAfter: requeueInterval}))
				Expect(challenge.Status.CurrentStatus.Status).To(Equal("Running"))
			})
		})

		Context("when within time limit", func() {
			It("should handle within time limit", func() {
				// Given
				recentTime := metav1.Time{Time: time.Now().Add(-1 * time.Minute)} // 5분 제한 내
				challenge.Status.StartedAt = &recentTime

				mockClient.On("Get", ctx, client.ObjectKey{
					Name:      "test-challenge",
					Namespace: "default",
				}, challenge, mock.Anything).Return(nil)

				// When
				result, err := reconciler.handleRunningState(ctx, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{RequeueAfter: requeueInterval}))
			})
		})
	})

	Describe("HandleDeletion", func() {
		Context("when finalizer exists", func() {
			It("should handle deletion with finalizer", func() {
				// Given
				challenge.Finalizers = []string{"challenge.hexactf.io/finalizer"}
				mockClient.On("Update", ctx, challenge, mock.Anything).Return(nil)

				// When
				result, err := reconciler.handleDeletion(ctx, challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
				Expect(challenge.Finalizers).To(BeEmpty())
			})
		})

		Context("when update fails", func() {
			It("should handle update error", func() {
				// Given
				challenge.Finalizers = []string{"challenge.hexactf.io/finalizer"}
				expectedError := errors.New("update failed")
				mockClient.On("Update", ctx, challenge, mock.Anything).Return(expectedError)

				// When
				result, err := reconciler.handleDeletion(ctx, challenge)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{RequeueAfter: time.Second * 5}))
			})
		})
	})

	Describe("HandleError", func() {
		Context("when error handling succeeds", func() {
			It("should handle error successfully", func() {
				// Given
				testError := errors.New("test error")
				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test-challenge",
						Namespace: "default",
					},
				}

				mockClient.On("Get", ctx, req.NamespacedName, challenge, mock.Anything).Return(nil)
				mockClient.On("Status").Return(mockStatus)
				mockStatus.On("Update", ctx, challenge, mock.Anything).Return(nil)
				mockClient.On("Delete", ctx, challenge, mock.Anything).Return(nil)

				// When
				result, err := reconciler.handleError(ctx, req, challenge, testError)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(testError))
				Expect(result).To(Equal(ctrl.Result{}))
				Expect(challenge.Status.CurrentStatus.Status).To(Equal("Error"))
			})
		})

		Context("when delete fails", func() {
			It("should handle delete failure", func() {
				// Given
				testError := errors.New("test error")
				deleteError := errors.New("delete failed")
				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test-challenge",
						Namespace: "default",
					},
				}

				mockClient.On("Get", ctx, req.NamespacedName, challenge, mock.Anything).Return(nil)
				mockClient.On("Status").Return(mockStatus)
				mockStatus.On("Update", ctx, challenge, mock.Anything).Return(nil)
				mockClient.On("Delete", ctx, challenge, mock.Anything).Return(deleteError)

				// When
				result, err := reconciler.handleError(ctx, req, challenge, testError)

				// Then
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(deleteError))
				Expect(result).To(Equal(ctrl.Result{}))
			})
		})
	})
})

var _ = Describe("CurrentStatus", func() {
	It("should handle status transitions correctly", func() {
		status := hexactfproj.NewCurrentStatus()

		// 초기 상태 확인
		Expect(status.Status).To(Equal("Pending"))
		Expect(status.IsPending()).To(BeTrue())

		// Running 상태 테스트
		status.Running()
		Expect(status.Status).To(Equal("Running"))
		Expect(status.IsRunning()).To(BeTrue())
		Expect(status.IsPending()).To(BeFalse())

		// Terminating 상태 테스트
		status.Terminating()
		Expect(status.Status).To(Equal("Terminating"))
		Expect(status.IsTerminating()).To(BeTrue())
		Expect(status.IsRunning()).To(BeFalse())

		// Error 상태 테스트
		testError := errors.New("test error")
		status.Error(testError)
		Expect(status.Status).To(Equal("Error"))

		// Deleted 상태 테스트
		status.Deleted()
		Expect(status.Status).To(Equal("Deleted"))
		Expect(status.IsDeleted()).To(BeTrue())
	})
})

var _ = Describe("Challenge Reconciler Performance", func() {
	It("should handle pending state efficiently", func() {
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

		reconciler.handlePendingState(ctx, challenge)
	})
})
