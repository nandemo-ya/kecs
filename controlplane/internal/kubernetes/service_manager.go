package kubernetes

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// ServiceManager manages Kubernetes Deployments and Services for ECS services
type ServiceManager struct {
	storage     storage.Storage
	kindManager *KindManager
}

// NewServiceManager creates a new ServiceManager
func NewServiceManager(storage storage.Storage, kindManager *KindManager) *ServiceManager {
	return &ServiceManager{
		storage:     storage,
		kindManager: kindManager,
	}
}

// CreateService creates a Kubernetes Deployment and Service for an ECS service
func (sm *ServiceManager) CreateService(
	ctx context.Context,
	deployment *appsv1.Deployment,
	kubeService *corev1.Service,
	cluster *storage.Cluster,
	storageService *storage.Service,
) error {
	// Get Kubernetes client for the cluster
	kubeClient, err := sm.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Ensure namespace exists
	if err := sm.ensureNamespace(ctx, kubeClient, deployment.Namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create Deployment
	if err := sm.createDeployment(ctx, kubeClient, deployment); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create Service (if provided)
	if kubeService != nil {
		if err := sm.createKubernetesService(ctx, kubeClient, kubeService); err != nil {
			// Don't fail the entire operation if service creation fails
			log.Printf("Warning: failed to create kubernetes service: %v", err)
		}
	}

	// Update service status to ACTIVE after successful deployment using transaction
	if err := sm.updateServiceStatusSafely(ctx, cluster, storageService); err != nil {
		log.Printf("Warning: failed to update service status to ACTIVE: %v", err)
		// Don't fail the entire operation for status update issues
	} else {
		log.Printf("Updated service status to ACTIVE for service %s (ID: %s)", 
			storageService.ServiceName, storageService.ID)
	}

	log.Printf("Successfully created service %s as deployment %s in namespace %s",
		storageService.ServiceName, deployment.Name, deployment.Namespace)

	return nil
}

// UpdateService updates a Kubernetes Deployment for an ECS service
func (sm *ServiceManager) UpdateService(
	ctx context.Context,
	deployment *appsv1.Deployment,
	cluster *storage.Cluster,
	storageService *storage.Service,
) error {
	// Get Kubernetes client for the cluster
	kubeClient, err := sm.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Update Deployment
	_, err = kubeClient.AppsV1().Deployments(deployment.Namespace).Update(
		ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// Update storage service
	storageService.Status = "ACTIVE"
	if err := sm.storage.ServiceStore().Update(ctx, storageService); err != nil {
		log.Printf("Warning: failed to update service in storage: %v", err)
	}

	log.Printf("Successfully updated service %s deployment %s in namespace %s",
		storageService.ServiceName, deployment.Name, deployment.Namespace)

	return nil
}

// DeleteService deletes a Kubernetes Deployment and Service for an ECS service
func (sm *ServiceManager) DeleteService(
	ctx context.Context,
	cluster *storage.Cluster,
	storageService *storage.Service,
) error {
	// Get Kubernetes client for the cluster
	kubeClient, err := sm.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	namespace := storageService.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	}

	deploymentName := storageService.DeploymentName
	if deploymentName == "" {
		deploymentName = fmt.Sprintf("ecs-service-%s", storageService.ServiceName)
	}

	// Delete Deployment
	err = kubeClient.AppsV1().Deployments(namespace).Delete(
		ctx, deploymentName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Warning: failed to delete deployment %s: %v", deploymentName, err)
	}

	// Delete Service (if exists)
	serviceName := fmt.Sprintf("ecs-service-%s", storageService.ServiceName)
	err = kubeClient.CoreV1().Services(namespace).Delete(
		ctx, serviceName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		log.Printf("Warning: failed to delete kubernetes service %s: %v", serviceName, err)
	}

	log.Printf("Successfully deleted service %s deployment and service in namespace %s",
		storageService.ServiceName, namespace)

	return nil
}

// GetServiceStatus gets the current status of a Kubernetes Deployment
func (sm *ServiceManager) GetServiceStatus(
	ctx context.Context,
	cluster *storage.Cluster,
	storageService *storage.Service,
) (*ServiceStatus, error) {
	// Get Kubernetes client for the cluster
	kubeClient, err := sm.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	namespace := storageService.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	}

	deploymentName := storageService.DeploymentName
	if deploymentName == "" {
		deploymentName = fmt.Sprintf("ecs-service-%s", storageService.ServiceName)
	}

	// Get Deployment status
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(
		ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return &ServiceStatus{
				Status:       "INACTIVE",
				RunningCount: 0,
				PendingCount: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	status := &ServiceStatus{
		Status:       "ACTIVE",
		DesiredCount: int(*deployment.Spec.Replicas),
		RunningCount: int(deployment.Status.ReadyReplicas),
		PendingCount: int(deployment.Status.Replicas - deployment.Status.ReadyReplicas),
	}

	// Determine status based on deployment conditions
	if deployment.Status.Replicas == 0 {
		status.Status = "INACTIVE"
	} else if deployment.Status.ReadyReplicas == 0 {
		status.Status = "PENDING"
	} else if deployment.Status.ReadyReplicas < deployment.Status.Replicas {
		status.Status = "UPDATING"
	} else {
		status.Status = "ACTIVE"
	}

	return status, nil
}

// ensureNamespace creates a namespace if it doesn't exist
func (sm *ServiceManager) ensureNamespace(ctx context.Context, kubeClient kubernetes.Interface, namespace string) error {
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						"kecs.dev/managed-by": "kecs",
					},
				},
			}
			_, err = kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace: %w", err)
			}
			log.Printf("Created namespace: %s", namespace)
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	return nil
}

