package controller

import (
	"fmt"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
)

// ChallengeIdentifier
// 도메인에 맞는 식별자를 생성해주는 구조체
type ChallengeIdentifier struct {
	prefix string
	labels map[string]string
}

func NewChallengeIdentifier(challenge *hexactfproj.Challenge, component hexactfproj.Component) *ChallengeIdentifier {
	// prefix 생성 (리소스 이름에 사용)
	prefix := fmt.Sprintf("chall-%s-%s-%s",
		challenge.Labels["hexactf.io/challengeId"],
		component.Name,
		challenge.Labels["hexactf.io/user"])

	// 단일 레이블 맵 사용
	labels := map[string]string{
		"apps.hexactf.io/instance":   prefix,
		"apps.hexactf.io/name":       component.Name,
		"apps.hexactf.io/part-of":    challenge.Name,
		"apps.hexactf.io/managed-by": "challenge-operator",
	}

	return &ChallengeIdentifier{
		prefix: prefix,
		labels: labels,
	}
}

func (c *ChallengeIdentifier) GetLabels() map[string]string {
	return c.labels
}

// Selector 메서드 추가 - Service와 Deployment의 레이블 매칭에 사용
func (c *ChallengeIdentifier) GetSelector() map[string]string {
	return map[string]string{
		"apps.hexactf.io/instance": c.prefix,
	}
}

func (c *ChallengeIdentifier) GetDeploymentPrefix() string {
	return c.prefix + "-deploy"
}

func (c *ChallengeIdentifier) GetServicePrefix() string {
	return c.prefix + "-svc"
}
