/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChallengeDefinitionSpec defines the desired state of ChallengeDefinition.
type ChallengeDefinitionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// IsOne: 하나만 생성할 경우
	// False일 경우 일정 시간 내에서만 작동된다.
	IsOne bool `json:"isOne,omitempty"`

	// Components: Challenge를 구성하는 컴포넌트들
	Components []Component `json:"components,omitempty"`
}

// Component 는 이름과 리소스를 정의
type Component struct {
	Name       string            `json:"name,omitempty"`
	Deployment *CustomDeployment `json:"deployment,omitempty"`
	Service    *corev1.Service   `json:"service,omitempty"`
}

// Deployment 관련 구조체
// CustomDeploymentSpec 는 Replicas와 Template을 정의
// 자세한 내용은 Kubernetes Deployment API 문서 참고
// https://kubernetes.io/docs/concepts/workloads/controllers/deployment/

type CustomDeployment struct {
	Spec CustomDeploymentSpec `json:"spec,omitempty"`
}

type CustomDeploymentSpec struct {
	Replicas int32                 `json:"replicas,omitempty"`
	Template CustomPodTemplateSpec `json:"template,omitempty"`
}

type CustomPodTemplateSpec struct {
	Spec CustomPodSpec `json:"spec,omitempty"`
}

type CustomPodSpec struct {
	// +optional
	NodeSelector map[string]string  `json:"nodeSelector,omitempty"`
	Containers   []corev1.Container `json:"containers,omitempty"`
}

// ChallengeDefinitionStatus defines the observed state of ChallengeDefinition.
type ChallengeDefinitionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ChallengeDefinition is the Schema for the challengedefinitions API.
type ChallengeDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChallengeDefinitionSpec   `json:"spec,omitempty"`
	Status ChallengeDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChallengeDefinitionList contains a list of ChallengeDefinition.
type ChallengeDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChallengeDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChallengeDefinition{}, &ChallengeDefinitionList{})
}
