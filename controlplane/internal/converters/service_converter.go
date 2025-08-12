package converters

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// SecretInfo holds parsed information from a secret ARN
type SecretInfo struct {
	SecretName string
	Key        string
	Source     string // "secretsmanager" or "ssm"
}

// ServiceConverter converts ECS service definitions to Kubernetes Deployments
type ServiceConverter struct {
	region           string
	accountID        string
	networkConverter *NetworkConverter
}

// NewServiceConverter creates a new ServiceConverter
func NewServiceConverter(region, accountID string) *ServiceConverter {
	return &ServiceConverter{
		region:           region,
		accountID:        accountID,
		networkConverter: NewNetworkConverter(region, accountID),
	}
}

// ConvertServiceToDeployment converts an ECS service to a Kubernetes Deployment
func (c *ServiceConverter) ConvertServiceToDeployment(
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	cluster *storage.Cluster,
) (*appsv1.Deployment, *corev1.Service, error) {
	return c.ConvertServiceToDeploymentWithNetworkConfig(service, taskDef, cluster, nil)
}

// ConvertServiceToDeploymentWithNetworkConfig converts an ECS service to a Kubernetes Deployment with network configuration
func (c *ServiceConverter) ConvertServiceToDeploymentWithNetworkConfig(
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	cluster *storage.Cluster,
	networkConfig *generated.NetworkConfiguration,
) (*appsv1.Deployment, *corev1.Service, error) {
	// Parse container definitions from task definition
	var containerDefs []map[string]interface{}
	if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err != nil {
		return nil, nil, fmt.Errorf("failed to parse container definitions: %w", err)
	}

	if len(containerDefs) == 0 {
		return nil, nil, fmt.Errorf("no container definitions found in task definition")
	}
	
	// Debug: Check if secrets are in containerDefs
	for i, containerDef := range containerDefs {
		containerName, _ := containerDef["name"].(string)
		if secrets, exists := containerDef["secrets"]; exists {
			logging.Info("Container has secrets field", "index", i, "name", containerName, "hasSecrets", secrets != nil)
			if secretList, ok := secrets.([]interface{}); ok {
				logging.Info("Container secrets count", "index", i, "name", containerName, "count", len(secretList))
			}
		} else {
			logging.Info("Container has NO secrets field", "index", i, "name", containerName)
		}
	}

	// Parse volumes from task definition
	var volumes []map[string]interface{}
	if taskDef.Volumes != "" {
		if err := json.Unmarshal([]byte(taskDef.Volumes), &volumes); err != nil {
			// Log error but continue - volumes are optional
			// In production, this should use proper logging instead of fmt.Printf
		}
	}

	// Determine network mode from task definition
	networkMode := types.GetNetworkMode(&taskDef.NetworkMode)

	// Create Deployment
	deployment, err := c.createDeployment(service, containerDefs, volumes, cluster, networkConfig, networkMode, taskDef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service (if needed for load balancing)
	kubeService, err := c.createKubernetesService(service, containerDefs, cluster, networkConfig, networkMode)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes service: %w", err)
	}

	return deployment, kubeService, nil
}

