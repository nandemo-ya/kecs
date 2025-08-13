package logging

import (
	"io"
	"log/slog"
	"os"

	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
)

// Config holds logging configuration
type Config struct {
	// Level is the minimum log level
	Level slog.Level

	// Format is the output format (text or json)
	Format string

	// Output is the output writer
	Output io.Writer

	// ProgressCapture is the progress system log capture (optional)
	ProgressCapture *progress.LogCapture

	// UseCustomFormat uses our custom format for progress display
	UseCustomFormat bool
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:           slog.LevelInfo,
		Format:          "text",
		Output:          os.Stderr,
		UseCustomFormat: true,
	}
}

// Initialize initializes the global logger with the given configuration
func Initialize(cfg *Config) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Create the output writer
	var output io.Writer = cfg.Output

	// If progress capture is provided, wrap the output
	if cfg.ProgressCapture != nil {
		pw := NewProgressWriter(cfg.Output)
		pw.SetLogCapture(cfg.ProgressCapture)
		output = pw
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: cfg.Level,
	}

	// Create the logger based on format
	var logger *slog.Logger
	switch cfg.Format {
	case "json":
		logger = slog.New(slog.NewJSONHandler(output, opts))
	default:
		if cfg.UseCustomFormat {
			logger = slog.New(NewCustomTextHandler(output, opts))
		} else {
			logger = slog.New(slog.NewTextHandler(output, opts))
		}
	}

	// Set as global logger
	SetGlobalLogger(logger)
}

// InitializeForProgress initializes logging for use with progress display
func InitializeForProgress(capture *progress.LogCapture, verbose bool) {
	cfg := DefaultConfig()

	if verbose {
		// In verbose mode, log directly to stderr
		cfg.ProgressCapture = nil
		cfg.UseCustomFormat = true
	} else {
		// In progress mode, capture logs
		cfg.ProgressCapture = capture
		cfg.UseCustomFormat = true
	}

	Initialize(cfg)
}

// SetLevel sets the global log level
func SetLevel(level slog.Level) {
	cfg := DefaultConfig()
	cfg.Level = level
	Initialize(cfg)
}

// ParseLevel parses a string log level
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug", "DEBUG":
		return slog.LevelDebug
	case "info", "INFO":
		return slog.LevelInfo
	case "warn", "WARN", "warning", "WARNING":
		return slog.LevelWarn
	case "error", "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
