package bubbletea

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
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
	// Create silent writers for stdout/stderr
	originalLogWriter := log.Writer()
	silentLogWriter := newSilentWriter(originalLogWriter)
	
	// Save original stdout and stderr
	originalStdout := os.Stdout
	originalStderr := os.Stderr
	
	// Create pipes to capture stdout and stderr
	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()
	os.Stdout = stdoutW
	os.Stderr = stderrW
	
	// Restore stdout/stderr on exit
	defer func() {
		os.Stdout = originalStdout
		os.Stderr = originalStderr
		stdoutW.Close()
		stderrW.Close()
	}()
	
	// Channel to signal when adapter is ready
	adapterReady := make(chan *Adapter, 1)
	
	// Read from pipes in background and send to adapter
	captureOutput := func(r io.Reader, prefix string) {
		// Wait for adapter to be ready
		var adapter *Adapter
		select {
		case adapter = <-adapterReady:
			// Put it back for the other goroutine
			adapterReady <- adapter
		case <-time.After(5 * time.Second):
			// Timeout - just discard output
			io.Copy(io.Discard, r)
			return
		}
		
		scanner := bufio.NewScanner(r)
		// Increase buffer size for long lines
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		
		for scanner.Scan() {
			line := scanner.Text()
			if adapter != nil && adapter.program != nil {
				// Skip empty lines
				if strings.TrimSpace(line) == "" {
					continue
				}
				
				// Parse logrus-style logs (INFO[0001] message)
				level := progress.LogLevelInfo
				if strings.Contains(line, "INFO[") || strings.Contains(line, "info[") {
					level = progress.LogLevelInfo
				} else if strings.Contains(line, "ERROR[") || strings.Contains(line, "ERRO[") {
					level = progress.LogLevelError
				} else if strings.Contains(line, "WARN[") || strings.Contains(line, "warning[") {
					level = progress.LogLevelWarning  
				} else if strings.Contains(line, "DEBUG[") || strings.Contains(line, "DEBU[") {
					level = progress.LogLevelDebug
				}
				
				// Clean up the line - remove ANSI codes if present
				cleanLine := stripANSI(line)
				adapter.Log(level, "%s", cleanLine)
			}
		}
	}
	
	go captureOutput(stdoutR, "stdout")
	go captureOutput(stderrR, "stderr")
	
	// Temporarily redirect all log output
	log.SetOutput(silentLogWriter)
	defer log.SetOutput(originalLogWriter)
	
	// Create and start the program
	adapter := NewAdapter(title)
	
	// Send adapter to waiting goroutines
	adapterReady <- adapter
	
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
			// Restore original stdout/stderr first
			os.Stdout = originalStdout
			os.Stderr = originalStderr
			stdoutW.Close()
			stderrW.Close()
			
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

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Simple approach - remove common ANSI codes
	// This covers most color and style codes
	var result strings.Builder
	i := 0
	for i < len(s) {
		if i+1 < len(s) && s[i] == '\x1b' && s[i+1] == '[' {
			// Skip until we find the end of the escape sequence
			j := i + 2
			for j < len(s) && !isANSITerminator(s[j]) {
				j++
			}
			if j < len(s) {
				j++ // Skip the terminator
			}
			i = j
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

func isANSITerminator(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

