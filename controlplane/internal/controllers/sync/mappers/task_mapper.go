package mappers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	corev1 "k8s.io/api/core/v1"
)

// TaskStateMapper maps Kubernetes pod state to ECS task state
type TaskStateMapper struct{
	accountID string
	region    string
}

// NewTaskStateMapper creates a new task state mapper
func NewTaskStateMapper(accountID, region string) *TaskStateMapper {
	return &TaskStateMapper{
		accountID: accountID,
		region:    region,
	}
}

// MapPodPhaseToTaskStatus maps Kubernetes pod phase to ECS task status
func (m *TaskStateMapper) MapPodPhaseToTaskStatus(pod *corev1.Pod) (desiredStatus, lastStatus string) {
	// Check if pod is being deleted
	if pod.DeletionTimestamp != nil {
		return "STOPPED", "DEPROVISIONING"
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		// Check container statuses for more detail
		if len(pod.Status.ContainerStatuses) == 0 {
			return "RUNNING", "PROVISIONING"
		}
		// Check if any containers are being created
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason == "ContainerCreating" {
				return "RUNNING", "PENDING"
			}
		}
		return "RUNNING", "PROVISIONING"

	case corev1.PodRunning:
		// Check if all containers are ready
		allReady := true
		anyRunning := false
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				allReady = false
			}
			if cs.State.Running != nil {
				anyRunning = true
			}
		}
		
		// If all containers are ready, task is RUNNING
		if allReady {
			return "RUNNING", "RUNNING"
		}
		
		// If at least one container is running, task is no longer PENDING
		if anyRunning {
			return "RUNNING", "RUNNING"
		}
		
		// Otherwise, still activating
		return "RUNNING", "ACTIVATING"

	case corev1.PodSucceeded:
		return "STOPPED", "STOPPED"

	case corev1.PodFailed:
		return "STOPPED", "STOPPED"

	case corev1.PodUnknown:
		return "STOPPED", "STOPPED"

	default:
		return "STOPPED", "STOPPED"
	}
}

// MapPodToTask converts a Kubernetes pod to an ECS task
func (m *TaskStateMapper) MapPodToTask(pod *corev1.Pod) *storage.Task {
	if pod == nil {
		return nil
	}

	// Extract task ARN from pod labels or generate one
	taskARN := pod.Labels["ecs.amazonaws.com/task-arn"]
	if taskARN == "" {
		taskARN = m.generateTaskARN(pod)
	}

	// Get cluster ARN from namespace
	clusterARN := m.getClusterARNFromNamespace(pod.Namespace)

	// Get task definition ARN from pod labels or annotations
	taskDefARN := pod.Labels["ecs.amazonaws.com/task-definition-arn"]
	if taskDefARN == "" {
		// Try annotations
		taskDefARN = pod.Annotations["ecs.amazonaws.com/task-definition-arn"]
	}
	if taskDefARN == "" {
		// Try kecs.dev/task-definition annotation
		taskDefARN = pod.Annotations["kecs.dev/task-definition"]
	}

	// Map pod phase to task status
	desiredStatus, lastStatus := m.MapPodPhaseToTaskStatus(pod)

	// Extract service name from pod labels
	serviceName := pod.Labels["ecs.amazonaws.com/service-name"]
	if serviceName == "" {
		// Try kecs.dev/service label
		serviceName = pod.Labels["kecs.dev/service"]
	}
	startedBy := ""
	if serviceName != "" {
		startedBy = fmt.Sprintf("ecs-svc/%s", serviceName)
	}

	// Get task ID from pod label or use pod name as fallback
	taskID := pod.Labels["kecs.dev/task-id"]
	if taskID == "" {
		// Fallback to pod name for compatibility with existing pods
		taskID = pod.Name
	}

	// Create task object
	task := &storage.Task{
		ID:                taskID,
		ARN:               taskARN,
		ClusterARN:        clusterARN,
		TaskDefinitionARN: taskDefARN,
		DesiredStatus:     desiredStatus,
		LastStatus:        lastStatus,
		LaunchType:        "FARGATE",
		CreatedAt:         pod.CreationTimestamp.Time,
		Connectivity:      "CONNECTED",
		HealthStatus:      m.extractHealthStatus(pod),
		Containers:        m.serializeContainers(m.mapPodContainers(pod)),
		PullStartedAt:     m.getPullStartedTime(pod),
		PullStoppedAt:     m.getPullStoppedTime(pod),
		StartedAt:         m.getPodStartTime(pod),
		StoppedAt:         m.getPodStopTime(pod),
		StoppingAt:        m.getPodStoppingTime(pod),
		StoppedReason:     m.getPodStopReason(pod),
		StartedBy:         startedBy,
		Version:           1,
		PodName:           pod.Name,
		Namespace:         pod.Namespace,
		AccountID:         m.accountID,
		Region:            m.region,
	}

	// Set connectivity time if pod is running
	if pod.Status.Phase == corev1.PodRunning {
		task.ConnectivityAt = m.getPodStartTime(pod)
	}

	// Extract service name from pod labels
	if _, exists := pod.Labels["ecs.amazonaws.com/service-name"]; exists {
		// ServiceName field doesn't exist in storage.Task
		// This info would be part of the Task's relationship to Service
	}

	// Extract CPU and memory from pod spec
	task.CPU, task.Memory = m.extractResourceLimits(pod)

	return task
}

