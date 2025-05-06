package utils

import (
	"testing"

	"github.com/hexactf/challenge-operator/api/v2alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNewLabelInjector(t *testing.T) {
	// Given
	challengeId := "test-challenge"
	userId := "test-user"

	// When
	li := NewLabelInjector(challengeId, userId)

	// Then
	assert.NotNil(t, li)
	assert.Equal(t, challengeId, li.challengeId)
	assert.Equal(t, userId, li.userId)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/challengeId":   challengeId,
		"apps.hexactf.io/userId":        userId,
		"apps.hexactf.io/createdBy":     "challenge-operator",
		"apps.hexactf.io/challengeName": "challenge-test-challenge-test-user",
	}, li.labels)
}

func TestInjectPod(t *testing.T) {
	// Given
	li := NewLabelInjector("test-challenge", "test-user")
	podConfig := &v2alpha1.PodConfig{
		Containers: []v2alpha1.ContainerConfig{
			{
				Name:    "test-container",
				Image:   "ubuntu:22.04",
				Command: []string{"/bin/bash", "-c"},
				Args:    []string{"sleep infinity"},
				Ports: []v2alpha1.ContainerPort{
					{
						ContainerPort: 22,
						Protocol:      "TCP",
					},
				},
				Resources: v2alpha1.ResourceRequirements{
					Limits: v2alpha1.ResourceList{
						CPU:    "500m",
						Memory: "512Mi",
					},
					Requests: v2alpha1.ResourceList{
						CPU:    "200m",
						Memory: "256Mi",
					},
				},
			},
		},
	}

	// When
	pod := li.InjectPod(podConfig)

	// Then
	assert.NotNil(t, pod)
	assert.Equal(t, "challenge-test-challenge-test-user-pod", pod.Name)
	assert.Equal(t, li.labels, pod.Labels)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/createdBy": "challenge-operator",
	}, pod.Annotations)

	// Check container configuration
	assert.Len(t, pod.Spec.Containers, 1)
	container := pod.Spec.Containers[0]
	assert.Equal(t, "test-container", container.Name)
	assert.Equal(t, "ubuntu:22.04", container.Image)
	assert.Equal(t, []string{"/bin/bash", "-c"}, container.Command)
	assert.Equal(t, []string{"sleep infinity"}, container.Args)

	// Check ports
	assert.Len(t, container.Ports, 1)
	assert.Equal(t, int32(22), container.Ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, container.Ports[0].Protocol)

	// Check resources
	assert.Equal(t, resource.MustParse("500m"), container.Resources.Limits[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("512Mi"), container.Resources.Limits[corev1.ResourceMemory])
	assert.Equal(t, resource.MustParse("200m"), container.Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("256Mi"), container.Resources.Requests[corev1.ResourceMemory])
}

func TestInjectService(t *testing.T) {
	// Given
	li := NewLabelInjector("test-challenge", "test-user")
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-service",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     22,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// When
	newService := li.InjectService(service)

	// Then
	assert.NotNil(t, newService)
	assert.Equal(t, "challenge-test-challenge-test-user-svc", newService.Name)
	assert.Equal(t, li.labels, newService.Labels)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/createdBy": "challenge-operator",
	}, newService.Annotations)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/challengeName": "challenge-test-challenge-test-user",
	}, newService.Spec.Selector)

	// Check that original service ports are preserved
	assert.Len(t, newService.Spec.Ports, 1)
	assert.Equal(t, int32(22), newService.Spec.Ports[0].Port)
	assert.Equal(t, corev1.ProtocolTCP, newService.Spec.Ports[0].Protocol)
}

func TestInjectPodWithMultipleContainers(t *testing.T) {
	// Given
	li := NewLabelInjector("test-challenge", "test-user")
	podConfig := &v2alpha1.PodConfig{
		Containers: []v2alpha1.ContainerConfig{
			{
				Name:  "container1",
				Image: "ubuntu:22.04",
			},
			{
				Name:  "container2",
				Image: "nginx:latest",
			},
		},
	}

	// When
	pod := li.InjectPod(podConfig)

	// Then
	assert.NotNil(t, pod)
	assert.Len(t, pod.Spec.Containers, 2)
	assert.Equal(t, "container1", pod.Spec.Containers[0].Name)
	assert.Equal(t, "ubuntu:22.04", pod.Spec.Containers[0].Image)
	assert.Equal(t, "container2", pod.Spec.Containers[1].Name)
	assert.Equal(t, "nginx:latest", pod.Spec.Containers[1].Image)
}

