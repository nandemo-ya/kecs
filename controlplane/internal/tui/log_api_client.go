package tui

import (
	"bufio"
	"bytes"
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
	// Extract cluster from ARN
	cluster := extractClusterFromArn(taskArn)
	if cluster == "" {
		cluster = "default"
	}

	// Build request body for /v1/GetTaskLogs endpoint
	reqBody := struct {
		Cluster    string `json:"cluster"`
		TaskArn    string `json:"taskArn"`
		Timestamps bool   `json:"timestamps"`
		Tail       *int64 `json:"tail,omitempty"`
	}{
		Cluster:    cluster,
		TaskArn:    taskArn,
		Timestamps: true,
	}

	// Set tail limit
	tailLimit := int64(1000)
	reqBody.Tail = &tailLimit

	// Marshal request body
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Use the correct API endpoint /v1/GetTaskLogs
	endpoint := fmt.Sprintf("%s/v1/GetTaskLogs", c.baseURL)

	// Create POST request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

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
		Logs []struct {
			Timestamp time.Time `json:"timestamp"`
			Level     string    `json:"level"`
			Message   string    `json:"message"`
			Container string    `json:"container,omitempty"`
		} `json:"logs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to storage.TaskLog format
	logs := make([]storage.TaskLog, 0, len(result.Logs))
	for _, log := range result.Logs {
		logs = append(logs, storage.TaskLog{
			TaskArn:       taskArn,
			ContainerName: log.Container,
			Timestamp:     log.Timestamp,
			LogLine:       log.Message,
			LogLevel:      log.Level,
			CreatedAt:     log.Timestamp,
		})
	}

	return logs, nil
}

// StreamLogs streams logs from the API using SSE
func (c *DefaultLogAPIClient) StreamLogs(ctx context.Context, taskArn, container string) (<-chan storage.TaskLog, error) {
	// Extract task ID from ARN
	taskId := extractTaskIdFromArn(taskArn)
	cluster := extractClusterFromArn(taskArn)

	// Build URL with query parameters
	params := url.Values{}
	params.Set("follow", "true")
	if cluster != "" && cluster != "default" {
		params.Set("cluster", cluster)
	}

	// Use the correct API endpoint path with task ID instead of full ARN
	// Format: /api/tasks/{taskId}/containers/{containerName}/logs/stream
	endpoint := fmt.Sprintf("%s/api/tasks/%s/containers/%s/logs/stream?%s",
		c.baseURL,
		taskId,
		container,
		params.Encode())

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

// extractTaskIdFromArn extracts task ID from a task ARN
// ARN format: arn:aws:ecs:region:account:task/cluster/task-id
func extractTaskIdFromArn(taskArn string) string {
	// If it's already just a task ID, return it as is
	if !strings.Contains(taskArn, ":") {
		return taskArn
	}

	parts := strings.Split(taskArn, "/")
	if len(parts) >= 1 {
		return parts[len(parts)-1]
	}
	return taskArn
}

// extractClusterFromArn extracts cluster name from a task ARN
// ARN format: arn:aws:ecs:region:account:task/cluster/task-id
func extractClusterFromArn(taskArn string) string {
	// If it's not an ARN, return empty
	if !strings.Contains(taskArn, ":") {
		return ""
	}

	parts := strings.Split(taskArn, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-2]
	}
	return "default"
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
