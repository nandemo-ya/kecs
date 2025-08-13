package logging

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
)

// ProgressWriter is an io.Writer that integrates with the progress display system
type ProgressWriter struct {
	mu       sync.Mutex
	capture  *progress.LogCapture
	fallback io.Writer
}

// NewProgressWriter creates a new writer that integrates with the progress system
func NewProgressWriter(fallback io.Writer) *ProgressWriter {
	return &ProgressWriter{
		fallback: fallback,
	}
}

// SetLogCapture sets the log capture for progress integration
func (pw *ProgressWriter) SetLogCapture(capture *progress.LogCapture) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	pw.capture = capture
}

// Write implements io.Writer
func (pw *ProgressWriter) Write(p []byte) (n int, err error) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	// If no capture is set, write to fallback
	if pw.capture == nil {
		return pw.fallback.Write(p)
	}

	// Parse the log level from slog output
	// slog text format: "time=2025-07-21T15:04:05.000+09:00 level=INFO msg=\"message\" key=value"
	// slog json format: {"time":"2025-07-21T15:04:05.000+09:00","level":"INFO","msg":"message","key":"value"}
	level := pw.parseLogLevel(p)

	// Convert bytes to string and trim whitespace
	message := strings.TrimSpace(string(p))

	// Add to capture with parsed level
	pw.capture.Add(time.Now(), level, message)

	return len(p), nil
}

// parseLogLevel extracts the log level from slog output
func (pw *ProgressWriter) parseLogLevel(p []byte) progress.LogLevel {
	s := string(p)

	// Check for text format
	if strings.Contains(s, "level=") {
		if strings.Contains(s, "level=DEBUG") {
			return progress.LogLevelDebug
		} else if strings.Contains(s, "level=INFO") {
			return progress.LogLevelInfo
		} else if strings.Contains(s, "level=WARN") {
			return progress.LogLevelWarning
		} else if strings.Contains(s, "level=ERROR") {
			return progress.LogLevelError
		}
	}

	// Check for JSON format
	if strings.Contains(s, `"level":`) {
		if strings.Contains(s, `"level":"DEBUG"`) {
			return progress.LogLevelDebug
		} else if strings.Contains(s, `"level":"INFO"`) {
			return progress.LogLevelInfo
		} else if strings.Contains(s, `"level":"WARN"`) {
			return progress.LogLevelWarning
		} else if strings.Contains(s, `"level":"ERROR"`) {
			return progress.LogLevelError
		}
	}

	// Default to info if we can't determine
	return progress.LogLevelInfo
}

// CustomTextHandler wraps slog.TextHandler to format output for progress display
type CustomTextHandler struct {
	slog.Handler
	writer io.Writer
}

// NewCustomTextHandler creates a handler that formats logs for progress display
func NewCustomTextHandler(w io.Writer, opts *slog.HandlerOptions) *CustomTextHandler {
	return &CustomTextHandler{
		Handler: slog.NewTextHandler(w, opts),
		writer:  w,
	}
}

// Handle formats the record and writes it
func (h *CustomTextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Format: "HH:MM:SS LEVEL  message [attrs]"
	var buf bytes.Buffer

	// Time
	fmt.Fprintf(&buf, "%s ", r.Time.Format("15:04:05"))

	// Level (padded to 5 chars)
	level := r.Level.String()
	fmt.Fprintf(&buf, "%-5s  ", level)

	// Message
	buf.WriteString(r.Message)

	// Attributes
	if r.NumAttrs() > 0 {
		buf.WriteString(" ")
		r.Attrs(func(a slog.Attr) bool {
			fmt.Fprintf(&buf, "%s=%v ", a.Key, a.Value)
			return true
		})
	}

	buf.WriteByte('\n')

	_, err := h.writer.Write(buf.Bytes())
	return err
}