// createDeployment creates a Kubernetes Deployment from ECS service
func (c *ServiceConverter) createDeployment(
	service *storage.Service,
	containerDefs []map[string]interface{},
	volumes []map[string]interface{},
	cluster *storage.Cluster,
	networkConfig *generated.NetworkConfiguration,
	networkMode types.NetworkMode,
	taskDef *storage.TaskDefinition,
) (*appsv1.Deployment, error) {
	// Create namespace name
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)

	// Create deployment name (ECS service name with prefix)
	deploymentName := fmt.Sprintf("ecs-service-%s", service.ServiceName)

	// Create containers from container definitions
	containers, k8sVolumes, err := c.createContainersAndVolumes(containerDefs, volumes)
	if err != nil {
		return nil, fmt.Errorf("failed to create containers: %w", err)
	}

	// Create replica count (desired count)
	replicas := int32(service.DesiredCount)

	// Create labels
	labels := map[string]string{
		"kecs.dev/service":     service.ServiceName,
		"kecs.dev/cluster":     cluster.Name,
		"kecs.dev/launch-type": service.LaunchType,
		"kecs.dev/managed-by":  "kecs",
		"app":                  service.ServiceName, // Standard Kubernetes label
	}

	// Create annotations
	annotations := map[string]string{
		"kecs.dev/service-arn":         service.ARN,
		"kecs.dev/task-definition":     service.TaskDefinitionARN,
		"kecs.dev/scheduling-strategy": service.SchedulingStrategy,
	}

	// Add network configuration annotations
	if networkConfig != nil {
		networkAnnotations := c.networkConverter.ConvertNetworkConfiguration(networkConfig, networkMode)
		for k, v := range networkAnnotations {
			annotations[k] = v
		}
	}

	// Create pod template annotations
	podAnnotations := make(map[string]string)
	for k, v := range annotations {
		podAnnotations[k] = v
	}

	// Add secret annotations to pod template
	secretIndex := 0
	logging.Info("Processing containers for secrets", "containerCount", len(containerDefs))
	for _, containerDef := range containerDefs {
		containerName, _ := containerDef["name"].(string)
		logging.Info("Processing container for pod annotations", "container", containerName)
		if secrets, exists := containerDef["secrets"]; exists {
			if secretList, ok := secrets.([]interface{}); ok {
				for _, secret := range secretList {
					if secretMap, ok := secret.(map[string]interface{}); ok {
						name, nameOk := secretMap["name"].(string)
						valueFrom, valueFromOk := secretMap["valueFrom"].(string)
						if nameOk && valueFromOk && name != "" && valueFrom != "" {
							annotationKey := fmt.Sprintf("kecs.dev/secret-%d-arn", secretIndex)
							annotationValue := fmt.Sprintf("%s:%s:%s", containerName, name, valueFrom)
							podAnnotations[annotationKey] = annotationValue
							secretIndex++
							// Debug log
							logging.Info("Added secret annotation", "key", annotationKey, "value", annotationValue)
						}
					}
				}
			}
		}
	}
	if secretIndex > 0 {
		podAnnotations["kecs.dev/secret-count"] = fmt.Sprintf("%d", secretIndex)
		logging.Info("Total secrets found and annotated", "count", secretIndex)
	}

	// Create Deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        deploymentName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":              service.ServiceName,
					"kecs.dev/service": service.ServiceName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: podAnnotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					Containers:    containers,
					Volumes:       k8sVolumes,
				},
			},
		},
	}

	// Add strategy based on scheduling strategy
	if service.SchedulingStrategy == "DAEMON" {
		// For DAEMON services, we should use DaemonSet instead
		// But for now, we'll use Deployment with node affinity
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
		}
	} else {
		// REPLICA strategy - standard rolling update
		deployment.Spec.Strategy = appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
				MaxSurge:       &intstr.IntOrString{Type: intstr.String, StrVal: "25%"},
			},
		}
	}

	return deployment, nil
}

// createContainersAndVolumes creates Kubernetes containers and volumes from ECS definitions
func (c *ServiceConverter) createContainersAndVolumes(containerDefs []map[string]interface{}, volumes []map[string]interface{}) ([]corev1.Container, []corev1.Volume, error) {
	var containers []corev1.Container
	var k8sVolumes []corev1.Volume

	// First, convert ECS volumes to Kubernetes volumes
	for _, vol := range volumes {
		if name, ok := vol["name"].(string); ok && name != "" {
			k8sVol := corev1.Volume{
				Name: name,
			}

			// Check for host volume configuration
			if hostConfig, ok := vol["host"].(map[string]interface{}); ok {
				if sourcePath, ok := hostConfig["sourcePath"].(string); ok && sourcePath != "" {
					// Host path volume
					k8sVol.VolumeSource = corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: sourcePath,
						},
					}
				} else {
					// Empty host configuration - use emptyDir
					k8sVol.VolumeSource = corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					}
				}
			} else {
				// No host configuration - default to emptyDir
				k8sVol.VolumeSource = corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				}
			}

			k8sVolumes = append(k8sVolumes, k8sVol)
		}
	}

	// Then, create containers with volume mounts
	for _, containerDef := range containerDefs {
		container, err := c.createContainer(containerDef)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create container: %w", err)
		}
		containers = append(containers, container)
	}

	return containers, k8sVolumes, nil
}

