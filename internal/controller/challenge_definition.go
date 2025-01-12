package controller

import (
	"context"
	"fmt"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ChallengeReconciler) loadChallengeDefinition(ctx context.Context, challenge *hexactfproj.Challenge) error {
	definition, err := r.getChallengeDefinition(ctx, challenge)
	if err != nil {
		log.Error(err, "failed to get ChallengeDefinition %s", challenge.Spec.Definition)
		return err

	}

	// IsOne 설정
	challenge.Status.IsOne = definition.Spec.IsOne
	// Update the status in the cluster
	if err := r.Status().Update(ctx, challenge); err != nil {
		log.Error(err, "failed to update Challenge status")
		return err
	}

	for _, component := range definition.Spec.Components {
		identifier := NewChallengeIdentifier(challenge, component)

		err = r.loadDeployment(ctx, challenge, component, identifier)
		if err != nil {
			log.Error(err, "failed to load Deployment")
			return err
		}

		err = r.loadService(ctx, challenge, component, identifier)
		if err != nil {
			log.Error(err, "failed to load Service")
			return err
		}
	}

	// 메세지 전송
	err = r.KafkaClient.SendStatusChange(challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"], "Creating")
	if err != nil {
		log.Error(err, "Failed to send status change message")
		return err
	}

	return nil
}

// GetChallengeDefinition ChallengeDefinition 리소스를 로드
func (r *ChallengeReconciler) getChallengeDefinition(ctx context.Context, challenge *hexactfproj.Challenge) (*hexactfproj.ChallengeDefinition, error) {
	var definition hexactfproj.ChallengeDefinition
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      challenge.Spec.Definition,
	}, &definition); err != nil {
		log.Error(err, "failed to get ChallengeDefinition")
		return nil, fmt.Errorf("failed to load definition %s: %w", challenge.Spec.Definition, err)
	}
	return &definition, nil
}

func (r *ChallengeReconciler) loadDeployment(ctx context.Context, challenge *hexactfproj.Challenge, component hexactfproj.Component, identifier *ChallengeIdentifier) error {

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identifier.GetDeploymentPrefix(),
			Namespace: challenge.Namespace,
			Labels:    identifier.GetLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &component.Deployment.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: identifier.GetSelector(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: identifier.GetLabels(),
				},
				Spec: corev1.PodSpec{
					Containers: component.Deployment.Spec.Template.Spec.Containers,
				},
			},
		},
	}

	// Deployment가 존재하는지 확인
	err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      deploy.Name,
	}, deploy)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Deployment", "Deployment.Namespace", deploy.Namespace, "Deployment.Name", deploy.Name)

			// Owner Reference 설정
			if err := ctrl.SetControllerReference(challenge, deploy, r.Scheme); err != nil {
				log.Error(err, "failed to set controller reference")
				return err
			}

			err = r.Client.Create(ctx, deploy)
			if err != nil {
				log.Error(err, "failed to create Deployment")
				return err
			}
		} else {
			log.Error(err, "failed to get Deployment")
			return err
		}
	}

	return nil
}

// LoadService Service 리소스 생성
// TODO(Addition): 랜덤으로 NodePort를 할당
func (r *ChallengeReconciler) loadService(ctx context.Context, challenge *hexactfproj.Challenge,
	component hexactfproj.Component, identifier *ChallengeIdentifier) error {

	log.Info("Loading service",
		"challenge", challenge.Name,
		"component", component.Name,
		"prefix", identifier.GetServicePrefix())

	// Service가 nil이면 처리하지 않음
	if component.Service == nil {
		log.Info("No service defined for component",
			"component", component.Name)
		return nil
	}

	// 새로운 Service 객체 생성
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identifier.GetServicePrefix(),
			Namespace: challenge.Namespace,
			Labels:    identifier.GetLabels(),
		},
		Spec: corev1.ServiceSpec{
			Selector: identifier.GetSelector(),
			Ports:    component.Service.Spec.Ports,
			Type:     component.Service.Spec.Type,
		},
	}

	// Service가 존재하는지 확인
	err := r.Get(ctx, types.NamespacedName{
		Name:      identifier.GetServicePrefix(),
		Namespace: challenge.Namespace,
	}, service)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating new service",
				"name", identifier.GetServicePrefix(),
				"namespace", challenge.Namespace)

			// Owner Reference 설정
			if err := ctrl.SetControllerReference(challenge, service, r.Scheme); err != nil {
				log.Error(err, "Failed to set controller reference for Service")
				return err
			}

			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create Service")
				return err
			}

			log.Info("Successfully created service",
				"name", identifier.GetServicePrefix())

			// NodePort 확인하기
			// Service 생성 후 해당 객체를 다시 조회하여 NodePort 확인
			// createdService := &corev1.Service{}
			// if err := r.Get(ctx, types.NamespacedName{
			// 	Name:      identifier.GetServicePrefix(),
			// 	Namespace: challenge.Namespace,
			// }, createdService); err != nil {
			// 	log.Error(err, "Failed to get created Service")
			// 	return err
			// }

			// // NodePort 타입인 경우 포트 번호 로깅
			// // !NOTICE NodePort가 하나인 경우만 가정하고 구현
			// if createdService.Spec.Type == corev1.ServiceTypeNodePort {
			// 	challenge.Status.Endpoint = int(createdService.Spec.Ports[0].NodePort)
			// 	log.Info("NodePort created",
			// 		"port", createdService.Spec.Ports[0].NodePort)
			// }
			// if err := r.Status().Update(ctx, challenge); err != nil {
			// 	log.Error(err, "Failed to update Challenge status with NodePort information")
			// 	return err
			// }

			if service.Spec.Type == corev1.ServiceTypeNodePort {
				challenge.Status.Endpoint = int(service.Spec.Ports[0].NodePort)
				log.Info("NodePort created",
					"port", service.Spec.Ports[0].NodePort)
			}

			if err := r.Status().Update(ctx, challenge); err != nil {
				log.Error(err, "Failed to update Challenge status with NodePort information")
				return err
			}

			return nil
		}
		log.Error(err, "Failed to get Service")
		return err
	}

	return nil
}
