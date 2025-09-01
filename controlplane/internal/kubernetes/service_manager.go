package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// ServiceManager manages Kubernetes Deployments and Services for ECS services
type ServiceManager struct {
	storage     storage.Storage
	clientset   kubernetes.Interface
	taskManager *TaskManager
}

// SetTaskManager sets or updates the task manager
func (sm *ServiceManager) SetTaskManager(taskManager *TaskManager) {
	sm.taskManager = taskManager
	logging.Info("ServiceManager TaskManager updated")
}

// NewServiceManager creates a new ServiceManager
func NewServiceManager(storage storage.Storage) *ServiceManager {
	// Create TaskManager if storage is available
	var taskManager *TaskManager
	if storage != nil {
		tm, err := NewTaskManager(storage)
		if err != nil {
			logging.Warn("Failed to create TaskManager for ServiceManager", "error", err)
		} else {
			taskManager = tm
		}
	}

	sm := &ServiceManager{
		storage:     storage,
		taskManager: taskManager,
	}

	// Don't initialize kubernetes client here - it will be initialized on first use
	// This ensures we get the correct in-cluster config when running inside Kubernetes
	logging.Debug("ServiceManager created, kubernetes client will be initialized on first use")

	// Process existing pods after a delay to allow proper initialization
	go func() {
		time.Sleep(10 * time.Second) // Wait for server to be fully initialized
		sm.processExistingPods()
	}()

	return sm
}

// processExistingPods processes existing pods to ensure they are tracked
func (sm *ServiceManager) processExistingPods() {
	// Wait a bit for initialization to complete
	time.Sleep(5 * time.Second)

	// Initialize client if needed
	if sm.clientset == nil {
		if err := sm.initializeClient(); err != nil {
			logging.Warn("Failed to initialize client for existing pods processing", "error", err)
			return
		}
	}

	ctx := context.Background()

	// List all namespaces
	namespaces, err := sm.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		logging.Warn("Failed to list namespaces for existing pods processing", "error", err)
		return
	}

	processedCount := 0
	for _, ns := range namespaces.Items {
		// Skip system namespaces
		if ns.Name == "kube-system" || ns.Name == "kube-public" || ns.Name == "kube-node-lease" || ns.Name == "kecs-system" {
			continue
		}

		// List pods in namespace
		pods, err := sm.clientset.CoreV1().Pods(ns.Name).List(ctx, metav1.ListOptions{})
		if err != nil {
			logging.Warn("Failed to list pods in namespace", "namespace", ns.Name, "error", err)
			continue
		}

		for _, pod := range pods.Items {
			// Check if pod is managed by ECS
			if _, exists := pod.Labels["ecs.amazonaws.com/task-arn"]; !exists {
				continue
			}

			// Check if pod is running
			if pod.Status.Phase != corev1.PodRunning {
				continue
			}

			// Get service name from labels
			serviceName := pod.Labels["ecs.amazonaws.com/service-name"]
			if serviceName == "" {
				serviceName = pod.Labels["kecs.dev/service"]
			}
			if serviceName == "" {
				continue
			}

			// Extract cluster name from namespace
			clusterName := "default"
			if strings.Contains(ns.Name, "-") {
				parts := strings.Split(ns.Name, "-")
				if len(parts) > 0 {
					clusterName = parts[0]
				}
			}

			// Get cluster and service from storage
			cluster, err := sm.storage.ClusterStore().Get(ctx, clusterName)
			if err != nil || cluster == nil {
				logging.Debug("Cluster not found for existing pod", "cluster", clusterName, "pod", pod.Name)
				continue
			}

			service, err := sm.storage.ServiceStore().Get(ctx, clusterName, serviceName)
			if err != nil || service == nil {
				logging.Debug("Service not found for existing pod", "service", serviceName, "pod", pod.Name)
				continue
			}

			// Process the pod
			logging.Info("Processing existing pod", "pod", pod.Name, "namespace", ns.Name, "service", serviceName)
			sm.registerPodAsTask(ctx, &pod, cluster, service)
			processedCount++
		}
	}

	if processedCount > 0 {
		logging.Info("Processed existing pods", "count", processedCount)
	}
}

