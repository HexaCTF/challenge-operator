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
// +kubebuilder:object:generate=true
type PodSpecTemplate struct {
	Containers []corev1.Container `json:"containers,omitempty"`
}

// +kubebuilder:object:generate=true
type ServiceSpecTemplate struct {
	Type  corev1.ServiceType   `json:"type,omitempty"`
	Ports []corev1.ServicePort `json:"ports,omitempty"`
}

// +kubebuilder:object:generate=true
type ChallengeTemplateResources struct {
	Pod     PodSpecTemplate     `json:"pod,omitempty"`
	Service ServiceSpecTemplate `json:"service,omitempty"`
}

// ChallengeTemplateSpec defines the desired state of ChallengeTemplate.
type ChallengeTemplateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type: Challenge types
	// example: web, pwn, crypto
	Type string `json:"type"`
	// Resources: Define K8s Resources
	Resources ChallengeTemplateResources `json:"resources"`
}

// ChallengeTemplateStatus defines the observed state of ChallengeTemplate.
type ChallengeTemplateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ChallengeTemplate is the Schema for the challengetemplates API.
type ChallengeTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChallengeTemplateSpec   `json:"spec,omitempty"`
	Status ChallengeTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChallengeTemplateList contains a list of ChallengeTemplate.
type ChallengeTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ChallengeTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChallengeTemplate{}, &ChallengeTemplateList{})
}
