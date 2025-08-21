package converters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// TaskSetConverterWithSD extends TaskSetConverter with Service Discovery support
type TaskSetConverterWithSD struct {
	*TaskSetConverter
}

// NewTaskSetConverterWithSD creates a new TaskSetConverter with Service Discovery integration
func NewTaskSetConverterWithSD(taskConverter *TaskConverter) *TaskSetConverterWithSD {
	return &TaskSetConverterWithSD{
		TaskSetConverter: NewTaskSetConverter(taskConverter),
	}
}

// AddServiceDiscoveryAnnotations adds Service Discovery related annotations to resources
func (c *TaskSetConverterWithSD) AddServiceDiscoveryAnnotations(
	taskSet *storage.TaskSet,
	deployment *metav1.ObjectMeta,
	podTemplate *metav1.ObjectMeta,
) error {
	if taskSet.ServiceRegistries == "" {
		return nil
	}

	// Parse service registries
	var serviceRegistries []generated.ServiceRegistry
	if err := json.Unmarshal([]byte(taskSet.ServiceRegistries), &serviceRegistries); err != nil {
		return fmt.Errorf("failed to parse service registries: %w", err)
	}

	if len(serviceRegistries) == 0 {
		return nil
	}

	// Add annotations to deployment
	if deployment.Annotations == nil {
		deployment.Annotations = make(map[string]string)
	}
	deployment.Annotations["kecs.io/service-registries"] = taskSet.ServiceRegistries
	deployment.Annotations["kecs.io/service-discovery-enabled"] = "true"

	// Add annotations to pod template
	if podTemplate.Annotations == nil {
		podTemplate.Annotations = make(map[string]string)
	}

	// Process each service registry
	for i, registry := range serviceRegistries {
		// Add registry ARN
		if registry.RegistryArn != nil {
			podTemplate.Annotations[fmt.Sprintf("kecs.io/service-registry-%d-arn", i)] = *registry.RegistryArn
		}

		// Add container name and port if specified
		if registry.ContainerName != nil {
			podTemplate.Annotations[fmt.Sprintf("kecs.io/service-registry-%d-container", i)] = *registry.ContainerName
		}
		if registry.ContainerPort != nil {
			podTemplate.Annotations[fmt.Sprintf("kecs.io/service-registry-%d-port", i)] = fmt.Sprintf("%d", *registry.ContainerPort)
		}

		// Extract namespace and service name from registry ARN
		if registry.RegistryArn != nil {
			namespace, serviceName := c.extractServiceDiscoveryInfo(*registry.RegistryArn)
			if namespace != "" {
				podTemplate.Annotations[fmt.Sprintf("kecs.io/sd-namespace-%d", i)] = namespace
			}
			if serviceName != "" {
				podTemplate.Annotations[fmt.Sprintf("kecs.io/sd-service-%d", i)] = serviceName
			}
		}
	}

	// Add labels for Service Discovery
	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}
	deployment.Labels["kecs.io/service-discovery"] = "enabled"

	if podTemplate.Labels == nil {
		podTemplate.Labels = make(map[string]string)
	}
	podTemplate.Labels["kecs.io/service-discovery"] = "enabled"

	return nil
}

// extractServiceDiscoveryInfo extracts namespace and service name from registry ARN
func (c *TaskSetConverterWithSD) extractServiceDiscoveryInfo(registryArn string) (namespace, serviceName string) {
	// ARN format: arn:aws:servicediscovery:region:account:service/srv-xxxx
	// or: arn:aws:servicediscovery:region:account:namespace/ns-xxxx/service/srv-xxxx
	parts := strings.Split(registryArn, ":")
	if len(parts) < 6 {
		return "", ""
	}

	resourcePart := parts[5]
	if strings.Contains(resourcePart, "/") {
		resourceParts := strings.Split(resourcePart, "/")
		if len(resourceParts) >= 2 {
			if resourceParts[0] == "namespace" && len(resourceParts) >= 4 {
				// namespace/ns-xxxx/service/srv-xxxx format
				namespace = resourceParts[1]
				if len(resourceParts) >= 4 && resourceParts[2] == "service" {
					serviceName = resourceParts[3]
				}
			} else if resourceParts[0] == "service" {
				// service/srv-xxxx format
				serviceName = resourceParts[1]
			}
		}
	}

	return namespace, serviceName
}

