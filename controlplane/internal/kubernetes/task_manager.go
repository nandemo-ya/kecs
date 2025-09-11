package kubernetes

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// TaskManager manages ECS tasks as Kubernetes pods
type TaskManager struct {
	Clientset               kubernetes.Interface
	storage                 storage.Storage
	serviceDiscoveryManager servicediscovery.Manager
	logCollector            *LogCollector
}

// NewTaskManager creates a new task manager
func NewTaskManager(storage storage.Storage) (*TaskManager, error) {
	return NewTaskManagerWithServiceDiscovery(storage, nil)
}

// NewTaskManagerWithServiceDiscovery creates a new task manager with optional service discovery
func NewTaskManagerWithServiceDiscovery(storage storage.Storage, sdManager servicediscovery.Manager) (*TaskManager, error) {
	if sdManager != nil {
		logging.Info("Creating TaskManager with Service Discovery integration")
	} else {
		logging.Debug("Creating TaskManager without Service Discovery")
	}

	// In container mode or test mode, defer kubernetes client creation
	if config.GetBool("features.containerMode") || os.Getenv("KECS_TEST_MODE") == "true" {
		logging.Debug("Container/test mode enabled - deferring kubernetes client initialization")
		return &TaskManager{
			Clientset:               nil, // Will be initialized later
			storage:                 storage,
			serviceDiscoveryManager: sdManager,
			logCollector:            nil, // Will be initialized when clientset is available
		}, nil
	}

	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		config, err = GetKubeConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	tm := &TaskManager{
		Clientset:               clientset,
		storage:                 storage,
		serviceDiscoveryManager: sdManager,
	}

	// Initialize log collector if storage supports it
	if storage != nil && storage.TaskLogStore() != nil {
		tm.logCollector = NewLogCollector(clientset, storage)
		logging.Info("Task log collection enabled")
	} else {
		logging.Debug("Task log collection disabled - no TaskLogStore available")
	}

	return tm, nil
}

