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
	"io"
	"net/http"
	"net/url"
	"time"
)

// HTTPClient implements the Client interface using HTTP
type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPClient creates a new HTTP-based API client
func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request and handles common error cases
func (c *HTTPClient) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for error status codes
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("API error: %s - %s", errResp.Type, errResp.Message)
	}

	// Decode successful response
	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Instance operations

func (c *HTTPClient) ListInstances(ctx context.Context) ([]Instance, error) {
	var instances []Instance
	err := c.doRequest(ctx, "GET", "/api/instances", nil, &instances)
	return instances, err
}

func (c *HTTPClient) GetInstance(ctx context.Context, name string) (*Instance, error) {
	var instance Instance
	path := fmt.Sprintf("/api/instances/%s", url.PathEscape(name))
	err := c.doRequest(ctx, "GET", path, nil, &instance)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (c *HTTPClient) CreateInstance(ctx context.Context, opts CreateInstanceOptions) (*Instance, error) {
	var instance Instance
	err := c.doRequest(ctx, "POST", "/api/instances", opts, &instance)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (c *HTTPClient) DeleteInstance(ctx context.Context, name string) error {
	path := fmt.Sprintf("/api/instances/%s", url.PathEscape(name))
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

func (c *HTTPClient) GetInstanceLogs(ctx context.Context, name string, follow bool) (<-chan LogEntry, error) {
	// TODO: Implement streaming logs
	ch := make(chan LogEntry)
	close(ch)
	return ch, fmt.Errorf("streaming logs not implemented")
}

// ECS Cluster operations

func (c *HTTPClient) ListClusters(ctx context.Context, instanceName string) ([]string, error) {
	path := fmt.Sprintf("/api/instances/%s/clusters", url.PathEscape(instanceName))
	req := ListClustersRequest{}
	var resp ListClustersResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ClusterArns, nil
}

func (c *HTTPClient) DescribeClusters(ctx context.Context, instanceName string, clusterNames []string) ([]Cluster, error) {
	path := fmt.Sprintf("/api/instances/%s/clusters/describe", url.PathEscape(instanceName))
	req := DescribeClustersRequest{
		Clusters: clusterNames,
	}
	var resp DescribeClustersResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Clusters, nil
}

func (c *HTTPClient) CreateCluster(ctx context.Context, instanceName, clusterName string) (*Cluster, error) {
	path := fmt.Sprintf("/api/instances/%s/clusters", url.PathEscape(instanceName))
	req := map[string]string{"clusterName": clusterName}
	var cluster Cluster
	err := c.doRequest(ctx, "POST", path, req, &cluster)
	if err != nil {
		return nil, err
	}
	return &cluster, nil
}

func (c *HTTPClient) DeleteCluster(ctx context.Context, instanceName, clusterName string) error {
	path := fmt.Sprintf("/api/instances/%s/clusters/%s", url.PathEscape(instanceName), url.PathEscape(clusterName))
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ECS Service operations

func (c *HTTPClient) ListServices(ctx context.Context, instanceName, clusterName string) ([]string, error) {
	path := fmt.Sprintf("/api/instances/%s/services", url.PathEscape(instanceName))
	req := ListServicesRequest{
		Cluster: clusterName,
	}
	var resp ListServicesResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.ServiceArns, nil
}

func (c *HTTPClient) DescribeServices(ctx context.Context, instanceName, clusterName string, serviceNames []string) ([]Service, error) {
	path := fmt.Sprintf("/api/instances/%s/services/describe", url.PathEscape(instanceName))
	req := DescribeServicesRequest{
		Cluster:  clusterName,
		Services: serviceNames,
	}
	var resp DescribeServicesResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Services, nil
}

func (c *HTTPClient) CreateService(ctx context.Context, instanceName, clusterName string, service Service) (*Service, error) {
	path := fmt.Sprintf("/api/instances/%s/services", url.PathEscape(instanceName))
	service.ClusterArn = clusterName
	var createdService Service
	err := c.doRequest(ctx, "POST", path, service, &createdService)
	if err != nil {
		return nil, err
	}
	return &createdService, nil
}

func (c *HTTPClient) UpdateService(ctx context.Context, instanceName, clusterName string, service Service) (*Service, error) {
	path := fmt.Sprintf("/api/instances/%s/services/%s", url.PathEscape(instanceName), url.PathEscape(service.ServiceName))
	service.ClusterArn = clusterName
	var updatedService Service
	err := c.doRequest(ctx, "PUT", path, service, &updatedService)
	if err != nil {
		return nil, err
	}
	return &updatedService, nil
}

func (c *HTTPClient) DeleteService(ctx context.Context, instanceName, clusterName, serviceName string) error {
	path := fmt.Sprintf("/api/instances/%s/services/%s", url.PathEscape(instanceName), url.PathEscape(serviceName))
	req := map[string]string{"cluster": clusterName}
	return c.doRequest(ctx, "DELETE", path, req, nil)
}

// ECS Task operations

func (c *HTTPClient) ListTasks(ctx context.Context, instanceName, clusterName string, serviceName string) ([]string, error) {
	path := fmt.Sprintf("/api/instances/%s/tasks", url.PathEscape(instanceName))
	req := ListTasksRequest{
		Cluster:     clusterName,
		ServiceName: serviceName,
	}
	var resp ListTasksResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.TaskArns, nil
}

func (c *HTTPClient) DescribeTasks(ctx context.Context, instanceName, clusterName string, taskArns []string) ([]Task, error) {
	path := fmt.Sprintf("/api/instances/%s/tasks/describe", url.PathEscape(instanceName))
	req := DescribeTasksRequest{
		Cluster: clusterName,
		Tasks:   taskArns,
	}
	var resp DescribeTasksResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Tasks, nil
}

func (c *HTTPClient) RunTask(ctx context.Context, instanceName, clusterName string, taskDef string) (*Task, error) {
	path := fmt.Sprintf("/api/instances/%s/tasks/run", url.PathEscape(instanceName))
	req := map[string]string{
		"cluster":        clusterName,
		"taskDefinition": taskDef,
	}
	var task Task
	err := c.doRequest(ctx, "POST", path, req, &task)
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (c *HTTPClient) StopTask(ctx context.Context, instanceName, clusterName, taskArn string) error {
	path := fmt.Sprintf("/api/instances/%s/tasks/%s", url.PathEscape(instanceName), url.PathEscape(taskArn))
	req := map[string]string{"cluster": clusterName}
	return c.doRequest(ctx, "DELETE", path, req, nil)
}

// Task Definition operations

func (c *HTTPClient) ListTaskDefinitions(ctx context.Context, instanceName string) ([]string, error) {
	path := fmt.Sprintf("/api/instances/%s/task-definitions", url.PathEscape(instanceName))
	var taskDefs []string
	err := c.doRequest(ctx, "GET", path, nil, &taskDefs)
	return taskDefs, err
}

func (c *HTTPClient) RegisterTaskDefinition(ctx context.Context, instanceName string, taskDef interface{}) (string, error) {
	path := fmt.Sprintf("/api/instances/%s/task-definitions", url.PathEscape(instanceName))
	var result map[string]string
	err := c.doRequest(ctx, "POST", path, taskDef, &result)
	if err != nil {
		return "", err
	}
	return result["taskDefinitionArn"], nil
}

func (c *HTTPClient) DeregisterTaskDefinition(ctx context.Context, instanceName string, taskDefArn string) error {
	path := fmt.Sprintf("/api/instances/%s/task-definitions/%s", url.PathEscape(instanceName), url.PathEscape(taskDefArn))
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// Health check

func (c *HTTPClient) HealthCheck(ctx context.Context, instanceName string) error {
	path := fmt.Sprintf("/api/instances/%s/health", url.PathEscape(instanceName))
	return c.doRequest(ctx, "GET", path, nil, nil)
}