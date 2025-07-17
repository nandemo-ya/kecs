package progress

import (
	"io"
	"log"
	"os"
)

// LogRedirector helps redirect standard logging to a LogCapture
type LogRedirector struct {
	originalWriter io.Writer
	capture        *LogCapture
	level          LogLevel
}

// NewLogRedirector creates a new log redirector
func NewLogRedirector(capture *LogCapture, level LogLevel) *LogRedirector {
	return &LogRedirector{
		capture: capture,
		level:   level,
	}
}

// RedirectStandardLog redirects the standard log package output
func (lr *LogRedirector) RedirectStandardLog() {
	lr.originalWriter = log.Writer()
	log.SetOutput(lr.capture.LogWriter(lr.level))
}

// Restore restores the original log output
func (lr *LogRedirector) Restore() {
	if lr.originalWriter != nil {
		log.SetOutput(lr.originalWriter)
	}
}

// CaptureCommand creates a command with output redirected to log capture
type CaptureCommand struct {
	logCapture *LogCapture
	stdout     io.Writer
	stderr     io.Writer
}

// NewCaptureCommand creates a wrapper for capturing command output
func NewCaptureCommand(logCapture *LogCapture) *CaptureCommand {
	return &CaptureCommand{
		logCapture: logCapture,
		stdout:     logCapture.LogWriter(LogLevelInfo),
		stderr:     logCapture.LogWriter(LogLevelError),
	}
}

// Stdout returns a writer for stdout that captures to info level
func (cc *CaptureCommand) Stdout() io.Writer {
	return cc.stdout
}

// Stderr returns a writer for stderr that captures to error level
func (cc *CaptureCommand) Stderr() io.Writer {
	return cc.stderr
}

// OverrideGlobalLogger temporarily overrides os.Stdout and os.Stderr
// This is useful for capturing output from third-party libraries
// WARNING: This should be used with caution as it affects global state
type GlobalLogOverride struct {
	originalStdout *os.File
	originalStderr *os.File
	stdoutPipe     *os.File
	stderrPipe     *os.File
	capture        *LogCapture
}

// NewGlobalLogOverride creates a global log override
// This is a more aggressive approach that redirects os.Stdout/Stderr
func NewGlobalLogOverride(capture *LogCapture) *GlobalLogOverride {
	return &GlobalLogOverride{
		capture:        capture,
		originalStdout: os.Stdout,
		originalStderr: os.Stderr,
	}
}

// Note: Global redirection is complex and may not work reliably in all cases
// For now, we'll keep it simple and focus on redirecting log package output
// and providing writers for commands