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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChallengeSpec defines the desired state of Challenge.
type ChallengeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Definition: ChallengeDefinition 이름
	// ChallengeDefinition을 참조하여 실행됩니다.
	Definition string `json:"definition,omitempty"`
}

// ChallengeStatus defines the observed state of Challenge.
type ChallengeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// StartedAt: Challenge 시작 시간
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// CurrentStatus: Challenge 현재 상태
	CurrentStatus CurrentStatus `json:"currentStatus,omitempty"`

	Endpoint int `json:"endpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// Challenge is the Schema for the challenges API.
type Challenge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChallengeSpec   `json:"spec,omitempty"`
	Status ChallengeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChallengeList contains a list of Challenge.
type ChallengeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Challenge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Challenge{}, &ChallengeList{})
}
