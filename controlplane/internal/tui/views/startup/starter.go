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
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

// StartKECSCmd starts KECS and streams logs
func StartKECSCmd(instanceName string, apiPort int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Build the start command
		args := []string{"start"}
		if instanceName != "" && instanceName != "default" {
			args = append(args, "--name", instanceName)
		}
		if apiPort > 0 && apiPort != 8080 {
			args = append(args, "--api-port", fmt.Sprintf("%d", apiPort))
		}
		args = append(args, "--verbose") // Enable verbose logging
		
		// Execute the command
		cmd := exec.CommandContext(ctx, "kecs", args...)
		
		// Set up pipes for stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return startupErrorMsg{err: fmt.Errorf("failed to create stdout pipe: %w", err)}
		}
		
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return startupErrorMsg{err: fmt.Errorf("failed to create stderr pipe: %w", err)}
		}
		
		// Start the command
		if err := cmd.Start(); err != nil {
			return startupErrorMsg{err: fmt.Errorf("failed to start KECS: %w", err)}
		}
		
		// Create a channel to receive logs
		logChan := make(chan tea.Msg)
		
		// Start goroutines to read stdout and stderr
		go streamLogs(stdout, logChan, false)
		go streamLogs(stderr, logChan, true)
		
		// Monitor the startup process
		go func() {
			// Wait for the command to complete or for KECS to be ready
			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()
			
			// Check for KECS readiness
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			
			timeout := time.After(2 * time.Minute)
			retries := 0
			maxRetries := 60
			
			for {
				select {
				case err := <-done:
					// Command exited
					if err != nil {
						logChan <- startupErrorMsg{err: fmt.Errorf("KECS process exited: %w", err)}
					}
					close(logChan)
					return
					
				case <-ticker.C:
					// Check if KECS is ready
					retries++
					if isKECSReady(instanceName, apiPort) {
						logChan <- startupCompleteMsg{}
						// Don't close logChan yet, let the process continue running
						return
					}
					
					if retries%5 == 0 {
						logChan <- startupProgressMsg{
							message: fmt.Sprintf("Waiting for KECS to be ready... (%ds)", retries),
						}
					}
					
					if retries >= maxRetries {
						logChan <- startupErrorMsg{
							err: fmt.Errorf("timeout waiting for KECS to start"),
						}
						cmd.Process.Kill()
						close(logChan)
						return
					}
					
				case <-timeout:
					logChan <- startupErrorMsg{
						err: fmt.Errorf("timeout waiting for KECS to start"),
					}
					cmd.Process.Kill()
					close(logChan)
					return
				}
			}
		}()
		
		// Return a command that sends the first log message
		// The log viewer will continue to receive messages through updates
		return func() tea.Msg {
			// Send initial progress message
			return startupProgressMsg{message: "Starting KECS..."}
		}
	}
}

// streamLogs reads from a reader and sends log messages
func streamLogs(reader io.Reader, logChan chan<- tea.Msg, isError bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// Filter out some verbose logs for cleaner display
		if shouldDisplayLog(line) {
			logChan <- startupLogMsg{line: formatLogLine(line)}
		}
		
		// Check for specific progress indicators
		if progress := extractProgress(line); progress != "" {
			logChan <- startupProgressMsg{message: progress}
		}
	}
}

// isKECSReady checks if KECS is ready by trying to connect to the API
func isKECSReady(instanceName string, apiPort int) bool {
	if apiPort == 0 {
		apiPort = 8080
	}
	
	endpoint := fmt.Sprintf("http://localhost:%d", apiPort)
	client := api.NewClient(endpoint)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Try to list clusters as a health check
	_, err := client.ListClusters(ctx)
	return err == nil
}


// CheckKECSStatus checks if KECS is running at the given endpoint
func CheckKECSStatus(endpoint string) (bool, error) {
	client := api.NewClient(endpoint)
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	// Try to list clusters as a health check
	_, err := client.ListClusters(ctx)
	if err != nil {
		// Check if it's a connection error
		if strings.Contains(err.Error(), "connection refused") ||
			strings.Contains(err.Error(), "no such host") ||
			strings.Contains(err.Error(), "timeout") {
			return false, nil
		}
		// Other errors might mean KECS is running but having issues
		return false, err
	}
	
	return true, nil
}