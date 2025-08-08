package controller

import (
	"context"

	hexactfproj "github.com/hexactf/challenge-operator/api/v2alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Finalizer Operations", func() {
	var (
		s *runtime.Scheme
	)

	BeforeEach(func() {
		// 스키마 설정
		s = runtime.NewScheme()
		_ = scheme.AddToScheme(s)
		_ = hexactfproj.AddToScheme(s)
	})

	// 테스트용 Challenge 생성 헬퍼 함수
	createTestChallenge := func(name string, finalizers []string) *hexactfproj.Challenge {
		return &hexactfproj.Challenge{
			ObjectMeta: metav1.ObjectMeta{
				Name:       name,
				Namespace:  "default",
				Finalizers: finalizers,
			},
		}
	}

	Describe("AddFinalizer", func() {
		Context("when finalizer is added successfully", func() {
			It("should add finalizer successfully", func() {
				// Given
				challenge := createTestChallenge("test-1", nil)

				// fake client 생성
				client := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(challenge).
					Build()

				r := &ChallengeReconciler{Client: client}

				// When
				_, err := r.addFinalizer(context.Background(), challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Challenge 다시 가져와서 finalizer 확인
				updated := &hexactfproj.Challenge{}
				err = client.Get(context.Background(),
					types.NamespacedName{
						Name:      challenge.Name,
						Namespace: challenge.Namespace,
					},
					updated)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Finalizers).To(ContainElement(challengeFinalizer))
			})
		})

		Context("when finalizer already exists", func() {
			It("should handle existing finalizer", func() {
				// Given
				challenge := createTestChallenge("test-2", []string{challengeFinalizer})

				// fake client 생성
				client := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(challenge).
					Build()

				r := &ChallengeReconciler{Client: client}

				// When
				_, err := r.addFinalizer(context.Background(), challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Challenge 다시 가져와서 finalizer 확인
				updated := &hexactfproj.Challenge{}
				err = client.Get(context.Background(),
					types.NamespacedName{
						Name:      challenge.Name,
						Namespace: challenge.Namespace,
					},
					updated)
				Expect(err).NotTo(HaveOccurred())
				Expect(updated.Finalizers).To(ContainElement(challengeFinalizer))
			})
		})
	})

	Describe("RemoveFinalizer", func() {
		Context("when finalizer is removed successfully", func() {
			It("should remove finalizer successfully", func() {
				// Given
				challenge := createTestChallenge("test-3", []string{challengeFinalizer})

				// fake client 생성
				client := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(challenge).
					Build()

				r := &ChallengeReconciler{Client: client}

				// When
				err := r.removeFinalizer(context.Background(), challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Challenge가 존재하는 경우 finalizer 확인
				updated := &hexactfproj.Challenge{}
				err = client.Get(context.Background(),
					types.NamespacedName{
						Name:      challenge.Name,
						Namespace: challenge.Namespace,
					},
					updated)

				if err == nil {
					Expect(updated.Finalizers).NotTo(ContainElement(challengeFinalizer))
				}
			})
		})

		Context("when finalizer does not exist", func() {
			It("should handle missing finalizer", func() {
				// Given
				challenge := createTestChallenge("test-4", nil)

				// fake client 생성
				client := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(challenge).
					Build()

				r := &ChallengeReconciler{Client: client}

				// When
				err := r.removeFinalizer(context.Background(), challenge)

				// Then
				Expect(err).NotTo(HaveOccurred())

				// Challenge가 존재하는 경우 finalizer 확인
				updated := &hexactfproj.Challenge{}
				err = client.Get(context.Background(),
					types.NamespacedName{
						Name:      challenge.Name,
						Namespace: challenge.Namespace,
					},
					updated)

				if err == nil {
					Expect(updated.Finalizers).NotTo(ContainElement(challengeFinalizer))
				}
			})
		})
	})
})

var _ = Describe("Finalizer Helper Functions", func() {
	Describe("containsString", func() {
		Context("with empty slice", func() {
			It("should return false", func() {
				// Given
				slice := []string{}

				// When
				result := containsString(slice)

				// Then
				Expect(result).To(BeFalse())
			})
		})

		Context("with finalizer included", func() {
			It("should return true", func() {
				// Given
				slice := []string{challengeFinalizer, "other-finalizer"}

				// When
				result := containsString(slice)

				// Then
				Expect(result).To(BeTrue())
			})
		})

		Context("with finalizer not included", func() {
			It("should return false", func() {
				// Given
				slice := []string{"other-finalizer"}

				// When
				result := containsString(slice)

				// Then
				Expect(result).To(BeFalse())
			})
		})
	})

	Describe("removeString", func() {
		Context("with empty slice", func() {
			It("should return nil", func() {
				// Given
				slice := []string{}

				// When
				result := removeString(slice)

				// Then
				Expect(result).To(BeNil())
			})
		})

		Context("with only finalizer", func() {
			It("should return nil", func() {
				// Given
				slice := []string{challengeFinalizer}

				// When
				result := removeString(slice)

				// Then
				Expect(result).To(BeNil())
			})
		})

		Context("with multiple finalizers", func() {
			It("should return other finalizers", func() {
				// Given
				slice := []string{challengeFinalizer, "other-finalizer"}

				// When
				result := removeString(slice)

				// Then
				Expect(result).To(Equal([]string{"other-finalizer"}))
			})
		})

		Context("with other finalizers only", func() {
			It("should return unchanged slice", func() {
				// Given
				slice := []string{"other-finalizer"}

				// When
				result := removeString(slice)

				// Then
				Expect(result).To(Equal([]string{"other-finalizer"}))
			})
		})
	})
})
