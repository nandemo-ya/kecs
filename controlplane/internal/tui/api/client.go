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
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// HTTPClient implements the Client interface using HTTP
type HTTPClient struct {
	baseURL     string
	httpClient  *http.Client
	k3dProvider *K3dInstanceProvider // For direct k3d access when API is not available
}

// NewHTTPClient creates a new HTTP-based API client
func NewHTTPClient(baseURL string) *HTTPClient {
	// Create k3d provider for direct instance listing
	k3dProvider, _ := NewK3dInstanceProvider() // Ignore error, will fallback to API

	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		k3dProvider: k3dProvider,
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
	// Always use k3d provider for listing instances
	// This ensures we can see instances even when their API is not running
	if c.k3dProvider != nil {
		return c.k3dProvider.ListInstances(ctx)
	}

	// Fallback to API if k3d provider is not available
	var instances []Instance
	err := c.doRequest(ctx, "GET", "/api/instances", nil, &instances)
	return instances, err
}

func (c *HTTPClient) GetInstance(ctx context.Context, name string) (*Instance, error) {
	// Always use k3d provider for getting instance info
	if c.k3dProvider != nil {
		return c.k3dProvider.GetInstance(ctx, name)
	}

	// Fallback to API if k3d provider is not available
	var instance Instance
	path := fmt.Sprintf("/api/instances/%s", url.PathEscape(name))
	err := c.doRequest(ctx, "GET", path, nil, &instance)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (c *HTTPClient) CreateInstance(ctx context.Context, opts CreateInstanceOptions) (*Instance, error) {
	// Always use k3d provider for creating instances
	// This ensures we can create instances without any KECS API running
	if c.k3dProvider != nil {
		return c.k3dProvider.CreateInstance(ctx, opts)
	}

	// Fallback to API if k3d provider is not available
	var instance Instance
	err := c.doRequest(ctx, "POST", "/api/instances", opts, &instance)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (c *HTTPClient) StartInstance(ctx context.Context, name string) error {
	// Always use k3d provider for starting instances
	if c.k3dProvider != nil {
		return c.k3dProvider.StartInstance(ctx, name)
	}

	// Fallback to API if k3d provider is not available
	path := fmt.Sprintf("/api/instances/%s/start", url.PathEscape(name))
	return c.doRequest(ctx, "POST", path, nil, nil)
}

func (c *HTTPClient) StopInstance(ctx context.Context, name string) error {
	// Always use k3d provider for stopping instances
	if c.k3dProvider != nil {
		return c.k3dProvider.StopInstance(ctx, name)
	}

	// Fallback to API if k3d provider is not available
	path := fmt.Sprintf("/api/instances/%s/stop", url.PathEscape(name))
	return c.doRequest(ctx, "POST", path, nil, nil)
}

func (c *HTTPClient) DeleteInstance(ctx context.Context, name string) error {
	// Always use k3d provider for deleting instances
	if c.k3dProvider != nil {
		return c.k3dProvider.DeleteInstance(ctx, name)
	}

	// Fallback to API if k3d provider is not available
	path := fmt.Sprintf("/api/instances/%s", url.PathEscape(name))
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

func (c *HTTPClient) GetInstanceLogs(ctx context.Context, name string, follow bool) (<-chan LogEntry, error) {
	// Always use k3d provider for getting instance logs
	if c.k3dProvider != nil {
		return c.k3dProvider.GetInstanceLogs(ctx, name, follow)
	}

	// Fallback: streaming logs not implemented for HTTP API
	ch := make(chan LogEntry)
	close(ch)
	return ch, fmt.Errorf("streaming logs not implemented for HTTP API")
}

// ECS Cluster operations

func (c *HTTPClient) ListClusters(ctx context.Context, instanceName string) ([]string, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/ListClusters", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		resp, err := client.Post(url, "application/json", bytes.NewReader([]byte("{}")))
		if err != nil {
			return nil, fmt.Errorf("failed to call ListClusters: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ListClusters returned status %d", resp.StatusCode)
		}

		var result ListClustersResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.ClusterArns, nil
	}

	// Fallback to admin API path
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
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/DescribeClusters", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(DescribeClustersRequest{Clusters: clusterNames})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call DescribeClusters: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("DescribeClusters returned status %d", resp.StatusCode)
		}

		var result DescribeClustersResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.Clusters, nil
	}

	// Fallback to admin API path
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
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/CreateCluster", inst.APIPort)
		client := &http.Client{Timeout: 30 * time.Second}

		reqBody, _ := json.Marshal(CreateClusterRequest{ClusterName: clusterName})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call CreateCluster: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("CreateCluster returned status %d: %s", resp.StatusCode, string(body))
		}

		var result CreateClusterResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.Cluster, nil
	}

	// Fallback to admin API path
	path := fmt.Sprintf("/api/instances/%s/clusters", url.PathEscape(instanceName))
	req := CreateClusterRequest{ClusterName: clusterName}
	var resp CreateClusterResponse
	err := c.doRequest(ctx, "POST", path, req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Cluster, nil
}

