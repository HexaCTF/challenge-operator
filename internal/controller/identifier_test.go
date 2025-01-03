package controller

import (
	"testing"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewChallengeIdentifier(t *testing.T) {
	// 테스트 데이터 준비
	challenge := &hexactfproj.Challenge{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-challenge",
			Labels: map[string]string{
				"hexactf.io/challengeId": "test-id",
				"hexactf.io/user":        "test-user",
			},
		},
	}

	component := hexactfproj.Component{
		Name: "ubuntu",
	}

	// 테스트 실행
	identifier := NewChallengeIdentifier(challenge, component)

	// 검증
	t.Run("prefix 포맷 검증", func(t *testing.T) {
		expectedPrefix := "chall-test-id-ubuntu-test-user"
		assert.Equal(t, expectedPrefix, identifier.prefix)
	})

	t.Run("labels 검증", func(t *testing.T) {
		expectedLabels := map[string]string{
			"apps.hexactf.io/instance":   "chall-test-id-ubuntu-test-user",
			"apps.hexactf.io/name":       "ubuntu",
			"apps.hexactf.io/part-of":    "test-challenge",
			"apps.hexactf.io/managed-by": "challenge-operator",
		}
		assert.Equal(t, expectedLabels, identifier.GetLabels())
	})

	t.Run("selector 검증", func(t *testing.T) {
		expectedSelector := map[string]string{
			"apps.hexactf.io/instance": "chall-test-id-ubuntu-test-user",
		}
		assert.Equal(t, expectedSelector, identifier.GetSelector())
	})

	t.Run("deployment prefix 검증", func(t *testing.T) {
		expectedDeployPrefix := "chall-test-id-ubuntu-test-user-deploy"
		assert.Equal(t, expectedDeployPrefix, identifier.GetDeploymentPrefix())
	})

	t.Run("service prefix 검증", func(t *testing.T) {
		expectedSvcPrefix := "chall-test-id-ubuntu-test-user-svc"
		assert.Equal(t, expectedSvcPrefix, identifier.GetServicePrefix())
	})
}

func TestNewChallengeIdentifierEdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		challenge  *hexactfproj.Challenge
		component  hexactfproj.Component
		wantPrefix string
	}{
		{
			name: "빈 레이블",
			challenge: &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "test-challenge",
					Labels: map[string]string{},
				},
			},
			component: hexactfproj.Component{
				Name: "ubuntu",
			},
			wantPrefix: "chall---ubuntu-",
		},
		{
			name: "긴 이름",
			challenge: &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-challenge",
					Labels: map[string]string{
						"hexactf.io/challengeId": "very-long-challenge-id",
						"hexactf.io/user":        "very-long-user-name",
					},
				},
			},
			component: hexactfproj.Component{
				Name: "very-long-component-name",
			},
			wantPrefix: "chall-very-long-challenge-id-very-long-component-name-very-long-user-name",
		},
		{
			name: "특수문자 포함",
			challenge: &hexactfproj.Challenge{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-challenge",
					Labels: map[string]string{
						"hexactf.io/challengeId": "test@id",
						"hexactf.io/user":        "user#1",
					},
				},
			},
			component: hexactfproj.Component{
				Name: "test!comp",
			},
			wantPrefix: "chall-test@id-test!comp-user#1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identifier := NewChallengeIdentifier(tt.challenge, tt.component)

			// prefix 검증
			assert.Equal(t, tt.wantPrefix, identifier.prefix)

			// 기본 레이블 존재 여부 검증
			labels := identifier.GetLabels()
			assert.Contains(t, labels, "apps.hexactf.io/managed-by")
			assert.Equal(t, "challenge-operator", labels["apps.hexactf.io/managed-by"])

			// selector가 prefix를 포함하는지 검증
			selector := identifier.GetSelector()
			assert.Equal(t, tt.wantPrefix, selector["apps.hexactf.io/instance"])
		})
	}
}
