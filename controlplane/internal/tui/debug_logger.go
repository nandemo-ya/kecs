package tui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DebugLogger is a singleton logger for TUI debugging
type DebugLogger struct {
	logger *log.Logger
	file   *os.File
	mu     sync.Mutex
}

var (
	debugLogger     *DebugLogger
	debugLoggerOnce sync.Once
)

// GetDebugLogger returns the singleton debug logger instance
func GetDebugLogger() *DebugLogger {
	debugLoggerOnce.Do(func() {
		debugLogger = initDebugLogger()
	})
	return debugLogger
}

// initDebugLogger initializes the debug logger
func initDebugLogger() *DebugLogger {
	// Create ~/.kecs directory if it doesn't exist
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	kecsDir := filepath.Join(homeDir, ".kecs")
	if err := os.MkdirAll(kecsDir, 0755); err != nil {
		return nil
	}

	// Open or create the debug log file
	logPath := filepath.Join(kecsDir, "tui-debug.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil
	}

	return &DebugLogger{
		logger: log.New(file, "", log.LstdFlags|log.Lmicroseconds),
		file:   file,
	}
}

// Log writes a debug message to the log file
func (d *DebugLogger) Log(format string, args ...interface{}) {
	if d == nil || d.logger == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	d.logger.Println(msg)
}

// LogWithCaller writes a debug message with caller information
func (d *DebugLogger) LogWithCaller(caller string, format string, args ...interface{}) {
	if d == nil || d.logger == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	msg := fmt.Sprintf("[%s] %s", caller, fmt.Sprintf(format, args...))
	d.logger.Println(msg)
}

// Close closes the debug log file
func (d *DebugLogger) Close() {
	if d == nil || d.file == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.file.Close()
}

// StartSession writes a session start marker to the log
func (d *DebugLogger) StartSession() {
	if d == nil || d.logger == nil {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	d.logger.Println("========================================")
	d.logger.Printf("TUI Session Started at %s", time.Now().Format(time.RFC3339))
	d.logger.Println("========================================")
}
