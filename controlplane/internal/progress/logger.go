package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

// LogEntry represents a captured log entry
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

// LogCapture provides a simpler approach to capturing logs during progress display
type LogCapture struct {
	entries  []LogEntry
	mu       sync.Mutex
	minLevel LogLevel
	output   io.Writer // Where to write immediate logs
}

// NewLogCapture creates a new log capture instance
func NewLogCapture(output io.Writer, minLevel LogLevel) *LogCapture {
	return &LogCapture{
		entries:  make([]LogEntry, 0),
		minLevel: minLevel,
		output:   output,
	}
}

// Add captures a log entry with timestamp, level and message
func (lc *LogCapture) Add(timestamp time.Time, level LogLevel, message string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	entry := LogEntry{
		Timestamp: timestamp,
		Level:     level,
		Message:   message,
	}

	// Only capture if it meets minimum level
	if level >= lc.minLevel {
		lc.entries = append(lc.entries, entry)
		
		// Also write immediately if we're at warning or error level
		if level >= LogLevelWarning {
			lc.writeEntry(entry)
		}
	}
}

// Log captures a log message
func (lc *LogCapture) Log(level LogLevel, format string, args ...interface{}) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	message := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   strings.TrimSpace(message),
	}

	lc.entries = append(lc.entries, entry)

	// Immediately output warnings and errors
	if level >= LogLevelWarning && lc.output != nil {
		lc.writeEntry(entry)
	}
}

// writeEntry writes a single log entry
func (lc *LogCapture) writeEntry(entry LogEntry) {
	prefix := ""
	switch entry.Level {
	case LogLevelError:
		prefix = "❌ ERROR: "
	case LogLevelWarning:
		prefix = "⚠️  WARN: "
	case LogLevelInfo:
		prefix = "ℹ️  INFO: "
	case LogLevelDebug:
		prefix = "🔍 DEBUG: "
	}
	
	fmt.Fprintf(lc.output, "\n%s%s\n", prefix, entry.Message)
}

// Flush outputs all buffered logs that meet the minimum level
func (lc *LogCapture) Flush() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if len(lc.entries) == 0 || lc.output == nil {
		return
	}

	// Group logs by level
	var debugLogs, infoLogs []LogEntry

	for _, entry := range lc.entries {
		if entry.Level < lc.minLevel {
			continue
		}

		switch entry.Level {
		case LogLevelDebug:
			debugLogs = append(debugLogs, entry)
		case LogLevelInfo:
			infoLogs = append(infoLogs, entry)
		default:
			// Warnings and errors were already output immediately
			continue
		}
	}

	// Only show log section if there are logs to display
	if len(debugLogs) > 0 || len(infoLogs) > 0 {
		fmt.Fprintln(lc.output, "\n📋 Operation Logs:")
		fmt.Fprintln(lc.output, "─────────────────")

		// Output info logs
		for _, entry := range infoLogs {
			fmt.Fprintf(lc.output, "  • %s\n", entry.Message)
		}

		// Output debug logs if verbose
		if lc.minLevel <= LogLevelDebug && len(debugLogs) > 0 {
			fmt.Fprintln(lc.output, "\n  Debug Information:")
			for _, entry := range debugLogs {
				fmt.Fprintf(lc.output, "    - %s\n", entry.Message)
			}
		}

		fmt.Fprintln(lc.output)
	}

	// Clear entries after flushing
	lc.entries = lc.entries[:0]
}

// Clear removes all buffered logs without outputting them
func (lc *LogCapture) Clear() {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.entries = lc.entries[:0]
}

// LogWriter returns an io.Writer that captures logs at the specified level
func (lc *LogCapture) LogWriter(level LogLevel) io.Writer {
	return &logCaptureWriter{
		capture: lc,
		level:   level,
	}
}

// logCaptureWriter implements io.Writer for LogCapture
type logCaptureWriter struct {
	capture *LogCapture
	level   LogLevel
	buffer  []byte
}

func (w *logCaptureWriter) Write(p []byte) (n int, err error) {
	// Append to buffer
	w.buffer = append(w.buffer, p...)
	
	// Look for newlines to create log entries
	for {
		idx := -1
		for i, b := range w.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}
		
		if idx == -1 {
			break
		}
		
		// Extract the line
		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]
		
		// Skip empty lines
		if strings.TrimSpace(line) != "" {
			w.capture.Log(w.level, "%s", line)
		}
	}
	
	return len(p), nil
}