// InitializeClient initializes the kubernetes client if not already initialized
func (tm *TaskManager) InitializeClient() error {
	if tm.Clientset != nil {
		return nil // Already initialized
	}

	// Skip initialization in test mode
	if os.Getenv("KECS_TEST_MODE") == "true" {
		logging.Debug("Test mode enabled - skipping kubernetes client initialization")
		return nil
	}

	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		config, err = GetKubeConfig()
		if err != nil {
			return fmt.Errorf("failed to get kubernetes config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	tm.Clientset = clientset

	// Initialize log collector if not already initialized
	if tm.logCollector == nil && tm.storage != nil && tm.storage.TaskLogStore() != nil {
		tm.logCollector = NewLogCollector(clientset, tm.storage)
		logging.Info("Task log collection enabled (late initialization)")
	}

	logging.Debug("TaskManager kubernetes client initialized")
	return nil
}

// CreateTask creates a new task by deploying a pod
func (tm *TaskManager) CreateTask(ctx context.Context, pod *corev1.Pod, task *storage.Task, secrets map[string]*converters.SecretInfo) error {
	// In test mode, skip actual pod creation
	if tm.Clientset == nil {
		logging.Debug("Kubernetes client not initialized - simulating task creation")
		task.PodName = "test-pod-" + task.ID
		task.Namespace = pod.Namespace
		// Don't override LastStatus - keep what was set by the caller
		task.Connectivity = "CONNECTED"
		task.ConnectivityAt = &task.CreatedAt

		// Add pod name and namespace to task attributes for easy lookup
		if err := tm.addPodInfoToTaskAttributes(task, task.PodName, task.Namespace); err != nil {
			logging.Warn("Failed to add pod info to task attributes", "task", task.ARN, "error", err)
		}

		// Simulate container and network information for test mode
		if p := pod; p != nil {
			containers := tm.createTestContainers(p, task)
			if len(containers) > 0 {
				containersJSON, err := json.Marshal(containers)
				if err == nil {
					task.Containers = string(containersJSON)
				}
			}

			// Check for awsvpc network mode and create attachments
			if networkMode, ok := p.Annotations["ecs.amazonaws.com/network-mode"]; ok && networkMode == "awsvpc" {
				attachments := tm.createTestNetworkAttachments(p)
				if len(attachments) > 0 {
					attachmentsJSON, err := json.Marshal(attachments)
					if err == nil {
						task.Attachments = string(attachmentsJSON)
					}
				}
			}
		}

		// Store task in database
		if err := tm.storage.TaskStore().Create(ctx, task); err != nil {
			return fmt.Errorf("failed to store task: %w", err)
		}

		return nil
	}

	// Create secrets first if any
	if len(secrets) > 0 {
		if err := tm.createSecrets(ctx, pod.Namespace, secrets); err != nil {
			return fmt.Errorf("failed to create secrets: %w", err)
		}
	}

	// Create the pod
	createdPod, err := tm.Clientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Update task with pod information
	task.PodName = createdPod.Name
	task.Namespace = createdPod.Namespace
	task.LastStatus = "PENDING"
	task.Connectivity = "CONNECTED"
	task.ConnectivityAt = &task.CreatedAt

	// Add pod name and namespace to task attributes for easy lookup
	if err := tm.addPodInfoToTaskAttributes(task, createdPod.Name, createdPod.Namespace); err != nil {
		logging.Warn("Failed to add pod info to task attributes", "task", task.ARN, "error", err)
	}

	// Store task in database
	if err := tm.storage.TaskStore().Create(ctx, task); err != nil {
		// Try to clean up the pod if task storage fails
		_ = tm.Clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, createdPod.Name, metav1.DeleteOptions{})
		return fmt.Errorf("failed to store task: %w", err)
	}

	// Start watching the pod for status updates
	go tm.watchPodStatus(context.Background(), task.ARN, createdPod.Namespace, createdPod.Name)

	return nil
}

// StopTask stops a running task
func (tm *TaskManager) StopTask(ctx context.Context, cluster, taskID, reason string) error {
	// Get task from storage
	task, err := tm.storage.TaskStore().Get(ctx, cluster, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	// Update task status
	now := time.Now()
	task.DesiredStatus = "STOPPED"
	task.StoppedReason = reason
	task.StoppingAt = &now
	task.Version++

	if err := tm.storage.TaskStore().Update(ctx, task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Collect logs before deleting the pod
	if tm.logCollector != nil && tm.Clientset != nil && task.PodName != "" && task.Namespace != "" {
		// Collect logs asynchronously to avoid blocking the stop operation
		tm.logCollector.CollectLogsBeforeDeletion(ctx, task.ARN, task.Namespace, task.PodName)
	}

	// Delete the pod (skip if no kubernetes client)
	if tm.Clientset != nil && task.PodName != "" && task.Namespace != "" {
		err := tm.Clientset.CoreV1().Pods(task.Namespace).Delete(ctx, task.PodName, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
	}

	return nil
}

// UpdateTaskStatus updates the status of a task based on pod status
func (tm *TaskManager) UpdateTaskStatus(ctx context.Context, taskARN string, pod *corev1.Pod) error {
	task, err := tm.storage.TaskStore().Get(ctx, "", taskARN)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	// Map pod phase to ECS task status
	previousStatus := task.LastStatus
	task.LastStatus = mapPodPhaseToTaskStatus(pod.Status.Phase)
	task.Version++

	// Update container statuses
	containers := tm.GetContainerStatuses(pod)
	containersJSON, err := json.Marshal(containers)
	if err != nil {
		return fmt.Errorf("failed to marshal container statuses: %w", err)
	}
	task.Containers = string(containersJSON)

	// Update timestamps based on pod status
	now := time.Now()

	// Set pull timestamps
	if previousStatus == "PENDING" && task.LastStatus == "PROVISIONING" {
		task.PullStartedAt = &now
	}
	if previousStatus == "PROVISIONING" && (task.LastStatus == "RUNNING" || task.LastStatus == "STOPPED") {
		task.PullStoppedAt = &now
	}

	// Set started timestamp
	if task.StartedAt == nil && pod.Status.StartTime != nil {
		startTime := pod.Status.StartTime.Time
		task.StartedAt = &startTime
	}

	// Handle stopped tasks
	if task.LastStatus == "STOPPED" {
		task.StoppedAt = &now
		task.ExecutionStoppedAt = &now

		// Determine stop reason
		if pod.Status.Reason != "" {
			task.StopCode = pod.Status.Reason
		}
		if pod.Status.Message != "" {
			task.StoppedReason = pod.Status.Message
		}

		// Check container statuses for more details
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Terminated != nil {
				if cs.State.Terminated.ExitCode != 0 {
					task.StopCode = "TaskFailed"
					task.StoppedReason = cs.State.Terminated.Reason
					if cs.State.Terminated.Message != "" {
						task.StoppedReason = cs.State.Terminated.Message
					}
					break
				}
			}
		}
	}

	// Update health status
	previousHealthStatus := task.HealthStatus
	task.HealthStatus = tm.getHealthStatus(pod)

	// Register/deregister with Service Discovery
	if previousStatus != task.LastStatus {
		if task.LastStatus == "RUNNING" && pod.Status.PodIP != "" {
			// Task is now running with an IP - register with Service Discovery
			logging.Debug("Task status changed to RUNNING, registering with Service Discovery",
				"task", task.ARN,
				"pod", pod.Name,
				"ip", pod.Status.PodIP,
				"hasServiceDiscovery", tm.serviceDiscoveryManager != nil)
			go tm.registerWithServiceDiscovery(context.Background(), task, pod)
		} else if task.LastStatus == "STOPPED" || task.LastStatus == "DEACTIVATING" {
			// Task is stopping - deregister from Service Discovery
			logging.Debug("Task status changed to STOPPED/DEACTIVATING, deregistering from Service Discovery",
				"task", task.ARN,
				"pod", pod.Name)
			go tm.deregisterFromServiceDiscovery(context.Background(), task)
		}
	}

	// Update health status in Service Discovery if changed
	if task.LastStatus == "RUNNING" && previousHealthStatus != task.HealthStatus {
		go tm.updateServiceDiscoveryHealth(context.Background(), task)
	}

	return tm.storage.TaskStore().Update(ctx, task)
}

// watchPodStatus watches a pod for status changes
func (tm *TaskManager) watchPodStatus(ctx context.Context, taskARN, namespace, podName string) {
	// First, get the current pod status and update immediately
	pod, err := tm.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err == nil && pod != nil {
		// Update task status with current pod state
		logging.Debug("Updating task status for existing pod", "pod", podName, "phase", pod.Status.Phase)
		if err := tm.UpdateTaskStatus(ctx, taskARN, pod); err != nil {
			logging.Error("Failed to update task status for existing pod", "task", taskARN, "namespace", namespace, "pod", podName, "error", err)
		}
	}

	// Use a field selector to watch only this specific pod
	fieldSelector := fields.OneTermEqualSelector("metadata.name", podName).String()

	watcher, err := tm.Clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		// Log error with more context and don't crash
		logging.Warn("Failed to watch pod for task", "pod", podName, "namespace", namespace, "task", taskARN, "error", err)
		return
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			continue
		}

		// Update task status
		if err := tm.UpdateTaskStatus(ctx, taskARN, pod); err != nil {
			logging.Error("Failed to update task status", "task", taskARN, "namespace", pod.Namespace, "pod", pod.Name, "error", err)
			// Continue processing other events despite this error
		}

		// Stop watching if pod is terminated
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			// Collect logs before pod is garbage collected
			if tm.logCollector != nil {
				logging.Info("Pod terminated, collecting logs", "taskARN", taskARN, "pod", pod.Name, "phase", pod.Status.Phase)
				tm.logCollector.CollectLogsBeforeDeletion(ctx, taskARN, namespace, podName)
			}
			return
		}
	}
}

