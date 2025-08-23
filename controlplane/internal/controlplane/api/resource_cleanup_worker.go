package api

import (
	"context"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// ResourceCleanupWorker manages cleanup of stale resources
type ResourceCleanupWorker struct {
	storage storage.Storage
	ticker  *time.Ticker
	done    chan struct{}

	// Configuration
	enabled           bool
	interval          time.Duration
	taskRetention     time.Duration
	serviceRetention  time.Duration
	instanceRetention time.Duration
	taskSetRetention  time.Duration
	logRetention      time.Duration
}

// NewResourceCleanupWorker creates a new resource cleanup worker
func NewResourceCleanupWorker(storage storage.Storage) *ResourceCleanupWorker {
	return &ResourceCleanupWorker{
		storage:           storage,
		done:              make(chan struct{}),
		enabled:           config.GetBool("cleanup.enabled"),
		interval:          config.GetDuration("cleanup.interval", 5*time.Minute),
		taskRetention:     config.GetDuration("cleanup.task.retention", 1*time.Hour),
		serviceRetention:  config.GetDuration("cleanup.service.retention", 24*time.Hour),
		instanceRetention: config.GetDuration("cleanup.containerInstance.retention", 1*time.Hour),
		taskSetRetention:  config.GetDuration("cleanup.taskSet.retention", 24*time.Hour),
		logRetention:      config.GetDuration("cleanup.log.retention", 7*24*time.Hour),
	}
}

// Start begins the background resource cleanup
func (w *ResourceCleanupWorker) Start(ctx context.Context) {
	if !w.enabled {
		logging.Info("Resource cleanup worker: Disabled by configuration")
		return
	}

	w.ticker = time.NewTicker(w.interval)

	go func() {
		logging.Info("Resource cleanup worker: Started successfully",
			"interval", w.interval,
			"taskRetention", w.taskRetention,
			"serviceRetention", w.serviceRetention,
			"instanceRetention", w.instanceRetention,
			"taskSetRetention", w.taskSetRetention,
			"logRetention", w.logRetention,
		)

		// Run initial cleanup
		w.cleanupResources(ctx)

		for {
			select {
			case <-ctx.Done():
				logging.Info("Resource cleanup worker: Stopping due to context cancellation")
				return
			case <-w.done:
				logging.Info("Resource cleanup worker: Stopping")
				return
			case <-w.ticker.C:
				w.cleanupResources(ctx)
			}
		}
	}()
}

// Stop halts the background resource cleanup
func (w *ResourceCleanupWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	close(w.done)
}

// cleanupResources orchestrates cleanup of all resource types
func (w *ResourceCleanupWorker) cleanupResources(ctx context.Context) {
	logging.Debug("Resource cleanup worker: Starting cleanup cycle")

	totalDeleted := 0

	// Cleanup stopped tasks
	if count := w.cleanupStoppedTasks(ctx); count > 0 {
		totalDeleted += count
		logging.Info("Resource cleanup worker: Deleted stopped tasks", "count", count)
	}

	// Cleanup deleted services
	if count := w.cleanupDeletedServices(ctx); count > 0 {
		totalDeleted += count
		logging.Info("Resource cleanup worker: Deleted services", "count", count)
	}

	// Cleanup stale container instances
	if count := w.cleanupStaleContainerInstances(ctx); count > 0 {
		totalDeleted += count
		logging.Info("Resource cleanup worker: Deleted container instances", "count", count)
	}

	// Cleanup orphaned task sets
	if count := w.cleanupOrphanedTaskSets(ctx); count > 0 {
		totalDeleted += count
		logging.Info("Resource cleanup worker: Deleted task sets", "count", count)
	}

	// Cleanup old task logs
	if count := w.cleanupOldTaskLogs(ctx); count > 0 {
		totalDeleted += count
		logging.Info("Resource cleanup worker: Deleted old logs", "count", count)
	}

	if totalDeleted > 0 {
		logging.Info("Resource cleanup worker: Cleanup cycle completed", "totalDeleted", totalDeleted)
	} else {
		logging.Debug("Resource cleanup worker: Cleanup cycle completed, nothing to clean")
	}
}

// cleanupStoppedTasks removes tasks that have been stopped for longer than retention period
func (w *ResourceCleanupWorker) cleanupStoppedTasks(ctx context.Context) int {
	cutoff := time.Now().Add(-w.taskRetention)

	// Get all clusters
	clusters, err := w.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Resource cleanup worker: Failed to list clusters", "error", err)
		return 0
	}

	totalDeleted := 0
	for _, cluster := range clusters {
		// Clean up STOPPED tasks
		count, err := w.storage.TaskStore().DeleteOlderThan(ctx, cluster.ARN, cutoff, "STOPPED")
		if err != nil {
			logging.Error("Resource cleanup worker: Failed to delete stopped tasks",
				"cluster", cluster.Name, "error", err)
			continue
		}
		totalDeleted += count

		// Also clean up DEPROVISIONING tasks that are stuck
		// This handles cases where pod deletion events were not properly processed
		count, err = w.storage.TaskStore().DeleteOlderThan(ctx, cluster.ARN, cutoff, "DEPROVISIONING")
		if err != nil {
			logging.Error("Resource cleanup worker: Failed to delete deprovisioning tasks",
				"cluster", cluster.Name, "error", err)
			continue
		}
		totalDeleted += count
	}

	return totalDeleted
}

