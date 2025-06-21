package ecs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
)

const (
	serviceName = "ecs"
	serviceID   = "AmazonEC2ContainerServiceV20141113"
	jsonVersion = "1.1"
	contentType = "application/x-amz-json-1.1"
)

// Client is an ECS service client
type Client struct {
	client   *awsclient.Client
	endpoint string
}

// NewClient creates a new ECS client
func NewClient(config awsclient.Config) *Client {
	client := awsclient.NewClient(config)
	endpoint := client.BuildEndpoint(serviceName)
	
	return &Client{
		client:   client,
		endpoint: endpoint,
	}
}

// ListClusters lists ECS clusters
func (c *Client) ListClusters(ctx context.Context, input *api.ListClustersRequest) (*api.ListClustersResponse, error) {
	return doRequest[api.ListClustersRequest, api.ListClustersResponse](ctx, c, "ListClusters", input)
}

// CreateCluster creates a new ECS cluster
func (c *Client) CreateCluster(ctx context.Context, input *api.CreateClusterRequest) (*api.CreateClusterResponse, error) {
	return doRequest[api.CreateClusterRequest, api.CreateClusterResponse](ctx, c, "CreateCluster", input)
}

// DeleteCluster deletes an ECS cluster
func (c *Client) DeleteCluster(ctx context.Context, input *api.DeleteClusterRequest) (*api.DeleteClusterResponse, error) {
	return doRequest[api.DeleteClusterRequest, api.DeleteClusterResponse](ctx, c, "DeleteCluster", input)
}

// DescribeClusters describes ECS clusters
func (c *Client) DescribeClusters(ctx context.Context, input *api.DescribeClustersRequest) (*api.DescribeClustersResponse, error) {
	return doRequest[api.DescribeClustersRequest, api.DescribeClustersResponse](ctx, c, "DescribeClusters", input)
}

// RegisterTaskDefinition registers a new task definition
func (c *Client) RegisterTaskDefinition(ctx context.Context, input *api.RegisterTaskDefinitionRequest) (*api.RegisterTaskDefinitionResponse, error) {
	return doRequest[api.RegisterTaskDefinitionRequest, api.RegisterTaskDefinitionResponse](ctx, c, "RegisterTaskDefinition", input)
}

// DeregisterTaskDefinition deregisters a task definition
func (c *Client) DeregisterTaskDefinition(ctx context.Context, input *api.DeregisterTaskDefinitionRequest) (*api.DeregisterTaskDefinitionResponse, error) {
	return doRequest[api.DeregisterTaskDefinitionRequest, api.DeregisterTaskDefinitionResponse](ctx, c, "DeregisterTaskDefinition", input)
}

// DescribeTaskDefinition describes a task definition
func (c *Client) DescribeTaskDefinition(ctx context.Context, input *api.DescribeTaskDefinitionRequest) (*api.DescribeTaskDefinitionResponse, error) {
	return doRequest[api.DescribeTaskDefinitionRequest, api.DescribeTaskDefinitionResponse](ctx, c, "DescribeTaskDefinition", input)
}

// ListTaskDefinitions lists task definitions
func (c *Client) ListTaskDefinitions(ctx context.Context, input *api.ListTaskDefinitionsRequest) (*api.ListTaskDefinitionsResponse, error) {
	return doRequest[api.ListTaskDefinitionsRequest, api.ListTaskDefinitionsResponse](ctx, c, "ListTaskDefinitions", input)
}

// CreateService creates a new service
func (c *Client) CreateService(ctx context.Context, input *api.CreateServiceRequest) (*api.CreateServiceResponse, error) {
	return doRequest[api.CreateServiceRequest, api.CreateServiceResponse](ctx, c, "CreateService", input)
}

// UpdateService updates a service
func (c *Client) UpdateService(ctx context.Context, input *api.UpdateServiceRequest) (*api.UpdateServiceResponse, error) {
	return doRequest[api.UpdateServiceRequest, api.UpdateServiceResponse](ctx, c, "UpdateService", input)
}

// DeleteService deletes a service
func (c *Client) DeleteService(ctx context.Context, input *api.DeleteServiceRequest) (*api.DeleteServiceResponse, error) {
	return doRequest[api.DeleteServiceRequest, api.DeleteServiceResponse](ctx, c, "DeleteService", input)
}

// DescribeServices describes services
func (c *Client) DescribeServices(ctx context.Context, input *api.DescribeServicesRequest) (*api.DescribeServicesResponse, error) {
	return doRequest[api.DescribeServicesRequest, api.DescribeServicesResponse](ctx, c, "DescribeServices", input)
}

// ListServices lists services
func (c *Client) ListServices(ctx context.Context, input *api.ListServicesRequest) (*api.ListServicesResponse, error) {
	return doRequest[api.ListServicesRequest, api.ListServicesResponse](ctx, c, "ListServices", input)
}

// RunTask runs a task
func (c *Client) RunTask(ctx context.Context, input *api.RunTaskRequest) (*api.RunTaskResponse, error) {
	return doRequest[api.RunTaskRequest, api.RunTaskResponse](ctx, c, "RunTask", input)
}

// StopTask stops a task
func (c *Client) StopTask(ctx context.Context, input *api.StopTaskRequest) (*api.StopTaskResponse, error) {
	return doRequest[api.StopTaskRequest, api.StopTaskResponse](ctx, c, "StopTask", input)
}

// DescribeTasks describes tasks
func (c *Client) DescribeTasks(ctx context.Context, input *api.DescribeTasksRequest) (*api.DescribeTasksResponse, error) {
	return doRequest[api.DescribeTasksRequest, api.DescribeTasksResponse](ctx, c, "DescribeTasks", input)
}

// ListTasks lists tasks
func (c *Client) ListTasks(ctx context.Context, input *api.ListTasksRequest) (*api.ListTasksResponse, error) {
	return doRequest[api.ListTasksRequest, api.ListTasksResponse](ctx, c, "ListTasks", input)
}

// doRequest performs a generic AWS API request
func doRequest[TInput any, TOutput any](ctx context.Context, c *Client, operation string, input *TInput) (*TOutput, error) {
	// Marshal input
	var body io.Reader
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Target", fmt.Sprintf("%s.%s", serviceID, operation))

	// Send request
	resp, err := c.client.DoRequest(ctx, req, serviceName)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if resp.StatusCode >= 400 {
		var errResp struct {
			Type    string `json:"__type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respData, &errResp); err == nil {
			return nil, &awsError{
				Code:       errResp.Type,
				Message:    errResp.Message,
				StatusCode: resp.StatusCode,
			}
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	// Unmarshal response
	var output TOutput
	if len(respData) > 0 {
		if err := json.Unmarshal(respData, &output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return &output, nil
}

// awsError represents an AWS API error
type awsError struct {
	Code       string
	Message    string
	StatusCode int
}

func (e *awsError) Error() string {
	return fmt.Sprintf("%s: %s (status: %d)", e.Code, e.Message, e.StatusCode)
}