// GetContainerStatuses extracts container status information
func (tm *TaskManager) GetContainerStatuses(pod *corev1.Pod) []types.Container {
	containers := make([]types.Container, 0, len(pod.Status.ContainerStatuses))

	for _, cs := range pod.Status.ContainerStatuses {
		container := types.Container{
			Name:         cs.Name,
			ContainerArn: fmt.Sprintf("arn:aws:ecs:container/%s", cs.ContainerID),
			TaskArn:      pod.Annotations["kecs.dev/task-arn"],
			Image:        cs.Image,
			ImageDigest:  cs.ImageID,
		}

		// Set last status based on container state
		if cs.State.Running != nil {
			container.LastStatus = "RUNNING"
		} else if cs.State.Terminated != nil {
			container.LastStatus = "STOPPED"
			container.ExitCode = int(cs.State.Terminated.ExitCode)
			container.Reason = cs.State.Terminated.Reason
		} else if cs.State.Waiting != nil {
			container.LastStatus = "PENDING"
			container.Reason = cs.State.Waiting.Reason
		}

		// Health status
		if cs.Ready {
			container.HealthStatus = "HEALTHY"
		} else {
			container.HealthStatus = "UNHEALTHY"
		}

		containers = append(containers, container)
	}

	return containers
}

// getHealthStatus determines the overall health status of the task
func (tm *TaskManager) getHealthStatus(pod *corev1.Pod) string {
	allHealthy := true
	hasUnhealthy := false

	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			allHealthy = false
			if cs.RestartCount > 0 || (cs.State.Terminated != nil && cs.State.Terminated.ExitCode != 0) {
				hasUnhealthy = true
				break
			}
		}
	}

	if allHealthy {
		return "HEALTHY"
	} else if hasUnhealthy {
		return "UNHEALTHY"
	}
	return "UNKNOWN"
}

// mapPodPhaseToTaskStatus maps Kubernetes pod phase to ECS task status
func mapPodPhaseToTaskStatus(phase corev1.PodPhase) string {
	switch phase {
	case corev1.PodPending:
		return "PENDING"
	case corev1.PodRunning:
		return "RUNNING"
	case corev1.PodSucceeded:
		return "STOPPED"
	case corev1.PodFailed:
		return "STOPPED"
	case corev1.PodUnknown:
		return "PENDING"
	default:
		return "PENDING"
	}
}

