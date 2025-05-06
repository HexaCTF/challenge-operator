package utils

import (
	"fmt"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
)

type NameBuilder struct {
	prefix string
}

func NewNameBuilder(challenge *hexactfproj.Challenge) *NameBuilder {
	// prefix 생성 (리소스 이름에 사용)
	prefix := fmt.Sprintf("challenge-%s-%s",
		challenge.Labels["apps.hexactf.io/challengeId"],
		challenge.Labels["apps.hexactf.io/userId"])

	return &NameBuilder{
		prefix: prefix,
	}
}

func (n *NameBuilder) GetServiceName() string {
	return fmt.Sprintf("%s-svc", n.prefix)
}

func (n *NameBuilder) GetPodName() string {
	return fmt.Sprintf("%s-pod", n.prefix)
}