// initializeClient initializes the kubernetes client
func (sm *ServiceManager) initializeClient() error {
	if sm.clientset != nil {
		logging.Debug("ServiceManager client already initialized")
		return nil // Already initialized
	}

	// Check if running in test mode
	if config.GetBool("features.testMode") {
		logging.Debug("Test mode enabled - skipping kubernetes client initialization")
		return nil
	}

	// Control Plane always runs inside Kubernetes, use in-cluster config
	logging.Info("Initializing Kubernetes client for ServiceManager")

	cfg, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	// Adjust config for better performance
	cfg.QPS = 100
	cfg.Burst = 200
	logging.Info("Successfully obtained in-cluster config", "host", cfg.Host)

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	sm.clientset = clientset
	logging.Info("ServiceManager kubernetes client initialized with in-cluster config")
	return nil
}

// CreateService creates a Kubernetes Deployment and Service for an ECS service
func (sm *ServiceManager) CreateService(
	ctx context.Context,
	deployment *appsv1.Deployment,
	kubeService *corev1.Service,
	cluster *storage.Cluster,
	storageService *storage.Service,
) error {
	logging.Info("ServiceManager.CreateService called",
		"service", storageService.ServiceName,
		"cluster", cluster.Name,
		"namespace", deployment.Namespace)
	// Check if running in test mode
	if config.GetBool("features.testMode") {
		// In test mode, simulate service creation
		logging.Debug("TEST MODE: Simulated service creation",
			"serviceName", storageService.ServiceName,
			"namespace", deployment.Namespace)

		// Update service status to ACTIVE
		storageService.Status = "ACTIVE"
		storageService.RunningCount = storageService.DesiredCount
		storageService.PendingCount = 0

		// Simulate creating initial tasks for the service
		if sm.storage != nil && storageService.DesiredCount > 0 {
			taskStore := sm.storage.TaskStore()
			for i := 0; i < storageService.DesiredCount; i++ {
				// Generate task ID and ARN
				taskID := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s-%d",
					storageService.Region, storageService.AccountID, cluster.Name,
					storageService.ServiceName, i)

				task := &storage.Task{
					ARN:               taskID,
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: storageService.TaskDefinitionARN,
					DesiredStatus:     "RUNNING",
					LastStatus:        "RUNNING",
					LaunchType:        storageService.LaunchType,
					StartedBy:         fmt.Sprintf("ecs-svc/%s", storageService.ServiceName),
					Region:            storageService.Region,
					AccountID:         storageService.AccountID,
				}

				if err := taskStore.Create(ctx, task); err != nil {
					logging.Debug("TEST MODE: Failed to create task for service",
						"serviceName", storageService.ServiceName,
						"error", err)
				} else {
					logging.Debug("TEST MODE: Created task for service",
						"taskID", taskID,
						"serviceName", storageService.ServiceName)
				}
			}
		}

		return nil
	}

	// Ensure kubernetes client is initialized
	if err := sm.initializeClient(); err != nil {
		return fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	// Use the service manager's client
	kubeClient := sm.clientset

	// Get the client config to log the endpoint
	var endpoint string
	if kubeClient != nil {
		// Try to get the endpoint from the client - this is for debugging
		endpoint = "client initialized"
	}

	logging.Info("Kubernetes client initialized, proceeding with deployment creation",
		"service", storageService.ServiceName,
		"clientNil", kubeClient == nil,
		"endpoint", endpoint)

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
			logging.Warn("Failed to create kubernetes service",
				"error", err)
		}
	}

	// Start watching pods for this deployment to create ECS tasks
	if sm.taskManager != nil {
		go sm.watchServicePods(context.Background(), deployment, cluster, storageService)
	}

	// Update service status to ACTIVE after successful deployment using transaction
	if err := sm.updateServiceStatusSafely(ctx, cluster, storageService); err != nil {
		logging.Warn("Failed to update service status to ACTIVE",
			"error", err)
		// Don't fail the entire operation for status update issues
	} else {
		logging.Info("Updated service status to ACTIVE",
			"serviceName", storageService.ServiceName,
			"serviceID", storageService.ID)
	}

	logging.Info("Successfully created service",
		"serviceName", storageService.ServiceName,
		"deploymentName", deployment.Name,
		"namespace", deployment.Namespace)

	return nil
}

