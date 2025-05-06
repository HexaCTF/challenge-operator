package controller

import (
	"context"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	"github.com/hexactf/challenge-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	labelInjector.InjectChallengeLabels(challenge)
	if err := r.Update(ctx, challenge); err != nil {
		log.Error(err, "Failed to update Challenge", "challenge", challenge.Name)
		return err
	}

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
	podName := challenge.Labels["apps.hexactf.io/podName"]
	pod.Namespace = challenge.Namespace
	pod.Name = podName

	if err := r.Get(ctx, client.ObjectKey{
		Name:      podName,
		Namespace: challenge.Namespace,
	}, pod); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

	if err := ctrl.SetControllerReference(challenge, pod, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference for Pod")
		return err
	}

	if err := r.Client.Create(ctx, pod); err != nil {
		log.Error(err, "Failed to create Pod", "podName", pod.Name)
		return err
	}

	challenge.Status.CurrentStatus.Pending()
	return nil
}

func (r *ChallengeReconciler) loadService(ctx context.Context, challenge *hexactfproj.Challenge, service *corev1.Service) error {
	serviceName := challenge.Labels["apps.hexactf.io/svcName"]
	service.Namespace = challenge.Namespace
	service.Name = serviceName

	if err := r.Get(ctx, client.ObjectKey{
		Name:      serviceName,
		Namespace: challenge.Namespace,
	}, service); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}

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
	return nil
}

func (r *ChallengeReconciler) getChallengeDefinition(ctx context.Context, challenge *hexactfproj.Challenge) (*hexactfproj.ChallengeDefinition, error) {
	definition := &hexactfproj.ChallengeDefinition{}
	err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      challenge.Spec.Definition,
	}, definition)
	if err != nil {
		return nil, err
	}
	return definition, nil
}