// createContainers creates Kubernetes containers from ECS container definitions
func (c *ServiceConverter) createContainers(containerDefs []map[string]interface{}) ([]corev1.Container, error) {
	var containers []corev1.Container

	for _, containerDef := range containerDefs {
		container, err := c.createContainer(containerDef)
		if err != nil {
			return nil, fmt.Errorf("failed to create container: %w", err)
		}
		containers = append(containers, container)
	}

	return containers, nil
}

// createContainer creates a single Kubernetes container from ECS container definition
func (c *ServiceConverter) createContainer(containerDef map[string]interface{}) (corev1.Container, error) {
	// Extract basic properties
	name, _ := containerDef["name"].(string)
	image, _ := containerDef["image"].(string)

	if name == "" || image == "" {
		return corev1.Container{}, fmt.Errorf("container name and image are required")
	}

	container := corev1.Container{
		Name:  name,
		Image: image,
	}

	// Extract CPU and memory
	if cpu, exists := containerDef["cpu"]; exists {
		if cpuFloat, ok := cpu.(float64); ok {
			// ECS CPU units: 1024 units = 1 vCPU
			// Kubernetes: "1000m" = 1 CPU
			cpuMillis := fmt.Sprintf("%dm", int(cpuFloat*1000/1024))
			container.Resources.Requests = corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse(cpuMillis),
			}
		}
	}

	if memory, exists := containerDef["memory"]; exists {
		if memFloat, ok := memory.(float64); ok {
			// ECS memory is in MiB, Kubernetes expects Mi or Gi
			memoryStr := fmt.Sprintf("%dMi", int(memFloat))
			if container.Resources.Requests == nil {
				container.Resources.Requests = corev1.ResourceList{}
			}
			container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(memoryStr)
		}
	}

	// Extract environment variables
	if env, exists := containerDef["environment"]; exists {
		if envList, ok := env.([]interface{}); ok {
			for _, envVar := range envList {
				if envMap, ok := envVar.(map[string]interface{}); ok {
					name, _ := envMap["name"].(string)
					value, _ := envMap["value"].(string)
					if name != "" {
						container.Env = append(container.Env, corev1.EnvVar{
							Name:  name,
							Value: value,
						})
					}
				}
			}
		}
	}

	// Extract secrets from environment
	if secrets, exists := containerDef["secrets"]; exists {
		if secretList, ok := secrets.([]interface{}); ok {
			secretEnvVars := c.convertSecrets(secretList)
			container.Env = append(container.Env, secretEnvVars...)
		}
	}

	// Extract port mappings
	if ports, exists := containerDef["portMappings"]; exists {
		if portList, ok := ports.([]interface{}); ok {
			for _, portMapping := range portList {
				if portMap, ok := portMapping.(map[string]interface{}); ok {
					containerPort, _ := portMap["containerPort"].(float64)
					protocol, _ := portMap["protocol"].(string)
					if protocol == "" {
						protocol = "TCP"
					}

					if containerPort > 0 {
						container.Ports = append(container.Ports, corev1.ContainerPort{
							ContainerPort: int32(containerPort),
							Protocol:      corev1.Protocol(strings.ToUpper(protocol)),
						})
					}
				}
			}
		}
	}

	// Extract mount points
	if mountPoints, exists := containerDef["mountPoints"]; exists {
		if mountList, ok := mountPoints.([]interface{}); ok {
			for _, mount := range mountList {
				if mountMap, ok := mount.(map[string]interface{}); ok {
					sourceVolume, _ := mountMap["sourceVolume"].(string)
					containerPath, _ := mountMap["containerPath"].(string)
					readOnly, _ := mountMap["readOnly"].(bool)

					if sourceVolume != "" && containerPath != "" {
						container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
							Name:      sourceVolume,
							MountPath: containerPath,
							ReadOnly:  readOnly,
						})
					}
				}
			}
		}
	}

	// Extract command and args
	if command, exists := containerDef["command"]; exists {
		if cmdList, ok := command.([]interface{}); ok {
			for _, cmd := range cmdList {
				if cmdStr, ok := cmd.(string); ok {
					container.Command = append(container.Command, cmdStr)
				}
			}
		}
	}

	if entryPoint, exists := containerDef["entryPoint"]; exists {
		if epList, ok := entryPoint.([]interface{}); ok {
			for _, ep := range epList {
				if epStr, ok := ep.(string); ok {
					container.Args = append(container.Args, epStr)
				}
			}
		}
	}

	return container, nil
}

