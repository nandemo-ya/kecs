package bubbletea

import (
	"context"
	"fmt"
	"sync"

	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
)

// Adapter provides a compatibility layer between the existing ParallelTracker interface
// and the new Bubble Tea implementation
type Adapter struct {
	program *Program
	tasks   map[string]struct {
		total int
	}
	mu sync.Mutex
}

// NewAdapter creates a new Bubble Tea adapter that mimics ParallelTracker
func NewAdapter(title string) *Adapter {
	return &Adapter{
		program: NewProgram(title),
		tasks: make(map[string]struct {
			total int
		}),
	}
}

// Start begins the progress display
func (a *Adapter) Start() error {
	if a.program != nil {
		return a.program.Start()
	}
	return nil
}

// Stop stops the progress display
func (a *Adapter) Stop() {
	if a.program != nil {
		a.program.Stop()
	}
}

// AddTask adds a new task to track
func (a *Adapter) AddTask(id, name string, total int) {
	a.mu.Lock()
	a.tasks[id] = struct{ total int }{total: total}
	a.mu.Unlock()

	if a.program != nil {
		a.program.AddTask(id, name, float64(total))
	} else {
		// Fallback to console output
		fmt.Printf("• %s\n", name)
	}
}

// StartTask marks a task as running
func (a *Adapter) StartTask(id string) {
	if a.program != nil {
		a.program.UpdateTask(id, 0, "Starting...")
	}
}

// UpdateTask updates the progress of a task
func (a *Adapter) UpdateTask(id string, progress int, message string) {
	if a.program != nil {
		a.program.UpdateTask(id, float64(progress), message)
	} else {
		// Fallback to console output
		fmt.Printf("  ↳ %s (%d%%)\n", message, progress)
	}
}

// CompleteTask marks a task as completed
func (a *Adapter) CompleteTask(id string) {
	if a.program != nil {
		a.program.CompleteTask(id)
	} else {
		// Fallback to console output
		fmt.Printf("  ✓ Completed\n")
	}
}

// FailTask marks a task as failed
func (a *Adapter) FailTask(id string, err error) {
	if a.program != nil {
		a.program.FailTask(id, err)
	} else {
		// Fallback to console output
		fmt.Printf("  ✗ Failed: %v\n", err)
	}
}

// Log adds a log entry
func (a *Adapter) Log(level progress.LogLevel, format string, args ...interface{}) {
	levelStr := "INFO"
	switch level {
	case progress.LogLevelDebug:
		levelStr = "DEBUG"
	case progress.LogLevelWarning:
		levelStr = "WARN"
	case progress.LogLevelError:
		levelStr = "ERROR"
	}

	if a.program != nil {
		a.program.Log(levelStr, format, args...)
	} else {
		// Fallback to console output
		prefix := ""
		switch level {
		case progress.LogLevelError:
			prefix = "ERROR: "
		case progress.LogLevelWarning:
			prefix = "WARN: "
		case progress.LogLevelDebug:
			prefix = "DEBUG: "
		}
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// Summary returns a summary of all tasks (not implemented for Bubble Tea)
func (a *Adapter) Summary() string {
	return ""
}

// WithLogCapture is a no-op for Bubble Tea adapter as it handles logs internally
func (a *Adapter) WithLogCapture(logCapture *progress.LogCapture) *Adapter {
	// Bubble Tea handles log capture internally
	return a
}

// RunWithBubbleTea is a helper function to run a function with Bubble Tea progress
func RunWithBubbleTea(ctx context.Context, title string, fn func(*Adapter) error) error {
	adapter := NewAdapter(title)

	if err := adapter.Start(); err != nil {
		return fmt.Errorf("failed to start progress display: %w", err)
	}
	defer adapter.Stop()

	// Run the function
	if err := fn(adapter); err != nil {
		return err
	}

	// Mark as complete
	if adapter.program != nil {
		adapter.program.Complete()
	}

	return nil
}
