package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	generated_v2 "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
)

// GeneratedClient uses generated types for ECS operations
type GeneratedClient struct {
	endpoint string
	region   string
}

// NewGeneratedClient creates a new client using generated types
func NewGeneratedClient(endpoint string) *GeneratedClient {
	return &GeneratedClient{
		endpoint: endpoint,
		region:   "us-east-1",
	}
}

// doRequest performs a request to KECS API
func (c *GeneratedClient) doRequest(action string, request interface{}) ([]byte, error) {
	// Marshal request using generated types
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/", c.endpoint)
	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", fmt.Sprintf("AmazonEC2ContainerServiceV20141113.%s", action))

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: status=%d, body=%s", resp.StatusCode, respBody)
	}

	return respBody, nil
}

// CreateCluster creates a new ECS cluster
func (c *GeneratedClient) CreateCluster(name string) error {
	req := generated_v2.CreateClusterRequest{
		ClusterName: &name,
	}
	
	_, err := c.doRequest("CreateCluster", req)
	return err
}

// DescribeCluster describes an ECS cluster
func (c *GeneratedClient) DescribeCluster(name string) (*Cluster, error) {
	req := generated_v2.DescribeClustersRequest{
		Clusters: []string{name},
	}
	
	output, err := c.doRequest("DescribeClusters", req)
	if err != nil {
		return nil, err
	}

	var resp generated_v2.DescribeClustersResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(resp.Failures) > 0 {
		return nil, fmt.Errorf("cluster not found: %s", *resp.Failures[0].Reason)
	}

	if len(resp.Clusters) == 0 {
		return nil, fmt.Errorf("no clusters returned")
	}

	// Convert generated type to test type
	genCluster := resp.Clusters[0]
	cluster := &Cluster{
		ClusterArn:                        derefString(genCluster.ClusterArn),
		ClusterName:                       derefString(genCluster.ClusterName),
		Status:                            derefString(genCluster.Status),
		RegisteredContainerInstancesCount: int(derefInt32(genCluster.RegisteredContainerInstancesCount)),
		RunningTasksCount:                 int(derefInt32(genCluster.RunningTasksCount)),
		PendingTasksCount:                 int(derefInt32(genCluster.PendingTasksCount)),
		ActiveServicesCount:               int(derefInt32(genCluster.ActiveServicesCount)),
	}

	// Convert settings
	if genCluster.Settings != nil {
		cluster.Settings = make([]ClusterSetting, len(genCluster.Settings))
		for i, s := range genCluster.Settings {
			cluster.Settings[i] = ClusterSetting{
				Name:  fmt.Sprintf("%v", derefInterface(s.Name)),
				Value: derefString(s.Value),
			}
		}
	}

	// Convert tags
	if genCluster.Tags != nil {
		cluster.Tags = make(map[string]string)
		for _, tag := range genCluster.Tags {
			if tag.Key != nil && tag.Value != nil {
				cluster.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return cluster, nil
}

// ListClusters lists all ECS clusters
func (c *GeneratedClient) ListClusters() ([]string, error) {
	req := generated_v2.ListClustersRequest{}
	
	output, err := c.doRequest("ListClusters", req)
	if err != nil {
		return nil, err
	}

	var resp generated_v2.ListClustersResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return resp.ClusterArns, nil
}

// DeleteCluster deletes an ECS cluster
func (c *GeneratedClient) DeleteCluster(name string) error {
	req := generated_v2.DeleteClusterRequest{
		Cluster: name,
	}
	
	_, err := c.doRequest("DeleteCluster", req)
	return err
}

// RegisterTaskDefinition registers a task definition
func (c *GeneratedClient) RegisterTaskDefinition(family string, definition string) (*TaskDefinition, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("RegisterTaskDefinition not yet implemented with generated types")
}

// DescribeTaskDefinition describes a task definition
func (c *GeneratedClient) DescribeTaskDefinition(taskDefArn string) (*TaskDefinition, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("DescribeTaskDefinition not yet implemented with generated types")
}

// ListTaskDefinitions lists task definitions
func (c *GeneratedClient) ListTaskDefinitions() ([]string, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("ListTaskDefinitions not yet implemented with generated types")
}

// DeregisterTaskDefinition deregisters a task definition
func (c *GeneratedClient) DeregisterTaskDefinition(taskDefArn string) error {
	// TODO: Implement using generated types
	return fmt.Errorf("DeregisterTaskDefinition not yet implemented with generated types")
}

// CreateService creates an ECS service
func (c *GeneratedClient) CreateService(clusterName, serviceName, taskDef string, desiredCount int) error {
	// TODO: Implement using generated types
	return fmt.Errorf("CreateService not yet implemented with generated types")
}

// DescribeService describes an ECS service
func (c *GeneratedClient) DescribeService(clusterName, serviceName string) (*Service, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("DescribeService not yet implemented with generated types")
}

// ListServices lists services in a cluster
func (c *GeneratedClient) ListServices(clusterName string) ([]string, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("ListServices not yet implemented with generated types")
}

// UpdateService updates an ECS service
func (c *GeneratedClient) UpdateService(clusterName, serviceName string, desiredCount *int, taskDef string) error {
	// TODO: Implement using generated types
	return fmt.Errorf("UpdateService not yet implemented with generated types")
}

// DeleteService deletes an ECS service
func (c *GeneratedClient) DeleteService(clusterName, serviceName string) error {
	// TODO: Implement using generated types
	return fmt.Errorf("DeleteService not yet implemented with generated types")
}

// RunTask runs a task
func (c *GeneratedClient) RunTask(clusterName, taskDefArn string, count int) (*RunTaskResponse, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("RunTask not yet implemented with generated types")
}

// DescribeTasks describes tasks
func (c *GeneratedClient) DescribeTasks(clusterName string, taskArns []string) ([]Task, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("DescribeTasks not yet implemented with generated types")
}

// ListTasks lists tasks
func (c *GeneratedClient) ListTasks(clusterName string, serviceName string) ([]string, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("ListTasks not yet implemented with generated types")
}

// StopTask stops a task
func (c *GeneratedClient) StopTask(clusterName, taskArn, reason string) error {
	// TODO: Implement using generated types
	return fmt.Errorf("StopTask not yet implemented with generated types")
}

// TagResource tags a resource
func (c *GeneratedClient) TagResource(resourceArn string, tags map[string]string) error {
	// TODO: Implement using generated types
	return fmt.Errorf("TagResource not yet implemented with generated types")
}

// UntagResource untags a resource
func (c *GeneratedClient) UntagResource(resourceArn string, tagKeys []string) error {
	// TODO: Implement using generated types
	return fmt.Errorf("UntagResource not yet implemented with generated types")
}

// ListTagsForResource lists tags for a resource
func (c *GeneratedClient) ListTagsForResource(resourceArn string) (map[string]string, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("ListTagsForResource not yet implemented with generated types")
}

// PutAttributes puts attributes
func (c *GeneratedClient) PutAttributes(clusterName string, attributes []Attribute) error {
	// TODO: Implement using generated types
	return fmt.Errorf("PutAttributes not yet implemented with generated types")
}

// ListAttributes lists attributes
func (c *GeneratedClient) ListAttributes(clusterName, targetType string) ([]Attribute, error) {
	// TODO: Implement using generated types
	return nil, fmt.Errorf("ListAttributes not yet implemented with generated types")
}

// DeleteAttributes deletes attributes
func (c *GeneratedClient) DeleteAttributes(clusterName string, attributes []Attribute) error {
	// TODO: Implement using generated types
	return fmt.Errorf("DeleteAttributes not yet implemented with generated types")
}

// GetLocalStackStatus gets LocalStack status
func (c *GeneratedClient) GetLocalStackStatus(clusterName string) (string, error) {
	// This is specific to LocalStack integration
	// For now, return not implemented
	return "", fmt.Errorf("GetLocalStackStatus not yet implemented with generated types")
}

// Helper functions
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func derefInterface(i *interface{}) interface{} {
	if i == nil {
		return nil
	}
	return *i
}

// extractNameFromArn extracts the name from an ARN
func extractNameFromArn(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}