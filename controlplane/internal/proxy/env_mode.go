package proxy

import (
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// EnvironmentVariableProxy handles environment variable injection for AWS SDK configuration
type EnvironmentVariableProxy struct {
	localStackEndpoint string
}

// NewEnvironmentVariableProxy creates a new environment variable proxy
func NewEnvironmentVariableProxy(endpoint string) *EnvironmentVariableProxy {
	return &EnvironmentVariableProxy{
		localStackEndpoint: endpoint,
	}
}

// UpdateEndpoint updates the LocalStack endpoint
func (evp *EnvironmentVariableProxy) UpdateEndpoint(endpoint string) {
	evp.localStackEndpoint = endpoint
}

// ShouldInjectProxy checks if a pod should have proxy environment variables injected
func (evp *EnvironmentVariableProxy) ShouldInjectProxy(pod *corev1.Pod) bool {
	// Check annotations
	if pod.Annotations != nil {
		// Check if proxy is explicitly disabled
		if val, ok := pod.Annotations["kecs.io/aws-proxy-enabled"]; ok && val == "false" {
			return false
		}

		// Check if proxy mode is set
		if val, ok := pod.Annotations["kecs.io/aws-proxy-mode"]; ok {
			return val == "environment"
		}
	}

	// Check namespace labels
	if pod.Namespace != "" && pod.Labels != nil {
		if val, ok := pod.Labels["kecs.io/aws-proxy"]; ok && val == "enabled" {
			return true
		}
	}

	// Default behavior - inject for ECS tasks
	if pod.Labels != nil {
		if _, ok := pod.Labels["ecs.task.definition.family"]; ok {
			return true
		}
	}

	return false
}

// InjectEnvironmentVariables injects AWS SDK environment variables into a pod
func (evp *EnvironmentVariableProxy) InjectEnvironmentVariables(pod *corev1.Pod) ([]PatchOperation, error) {
	if !evp.ShouldInjectProxy(pod) {
		return nil, nil
	}

	logging.Debug("Injecting AWS environment variables into pod", "namespace", pod.Namespace, "name", pod.Name)

	patches := []PatchOperation{}

	// Get custom endpoint from annotations if present
	endpoint := evp.localStackEndpoint
	if pod.Annotations != nil {
		if customEndpoint, ok := pod.Annotations["kecs.io/localstack-endpoint"]; ok {
			endpoint = customEndpoint
		}
	}

	// AWS environment variables to inject
	awsEnvVars := []corev1.EnvVar{
		{Name: "AWS_ENDPOINT_URL", Value: endpoint},
		{Name: "AWS_ENDPOINT_URL_S3", Value: endpoint},
		{Name: "AWS_ENDPOINT_URL_IAM", Value: endpoint},
		{Name: "AWS_ENDPOINT_URL_LOGS", Value: endpoint},
		{Name: "AWS_ENDPOINT_URL_SSM", Value: endpoint},
		{Name: "AWS_ENDPOINT_URL_SECRETSMANAGER", Value: endpoint},
		{Name: "AWS_ENDPOINT_URL_ELB", Value: endpoint},
		{Name: "AWS_ACCESS_KEY_ID", Value: "test"},
		{Name: "AWS_SECRET_ACCESS_KEY", Value: "test"},
		{Name: "AWS_DEFAULT_REGION", Value: "us-east-1"},
		{Name: "AWS_REGION", Value: "us-east-1"},
	}

	// Apply environment variables to all containers
	for i := range pod.Spec.Containers {
		containerPath := fmt.Sprintf("/spec/containers/%d/env", i)

		// Check if env array exists
		if pod.Spec.Containers[i].Env == nil {
			// Create env array
			patches = append(patches, PatchOperation{
				Op:    "add",
				Path:  containerPath,
				Value: awsEnvVars,
			})
		} else {
			// Add to existing env array
			for _, envVar := range awsEnvVars {
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

	// Add annotation to indicate proxy was injected
	if pod.Annotations == nil {
		patches = append(patches, PatchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: map[string]string{},
		})
	}

	patches = append(patches, PatchOperation{
		Op:    "add",
		Path:  "/metadata/annotations/kecs.io~1aws-proxy-injected",
		Value: "true",
	})

	return patches, nil
}

// PatchOperation represents a JSON patch operation
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// CreatePatch creates a JSON patch from patch operations
func CreatePatch(operations []PatchOperation) ([]byte, error) {
	return json.Marshal(operations)
}
