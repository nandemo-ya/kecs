package bubbletea

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"k8s.io/klog/v2"
)

// Program wraps the Bubble Tea program for progress tracking
type Program struct {
	program     *tea.Program
	model       Model
	logCapture  *logCapture
	mu          sync.Mutex
	started     bool
	done        chan struct{}
}

// NewProgram creates a new progress tracking program
func NewProgram(title string) *Program {
	model := New(title)
	
	// Create log capture
	lc := &logCapture{
		program: nil, // Will be set after program creation
		buffer:  make([]logEntry, 0),
	}
	
	// Create the program with full screen mode
	teaProgram := tea.NewProgram(
		model,
		tea.WithAltScreen(),     // Use alternate screen buffer
		// No mouse capture - allows text selection while keyboard controls work for scrolling
	)
	
	lc.program = teaProgram
	
	return &Program{
		program:    teaProgram,
		model:      model,
		logCapture: lc,
		done:       make(chan struct{}),
	}
}

// Start begins the progress display
func (p *Program) Start() error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return nil
	}
	p.started = true
	p.mu.Unlock()
	
	// Redirect log output first
	p.logCapture.Start()
	
	// Also capture stderr for external tools
	p.logCapture.StartStderrCapture()
	
	// Run the program in a goroutine
	go func() {
		if _, err := p.program.Run(); err != nil {
			// Don't use log here as it might be captured
			fmt.Fprintf(os.Stderr, "Error running progress display: %v\n", err)
		}
		close(p.done)
	}()
	
	// Wait a bit longer to ensure the TUI is fully initialized
	time.Sleep(300 * time.Millisecond)
	
	return nil
}

// Stop stops the progress display
func (p *Program) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.started {
		return
	}
	
	// Restore log output and stderr
	p.logCapture.Stop()
	p.logCapture.StopStderrCapture()
	
	// Send quit message
	p.program.Send(tea.Quit())
	
	// Wait for program to finish
	<-p.done
}

// AddTask adds a new task to track
func (p *Program) AddTask(id, name string, total float64) {
	if p.program != nil {
		p.program.Send(AddTaskMsg{
			ID:    id,
			Name:  name,
			Total: total,
		})
	}
}

// UpdateTask updates a task's progress
func (p *Program) UpdateTask(id string, progress float64, message string) {
	if p.program != nil {
		p.program.Send(UpdateTaskMsg{
			ID:       id,
			Progress: progress,
			Message:  message,
			Status:   TaskStatusRunning,
		})
	}
}

// CompleteTask marks a task as completed
func (p *Program) CompleteTask(id string) {
	if p.program != nil {
		p.program.Send(UpdateTaskMsg{
			ID:       id,
			Progress: 100,
			Message:  "Completed",
			Status:   TaskStatusCompleted,
		})
	}
}

// FailTask marks a task as failed
func (p *Program) FailTask(id string, err error) {
	if p.program != nil {
		p.program.Send(UpdateTaskMsg{
			ID:       id,
			Progress: 0,
			Message:  err.Error(),
			Status:   TaskStatusFailed,
		})
	}
}

// Complete marks the entire operation as complete
func (p *Program) Complete() {
	if p.program != nil {
		p.program.Send(CompleteMsg{})
	}
}

// Log adds a log entry
func (p *Program) Log(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	if p.program != nil {
		p.program.Send(AddLogMsg{
			Level:   level,
			Message: message,
		})
	}
}

// logCapture captures log output and sends it to the program
type logCapture struct {
	program        *tea.Program
	originalOut    io.Writer
	buffer         []logEntry
	mu             sync.Mutex
	stderrPipe     *os.File
	originalStderr *os.File
	stderrReader   *os.File
	stderrStop     chan struct{}
}

type logEntry struct {
	timestamp time.Time
	message   string
}

// Start begins capturing log output
func (lc *logCapture) Start() {
	lc.originalOut = log.Writer()
	log.SetOutput(lc)
	
	// Set environment variables to suppress k3d logs
	os.Setenv("K3D_LOG_LEVEL", "panic")
	os.Setenv("DOCKER_CLI_HINTS", "false")
	
	// Redirect klog output to our writer
	klog.SetOutput(lc)
}

// Stop stops capturing and restores original output
func (lc *logCapture) Stop() {
	if lc.originalOut != nil {
		log.SetOutput(lc.originalOut)
		// Flush klog before restoring
		klog.Flush()
		// Also restore klog to stderr
		klog.SetOutput(os.Stderr)
	}
}

// StartStderrCapture starts capturing stderr output
func (lc *logCapture) StartStderrCapture() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	
	// Create a pipe to capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		return
	}
	
	lc.originalStderr = os.Stderr
	lc.stderrReader = r
	lc.stderrPipe = w
	lc.stderrStop = make(chan struct{})
	
	// Redirect stderr to our pipe
	os.Stderr = w
	
	// Start a goroutine to read from the pipe
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-lc.stderrStop:
				return
			default:
				n, err := r.Read(buf)
				if err != nil {
					return
				}
				if n > 0 && lc.program != nil {
					// Send stderr output as log message
					message := strings.TrimRight(string(buf[:n]), "\n\r")
					if message != "" {
						lc.program.Send(AddLogMsg{
							Level:   "INFO",
							Message: message,
						})
					}
				}
			}
		}
	}()
}

// StopStderrCapture stops capturing stderr and restores original
func (lc *logCapture) StopStderrCapture() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	
	if lc.stderrStop != nil {
		close(lc.stderrStop)
	}
	
	if lc.originalStderr != nil {
		os.Stderr = lc.originalStderr
		lc.originalStderr = nil
	}
	
	if lc.stderrPipe != nil {
		lc.stderrPipe.Close()
		lc.stderrPipe = nil
	}
	
	if lc.stderrReader != nil {
		lc.stderrReader.Close()
		lc.stderrReader = nil
	}
}

// Write implements io.Writer
func (lc *logCapture) Write(p []byte) (n int, err error) {
	message := string(p)
	
	// Send to the program as a log message
	if lc.program != nil {
		// Determine log level from content
		level := "INFO"
		
		// Check for klog format (I0717, E0717, W0717, etc.)
		if len(message) > 0 {
			switch message[0] {
			case 'I':
				level = "INFO"
			case 'E':
				level = "ERROR"
			case 'W':
				level = "WARN"
			case 'F':
				level = "FATAL"
			default:
				// Fallback to content-based detection
				if contains(message, "error", "ERROR", "Error") {
					level = "ERROR"
				} else if contains(message, "warn", "WARN", "Warn", "warning", "WARNING") {
					level = "WARN"
				} else if contains(message, "debug", "DEBUG", "Debug") {
					level = "DEBUG"
				}
			}
		}
		
		lc.program.Send(AddLogMsg{
			Level:   level,
			Message: message,
		})
	}
	
	// Also write to original output for debugging
	if lc.originalOut != nil && os.Getenv("KECS_DEBUG") != "" {
		lc.originalOut.Write(p)
	}
	
	return len(p), nil
}

// contains checks if a string contains any of the given substrings
func contains(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			// Simple substring check
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// Helper functions for common operations

// RunWithProgress runs a function with progress tracking
func RunWithProgress(title string, fn func(*Program) error) error {
	prog := NewProgram(title)
	
	if err := prog.Start(); err != nil {
		return fmt.Errorf("failed to start progress display: %w", err)
	}
	defer prog.Stop()
	
	// Run the function
	if err := fn(prog); err != nil {
		return err
	}
	
	// Mark as complete
	prog.Complete()
	
	// Give user a moment to see the final state
	time.Sleep(1 * time.Second)
	
	return nil
}