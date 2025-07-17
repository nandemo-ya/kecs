package progress

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogCapture(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelInfo)
	
	require.NotNil(t, lc)
	assert.Equal(t, LogLevelInfo, lc.minLevel)
	assert.Equal(t, &buf, lc.output)
	assert.Empty(t, lc.entries)
}

func TestLogCapture_Log(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelInfo)
	
	// Test info log (should be buffered)
	lc.Log(LogLevelInfo, "Test info message")
	assert.Len(t, lc.entries, 1)
	assert.Equal(t, "Test info message", lc.entries[0].Message)
	assert.Equal(t, LogLevelInfo, lc.entries[0].Level)
	assert.Empty(t, buf.String()) // Info logs are not immediate
	
	// Test warning log (should be immediate)
	buf.Reset()
	lc.Log(LogLevelWarning, "Test warning message")
	assert.Len(t, lc.entries, 2)
	assert.Contains(t, buf.String(), "Test warning message")
	assert.Contains(t, buf.String(), "⚠️  WARN:")
	
	// Test error log (should be immediate)
	buf.Reset()
	lc.Log(LogLevelError, "Test error message")
	assert.Len(t, lc.entries, 3)
	assert.Contains(t, buf.String(), "Test error message")
	assert.Contains(t, buf.String(), "❌ ERROR:")
}

func TestLogCapture_Flush(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelInfo)
	
	// Add various log levels
	lc.Log(LogLevelDebug, "Debug message")
	lc.Log(LogLevelInfo, "Info message 1")
	lc.Log(LogLevelInfo, "Info message 2")
	lc.Log(LogLevelWarning, "Warning message") // This one outputs immediately
	
	// Clear buffer from warning output
	buf.Reset()
	
	// Flush should output info logs but not debug (min level is Info)
	lc.Flush()
	
	output := buf.String()
	assert.Contains(t, output, "📋 Operation Logs:")
	assert.Contains(t, output, "Info message 1")
	assert.Contains(t, output, "Info message 2")
	assert.NotContains(t, output, "Debug message") // Below min level
	assert.NotContains(t, output, "Warning message") // Already output
	
	// After flush, entries should be cleared
	assert.Empty(t, lc.entries)
}

func TestLogCapture_FlushWithDebugLevel(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelDebug)
	
	// Add debug logs
	lc.Log(LogLevelDebug, "Debug message 1")
	lc.Log(LogLevelDebug, "Debug message 2")
	lc.Log(LogLevelInfo, "Info message")
	
	lc.Flush()
	
	output := buf.String()
	assert.Contains(t, output, "Debug Information:")
	assert.Contains(t, output, "Debug message 1")
	assert.Contains(t, output, "Debug message 2")
	assert.Contains(t, output, "Info message")
}

func TestLogCaptureWriter(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelInfo)
	
	// Get a writer
	writer := lc.LogWriter(LogLevelInfo)
	
	// Write multiple lines
	_, err := writer.Write([]byte("Line 1\nLine 2\n"))
	require.NoError(t, err)
	
	assert.Len(t, lc.entries, 2)
	assert.Equal(t, "Line 1", lc.entries[0].Message)
	assert.Equal(t, "Line 2", lc.entries[1].Message)
	
	// Write partial line
	_, err = writer.Write([]byte("Partial"))
	require.NoError(t, err)
	assert.Len(t, lc.entries, 2) // No new entry yet
	
	// Complete the line
	_, err = writer.Write([]byte(" line\n"))
	require.NoError(t, err)
	assert.Len(t, lc.entries, 3)
	assert.Equal(t, "Partial line", lc.entries[2].Message)
}

func TestLogCaptureWriter_EmptyLines(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelInfo)
	writer := lc.LogWriter(LogLevelInfo)
	
	// Write with empty lines
	_, err := writer.Write([]byte("Line 1\n\n\nLine 2\n"))
	require.NoError(t, err)
	
	// Empty lines should be skipped
	assert.Len(t, lc.entries, 2)
	assert.Equal(t, "Line 1", lc.entries[0].Message)
	assert.Equal(t, "Line 2", lc.entries[1].Message)
}

func TestLogRedirector(t *testing.T) {
	var buf bytes.Buffer
	lc := NewLogCapture(&buf, LogLevelInfo)
	lr := NewLogRedirector(lc, LogLevelInfo)
	
	// Test that it doesn't panic
	lr.RedirectStandardLog()
	lr.Restore()
}