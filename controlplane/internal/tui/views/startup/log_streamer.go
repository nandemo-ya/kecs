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
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

// LogStreamer handles streaming logs from KECS startup
type LogStreamer struct {
	cmd          *exec.Cmd
	instanceName string
	apiPort      int
	program      *tea.Program
}

// NewLogStreamer creates a new log streamer
func NewLogStreamer(instanceName string, apiPort int) *LogStreamer {
	return &LogStreamer{
		instanceName: instanceName,
		apiPort:      apiPort,
	}
}

// SetProgram sets the tea.Program for sending messages
func (s *LogStreamer) SetProgram(p *tea.Program) {
	s.program = p
}

// Start begins the KECS startup process
func (s *LogStreamer) Start(ctx context.Context) error {
	// Build the start command
	args := []string{"start"}
	if s.instanceName != "" && s.instanceName != "default" {
		args = append(args, "--name", s.instanceName)
	}
	if s.apiPort > 0 && s.apiPort != 8080 {
		args = append(args, "--api-port", fmt.Sprintf("%d", s.apiPort))
	}
	args = append(args, "--verbose")
	
	// Create the command
	s.cmd = exec.CommandContext(ctx, "kecs", args...)
	
	// Set up pipes
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start KECS: %w", err)
	}
	
	// Start streaming logs
	go s.streamLogs(stdout, false)
	go s.streamLogs(stderr, true)
	
	// Monitor the process
	go s.monitorProcess()
	
	return nil
}

// streamLogs reads from a reader and sends log messages
func (s *LogStreamer) streamLogs(reader io.Reader, isError bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if shouldDisplayLog(line) {
			s.sendMessage(startupLogMsg{line: formatLogLine(line)})
		}
		
		// Check for progress indicators
		if progress := extractProgress(line); progress != "" {
			s.sendMessage(startupProgressMsg{message: progress})
		}
	}
}

// monitorProcess monitors the KECS process and checks for readiness
func (s *LogStreamer) monitorProcess() {
	// Monitor process exit
	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()
	
	// Check for readiness
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(2 * time.Minute)
	retries := 0
	maxRetries := 60
	
	for {
		select {
		case err := <-done:
			// Process exited
			if err != nil {
				s.sendMessage(startupErrorMsg{err: fmt.Errorf("KECS process exited: %w", err)})
			}
			return
			
		case <-ticker.C:
			// Check if KECS is ready
			retries++
			if s.isKECSReady() {
				s.sendMessage(startupCompleteMsg{})
				return
			}
			
			if retries%5 == 0 {
				s.sendMessage(startupProgressMsg{
					message: fmt.Sprintf("Waiting for KECS to be ready... (%ds)", retries),
				})
			}
			
			if retries >= maxRetries {
				s.sendMessage(startupErrorMsg{
					err: fmt.Errorf("timeout waiting for KECS to start"),
				})
				s.cmd.Process.Kill()
				return
			}
			
		case <-timeout:
			s.sendMessage(startupErrorMsg{
				err: fmt.Errorf("timeout waiting for KECS to start"),
			})
			s.cmd.Process.Kill()
			return
		}
	}
}

// isKECSReady checks if KECS is ready
func (s *LogStreamer) isKECSReady() bool {
	if s.apiPort == 0 {
		s.apiPort = 8080
	}
	
	endpoint := fmt.Sprintf("http://localhost:%d", s.apiPort)
	client := api.NewClient(endpoint)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	_, err := client.ListClusters(ctx)
	return err == nil
}

// sendMessage sends a message to the tea.Program if available
func (s *LogStreamer) sendMessage(msg tea.Msg) {
	if s.program != nil {
		s.program.Send(msg)
	}
}

// StartKECSWithStreamer starts KECS with a log streamer
func StartKECSWithStreamer(instanceName string, apiPort int) tea.Cmd {
	return func() tea.Msg {
		// Start the process immediately and return progress message
		ctx := context.Background()
		
		// Build the start command
		args := []string{"start"}
		if instanceName != "" && instanceName != "default" {
			args = append(args, "--name", instanceName)
		}
		if apiPort > 0 && apiPort != 8080 {
			args = append(args, "--api-port", fmt.Sprintf("%d", apiPort))
		}
		args = append(args, "--verbose")
		
		// Create the command
		cmd := exec.CommandContext(ctx, "kecs", args...)
		
		// Set up pipes
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
		
		// Create message channel
		msgChan := make(chan tea.Msg, 100)
		
		// Start goroutines to read logs
		go streamLogsToChannel(stdout, msgChan, false)
		go streamLogsToChannel(stderr, msgChan, true)
		
		// Monitor the process
		go monitorProcessWithChannel(cmd, instanceName, apiPort, msgChan)
		
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

// streamLogsToChannel reads from a reader and sends log messages to a channel
func streamLogsToChannel(reader io.Reader, msgChan chan<- tea.Msg, isError bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if shouldDisplayLog(line) {
			msgChan <- startupLogMsg{line: formatLogLine(line)}
		}
		
		// Check for progress indicators
		if progress := extractProgress(line); progress != "" {
			msgChan <- startupProgressMsg{message: progress}
		}
	}
}

// monitorProcessWithChannel monitors the KECS process and sends messages to a channel
func monitorProcessWithChannel(cmd *exec.Cmd, instanceName string, apiPort int, msgChan chan<- tea.Msg) {
	defer close(msgChan)
	
	// Monitor process exit
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	// Check for readiness
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	timeout := time.After(2 * time.Minute)
	retries := 0
	maxRetries := 60
	
	for {
		select {
		case err := <-done:
			// Process exited
			if err != nil {
				msgChan <- startupErrorMsg{err: fmt.Errorf("KECS process exited: %w", err)}
			}
			return
			
		case <-ticker.C:
			// Check if KECS is ready
			retries++
			if isKECSReadyForPort(instanceName, apiPort) {
				msgChan <- startupCompleteMsg{}
				return
			}
			
			if retries%5 == 0 {
				msgChan <- startupProgressMsg{
					message: fmt.Sprintf("Waiting for KECS to be ready... (%ds)", retries),
				}
			}
			
			if retries >= maxRetries {
				msgChan <- startupErrorMsg{
					err: fmt.Errorf("timeout waiting for KECS to start"),
				}
				cmd.Process.Kill()
				return
			}
			
		case <-timeout:
			msgChan <- startupErrorMsg{
				err: fmt.Errorf("timeout waiting for KECS to start"),
			}
			cmd.Process.Kill()
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

// StartupStreamerMsg carries the log streamer instance
type StartupStreamerMsg struct {
	Streamer *LogStreamer
}