// createSecrets creates Kubernetes secrets for the task
func (tm *TaskManager) createSecrets(ctx context.Context, namespace string, secrets map[string]*converters.SecretInfo) error {
	for arn, info := range secrets {
		// Check if secret already exists
		existingSecret, err := tm.Clientset.CoreV1().Secrets(namespace).Get(ctx, info.SecretName, metav1.GetOptions{})
		if err == nil && existingSecret != nil {
			// Secret already exists, skip creation
			continue
		}

		// Create the secret with placeholder data
		// In a real implementation, this would fetch the actual secret value from AWS
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      info.SecretName,
				Namespace: namespace,
				Labels: map[string]string{
					"kecs.dev/managed-by": "kecs",
					"kecs.dev/source":     info.Source,
				},
				Annotations: map[string]string{
					"kecs.dev/arn": arn,
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				info.Key: []byte("placeholder-secret-value"), // TODO: Fetch actual secret from AWS
			},
		}

		_, err = tm.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create secret %s: %w", info.SecretName, err)
		}
	}

	return nil
}

// CreateServiceDeployment creates a Kubernetes deployment for an ECS service
func (tm *TaskManager) CreateServiceDeployment(ctx context.Context, cluster *storage.Cluster, service *storage.Service, taskDef *storage.TaskDefinition) error {
	// Ensure namespace exists
	namespace := fmt.Sprintf("kecs-%s", cluster.Name)
	if err := EnsureNamespace(ctx, tm.Clientset, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Parse container definitions from storage
	var containerDefs []types.ContainerDefinition
	if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err != nil {
		return fmt.Errorf("failed to parse container definitions: %w", err)
	}

	// Create deployment info
	deploymentInfo := converters.ConvertServiceToDeployment(service, taskDef, namespace)

	// Convert to Kubernetes deployment
	k8sDeployment := converters.ConvertDeploymentToK8s(deploymentInfo, containerDefs)

	// Create deployment in Kubernetes
	_, err := tm.Clientset.AppsV1().Deployments(namespace).Create(ctx, k8sDeployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing deployment
			_, err = tm.Clientset.AppsV1().Deployments(namespace).Update(ctx, k8sDeployment, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update existing deployment: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create deployment: %w", err)
		}
	}

	logging.Info("Created/updated deployment for service",
		"service", service.ServiceName, "namespace", namespace)

	return nil
}

// generateShortID generates a short random ID for resource naming
func generateShortID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ptr returns a pointer to the given string
func ptr(s string) *string {
	return &s
}

// createTestContainers creates container information for test mode
func (tm *TaskManager) createTestContainers(pod *corev1.Pod, task *storage.Task) []types.Container {
	var containers []types.Container

	// Use a test IP address
	podIP := "10.0.0.1"

	for _, container := range pod.Spec.Containers {
		testContainer := types.Container{
			Name:         container.Name,
			ContainerArn: fmt.Sprintf("arn:aws:ecs:container/%s", task.ID),
			TaskArn:      task.ARN,
			Image:        container.Image,
			LastStatus:   "PENDING",
		}

		// Add network interfaces for awsvpc mode
		if networkMode := pod.Annotations["ecs.amazonaws.com/network-mode"]; networkMode == "awsvpc" {
			testContainer.NetworkInterfaces = []types.NetworkInterface{
				{
					AttachmentId:       fmt.Sprintf("eni-attach-%s", task.ID),
					PrivateIpv4Address: podIP,
				},
			}
		}

		// Add network bindings from container ports
		for _, port := range container.Ports {
			testContainer.NetworkBindings = append(testContainer.NetworkBindings, types.NetworkBinding{
				ContainerPort: int(port.ContainerPort),
				Protocol:      string(port.Protocol),
				BindIP:        podIP,
			})
		}

		containers = append(containers, testContainer)
	}

	return containers
}

// createTestNetworkAttachments creates network attachments for test mode
func (tm *TaskManager) createTestNetworkAttachments(pod *corev1.Pod) []types.Attachment {
	var attachments []types.Attachment

	// Get network configuration from annotations
	subnets := pod.Annotations["ecs.amazonaws.com/subnets"]
	privateIP := "10.0.0.1"

	// Create elastic network interface attachment
	var details []types.KeyValuePair
	if subnets != "" {
		subnetID := strings.Split(subnets, ",")[0]
		details = append(details, types.KeyValuePair{
			Name:  ptr("subnetId"),
			Value: ptr(subnetID),
		})
	}
	details = append(details,
		types.KeyValuePair{
			Name:  ptr("networkInterfaceId"),
			Value: ptr(fmt.Sprintf("eni-%s", pod.UID)),
		},
		types.KeyValuePair{
			Name:  ptr("macAddress"),
			Value: ptr("02:00:00:00:00:01"),
		},
		types.KeyValuePair{
			Name:  ptr("privateDnsName"),
			Value: ptr(fmt.Sprintf("ip-%s.ec2.internal", strings.ReplaceAll(privateIP, ".", "-"))),
		},
		types.KeyValuePair{
			Name:  ptr("privateIPv4Address"),
			Value: ptr(privateIP),
		},
	)

	attachment := types.Attachment{
		Id:      ptr(fmt.Sprintf("eni-attach-%s", pod.UID)),
		Type:    ptr("ElasticNetworkInterface"),
		Status:  ptr("ATTACHED"),
		Details: details,
	}

	attachments = append(attachments, attachment)
	return attachments
}

// registerWithServiceDiscovery registers a task with Service Discovery
func (tm *TaskManager) registerWithServiceDiscovery(ctx context.Context, task *storage.Task, pod *corev1.Pod) {
	// Check if Service Discovery manager is available
	if tm.serviceDiscoveryManager == nil {
		logging.Debug("Service Discovery manager not available, skipping registration")
		return
	}

	// Check if the task was started by a service
	if !strings.HasPrefix(task.StartedBy, "ecs-svc/") {
		return
	}

	// Extract service name from StartedBy field
	serviceName := strings.TrimPrefix(task.StartedBy, "ecs-svc/")

	// Extract cluster name from ClusterARN
	clusterName := "default"
	if task.ClusterARN != "" {
		parts := strings.Split(task.ClusterARN, "/")
		if len(parts) > 0 {
			clusterName = parts[len(parts)-1]
		}
	}

	// Get the service from storage to check for ServiceRegistry metadata
	service, err := tm.storage.ServiceStore().Get(ctx, clusterName, serviceName)
	if err != nil || service == nil {
		logging.Debug("Service not found in storage",
			"serviceName", serviceName,
			"clusterName", clusterName,
			"error", err)
		return
	}

	// Try to get ServiceRegistries from task first (set from pod annotations)
	var serviceRegistries []map[string]interface{}
	if task.ServiceRegistries != "" {
		if err := json.Unmarshal([]byte(task.ServiceRegistries), &serviceRegistries); err == nil {
			logging.Debug("Using ServiceRegistries from task",
				"task", task.ARN,
				"serviceRegistries", task.ServiceRegistries)
		}
	}

	// Fallback to service.ServiceRegistries if not in task
	if len(serviceRegistries) == 0 && service.ServiceRegistries != "" {
		if err := json.Unmarshal([]byte(service.ServiceRegistries), &serviceRegistries); err == nil {
			logging.Debug("Using ServiceRegistries from service",
				"serviceName", serviceName,
				"serviceRegistries", service.ServiceRegistries)
		}
	}

	// If still no service registries, try legacy ServiceRegistryMetadata
	if len(serviceRegistries) == 0 && service.ServiceRegistryMetadata != nil {
		logging.Debug("Falling back to ServiceRegistryMetadata",
			"serviceName", serviceName)
		// Register each service discovery service using legacy metadata
		for serviceID, metadataJSON := range service.ServiceRegistryMetadata {
			// Parse metadata
			var metadata struct {
				ContainerName string `json:"containerName"`
				ContainerPort int32  `json:"containerPort"`
			}
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
				logging.Warn("Failed to parse service registry metadata", "serviceID", serviceID, "error", err)
				continue
			}

			tm.registerInstance(ctx, task, pod, serviceID, metadata.ContainerName, metadata.ContainerPort, service.ServiceName, clusterName)
		}
		return
	}

	// Register using ServiceRegistries
	for _, registry := range serviceRegistries {
		registryArn, _ := registry["registryArn"].(string)
		containerName, _ := registry["containerName"].(string)
		containerPort := int32(0)
		if cp, ok := registry["containerPort"].(float64); ok {
			containerPort = int32(cp)
		}

		// Extract service ID from registry ARN
		// Format: arn:aws:servicediscovery:region:account:service/srv-xxx
		serviceID := ""
		if registryArn != "" {
			parts := strings.Split(registryArn, "/")
			if len(parts) > 0 {
				serviceID = parts[len(parts)-1]
			}
		}

		if serviceID == "" {
			logging.Warn("Invalid registry ARN", "registryArn", registryArn)
			continue
		}

		tm.registerInstance(ctx, task, pod, serviceID, containerName, containerPort, service.ServiceName, clusterName)
	}
}

