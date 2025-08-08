package utils

import (
	"testing"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
)

func TestNameBuilder_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		challengeLabels map[string]string
		wantPrefix      string
		wantServiceName string
		wantPodName     string
	}{
		{
			name: "기본 케이스",
			challengeLabels: map[string]string{
				"apps.hexactf.io/challengeId": "123",
				"apps.hexactf.io/userId":      "456",
			},
			wantPrefix:      "challenge-123-456",
			wantServiceName: "challenge-123-456-svc",
			wantPodName:     "challenge-123-456-pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			challenge := &hexactfproj.Challenge{
				// 필요한 경우 다른 필드도 추가
			}
			challenge.Labels = tt.challengeLabels

			nb := NewNameBuilder(challenge)

			if nb.prefix != tt.wantPrefix {
				t.Errorf("prefix: got %s, want %s", nb.prefix, tt.wantPrefix)
			}
			if nb.GetServiceName() != tt.wantServiceName {
				t.Errorf("GetServiceName(): got %s, want %s", nb.GetServiceName(), tt.wantServiceName)
			}
			if nb.GetPodName() != tt.wantPodName {
				t.Errorf("GetPodName(): got %s, want %s", nb.GetPodName(), tt.wantPodName)
			}
		})
	}
}
