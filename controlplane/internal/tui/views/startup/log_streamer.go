// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package startup

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

// LogStreamer handles streaming logs from KECS startup
type LogStreamer struct {
	instanceName string
	apiPort      int
	manager      *instance.Manager
	cancel       context.CancelFunc
}

// NewLogStreamer creates a new log streamer
func NewLogStreamer(instanceName string, apiPort int) (*LogStreamer, error) {
	manager, err := instance.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create instance manager: %w", err)
	}
	
	return &LogStreamer{
		instanceName: instanceName,
		apiPort:      apiPort,
		manager:      manager,
	}, nil
}

// Stop stops the KECS startup process
func (s *LogStreamer) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}


// StartKECSWithStreamer starts KECS with a log streamer
func StartKECSWithStreamer(instanceName string, apiPort int) tea.Cmd {
	return func() tea.Msg {
		// Create message channel
		msgChan := make(chan tea.Msg, 100)
		
		// Start the instance manager in a goroutine
		go func() {
			defer close(msgChan)
			
			// Create instance manager
			manager, err := instance.NewManager()
			if err != nil {
				msgChan <- startupErrorMsg{err: fmt.Errorf("failed to create instance manager: %w", err)}
				return
			}
			
			// Set up logging to redirect to TUI
			logWriter := &tuiLogWriter{msgChan: msgChan}
			logCapture := progress.NewLogCapture(logWriter, progress.LogLevelInfo)
			logging.InitializeForProgress(logCapture, false)
			
			// Use default instance name if empty
			instanceNameToUse := instanceName
			if instanceNameToUse == "" {
				instanceNameToUse = "default"
			}
			
			// Start options
			opts := instance.StartOptions{
				InstanceName: instanceNameToUse,
				ApiPort:      apiPort,
				AdminPort:    8081, // Default admin port
			}
			
			// Context for cancellation
			ctx := context.Background()
			
			// Start the instance
			msgChan <- startupLogMsg{line: fmt.Sprintf("Starting KECS instance '%s'...", instanceNameToUse)}
			if err := manager.Start(ctx, opts); err != nil {
				msgChan <- startupErrorMsg{err: err}
				return
			}
			
			// Check readiness
			go monitorReadiness(instanceNameToUse, apiPort, msgChan)
		}()
		
		// Return a batch command that reads from the channel
		return tea.Batch(
			func() tea.Msg {
				return startupProgressMsg{message: "Starting KECS..."}
			},
			readFromChannel(msgChan),
		)
	}
}

// readFromChannel creates a command that reads messages from a channel
func readFromChannel(msgChan <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-msgChan
		if !ok {
			return nil
		}
		return tea.Batch(
			func() tea.Msg { return msg },
			readFromChannel(msgChan), // Continue reading
		)
	}
}

// monitorReadiness monitors KECS readiness
func monitorReadiness(instanceName string, apiPort int, msgChan chan<- tea.Msg) {
	// Check for readiness
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(2 * time.Minute)
	retries := 0
	maxRetries := 60
	
	for {
		select {
		case <-ticker.C:
			// Check if KECS is ready
			retries++
			if isKECSReadyForPort(instanceName, apiPort) {
				msgChan <- startupCompleteMsg{}
				return
			}
			
			if retries%5 == 0 {
				msgChan <- startupProgressMsg{
					message: fmt.Sprintf("Waiting for API to be ready... (%ds)", retries*2),
				}
			}
			
			if retries >= maxRetries {
				msgChan <- startupErrorMsg{
					err: fmt.Errorf("timeout waiting for KECS API to be ready"),
				}
				return
			}
			
		case <-timeout:
			msgChan <- startupErrorMsg{
				err: fmt.Errorf("timeout waiting for KECS API to be ready"),
			}
			return
		}
	}
}

// isKECSReadyForPort checks if KECS is ready at the specified port
func isKECSReadyForPort(instanceName string, apiPort int) bool {
	if apiPort == 0 {
		apiPort = 8080
	}
	
	endpoint := fmt.Sprintf("http://localhost:%d", apiPort)
	client := api.NewClient(endpoint)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	_, err := client.ListClusters(ctx)
	return err == nil
}

// Helper functions

// shouldDisplayLog determines if a log line should be displayed
func shouldDisplayLog(line string) bool {
	// Filter out empty lines and whitespace
	trimmed := strings.TrimSpace(line)
	return trimmed != ""
}

// formatLogLine formats a log line for display
func formatLogLine(line string) string {
	// Clean and format the line
	line = strings.TrimSpace(line)
	
	// Remove timestamp and log level if present
	// Pattern: 2025-01-28T12:00:00Z INFO Starting server -> Starting server
	parts := strings.SplitN(line, " ", 3)
	if len(parts) >= 3 && strings.Contains(parts[0], "T") && strings.Contains(parts[0], "Z") {
		// Looks like timestamp + level + message
		return parts[2]
	}
	
	return line
}

// extractProgress extracts progress information from log lines
func extractProgress(line string) string {
	// Look for progress indicators in the log
	lower := strings.ToLower(line)
	
	if strings.Contains(lower, "creating") && strings.Contains(lower, "kubernetes cluster") {
		return "Creating Kubernetes cluster..."
	}
	if strings.Contains(line, "Waiting for cluster to be ready") {
		return "Waiting for cluster to be ready..."
	}
	if strings.Contains(line, "Deploying KECS control plane") {
		return "Deploying KECS components..."
	}
	if strings.Contains(line, "Starting API server") {
		return "Starting API server..."
	}
	if strings.Contains(line, "KECS is ready") {
		return "KECS is ready!"
	}
	
	return ""
}

// tuiLogWriter implements io.Writer to redirect logs to the TUI
type tuiLogWriter struct {
	msgChan chan<- tea.Msg
}

// Write implements io.Writer
func (w *tuiLogWriter) Write(p []byte) (n int, err error) {
	line := strings.TrimSpace(string(p))
	if line != "" {
		// Send log as a regular log message
		w.msgChan <- startupLogMsg{line: line}
	}
	return len(p), nil
}

