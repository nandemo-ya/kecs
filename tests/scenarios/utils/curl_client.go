package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// CurlClient uses curl commands for ECS operations
type CurlClient struct {
	endpoint string
	region   string
}

// NewCurlClient creates a new curl-based ECS client
func NewCurlClient(endpoint string) *CurlClient {
	return &CurlClient{
		endpoint: endpoint,
		region:   "us-east-1", // Default region
	}
}

// executeCurl executes a curl command with ECS headers
func (c *CurlClient) executeCurl(action string, payload string) ([]byte, error) {
	cmd := exec.Command("curl", "-s", "-X", "POST",
		fmt.Sprintf("%s/v1/%s", c.endpoint, action),
		"-H", "Content-Type: application/x-amz-json-1.1",
		"-H", fmt.Sprintf("X-Amz-Target: AmazonEC2ContainerServiceV20141113.%s", action),
		"-d", payload,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("curl command failed: %w\nOutput: %s", err, output)
	}
	
	// Check if response contains error
	if strings.Contains(string(output), "error") || strings.Contains(string(output), "Error") {
		return output, fmt.Errorf("API error: %s", output)
	}
	
	return output, nil
}

// CreateCluster creates a new ECS cluster
func (c *CurlClient) CreateCluster(name string) error {
	payload := fmt.Sprintf(`{"clusterName": "%s"}`, name)
	_, err := c.executeCurl("CreateCluster", payload)
	return err
}