// registerInstance registers a single instance with Service Discovery
func (tm *TaskManager) registerInstance(ctx context.Context, task *storage.Task, pod *corev1.Pod,
	serviceID, containerName string, containerPort int32, serviceName, clusterName string) {

	// Map ECS health status to Service Discovery health status
	var sdHealthStatus string
	switch task.HealthStatus {
	case "HEALTHY":
		sdHealthStatus = "HEALTHY"
	case "UNHEALTHY":
		sdHealthStatus = "UNHEALTHY"
	case "UNKNOWN":
		sdHealthStatus = "UNKNOWN"
	default:
		sdHealthStatus = "UNKNOWN"
	}

	// Create instance
	instance := &servicediscovery.Instance{
		ID:           pod.Name,
		ServiceID:    serviceID,
		HealthStatus: sdHealthStatus,
		Attributes: map[string]string{
			"AWS_INSTANCE_IPV4": pod.Status.PodIP,
			"AWS_INSTANCE_ID":   pod.Name,
			"ECS_SERVICE_NAME":  serviceName,
			"ECS_TASK_ARN":      task.ARN,
			"ECS_CLUSTER":       clusterName,
			"K8S_NAMESPACE":     pod.Namespace,
		},
	}

	// Add container name if specified
	if containerName != "" {
		instance.Attributes["CONTAINER_NAME"] = containerName
	}

	// If container port is specified, add it to attributes
	if containerPort > 0 {
		instance.Attributes["AWS_INSTANCE_PORT"] = fmt.Sprintf("%d", containerPort)
		instance.Attributes["PORT"] = fmt.Sprintf("%d", containerPort)
	}

	// Register the instance
	if err := tm.serviceDiscoveryManager.RegisterInstance(ctx, instance); err != nil {
		logging.Warn("Failed to register task with service discovery",
			"task", task.ARN,
			"serviceID", serviceID,
			"instanceID", pod.Name,
			"error", err)
	} else {
		logging.Info("Task registered with service discovery",
			"task", task.ARN,
			"serviceID", serviceID,
			"instanceID", pod.Name,
			"ip", pod.Status.PodIP)
	}
}

