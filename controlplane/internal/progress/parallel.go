package progress

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pterm/pterm"
)

// ParallelTracker manages multiple operations running in parallel with visual progress
type ParallelTracker struct {
	tasks      map[string]*Task
	mu         sync.RWMutex
	area       *pterm.AreaPrinter
	title      string
	startTime  time.Time
	updateChan chan struct{}
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// Task represents a single task in parallel execution
type Task struct {
	Name        string
	Status      TaskStatus
	Progress    int
	Total       int
	Message     string
	StartTime   time.Time
	EndTime     time.Time
	Error       error
}

// TaskStatus represents the current status of a task
type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskCompleted
	TaskFailed
)

func (ts TaskStatus) String() string {
	switch ts {
	case TaskPending:
		return "Pending"
	case TaskRunning:
		return "Running"
	case TaskCompleted:
		return "Completed"
	case TaskFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

func (ts TaskStatus) Icon() string {
	switch ts {
	case TaskPending:
		return "⏳"
	case TaskRunning:
		return "🔄"
	case TaskCompleted:
		return "✅"
	case TaskFailed:
		return "❌"
	default:
		return "❓"
	}
}

// NewParallelTracker creates a new parallel progress tracker
func NewParallelTracker(title string) *ParallelTracker {
	pt := &ParallelTracker{
		tasks:      make(map[string]*Task),
		title:      title,
		startTime:  time.Now(),
		updateChan: make(chan struct{}, 1),
		stopChan:   make(chan struct{}),
	}

	// Initialize the area printer
	area, _ := pterm.DefaultArea.Start()
	pt.area = area

	// Start the render loop
	pt.wg.Add(1)
	go pt.renderLoop()

	return pt
}

// AddTask adds a new task to track
func (pt *ParallelTracker) AddTask(id, name string, total int) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.tasks[id] = &Task{
		Name:      name,
		Status:    TaskPending,
		Progress:  0,
		Total:     total,
		StartTime: time.Now(),
	}

	pt.triggerUpdate()
}

// StartTask marks a task as running
func (pt *ParallelTracker) StartTask(id string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if task, ok := pt.tasks[id]; ok {
		task.Status = TaskRunning
		task.StartTime = time.Now()
		pt.triggerUpdate()
	}
}

// UpdateTask updates the progress of a task
func (pt *ParallelTracker) UpdateTask(id string, progress int, message string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if task, ok := pt.tasks[id]; ok {
		task.Progress = progress
		task.Message = message
		if task.Status == TaskPending {
			task.Status = TaskRunning
		}
		pt.triggerUpdate()
	}
}

// CompleteTask marks a task as completed
func (pt *ParallelTracker) CompleteTask(id string) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if task, ok := pt.tasks[id]; ok {
		task.Status = TaskCompleted
		task.Progress = task.Total
		task.EndTime = time.Now()
		pt.triggerUpdate()
	}
}

// FailTask marks a task as failed
func (pt *ParallelTracker) FailTask(id string, err error) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	if task, ok := pt.tasks[id]; ok {
		task.Status = TaskFailed
		task.Error = err
		task.EndTime = time.Now()
		pt.triggerUpdate()
	}
}

// Stop stops the parallel tracker and cleans up
func (pt *ParallelTracker) Stop() {
	close(pt.stopChan)
	pt.wg.Wait()
	pt.area.Stop()
}

// triggerUpdate triggers a render update
func (pt *ParallelTracker) triggerUpdate() {
	select {
	case pt.updateChan <- struct{}{}:
	default:
	}
}

// renderLoop continuously renders the progress display
func (pt *ParallelTracker) renderLoop() {
	defer pt.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-pt.stopChan:
			return
		case <-pt.updateChan:
			pt.render()
		case <-ticker.C:
			pt.render()
		}
	}
}

// render renders the current state
func (pt *ParallelTracker) render() {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var output strings.Builder

	// Title with elapsed time
	elapsed := time.Since(pt.startTime).Round(time.Second)
	output.WriteString(pterm.DefaultHeader.Sprintf("%s (%s)", pt.title, elapsed))
	output.WriteString("\n\n")

	// Render each task
	for _, task := range pt.tasks {
		output.WriteString(pt.renderTask(task))
		output.WriteString("\n")
	}

	// Update the display
	pt.area.Update(output.String())
}

// renderTask renders a single task
func (pt *ParallelTracker) renderTask(task *Task) string {
	var output strings.Builder

	// Status icon and name
	output.WriteString(fmt.Sprintf("%s %s ", task.Status.Icon(), task.Name))

	// Progress bar or status
	switch task.Status {
	case TaskRunning:
		if task.Total > 0 {
			// Show progress bar
			percentage := float64(task.Progress) / float64(task.Total) * 100
			barWidth := 20
			filled := int(float64(barWidth) * (percentage / 100))
			empty := barWidth - filled

			output.WriteString("[")
			output.WriteString(strings.Repeat("█", filled))
			output.WriteString(strings.Repeat("░", empty))
			output.WriteString("] ")
			output.WriteString(fmt.Sprintf("%.0f%%", percentage))

			if task.Message != "" {
				output.WriteString(fmt.Sprintf(" - %s", task.Message))
			}
		} else {
			// Indeterminate progress
			elapsed := time.Since(task.StartTime).Round(time.Second)
			output.WriteString(fmt.Sprintf("(%s)", elapsed))
			if task.Message != "" {
				output.WriteString(fmt.Sprintf(" %s", task.Message))
			}
		}

	case TaskCompleted:
		duration := task.EndTime.Sub(task.StartTime).Round(time.Second)
		output.WriteString(pterm.Success.Sprintf("Completed in %s", duration))

	case TaskFailed:
		if task.Error != nil {
			output.WriteString(pterm.Error.Sprintf("Failed: %v", task.Error))
		} else {
			output.WriteString(pterm.Error.Sprint("Failed"))
		}

	case TaskPending:
		output.WriteString(pterm.Gray("Waiting..."))
	}

	return output.String()
}

// Summary returns a summary of all tasks
func (pt *ParallelTracker) Summary() string {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	completed := 0
	failed := 0
	running := 0
	pending := 0

	for _, task := range pt.tasks {
		switch task.Status {
		case TaskCompleted:
			completed++
		case TaskFailed:
			failed++
		case TaskRunning:
			running++
		case TaskPending:
			pending++
		}
	}

	totalDuration := time.Since(pt.startTime).Round(time.Second)

	var summary strings.Builder
	summary.WriteString(pterm.DefaultBox.Sprint(fmt.Sprintf(
		"Summary: %d completed, %d failed, %d running, %d pending\nTotal time: %s",
		completed, failed, running, pending, totalDuration,
	)))

	return summary.String()
}