// createKubernetesService creates a Kubernetes Service for load balancing (if needed)
func (c *ServiceConverter) createKubernetesService(
	service *storage.Service,
	containerDefs []map[string]interface{},
	cluster *storage.Cluster,
	networkConfig *generated.NetworkConfiguration,
	networkMode types.NetworkMode,
) (*corev1.Service, error) {
	// Check if service has load balancers configured
	if service.LoadBalancers == "" || service.LoadBalancers == "null" || service.LoadBalancers == "[]" {
		// No load balancer configured, no need for Kubernetes Service
		return nil, nil
	}

	// Parse load balancers
	var loadBalancers []map[string]interface{}
	if err := json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers); err != nil {
		return nil, fmt.Errorf("failed to parse load balancers: %w", err)
	}

	if len(loadBalancers) == 0 {
		return nil, nil
	}

	// Create namespace name
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)

	// Create service name
	serviceName := fmt.Sprintf("ecs-service-%s", service.ServiceName)

	// Extract ports from container definitions
	var servicePorts []corev1.ServicePort
	for _, containerDef := range containerDefs {
		if ports, exists := containerDef["portMappings"]; exists {
			if portList, ok := ports.([]interface{}); ok {
				for _, portMapping := range portList {
					if portMap, ok := portMapping.(map[string]interface{}); ok {
						containerPort, _ := portMap["containerPort"].(float64)
						protocol, _ := portMap["protocol"].(string)
						if protocol == "" {
							protocol = "TCP"
						}

						if containerPort > 0 {
							servicePorts = append(servicePorts, corev1.ServicePort{
								Port:       int32(containerPort),
								TargetPort: intstr.FromInt(int(containerPort)),
								Protocol:   corev1.Protocol(strings.ToUpper(protocol)),
								Name:       fmt.Sprintf("port-%d", int(containerPort)),
							})
						}
					}
				}
			}
		}
	}

	if len(servicePorts) == 0 {
		// No ports defined, no need for Kubernetes Service
		return nil, nil
	}

	// Create annotations for the service
	serviceAnnotations := map[string]string{
		"kecs.dev/service-arn": service.ARN,
	}

	// Add network configuration annotations if available
	if networkConfig != nil {
		networkAnnotations := c.networkConverter.ConvertNetworkConfiguration(networkConfig, networkMode)
		for k, v := range networkAnnotations {
			serviceAnnotations[k] = v
		}
	}

	// Create Kubernetes Service
	kubeService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels: map[string]string{
				"kecs.dev/service":    service.ServiceName,
				"kecs.dev/cluster":    cluster.Name,
				"kecs.dev/managed-by": "kecs",
				"app":                 service.ServiceName,
			},
			Annotations: serviceAnnotations,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app":              service.ServiceName,
				"kecs.dev/service": service.ServiceName,
			},
			Ports: servicePorts,
			Type:  corev1.ServiceTypeClusterIP, // Default to ClusterIP
		},
	}

	// Check for load balancer type
	for _, lb := range loadBalancers {
		if lbType, exists := lb["type"]; exists {
			if lbTypeStr, ok := lbType.(string); ok {
				switch strings.ToLower(lbTypeStr) {
				case "application", "network":
					// AWS ALB/NLB - use LoadBalancer type
					kubeService.Spec.Type = corev1.ServiceTypeLoadBalancer
				}
			}
		}
	}

	return kubeService, nil
}

