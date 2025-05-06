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

package v2alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Resource 는 ChallengeDefinition이 생성하는 리소스를 정의
// Service와 Pod를 포함한다.
type Resource struct {
	Name    string          `json:"name,omitempty"`
	Service *corev1.Service `json:"service,omitempty"`
	Pod     *corev1.Pod     `json:"pod,omitempty"`
}

// ChallengeDefinitionSpec defines the desired state of ChallengeDefinition.
type ChallengeDefinitionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Resource Resource `json:"resource,omitempty"`
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
