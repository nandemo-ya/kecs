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

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GetTaskLogsRequest represents the request for getting task logs
type GetTaskLogsRequest struct {
	Cluster    string `json:"cluster"`
	TaskArn    string `json:"taskArn"`
	Follow     bool   `json:"follow,omitempty"`
	Timestamps bool   `json:"timestamps,omitempty"`
	Since      string `json:"since,omitempty"`
	Tail       *int64 `json:"tail,omitempty"`
}

// GetTaskLogsResponse represents the response for getting task logs
type GetTaskLogsResponse struct {
	Logs []LogEntry `json:"logs"`
}

// GetTaskLogs retrieves logs for a specific task
func (c *HTTPClient) GetTaskLogs(ctx context.Context, instanceName, clusterName, taskArn string, tail int64) ([]LogEntry, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}
		
		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/GetTaskLogs", inst.APIPort)
		client := &http.Client{Timeout: 10 * time.Second}
		
		// Build request
		reqBody := GetTaskLogsRequest{
			Cluster:    clusterName,
			TaskArn:    taskArn,
			Timestamps: true,
			Tail:       &tail,
		}
		
		body, _ := json.Marshal(reqBody)
		resp, err := client.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to call GetTaskLogs: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GetTaskLogs returned status %d", resp.StatusCode)
		}
		
		var result GetTaskLogsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		
		return result.Logs, nil
	}
	
	// Fallback to mock data if k3d provider is not available
	return generateMockLogs(taskArn), nil
}

// generateMockLogs generates mock log entries for testing
func generateMockLogs(taskArn string) []LogEntry {
	now := time.Now()
	return []LogEntry{
		{
			Timestamp: now.Add(-5 * time.Minute),
			Level:     "INFO",
			Message:   fmt.Sprintf("Starting task %s", taskArn),
		},
		{
			Timestamp: now.Add(-4 * time.Minute),
			Level:     "INFO",
			Message:   "Pulling container image nginx:latest",
		},
		{
			Timestamp: now.Add(-3 * time.Minute),
			Level:     "INFO",
			Message:   "Successfully pulled image nginx:latest",
		},
		{
			Timestamp: now.Add(-2 * time.Minute),
			Level:     "INFO",
			Message:   "Created container nginx",
		},
		{
			Timestamp: now.Add(-1 * time.Minute),
			Level:     "INFO",
			Message:   "Started container nginx",
		},
		{
			Timestamp: now.Add(-30 * time.Second),
			Level:     "INFO",
			Message:   "nginx: [notice] nginx/1.21.0 built by gcc 9.3.0 (Alpine 9.3.0)",
		},
		{
			Timestamp: now.Add(-20 * time.Second),
			Level:     "INFO",
			Message:   "nginx: [notice] start worker processes",
		},
		{
			Timestamp: now.Add(-10 * time.Second),
			Level:     "WARN",
			Message:   "nginx: [warn] server name conflict on 0.0.0.0:80, ignored",
		},
		{
			Timestamp: now.Add(-5 * time.Second),
			Level:     "INFO",
			Message:   "10.0.0.1 - - [08/Jan/2025:10:30:45 +0000] \"GET / HTTP/1.1\" 200 612",
		},
		{
			Timestamp: now,
			Level:     "INFO",
			Message:   "10.0.0.1 - - [08/Jan/2025:10:30:50 +0000] \"GET /favicon.ico HTTP/1.1\" 404 153",
		},
	}
}