// UpdateService updates a Kubernetes Deployment for an ECS service
func (sm *ServiceManager) UpdateService(
	ctx context.Context,
	deployment *appsv1.Deployment,
	kubeService *corev1.Service,
	cluster *storage.Cluster,
	storageService *storage.Service,
) error {
	// Check if running in test mode
	if config.GetBool("features.testMode") {
		// In test mode, simulate service update
		logging.Debug("TEST MODE: Simulated service update",
			"serviceName", storageService.ServiceName,
			"namespace", deployment.Namespace)

		// Update service counts based on replica changes
		if deployment.Spec.Replicas != nil {
			newDesiredCount := int(*deployment.Spec.Replicas)
			if newDesiredCount != storageService.DesiredCount {
				oldDesiredCount := storageService.DesiredCount
				storageService.DesiredCount = newDesiredCount

				// Simulate task scaling
				if sm.storage != nil {
					taskStore := sm.storage.TaskStore()

					if newDesiredCount > oldDesiredCount {
						// Scale up - create new tasks
						for i := oldDesiredCount; i < newDesiredCount; i++ {
							taskID := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s-%d-%d",
								storageService.Region, storageService.AccountID, cluster.Name,
								storageService.ServiceName, time.Now().Unix(), i)

							task := &storage.Task{
								ARN:               taskID,
								ClusterARN:        cluster.ARN,
								TaskDefinitionARN: storageService.TaskDefinitionARN,
								DesiredStatus:     "RUNNING",
								LastStatus:        "RUNNING",
								LaunchType:        storageService.LaunchType,
								StartedBy:         fmt.Sprintf("ecs-svc/%s", storageService.ServiceName),
								Region:            storageService.Region,
								AccountID:         storageService.AccountID,
							}

							if err := taskStore.Create(ctx, task); err != nil {
								logging.Debug("TEST MODE: Failed to create task during scale-up",
									"error", err)
							}
						}
					} else if newDesiredCount < oldDesiredCount {
						// Scale down - stop excess tasks
						tasks, err := taskStore.List(ctx, cluster.ARN, storage.TaskFilters{
							ServiceName:   storageService.ServiceName,
							DesiredStatus: "RUNNING",
						})
						if err == nil {
							// Stop tasks to match new desired count
							tasksToStop := len(tasks) - newDesiredCount
							if tasksToStop > 0 {
								for i := 0; i < tasksToStop && i < len(tasks); i++ {
									task := tasks[i]
									task.DesiredStatus = "STOPPED"
									task.LastStatus = "STOPPED"
									task.StoppedReason = "Service scaled down"
									if err := taskStore.Update(ctx, task); err != nil {
										logging.Debug("TEST MODE: Failed to stop task",
											"taskARN", task.ARN,
											"error", err)
									} else {
										logging.Debug("TEST MODE: Stopped task for scale down",
											"taskARN", task.ARN)
									}
								}
							}
						}
					}
				}

				// Update running count to match desired count
				storageService.RunningCount = newDesiredCount
				storageService.PendingCount = 0
				logging.Debug("TEST MODE: Service scaled",
					"serviceName", storageService.ServiceName,
					"oldDesiredCount", oldDesiredCount,
					"newDesiredCount", newDesiredCount)
			}
		}

		// Ensure status remains ACTIVE in test mode
		storageService.Status = "ACTIVE"

		return nil
	}

	// Ensure kubernetes client is initialized
	if err := sm.initializeClient(); err != nil {
		return fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	// Use the service manager's client
	kubeClient := sm.clientset

	// Update Deployment
	_, err := kubeClient.AppsV1().Deployments(deployment.Namespace).Update(
		ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// Update Service if provided
	if kubeService != nil {
		existingService, err := kubeClient.CoreV1().Services(kubeService.Namespace).Get(
			ctx, kubeService.Name, metav1.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				logging.Warn("Failed to get existing kubernetes service",
					"error", err)
			}
		} else {
			// Preserve ClusterIP and update other fields
			kubeService.Spec.ClusterIP = existingService.Spec.ClusterIP
			kubeService.ResourceVersion = existingService.ResourceVersion
			_, err = kubeClient.CoreV1().Services(kubeService.Namespace).Update(
				ctx, kubeService, metav1.UpdateOptions{})
			if err != nil {
				logging.Warn("Failed to update kubernetes service",
					"error", err)
			}
		}
	}

	logging.Info("Successfully updated service",
		"serviceName", storageService.ServiceName,
		"deploymentName", deployment.Name,
		"namespace", deployment.Namespace)

	return nil
}

