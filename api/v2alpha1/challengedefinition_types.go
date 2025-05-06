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

// ContainerConfig defines the configuration for a container
type ContainerConfig struct {
	// Name of the container
	Name string `json:"name"`

	// Container image to use
	Image string `json:"image"`

	// Command to run in the container
	Command []string `json:"command,omitempty"`

	// Arguments to the command
	Args []string `json:"args,omitempty"`

	// Container ports to expose
	Ports []ContainerPort `json:"ports,omitempty"`

	// Resource requirements
	Resources ResourceRequirements `json:"resources,omitempty"`
}

// ContainerPort represents a port to expose from the container
type ContainerPort struct {
	// Port number to expose
	ContainerPort int32 `json:"containerPort"`

	// Protocol for the port (TCP/UDP)
	Protocol string `json:"protocol,omitempty"`
}

// ResourceRequirements describes the compute resource requirements
type ResourceRequirements struct {
	// Limits describes the maximum amount of compute resources allowed
	Limits ResourceList `json:"limits,omitempty"`

	// Requests describes the minimum amount of compute resources required
	Requests ResourceList `json:"requests,omitempty"`
}

// ResourceList is a map of resource names to resource quantities
type ResourceList struct {
	// CPU resource limit/request
	CPU string `json:"cpu,omitempty"`

	// Memory resource limit/request
	Memory string `json:"memory,omitempty"`
}

// Resource 는 ChallengeDefinition이 생성하는 리소스를 정의
// Service와 Pod를 포함한다.
type Resource struct {
	Name    string          `json:"name,omitempty"`
	Service *corev1.Service `json:"service,omitempty"`
	Pod     *PodConfig      `json:"pod,omitempty"`
}

// PodConfig defines the configuration for a pod
type PodConfig struct {
	// Container configuration
	Containers []ContainerConfig `json:"containers"`
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
// +kubebuilder:storageversion

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