// deregisterFromServiceDiscovery deregisters a task from Service Discovery
func (tm *TaskManager) deregisterFromServiceDiscovery(ctx context.Context, task *storage.Task) {
	// Check if Service Discovery manager is available
	if tm.serviceDiscoveryManager == nil {
		return
	}

	// Check if the task was started by a service
	if !strings.HasPrefix(task.StartedBy, "ecs-svc/") {
		return
	}

	// Extract service name from StartedBy field
	serviceName := strings.TrimPrefix(task.StartedBy, "ecs-svc/")

	// Extract cluster name from ClusterARN
	clusterName := "default"
	if task.ClusterARN != "" {
		parts := strings.Split(task.ClusterARN, "/")
		if len(parts) > 0 {
			clusterName = parts[len(parts)-1]
		}
	}

	// Get the service from storage to check for ServiceRegistry metadata
	service, err := tm.storage.ServiceStore().Get(ctx, clusterName, serviceName)
	if err != nil || service == nil {
		return
	}

	// Try to get ServiceRegistries from task first
	var serviceRegistries []map[string]interface{}
	if task.ServiceRegistries != "" {
		if err := json.Unmarshal([]byte(task.ServiceRegistries), &serviceRegistries); err == nil {
			// Deregister using ServiceRegistries
			for _, registry := range serviceRegistries {
				registryArn, _ := registry["registryArn"].(string)

				// Extract service ID from registry ARN
				serviceID := ""
				if registryArn != "" {
					parts := strings.Split(registryArn, "/")
					if len(parts) > 0 {
						serviceID = parts[len(parts)-1]
					}
				}

				if serviceID != "" {
					if err := tm.serviceDiscoveryManager.DeregisterInstance(ctx, serviceID, task.PodName); err != nil {
						logging.Warn("Failed to deregister task from service discovery",
							"task", task.ARN,
							"serviceID", serviceID,
							"instanceID", task.PodName,
							"error", err)
					} else {
						logging.Info("Task deregistered from service discovery",
							"task", task.ARN,
							"serviceID", serviceID,
							"instanceID", task.PodName)
					}
				}
			}
			return
		}
	}

	// Fallback to service.ServiceRegistries
	if service.ServiceRegistries != "" {
		if err := json.Unmarshal([]byte(service.ServiceRegistries), &serviceRegistries); err == nil {
			for _, registry := range serviceRegistries {
				registryArn, _ := registry["registryArn"].(string)

				// Extract service ID from registry ARN
				serviceID := ""
				if registryArn != "" {
					parts := strings.Split(registryArn, "/")
					if len(parts) > 0 {
						serviceID = parts[len(parts)-1]
					}
				}

				if serviceID != "" {
					if err := tm.serviceDiscoveryManager.DeregisterInstance(ctx, serviceID, task.PodName); err != nil {
						logging.Warn("Failed to deregister task from service discovery",
							"task", task.ARN,
							"serviceID", serviceID,
							"instanceID", task.PodName,
							"error", err)
					} else {
						logging.Info("Task deregistered from service discovery",
							"task", task.ARN,
							"serviceID", serviceID,
							"instanceID", task.PodName)
					}
				}
			}
			return
		}
	}

	// Fallback to legacy ServiceRegistryMetadata
	if service.ServiceRegistryMetadata != nil {
		// Deregister from each service discovery service
		for serviceID := range service.ServiceRegistryMetadata {
			// Deregister the instance (use pod name as instance ID)
			if err := tm.serviceDiscoveryManager.DeregisterInstance(ctx, serviceID, task.PodName); err != nil {
				logging.Warn("Failed to deregister task from service discovery",
					"task", task.ARN,
					"serviceID", serviceID,
					"instanceID", task.PodName,
					"error", err)
			} else {
				logging.Info("Task deregistered from service discovery",
					"task", task.ARN,
					"serviceID", serviceID,
					"instanceID", task.PodName)
			}
		}
	}
}

