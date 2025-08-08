package utils

import (
	"fmt"

	"github.com/hexactf/challenge-operator/api/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			"apps.hexactf.io/podName":       fmt.Sprintf("challenge-%s-%s-pod", challengeId, userId),
			"apps.hexactf.io/svcName":       fmt.Sprintf("challenge-%s-%s-svc", challengeId, userId),
		},
	}
}

func (li LabelInjector) InjectChallengeLabels(challenge *v2alpha1.Challenge) {
	challenge.Labels = li.labels
}

func (li *LabelInjector) InjectPod(pod *v2alpha1.PodConfig) *corev1.Pod {
	// Create a new Pod object
	newPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("challenge-%s-%s-pod", li.challengeId, li.userId),
			Labels: li.labels,
			Annotations: map[string]string{
				"apps.hexactf.io/createdBy": "challenge-operator",
			},
		},
		Spec: corev1.PodSpec{
			Containers: make([]corev1.Container, len(pod.Containers)),
		},
	}

	// Convert each container config to corev1.Container
	for i, container := range pod.Containers {
		newPod.Spec.Containers[i] = corev1.Container{
			Name:    container.Name,
			Image:   container.Image,
			Command: container.Command,
			Args:    container.Args,
			Ports:   make([]corev1.ContainerPort, len(container.Ports)),
			Resources: corev1.ResourceRequirements{
				Limits:   make(corev1.ResourceList),
				Requests: make(corev1.ResourceList),
			},
		}

		// Set resource limits if they are not empty
		if container.Resources.Limits.CPU != "" {
			newPod.Spec.Containers[i].Resources.Limits[corev1.ResourceCPU] = resource.MustParse(container.Resources.Limits.CPU)
		}
		if container.Resources.Limits.Memory != "" {
			newPod.Spec.Containers[i].Resources.Limits[corev1.ResourceMemory] = resource.MustParse(container.Resources.Limits.Memory)
		}

		// Set resource requests if they are not empty
		if container.Resources.Requests.CPU != "" {
			newPod.Spec.Containers[i].Resources.Requests[corev1.ResourceCPU] = resource.MustParse(container.Resources.Requests.CPU)
		}
		if container.Resources.Requests.Memory != "" {
			newPod.Spec.Containers[i].Resources.Requests[corev1.ResourceMemory] = resource.MustParse(container.Resources.Requests.Memory)
		}

		// Convert ports
		for j, port := range container.Ports {
			newPod.Spec.Containers[i].Ports[j] = corev1.ContainerPort{
				ContainerPort: port.ContainerPort,
				Protocol:      corev1.Protocol(port.Protocol),
			}
		}
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
