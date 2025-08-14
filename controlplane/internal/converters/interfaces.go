package converters

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// ServiceConverterInterface defines the interface for service converters
type ServiceConverterInterface interface {
	// ConvertServiceToDeployment converts an ECS service to a Kubernetes Deployment
	ConvertServiceToDeployment(
		service *storage.Service,
		taskDef *storage.TaskDefinition,
		cluster *storage.Cluster,
	) (*appsv1.Deployment, *corev1.Service, error)

	// ConvertServiceToDeploymentWithNetworkConfig converts an ECS service to a Kubernetes Deployment with network configuration
	ConvertServiceToDeploymentWithNetworkConfig(
		service *storage.Service,
		taskDef *storage.TaskDefinition,
		cluster *storage.Cluster,
		networkConfig *generated.NetworkConfiguration,
	) (*appsv1.Deployment, *corev1.Service, error)
}