// createDeployment creates a Kubernetes Deployment
func (sm *ServiceManager) createDeployment(ctx context.Context, kubeClient kubernetes.Interface, deployment *appsv1.Deployment) error {
	_, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Create(
		ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing deployment
			_, err = kubeClient.AppsV1().Deployments(deployment.Namespace).Update(
				ctx, deployment, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update existing deployment: %w", err)
			}
			log.Printf("Updated existing deployment: %s", deployment.Name)
		} else {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	} else {
		log.Printf("Created deployment: %s", deployment.Name)
	}
	return nil
}

// createKubernetesService creates a Kubernetes Service
func (sm *ServiceManager) createKubernetesService(ctx context.Context, kubeClient kubernetes.Interface, service *corev1.Service) error {
	_, err := kubeClient.CoreV1().Services(service.Namespace).Create(
		ctx, service, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing service
			_, err = kubeClient.CoreV1().Services(service.Namespace).Update(
				ctx, service, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update existing service: %w", err)
			}
			log.Printf("Updated existing kubernetes service: %s", service.Name)
		} else {
			return fmt.Errorf("failed to create kubernetes service: %w", err)
		}
	} else {
		log.Printf("Created kubernetes service: %s", service.Name)
	}
	return nil
}

// updateServiceStatusSafely updates service status with safe error handling
func (sm *ServiceManager) updateServiceStatusSafely(ctx context.Context, cluster *storage.Cluster, storageService *storage.Service) error {
	log.Printf("DEBUG: Starting status update for service %s (ID: %s)", storageService.ServiceName, storageService.ID)
	
	// Get fresh service data from storage to avoid conflicts
	freshService, err := sm.storage.ServiceStore().Get(ctx, cluster.ARN, storageService.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get fresh service for status update: %w", err)
	}
	
	log.Printf("DEBUG: Retrieved fresh service ID: %s, current status: %s", freshService.ID, freshService.Status)
	
	// Update only the status field
	freshService.Status = "ACTIVE"
	
	log.Printf("DEBUG: About to update service ID: %s to status: ACTIVE", freshService.ID)
	
	// Use a simple approach: since we have the fresh service, just update it
	return sm.storage.ServiceStore().Update(ctx, freshService)
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Status       string
	DesiredCount int
	RunningCount int
	PendingCount int
}