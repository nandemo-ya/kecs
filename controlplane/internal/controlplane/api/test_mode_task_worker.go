package api

import (
	"context"
	"log"
	"os"
	"time"

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
	if os.Getenv("KECS_TEST_MODE") != "true" {
		return
	}

	// Check every 500ms for fast test execution
	w.ticker = time.NewTicker(500 * time.Millisecond)

	go func() {
		log.Println("Starting test mode task worker")
		for {
			select {
			case <-ctx.Done():
				log.Println("Test mode task worker stopping due to context cancellation")
				return
			case <-w.done:
				log.Println("Test mode task worker stopping")
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
			continue
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
				if time.Since(task.CreatedAt) > 100*time.Millisecond {
					task.LastStatus = "PENDING"
					updated = true
				}

			case "PENDING":
				// Move to RUNNING after another short delay
				// Check against CreatedAt since we don't have UpdatedAt
				if time.Since(task.CreatedAt) > 200*time.Millisecond {
					task.LastStatus = "RUNNING"
					task.StartedAt = &now
					task.PullStartedAt = &now
					task.PullStoppedAt = &now

					// Set container status
					// Note: This is a simple container status - actual implementation would parse task definition
					task.Containers = `[{"containerArn":"` + task.ARN + `/container-1","taskArn":"` + task.ARN + `","name":"app","lastStatus":"RUNNING","cpu":"256","memory":"512"}]`
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

					// Update container status
					task.Containers = `[{"containerArn":"` + task.ARN + `/container-1","taskArn":"` + task.ARN + `","name":"app","lastStatus":"STOPPED","exitCode":0,"cpu":"256","memory":"512"}]`
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

						// Set exit code 0 for successful completion
						task.Containers = `[{"containerArn":"` + task.ARN + `/container-1","taskArn":"` + task.ARN + `","name":"app","lastStatus":"STOPPED","exitCode":0,"cpu":"256","memory":"512"}]`
						updated = true
					}
				}
			}

			// Update the task if changed
			if updated {
				task.Version++
				if err := w.storage.TaskStore().Update(ctx, task); err != nil {
					log.Printf("Failed to update task %s: %v", task.ID, err)
				}
			}
		}
	}
}