func (c *HTTPClient) DeleteCluster(ctx context.Context, instanceName, clusterName string) error {
	path := fmt.Sprintf("/api/instances/%s/clusters/%s", url.PathEscape(instanceName), url.PathEscape(clusterName))
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ECS Service operations

func (c *HTTPClient) ListServices(ctx context.Context, instanceName, clusterName string) ([]string, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/ListServices", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(ListServicesRequest{Cluster: clusterName})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call ListServices: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ListServices returned status %d", resp.StatusCode)
		}

		var result ListServicesResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.ServiceArns, nil
	}

	// Fallback to admin API path
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
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/DescribeServices", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(DescribeServicesRequest{
			Cluster:  clusterName,
			Services: serviceNames,
		})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call DescribeServices: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("DescribeServices returned status %d", resp.StatusCode)
		}

		var result DescribeServicesResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.Services, nil
	}

	// Fallback to admin API path
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

func (c *HTTPClient) UpdateServiceDesiredCount(instanceName, clusterName, serviceNameOrArn string, desiredCount int) error {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(context.Background(), instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly using UpdateService
		url := fmt.Sprintf("http://localhost:%d/v1/UpdateService", inst.APIPort)
		client := &http.Client{Timeout: 10 * time.Second}

		// Create UpdateService request with only desired count change
		// ECS API accepts either service name or ARN
		reqBody := map[string]interface{}{
			"service":      serviceNameOrArn,
			"cluster":      clusterName,
			"desiredCount": desiredCount,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
		req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.UpdateService")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to call UpdateService: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("UpdateService failed: %s", string(body))
		}

		return nil
	}

	// Fallback to regular API call
	return fmt.Errorf("UpdateServiceDesiredCount not implemented for this client type")
}

func (c *HTTPClient) UpdateServiceTaskDefinition(instanceName, clusterName, serviceName, taskDefinition string) error {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(context.Background(), instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly using UpdateService
		url := fmt.Sprintf("http://localhost:%d/v1/UpdateService", inst.APIPort)
		client := &http.Client{Timeout: 10 * time.Second}

		// Create UpdateService request with task definition change
		reqBody := map[string]interface{}{
			"service":        serviceName,
			"cluster":        clusterName,
			"taskDefinition": taskDefinition,
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/x-amz-json-1.1")
		req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.UpdateService")

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to call UpdateService: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("UpdateService failed: %s", string(body))
		}

		return nil
	}

	// Fallback to regular API call
	return fmt.Errorf("UpdateServiceTaskDefinition not implemented for this client type")
}

func (c *HTTPClient) DeleteService(ctx context.Context, instanceName, clusterName, serviceName string) error {
	path := fmt.Sprintf("/api/instances/%s/services/%s", url.PathEscape(instanceName), url.PathEscape(serviceName))
	req := map[string]string{"cluster": clusterName}
	return c.doRequest(ctx, "DELETE", path, req, nil)
}

// ECS Task operations

func (c *HTTPClient) ListTasks(ctx context.Context, instanceName, clusterName string, serviceName string) ([]string, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/ListTasks", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(ListTasksRequest{
			Cluster:     clusterName,
			ServiceName: serviceName,
		})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call ListTasks: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ListTasks returned status %d", resp.StatusCode)
		}

		var result ListTasksResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.TaskArns, nil
	}

	// Fallback to admin API path
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
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/DescribeTasks", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(DescribeTasksRequest{
			Cluster: clusterName,
			Tasks:   taskArns,
		})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call DescribeTasks: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("DescribeTasks returned status %d", resp.StatusCode)
		}

		var result DescribeTasksResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result.Tasks, nil
	}

	// Fallback to admin API path
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

