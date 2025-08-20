package kubernetes

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// TaskSetManager manages TaskSet resources in Kubernetes
type TaskSetManager struct {
	kubeClient       kubernetes.Interface
	taskSetConverter *converters.TaskSetConverter
	taskManager      *TaskManager
}

// NewTaskSetManager creates a new TaskSet manager
func NewTaskSetManager(kubeClient kubernetes.Interface, taskManager *TaskManager) *TaskSetManager {
	// Use default region and account ID for converter
	// These will be overridden by actual values in the TaskSet/Service objects
	taskConverter := converters.NewTaskConverter("us-east-1", "000000000000")
	taskSetConverter := converters.NewTaskSetConverter(taskConverter)

	return &TaskSetManager{
		kubeClient:       kubeClient,
		taskSetConverter: taskSetConverter,
		taskManager:      taskManager,
	}
}

// CreateTaskSet creates a new TaskSet in Kubernetes
func (m *TaskSetManager) CreateTaskSet(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	taskDef *storage.TaskDefinition,
	clusterName string,
) error {
	logging.Info("Creating TaskSet in Kubernetes",
		"taskSetId", taskSet.ID,
		"service", service.ServiceName,
		"cluster", clusterName)

	// Convert TaskSet to Deployment
	deployment, err := m.taskSetConverter.ConvertTaskSetToDeployment(taskSet, service, taskDef, clusterName)
	if err != nil {
		return fmt.Errorf("failed to convert TaskSet to deployment: %w", err)
	}

	// Create namespace if it doesn't exist
	namespace := m.taskSetConverter.GetNamespace(clusterName, taskSet.Region)
	if err := m.ensureNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create the deployment
	createdDeployment, err := m.kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			logging.Warn("Deployment already exists, updating instead",
				"deployment", deployment.Name,
				"namespace", namespace)
			// Update the existing deployment
			createdDeployment, err = m.kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update existing deployment: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	}

	logging.Info("Created deployment for TaskSet",
		"deployment", createdDeployment.Name,
		"namespace", namespace,
		"replicas", *createdDeployment.Spec.Replicas)

	// Create service if needed
	k8sService, err := m.taskSetConverter.ConvertTaskSetToService(taskSet, service, taskDef, clusterName, false)
	if err != nil {
		logging.Warn("Failed to convert TaskSet to service",
			"error", err)
	} else if k8sService != nil {
		_, err = m.kubeClient.CoreV1().Services(namespace).Create(ctx, k8sService, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			logging.Warn("Failed to create service for TaskSet",
				"service", k8sService.Name,
				"error", err)
		}
	}

	return nil
}

// UpdateTaskSet updates an existing TaskSet in Kubernetes
func (m *TaskSetManager) UpdateTaskSet(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	clusterName string,
) error {
	logging.Info("Updating TaskSet in Kubernetes",
		"taskSetId", taskSet.ID,
		"service", service.ServiceName,
		"cluster", clusterName)

	namespace := m.taskSetConverter.GetNamespace(clusterName, taskSet.Region)
	deploymentName := m.taskSetConverter.GetDeploymentName(service.ServiceName, taskSet.ID)

	// Get existing deployment
	deployment, err := m.kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			logging.Warn("Deployment not found for TaskSet update",
				"deployment", deploymentName,
				"namespace", namespace)
			// TaskSet doesn't exist in Kubernetes, we might need to create it
			// This can happen if the TaskSet was created before Kubernetes integration
			return nil
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update deployment scale
	m.taskSetConverter.UpdateDeploymentScale(deployment, taskSet, service)

	// Update deployment
	updatedDeployment, err := m.kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	logging.Info("Updated deployment for TaskSet",
		"deployment", updatedDeployment.Name,
		"namespace", namespace,
		"replicas", *updatedDeployment.Spec.Replicas)

	return nil
}