func TestInjectPodWithEmptyConfig(t *testing.T) {
	// Given
	li := NewLabelInjector("test-challenge", "test-user")
	podConfig := &v2alpha1.PodConfig{
		Containers: []v2alpha1.ContainerConfig{},
	}

	// When
	pod := li.InjectPod(podConfig)

	// Then
	assert.NotNil(t, pod)
	assert.Empty(t, pod.Spec.Containers)
	assert.Equal(t, "challenge-test-challenge-test-user-pod", pod.Name)
	assert.Equal(t, li.labels, pod.Labels)
}

func TestInjectPodWithUbuntuBasicExample(t *testing.T) {
	// Given
	li := NewLabelInjector("ubuntu-basic", "test-user")
	podConfig := &v2alpha1.PodConfig{
		Containers: []v2alpha1.ContainerConfig{
			{
				Name:    "ubuntu",
				Image:   "ubuntu:22.04",
				Command: []string{"/bin/bash", "-c"},
				Args: []string{
					`apt-get update && \
DEBIAN_FRONTEND=noninteractive apt-get install -y \
  openssh-server \
  curl \
  wget \
  vim \
  net-tools \
  iputils-ping && \
mkdir -p /run/sshd && \
echo 'root:toor' | chpasswd && \
sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
/usr/sbin/sshd && \
sleep infinity`,
				},
				Ports: []v2alpha1.ContainerPort{
					{
						ContainerPort: 22,
						Protocol:      "TCP",
					},
				},
				Resources: v2alpha1.ResourceRequirements{
					Limits: v2alpha1.ResourceList{
						CPU:    "500m",
						Memory: "512Mi",
					},
					Requests: v2alpha1.ResourceList{
						CPU:    "200m",
						Memory: "256Mi",
					},
				},
			},
		},
	}

	// When
	pod := li.InjectPod(podConfig)

	// Then
	assert.NotNil(t, pod)
	assert.Equal(t, "challenge-ubuntu-basic-test-user-pod", pod.Name)
	assert.Equal(t, li.labels, pod.Labels)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/createdBy": "challenge-operator",
	}, pod.Annotations)

	// Check container configuration
	assert.Len(t, pod.Spec.Containers, 1)
	container := pod.Spec.Containers[0]
	assert.Equal(t, "ubuntu", container.Name)
	assert.Equal(t, "ubuntu:22.04", container.Image)
	assert.Equal(t, []string{"/bin/bash", "-c"}, container.Command)
	assert.Contains(t, container.Args[0], "apt-get update")
	assert.Contains(t, container.Args[0], "openssh-server")
	assert.Contains(t, container.Args[0], "root:toor")
	assert.Contains(t, container.Args[0], "PermitRootLogin yes")

	// Check ports
	assert.Len(t, container.Ports, 1)
	assert.Equal(t, int32(22), container.Ports[0].ContainerPort)
	assert.Equal(t, corev1.ProtocolTCP, container.Ports[0].Protocol)

	// Check resources
	assert.Equal(t, resource.MustParse("500m"), container.Resources.Limits[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("512Mi"), container.Resources.Limits[corev1.ResourceMemory])
	assert.Equal(t, resource.MustParse("200m"), container.Resources.Requests[corev1.ResourceCPU])
	assert.Equal(t, resource.MustParse("256Mi"), container.Resources.Requests[corev1.ResourceMemory])
}

func TestInjectServiceWithUbuntuBasicExample(t *testing.T) {
	// Given
	li := NewLabelInjector("ubuntu-basic", "test-user")
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ubuntu-basic",
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "ssh",
					Protocol:   corev1.ProtocolTCP,
					Port:       22,
					TargetPort: intstr.FromInt(22),
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	// When
	newService := li.InjectService(service)

	// Then
	assert.NotNil(t, newService)
	assert.Equal(t, "challenge-ubuntu-basic-test-user-svc", newService.Name)
	assert.Equal(t, li.labels, newService.Labels)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/createdBy": "challenge-operator",
	}, newService.Annotations)
	assert.Equal(t, map[string]string{
		"apps.hexactf.io/challengeName": "challenge-ubuntu-basic-test-user",
	}, newService.Spec.Selector)

	// Check service configuration
	assert.Equal(t, corev1.ServiceTypeNodePort, newService.Spec.Type)
	assert.Len(t, newService.Spec.Ports, 1)
	assert.Equal(t, "ssh", newService.Spec.Ports[0].Name)
	assert.Equal(t, corev1.ProtocolTCP, newService.Spec.Ports[0].Protocol)
	assert.Equal(t, int32(22), newService.Spec.Ports[0].Port)
	assert.Equal(t, intstr.FromInt(22), newService.Spec.Ports[0].TargetPort)
}
