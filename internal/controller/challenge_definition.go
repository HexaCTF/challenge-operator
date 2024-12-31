package controller

import (
	"context"
	"fmt"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ChallengeReconciler) LoadChallengeDefinition(ctx context.Context, challenge *hexactfproj.Challenge) error {
	definition, err := r.GetChallengeDefinition(ctx, challenge)
	if err != nil {
		log.Error(err, "failed to get ChallengeDefinition %s", challenge.Spec.Definition)
		return err
		}
		return r.Client.Create(ctx, deploy)
	}

	for _, component := range definition.Spec.Components {
		identifier := NewChallengeIdentifier(challenge, component)

		err := r.LoadDeployment(ctx, challenge, component, identifier)
		if err != nil {
			log.Error(err, "failed to load Deployment %s", identifier.GetPrefix())
			return err
		}

	}

}

// GetChallengeDefinition ChallengeDefinition 리소스를 로드
func (r *ChallengeReconciler) GetChallengeDefinition(ctx context.Context, challenge *hexactfproj.Challenge) (*hexactfproj.ChallengeDefinition, error) {
	var definition hexactfproj.ChallengeDefinition
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: challenge.Namespace,
		Name:      challenge.Spec.Definition,
	}, &definition); err != nil {
		log.Error(err, "failed to get ChallengeDefinition %s", challenge.Spec.Definition)
		return nil, fmt.Errorf("failed to load definition %s: %w", challenge.Spec.Definition, err)
	}
	return &definition, nil
}

func (r *ChallengeReconciler) LoadDeployment(ctx context.Context, challenge *hexactfproj.Challenge, component hexactfproj.Component, identifier *ChallengeIdentifier) error {

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      identifier.GetDeploymentPrefix(),
			Namespace: challenge.Namespace,
			Labels:    identifier.GetLabels(),
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

			deploy.Spec = *component.Deployment.Spec

			// Owner Reference 설정
			if err := ctrl.SetControllerReference(challenge, deploy, r.Scheme); err != nil {
				log.Error(err, "failed to set controller reference")
				return err
			}

			err = r.Client.Create(ctx, deploy)
			if err != nil {
				log.Error(err, "failed to create Deployment %s", deploy.Name)
				return err
			}
		} else {
			log.Error(err, "failed to get Deployment %s", deploy.Name)
			return err
		}
	}

	return nil

}