// DeleteTaskSet deletes a TaskSet from Kubernetes
func (m *TaskSetManager) DeleteTaskSet(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	clusterName string,
	force bool,
) error {
	logging.Info("Deleting TaskSet from Kubernetes",
		"taskSetId", taskSet.ID,
		"service", service.ServiceName,
		"cluster", clusterName,
		"force", force)

	namespace := m.taskSetConverter.GetNamespace(clusterName, taskSet.Region)
	deploymentName := m.taskSetConverter.GetDeploymentName(service.ServiceName, taskSet.ID)

	// Delete deployment
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	if force {
		// Use background deletion for force delete
		deletePolicy = metav1.DeletePropagationBackground
		deleteOptions.PropagationPolicy = &deletePolicy
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}

	err := m.kubeClient.AppsV1().Deployments(namespace).Delete(ctx, deploymentName, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete service if exists
	serviceName := m.taskSetConverter.GetServiceName(service.ServiceName, taskSet.ID)
	err = m.kubeClient.CoreV1().Services(namespace).Delete(ctx, serviceName, deleteOptions)
	if err != nil && !errors.IsNotFound(err) {
		logging.Warn("Failed to delete service for TaskSet",
			"service", serviceName,
			"error", err)
	}

	logging.Info("Deleted TaskSet resources from Kubernetes",
		"deployment", deploymentName,
		"namespace", namespace)

	return nil
}

// GetTaskSetStatus gets the current status of a TaskSet from Kubernetes
func (m *TaskSetManager) GetTaskSetStatus(
	ctx context.Context,
	taskSet *storage.TaskSet,
	service *storage.Service,
	clusterName string,
) (runningCount, pendingCount int64, stabilityStatus string, err error) {
	namespace := m.taskSetConverter.GetNamespace(clusterName, taskSet.Region)
	deploymentName := m.taskSetConverter.GetDeploymentName(service.ServiceName, taskSet.ID)

	// Get deployment
	deployment, err := m.kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// TaskSet doesn't exist in Kubernetes
			return 0, 0, "STEADY_STATE", nil
		}
		return 0, 0, "", fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get counts from deployment status
	runningCount, pendingCount = m.taskSetConverter.GetTaskSetStatusFromDeployment(deployment)

	// Determine stability status
	stabilityStatus = m.getStabilityStatus(deployment)

	return runningCount, pendingCount, stabilityStatus, nil
}

// SetPrimaryTaskSet updates services to route traffic to the primary TaskSet
func (m *TaskSetManager) SetPrimaryTaskSet(
	ctx context.Context,
	primaryTaskSet *storage.TaskSet,
	service *storage.Service,
	clusterName string,
) error {
	logging.Info("Setting primary TaskSet",
		"taskSetId", primaryTaskSet.ID,
		"service", service.ServiceName,
		"cluster", clusterName)

	namespace := m.taskSetConverter.GetNamespace(clusterName, primaryTaskSet.Region)

	// Update the main service selector to point to the primary TaskSet
	mainServiceName := strings.ToLower(service.ServiceName)
	mainService, err := m.kubeClient.CoreV1().Services(namespace).Get(ctx, mainServiceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the main service
			mainService = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      mainServiceName,
					Namespace: namespace,
					Labels: map[string]string{
						"kecs.io/cluster": clusterName,
						"kecs.io/service": service.ServiceName,
						"kecs.io/role":    "main-service",
						"kecs.io/managed": "true",
					},
				},
				Spec: corev1.ServiceSpec{
					Selector: map[string]string{
						"kecs.io/taskset": primaryTaskSet.ID,
					},
					Type: corev1.ServiceTypeClusterIP,
				},
			}

			// Get ports from TaskSet service
			taskSetServiceName := m.taskSetConverter.GetServiceName(service.ServiceName, primaryTaskSet.ID)
			taskSetService, err := m.kubeClient.CoreV1().Services(namespace).Get(ctx, taskSetServiceName, metav1.GetOptions{})
			if err == nil && taskSetService != nil {
				mainService.Spec.Ports = taskSetService.Spec.Ports
			}

			_, err = m.kubeClient.CoreV1().Services(namespace).Create(ctx, mainService, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create main service: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get main service: %w", err)
		}
	} else {
		// Update existing service selector
		mainService.Spec.Selector = map[string]string{
			"kecs.io/taskset": primaryTaskSet.ID,
		}
		mainService.Labels["kecs.io/primary-taskset"] = primaryTaskSet.ID

		_, err = m.kubeClient.CoreV1().Services(namespace).Update(ctx, mainService, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update main service: %w", err)
		}
	}

	logging.Info("Updated main service to route to primary TaskSet",
		"service", mainServiceName,
		"taskSet", primaryTaskSet.ID)

	return nil
}