func (c *HTTPClient) StopTask(ctx context.Context, instanceName, clusterName, taskArn string, reason string) error {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's ECS API directly
		apiURL := fmt.Sprintf("http://localhost:%d/v1/StopTask", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(map[string]interface{}{
			"cluster": clusterName,
			"task":    taskArn,
			"reason":  reason,
		})

		resp, err := client.Post(apiURL, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return fmt.Errorf("failed to call StopTask: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("StopTask returned status %d: %s", resp.StatusCode, string(body))
		}

		return nil
	}

	// Fallback to admin API path
	path := fmt.Sprintf("/api/instances/%s/tasks/%s", url.PathEscape(instanceName), url.PathEscape(taskArn))
	req := map[string]string{"cluster": clusterName, "reason": reason}
	return c.doRequest(ctx, "DELETE", path, req, nil)
}

// Task Definition operations

func (c *HTTPClient) ListTaskDefinitions(ctx context.Context, instanceName string) ([]string, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/ListTaskDefinitions", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(map[string]interface{}{})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call ListTaskDefinitions: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("ListTaskDefinitions returned status %d: %s", resp.StatusCode, string(body))
		}

		var result map[string][]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result["taskDefinitionArns"], nil
	}

	// Fallback to admin API path (which doesn't exist yet)
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

func (c *HTTPClient) ListTaskDefinitionFamilies(ctx context.Context, instanceName string) ([]string, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/ListTaskDefinitionFamilies", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(map[string]interface{}{})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call ListTaskDefinitionFamilies: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ListTaskDefinitionFamilies returned status %d", resp.StatusCode)
		}

		var result map[string][]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		return result["families"], nil
	}

	// Fallback to admin API path
	path := fmt.Sprintf("/api/instances/%s/task-definition-families", url.PathEscape(instanceName))
	var families []string
	err := c.doRequest(ctx, "GET", path, nil, &families)
	return families, err
}

func (c *HTTPClient) ListTaskDefinitionRevisions(ctx context.Context, instanceName string, family string) ([]TaskDefinitionRevision, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly to list task definitions for this family
		url := fmt.Sprintf("http://localhost:%d/v1/ListTaskDefinitions", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(map[string]interface{}{
			"familyPrefix": family,
		})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call ListTaskDefinitions: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ListTaskDefinitions returned status %d", resp.StatusCode)
		}

		var result map[string][]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Convert task definition ARNs to revisions
		var revisions []TaskDefinitionRevision
		for _, arn := range result["taskDefinitionArns"] {
			// Parse ARN to extract revision number
			// Format: arn:aws:ecs:region:account:task-definition/family:revision
			parts := strings.Split(arn, ":")
			if len(parts) > 0 {
				revStr := parts[len(parts)-1]
				revNum, _ := strconv.Atoi(revStr)
				revisions = append(revisions, TaskDefinitionRevision{
					Family:    family,
					Revision:  revNum,
					Status:    "ACTIVE",
					CreatedAt: time.Now(), // Mock for now
				})
			}
		}

		return revisions, nil
	}

	// Fallback to admin API path
	path := fmt.Sprintf("/api/instances/%s/task-definition-families/%s/revisions", url.PathEscape(instanceName), url.PathEscape(family))
	var revisions []TaskDefinitionRevision
	err := c.doRequest(ctx, "GET", path, nil, &revisions)
	return revisions, err
}

