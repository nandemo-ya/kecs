// Package logging provides a unified logging interface for KECS using slog.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	// global logger instance
	globalLogger *slog.Logger
	globalMu     sync.RWMutex
	
	// default text handler options
	defaultTextOptions = &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
)

func init() {
	// Initialize with a default text handler
	globalLogger = slog.New(slog.NewTextHandler(os.Stderr, defaultTextOptions))
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger *slog.Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// Debug logs a debug message
func Debug(msg string, args ...any) {
	GetGlobalLogger().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	GetGlobalLogger().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	GetGlobalLogger().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	GetGlobalLogger().Error(msg, args...)
}

// With returns a logger with the given attributes
func With(args ...any) *slog.Logger {
	return GetGlobalLogger().With(args...)
}

// WithContext returns a logger with context
func WithContext(ctx context.Context) *slog.Logger {
	return GetGlobalLogger()
}

// NewLogger creates a new logger with the specified writer and options
func NewLogger(w io.Writer, opts *slog.HandlerOptions) *slog.Logger {
	if opts == nil {
		opts = defaultTextOptions
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// NewJSONLogger creates a new JSON format logger
func NewJSONLogger(w io.Writer, opts *slog.HandlerOptions) *slog.Logger {
	if opts == nil {
		opts = defaultTextOptions
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}

// Component returns a logger with a component field
func Component(name string) *slog.Logger {
	return With("component", name)
}

// Operation returns a logger with an operation field
func Operation(name string) *slog.Logger {
	return With("operation", name)
}

// SetOutput sets the output writer for the global logger.
// This is useful for redirecting or discarding log output.
func SetOutput(w io.Writer) {
	globalMu.Lock()
	defer globalMu.Unlock()
	
	// Get current handler options from the existing logger if possible
	opts := defaultTextOptions
	
	// Create new logger with the specified writer
	globalLogger = slog.New(slog.NewTextHandler(w, opts))
}