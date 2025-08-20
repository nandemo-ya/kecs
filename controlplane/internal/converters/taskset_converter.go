package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// TaskSetConverter converts TaskSet to Kubernetes resources
type TaskSetConverter struct {
	taskConverter *TaskConverter
}

// NewTaskSetConverter creates a new TaskSet converter
func NewTaskSetConverter(taskConverter *TaskConverter) *TaskSetConverter {
	return &TaskSetConverter{
		taskConverter: taskConverter,
	}
}

// ConvertTaskSetToDeployment converts a TaskSet to a Kubernetes Deployment
func (c *TaskSetConverter) ConvertTaskSetToDeployment(
	taskSet *storage.TaskSet,
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	clusterName string,
) (*appsv1.Deployment, error) {
	// Parse network configuration
	var networkConfig *generated.NetworkConfiguration
	if taskSet.NetworkConfiguration != "" {
		if err := json.Unmarshal([]byte(taskSet.NetworkConfiguration), &networkConfig); err != nil {
			return nil, fmt.Errorf("failed to parse network configuration: %w", err)
		}
	}

	// Create RunTask request JSON for converter
	runTaskReq := map[string]interface{}{
		"taskDefinition": taskSet.TaskDefinition,
		"cluster":        clusterName,
		"launchType":     taskSet.LaunchType,
	}

	if networkConfig != nil {
		runTaskReq["networkConfiguration"] = networkConfig
	}

	if taskSet.PlatformVersion != "" {
		runTaskReq["platformVersion"] = taskSet.PlatformVersion
	}

	runTaskJSON, err := json.Marshal(runTaskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run task request: %w", err)
	}

	// Get cluster object
	clusterObj := &storage.Cluster{
		Name:      clusterName,
		Region:    taskSet.Region,
		AccountID: taskSet.AccountID,
	}

	// Generate task ID for the deployment
	taskID := fmt.Sprintf("taskset-%s-%s", service.ServiceName, taskSet.ID)

	// Convert task to pod using existing converter
	pod, err := c.taskConverter.ConvertTaskToPod(taskDef, runTaskJSON, clusterObj, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert task to pod: %w", err)
	}

	// Create deployment from pod template
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetDeploymentName(service.ServiceName, taskSet.ID),
			Namespace: pod.Namespace,
			Labels: map[string]string{
				"kecs.io/cluster":             clusterName,
				"kecs.io/service":             service.ServiceName,
				"kecs.io/taskset":             taskSet.ID,
				"kecs.io/taskset-external-id": taskSet.ExternalID,
				"kecs.io/role":                "taskset",
				"kecs.io/managed":             "true",
			},
			Annotations: map[string]string{
				"kecs.io/taskset-arn":      taskSet.ARN,
				"kecs.io/service-arn":      taskSet.ServiceARN,
				"kecs.io/task-definition":  taskSet.TaskDefinition,
				"kecs.io/launch-type":      taskSet.LaunchType,
				"kecs.io/platform-version": taskSet.PlatformVersion,
				"kecs.io/stability-status": taskSet.StabilityStatus,
				"kecs.io/status":           taskSet.Status,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: c.GetReplicas(taskSet, service),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kecs.io/taskset": taskSet.ID,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kecs.io/cluster":             clusterName,
						"kecs.io/service":             service.ServiceName,
						"kecs.io/taskset":             taskSet.ID,
						"kecs.io/taskset-external-id": taskSet.ExternalID,
						"kecs.io/role":                "taskset-pod",
					},
					Annotations: pod.Annotations,
				},
				Spec: func() corev1.PodSpec {
					// Copy pod spec and adjust for deployment
					podSpec := pod.Spec
					// Deployments require RestartPolicy to be Always
					podSpec.RestartPolicy = corev1.RestartPolicyAlways
					return podSpec
				}(),
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: intOrStringPtr("25%"),
					MaxSurge:       intOrStringPtr("25%"),
				},
			},
			ProgressDeadlineSeconds: int32Ptr(600), // 10 minutes
		},
	}

	// Add load balancer labels if configured
	if taskSet.LoadBalancers != "" {
		deployment.Spec.Template.Labels["kecs.io/load-balancer-enabled"] = "true"
	}

	// Add service registry labels if configured
	if taskSet.ServiceRegistries != "" {
		deployment.Spec.Template.Labels["kecs.io/service-discovery-enabled"] = "true"
	}

	return deployment, nil
}

