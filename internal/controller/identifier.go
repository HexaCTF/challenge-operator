package controller

import (
	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
)

type ChallengeIdentifier struct {
	prefix string
	labels map[string]string
}

func NewChallengeIdentifier(challenge *hexactfproj.Challenge, component hexactfproj.Component) *ChallengeIdentifier {
	return &ChallengeIdentifier{
		prefix: challenge.Labels["hexactf.io/challengeId"] + "-" + component.Name + "-" + challenge.Labels["hexactf.io/user"],
		labels: map[string]string{
			"hexactf.io/user":        challenge.Labels["hexactf.io/user"],
			"hexactf.io/challengeId": challenge.Labels["hexactf.io/challengeId"],
			"hexactf.io/component":   component.Name,
		},
	}
}

func (c *ChallengeIdentifier) GetLabels() map[string]string {
	return c.labels
}

func (c *ChallengeIdentifier) GetDeploymentPrefix() string {
	return c.prefix + "-deplioy"
}
