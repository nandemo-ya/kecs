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
	"time"
)

// Client represents an ECS API client
type Client struct {
	endpoint   string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// makeRequest makes an API request
func (c *Client) makeRequest(ctx context.Context, action string, payload interface{}, result interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint+"/v1/"+action, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113."+action)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Type    string `json:"__type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			return fmt.Errorf("API error: %s - %s", errorResp.Type, errorResp.Message)
		}
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// ListClusters lists all ECS clusters
func (c *Client) ListClusters(ctx context.Context) (*ListClustersResponse, error) {
	var resp ListClustersResponse
	err := c.makeRequest(ctx, "ListClusters", &ListClustersRequest{}, &resp)
	return &resp, err
}

// DescribeClusters describes one or more clusters
func (c *Client) DescribeClusters(ctx context.Context, clusterArns []string) (*DescribeClustersResponse, error) {
	var resp DescribeClustersResponse
	err := c.makeRequest(ctx, "DescribeClusters", &DescribeClustersRequest{
		Clusters: clusterArns,
	}, &resp)
	return &resp, err
}

// CreateCluster creates a new cluster
func (c *Client) CreateCluster(ctx context.Context, clusterName string) (*CreateClusterResponse, error) {
	var resp CreateClusterResponse
	err := c.makeRequest(ctx, "CreateCluster", &CreateClusterRequest{
		ClusterName: clusterName,
	}, &resp)
	return &resp, err
}

// DeleteCluster deletes a cluster
func (c *Client) DeleteCluster(ctx context.Context, cluster string) (*DeleteClusterResponse, error) {
	var resp DeleteClusterResponse
	err := c.makeRequest(ctx, "DeleteCluster", &DeleteClusterRequest{
		Cluster: cluster,
	}, &resp)
	return &resp, err
}

// ListServices lists services in a cluster
func (c *Client) ListServices(ctx context.Context, cluster string) (*ListServicesResponse, error) {
	var resp ListServicesResponse
	err := c.makeRequest(ctx, "ListServices", &ListServicesRequest{
		Cluster: cluster,
	}, &resp)
	return &resp, err
}

// DescribeServices describes one or more services
func (c *Client) DescribeServices(ctx context.Context, cluster string, services []string) (*DescribeServicesResponse, error) {
	var resp DescribeServicesResponse
	err := c.makeRequest(ctx, "DescribeServices", &DescribeServicesRequest{
		Cluster:  cluster,
		Services: services,
	}, &resp)
	return &resp, err
}

// CreateService creates a new service
func (c *Client) CreateService(ctx context.Context, req *CreateServiceRequest) (*CreateServiceResponse, error) {
	var resp CreateServiceResponse
	err := c.makeRequest(ctx, "CreateService", req, &resp)
	return &resp, err
}

// UpdateService updates a service
func (c *Client) UpdateService(ctx context.Context, req *UpdateServiceRequest) (*UpdateServiceResponse, error) {
	var resp UpdateServiceResponse
	err := c.makeRequest(ctx, "UpdateService", req, &resp)
	return &resp, err
}

// DeleteService deletes a service
func (c *Client) DeleteService(ctx context.Context, cluster, service string) (*DeleteServiceResponse, error) {
	var resp DeleteServiceResponse
	err := c.makeRequest(ctx, "DeleteService", &DeleteServiceRequest{
		Cluster: cluster,
		Service: service,
	}, &resp)
	return &resp, err
}