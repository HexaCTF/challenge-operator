package controller

import (
	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
)

type ChallengeIdentifier struct {
	prefix     string
	matchlabel map[string]string
	labels     map[string]string
}

func NewChallengeIdentifier(challenge *hexactfproj.Challenge, component hexactfproj.Component) *ChallengeIdentifier {
	label := challenge.Labels["hexactf.io/challengeId"] + "-" + component.Name + "-" + challenge.Labels["hexactf.io/user"]
	return &ChallengeIdentifier{
		prefix: label,
		matchlabel: map[string]string{
			"hexactf.io/label": label,
		},
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

func (c *ChallengeIdentifier) GetServicePrefix() string {
	return c.prefix + "-service"
}

func (c *ChallengeIdentifier) GetMatchLabels() map[string]string {
	return c.matchlabel
}
