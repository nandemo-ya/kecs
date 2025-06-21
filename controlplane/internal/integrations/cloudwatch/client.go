package cloudwatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	cloudwatchlogsapi "github.com/nandemo-ya/kecs/controlplane/internal/cloudwatchlogs/generated"
)

// cloudWatchLogsClient implements CloudWatchLogsClient interface using HTTP calls
type cloudWatchLogsClient struct {
	endpoint   string
	httpClient *http.Client
}

// newCloudWatchLogsClient creates a new CloudWatch Logs client
func newCloudWatchLogsClient(endpoint string) CloudWatchLogsClient {
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}
	
	return &cloudWatchLogsClient{
		endpoint:   endpoint,
		httpClient: &http.Client{},
	}
}

// CreateLogGroup creates a log group
func (c *cloudWatchLogsClient) CreateLogGroup(ctx context.Context, params *cloudwatchlogsapi.CreateLogGroupRequest) (*cloudwatchlogsapi.Unit, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.CreateLogGroup")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific error types
		if strings.Contains(string(body), "ResourceAlreadyExistsException") {
			return nil, fmt.Errorf("ResourceAlreadyExistsException: log group already exists")
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &cloudwatchlogsapi.Unit{}, nil
}

// DeleteLogGroup deletes a log group
func (c *cloudWatchLogsClient) DeleteLogGroup(ctx context.Context, params *cloudwatchlogsapi.DeleteLogGroupRequest) (*cloudwatchlogsapi.Unit, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.DeleteLogGroup")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific error types
		if strings.Contains(string(body), "ResourceNotFoundException") {
			return nil, fmt.Errorf("ResourceNotFoundException: log group not found")
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &cloudwatchlogsapi.Unit{}, nil
}

// CreateLogStream creates a log stream
func (c *cloudWatchLogsClient) CreateLogStream(ctx context.Context, params *cloudwatchlogsapi.CreateLogStreamRequest) (*cloudwatchlogsapi.Unit, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.CreateLogStream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific error types
		if strings.Contains(string(body), "ResourceAlreadyExistsException") {
			return nil, fmt.Errorf("ResourceAlreadyExistsException: log stream already exists")
		}
		if strings.Contains(string(body), "ResourceNotFoundException") {
			return nil, fmt.Errorf("ResourceNotFoundException: log group not found")
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &cloudwatchlogsapi.Unit{}, nil
}

// PutRetentionPolicy sets the retention policy for a log group
func (c *cloudWatchLogsClient) PutRetentionPolicy(ctx context.Context, params *cloudwatchlogsapi.PutRetentionPolicyRequest) (*cloudwatchlogsapi.Unit, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328.PutRetentionPolicy")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Check for specific error types
		if strings.Contains(string(body), "ResourceNotFoundException") {
			return nil, fmt.Errorf("ResourceNotFoundException: log group not found")
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &cloudwatchlogsapi.Unit{}, nil
}