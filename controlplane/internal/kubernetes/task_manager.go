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
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// TaskManager manages ECS tasks as Kubernetes pods
type TaskManager struct {
	clientset kubernetes.Interface
	storage   storage.Storage
}

// NewTaskManager creates a new task manager
func NewTaskManager(storage storage.Storage) (*TaskManager, error) {
	// In container mode or test mode, defer kubernetes client creation
	if config.GetBool("features.containerMode") || os.Getenv("KECS_TEST_MODE") == "true" {
		logging.Debug("Container/test mode enabled - deferring kubernetes client initialization")
		return &TaskManager{
			clientset: nil, // Will be initialized later
			storage:   storage,
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

	return &TaskManager{
		clientset: clientset,
		storage:   storage,
	}, nil
}

// InitializeClient initializes the kubernetes client if not already initialized
func (tm *TaskManager) InitializeClient() error {
	if tm.clientset != nil {
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

	tm.clientset = clientset
	logging.Debug("TaskManager kubernetes client initialized")
	return nil
}

// CreateTask creates a new task by deploying a pod
func (tm *TaskManager) CreateTask(ctx context.Context, pod *corev1.Pod, task *storage.Task, secrets map[string]*converters.SecretInfo) error {
	// In test mode, skip actual pod creation
	if tm.clientset == nil {
		logging.Debug("Kubernetes client not initialized - simulating task creation")
		task.PodName = "test-pod-" + task.ID
		task.Namespace = pod.Namespace
		// Don't override LastStatus - keep what was set by the caller
		task.Connectivity = "CONNECTED"
		task.ConnectivityAt = &task.CreatedAt
		
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
	createdPod, err := tm.clientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Update task with pod information
	task.PodName = createdPod.Name
	task.Namespace = createdPod.Namespace
	task.LastStatus = "PENDING"
	task.Connectivity = "CONNECTED"
	task.ConnectivityAt = &task.CreatedAt

	// Store task in database
	if err := tm.storage.TaskStore().Create(ctx, task); err != nil {
		// Try to clean up the pod if task storage fails
		_ = tm.clientset.CoreV1().Pods(pod.Namespace).Delete(ctx, createdPod.Name, metav1.DeleteOptions{})
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

	// Delete the pod (skip if no kubernetes client)
	if tm.clientset != nil && task.PodName != "" && task.Namespace != "" {
		err := tm.clientset.CoreV1().Pods(task.Namespace).Delete(ctx, task.PodName, metav1.DeleteOptions{})
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
	containers := tm.getContainerStatuses(pod)
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
	task.HealthStatus = tm.getHealthStatus(pod)

	return tm.storage.TaskStore().Update(ctx, task)
}

// watchPodStatus watches a pod for status changes
func (tm *TaskManager) watchPodStatus(ctx context.Context, taskARN, namespace, podName string) {
	// Use a field selector to watch only this specific pod
	fieldSelector := fields.OneTermEqualSelector("metadata.name", podName).String()

	watcher, err := tm.clientset.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
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
			return
		}
	}
}

// getContainerStatuses extracts container status information
func (tm *TaskManager) getContainerStatuses(pod *corev1.Pod) []types.Container {
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
		existingSecret, err := tm.clientset.CoreV1().Secrets(namespace).Get(ctx, info.SecretName, metav1.GetOptions{})
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

		_, err = tm.clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
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
	if err := EnsureNamespace(ctx, tm.clientset, namespace); err != nil {
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
	_, err := tm.clientset.AppsV1().Deployments(namespace).Create(ctx, k8sDeployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Update existing deployment
			_, err = tm.clientset.AppsV1().Deployments(namespace).Update(ctx, k8sDeployment, metav1.UpdateOptions{})
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