// DeleteService deletes a Kubernetes Deployment and Service for an ECS service
func (sm *ServiceManager) DeleteService(
	ctx context.Context,
	cluster *storage.Cluster,
	storageService *storage.Service,
) error {
	// Check if running in test mode
	if config.GetBool("features.testMode") {
		// In test mode, simulate service deletion
		logging.Debug("TEST MODE: Simulated service deletion",
			"serviceName", storageService.ServiceName)

		// Update service status to INACTIVE
		storageService.Status = "INACTIVE"
		storageService.RunningCount = 0
		storageService.PendingCount = 0

		// Simulate deleting tasks for the service
		if sm.storage != nil {
			taskStore := sm.storage.TaskStore()

			// Get all tasks for the service
			tasks, err := taskStore.List(ctx, cluster.ARN, storage.TaskFilters{
				ServiceName: storageService.ServiceName,
			})
			if err == nil {
				for _, task := range tasks {
					// Mark task as stopped
					task.DesiredStatus = "STOPPED"
					task.LastStatus = "STOPPED"
					task.StoppedReason = "Service deleted"
					if err := taskStore.Update(ctx, task); err != nil {
						logging.Debug("TEST MODE: Failed to stop task",
							"taskARN", task.ARN,
							"error", err)
					} else {
						logging.Debug("TEST MODE: Stopped task for service deletion",
							"taskARN", task.ARN)
					}
				}
			}
		}

		return nil
	}

	// Ensure kubernetes client is initialized
	if err := sm.initializeClient(); err != nil {
		return fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	// Use the service manager's client
	kubeClient := sm.clientset

	namespace := storageService.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	}

	deploymentName := storageService.DeploymentName
	if deploymentName == "" {
		deploymentName = storageService.ServiceName
	}

	// Delete Deployment
	err := kubeClient.AppsV1().Deployments(namespace).Delete(
		ctx, deploymentName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logging.Warn("Failed to delete deployment",
			"deploymentName", deploymentName,
			"error", err)
	}

	// Delete Service (if exists)
	serviceName := storageService.ServiceName
	err = kubeClient.CoreV1().Services(namespace).Delete(
		ctx, serviceName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		logging.Warn("Failed to delete kubernetes service",
			"serviceName", serviceName,
			"error", err)
	}

	logging.Info("Successfully deleted service deployment and service",
		"serviceName", storageService.ServiceName,
		"namespace", namespace)

	return nil
}

// GetServiceStatus gets the current status of a Kubernetes Deployment
func (sm *ServiceManager) GetServiceStatus(
	ctx context.Context,
	cluster *storage.Cluster,
	storageService *storage.Service,
) (*ServiceStatus, error) {
	// Check if running in test mode
	if config.GetBool("features.testMode") {
		// In test mode, return simulated status based on stored service data
		logging.Debug("TEST MODE: Returning simulated status",
			"serviceName", storageService.ServiceName)

		return &ServiceStatus{
			Status:       storageService.Status,
			DesiredCount: storageService.DesiredCount,
			RunningCount: storageService.RunningCount,
			PendingCount: storageService.PendingCount,
		}, nil
	}

	// Ensure kubernetes client is initialized
	if sm.clientset == nil {
		if err := sm.initializeClient(); err != nil {
			return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
		}
	}
	kubeClient := sm.clientset

	namespace := storageService.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	}

	deploymentName := storageService.DeploymentName
	if deploymentName == "" {
		deploymentName = storageService.ServiceName
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
	logging.Info("ensureNamespace called",
		"namespace", namespace,
		"clientNil", kubeClient == nil,
		"smClientNil", sm.clientset == nil,
		"sameClient", kubeClient == sm.clientset)

	// Double-check we're using the right client
	if kubeClient != sm.clientset {
		logging.Warn("ensureNamespace received different client than sm.clientset!")
	}

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
			logging.Info("Created namespace",
				"namespace", namespace)
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
			logging.Info("Updated existing deployment",
				"deploymentName", deployment.Name)
		} else {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	} else {
		logging.Info("Created deployment",
			"deploymentName", deployment.Name)
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
			logging.Info("Updated existing kubernetes service",
				"serviceName", service.Name)
		} else {
			return fmt.Errorf("failed to create kubernetes service: %w", err)
		}
	} else {
		logging.Info("Created kubernetes service",
			"serviceName", service.Name)
	}
	return nil
}

