package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// TestModeTaskWorker manages task lifecycle in test mode
type TestModeTaskWorker struct {
	storage storage.Storage
	ticker  *time.Ticker
	done    chan struct{}
}

// NewTestModeTaskWorker creates a new test mode task worker
func NewTestModeTaskWorker(storage storage.Storage) *TestModeTaskWorker {
	return &TestModeTaskWorker{
		storage: storage,
		done:    make(chan struct{}),
	}
}

// Start begins the background task processing
func (w *TestModeTaskWorker) Start(ctx context.Context) {
	// Only run in test mode
	if !config.GetBool("features.testMode") {
		return
	}

	// Check every 100ms for fast test execution
	w.ticker = time.NewTicker(100 * time.Millisecond)

	go func() {
		logging.Info("Test mode task worker: Started successfully, checking tasks every 100ms")
		for {
			select {
			case <-ctx.Done():
				logging.Info("Test mode task worker: Stopping due to context cancellation")
				return
			case <-w.done:
				logging.Info("Test mode task worker: Stopping")
				return
			case <-w.ticker.C:
				w.processTasks(ctx)
			}
		}
	}()
}

// Stop halts the background task processing
func (w *TestModeTaskWorker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	close(w.done)
}

// processTasks simulates task lifecycle transitions in test mode
func (w *TestModeTaskWorker) processTasks(ctx context.Context) {
	// Get all clusters first
	clusters, err := w.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Test mode worker: Failed to list clusters", "error", err)
		return
	}

	// Process tasks for each cluster
	for _, cluster := range clusters {
		// List tasks with no filters to get all tasks
		filters := storage.TaskFilters{
			MaxResults: 1000, // Get a large batch
		}
		tasks, err := w.storage.TaskStore().List(ctx, cluster.ARN, filters)
		if err != nil {
			logging.Error("Test mode worker: Failed to list tasks for cluster", "cluster", cluster.Name, "error", err)
			continue
		}
		
		if len(tasks) > 0 {
			logging.Debug("Test mode worker: Processing tasks for cluster", "count", len(tasks), "cluster", cluster.Name)
		}

		for _, task := range tasks {
			// Skip if task is already in final state
			if task.LastStatus == "STOPPED" || task.DesiredStatus == "STOPPED" {
				continue
			}

			// Simulate task lifecycle transitions
			updated := false
			now := time.Now()

			switch task.LastStatus {
			case "PROVISIONING":
				// Move to PENDING after a short delay
				timeSinceCreated := time.Since(task.CreatedAt)
				if timeSinceCreated > 50*time.Millisecond {
					logging.Debug("Test mode worker: Task transitioning from PROVISIONING to PENDING", "taskId", task.ID, "age", timeSinceCreated)
					task.LastStatus = "PENDING"
					updated = true
				}

			case "PENDING":
				// Move to RUNNING after another short delay
				// Check against CreatedAt since we don't have UpdatedAt
				timeSinceCreated := time.Since(task.CreatedAt)
				if timeSinceCreated > 100*time.Millisecond {
					logging.Debug("Test mode worker: Task transitioning from PENDING to RUNNING", "taskId", task.ID, "age", timeSinceCreated)
					task.LastStatus = "RUNNING"
					task.StartedAt = &now
					task.PullStartedAt = &now
					task.PullStoppedAt = &now

					// Update container status to RUNNING
					// Parse existing containers JSON and update status
					var containers []generated.Container
					if err := json.Unmarshal([]byte(task.Containers), &containers); err == nil && len(containers) > 0 {
						// Update all containers to RUNNING
						for i := range containers {
							containers[i].LastStatus = ptr.String("RUNNING")
						}
						if updatedContainersJSON, err := json.Marshal(containers); err == nil {
							task.Containers = string(updatedContainersJSON)
						}
					}
					updated = true
				}

			case "RUNNING":
				// If desired status is STOPPED, move to STOPPED
				if task.DesiredStatus == "STOPPED" {
					task.LastStatus = "STOPPED"
					task.StoppedAt = &now
					task.ExecutionStoppedAt = &now
					if task.StoppedReason == "" {
						task.StoppedReason = "Task stopped"
					}
					task.StopCode = "TaskStoppedByUser"

					// Update container status to STOPPED
					var containers []generated.Container
					if err := json.Unmarshal([]byte(task.Containers), &containers); err == nil && len(containers) > 0 {
						// Update all containers to STOPPED
						for i := range containers {
							containers[i].LastStatus = ptr.String("STOPPED")
							containers[i].ExitCode = ptr.Int32(0)
						}
						if updatedContainersJSON, err := json.Marshal(containers); err == nil {
							task.Containers = string(updatedContainersJSON)
						}
					}
					updated = true
				} else {
					// For short-lived tasks (like echo commands), auto-stop after a brief period
					if task.StartedAt != nil && time.Since(*task.StartedAt) > 2*time.Second {
						// Check if this is a quick task (echo, true, etc)
						task.LastStatus = "STOPPED"
						task.DesiredStatus = "STOPPED"
						task.StoppedAt = &now
						task.ExecutionStoppedAt = &now
						task.StoppedReason = "Essential container in task exited"
						task.StopCode = "EssentialContainerExited"

						// Update container status to STOPPED with exit code 0
						var containers []generated.Container
						if err := json.Unmarshal([]byte(task.Containers), &containers); err == nil && len(containers) > 0 {
							// Update all containers to STOPPED
							for i := range containers {
								containers[i].LastStatus = ptr.String("STOPPED")
								containers[i].ExitCode = ptr.Int32(0)
							}
							if updatedContainersJSON, err := json.Marshal(containers); err == nil {
								task.Containers = string(updatedContainersJSON)
							}
						}
						updated = true
					}
				}
			}

			// Update the task if changed
			if updated {
				task.Version++
				if err := w.storage.TaskStore().Update(ctx, task); err != nil {
					logging.Error("Test mode worker: Failed to update task", "taskId", task.ID, "error", err)
				} else {
					logging.Debug("Test mode worker: Successfully updated task", "taskId", task.ID, "status", task.LastStatus)
				}
			}
		}
	}
}
