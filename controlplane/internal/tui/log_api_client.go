package tui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultLogAPIClient implements LogAPIClient interface
type DefaultLogAPIClient struct {
	baseURL string
	client  *http.Client
}

// NewLogAPIClient creates a new log API client
func NewLogAPIClient(baseURL string) *DefaultLogAPIClient {
	return &DefaultLogAPIClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetLogs retrieves logs from the API
func (c *DefaultLogAPIClient) GetLogs(ctx context.Context, taskArn, container string, follow bool) ([]storage.TaskLog, error) {
	// Build URL with query parameters
	params := url.Values{}
	params.Set("taskArn", taskArn)
	if container != "" {
		params.Set("containerName", container)
	}
	params.Set("limit", "1000")
	
	endpoint := fmt.Sprintf("%s/v1/logs?%s", c.baseURL, params.Encode())
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Parse response
	var result struct {
		Logs []storage.TaskLog `json:"logs"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return result.Logs, nil
}

// StreamLogs streams logs from the API using SSE
func (c *DefaultLogAPIClient) StreamLogs(ctx context.Context, taskArn, container string) (<-chan storage.TaskLog, error) {
	// Build URL with query parameters
	params := url.Values{}
	params.Set("taskArn", taskArn)
	if container != "" {
		params.Set("containerName", container)
	}
	params.Set("follow", "true")
	
	endpoint := fmt.Sprintf("%s/v1/logs/stream?%s", c.baseURL, params.Encode())
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	
	// Execute request without timeout for streaming
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Create channel for logs
	logChan := make(chan storage.TaskLog, 100)
	
	// Start goroutine to read SSE stream
	go func() {
		defer resp.Body.Close()
		defer close(logChan)
		
		scanner := bufio.NewScanner(resp.Body)
		var eventData strings.Builder
		
		for scanner.Scan() {
			line := scanner.Text()
			
			// SSE format: "data: {json}"
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				eventData.WriteString(data)
			} else if line == "" && eventData.Len() > 0 {
				// Empty line indicates end of event
				var log storage.TaskLog
				if err := json.Unmarshal([]byte(eventData.String()), &log); err == nil {
					select {
					case logChan <- log:
					case <-ctx.Done():
						return
					}
				}
				eventData.Reset()
			}
		}
	}()
	
	return logChan, nil
}

// MockLogAPIClient provides a mock implementation for testing
type MockLogAPIClient struct {
	logs []storage.TaskLog
}

// NewMockLogAPIClient creates a new mock log API client
func NewMockLogAPIClient() *MockLogAPIClient {
	// Generate some sample logs
	logs := []storage.TaskLog{}
	baseTime := time.Now().Add(-1 * time.Hour)
	
	logLines := []struct {
		level string
		line  string
	}{
		{"INFO", "Starting application..."},
		{"INFO", "Loading configuration from /etc/config.yaml"},
		{"DEBUG", "Configuration loaded successfully"},
		{"INFO", "Connecting to database..."},
		{"INFO", "Database connection established"},
		{"INFO", "Starting HTTP server on port 8080"},
		{"INFO", "Server started successfully"},
		{"INFO", "Received GET request: /health"},
		{"DEBUG", "Health check passed"},
		{"INFO", "Received POST request: /api/tasks"},
		{"INFO", "Processing task: task-123"},
		{"WARN", "Task processing took longer than expected: 5.2s"},
		{"ERROR", "Failed to connect to external service: timeout"},
		{"INFO", "Retrying connection..."},
		{"INFO", "Connection successful on retry"},
		{"INFO", "Task completed successfully"},
		{"INFO", "Received GET request: /metrics"},
		{"DEBUG", "Generating metrics report"},
		{"INFO", "Metrics report generated"},
		{"INFO", "Shutting down server..."},
		{"INFO", "Server shutdown complete"},
	}
	
	for i, entry := range logLines {
		logs = append(logs, storage.TaskLog{
			TaskArn:       "arn:aws:ecs:us-east-1:123456789012:task/test/task-abc123",
			ContainerName: "app",
			Timestamp:     baseTime.Add(time.Duration(i) * time.Second),
			LogLine:       entry.line,
			LogLevel:      entry.level,
			CreatedAt:     baseTime.Add(time.Duration(i) * time.Second),
		})
	}
	
	return &MockLogAPIClient{
		logs: logs,
	}
}

// GetLogs returns mock logs
func (m *MockLogAPIClient) GetLogs(ctx context.Context, taskArn, container string, follow bool) ([]storage.TaskLog, error) {
	return m.logs, nil
}

// StreamLogs streams mock logs
func (m *MockLogAPIClient) StreamLogs(ctx context.Context, taskArn, container string) (<-chan storage.TaskLog, error) {
	logChan := make(chan storage.TaskLog, 100)
	
	go func() {
		defer close(logChan)
		
		// Send existing logs
		for _, log := range m.logs {
			select {
			case logChan <- log:
			case <-ctx.Done():
				return
			}
		}
		
		// Simulate streaming new logs
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		
		counter := 0
		for {
			select {
			case <-ticker.C:
				counter++
				log := storage.TaskLog{
					TaskArn:       taskArn,
					ContainerName: container,
					Timestamp:     time.Now(),
					LogLine:       fmt.Sprintf("Streaming log entry #%d", counter),
					LogLevel:      "INFO",
					CreatedAt:     time.Now(),
				}
				
				select {
				case logChan <- log:
				case <-ctx.Done():
					return
				}
				
			case <-ctx.Done():
				return
			}
		}
	}()
	
	return logChan, nil
}