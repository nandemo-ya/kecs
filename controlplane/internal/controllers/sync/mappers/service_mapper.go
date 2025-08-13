package mappers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// ServiceStateMapper maps Kubernetes deployment state to ECS service state
type ServiceStateMapper struct {
	accountID string
	region    string
}

// NewServiceStateMapper creates a new service state mapper
func NewServiceStateMapper(accountID, region string) *ServiceStateMapper {
	return &ServiceStateMapper{
		accountID: accountID,
		region:    region,
	}
}

// MapDeploymentToServiceStatus maps a Kubernetes deployment to ECS service status
func (m *ServiceStateMapper) MapDeploymentToServiceStatus(deployment *appsv1.Deployment) string {
	if deployment == nil {
		return "INACTIVE"
	}

	// Check if deployment is being deleted
	if deployment.DeletionTimestamp != nil {
		return "DRAINING"
	}

	// Check deployment conditions for failures
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse {
			// Check if it's a progress deadline exceeded
			if strings.Contains(condition.Reason, "ProgressDeadlineExceeded") {
				return "FAILED"
			}
		}
		if condition.Type == appsv1.DeploymentReplicaFailure && condition.Status == corev1.ConditionTrue {
			return "FAILED"
		}
	}

	// Map based on replica counts
	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	if replicas == 0 {
		return "DRAINING"
	}

	readyReplicas := deployment.Status.ReadyReplicas
	updatedReplicas := deployment.Status.UpdatedReplicas

	// All replicas are ready and updated
	if readyReplicas == replicas && updatedReplicas == replicas {
		return "ACTIVE"
	}

	// No ready replicas yet
	if readyReplicas == 0 && replicas > 0 {
		return "PROVISIONING"
	}

	// Some replicas are ready but not all, or update in progress
	if readyReplicas < replicas || updatedReplicas < replicas {
		return "UPDATING"
	}

	return "ACTIVE"
}

// MapDeploymentToServiceCounts extracts desired, running, and pending counts from deployment
func (m *ServiceStateMapper) MapDeploymentToServiceCounts(deployment *appsv1.Deployment) (desired, running, pending int32) {
	if deployment == nil {
		return 0, 0, 0
	}

	// Desired count from spec
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	// Running count is ready replicas
	running = deployment.Status.ReadyReplicas

	// Pending count is total replicas minus ready replicas
	totalReplicas := deployment.Status.Replicas
	if totalReplicas > running {
		pending = totalReplicas - running
	}

	return desired, running, pending
}

// ExtractServiceNameFromDeployment extracts the ECS service name from deployment name
func (m *ServiceStateMapper) ExtractServiceNameFromDeployment(deploymentName string) string {
	// Remove the "ecs-service-" prefix if present
	return strings.TrimPrefix(deploymentName, "ecs-service-")
}

// ExtractClusterInfoFromNamespace extracts cluster name and region from namespace
func (m *ServiceStateMapper) ExtractClusterInfoFromNamespace(namespace string) (clusterName, region string) {
	// Expected format: <cluster-name>-<region>
	// Example: default-us-east-1
	parts := strings.Split(namespace, "-")
	if len(parts) >= 3 {
		// Extract region (last 3 parts: us-east-1)
		region = strings.Join(parts[len(parts)-3:], "-")
		// Extract cluster name (everything before region)
		clusterName = strings.Join(parts[:len(parts)-3], "-")
		// Handle case where cluster name is "default"
		if clusterName == "" && len(parts) == 3 {
			clusterName = parts[0]
		}
	}

	return clusterName, region
}

// MapDeploymentToService creates an ECS service object from a deployment
func (m *ServiceStateMapper) MapDeploymentToService(deployment *appsv1.Deployment, existingService *storage.Service) *storage.Service {
	if deployment == nil {
		return nil
	}

	serviceName := m.ExtractServiceNameFromDeployment(deployment.Name)
	clusterName, region := m.ExtractClusterInfoFromNamespace(deployment.Namespace)

	// Start with existing service or create new one
	service := existingService
	if service == nil {
		service = &storage.Service{
			ServiceName: serviceName,
			ClusterARN:  m.generateClusterARN(region, clusterName),
			Region:      region,
			CreatedAt:   deployment.CreationTimestamp.Time,
			LaunchType:  "FARGATE", // Default launch type
		}
	}

	// Update status and counts
	service.Status = m.MapDeploymentToServiceStatus(deployment)
	desired, running, pending := m.MapDeploymentToServiceCounts(deployment)
	service.DesiredCount = int(desired)
	service.RunningCount = int(running)
	service.PendingCount = int(pending)

	// Update timestamps
	service.UpdatedAt = time.Now()

	// Extract task definition from deployment annotations
	if taskDef, exists := deployment.Annotations["ecs.amazonaws.com/task-definition"]; exists {
		service.TaskDefinitionARN = taskDef
	}

	// Create deployment info to store in DeploymentConfiguration field
	deploymentInfo := generated.Deployment{
		Id:              &deployment.Name,
		Status:          &service.Status,
		TaskDefinition:  &service.TaskDefinitionARN,
		DesiredCount:    int32Ptr(int32(service.DesiredCount)),
		RunningCount:    int32Ptr(int32(service.RunningCount)),
		PendingCount:    int32Ptr(int32(service.PendingCount)),
		CreatedAt:       timePtr(deployment.CreationTimestamp.Time),
		UpdatedAt:       timePtr(time.Now()),
		LaunchType:      (*generated.LaunchType)(stringPtr(service.LaunchType)),
		PlatformVersion: stringPtr(service.PlatformVersion),
	}

	// Serialize deployment info to JSON for storage
	deployments := []generated.Deployment{deploymentInfo}
	if _, err := json.Marshal(deployments); err == nil {
		// Store deployments in a field - we could use the DeploymentConfiguration field
		// or add metadata to track this information
		// For now, we'll store it in DeploymentConfiguration as it's JSON
		deploymentConfig := map[string]interface{}{
			"deployments": deployments,
		}
		if configJSON, err := json.Marshal(deploymentConfig); err == nil {
			service.DeploymentConfiguration = string(configJSON)
		}
	}

	return service
}

// generateClusterARN generates an ECS cluster ARN

// Helper functions for pointer conversions
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// generateClusterARN generates an ECS cluster ARN
func (m *ServiceStateMapper) generateClusterARN(region, clusterName string) string {
	if region == "" {
		region = m.region
	}
	return "arn:aws:ecs:" + region + ":" + m.accountID + ":cluster/" + clusterName
}
