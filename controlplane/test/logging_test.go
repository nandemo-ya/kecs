package test

import (
	"os"
	"testing"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
)

// TestLoggingVerboseMode tests logging in verbose mode
func TestLoggingVerboseMode(t *testing.T) {
	// Initialize logging for verbose mode
	logging.InitializeForProgress(nil, true)

	// Test different log levels
	logging.Debug("This is a debug message", "key", "value")
	logging.Info("This is an info message", "port", 8080)
	logging.Warn("This is a warning message", "error", "something went wrong")
	logging.Error("This is an error message", "error", "critical failure")

	// Test with component
	logger := logging.Component("api-server")
	logger.Info("Server started", "address", "localhost:8080")
}

// TestLoggingWithProgress tests logging with progress capture
func TestLoggingWithProgress(t *testing.T) {
	// Create log capture
	logCapture := progress.NewLogCapture(os.Stdout, progress.LogLevelInfo)

	// Initialize logging with progress capture
	logging.InitializeForProgress(logCapture, false)

	// Test logging
	logging.Info("Starting process...")
	logging.Debug("Debug message (should be captured)")
	logging.Warn("Warning message", "code", 404)
	logging.Error("Error occurred", "error", "test error")

	// Give time for logs to be captured
	time.Sleep(100 * time.Millisecond)

	// Flush logs
	logCapture.Flush()
}

// TestLoggingFormats tests different log formats
func TestLoggingFormats(t *testing.T) {
	t.Run("Custom Text Format", func(t *testing.T) {
		logging.Initialize(&logging.Config{
			Level:           logging.ParseLevel("DEBUG"),
			Format:          "text",
			Output:          os.Stdout,
			UseCustomFormat: true,
		})

		logging.Info("Custom format test", "component", "test", "version", "1.0")
	})

	t.Run("JSON Format", func(t *testing.T) {
		logging.Initialize(&logging.Config{
			Level:  logging.ParseLevel("INFO"),
			Format: "json",
			Output: os.Stdout,
		})

		logging.Info("JSON format test", "component", "test", "version", "1.0")
	})
}