// mapPodContainers maps pod containers to ECS task containers
func (m *TaskStateMapper) mapPodContainers(pod *corev1.Pod) []generated.Container {
	var containers []generated.Container

	for _, container := range pod.Spec.Containers {
		// Find corresponding container status
		var status *corev1.ContainerStatus
		for j := range pod.Status.ContainerStatuses {
			if pod.Status.ContainerStatuses[j].Name == container.Name {
				status = &pod.Status.ContainerStatuses[j]
				break
			}
		}

		containerARN := fmt.Sprintf("%s/container/%s", pod.Labels["ecs.amazonaws.com/task-arn"], container.Name)
		taskARN := pod.Labels["ecs.amazonaws.com/task-arn"]
		taskContainer := generated.Container{
			Name:              &container.Name,
			Image:             &container.Image,
			LastStatus:        stringPtr(m.mapContainerStatus(status)),
			ContainerArn:      &containerARN,
			TaskArn:           &taskARN,
			NetworkInterfaces: m.getNetworkInterfaces(pod),
		}

		// Extract container state details
		if status != nil {
			taskContainer.ExitCode = m.getExitCodeInt32(status)
			taskContainer.Reason = stringPtr(m.getContainerReason(status))
			taskContainer.RuntimeId = &status.ContainerID
			taskContainer.ImageDigest = &status.ImageID
		}

		// Extract resource limits
		if container.Resources.Limits != nil {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				cpuStr := fmt.Sprintf("%d", cpu.MilliValue()/1000)
				taskContainer.Cpu = &cpuStr
			}
			if mem := container.Resources.Limits.Memory(); mem != nil {
				memStr := fmt.Sprintf("%d", mem.Value()/(1024*1024))
				taskContainer.Memory = &memStr
			}
		}

		containers = append(containers, taskContainer)
	}

	return containers
}

// mapContainerStatus maps Kubernetes container status to ECS container status
func (m *TaskStateMapper) mapContainerStatus(status *corev1.ContainerStatus) string {
	if status == nil {
		return "PENDING"
	}

	if status.State.Running != nil {
		if status.Ready {
			return "RUNNING"
		}
		return "ACTIVATING"
	}

	if status.State.Terminated != nil {
		return "STOPPED"
	}

	if status.State.Waiting != nil {
		switch status.State.Waiting.Reason {
		case "ContainerCreating":
			return "PENDING"
		case "PodInitializing":
			return "PROVISIONING"
		case "ImagePullBackOff", "ErrImagePull":
			return "PENDING"
		default:
			return "PROVISIONING"
		}
	}

	return "PENDING"
}

// mapContainerDesiredStatus maps container state to desired status
func (m *TaskStateMapper) mapContainerDesiredStatus(status *corev1.ContainerStatus) string {
	if status == nil {
		return "RUNNING"
	}

	if status.State.Terminated != nil {
		return "STOPPED"
	}

	return "RUNNING"
}

// Helper methods

func (m *TaskStateMapper) generateTaskARN(pod *corev1.Pod) string {
	clusterName, region := extractClusterInfoFromNamespace(pod.Namespace)
	if region == "" {
		region = m.region
	}
	if clusterName == "" {
		// Fallback to default if cluster name extraction failed
		clusterName = "default"
	}
	
	// Get task ID from pod label or use pod name as fallback
	taskID := pod.Labels["kecs.dev/task-id"]
	if taskID == "" {
		// Fallback to pod name for compatibility with existing pods
		taskID = pod.Name
	}
	
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", region, m.accountID, clusterName, taskID)
}

func (m *TaskStateMapper) getClusterARNFromNamespace(namespace string) string {
	// Extract cluster name and region from namespace
	clusterName, region := extractClusterInfoFromNamespace(namespace)
	if region == "" {
		region = m.region
	}
	return fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, m.accountID, clusterName)
}