// Helper functions

func (m *TaskSetManager) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := m.kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						"kecs.io/managed": "true",
					},
				},
			}
			_, err = m.kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				return fmt.Errorf("failed to create namespace: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	return nil
}

func (m *TaskSetManager) getStabilityStatus(deployment *appsv1.Deployment) string {
	// Check deployment conditions
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing {
			if condition.Status == corev1.ConditionTrue &&
				condition.Reason == "NewReplicaSetAvailable" {
				return "STEADY_STATE"
			}
			if condition.Status == corev1.ConditionTrue {
				return "STABILIZING"
			}
		}
		if condition.Type == appsv1.DeploymentReplicaFailure {
			if condition.Status == corev1.ConditionTrue {
				return "UNSTABLE"
			}
		}
	}

	// Check replica status
	if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
		return "STEADY_STATE"
	}

	if deployment.Status.ReadyReplicas < *deployment.Spec.Replicas {
		return "STABILIZING"
	}

	return "STEADY_STATE"
}

// ListTaskSetsForService lists all TaskSets for a service
func (m *TaskSetManager) ListTaskSetsForService(
	ctx context.Context,
	service *storage.Service,
	clusterName string,
	region string,
) ([]*TaskSetInfo, error) {
	namespace := m.taskSetConverter.GetNamespace(clusterName, region)

	// List deployments with service label
	labelSelector := fmt.Sprintf("kecs.io/service=%s,kecs.io/role=taskset", service.ServiceName)
	deployments, err := m.kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var taskSets []*TaskSetInfo
	for _, deployment := range deployments.Items {
		taskSetID := deployment.Labels["kecs.io/taskset"]
		externalID := deployment.Labels["kecs.io/taskset-external-id"]

		runningCount, pendingCount := m.taskSetConverter.GetTaskSetStatusFromDeployment(&deployment)
		stabilityStatus := m.getStabilityStatus(&deployment)

		taskSets = append(taskSets, &TaskSetInfo{
			TaskSetID:       taskSetID,
			ExternalID:      externalID,
			RunningCount:    runningCount,
			PendingCount:    pendingCount,
			StabilityStatus: stabilityStatus,
		})
	}

	return taskSets, nil
}

// TaskSetInfo contains information about a TaskSet in Kubernetes
type TaskSetInfo struct {
	TaskSetID       string
	ExternalID      string
	RunningCount    int64
	PendingCount    int64
	StabilityStatus string
}

// UpdatePrimaryTaskSet updates the primary TaskSet labels and annotations
func (m *TaskSetManager) UpdatePrimaryTaskSet(
	ctx context.Context,
	primaryTaskSet *storage.TaskSet,
	service *storage.Service,
	clusterName string,
) error {
	logging.Info("Updating primary TaskSet in Kubernetes",
		"taskSetId", primaryTaskSet.ID,
		"service", service.ServiceName,
		"cluster", clusterName)

	namespace := m.taskSetConverter.GetNamespace(clusterName, primaryTaskSet.Region)

	// List all deployments for this service
	labelSelector := fmt.Sprintf("kecs.io/service=%s", service.ServiceName)
	deployments, err := m.kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	// Update all deployments to mark which one is primary
	for _, deployment := range deployments.Items {
		taskSetID := deployment.Labels["kecs.io/taskset"]
		isPrimary := taskSetID == primaryTaskSet.ID

		// Update labels to indicate primary status
		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
		}
		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}

		if isPrimary {
			deployment.Labels["kecs.io/primary"] = "true"
			deployment.Annotations["kecs.io/primary-taskset"] = "true"
		} else {
			delete(deployment.Labels, "kecs.io/primary")
			delete(deployment.Annotations, "kecs.io/primary-taskset")
		}

		// Update the deployment
		_, err := m.kubeClient.AppsV1().Deployments(namespace).Update(ctx, &deployment, metav1.UpdateOptions{})
		if err != nil {
			logging.Warn("Failed to update deployment primary status",
				"deployment", deployment.Name,
				"error", err)
			// Continue with other deployments
		}
	}

	// Also call SetPrimaryTaskSet to update service selectors if needed
	return m.SetPrimaryTaskSet(ctx, primaryTaskSet, service, clusterName)
}