// DescribeCluster describes an ECS cluster
func (c *CurlClient) DescribeCluster(name string) (*Cluster, error) {
	payload := fmt.Sprintf(`{"clusters": ["%s"]}`, name)
	output, err := c.executeCurl("DescribeClusters", payload)
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
		return nil, fmt.Errorf("failed to parse response: %w\nOutput: %s", err, output)
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
func (c *CurlClient) ListClusters() ([]string, error) {
	output, err := c.executeCurl("ListClusters", "{}")
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
func (c *CurlClient) DeleteCluster(name string) error {
	payload := fmt.Sprintf(`{"cluster": "%s"}`, name)
	_, err := c.executeCurl("DeleteCluster", payload)
	return err
}

// RegisterTaskDefinition registers a new task definition
func (c *CurlClient) RegisterTaskDefinition(family string, definition string) (*TaskDefinition, error) {
	output, err := c.executeCurl("RegisterTaskDefinition", definition)
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

// DescribeTaskDefinition describes a task definition
func (c *CurlClient) DescribeTaskDefinition(taskDefArn string) (*TaskDefinition, error) {
	payload := fmt.Sprintf(`{"taskDefinition": "%s"}`, taskDefArn)
	output, err := c.executeCurl("DescribeTaskDefinition", payload)
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

// ListTaskDefinitions lists all task definitions
func (c *CurlClient) ListTaskDefinitions() ([]string, error) {
	output, err := c.executeCurl("ListTaskDefinitions", "{}")
	if err != nil {
		return nil, err
	}

	var result struct {
		TaskDefinitionArns []string `json:"taskDefinitionArns"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.TaskDefinitionArns, nil
}

// DeregisterTaskDefinition deregisters a task definition
func (c *CurlClient) DeregisterTaskDefinition(taskDefArn string) error {
	payload := fmt.Sprintf(`{"taskDefinition": "%s"}`, taskDefArn)
	_, err := c.executeCurl("DeregisterTaskDefinition", payload)
	return err
}

// CreateService creates a new ECS service
func (c *CurlClient) CreateService(clusterName, serviceName, taskDef string, desiredCount int) error {
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"serviceName": "%s",
		"taskDefinition": "%s",
		"desiredCount": %d
	}`, clusterName, serviceName, taskDef, desiredCount)
	
	_, err := c.executeCurl("CreateService", payload)
	return err
}

// DescribeService describes an ECS service
func (c *CurlClient) DescribeService(clusterName, serviceName string) (*Service, error) {
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"services": ["%s"]
	}`, clusterName, serviceName)
	
	output, err := c.executeCurl("DescribeServices", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Services []Service `json:"services"`
		Failures []struct {
			Arn    string `json:"arn"`
			Reason string `json:"reason"`
		} `json:"failures"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Failures) > 0 {
		return nil, fmt.Errorf("service not found: %s", result.Failures[0].Reason)
	}

	if len(result.Services) == 0 {
		return nil, fmt.Errorf("no services returned")
	}

	return &result.Services[0], nil
}

// ListServices lists all services in a cluster
func (c *CurlClient) ListServices(clusterName string) ([]string, error) {
	payload := fmt.Sprintf(`{"cluster": "%s"}`, clusterName)
	output, err := c.executeCurl("ListServices", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		ServiceArns []string `json:"serviceArns"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ServiceArns, nil
}

// UpdateService updates an ECS service
func (c *CurlClient) UpdateService(clusterName, serviceName string, desiredCount *int, taskDef string) error {
	payload := fmt.Sprintf(`{"cluster": "%s", "service": "%s"`, clusterName, serviceName)
	
	if desiredCount != nil {
		payload += fmt.Sprintf(`, "desiredCount": %d`, *desiredCount)
	}
	
	if taskDef != "" {
		payload += fmt.Sprintf(`, "taskDefinition": "%s"`, taskDef)
	}
	
	payload += "}"
	
	_, err := c.executeCurl("UpdateService", payload)
	return err
}

// DeleteService deletes an ECS service
func (c *CurlClient) DeleteService(clusterName, serviceName string) error {
	// First, scale down to 0
	zero := 0
	if err := c.UpdateService(clusterName, serviceName, &zero, ""); err != nil {
		return fmt.Errorf("failed to scale down service: %w", err)
	}
	
	// Then delete the service
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"service": "%s"
	}`, clusterName, serviceName)
	
	_, err := c.executeCurl("DeleteService", payload)
	return err
}

// RunTask runs a task on ECS
func (c *CurlClient) RunTask(clusterName, taskDefArn string, count int) (*RunTaskResponse, error) {
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"taskDefinition": "%s",
		"count": %d
	}`, clusterName, taskDefArn, count)
	
	output, err := c.executeCurl("RunTask", payload)
	if err != nil {
		return nil, err
	}

	var result RunTaskResponse
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DescribeTasks describes tasks in a cluster
func (c *CurlClient) DescribeTasks(clusterName string, taskArns []string) ([]Task, error) {
	if len(taskArns) == 0 {
		return []Task{}, nil
	}

	arnsJSON, _ := json.Marshal(taskArns)
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"tasks": %s
	}`, clusterName, arnsJSON)
	
	output, err := c.executeCurl("DescribeTasks", payload)
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

// ListTasks lists tasks in a cluster
func (c *CurlClient) ListTasks(clusterName string, serviceName string) ([]string, error) {
	payload := fmt.Sprintf(`{"cluster": "%s"`, clusterName)
	
	if serviceName != "" {
		payload += fmt.Sprintf(`, "serviceName": "%s"`, serviceName)
	}
	
	payload += "}"
	
	output, err := c.executeCurl("ListTasks", payload)
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

// StopTask stops a running task
func (c *CurlClient) StopTask(clusterName, taskArn, reason string) error {
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"task": "%s",
		"reason": "%s"
	}`, clusterName, taskArn, reason)
	
	_, err := c.executeCurl("StopTask", payload)
	return err
}

// TagResource adds tags to a resource
func (c *CurlClient) TagResource(resourceArn string, tags map[string]string) error {
	tagList := []map[string]string{}
	for key, value := range tags {
		tagList = append(tagList, map[string]string{
			"key":   key,
			"value": value,
		})
	}
	
	tagsJSON, _ := json.Marshal(tagList)
	payload := fmt.Sprintf(`{
		"resourceArn": "%s",
		"tags": %s
	}`, resourceArn, tagsJSON)
	
	_, err := c.executeCurl("TagResource", payload)
	return err
}

// UntagResource removes tags from a resource
func (c *CurlClient) UntagResource(resourceArn string, tagKeys []string) error {
	keysJSON, _ := json.Marshal(tagKeys)
	payload := fmt.Sprintf(`{
		"resourceArn": "%s",
		"tagKeys": %s
	}`, resourceArn, keysJSON)
	
	_, err := c.executeCurl("UntagResource", payload)
	return err
}

// ListTagsForResource lists tags for a resource
func (c *CurlClient) ListTagsForResource(resourceArn string) (map[string]string, error) {
	payload := fmt.Sprintf(`{"resourceArn": "%s"}`, resourceArn)
	output, err := c.executeCurl("ListTagsForResource", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Tags []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"tags"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	tags := make(map[string]string)
	for _, tag := range result.Tags {
		tags[tag.Key] = tag.Value
	}

	return tags, nil
}

// PutAttributes puts attributes in a cluster
func (c *CurlClient) PutAttributes(clusterName string, attributes []Attribute) error {
	attrsJSON, _ := json.Marshal(attributes)
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"attributes": %s
	}`, clusterName, attrsJSON)
	
	_, err := c.executeCurl("PutAttributes", payload)
	return err
}

// ListAttributes lists attributes in a cluster
func (c *CurlClient) ListAttributes(clusterName, targetType string) ([]Attribute, error) {
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"targetType": "%s"
	}`, clusterName, targetType)
	
	output, err := c.executeCurl("ListAttributes", payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Attributes []Attribute `json:"attributes"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Attributes, nil
}

// DeleteAttributes deletes attributes from a cluster
func (c *CurlClient) DeleteAttributes(clusterName string, attributes []Attribute) error {
	attrsJSON, _ := json.Marshal(attributes)
	payload := fmt.Sprintf(`{
		"cluster": "%s",
		"attributes": %s
	}`, clusterName, attrsJSON)
	
	_, err := c.executeCurl("DeleteAttributes", payload)
	return err
}

// GetLocalStackStatus gets the LocalStack status (KECS-specific)
func (c *CurlClient) GetLocalStackStatus(clusterName string) (string, error) {
	// This would be a custom KECS endpoint
	url := fmt.Sprintf("%s/localstack/status?cluster=%s", c.endpoint, clusterName)
	
	args := []string{
		"-s", "-X", "GET",
		url,
	}

	output, err := c.runCommand(args...)
	if err != nil {
		return "", err
	}

	var result struct {
		Status string `json:"status"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Status, nil
}