func (c *HTTPClient) DescribeTaskDefinition(ctx context.Context, instanceName string, taskDefArn string) (*TaskDefinition, error) {
	// Get instance info to find API port
	if c.k3dProvider != nil {
		inst, err := c.k3dProvider.GetInstance(ctx, instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance: %w", err)
		}

		// Call the instance's API directly
		url := fmt.Sprintf("http://localhost:%d/v1/DescribeTaskDefinition", inst.APIPort)
		client := &http.Client{Timeout: 5 * time.Second}

		reqBody, _ := json.Marshal(map[string]interface{}{
			"taskDefinition": taskDefArn,
		})
		resp, err := client.Post(url, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to call DescribeTaskDefinition: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("DescribeTaskDefinition returned status %d", resp.StatusCode)
		}

		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Extract taskDefinition from response
		if td, ok := result["taskDefinition"].(map[string]interface{}); ok {
			// Convert to TaskDefinition struct
			taskDefJSON, _ := json.Marshal(td)
			var taskDef TaskDefinition
			if err := json.Unmarshal(taskDefJSON, &taskDef); err != nil {
				return nil, fmt.Errorf("failed to parse task definition: %w", err)
			}
			return &taskDef, nil
		}

		return nil, fmt.Errorf("task definition not found in response")
	}

	// Fallback to admin API path
	path := fmt.Sprintf("/api/instances/%s/task-definitions/%s", url.PathEscape(instanceName), url.PathEscape(taskDefArn))
	var taskDef TaskDefinition
	err := c.doRequest(ctx, "GET", path, nil, &taskDef)
	if err != nil {
		return nil, err
	}
	return &taskDef, nil
}

// Health check

func (c *HTTPClient) HealthCheck(ctx context.Context, instanceName string) error {
	path := fmt.Sprintf("/api/instances/%s/health", url.PathEscape(instanceName))
	return c.doRequest(ctx, "GET", path, nil, nil)
}