// updateServiceDiscoveryHealth updates the health status of a task in Service Discovery
func (tm *TaskManager) updateServiceDiscoveryHealth(ctx context.Context, task *storage.Task) {
	// Check if Service Discovery manager is available
	if tm.serviceDiscoveryManager == nil {
		return
	}

	// Check if the task was started by a service
	if !strings.HasPrefix(task.StartedBy, "ecs-svc/") {
		return
	}

	// Extract service name from StartedBy field
	serviceName := strings.TrimPrefix(task.StartedBy, "ecs-svc/")

	// Extract cluster name from ClusterARN
	clusterName := "default"
	if task.ClusterARN != "" {
		parts := strings.Split(task.ClusterARN, "/")
		if len(parts) > 0 {
			clusterName = parts[len(parts)-1]
		}
	}

	// Get the service from storage to check for ServiceRegistry metadata
	service, err := tm.storage.ServiceStore().Get(ctx, clusterName, serviceName)
	if err != nil || service == nil || service.ServiceRegistryMetadata == nil {
		return
	}

	// Update health status for each service discovery service
	for serviceID := range service.ServiceRegistryMetadata {
		// Map ECS health status to Service Discovery health status
		var sdHealthStatus string
		switch task.HealthStatus {
		case "HEALTHY":
			sdHealthStatus = "HEALTHY"
		case "UNHEALTHY":
			sdHealthStatus = "UNHEALTHY"
		case "UNKNOWN":
			sdHealthStatus = "UNKNOWN"
		default:
			sdHealthStatus = "UNKNOWN"
		}

		// Update the instance health status (use pod name as instance ID)
		if err := tm.serviceDiscoveryManager.UpdateInstanceHealthStatus(ctx, serviceID, task.PodName, sdHealthStatus); err != nil {
			logging.Warn("Failed to update task health status in service discovery",
				"task", task.ARN,
				"serviceID", serviceID,
				"instanceID", task.PodName,
				"healthStatus", sdHealthStatus,
				"error", err)
		} else {
			logging.Info("Task health status updated in service discovery",
				"task", task.ARN,
				"serviceID", serviceID,
				"instanceID", task.PodName,
				"healthStatus", sdHealthStatus)
		}
	}
}

// addPodInfoToTaskAttributes adds the pod name and namespace to the task's Attributes field
func (tm *TaskManager) addPodInfoToTaskAttributes(task *storage.Task, podName, namespace string) error {
	// Parse existing attributes or create new slice
	var attributes []map[string]interface{}
	if task.Attributes != "" && task.Attributes != "[]" {
		if err := json.Unmarshal([]byte(task.Attributes), &attributes); err != nil {
			logging.Warn("Failed to unmarshal existing task attributes", "task", task.ARN, "error", err)
			// Initialize empty slice if unmarshal fails
			attributes = []map[string]interface{}{}
		}
	}

	// Add pod name attribute
	podNameAttr := map[string]interface{}{
		"name":  "kecs.dev/pod-name",
		"value": podName,
	}
	attributes = append(attributes, podNameAttr)

	// Add namespace attribute
	namespaceAttr := map[string]interface{}{
		"name":  "kecs.dev/pod-namespace",
		"value": namespace,
	}
	attributes = append(attributes, namespaceAttr)

	// Serialize updated attributes back to JSON
	attributesJSON, err := json.Marshal(attributes)
	if err != nil {
		return fmt.Errorf("failed to marshal task attributes: %w", err)
	}

	task.Attributes = string(attributesJSON)
	return nil
}

