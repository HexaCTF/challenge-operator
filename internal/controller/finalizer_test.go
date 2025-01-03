package controller

import (
	"context"
	"testing"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFinalizerOperations(t *testing.T) {
	// 스키마 설정
	s := runtime.NewScheme()
	_ = scheme.AddToScheme(s)
	_ = hexactfproj.AddToScheme(s)

	// 테스트용 Challenge 생성 헬퍼 함수
	createTestChallenge := func(name string, finalizers []string) *hexactfproj.Challenge {
		return &hexactfproj.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Namespace:  "default",
				Finalizers: finalizers,
			},
		}
	}

	t.Run("addFinalizer", func(t *testing.T) {
		tests := []struct {
			name      string
			challenge *hexactfproj.Challenge
			wantErr   bool
		}{
			{
				name:      "finalizer 추가 성공",
				challenge: createTestChallenge("test-1", nil),
				wantErr:   false,
			},
			{
				name:      "이미 finalizer가 있는 경우",
				challenge: createTestChallenge("test-2", []string{challengeFinalizer}),
				wantErr:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// fake client 생성
				client := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(tt.challenge).
					Build()

				r := &ChallengeReconciler{Client: client}

				// addFinalizer 실행
				_, err := r.addFinalizer(context.Background(), tt.challenge)

				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				// Challenge 다시 가져와서 finalizer 확인
				updated := &hexactfproj.Challenge{}
				err = client.Get(context.Background(),
					types.NamespacedName{
						Name:      tt.challenge.Name,
						Namespace: tt.challenge.Namespace,
					},
					updated)
				require.NoError(t, err)
				assert.Contains(t, updated.Finalizers, challengeFinalizer)
			})
		}
	})

	t.Run("removeFinalizer", func(t *testing.T) {
		tests := []struct {
			name      string
			challenge *hexactfproj.Challenge
			wantErr   bool
		}{
			{
				name:      "finalizer 제거 성공",
				challenge: createTestChallenge("test-3", []string{challengeFinalizer}),
				wantErr:   false,
			},
			{
				name:      "finalizer가 없는 경우",
				challenge: createTestChallenge("test-4", nil),
				wantErr:   false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// fake client 생성
				client := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(tt.challenge).
					Build()

				r := &ChallengeReconciler{Client: client}

				// removeFinalizer 실행
				err := r.removeFinalizer(context.Background(), tt.challenge)

				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)

				// Challenge가 존재하는 경우 finalizer 확인
				updated := &hexactfproj.Challenge{}
				err = client.Get(context.Background(),
					types.NamespacedName{
						Name:      tt.challenge.Name,
						Namespace: tt.challenge.Namespace,
					},
					updated)

				if err == nil {
					assert.NotContains(t, updated.Finalizers, challengeFinalizer)
				}
			})
		}
	})
}

func TestFinalizerHelperFunctions(t *testing.T) {
	t.Run("containsString", func(t *testing.T) {
		tests := []struct {
			name     string
			slice    []string
			expected bool
		}{
			{
				name:     "빈 슬라이스",
				slice:    []string{},
				expected: false,
			},
			{
				name:     "finalizer 포함",
				slice:    []string{challengeFinalizer, "other-finalizer"},
				expected: true,
			},
			{
				name:     "finalizer 미포함",
				slice:    []string{"other-finalizer"},
				expected: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := containsString(tt.slice)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("removeString", func(t *testing.T) {
		tests := []struct {
			name     string
			slice    []string
			expected []string
		}{
			{
				name:     "빈 슬라이스",
				slice:    []string{},
				expected: nil,
			},
			{
				name:     "finalizer만 있는 경우",
				slice:    []string{challengeFinalizer},
				expected: nil,
			},
			{
				name:     "여러 finalizer가 있는 경우",
				slice:    []string{challengeFinalizer, "other-finalizer"},
				expected: []string{"other-finalizer"},
			},
			{
				name:     "다른 finalizer만 있는 경우",
				slice:    []string{"other-finalizer"},
				expected: []string{"other-finalizer"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := removeString(tt.slice)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}