// GetInstanceCreationStatus gets the creation status of an instance
func (c *HTTPClient) GetInstanceCreationStatus(ctx context.Context, name string) (*CreationStatus, error) {
	// Always use k3d provider first for getting creation status
	// This is important because during instance creation, the API may not be available yet
	if c.k3dProvider != nil {
		return c.k3dProvider.GetInstanceCreationStatus(ctx, name)
	}

	// Fallback to API if k3d provider is not available
	path := fmt.Sprintf("/api/instances/%s/creation-status", url.PathEscape(name))
	var status CreationStatus
	err := c.doRequest(ctx, "GET", path, nil, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// ELBv2 operations

// XML response structures for ELBv2 API
type describeLoadBalancersResponse struct {
	XMLName xml.Name                    `xml:"DescribeLoadBalancersResponse"`
	Result  describeLoadBalancersResult `xml:"DescribeLoadBalancersResult"`
}

type describeLoadBalancersResult struct {
	LoadBalancers []xmlLoadBalancer `xml:"LoadBalancers>member"`
}

type xmlLoadBalancer struct {
	LoadBalancerArn       string     `xml:"LoadBalancerArn"`
	LoadBalancerName      string     `xml:"LoadBalancerName"`
	DNSName               string     `xml:"DNSName"`
	CanonicalHostedZoneId string     `xml:"CanonicalHostedZoneId"`
	Type                  string     `xml:"Type"`
	Scheme                string     `xml:"Scheme"`
	VpcId                 string     `xml:"VpcId"`
	State                 xmlLBState `xml:"State"`
	CreatedTime           string     `xml:"CreatedTime"`
	IpAddressType         string     `xml:"IpAddressType"`
}

type xmlLBState struct {
	Code   string `xml:"Code"`
	Reason string `xml:"Reason"`
}

// ListLoadBalancers retrieves all load balancers in the instance
func (c *HTTPClient) ListLoadBalancers(ctx context.Context, instanceName string) ([]ELBv2LoadBalancer, error) {
	// Construct the ELBv2 API endpoint using Form URL-encoded format
	apiURL := fmt.Sprintf("http://localhost:%d/", c.getPortForInstance(instanceName))

	// Prepare form data
	formData := url.Values{}
	formData.Set("Action", "DescribeLoadBalancers")
	formData.Set("Version", "2015-12-01")

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse XML response
	var xmlResp describeLoadBalancersResponse
	if err := xml.NewDecoder(resp.Body).Decode(&xmlResp); err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	// Convert to our internal types
	var loadBalancers []ELBv2LoadBalancer
	for _, lb := range xmlResp.Result.LoadBalancers {
		// Parse created time
		createdTime, _ := time.Parse(time.RFC3339, lb.CreatedTime)

		loadBalancer := ELBv2LoadBalancer{
			LoadBalancerArn:  lb.LoadBalancerArn,
			LoadBalancerName: lb.LoadBalancerName,
			DNSName:          lb.DNSName,
			Type:             lb.Type,
			Scheme:           lb.Scheme,
			VpcId:            lb.VpcId,
			CreatedTime:      createdTime,
		}

		// Handle State
		if lb.State.Code != "" {
			loadBalancer.State = &ELBv2LoadBalancerState{
				Code:   lb.State.Code,
				Reason: lb.State.Reason,
			}
		}

		loadBalancers = append(loadBalancers, loadBalancer)
	}

	return loadBalancers, nil
}

// XML response structures for Target Groups
type describeTargetGroupsResponse struct {
	XMLName xml.Name                   `xml:"DescribeTargetGroupsResponse"`
	Result  describeTargetGroupsResult `xml:"DescribeTargetGroupsResult"`
}

type describeTargetGroupsResult struct {
	TargetGroups []xmlTargetGroup `xml:"TargetGroups>member"`
}

type xmlTargetGroup struct {
	TargetGroupArn             string `xml:"TargetGroupArn"`
	TargetGroupName            string `xml:"TargetGroupName"`
	Protocol                   string `xml:"Protocol"`
	Port                       int32  `xml:"Port"`
	VpcId                      string `xml:"VpcId"`
	HealthCheckEnabled         bool   `xml:"HealthCheckEnabled"`
	HealthCheckPath            string `xml:"HealthCheckPath"`
	HealthCheckProtocol        string `xml:"HealthCheckProtocol"`
	HealthCheckPort            string `xml:"HealthCheckPort"`
	HealthCheckIntervalSeconds int32  `xml:"HealthCheckIntervalSeconds"`
	HealthCheckTimeoutSeconds  int32  `xml:"HealthCheckTimeoutSeconds"`
	HealthyThresholdCount      int32  `xml:"HealthyThresholdCount"`
	UnhealthyThresholdCount    int32  `xml:"UnhealthyThresholdCount"`
	TargetType                 string `xml:"TargetType"`
}

// ListTargetGroups retrieves all target groups in the instance
func (c *HTTPClient) ListTargetGroups(ctx context.Context, instanceName string) ([]ELBv2TargetGroup, error) {
	apiURL := fmt.Sprintf("http://localhost:%d/", c.getPortForInstance(instanceName))

	// Prepare form data
	formData := url.Values{}
	formData.Set("Action", "DescribeTargetGroups")
	formData.Set("Version", "2015-12-01")

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse XML response
	var xmlResp describeTargetGroupsResponse
	if err := xml.NewDecoder(resp.Body).Decode(&xmlResp); err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	// Convert to our internal types
	var targetGroups []ELBv2TargetGroup
	for _, tg := range xmlResp.Result.TargetGroups {
		targetGroup := ELBv2TargetGroup{
			TargetGroupArn:     tg.TargetGroupArn,
			TargetGroupName:    tg.TargetGroupName,
			Protocol:           tg.Protocol,
			Port:               tg.Port,
			VpcId:              tg.VpcId,
			TargetType:         tg.TargetType,
			HealthCheckEnabled: tg.HealthCheckEnabled,
			HealthCheckPath:    tg.HealthCheckPath,
		}

		// Fetch target health for each target group
		health, err := c.getTargetHealth(ctx, instanceName, tg.TargetGroupArn)
		if err == nil {
			targetGroup.HealthyTargetCount = health.Healthy
			targetGroup.UnhealthyTargetCount = health.Unhealthy
			targetGroup.RegisteredTargetsCount = health.Total
		}

		targetGroups = append(targetGroups, targetGroup)
	}

	return targetGroups, nil
}

// XML response structures for Listeners
type describeListenersResponse struct {
	XMLName xml.Name                `xml:"DescribeListenersResponse"`
	Result  describeListenersResult `xml:"DescribeListenersResult"`
}

type describeListenersResult struct {
	Listeners []xmlListener `xml:"Listeners>member"`
}

type xmlListener struct {
	ListenerArn     string              `xml:"ListenerArn"`
	LoadBalancerArn string              `xml:"LoadBalancerArn"`
	Port            int32               `xml:"Port"`
	Protocol        string              `xml:"Protocol"`
	DefaultActions  []xmlListenerAction `xml:"DefaultActions>member"`
}

type xmlListenerAction struct {
	Type           string `xml:"Type"`
	TargetGroupArn string `xml:"TargetGroupArn"`
}

// ListListeners retrieves all listeners for a specific load balancer
func (c *HTTPClient) ListListeners(ctx context.Context, instanceName, loadBalancerArn string) ([]ELBv2Listener, error) {
	apiURL := fmt.Sprintf("http://localhost:%d/", c.getPortForInstance(instanceName))

	// Prepare form data
	formData := url.Values{}
	formData.Set("Action", "DescribeListeners")
	formData.Set("LoadBalancerArn", loadBalancerArn)
	formData.Set("Version", "2015-12-01")

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse XML response
	var xmlResp describeListenersResponse
	if err := xml.NewDecoder(resp.Body).Decode(&xmlResp); err != nil {
		return nil, fmt.Errorf("failed to decode XML response: %w", err)
	}

	// Convert to our internal types
	var listeners []ELBv2Listener
	for _, lst := range xmlResp.Result.Listeners {
		listener := ELBv2Listener{
			ListenerArn:     lst.ListenerArn,
			LoadBalancerArn: lst.LoadBalancerArn,
			Port:            lst.Port,
			Protocol:        lst.Protocol,
		}

		// Convert actions
		for _, act := range lst.DefaultActions {
			listener.DefaultActions = append(listener.DefaultActions, ELBv2ListenerAction{
				Type:           act.Type,
				TargetGroupArn: act.TargetGroupArn,
			})
		}

		listeners = append(listeners, listener)
	}

	return listeners, nil
}

// Helper methods

// instanceConfig represents the minimal instance configuration we need
type instanceConfig struct {
	APIPort int `yaml:"apiPort"`
}

// getPortForInstance returns the API port for the given instance
func (c *HTTPClient) getPortForInstance(instanceName string) int {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to default port if we can't get home directory
		return 5373
	}

	// Build config file path
	configPath := filepath.Join(home, ".kecs", "instances", instanceName, "config.yaml")

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Fall back to default port if config file doesn't exist
		return 5373
	}

	// Parse YAML
	var config instanceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		// Fall back to default port if we can't parse the config
		return 5373
	}

	// Return the configured port or default if not set
	if config.APIPort > 0 {
		return config.APIPort
	}
	return 5373
}

