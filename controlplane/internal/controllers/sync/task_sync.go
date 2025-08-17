package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"

	"github.com/nandemo-ya/kecs/controlplane/internal/controllers/sync/mappers"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
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

	// Add to batch updater for efficient storage update
	c.batchUpdater.AddTaskUpdate(task)
	logging.Debug("Queued task update", "namespace", namespace, "name", name)

	// Log the sync result
	logging.Debug("Successfully synced task",
		"taskArn", task.ARN, "status", task.LastStatus, "health", task.HealthStatus)

	return nil
}

// handleDeletedPod handles the case when a pod is deleted
func (c *SyncController) handleDeletedPod(ctx context.Context, namespace, podName string) error {
	// Extract cluster info from namespace
	mapper := mappers.NewServiceStateMapper(c.accountID, c.region)
	clusterName, region := mapper.ExtractClusterInfoFromNamespace(namespace)
	if region == "" {
		region = c.region
	}

	// Generate cluster ARN
	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, c.accountID, clusterName)

	// Generate the task ARN that would have been used
	taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", region, c.accountID, namespace, podName)

	task, err := c.storage.TaskStore().Get(ctx, clusterARN, taskARN)
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

	logging.Debug("Marked task as STOPPED due to pod deletion", "taskArn", taskARN)
	return nil
}

// syncPod is the main entry point for pod synchronization
func (c *SyncController) syncPod(ctx context.Context, key string) error {
	logging.Debug("Syncing pod", "key", key)
	return c.syncTask(ctx, key)
}

// stringPtr returns a pointer to the string
func stringPtr(s string) *string {
	return &s
}
