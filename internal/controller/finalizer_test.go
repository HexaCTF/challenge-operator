package controller

import (
	"context"
	"fmt"
	"time"

	hexactfproj "github.com/hexactf/challenge-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("ChallengeReconciler Finalizer", func() {
	var (
		reconciler *ChallengeReconciler
		challenge  *hexactfproj.Challenge
	)

	BeforeEach(func() {
		ctx = context.Background()
		reconciler = &ChallengeReconciler{Client: k8sClient}
		challenge = &hexactfproj.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-challenge",
				Namespace: "default",
			},
		}
		err := k8sClient.Create(ctx, challenge)
		if apierrors.IsConflict(err) {
			err := k8sClient.Delete(ctx, challenge)
			if err != nil && !apierrors.IsNotFound(err) {
				Fail(fmt.Sprintf("Failed to delete challenge after test: %v", err))
			}
			time.Sleep(1 * time.Second)
			err = k8sClient.Create(ctx, challenge)
			if err != nil {
				Fail(fmt.Sprintf("Failed to create challenge after test: %v", err))
			}
		}
	})

	AfterEach(func() {
		By("cleaning up test challenge")
		err := k8sClient.Delete(ctx, challenge)
		if err != nil && !apierrors.IsNotFound(err) {
			Fail(fmt.Sprintf("Failed to delete challenge after test: %v", err))
		}
		time.Sleep(1 * time.Second)
	})

	Describe("Adding a Finalizer", func() {
		It("should successfully add a finalizer", func() {
			latestChallenge := &hexactfproj.Challenge{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(challenge), latestChallenge)
			Expect(err).NotTo(HaveOccurred())

			result, err := reconciler.addFinalizer(ctx, latestChallenge)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeZero())

			updatedChallenge := &hexactfproj.Challenge{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(challenge), updatedChallenge)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChallenge.Finalizers).To(ContainElement(challengeFinalizer))
		})
	})

	Describe("Removing a Finalizer", func() {
		BeforeEach(func() {
			latestChallenge := &hexactfproj.Challenge{}
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(challenge), latestChallenge)
			Expect(err).NotTo(HaveOccurred())

			latestChallenge.Finalizers = []string{challengeFinalizer}
			err = k8sClient.Update(ctx, latestChallenge)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully remove a finalizer", func() {
			err := reconciler.removeFinalizer(ctx, challenge)
			Expect(err).NotTo(HaveOccurred())

			updatedChallenge := &hexactfproj.Challenge{}
			err = k8sClient.Get(ctx, client.ObjectKeyFromObject(challenge), updatedChallenge)

			if apierrors.IsNotFound(err) {
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedChallenge.Finalizers).NotTo(ContainElement(challengeFinalizer))
		})
	})
})