// ConvertTaskSetToService creates a Kubernetes Service for the TaskSet if needed
func (c *TaskSetConverter) ConvertTaskSetToService(
	taskSet *storage.TaskSet,
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	clusterName string,
	isPrimary bool,
) (*corev1.Service, error) {
	// Parse container definitions from task definition
	var containerDefinitions []generated.ContainerDefinition
	if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefinitions); err != nil {
		return nil, fmt.Errorf("failed to parse container definitions: %w", err)
	}

	// Find port mappings
	var ports []corev1.ServicePort
	for _, container := range containerDefinitions {
		if container.PortMappings != nil {
			for _, pm := range container.PortMappings {
				protocol := "tcp"
				if pm.Protocol != nil {
					protocol = strings.ToLower(string(*pm.Protocol))
				}
				containerPort := int32(80) // default
				if pm.ContainerPort != nil {
					containerPort = *pm.ContainerPort
				}
				port := corev1.ServicePort{
					Name:     fmt.Sprintf("%s-%d", protocol, containerPort),
					Port:     containerPort,
					Protocol: corev1.ProtocolTCP,
				}
				if strings.ToUpper(protocol) == "UDP" {
					port.Protocol = corev1.ProtocolUDP
				}
				ports = append(ports, port)
			}
		}
	}

	// If no ports, don't create a service unless load balancer is configured
	if len(ports) == 0 && taskSet.LoadBalancers == "" {
		return nil, nil
	}

	// Determine service type based on load balancer configuration
	serviceType := corev1.ServiceTypeClusterIP
	if taskSet.LoadBalancers != "" {
		// Parse load balancers to check if external load balancer is needed
		var loadBalancers []generated.LoadBalancer
		if err := json.Unmarshal([]byte(taskSet.LoadBalancers), &loadBalancers); err == nil && len(loadBalancers) > 0 {
			// If load balancer is configured, create LoadBalancer type service
			serviceType = corev1.ServiceTypeLoadBalancer
		}
	}

	// Create service
	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.GetServiceName(service.ServiceName, taskSet.ID),
			Namespace: c.GetNamespace(clusterName, taskSet.Region),
			Labels: map[string]string{
				"kecs.io/cluster":             clusterName,
				"kecs.io/service":             service.ServiceName,
				"kecs.io/taskset":             taskSet.ID,
				"kecs.io/taskset-external-id": taskSet.ExternalID,
				"kecs.io/role":                "taskset-service",
				"kecs.io/managed":             "true",
			},
			Annotations: map[string]string{
				"kecs.io/taskset-arn":     taskSet.ARN,
				"kecs.io/service-arn":     taskSet.ServiceARN,
				"kecs.io/taskset-primary": fmt.Sprintf("%t", isPrimary),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"kecs.io/taskset": taskSet.ID,
			},
			Ports: ports,
			Type:  serviceType,
		},
	}

	// If this is the primary TaskSet, also update the main service selector
	if isPrimary {
		k8sService.Labels["kecs.io/primary"] = "true"
	}

	// Add load balancer annotations if configured
	if taskSet.LoadBalancers != "" {
		k8sService.Annotations["kecs.io/load-balancers"] = taskSet.LoadBalancers
		// Add annotations for common cloud providers
		if serviceType == corev1.ServiceTypeLoadBalancer {
			// For k3d/Traefik, we can use annotations
			k8sService.Annotations["metallb.universe.tf/loadBalancerIPs"] = "172.18.255.200" // Example IP for local testing
		}
	}

	return k8sService, nil
}

// GetReplicas calculates the desired replica count for the TaskSet
func (c *TaskSetConverter) GetReplicas(taskSet *storage.TaskSet, service *storage.Service) *int32 {
	// Parse scale configuration
	if taskSet.Scale != "" {
		var scale generated.Scale
		if err := json.Unmarshal([]byte(taskSet.Scale), &scale); err == nil {
			if scale.Value != nil && scale.Unit != nil {
				switch *scale.Unit {
				case generated.ScaleUnit("PERCENT"):
					// Calculate based on percentage of service desired count
					replicas := int32(float64(service.DesiredCount) * (*scale.Value / 100.0))
					return &replicas
				case generated.ScaleUnit("COUNT"):
					// Use absolute count
					replicas := int32(*scale.Value)
					return &replicas
				}
			}
		}
	}

	// Default to computed desired count if available
	if taskSet.ComputedDesiredCount > 0 {
		replicas := int32(taskSet.ComputedDesiredCount)
		return &replicas
	}

	// Fall back to 0 replicas
	replicas := int32(0)
	return &replicas
}

// GetDeploymentName generates the deployment name for a TaskSet
func (c *TaskSetConverter) GetDeploymentName(serviceName, taskSetID string) string {
	// Ensure the name is valid for Kubernetes
	name := fmt.Sprintf("%s-%s", serviceName, taskSetID)
	// Replace underscores with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	// Convert to lowercase
	name = strings.ToLower(name)
	// Truncate if too long (max 63 characters)
	if len(name) > 63 {
		name = name[:63]
	}
	// Remove trailing hyphens
	name = strings.TrimSuffix(name, "-")
	return name
}

// GetServiceName generates the service name for a TaskSet
func (c *TaskSetConverter) GetServiceName(serviceName, taskSetID string) string {
	return c.GetDeploymentName(serviceName, taskSetID) + "-svc"
}

// GetNamespace returns the namespace for the given cluster and region
func (c *TaskSetConverter) GetNamespace(clusterName, region string) string {
	// Use the same namespace pattern as tasks
	return fmt.Sprintf("%s-%s", clusterName, region)
}

// UpdateDeploymentScale updates the replica count of a deployment
func (c *TaskSetConverter) UpdateDeploymentScale(deployment *appsv1.Deployment, taskSet *storage.TaskSet, service *storage.Service) {
	deployment.Spec.Replicas = c.GetReplicas(taskSet, service)
}

// GetTaskSetStatusFromDeployment extracts TaskSet status from deployment
func (c *TaskSetConverter) GetTaskSetStatusFromDeployment(deployment *appsv1.Deployment) (runningCount, pendingCount int64) {
	if deployment.Status.ReadyReplicas > 0 {
		runningCount = int64(deployment.Status.ReadyReplicas)
	}

	if deployment.Status.Replicas > deployment.Status.ReadyReplicas {
		pendingCount = int64(deployment.Status.Replicas - deployment.Status.ReadyReplicas)
	}

	return runningCount, pendingCount
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}

func intOrStringPtr(s string) *intstr.IntOrString {
	val := intstr.FromString(s)
	return &val
}
