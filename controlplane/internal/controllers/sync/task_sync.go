package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"

	"github.com/nandemo-ya/kecs/controlplane/internal/controllers/sync/mappers"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// syncTask syncs a pod to ECS task state
func (c *SyncController) syncTask(ctx context.Context, key string) error {
	logging.Debug("syncTask called", "key", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Get the pod
	logging.Debug("Getting pod", "namespace", namespace, "name", name)
	pod, err := c.podLister.Pods(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod was deleted, update task to STOPPED
			logging.Debug("Pod not found, handling deletion", "namespace", namespace, "name", name)
			return c.handleDeletedPod(ctx, namespace, name)
		}
		return fmt.Errorf("error fetching pod: %v", err)
	}

	// Check if this is an ECS-managed pod
	if !isECSManagedPod(pod) {
		logging.Debug("Ignoring non-ECS pod", "name", name)
		return nil
	}

	// Map pod to task
	mapper := mappers.NewTaskStateMapper(c.accountID, c.region)
	task := mapper.MapPodToTask(pod)
	if task == nil {
		return fmt.Errorf("failed to map pod to task")
	}

	logging.Debug("Mapped pod to task", "taskArn", task.ARN, "status", task.LastStatus)

	// Check if task exists in storage
	existingTask, err := c.storage.TaskStore().Get(ctx, task.ClusterARN, task.ARN)
	if err != nil && !errors.IsNotFound(err) {
		logging.Warn("Failed to check existing task", "error", err)
	}

	// Track state transitions for cluster count updates
	var wasRunning, isRunning bool
	if existingTask != nil {
		wasRunning = existingTask.LastStatus == "RUNNING"
	}
	isRunning = task.LastStatus == "RUNNING"

	// Add to batch updater for efficient storage update
	c.batchUpdater.AddTaskUpdate(task)
	logging.Debug("Queued task update", "namespace", namespace, "name", name)

	// Update cluster counts if task state changed
	if wasRunning != isRunning {
		if err := c.updateClusterTaskCount(ctx, task.ClusterARN); err != nil {
			logging.Warn("Failed to update cluster task count", "error", err)
		}
	}

	// Log the sync result
	logging.Debug("Successfully synced task",
		"taskArn", task.ARN, "status", task.LastStatus, "health", task.HealthStatus)

	return nil
}

// handleDeletedPod handles the case when a pod is deleted
func (c *SyncController) handleDeletedPod(ctx context.Context, namespace, podName string) error {
	logging.Info("Handling deleted pod", "namespace", namespace, "pod", podName)

	// Try to get the pod from cache first to extract task ID
	// When a pod is deleted, it might still be in the informer cache with DeletionTimestamp set
	var taskID string
	pod, err := c.podLister.Pods(namespace).Get(podName)
	if err == nil && pod != nil {
		taskID = pod.Labels["kecs.dev/task-id"]
		if taskID != "" {
			logging.Info("Found task ID from pod labels", "taskID", taskID)
		}
	} else {
		logging.Debug("Pod not found in cache, will try to find task by pod name", "error", err)
	}

	// If we couldn't get the task ID from the pod, try to find the task by pod name
	// This is a fallback for pods that might not have the task-id label

	// Extract cluster info from namespace
	mapper := mappers.NewServiceStateMapper(c.accountID, c.region)
	clusterName, region := mapper.ExtractClusterInfoFromNamespace(namespace)
	if region == "" {
		region = c.region
	}

	// Generate cluster ARN
	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, c.accountID, clusterName)

	// Generate the task ARN or find by pod name
	var task *storage.Task
	if taskID != "" {
		// Use the actual task ID if we have it
		taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", region, c.accountID, clusterName, taskID)
		task, err = c.storage.TaskStore().Get(ctx, clusterARN, taskARN)
	} else {
		// Fallback: try to find task by pod name in the database
		// For service-managed pods, the pod name is stored in the pod_name field
		logging.Info("No task ID in pod labels, searching for task by pod name", "podName", podName)

		// Get all tasks in the cluster and find the one with matching pod name
		filters := storage.TaskFilters{}
		tasks, err := c.storage.TaskStore().List(ctx, clusterARN, filters)
		if err == nil {
			for _, t := range tasks {
				if t.PodName == podName {
					task = t
					logging.Info("Found task by pod name", "taskArn", t.ARN, "podName", podName)
					break
				}
			}
		}

		if task == nil {
			logging.Warn("Could not find task for deleted pod", "podName", podName, "namespace", namespace)
			return nil
		}
	}
	if err != nil {
		if isNotFound(err) {
			// Task doesn't exist, nothing to do
			logging.Debug("Task not found for deleted pod", "namespace", namespace, "pod", podName)
			return nil
		}
		return fmt.Errorf("error getting task: %v", err)
	}

	// Check if task is nil
	if task == nil {
		logging.Debug("Task is nil for deleted pod", "namespace", namespace, "pod", podName)
		return nil
	}

	// Update task to STOPPED
	previousStatus := task.LastStatus
	task.DesiredStatus = "STOPPED"
	task.LastStatus = "STOPPED"
	task.StoppedReason = "Pod deleted"
	task.StoppedAt = &[]time.Time{time.Now()}[0]

	// Update all containers to STOPPED
	var containers []generated.Container
	if task.Containers != "" {
		if err := json.Unmarshal([]byte(task.Containers), &containers); err == nil {
			for i := range containers {
				containers[i].LastStatus = stringPtr("STOPPED")
				// DesiredStatus field doesn't exist in generated.Container
			}
			// Serialize back to JSON
			if data, err := json.Marshal(containers); err == nil {
				task.Containers = string(data)
			}
		}
	}

	// Add to batch updater
	c.batchUpdater.AddTaskUpdate(task)

	// Update cluster task count after marking task as stopped
	if err := c.updateClusterTaskCount(ctx, clusterARN); err != nil {
		logging.Warn("Failed to update cluster task count after pod deletion", "error", err)
	}

	logging.Info("Marked task as STOPPED due to pod deletion",
		"taskArn", task.ARN,
		"podName", podName,
		"previousStatus", previousStatus)
	return nil
}

// syncPod is the main entry point for pod synchronization
func (c *SyncController) syncPod(ctx context.Context, key string) error {
	logging.Debug("Syncing pod", "key", key)
	return c.syncTask(ctx, key)
}

// updateClusterTaskCount updates the cluster's running and pending task counts
func (c *SyncController) updateClusterTaskCount(ctx context.Context, clusterARN string) error {
	// Extract cluster name from ARN
	parts := strings.Split(clusterARN, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid cluster ARN: %s", clusterARN)
	}
	clusterName := parts[len(parts)-1]

	// Get cluster from storage
	cluster, err := c.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}
	if cluster == nil {
		return fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Count running and pending tasks
	filters := storage.TaskFilters{}
	tasks, err := c.storage.TaskStore().List(ctx, clusterARN, filters)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	var runningCount, pendingCount int
	for _, task := range tasks {
		switch task.LastStatus {
		case "RUNNING":
			runningCount++
		case "PENDING":
			pendingCount++
		}
	}

	// Update cluster counts
	cluster.RunningTasksCount = runningCount
	cluster.PendingTasksCount = pendingCount

	// Save updated cluster
	if err := c.storage.ClusterStore().Update(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	logging.Info("Updated cluster task counts",
		"cluster", clusterName,
		"running", runningCount,
		"pending", pendingCount)

	return nil
}

// stringPtr returns a pointer to the string
func stringPtr(s string) *string {
	return &s
}
