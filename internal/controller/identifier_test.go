package controller

import (
	"github.com/hexactf/challenge-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ChallengeIdentifier", func() {
	var (
		challenge  *v1alpha1.Challenge
		component  v1alpha1.Component
		identifier *ChallengeIdentifier
	)

	BeforeEach(func() {
		challenge = &v1alpha1.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-challenge",
				Labels: map[string]string{
					"apps.hexactf.io/challengeId": "test-id",
					"apps.hexactf.io/user":        "test-user",
				},
			},
		}

		component = v1alpha1.Component{
			Name: "ubuntu",
		}

		identifier = NewChallengeIdentifier(challenge, component)
	})

	Describe("Identifier Generation", func() {
		It("should generate the correct prefix", func() {
			Expect(identifier.GetDeploymentPrefix()).To(Equal("chall-test-id-ubuntu-test-user-deploy"))
			Expect(identifier.GetServicePrefix()).To(Equal("chall-test-id-ubuntu-test-user-svc"))
		})

		It("should generate the correct labels", func() {
			expectedLabels := map[string]string{
				"apps.hexactf.io/instance":   "chall-test-id-ubuntu-test-user",
				"apps.hexactf.io/name":       "ubuntu",
				"apps.hexactf.io/part-of":    "test-challenge",
				"apps.hexactf.io/managed-by": "challenge-operator",
			}
			Expect(identifier.GetLabels()).To(Equal(expectedLabels))
		})

		It("should generate the correct selector", func() {
			expectedSelector := map[string]string{
				"apps.hexactf.io/instance": "chall-test-id-ubuntu-test-user",
			}
			Expect(identifier.GetSelector()).To(Equal(expectedSelector))
		})
	})

	DescribeTable("Edge Cases",
		func(challenge *v1alpha1.Challenge, component v1alpha1.Component, wantPrefix string) {
			identifier := NewChallengeIdentifier(challenge, component)
			Expect(identifier.GetDeploymentPrefix()).To(Equal(wantPrefix + "-deploy"))
			Expect(identifier.GetServicePrefix()).To(Equal(wantPrefix + "-svc"))
			Expect(identifier.GetSelector()["apps.hexactf.io/instance"]).To(Equal(wantPrefix))
			Expect(identifier.GetLabels()["apps.hexactf.io/managed-by"]).To(Equal("challenge-operator"))
		},

		Entry("Empty Labels", &v1alpha1.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-challenge",
				Labels: map[string]string{},
			},
		}, v1alpha1.Component{Name: "ubuntu"}, "chall--ubuntu-"),

		Entry("Long Names", &v1alpha1.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-challenge",
				Labels: map[string]string{
					"apps.hexactf.io/challengeId": "very-long-challenge-id",
					"apps.hexactf.io/user":        "very-long-user-name",
				},
			},
		}, v1alpha1.Component{Name: "very-long-component-name"}, "chall-very-long-challenge-id-very-long-component-name-very-long-user-name"),

		Entry("Special Characters", &v1alpha1.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-challenge",
				Labels: map[string]string{
					"apps.hexactf.io/challengeId": "test@id",
					"apps.hexactf.io/user":        "user#1",
				},
			},
		}, v1alpha1.Component{Name: "test!comp"}, "chall-test@id-test!comp-user#1"),
	)
})
