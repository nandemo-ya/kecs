package progress

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelTracker(t *testing.T) {
	// Use a custom tracker without automatic rendering to avoid race conditions
	pt := &ParallelTracker{
		tasks:      make(map[string]*Task),
		title:      "Test Parallel Operations",
		startTime:  time.Now(),
		updateChan: make(chan struct{}, 100),
		stopChan:   make(chan struct{}),
	}
	require.NotNil(t, pt)
	assert.Equal(t, "Test Parallel Operations", pt.title)

	// Add tasks
	pt.AddTask("task1", "Task One", 100)
	pt.AddTask("task2", "Task Two", 200)

	// Check task exists
	task1, ok := pt.tasks["task1"]
	require.True(t, ok)
	assert.Equal(t, "Task One", task1.Name)
	assert.Equal(t, 100, task1.Total)
	assert.Equal(t, TaskPending, task1.Status)

	// Start task
	pt.StartTask("task1")
	assert.Equal(t, TaskRunning, task1.Status)

	// Update task
	pt.UpdateTask("task1", 50, "Processing...")
	assert.Equal(t, 50, task1.Progress)
	assert.Equal(t, "Processing...", task1.Message)

	// Complete task
	pt.CompleteTask("task1")
	assert.Equal(t, TaskCompleted, task1.Status)
	assert.Equal(t, 100, task1.Progress)

	// Fail task
	testErr := errors.New("test error")
	pt.FailTask("task2", testErr)
	task2, _ := pt.tasks["task2"]
	assert.Equal(t, TaskFailed, task2.Status)
	assert.Equal(t, testErr, task2.Error)
}

func TestTaskStatus(t *testing.T) {
	tests := []struct {
		status   TaskStatus
		expected string
		icon     string
	}{
		{TaskPending, "Pending", "⏳"},
		{TaskRunning, "Running", "🔄"},
		{TaskCompleted, "Completed", "✅"},
		{TaskFailed, "Failed", "❌"},
		{TaskStatus(99), "Unknown", "❓"}, // Unknown status
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
			assert.Equal(t, tt.icon, tt.status.Icon())
		})
	}
}

func TestParallelTrackerRender(t *testing.T) {
	// Note: We need to disable automatic rendering for testing
	// to avoid race conditions with the area printer
	pt := &ParallelTracker{
		tasks:      make(map[string]*Task),
		title:      "Test Render",
		startTime:  time.Now(),
		updateChan: make(chan struct{}, 100),
		stopChan:   make(chan struct{}),
	}

	// Add and update tasks
	pt.AddTask("task1", "Task One", 100)
	pt.StartTask("task1")
	pt.UpdateTask("task1", 50, "Half way")

	// Test renderTask for different statuses
	task := &Task{
		Name:      "Test Task",
		Status:    TaskRunning,
		Progress:  75,
		Total:     100,
		Message:   "In progress",
		StartTime: time.Now(),
	}
	output := pt.renderTask(task)
	assert.Contains(t, output, "Test Task")
	assert.Contains(t, output, "75%")
	assert.Contains(t, output, "In progress")

	// Test completed task
	task.Status = TaskCompleted
	task.EndTime = time.Now().Add(1 * time.Second)
	output = pt.renderTask(task)
	assert.Contains(t, output, "Completed in")

	// Test failed task
	task.Status = TaskFailed
	task.Error = errors.New("test error")
	output = pt.renderTask(task)
	assert.Contains(t, output, "Failed:")
	assert.Contains(t, output, "test error")

	// Test pending task
	task.Status = TaskPending
	output = pt.renderTask(task)
	assert.Contains(t, output, "Waiting...")
}

func TestParallelTrackerSummary(t *testing.T) {
	// Use a custom tracker without automatic rendering
	pt := &ParallelTracker{
		tasks:      make(map[string]*Task),
		title:      "Test Summary",
		startTime:  time.Now(),
		updateChan: make(chan struct{}, 100),
		stopChan:   make(chan struct{}),
	}

	// Create tasks with different statuses
	pt.AddTask("completed1", "Completed Task 1", 100)
	pt.CompleteTask("completed1")

	pt.AddTask("completed2", "Completed Task 2", 100)
	pt.CompleteTask("completed2")

	pt.AddTask("failed", "Failed Task", 100)
	pt.FailTask("failed", errors.New("failed"))

	pt.AddTask("running", "Running Task", 100)
	pt.StartTask("running")

	pt.AddTask("pending", "Pending Task", 100)

	summary := pt.Summary()

	// Check summary contains expected counts
	assert.Contains(t, summary, "2 completed")
	assert.Contains(t, summary, "1 failed")
	assert.Contains(t, summary, "1 running")
	assert.Contains(t, summary, "1 pending")
	assert.Contains(t, summary, "Total time:")
}

func TestParallelTrackerConcurrency(t *testing.T) {
	// Use a custom tracker without automatic rendering to avoid race conditions
	pt := &ParallelTracker{
		tasks:      make(map[string]*Task),
		title:      "Test Concurrency",
		startTime:  time.Now(),
		updateChan: make(chan struct{}, 100),
		stopChan:   make(chan struct{}),
	}

	// Add tasks concurrently
	done := make(chan bool, 3)

	go func() {
		for i := 0; i < 10; i++ {
			pt.AddTask("task1", "Task 1", 100)
			pt.UpdateTask("task1", i*10, "Updating")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			pt.AddTask("task2", "Task 2", 100)
			pt.UpdateTask("task2", i*10, "Updating")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			pt.Summary() // Read operations
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we get here without deadlock or panic, the test passes
	assert.True(t, true)
}