// XML response structures for Target Health
type describeTargetHealthResponse struct {
	XMLName xml.Name                   `xml:"DescribeTargetHealthResponse"`
	Result  describeTargetHealthResult `xml:"DescribeTargetHealthResult"`
}

type describeTargetHealthResult struct {
	TargetHealthDescriptions []xmlTargetHealthDescription `xml:"TargetHealthDescriptions>member"`
}

type xmlTargetHealthDescription struct {
	Target       xmlTarget       `xml:"Target"`
	TargetHealth xmlTargetHealth `xml:"TargetHealth"`
}

type xmlTarget struct {
	Id   string `xml:"Id"`
	Port int32  `xml:"Port"`
}

type xmlTargetHealth struct {
	State string `xml:"State"`
}

// getTargetHealth retrieves target health counts for a target group
func (c *HTTPClient) getTargetHealth(ctx context.Context, instanceName, targetGroupArn string) (*struct {
	Healthy   int
	Unhealthy int
	Total     int
}, error) {
	apiURL := fmt.Sprintf("http://localhost:%d/", c.getPortForInstance(instanceName))

	// Prepare form data
	formData := url.Values{}
	formData.Set("Action", "DescribeTargetHealth")
	formData.Set("TargetGroupArn", targetGroupArn)
	formData.Set("Version", "2015-12-01")

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse XML response
	var xmlResp describeTargetHealthResponse
	if err := xml.NewDecoder(resp.Body).Decode(&xmlResp); err != nil {
		return nil, err
	}

	health := &struct {
		Healthy   int
		Unhealthy int
		Total     int
	}{
		Total: len(xmlResp.Result.TargetHealthDescriptions),
	}

	for _, thd := range xmlResp.Result.TargetHealthDescriptions {
		switch thd.TargetHealth.State {
		case "healthy":
			health.Healthy++
		case "unhealthy", "unavailable", "draining":
			health.Unhealthy++
		}
	}

	return health, nil
}

// Close cleans up resources
func (c *HTTPClient) Close() error {
	if c.k3dProvider != nil {
		return c.k3dProvider.Close()
	}
	return nil
}
