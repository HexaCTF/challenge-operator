package utils

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewLabelInjector(t *testing.T) {
	tests := []struct {
		name          string
		challengeId   string
		userId        string
		expectedLabel map[string]string
	}{
		{
			name:        "기본 생성 케이스",
			challengeId: "test123",
			userId:      "user456",
			expectedLabel: map[string]string{
				"apps.hexactf.io/challengeId":   "test123",
				"apps.hexactf.io/userId":        "user456",
				"apps.hexactf.io/createdBy":     "challenge-operator",
				"apps.hexactf.io/challengeName": "challenge-test123-user456",
			},
		},
		{
			name:        "빈 문자열 처리",
			challengeId: "",
			userId:      "",
			expectedLabel: map[string]string{
				"apps.hexactf.io/challengeId":   "",
				"apps.hexactf.io/userId":        "",
				"apps.hexactf.io/createdBy":     "challenge-operator",
				"apps.hexactf.io/challengeName": "challenge--",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			li := NewLabelInjector(tt.challengeId, tt.userId)

			for k, v := range tt.expectedLabel {
				if li.labels[k] != v {
					t.Errorf("Label[%s] = %s, want %s", k, li.labels[k], v)
				}
			}
		})
	}
}

func TestInjectPod(t *testing.T) {
	tests := []struct {
		name         string
		inputPod     *corev1.Pod
		challengeId  string
		userId       string
		expectedName string
	}{
		{
			name:         "기본 Pod 주입",
			inputPod:     &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "original-pod"}},
			challengeId:  "c1",
			userId:       "u1",
			expectedName: "challenge-c1-u1-pod",
		},
		{
			name:         "기존 데이터 보존 확인",
			inputPod:     &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"existing": "label"}}},
			challengeId:  "c2",
			userId:       "u2",
			expectedName: "challenge-c2-u2-pod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalPod := tt.inputPod.DeepCopy()
			li := NewLabelInjector(tt.challengeId, tt.userId)

			// 1. 새 객체 반환 검증
			newPod := li.InjectPod(tt.inputPod)

			// 2. 원본 객체 변경 여부 검증
			if originalPod.Name != tt.inputPod.Name {
				t.Error("Original pod should not be modified")
			}

			// 3. 이름 생성 규칙 검증
			if newPod.Name != tt.expectedName {
				t.Errorf("Name = %s, want %s", newPod.Name, tt.expectedName)
			}

			// 4. 라벨 오버라이드 검증
			for k, v := range li.labels {
				if newPod.Labels[k] != v {
					t.Errorf("Label[%s] = %s, want %s", k, newPod.Labels[k], v)
				}
			}

			// 5. 어노테이션 추가 검증
			if newPod.Annotations["apps.hexactf.io/createdBy"] != "challenge-operator" {
				t.Error("Missing createdBy annotation")
			}
		})
	}
}

func TestInjectService(t *testing.T) {
	tests := []struct {
		name             string
		inputService     *corev1.Service
		challengeId      string
		userId           string
		expectedName     string
		expectedSelector string
	}{
		{
			name:             "기본 Service 주입",
			inputService:     &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "original-svc"}},
			challengeId:      "s1",
			userId:           "u1",
			expectedName:     "challenge-s1-u1-svc",
			expectedSelector: "challenge-s1-u1",
		},
		{
			name:             "셀렉터 동작 검증",
			inputService:     &corev1.Service{Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "nginx"}}},
			challengeId:      "s2",
			userId:           "u2",
			expectedSelector: "challenge-s2-u2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalService := tt.inputService.DeepCopy()
			li := NewLabelInjector(tt.challengeId, tt.userId)

			newService := li.InjectService(tt.inputService)

			// 1. 원본 서비스 변경 여부 검증
			if originalService.Name != tt.inputService.Name {
				t.Error("Original service should not be modified")
			}

			// 2. 이름 생성 규칙
			if newService.Name != tt.expectedName {
				t.Errorf("Service name = %s, want %s", newService.Name, tt.expectedName)
			}

			// 3. 셀렉터 변경 검증
			if newService.Spec.Selector["apps.hexactf.io/challengeName"] != tt.expectedSelector {
				t.Errorf("Selector = %s, want %s",
					newService.Spec.Selector["apps.hexactf.io/challengeName"], tt.expectedSelector)
			}
		})
	}
}
