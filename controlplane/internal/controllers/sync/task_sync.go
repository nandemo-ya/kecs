package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controllers/sync/mappers"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// syncTask syncs a pod to ECS task state
func (c *SyncController) syncTask(ctx context.Context, key string) error {
	klog.Infof("syncTask called with key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Get the pod
	klog.Infof("Getting pod %s/%s", namespace, name)
	pod, err := c.podLister.Pods(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod was deleted, update task to STOPPED
			klog.Infof("Pod %s/%s not found, handling deletion", namespace, name)
			return c.handleDeletedPod(ctx, namespace, name)
		}
		return fmt.Errorf("error fetching pod: %v", err)
	}

	// Check if this is an ECS-managed pod
	if !isECSManagedPod(pod) {
		klog.Infof("Ignoring non-ECS pod: %s", name)
		return nil
	}

	// Map pod to task
	mapper := mappers.NewTaskStateMapper(c.accountID, c.region)
	task := mapper.MapPodToTask(pod)
	if task == nil {
		return fmt.Errorf("failed to map pod to task")
	}

	klog.Infof("Mapped pod to task - taskArn: %s, status: %s", task.ARN, task.LastStatus)

	// Add to batch updater for efficient storage update
	c.batchUpdater.AddTaskUpdate(task)
	klog.Infof("Queued task update for pod %s/%s", namespace, name)

	// Log the sync result
	klog.V(2).Infof("Successfully synced task %s: status=%s, health=%s",
		task.ARN, task.LastStatus, task.HealthStatus)

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
			klog.V(4).Infof("Task not found for deleted pod %s/%s", namespace, podName)
			return nil
		}
		return fmt.Errorf("error getting task: %v", err)
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

	klog.V(2).Infof("Marked task %s as STOPPED due to pod deletion", taskARN)
	return nil
}

// syncPod is the main entry point for pod synchronization
func (c *SyncController) syncPod(ctx context.Context, key string) error {
	klog.V(4).Infof("Syncing pod: %s", key)
	return c.syncTask(ctx, key)
}

// stringPtr returns a pointer to the string
func stringPtr(s string) *string {
	return &s
}
