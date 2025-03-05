package controller

import (
	"context"
	"fmt"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ChallengeReconciler) loadChallengeDefinition(ctx context.Context, req ctrl.Request, challenge *hexactfproj.Challenge) error {
	// Fetch the challenge definition
	definition, err := r.getChallengeDefinition(ctx, challenge)
	if err != nil {
		log.Error(err, "Failed to get ChallengeDefinition", "definition", challenge.Spec.Definition)
		return err
	}

	for _, component := range definition.Spec.Components {
		identifier := NewChallengeIdentifier(challenge, component)

		if err = r.loadDeployment(ctx, challenge, component, identifier); err != nil {
			log.Error(err, "Failed to load Deployment", "component", component)
			return err
		}

		if err = r.loadService(ctx, challenge, component, identifier); err != nil {
			log.Error(err, "Failed to load Service", "component", component)
			return err
		}
	}

	// Send Kafka message
	user, challengeID := challenge.Labels["apps.hexactf.io/user"], challenge.Labels["apps.hexactf.io/challengeId"]
	if err = r.KafkaClient.SendStatusChange(user, challengeID, "Creating"); err != nil {
		log.Error(err, "Failed to send status change message", "user", user, "challengeID", challengeID)
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