// updateServiceStatusSafely updates service status with safe error handling
func (sm *ServiceManager) updateServiceStatusSafely(ctx context.Context, cluster *storage.Cluster, storageService *storage.Service) error {
	logging.Debug("Starting status update for service",
		"serviceName", storageService.ServiceName,
		"serviceID", storageService.ID)

	// Get fresh service data from storage to avoid conflicts
	freshService, err := sm.storage.ServiceStore().Get(ctx, cluster.ARN, storageService.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get fresh service for status update: %w", err)
	}

	logging.Debug("Retrieved fresh service",
		"serviceID", freshService.ID,
		"currentStatus", freshService.Status)

	// Update only the status field
	freshService.Status = "ACTIVE"

	logging.Debug("About to update service status",
		"serviceID", freshService.ID,
		"newStatus", "ACTIVE")

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

// watchServicePods watches pods created by a deployment and registers them as ECS tasks
func (sm *ServiceManager) watchServicePods(ctx context.Context, deployment *appsv1.Deployment, cluster *storage.Cluster, service *storage.Service) {
	if sm.taskManager == nil {
		logging.Warn("TaskManager not available, cannot watch service pods")
		return
	}

	// Initialize TaskManager client if needed
	if err := sm.taskManager.InitializeClient(); err != nil {
		logging.Error("Failed to initialize TaskManager client", "error", err)
		return
	}

	// Use labels to watch pods for this deployment
	labelSelector := fmt.Sprintf("app=%s", deployment.Name)

	// Get clientset
	var clientset kubernetes.Interface
	if sm.clientset != nil {
		clientset = sm.clientset
	} else if sm.taskManager.Clientset != nil {
		clientset = sm.taskManager.Clientset
	} else {
		logging.Error("No kubernetes client available for watching pods")
		return
	}

	// List existing pods first to register them
	pods, err := clientset.CoreV1().Pods(deployment.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logging.Error("Failed to list pods for service", "service", service.ServiceName, "error", err)
		return
	}

	// Register existing pods as tasks
	for _, pod := range pods.Items {
		sm.registerPodAsTask(ctx, &pod, cluster, service)
	}

	// Watch for new pods
	watcher, err := clientset.CoreV1().Pods(deployment.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logging.Error("Failed to watch pods for service", "service", service.ServiceName, "error", err)
		return
	}
	defer watcher.Stop()

	logging.Info("Started watching pods for service", "service", service.ServiceName, "namespace", deployment.Namespace)

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		switch event.Type {
		case "ADDED", "MODIFIED":
			sm.registerPodAsTask(ctx, pod, cluster, service)
		case "DELETED":
			// Handle pod deletion - mark corresponding task as stopped
			sm.handlePodDeletion(ctx, pod, cluster, service)
		}
	}
}