func (m *TaskStateMapper) extractHealthStatus(pod *corev1.Pod) string {
	for _, container := range pod.Status.ContainerStatuses {
		if !container.Ready && container.State.Running != nil {
			// Container is running but not ready, might be failing health checks
			return "UNHEALTHY"
		}
	}

	// All containers ready or not running
	if pod.Status.Phase == corev1.PodRunning {
		allReady := true
		for _, container := range pod.Status.ContainerStatuses {
			if !container.Ready {
				allReady = false
				break
			}
		}
		if allReady {
			return "HEALTHY"
		}
	}

	return "UNKNOWN"
}

func (m *TaskStateMapper) getExitCodeInt32(status *corev1.ContainerStatus) *int32 {
	if status != nil && status.State.Terminated != nil {
		code := status.State.Terminated.ExitCode
		return &code
	}
	return nil
}

func (m *TaskStateMapper) getContainerReason(status *corev1.ContainerStatus) string {
	if status == nil {
		return ""
	}

	if status.State.Terminated != nil {
		return status.State.Terminated.Reason
	}

	if status.State.Waiting != nil {
		return status.State.Waiting.Reason
	}

	return ""
}

func (m *TaskStateMapper) getPodStartTime(pod *corev1.Pod) *time.Time {
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running != nil {
			return &container.State.Running.StartedAt.Time
		}
	}
	return nil
}

func (m *TaskStateMapper) getPodStopTime(pod *corev1.Pod) *time.Time {
	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		var latestStop *time.Time
		for _, container := range pod.Status.ContainerStatuses {
			if container.State.Terminated != nil {
				if latestStop == nil || container.State.Terminated.FinishedAt.Time.After(*latestStop) {
					latestStop = &container.State.Terminated.FinishedAt.Time
				}
			}
		}
		return latestStop
	}
	return nil
}

func (m *TaskStateMapper) getPodStoppingTime(pod *corev1.Pod) *time.Time {
	if pod.DeletionTimestamp != nil {
		return &pod.DeletionTimestamp.Time
	}
	return nil
}

func (m *TaskStateMapper) getPodStopReason(pod *corev1.Pod) string {
	if pod.Status.Phase == corev1.PodFailed {
		return pod.Status.Reason
	}

	// Check container reasons
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
			return container.State.Terminated.Reason
		}
	}

	if pod.DeletionTimestamp != nil {
		return "Task stopped by user"
	}

	return ""
}

func (m *TaskStateMapper) getPullStartedTime(pod *corev1.Pod) *time.Time {
	// This would need to be extracted from pod events
	// For now, return pod creation time as an approximation
	return &pod.CreationTimestamp.Time
}

func (m *TaskStateMapper) getPullStoppedTime(pod *corev1.Pod) *time.Time {
	// This would need to be extracted from pod events
	// For now, check if any container is running
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Running != nil {
			return &container.State.Running.StartedAt.Time
		}
	}
	return nil
}

func (m *TaskStateMapper) getNetworkInterfaces(pod *corev1.Pod) []generated.NetworkInterface {
	if pod.Status.PodIP == "" {
		return nil
	}

	attachmentID := fmt.Sprintf("eni-%s", pod.UID)
	return []generated.NetworkInterface{
		{
			AttachmentId:       &attachmentID,
			PrivateIpv4Address: &pod.Status.PodIP,
		},
	}
}

// serializeContainers converts container objects to JSON string for storage
func (m *TaskStateMapper) serializeContainers(containers []generated.Container) string {
	if len(containers) == 0 {
		return "[]"
	}
	data, err := json.Marshal(containers)
	if err != nil {
		return "[]"
	}
	return string(data)
}


func (m *TaskStateMapper) extractResourceLimits(pod *corev1.Pod) (cpu, memory string) {
	totalCPU := int64(0)
	totalMemory := int64(0)

	for _, container := range pod.Spec.Containers {
		if container.Resources.Limits != nil {
			if cpuLimit := container.Resources.Limits.Cpu(); cpuLimit != nil {
				totalCPU += cpuLimit.MilliValue()
			}
			if memLimit := container.Resources.Limits.Memory(); memLimit != nil {
				totalMemory += memLimit.Value()
			}
		}
	}

	if totalCPU > 0 {
		cpu = fmt.Sprintf("%d", totalCPU/1000) // Convert milliCPU to CPU units
	}

	if totalMemory > 0 {
		memory = fmt.Sprintf("%d", totalMemory/(1024*1024)) // Convert bytes to MiB
	}

	return cpu, memory
}

// extractClusterInfoFromNamespace is a helper function shared with service mapper
func extractClusterInfoFromNamespace(namespace string) (clusterName, region string) {
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