// convertSecrets converts ECS secrets to Kubernetes environment variables
func (c *ServiceConverter) convertSecrets(secrets []interface{}) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0, len(secrets))

	for _, secret := range secrets {
		if secretMap, ok := secret.(map[string]interface{}); ok {
			name, nameOk := secretMap["name"].(string)
			valueFrom, valueFromOk := secretMap["valueFrom"].(string)

			if !nameOk || !valueFromOk || name == "" || valueFrom == "" {
				continue
			}

			// Parse the secret ARN
			secretInfo, err := c.parseSecretArn(valueFrom)
			if err != nil {
				// If we can't parse it, skip it
				// In production, you might want to handle this differently
				continue
			}

			envVar := corev1.EnvVar{
				Name: name,
			}

			switch secretInfo.Source {
			case "secretsmanager":
				// Reference the synced Kubernetes secret
				envVar.ValueFrom = &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: c.getK8sSecretName("secretsmanager", secretInfo.SecretName),
						},
						Key: secretInfo.Key,
					},
				}

			case "ssm":
				// All SSM parameters are now stored as Secrets for consistency
				envVar.ValueFrom = &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: c.getK8sSecretName("ssm", secretInfo.SecretName),
						},
						Key: "value",
					},
				}
			}

			envVars = append(envVars, envVar)
		}
	}

	return envVars
}

// parseSecretArn parses an AWS secret ARN and extracts relevant information
func (c *ServiceConverter) parseSecretArn(arn string) (*SecretInfo, error) {
	// ARN formats:
	// Secrets Manager: arn:aws:secretsmanager:region:account-id:secret:name-6RandomChars:key::
	// SSM: arn:aws:ssm:region:account-id:parameter/name

	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid ARN format: %s", arn)
	}

	service := parts[2]
	info := &SecretInfo{}

	switch service {
	case "secretsmanager":
		info.Source = "secretsmanager"
		// Extract secret name and key from remaining parts
		// Format: arn:aws:secretsmanager:region:account-id:secret:name-6RandomChars:key::
		if len(parts) >= 7 {
			info.SecretName = parts[6]
			// Check if a JSON key is specified at index 7
			if len(parts) > 7 && parts[7] != "" && parts[7] != "*" {
				info.Key = parts[7]
			} else {
				// No JSON key specified, the entire secret value will be used
				// When synced by Secrets Manager integration, JSON secrets will have all keys available
				info.Key = "value"
			}
		} else {
			return nil, fmt.Errorf("invalid Secrets Manager ARN format: %s", arn)
		}

	case "ssm":
		info.Source = "ssm"
		// Extract parameter name from ARN
		// Format: arn:aws:ssm:region:account-id:parameter/path/to/param
		// The parameter path starts after "parameter/"
		resourcePart := parts[5]
		if strings.HasPrefix(resourcePart, "parameter/") {
			info.SecretName = strings.TrimPrefix(resourcePart, "parameter/")
		} else if strings.HasPrefix(resourcePart, "parameter") && len(parts) > 6 {
			// Sometimes the path might be in the next part
			info.SecretName = parts[6]
		} else {
			info.SecretName = resourcePart
		}
		info.Key = "value"

	default:
		return nil, fmt.Errorf("unsupported secret service: %s", service)
	}

	return info, nil
}

// sanitizeSecretName converts a secret name to be Kubernetes-compatible
func (c *ServiceConverter) sanitizeSecretName(name string) string {
	// Remove the random suffix from Secrets Manager secret names
	// Format: my-secret-AbCdEf -> my-secret
	if idx := strings.LastIndex(name, "-"); idx > 0 && len(name)-idx == 7 {
		// Check if last part looks like a random suffix (6 chars)
		suffix := name[idx+1:]
		if len(suffix) == 6 && regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(suffix) {
			name = name[:idx]
		}
	}

	// Handle path separators for hierarchical secrets
	name = strings.ReplaceAll(name, "/", "-")

	// Similar to volume names, but for secrets
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	// Return the sanitized name without prefix
	// The actual prefix (sm- or ssm-) will be added by the integration modules
	return name
}