// RestoreTask restores a standalone task from DuckDB to Kubernetes
func (tm *TaskManager) RestoreTask(ctx context.Context, task *storage.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}

	// Initialize client if needed
	if err := tm.InitializeClient(); err != nil {
		return fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}

	// Extract cluster name from ARN
	clusterName := tm.extractClusterNameFromARN(task.ClusterARN)
	namespace := fmt.Sprintf("ecs-%s", clusterName)

	// Check if pod already exists
	podName := fmt.Sprintf("task-%s", task.ID)
	_, err := tm.Clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err == nil {
		logging.Info("Pod already exists, skipping restoration",
			"pod", podName,
			"namespace", namespace)
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check pod existence: %w", err)
	}

	// Get task definition from storage
	family, revision := parseTaskDefinitionARN(task.TaskDefinitionARN)
	taskDef, err := tm.storage.TaskDefinitionStore().Get(ctx, family, revision)
	if err != nil {
		return fmt.Errorf("failed to get task definition: %w", err)
	}
	if taskDef == nil {
		return fmt.Errorf("task definition not found: %s", task.TaskDefinitionARN)
	}

	// Parse the task definition JSON from ContainerDefinitions field
	var taskDefData map[string]interface{}
	taskDefData = make(map[string]interface{})

	// Parse container definitions
	var containerDefs []interface{}
	if taskDef.ContainerDefinitions != "" {
		if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err != nil {
			return fmt.Errorf("failed to parse container definitions: %w", err)
		}
		taskDefData["containerDefinitions"] = containerDefs
	}

	// Create pod from task definition
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"ecs.amazonaws.com/task-arn": task.ARN,
				"ecs.cluster":                clusterName,
				"ecs.task":                   task.ID,
				"ecs.managed":                "true",
				"kecs.dev/task-id":           task.ID,
			},
		},
		Spec: corev1.PodSpec{
			Containers:    []corev1.Container{},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	// Add containers from task definition
	if containerDefs, ok := taskDefData["containerDefinitions"].([]interface{}); ok {
		for _, containerDef := range containerDefs {
			if containerMap, ok := containerDef.(map[string]interface{}); ok {
				container := tm.createContainerFromDefinition(containerMap)
				pod.Spec.Containers = append(pod.Spec.Containers, container)
			}
		}
	}

	// Ensure namespace exists
	if err := tm.ensureNamespace(ctx, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// Create the pod
	_, err = tm.Clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod in Kubernetes: %w", err)
	}

	// Start watching the pod for status updates
	go tm.watchPodStatus(context.Background(), task.ARN, namespace, podName)

	logging.Info("Successfully restored standalone task",
		"taskArn", task.ARN,
		"pod", podName,
		"namespace", namespace)

	return nil
}

// Helper functions for RestoreTask

// extractClusterNameFromARN extracts cluster name from ARN
func (tm *TaskManager) extractClusterNameFromARN(arn string) string {
	if arn == "" {
		return "default"
	}
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return "default"
}

// createContainerFromDefinition creates a Kubernetes container from ECS container definition
func (tm *TaskManager) createContainerFromDefinition(containerDef map[string]interface{}) corev1.Container {
	container := corev1.Container{
		Name: "main",
	}

	if name, ok := containerDef["name"].(string); ok {
		container.Name = name
	}

	if image, ok := containerDef["image"].(string); ok {
		container.Image = image
	}

	// Add port mappings
	if portMappings, ok := containerDef["portMappings"].([]interface{}); ok {
		for _, portMapping := range portMappings {
			if pm, ok := portMapping.(map[string]interface{}); ok {
				if containerPort, ok := pm["containerPort"].(float64); ok {
					container.Ports = append(container.Ports, corev1.ContainerPort{
						ContainerPort: int32(containerPort),
					})
				}
			}
		}
	}

	// Add environment variables
	if envVars, ok := containerDef["environment"].([]interface{}); ok {
		for _, envVar := range envVars {
			if ev, ok := envVar.(map[string]interface{}); ok {
				if name, ok := ev["name"].(string); ok {
					if value, ok := ev["value"].(string); ok {
						container.Env = append(container.Env, corev1.EnvVar{
							Name:  name,
							Value: value,
						})
					}
				}
			}
		}
	}

	return container
}

// ensureNamespace creates a namespace if it doesn't exist
func (tm *TaskManager) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := tm.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
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
			_, err = tm.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace: %w", err)
			}
			logging.Info("Created namespace", "namespace", namespace)
		} else {
			return fmt.Errorf("failed to get namespace: %w", err)
		}
	}
	return nil
}
