package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

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
