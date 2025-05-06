package utils

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

type LabelInjector struct {
	challengeId string
	userId      string
	labels      map[string]string
}

func NewLabelInjector(challengeId, userId string) *LabelInjector {
	return &LabelInjector{
		challengeId: challengeId,
		userId:      userId,
		labels: map[string]string{
			"apps.hexactf.io/challengeId":   challengeId,
			"apps.hexactf.io/userId":        userId,
			"apps.hexactf.io/createdBy":     "challenge-operator",
			"apps.hexactf.io/challengeName": fmt.Sprintf("challenge-%s-%s", challengeId, userId),
		},
	}
}

func (li *LabelInjector) InjectPod(pod *corev1.Pod) *corev1.Pod {

	// 원본 객체 훼손 방지를 위한 DeepCopy
	newPod := pod.DeepCopy()

	newPod.Name = fmt.Sprintf("challenge-%s-%s-pod", li.challengeId, li.userId)
	newPod.Labels = li.labels
	newPod.Annotations = map[string]string{
		"apps.hexactf.io/createdBy": "challenge-operator",
	}
	return newPod

}

func (li *LabelInjector) InjectService(service *corev1.Service) *corev1.Service {

	newService := service.DeepCopy()

	newService.Name = fmt.Sprintf("challenge-%s-%s-svc", li.challengeId, li.userId)
	newService.Labels = li.labels
	newService.Annotations = map[string]string{
		"apps.hexactf.io/createdBy": "challenge-operator",
	}
	newService.Spec.Selector = map[string]string{
		"apps.hexactf.io/challengeName": li.labels["apps.hexactf.io/challengeName"],
	}

	return newService

}