// registerPodAsTask registers a Kubernetes pod as an ECS task
func (sm *ServiceManager) registerPodAsTask(ctx context.Context, pod *corev1.Pod, cluster *storage.Cluster, service *storage.Service) {
	// Skip if pod is terminating
	if pod.DeletionTimestamp != nil {
		return
	}

	// Use task ID from webhook label if available, otherwise generate from pod name
	var taskID string
	if tid, exists := pod.Labels["kecs.dev/task-id"]; exists && tid != "" {
		taskID = tid
		logging.Debug("Using task ID from webhook label", "taskId", taskID, "pod", pod.Name)
	} else {
		// Fallback to generating from pod name for backward compatibility
		taskID = utils.GenerateTaskIDFromString(pod.Name)
		logging.Debug("Generated task ID from pod name", "taskId", taskID, "pod", pod.Name)
	}

	// Check if task already exists for this pod
	taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s",
		service.Region, service.AccountID, cluster.Name, taskID)

	existingTask, err := sm.storage.TaskStore().Get(ctx, cluster.ARN, taskARN)
	if err == nil && existingTask != nil {
		// Task already exists, update status if needed
		if sm.taskManager != nil {
			if err := sm.taskManager.UpdateTaskStatus(ctx, taskARN, pod); err != nil {
				logging.Warn("Failed to update task status", "task", taskARN, "error", err)
			}
		}
		return
	}

	// Get Service Registry metadata from pod annotations (if available)
	serviceRegistries := ""
	if pod.Annotations != nil {
		if sr, exists := pod.Annotations["kecs.dev/service-registries"]; exists {
			serviceRegistries = sr
			logging.Debug("Found Service Registry metadata in pod annotations",
				"pod", pod.Name,
				"serviceRegistries", serviceRegistries)
		}
	}
	// Fallback to service's ServiceRegistries if not in pod annotations
	if serviceRegistries == "" && service.ServiceRegistries != "" {
		serviceRegistries = service.ServiceRegistries
		logging.Debug("Using Service Registry metadata from service",
			"service", service.ServiceName,
			"serviceRegistries", serviceRegistries)
	}

	// Create new task
	now := time.Now()
	task := &storage.Task{
		ID:                taskID,
		ARN:               taskARN,
		ClusterARN:        cluster.ARN,
		TaskDefinitionARN: service.TaskDefinitionARN,
		LastStatus:        mapPodPhaseToTaskStatus(pod.Status.Phase),
		DesiredStatus:     "RUNNING",
		LaunchType:        service.LaunchType,
		StartedBy:         fmt.Sprintf("ecs-svc/%s", service.ServiceName),
		Group:             fmt.Sprintf("service:%s", service.ServiceName),
		PodName:           pod.Name,
		Namespace:         pod.Namespace,
		CreatedAt:         now,
		Region:            service.Region,
		AccountID:         service.AccountID,
		Version:           1,
		Connectivity:      "CONNECTED",
		ConnectivityAt:    &now,
		CPU:               "",                // Will be set from task definition
		Memory:            "",                // Will be set from task definition
		ServiceRegistries: serviceRegistries, // Use Service Registry metadata from pod or service
	}

	// Create containers info from pod
	containers := sm.taskManager.GetContainerStatuses(pod)
	if len(containers) > 0 {
		containersJSON, err := json.Marshal(containers)
		if err == nil {
			task.Containers = string(containersJSON)
		}
	}

	// Store the task
	if err := sm.storage.TaskStore().Create(ctx, task); err != nil {
		// Check if it's a duplicate key error
		if strings.Contains(err.Error(), "Duplicate key") {
			logging.Info("Task already exists, updating status", "pod", pod.Name, "task", taskARN)
			// Task already exists, update its status instead
			if sm.taskManager != nil {
				if err := sm.taskManager.UpdateTaskStatus(ctx, taskARN, pod); err != nil {
					logging.Error("Failed to update existing task status", "task", taskARN, "error", err)
				}
				// Still start watching for future updates
				go sm.taskManager.watchPodStatus(context.Background(), taskARN, pod.Namespace, pod.Name)
			}
			return
		}
		logging.Error("Failed to create task for pod", "pod", pod.Name, "error", err)
		return
	}

	logging.Info("Registered pod as ECS task", "pod", pod.Name, "task", taskARN, "service", service.ServiceName)

	// Start watching the pod for status updates
	if sm.taskManager != nil {
		go sm.taskManager.watchPodStatus(context.Background(), taskARN, pod.Namespace, pod.Name)
	}
}

// handlePodDeletion handles when a pod is deleted
func (sm *ServiceManager) handlePodDeletion(ctx context.Context, pod *corev1.Pod, cluster *storage.Cluster, service *storage.Service) {
	// Generate deterministic task ID from pod name
	taskID := utils.GenerateTaskIDFromString(pod.Name)
	taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s",
		service.Region, service.AccountID, cluster.Name, taskID)

	task, err := sm.storage.TaskStore().Get(ctx, cluster.ARN, taskARN)
	if err != nil || task == nil {
		return
	}

	// Update task status to STOPPED
	now := time.Now()
	task.DesiredStatus = "STOPPED"
	task.LastStatus = "STOPPED"
	task.StoppedAt = &now
	task.StoppedReason = "Service pod terminated"
	task.Version++

	if err := sm.storage.TaskStore().Update(ctx, task); err != nil {
		logging.Error("Failed to update task status to STOPPED", "task", taskARN, "error", err)
	}

	logging.Info("Marked task as STOPPED due to pod deletion", "pod", pod.Name, "task", taskARN)
}
