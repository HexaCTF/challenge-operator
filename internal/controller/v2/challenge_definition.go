package controller

import (
	"context"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	"github.com/hexactf/challenge-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ChallengeReconciler) loadChallengeDefinition(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
	definition, err := r.getChallengeDefinition(ctx, challenge)
	if err != nil {
		log.Error(err, "Failed to get ChallengeDefinition", "definition", challenge.Spec.Definition)
		return err
	}

	labelInjector := utils.NewLabelInjector(
		challenge.Labels["apps.hexactf.io/challengeId"],
		challenge.Labels["apps.hexactf.io/userId"],
	)

	pod := labelInjector.InjectPod(definition.Spec.Resource.Pod)
	if err := r.loadPod(ctx, challenge, pod); err != nil {
		log.Error(err, "Failed to load Pod", "pod", pod.Name)
		return err
	}

	service := labelInjector.InjectService(definition.Spec.Resource.Service)
	if err := r.loadService(ctx, challenge, service); err != nil {
		log.Error(err, "Failed to load Service", "service", service.Name)
		return err
	}

	return nil
}

func (r *ChallengeReconciler) loadPod(ctx context.Context, challenge *hexactfproj.Challenge, pod *corev1.Pod) error {
	podList := &corev1.PodList{}
	err := r.List(ctx, podList, &client.ListOptions{
		Namespace: CHALLENGE_NAMESPACE,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"apps.hexactf.io/challengeId": challenge.Labels["apps.hexactf.io/challengeId"],
			"apps.hexactf.io/userId":      challenge.Labels["apps.hexactf.io/userId"],
		}),
	})
	if err != nil {
		log.Error(err, "Failed to list Pods", "podName", pod.Name)
		return err
	}

	if len(podList.Items) == 0 {
		if err := ctrl.SetControllerReference(challenge, pod, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for Service")
			return err
		}

		// Create a new Pod
		if err := r.Client.Create(ctx, pod); err != nil {
			log.Error(err, "Failed to create Pod", "podName", pod.Name)
			return err
		}

		// Pod가 생성되면 Challenge 상태를 업데이트
		challenge.Status.CurrentStatus.Pending()
	}
	// TODO: pod가 여러개인 경우도 고려해보기

	return nil
}

func (r *ChallengeReconciler) loadService(ctx context.Context, challenge *hexactfproj.Challenge, service *corev1.Service) error {
	serviceList := &corev1.ServiceList{}
	err := r.List(ctx, serviceList, &client.ListOptions{
		Namespace: CHALLENGE_NAMESPACE,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"apps.hexactf.io/challengeId": challenge.Labels["apps.hexactf.io/challengeId"],
			"apps.hexactf.io/userId":      challenge.Labels["apps.hexactf.io/userId"],
		}),
	})
	if err != nil {
		log.Error(err, "Failed to list Services", "serviceName", service.Name)
	}

	if len(serviceList.Items) == 0 {
		// Create a new Service
		if err := ctrl.SetControllerReference(challenge, service, r.Scheme); err != nil {
			log.Error(err, "Failed to set controller reference for Service")
			return err
		}

		if err := r.Client.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service", "serviceName", service.Name)
			return err
		}

		if service.Spec.Type == corev1.ServiceTypeNodePort {
			challenge.Status.Endpoint = int(service.Spec.Ports[0].NodePort)

			if err := r.Status().Update(ctx, challenge); err != nil {
				log.Error(err, "Failed to update Challenge status")
				return err
			}
		}
	}

	return nil
}

func (r *ChallengeReconciler) getChallengeDefinition(ctx context.Context, challenge *hexactfproj.Challenge) (*hexactfproj.ChallengeDefinition, error) {
	// Fetch the ChallengeDefinition
	definition := &hexactfproj.ChallengeDefinition{}
	err := r.Get(ctx, client.ObjectKey{
		Name: challenge.Spec.Definition,
	}, definition)
	if err != nil {
		return nil, err
	}

	return definition, nil
}
