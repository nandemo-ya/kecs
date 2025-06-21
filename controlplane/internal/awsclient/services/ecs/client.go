package ecs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
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
func (c *Client) ListClusters(ctx context.Context, input *generated.ListClustersRequest) (*generated.ListClustersResponse, error) {
	return doRequest[generated.ListClustersRequest, generated.ListClustersResponse](ctx, c, "ListClusters", input)
}

// CreateCluster creates a new ECS cluster
func (c *Client) CreateCluster(ctx context.Context, input *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
	return doRequest[generated.CreateClusterRequest, generated.CreateClusterResponse](ctx, c, "CreateCluster", input)
}

// DeleteCluster deletes an ECS cluster
func (c *Client) DeleteCluster(ctx context.Context, input *generated.DeleteClusterRequest) (*generated.DeleteClusterResponse, error) {
	return doRequest[generated.DeleteClusterRequest, generated.DeleteClusterResponse](ctx, c, "DeleteCluster", input)
}

// DescribeClusters describes ECS clusters
func (c *Client) DescribeClusters(ctx context.Context, input *generated.DescribeClustersRequest) (*generated.DescribeClustersResponse, error) {
	return doRequest[generated.DescribeClustersRequest, generated.DescribeClustersResponse](ctx, c, "DescribeClusters", input)
}

// RegisterTaskDefinition registers a new task definition
func (c *Client) RegisterTaskDefinition(ctx context.Context, input *generated.RegisterTaskDefinitionRequest) (*generated.RegisterTaskDefinitionResponse, error) {
	return doRequest[generated.RegisterTaskDefinitionRequest, generated.RegisterTaskDefinitionResponse](ctx, c, "RegisterTaskDefinition", input)
}

// DeregisterTaskDefinition deregisters a task definition
func (c *Client) DeregisterTaskDefinition(ctx context.Context, input *generated.DeregisterTaskDefinitionRequest) (*generated.DeregisterTaskDefinitionResponse, error) {
	return doRequest[generated.DeregisterTaskDefinitionRequest, generated.DeregisterTaskDefinitionResponse](ctx, c, "DeregisterTaskDefinition", input)
}

// DescribeTaskDefinition describes a task definition
func (c *Client) DescribeTaskDefinition(ctx context.Context, input *generated.DescribeTaskDefinitionRequest) (*generated.DescribeTaskDefinitionResponse, error) {
	return doRequest[generated.DescribeTaskDefinitionRequest, generated.DescribeTaskDefinitionResponse](ctx, c, "DescribeTaskDefinition", input)
}

// ListTaskDefinitions lists task definitions
func (c *Client) ListTaskDefinitions(ctx context.Context, input *generated.ListTaskDefinitionsRequest) (*generated.ListTaskDefinitionsResponse, error) {
	return doRequest[generated.ListTaskDefinitionsRequest, generated.ListTaskDefinitionsResponse](ctx, c, "ListTaskDefinitions", input)
}

// CreateService creates a new service
func (c *Client) CreateService(ctx context.Context, input *generated.CreateServiceRequest) (*generated.CreateServiceResponse, error) {
	return doRequest[generated.CreateServiceRequest, generated.CreateServiceResponse](ctx, c, "CreateService", input)
}

// UpdateService updates a service
func (c *Client) UpdateService(ctx context.Context, input *generated.UpdateServiceRequest) (*generated.UpdateServiceResponse, error) {
	return doRequest[generated.UpdateServiceRequest, generated.UpdateServiceResponse](ctx, c, "UpdateService", input)
}

// DeleteService deletes a service
func (c *Client) DeleteService(ctx context.Context, input *generated.DeleteServiceRequest) (*generated.DeleteServiceResponse, error) {
	return doRequest[generated.DeleteServiceRequest, generated.DeleteServiceResponse](ctx, c, "DeleteService", input)
}

// DescribeServices describes services
func (c *Client) DescribeServices(ctx context.Context, input *generated.DescribeServicesRequest) (*generated.DescribeServicesResponse, error) {
	return doRequest[generated.DescribeServicesRequest, generated.DescribeServicesResponse](ctx, c, "DescribeServices", input)
}

// ListServices lists services
func (c *Client) ListServices(ctx context.Context, input *generated.ListServicesRequest) (*generated.ListServicesResponse, error) {
	return doRequest[generated.ListServicesRequest, generated.ListServicesResponse](ctx, c, "ListServices", input)
}

// RunTask runs a task
func (c *Client) RunTask(ctx context.Context, input *generated.RunTaskRequest) (*generated.RunTaskResponse, error) {
	return doRequest[generated.RunTaskRequest, generated.RunTaskResponse](ctx, c, "RunTask", input)
}

// StopTask stops a task
func (c *Client) StopTask(ctx context.Context, input *generated.StopTaskRequest) (*generated.StopTaskResponse, error) {
	return doRequest[generated.StopTaskRequest, generated.StopTaskResponse](ctx, c, "StopTask", input)
}

// DescribeTasks describes tasks
func (c *Client) DescribeTasks(ctx context.Context, input *generated.DescribeTasksRequest) (*generated.DescribeTasksResponse, error) {
	return doRequest[generated.DescribeTasksRequest, generated.DescribeTasksResponse](ctx, c, "DescribeTasks", input)
}

// ListTasks lists tasks
func (c *Client) ListTasks(ctx context.Context, input *generated.ListTasksRequest) (*generated.ListTasksResponse, error) {
	return doRequest[generated.ListTasksRequest, generated.ListTasksResponse](ctx, c, "ListTasks", input)
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