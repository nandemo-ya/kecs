package bubbletea

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/sirupsen/logrus"
	"k8s.io/klog/v2"
)

// silentWriter is a writer that discards all output until activated
type silentWriter struct {
	mu        sync.Mutex
	active    bool
	original  io.Writer
	buffer    []byte
	onWrite   func([]byte)
}

func newSilentWriter(original io.Writer) *silentWriter {
	return &silentWriter{
		original: original,
		buffer:   make([]byte, 0),
	}
}

func (sw *silentWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	
	if sw.onWrite != nil {
		sw.onWrite(p)
	}
	
	if sw.active {
		return sw.original.Write(p)
	}
	
	// Buffer the output while silent
	sw.buffer = append(sw.buffer, p...)
	return len(p), nil
}

func (sw *silentWriter) activate() {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.active = true
	
	// Flush buffered content
	if len(sw.buffer) > 0 && sw.original != nil {
		sw.original.Write(sw.buffer)
		sw.buffer = nil
	}
}

// RunWithBubbleTeaSilent runs a function with Bubble Tea progress, ensuring no output before TUI starts
func RunWithBubbleTeaSilent(ctx context.Context, title string, fn func(*Adapter) error) error {
	// Create silent writers for log output
	originalLogWriter := log.Writer()
	silentLogWriter := newSilentWriter(originalLogWriter)
	
	// Temporarily redirect all log output
	log.SetOutput(silentLogWriter)
	defer log.SetOutput(originalLogWriter)
	
	// Also redirect klog immediately
	klog.SetOutput(silentLogWriter)
	defer func() {
		klog.Flush()
		klog.SetOutput(os.Stderr)
	}()
	
	// Set environment to suppress logrus output from k3d
	// This is needed because k3d uses logrus internally
	os.Setenv("LOGRUS_LEVEL", "panic")
	defer os.Unsetenv("LOGRUS_LEVEL")
	
	// Also configure logrus directly
	originalLogrusOut := logrus.StandardLogger().Out
	logrus.SetOutput(silentLogWriter)
	logrus.SetLevel(logrus.PanicLevel)
	defer func() {
		logrus.SetOutput(originalLogrusOut)
		logrus.SetLevel(logrus.InfoLevel)
	}()
	
	// Create and start the program
	adapter := NewAdapter(title)
	
	// Initialize slog with the silent writer
	logging.Initialize(&logging.Config{
		Level:           logging.ParseLevel("INFO"),
		Format:          "text",
		Output:          silentLogWriter,
		UseCustomFormat: true,
	})
	
	// Set up the silent writer to send logs to the adapter once it's ready
	silentLogWriter.onWrite = func(p []byte) {
		if adapter != nil && adapter.program != nil {
			// Send to Bubble Tea as a log message
			adapter.Log(progress.LogLevelInfo, "%s", string(p))
		}
	}
	
	// Start the adapter
	if err := adapter.Start(); err != nil {
		// Restore logging before returning error
		log.SetOutput(originalLogWriter)
		// If it's a TTY error, fall back to non-TUI mode
		if contains(err.Error(), "TTY", "tty") {
			// Print title
			fmt.Println(title)
			fmt.Println()
			
			// Create adapter without TUI
			adapter = NewAdapter(title)
			// Just run the function without TUI - the adapter will handle nil program
			return fn(adapter)
		}
		return fmt.Errorf("failed to start progress display: %w", err)
	}
	defer adapter.Stop()
	
	// Now that Bubble Tea is running, we can activate the log writer
	// But we don't want to flush the buffer as it will mess up the display
	// silentLogWriter.activate()
	
	// Run the function
	if err := fn(adapter); err != nil {
		return err
	}
	
	// Mark as complete
	adapter.program.Complete()
	
	return nil
}


