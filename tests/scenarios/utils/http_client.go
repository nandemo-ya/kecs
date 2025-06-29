package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClient uses net/http for ECS operations
type HTTPClient struct {
	endpoint string
	region   string
	client   *http.Client
}

// NewHTTPClient creates a new HTTP-based ECS client
func NewHTTPClient(endpoint string) *HTTPClient {
	return &HTTPClient{
		endpoint: endpoint,
		region:   "us-east-1", // Default region
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// executeRequest executes an HTTP request with ECS headers
func (c *HTTPClient) executeRequest(action string, payload interface{}) ([]byte, error) {
	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Debug: Log the payload being sent
	fmt.Printf("DEBUG: Sending to %s: %s\n", action, string(jsonPayload))

	// Create request
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", fmt.Sprintf("AmazonEC2ContainerServiceV20141113.%s", action))

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Debug: Log the raw response
	fmt.Printf("DEBUG: Response status: %d\n", resp.StatusCode)
	fmt.Printf("DEBUG: Raw response: %s\n", string(body))

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// CreateCluster creates a new ECS cluster
func (c *HTTPClient) CreateCluster(name string) error {
	payload := map[string]string{}
	if name != "" {
		payload["clusterName"] = name
	}
	
	_, err := c.executeRequest("CreateCluster", payload)
	return err
}

// DescribeCluster describes an ECS cluster
func (c *HTTPClient) DescribeCluster(name string) (*Cluster, error) {
	payload := map[string][]string{
		"clusters": {name},
	}
	
	output, err := c.executeRequest("DescribeClusters", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Clusters []Cluster `json:"clusters"`
		Failures []struct {
			Arn    string `json:"arn"`
			Reason string `json:"reason"`
		} `json:"failures"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Failures) > 0 {
		return nil, fmt.Errorf("cluster not found: %s", result.Failures[0].Reason)
	}

	if len(result.Clusters) == 0 {
		return nil, fmt.Errorf("no clusters returned")
	}

	return &result.Clusters[0], nil
}

// ListClusters lists all ECS clusters
func (c *HTTPClient) ListClusters() ([]string, error) {
	output, err := c.executeRequest("ListClusters", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	var result struct {
		ClusterArns []string `json:"clusterArns"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ClusterArns, nil
}

// DeleteCluster deletes an ECS cluster
func (c *HTTPClient) DeleteCluster(name string) error {
	payload := map[string]string{
		"cluster": name,
	}
	
	_, err := c.executeRequest("DeleteCluster", payload)
	return err
}

// RegisterTaskDefinition registers a task definition
func (c *HTTPClient) RegisterTaskDefinition(family string, definition string) (*TaskDefinition, error) {
	// For backward compatibility - parse the definition JSON
	var defMap map[string]interface{}
	if err := json.Unmarshal([]byte(definition), &defMap); err != nil {
		return nil, fmt.Errorf("failed to parse definition: %w", err)
	}
	
	defMap["family"] = family
	
	output, err := c.executeRequest("RegisterTaskDefinition", defMap)
	if err != nil {
		return nil, err
	}

	var result struct {
		TaskDefinition TaskDefinition `json:"taskDefinition"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.TaskDefinition, nil
}

// RegisterTaskDefinitionFromJSON registers a task definition from JSON
func (c *HTTPClient) RegisterTaskDefinitionFromJSON(jsonDefinition string) (*TaskDefinition, error) {
	var taskDef map[string]interface{}
	if err := json.Unmarshal([]byte(jsonDefinition), &taskDef); err != nil {
		return nil, fmt.Errorf("failed to parse task definition JSON: %w", err)
	}
	
	output, err := c.executeRequest("RegisterTaskDefinition", taskDef)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		TaskDefinition *TaskDefinition `json:"taskDefinition"`
	}
	
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if result.TaskDefinition == nil {
		return nil, fmt.Errorf("no task definition in response")
	}
	
	return result.TaskDefinition, nil
}

// Implement remaining methods following the same pattern...
// For brevity, I'll just add the key ones needed for the test

// UpdateService updates an ECS service desired count
func (c *HTTPClient) UpdateService(clusterName, serviceName string, desiredCount int) error {
	payload := map[string]interface{}{
		"cluster":       clusterName,
		"service":       serviceName,
		"desiredCount":  desiredCount,
	}
	
	_, err := c.executeRequest("UpdateService", payload)
	return err
}

// UpdateServiceTaskDefinition updates an ECS service task definition
func (c *HTTPClient) UpdateServiceTaskDefinition(clusterName, serviceName, taskDef string) error {
	payload := map[string]interface{}{
		"cluster":         clusterName,
		"service":         serviceName,
		"taskDefinition":  taskDef,
	}
	
	_, err := c.executeRequest("UpdateService", payload)
	return err
}

// DescribeTask describes a single task
func (c *HTTPClient) DescribeTask(clusterName, taskArn string) (*Task, error) {
	tasks, err := c.DescribeTasks(clusterName, []string{taskArn})
	if err != nil {
		return nil, err
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskArn)
	}
	
	return &tasks[0], nil
}

// DescribeTasks describes tasks in a cluster
func (c *HTTPClient) DescribeTasks(clusterName string, taskArns []string) ([]Task, error) {
	if len(taskArns) == 0 {
		return []Task{}, nil
	}

	payload := map[string]interface{}{
		"cluster": clusterName,
		"tasks":   taskArns,
	}
	
	output, err := c.executeRequest("DescribeTasks", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Tasks    []Task `json:"tasks"`
		Failures []struct {
			Arn    string `json:"arn"`
			Reason string `json:"reason"`
		} `json:"failures"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Tasks, nil
}

// Additional stub methods to satisfy interface...
// These would need to be implemented following the same pattern

func (c *HTTPClient) DescribeTaskDefinition(taskDefArn string) (*TaskDefinition, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) ListTaskDefinitions() ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) DeregisterTaskDefinition(taskDefArn string) error {
	payload := map[string]string{
		"taskDefinition": taskDefArn,
	}
	
	_, err := c.executeRequest("DeregisterTaskDefinition", payload)
	return err
}

func (c *HTTPClient) CreateService(clusterName, serviceName, taskDef string, desiredCount int) error {
	payload := map[string]interface{}{
		"cluster":        clusterName,
		"serviceName":    serviceName,
		"taskDefinition": taskDef,
		"desiredCount":   desiredCount,
	}
	
	_, err := c.executeRequest("CreateService", payload)
	return err
}

func (c *HTTPClient) DescribeService(clusterName, serviceName string) (*Service, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) ListServices(clusterName string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) DeleteService(clusterName, serviceName string) error {
	// First, scale down to 0
	if err := c.UpdateService(clusterName, serviceName, 0); err != nil {
		// Ignore error if service doesn't exist
		fmt.Printf("Note: Failed to scale down service: %v\n", err)
	}
	
	// Then delete the service
	payload := map[string]string{
		"cluster": clusterName,
		"service": serviceName,
	}
	
	_, err := c.executeRequest("DeleteService", payload)
	return err
}

func (c *HTTPClient) RunTask(clusterName, taskDefArn string, count int) (*RunTaskResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) ListTasks(clusterName string, serviceName string) ([]string, error) {
	payload := map[string]string{
		"cluster": clusterName,
	}
	
	if serviceName != "" {
		payload["serviceName"] = serviceName
	}
	
	output, err := c.executeRequest("ListTasks", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		TaskArns []string `json:"taskArns"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.TaskArns, nil
}

func (c *HTTPClient) StopTask(clusterName, taskArn, reason string) error {
	return fmt.Errorf("not implemented")
}

func (c *HTTPClient) TagResource(resourceArn string, tags map[string]string) error {
	return fmt.Errorf("not implemented")
}

func (c *HTTPClient) UntagResource(resourceArn string, tagKeys []string) error {
	return fmt.Errorf("not implemented")
}

func (c *HTTPClient) ListTagsForResource(resourceArn string) (map[string]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) PutAttributes(clusterName string, attributes []Attribute) error {
	return fmt.Errorf("not implemented")
}

func (c *HTTPClient) ListAttributes(clusterName, targetType string) ([]Attribute, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *HTTPClient) DeleteAttributes(clusterName string, attributes []Attribute) error {
	return fmt.Errorf("not implemented")
}

func (c *HTTPClient) GetLocalStackStatus(clusterName string) (string, error) {
	// KECS LocalStack status is global, not per-cluster
	url := fmt.Sprintf("%s/api/localstack/status", c.endpoint)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Running bool   `json:"running"`
		Enabled bool   `json:"enabled"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Running {
		return "healthy", nil
	} else if result.Enabled {
		return "enabled", nil
	}
	return "disabled", nil
}