// ConvertTaskSetToServiceDiscoveryEndpoint creates a Service Discovery endpoint for TaskSet
func (c *TaskSetConverterWithSD) ConvertTaskSetToServiceDiscoveryEndpoint(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	clusterName string,
) (*corev1.Endpoints, error) {
	if taskSet.ServiceRegistries == "" {
		return nil, nil
	}

	// Parse service registries
	var serviceRegistries []generated.ServiceRegistry
	if err := json.Unmarshal([]byte(taskSet.ServiceRegistries), &serviceRegistries); err != nil {
		return nil, fmt.Errorf("failed to parse service registries: %w", err)
	}

	if len(serviceRegistries) == 0 {
		return nil, nil
	}

	// Create endpoints for Service Discovery
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetServiceDiscoveryEndpointName(service.ServiceName, taskSet.ID),
			Namespace: c.GetNamespace(clusterName, taskSet.Region),
			Labels: map[string]string{
				"kecs.io/cluster": clusterName,
				"kecs.io/service": service.ServiceName,
				"kecs.io/taskset": taskSet.ID,
				"kecs.io/role":    "service-discovery",
				"kecs.io/managed": "true",
			},
			Annotations: map[string]string{
				"kecs.io/taskset-arn":        taskSet.ARN,
				"kecs.io/service-arn":        taskSet.ServiceARN,
				"kecs.io/service-registries": taskSet.ServiceRegistries,
			},
		},
		Subsets: []corev1.EndpointSubset{},
	}

	// Add subset for each service registry
	for _, registry := range serviceRegistries {
		subset := corev1.EndpointSubset{
			Addresses: []corev1.EndpointAddress{},
			Ports:     []corev1.EndpointPort{},
		}

		// Add port if specified
		if registry.ContainerPort != nil {
			subset.Ports = append(subset.Ports, corev1.EndpointPort{
				Name: fmt.Sprintf("port-%d", *registry.ContainerPort),
				Port: *registry.ContainerPort,
			})
		}

		endpoints.Subsets = append(endpoints.Subsets, subset)
	}

	return endpoints, nil
}

// GetServiceDiscoveryEndpointName generates the endpoint name for Service Discovery
func (c *TaskSetConverterWithSD) GetServiceDiscoveryEndpointName(serviceName, taskSetID string) string {
	name := fmt.Sprintf("%s-%s-sd", serviceName, taskSetID)
	// Ensure the name is valid for Kubernetes
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ToLower(name)
	// Truncate if too long (max 63 characters)
	if len(name) > 63 {
		name = name[:63]
	}
	// Remove trailing hyphens
	name = strings.TrimSuffix(name, "-")
	return name
}

// RegisterTaskSetWithServiceDiscovery registers TaskSet instances with Service Discovery
func (c *TaskSetConverterWithSD) RegisterTaskSetWithServiceDiscovery(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	podIPs []string,
) error {
	if taskSet.ServiceRegistries == "" {
		return nil
	}

	// Parse service registries
	var serviceRegistries []generated.ServiceRegistry
	if err := json.Unmarshal([]byte(taskSet.ServiceRegistries), &serviceRegistries); err != nil {
		return fmt.Errorf("failed to parse service registries: %w", err)
	}

	// Register each pod IP with Service Discovery
	for _, registry := range serviceRegistries {
		for _, podIP := range podIPs {
			logging.Info("Registering TaskSet instance with Service Discovery",
				"taskSet", taskSet.ID,
				"registry", registry.RegistryArn,
				"podIP", podIP)

			// In a real implementation, this would call the Service Discovery API
			// to register the instance
			// For now, we'll just log the registration
		}
	}

	return nil
}

// UpdateServiceDiscoveryHealthStatus updates health status for Service Discovery instances
func (c *TaskSetConverterWithSD) UpdateServiceDiscoveryHealthStatus(
	ctx context.Context,
	taskSet *storage.TaskSet,
	instanceID string,
	healthy bool,
) error {
	if taskSet.ServiceRegistries == "" {
		return nil
	}

	status := "HEALTHY"
	if !healthy {
		status = "UNHEALTHY"
	}

	logging.Info("Updating Service Discovery instance health status",
		"taskSet", taskSet.ID,
		"instanceID", instanceID,
		"status", status)

	// In a real implementation, this would update the health status
	// in the Service Discovery registry

	return nil
}

// DeregisterTaskSetFromServiceDiscovery deregisters TaskSet instances from Service Discovery
func (c *TaskSetConverterWithSD) DeregisterTaskSetFromServiceDiscovery(
	ctx context.Context,
	taskSet *storage.TaskSet,
	instanceIDs []string,
) error {
	if taskSet.ServiceRegistries == "" {
		return nil
	}

	// Parse service registries
	var serviceRegistries []generated.ServiceRegistry
	if err := json.Unmarshal([]byte(taskSet.ServiceRegistries), &serviceRegistries); err != nil {
		return fmt.Errorf("failed to parse service registries: %w", err)
	}

	// Deregister each instance
	for _, registry := range serviceRegistries {
		for _, instanceID := range instanceIDs {
			logging.Info("Deregistering TaskSet instance from Service Discovery",
				"taskSet", taskSet.ID,
				"registry", registry.RegistryArn,
				"instanceID", instanceID)

			// In a real implementation, this would call the Service Discovery API
			// to deregister the instance
		}
	}

	return nil
}
