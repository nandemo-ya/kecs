package converters

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// createSecretSyncInitContainer creates an init container that syncs secrets from kecs-system namespace
func (c *TaskConverter) createSecretSyncInitContainer(secretNames []string, configMapNames []string) *corev1.Container {
	if len(secretNames) == 0 && len(configMapNames) == 0 {
		return nil
	}

	// Build the sync script
	script := `#!/bin/sh
set -e
echo "Starting secret synchronization from kecs-system namespace..."

# Create directory for temporary secrets
mkdir -p /tmp/secrets
`

	// Add commands to copy secrets
	for _, secretName := range secretNames {
		script += fmt.Sprintf(`
# Copy secret %s from kecs-system
echo "Copying secret %s..."
kubectl get secret %s -n kecs-system -o json | \
  jq 'del(.metadata.namespace, .metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp)' | \
  kubectl apply -n $POD_NAMESPACE -f -
`, secretName, secretName, secretName)
	}

	// Add commands to copy configmaps
	for _, configMapName := range configMapNames {
		script += fmt.Sprintf(`
# Copy configmap %s from kecs-system
echo "Copying configmap %s..."
kubectl get configmap %s -n kecs-system -o json | \
  jq 'del(.metadata.namespace, .metadata.uid, .metadata.resourceVersion, .metadata.creationTimestamp)' | \
  kubectl apply -n $POD_NAMESPACE -f -
`, configMapName, configMapName, configMapName)
	}

	script += `
echo "Secret synchronization completed successfully"
`

	return &corev1.Container{
		Name:    "secret-sync",
		Image:   "bitnami/kubectl:latest",
		Command: []string{"/bin/sh", "-c", script},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
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
	}
}

// extractSecretAndConfigMapNames extracts unique secret and configmap names from container definitions
func (c *TaskConverter) extractSecretAndConfigMapNames(containerDefs []types.ContainerDefinition) ([]string, []string) {
	secretMap := make(map[string]bool)
	configMapMap := make(map[string]bool)

	for _, containerDef := range containerDefs {
		if containerDef.Secrets != nil {
			for _, secret := range containerDef.Secrets {
				if secret.ValueFrom != nil {
					secretInfo, err := c.parseSecretArn(*secret.ValueFrom)
					if err != nil {
						continue
					}

					switch secretInfo.Source {
					case "secretsmanager":
						secretName := c.getK8sSecretName("secretsmanager", secretInfo.SecretName)
						secretMap[secretName] = true
					case "ssm":
						if c.isSSMParameterSensitive(secretInfo.SecretName) {
							secretName := c.getK8sSecretName("ssm", secretInfo.SecretName)
							secretMap[secretName] = true
						} else {
							configMapName := c.getK8sConfigMapName(secretInfo.SecretName)
							configMapMap[configMapName] = true
						}
					}
				}
			}
		}
	}

	// Convert maps to slices
	var secretNames []string
	for name := range secretMap {
		secretNames = append(secretNames, name)
	}

	var configMapNames []string
	for name := range configMapMap {
		configMapNames = append(configMapNames, name)
	}

	return secretNames, configMapNames
}