// getK8sSecretName returns the Kubernetes secret name for a given source and secret name
func (c *ServiceConverter) getK8sSecretName(source, secretName string) string {
	switch source {
	case "secretsmanager":
		// Remove the random suffix that Secrets Manager adds (e.g., -AbCdEf)
		re := regexp.MustCompile(`-[A-Za-z0-9]{6}$`)
		cleanName := re.ReplaceAllString(secretName, "")
		cleanName = strings.ToLower(cleanName)
		cleanName = strings.ReplaceAll(cleanName, "/", "-")
		cleanName = strings.Trim(cleanName, "-")
		return "sm-" + cleanName
	case "ssm":
		cleanName := strings.Trim(secretName, "/")
		cleanName = strings.ReplaceAll(cleanName, "/", "-")
		cleanName = strings.ToLower(cleanName)
		return "ssm-" + cleanName
	default:
		return "unknown-" + strings.ToLower(secretName)
	}
}

// getK8sConfigMapName returns the Kubernetes ConfigMap name for a given SSM parameter
// DEPRECATED: All SSM parameters are now stored as Secrets for consistency
func (c *ServiceConverter) getK8sConfigMapName(parameterName string) string {
	cleanName := strings.Trim(parameterName, "/")
	cleanName = strings.ReplaceAll(cleanName, "/", "-")
	cleanName = strings.ToLower(cleanName)
	return "ssm-cm-" + cleanName
}

// isSSMParameterSensitive determines if an SSM parameter should be treated as sensitive
// DEPRECATED: All SSM parameters are now stored as Secrets for consistency
func (c *ServiceConverter) isSSMParameterSensitive(parameterName string) bool {
	// All SSM parameters are now treated as sensitive and stored as Secrets
	return true
}

// getNamespacedSecretName returns the namespace-aware secret name for LocalStack
func (c *ServiceConverter) getNamespacedSecretName(cluster *storage.Cluster, secretName string) string {
	// Format: <namespace>/<secret-name>
	// The namespace already contains cluster and region information
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	return fmt.Sprintf("%s/%s", namespace, secretName)
}

// getPlaceholderSecretValue returns placeholder values for secrets
// NOTE: This is now deprecated in favor of actual Kubernetes secret references
// Kept for backward compatibility and testing
func (c *ServiceConverter) getPlaceholderSecretValue(source, secretName, key string) string {
	// Generate deterministic placeholder values based on the secret name and key
	// This ensures consistency across deployments while being obviously fake
	
	switch source {
	case "secretsmanager":
		// Generate different placeholder values for different secret types
		if strings.Contains(strings.ToLower(secretName), "db") || strings.Contains(strings.ToLower(secretName), "database") {
			// Check if key or secretName contains password/pass
			if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "pass") ||
				strings.Contains(strings.ToLower(secretName), "password") || strings.Contains(strings.ToLower(secretName), "pass") {
				return "placeholder-db-password-from-secrets-manager"
			}
			return "placeholder-db-connection-from-secrets-manager"
		}
		if strings.Contains(strings.ToLower(secretName), "jwt") {
			return "placeholder-jwt-secret-from-secrets-manager"
		}
		if strings.Contains(strings.ToLower(secretName), "encrypt") {
			return "placeholder-encryption-key-from-secrets-manager"
		}
		return fmt.Sprintf("placeholder-secret-from-secrets-manager-%s-%s", secretName, key)
		
	case "ssm":
		// Generate placeholder values for SSM parameters
		if strings.Contains(strings.ToLower(secretName), "database") {
			return "postgresql://placeholder:placeholder@localhost:5432/placeholder"
		}
		if strings.Contains(strings.ToLower(secretName), "api_key") {
			return "placeholder-api-key-from-ssm"
		}
		if strings.Contains(strings.ToLower(secretName), "feature") {
			return `{"new_ui": true, "beta_features": true, "maintenance_mode": false}`
		}
		return fmt.Sprintf("placeholder-parameter-from-ssm-%s", secretName)
		
	default:
		return fmt.Sprintf("placeholder-unknown-secret-%s", secretName)
	}
}