// cleanupDeletedServices removes services marked for deletion
func (w *ResourceCleanupWorker) cleanupDeletedServices(ctx context.Context) int {
	cutoff := time.Now().Add(-w.serviceRetention)

	// Get all clusters
	clusters, err := w.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Resource cleanup worker: Failed to list clusters", "error", err)
		return 0
	}

	totalDeleted := 0
	for _, cluster := range clusters {
		count, err := w.storage.ServiceStore().DeleteMarkedForDeletion(ctx, cluster.ARN, cutoff)
		if err != nil {
			logging.Error("Resource cleanup worker: Failed to delete services",
				"cluster", cluster.Name, "error", err)
			continue
		}
		totalDeleted += count
	}

	return totalDeleted
}

// cleanupStaleContainerInstances removes container instances that are no longer active
func (w *ResourceCleanupWorker) cleanupStaleContainerInstances(ctx context.Context) int {
	cutoff := time.Now().Add(-w.instanceRetention)

	// Get all clusters
	clusters, err := w.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Resource cleanup worker: Failed to list clusters", "error", err)
		return 0
	}

	totalDeleted := 0
	for _, cluster := range clusters {
		count, err := w.storage.ContainerInstanceStore().DeleteStale(ctx, cluster.ARN, cutoff)
		if err != nil {
			logging.Error("Resource cleanup worker: Failed to delete container instances",
				"cluster", cluster.Name, "error", err)
			continue
		}
		totalDeleted += count
	}

	return totalDeleted
}

// cleanupOrphanedTaskSets removes task sets without associated services
func (w *ResourceCleanupWorker) cleanupOrphanedTaskSets(ctx context.Context) int {
	// Get all clusters
	clusters, err := w.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Resource cleanup worker: Failed to list clusters", "error", err)
		return 0
	}

	totalDeleted := 0
	for _, cluster := range clusters {
		count, err := w.storage.TaskSetStore().DeleteOrphaned(ctx, cluster.ARN)
		if err != nil {
			logging.Error("Resource cleanup worker: Failed to delete orphaned task sets",
				"cluster", cluster.Name, "error", err)
			continue
		}
		totalDeleted += count
	}

	return totalDeleted
}

// cleanupOldTaskLogs removes log entries older than retention period
func (w *ResourceCleanupWorker) cleanupOldTaskLogs(ctx context.Context) int {
	// TODO: Implement when TaskLogStore is available
	// For now, return 0 as this store doesn't exist yet
	return 0
}
