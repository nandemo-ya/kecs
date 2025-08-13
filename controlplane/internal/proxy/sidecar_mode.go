package proxy

import (
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SidecarProxy handles sidecar injection for AWS proxy
type SidecarProxy struct {
	localStackEndpoint string
	proxyImage         string
}

// NewSidecarProxy creates a new sidecar proxy
func NewSidecarProxy(endpoint string) *SidecarProxy {
	return &SidecarProxy{
		localStackEndpoint: endpoint,
		proxyImage:         "kecs/aws-proxy:latest",
	}
}

// UpdateEndpoint updates the LocalStack endpoint
func (sp *SidecarProxy) UpdateEndpoint(endpoint string) {
	sp.localStackEndpoint = endpoint
}

// SetProxyImage sets the proxy image to use
func (sp *SidecarProxy) SetProxyImage(image string) {
	sp.proxyImage = image
}

// ShouldInjectSidecar checks if a pod should have the AWS proxy sidecar injected
func (sp *SidecarProxy) ShouldInjectSidecar(pod *corev1.Pod) bool {
	// Check annotations
	if pod.Annotations != nil {
		// Check if proxy is explicitly disabled
		if val, ok := pod.Annotations["kecs.io/aws-proxy-enabled"]; ok && val == "false" {
			return false
		}

		// Check if proxy mode is set to sidecar
		if val, ok := pod.Annotations["kecs.io/aws-proxy-mode"]; ok {
			return val == "sidecar"
		}

		// Check if sidecar injection is explicitly requested
		if val, ok := pod.Annotations["kecs.io/inject-aws-proxy"]; ok && val == "true" {
			return true
		}
	}

	// Check labels - inject for ECS tasks by default if proxy mode is not set
	if pod.Labels != nil {
		if _, ok := pod.Labels["kecs.dev/task-id"]; ok {
			// This is an ECS task, check if environment mode is not already set
			if pod.Annotations == nil || pod.Annotations["kecs.io/aws-proxy-mode"] == "" {
				return true // Default to sidecar for ECS tasks
			}
		}
	}

	return false
}

// CreateProxySidecar creates the AWS proxy sidecar container
func (sp *SidecarProxy) CreateProxySidecar(pod *corev1.Pod) *corev1.Container {
	// Get custom endpoint from annotations if present
	endpoint := sp.localStackEndpoint
	if pod.Annotations != nil {
		if customEndpoint, ok := pod.Annotations["kecs.io/localstack-endpoint"]; ok {
			endpoint = customEndpoint
		}
	}

	sidecar := &corev1.Container{
		Name:  "aws-proxy-sidecar",
		Image: sp.proxyImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          "proxy",
				ContainerPort: 4566,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "LOCALSTACK_ENDPOINT",
				Value: endpoint,
			},
			{
				Name:  "DEBUG",
				Value: "false",
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(4566),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
			TimeoutSeconds:      3,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/health",
					Port: intstr.FromInt(4566),
				},
			},
			InitialDelaySeconds: 10,
			PeriodSeconds:       30,
			TimeoutSeconds:      3,
		},
	}

	// Add debug mode if requested
	if pod.Annotations != nil && pod.Annotations["kecs.io/aws-proxy-debug"] == "true" {
		for i := range sidecar.Env {
			if sidecar.Env[i].Name == "DEBUG" {
				sidecar.Env[i].Value = "true"
				break
			}
		}
	}

	return sidecar
}

// InjectSidecar injects the AWS proxy sidecar and updates container environment variables
func (sp *SidecarProxy) InjectSidecar(pod *corev1.Pod) error {
	if !sp.ShouldInjectSidecar(pod) {
		return nil
	}

	logging.Debug("Injecting AWS proxy sidecar into pod", "namespace", pod.Namespace, "name", pod.Name)

	// Create sidecar container
	sidecar := sp.CreateProxySidecar(pod)

	// Add sidecar to pod
	pod.Spec.Containers = append(pod.Spec.Containers, *sidecar)

	// Update environment variables in all containers to use the sidecar
	for i := range pod.Spec.Containers {
		// Skip the sidecar itself
		if pod.Spec.Containers[i].Name == "aws-proxy-sidecar" {
			continue
		}

		// Add or update AWS endpoint environment variables
		awsEndpointEnvVars := []corev1.EnvVar{
			{Name: "AWS_ENDPOINT_URL", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_S3", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_IAM", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_LOGS", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_SSM", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_SECRETSMANAGER", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_ELB", Value: "http://localhost:4566"},
		}

		// Merge with existing environment variables
		envMap := make(map[string]int)
		for j, env := range pod.Spec.Containers[i].Env {
			envMap[env.Name] = j
		}

		for _, newEnv := range awsEndpointEnvVars {
			if idx, exists := envMap[newEnv.Name]; exists {
				// Update existing
				pod.Spec.Containers[i].Env[idx] = newEnv
			} else {
				// Add new
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, newEnv)
			}
		}
	}

	// Add annotation to indicate sidecar was injected
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations["kecs.io/aws-proxy-sidecar-injected"] = "true"

	return nil
}

// InjectSidecarPatches creates JSON patches for sidecar injection (for webhook mode)
func (sp *SidecarProxy) InjectSidecarPatches(pod *corev1.Pod) ([]PatchOperation, error) {
	if !sp.ShouldInjectSidecar(pod) {
		return nil, nil
	}

	logging.Debug("Creating patches for AWS proxy sidecar injection into pod", "namespace", pod.Namespace, "name", pod.Name)

	patches := []PatchOperation{}

	// Create sidecar container
	sidecar := sp.CreateProxySidecar(pod)

	// Add sidecar container
	patches = append(patches, PatchOperation{
		Op:    "add",
		Path:  "/spec/containers/-",
		Value: sidecar,
	})

	// Update environment variables in existing containers
	for i := range pod.Spec.Containers {
		containerPath := fmt.Sprintf("/spec/containers/%d/env", i)

		// AWS endpoint environment variables
		awsEndpointEnvVars := []corev1.EnvVar{
			{Name: "AWS_ENDPOINT_URL", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_S3", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_IAM", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_LOGS", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_SSM", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_SECRETSMANAGER", Value: "http://localhost:4566"},
			{Name: "AWS_ENDPOINT_URL_ELB", Value: "http://localhost:4566"},
		}

		if pod.Spec.Containers[i].Env == nil {
			// Create env array
			patches = append(patches, PatchOperation{
				Op:    "add",
				Path:  containerPath,
				Value: awsEndpointEnvVars,
			})
		} else {
			// Add to existing env array
			for _, envVar := range awsEndpointEnvVars {
				// Check if the env var already exists
				exists := false
				for _, existing := range pod.Spec.Containers[i].Env {
					if existing.Name == envVar.Name {
						exists = true
						break
					}
				}

				if !exists {
					patches = append(patches, PatchOperation{
						Op:    "add",
						Path:  fmt.Sprintf("%s/-", containerPath),
						Value: envVar,
					})
				}
			}
		}
	}

	// Add annotation
	if pod.Annotations == nil {
		patches = append(patches, PatchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: map[string]string{},
		})
	}

	patches = append(patches, PatchOperation{
		Op:    "add",
		Path:  "/metadata/annotations/kecs.io~1aws-proxy-sidecar-injected",
		Value: "true",
	})

	return patches, nil
}
