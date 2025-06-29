package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// AWSCLIClient uses AWS CLI for ECS operations
type AWSCLIClient struct {
	endpoint string
	region   string
}

// NewAWSCLIClient creates a new AWS CLI-based ECS client
func NewAWSCLIClient(endpoint string) *AWSCLIClient {
	return &AWSCLIClient{
		endpoint: endpoint,
		region:   "us-east-1", // Default region
	}
}

// runCommand executes an AWS ECS CLI command
func (c *AWSCLIClient) runCommand(args ...string) ([]byte, error) {
	// Build command arguments
	cmdArgs := []string{"ecs"}
	cmdArgs = append(cmdArgs, args...)
	cmdArgs = append(cmdArgs,
		"--endpoint-url", c.endpoint,
		"--region", c.region,
		"--no-verify-ssl",
		"--output", "json",
	)

	// Set AWS credentials (dummy values for local testing)
	cmd := exec.Command("aws", cmdArgs...)
	// Clear environment to avoid config conflicts
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"AWS_ACCESS_KEY_ID=dummy",
		"AWS_SECRET_ACCESS_KEY=dummy",
		"AWS_SESSION_TOKEN=dummy",
		"AWS_DEFAULT_REGION=" + c.region,
		"AWS_CONFIG_FILE=/dev/null",
		"AWS_SHARED_CREDENTIALS_FILE=/dev/null",
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the output contains error information
		if len(output) > 0 {
			return output, fmt.Errorf("AWS CLI command failed: %w\nOutput: %s", err, output)
		}
		return nil, fmt.Errorf("AWS CLI command failed: %w", err)
	}

	return output, nil
}

// CreateCluster creates a new ECS cluster
func (c *AWSCLIClient) CreateCluster(name string) error {
	args := []string{"create-cluster"}
	if name != "" {
		args = append(args, "--cluster-name", name)
	}
	output, err := c.runCommand(args...)
	if err != nil {
		// AWS CLI returns exit code 255 for validation errors
		// Parse the output to get the actual error message
		if len(output) > 0 {
			// Try AWS error format
			var awsError struct {
				Type    string `json:"__type"`
				Message string `json:"message"`
			}
			if jsonErr := json.Unmarshal(output, &awsError); jsonErr == nil && awsError.Message != "" {
				return fmt.Errorf(awsError.Message)
			}
			
			// Try CLI error format
			var errorResponse struct {
				Error struct {
					Code    string `json:"Code"`
					Message string `json:"Message"`
				} `json:"error"`
			}
			if jsonErr := json.Unmarshal(output, &errorResponse); jsonErr == nil && errorResponse.Error.Message != "" {
				return fmt.Errorf(errorResponse.Error.Message)
			}
			// If not JSON, return the raw output
			return fmt.Errorf(string(output))
		}
		return err
	}
	return nil
}

// DescribeCluster describes an ECS cluster
func (c *AWSCLIClient) DescribeCluster(name string) (*Cluster, error) {
	output, err := c.runCommand("describe-clusters", "--clusters", name)
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
func (c *AWSCLIClient) ListClusters() ([]string, error) {
	output, err := c.runCommand("list-clusters")
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
func (c *AWSCLIClient) DeleteCluster(name string) error {
	_, err := c.runCommand("delete-cluster", "--cluster", name)
	return err
}

// RegisterTaskDefinition registers a new task definition
func (c *AWSCLIClient) RegisterTaskDefinition(family string, definition string) (*TaskDefinition, error) {
	// AWS CLI expects individual parameters, not JSON
	// We need to parse the JSON and convert to CLI arguments
	var def map[string]interface{}
	if err := json.Unmarshal([]byte(definition), &def); err != nil {
		return nil, fmt.Errorf("failed to parse task definition: %w", err)
	}

	// Create a temporary file for the task definition
	tmpFile, err := os.CreateTemp("", "taskdef-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(definition)); err != nil {
		return nil, fmt.Errorf("failed to write task definition: %w", err)
	}
	tmpFile.Close()

	// Use CLI input from file
	output, err := c.runCommand("register-task-definition", "--cli-input-json", "file://"+tmpFile.Name())
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
func (c *AWSCLIClient) DescribeTaskDefinition(taskDefArn string) (*TaskDefinition, error) {
	output, err := c.runCommand("describe-task-definition", "--task-definition", taskDefArn)
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
func (c *AWSCLIClient) ListTaskDefinitions() ([]string, error) {
	output, err := c.runCommand("list-task-definitions")
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
func (c *AWSCLIClient) DeregisterTaskDefinition(taskDefArn string) error {
	_, err := c.runCommand("deregister-task-definition", "--task-definition", taskDefArn)
	return err
}

// CreateService creates a new ECS service
func (c *AWSCLIClient) CreateService(clusterName, serviceName, taskDef string, desiredCount int) error {
	_, err := c.runCommand("create-service",
		"--cluster", clusterName,
		"--service-name", serviceName,
		"--task-definition", taskDef,
		"--desired-count", fmt.Sprintf("%d", desiredCount),
	)
	return err
}

// DescribeService describes an ECS service
func (c *AWSCLIClient) DescribeService(clusterName, serviceName string) (*Service, error) {
	output, err := c.runCommand("describe-services",
		"--cluster", clusterName,
		"--services", serviceName,
	)
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
func (c *AWSCLIClient) ListServices(clusterName string) ([]string, error) {
	output, err := c.runCommand("list-services", "--cluster", clusterName)
	if err != nil {
		return nil, err
	}

	// Handle empty response
	if len(output) == 0 {
		return []string{}, nil
	}

	var result struct {
		ServiceArns []string `json:"serviceArns"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.ServiceArns, nil
}

// UpdateService updates an ECS service desired count
func (c *AWSCLIClient) UpdateService(clusterName, serviceName string, desiredCount int) error {
	args := []string{"update-service", "--cluster", clusterName, "--service", serviceName, "--desired-count", fmt.Sprintf("%d", desiredCount)}
	_, err := c.runCommand(args...)
	return err
}

// UpdateServiceTaskDefinition updates an ECS service task definition
func (c *AWSCLIClient) UpdateServiceTaskDefinition(clusterName, serviceName, taskDef string) error {
	args := []string{"update-service", "--cluster", clusterName, "--service", serviceName, "--task-definition", taskDef}
	_, err := c.runCommand(args...)
	return err
}

// DeleteService deletes an ECS service
func (c *AWSCLIClient) DeleteService(clusterName, serviceName string) error {
	// First, scale down to 0
	if err := c.UpdateService(clusterName, serviceName, 0); err != nil {
		return fmt.Errorf("failed to scale down service: %w", err)
	}
	
	// Then delete the service
	_, err := c.runCommand("delete-service",
		"--cluster", clusterName,
		"--service", serviceName,
		"--force",
	)
	return err
}

// RunTask runs a task on ECS
func (c *AWSCLIClient) RunTask(clusterName, taskDefArn string, count int) (*RunTaskResponse, error) {
	output, err := c.runCommand("run-task",
		"--cluster", clusterName,
		"--task-definition", taskDefArn,
		"--count", fmt.Sprintf("%d", count),
	)
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
func (c *AWSCLIClient) DescribeTasks(clusterName string, taskArns []string) ([]Task, error) {
	if len(taskArns) == 0 {
		return []Task{}, nil
	}

	args := []string{"describe-tasks", "--cluster", clusterName, "--tasks"}
	args = append(args, taskArns...)
	
	output, err := c.runCommand(args...)
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
func (c *AWSCLIClient) ListTasks(clusterName string, serviceName string) ([]string, error) {
	args := []string{"list-tasks", "--cluster", clusterName}
	
	if serviceName != "" {
		args = append(args, "--service-name", serviceName)
	}
	
	output, err := c.runCommand(args...)
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
func (c *AWSCLIClient) StopTask(clusterName, taskArn, reason string) error {
	_, err := c.runCommand("stop-task",
		"--cluster", clusterName,
		"--task", taskArn,
		"--reason", reason,
	)
	return err
}

// TagResource adds tags to a resource
func (c *AWSCLIClient) TagResource(resourceArn string, tags map[string]string) error {
	tagList := []string{}
	for key, value := range tags {
		tagList = append(tagList, fmt.Sprintf("key=%s,value=%s", key, value))
	}
	
	args := []string{"tag-resource", "--resource-arn", resourceArn, "--tags"}
	args = append(args, tagList...)
	
	_, err := c.runCommand(args...)
	return err
}

// UntagResource removes tags from a resource
func (c *AWSCLIClient) UntagResource(resourceArn string, tagKeys []string) error {
	args := []string{"untag-resource", "--resource-arn", resourceArn, "--tag-keys"}
	args = append(args, tagKeys...)
	
	_, err := c.runCommand(args...)
	return err
}

// ListTagsForResource lists tags for a resource
func (c *AWSCLIClient) ListTagsForResource(resourceArn string) (map[string]string, error) {
	output, err := c.runCommand("list-tags-for-resource", "--resource-arn", resourceArn)
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
func (c *AWSCLIClient) PutAttributes(clusterName string, attributes []Attribute) error {
	// AWS CLI doesn't have a direct put-attributes command
	// This would need to be implemented differently or use the API directly
	return fmt.Errorf("put-attributes not supported via AWS CLI")
}

// ListAttributes lists attributes in a cluster
func (c *AWSCLIClient) ListAttributes(clusterName, targetType string) ([]Attribute, error) {
	// AWS CLI doesn't have a direct list-attributes command
	// This would need to be implemented differently or use the API directly
	return nil, fmt.Errorf("list-attributes not supported via AWS CLI")
}

// DeleteAttributes deletes attributes from a cluster
func (c *AWSCLIClient) DeleteAttributes(clusterName string, attributes []Attribute) error {
	// AWS CLI doesn't have a direct delete-attributes command
	// This would need to be implemented differently or use the API directly
	return fmt.Errorf("delete-attributes not supported via AWS CLI")
}

// GetLocalStackStatus gets the LocalStack status
func (c *AWSCLIClient) GetLocalStackStatus(clusterName string) (string, error) {
	// This is a KECS-specific API endpoint
	url := fmt.Sprintf("%s/api/localstack/status", c.endpoint)
	
	cmd := exec.Command("curl", "-s", "-X", "GET", url)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get LocalStack status: %w\nOutput: %s", err, output)
	}

	var result struct {
		Running bool   `json:"running"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse LocalStack status: %w", err)
	}

	if result.Running {
		return "healthy", nil
	} else if result.Enabled {
		return "enabled", nil
	}
	return "disabled", nil
}

// UpdateClusterSettings updates cluster settings
func (c *AWSCLIClient) UpdateClusterSettings(clusterName string, settings []map[string]string) error {
	args := []string{"update-cluster-settings", "--cluster", clusterName, "--settings"}
	for _, setting := range settings {
		args = append(args, fmt.Sprintf("name=%s,value=%s", setting["name"], setting["value"]))
	}
	_, err := c.runCommand(args...)
	return err
}

// UpdateCluster updates a cluster configuration
func (c *AWSCLIClient) UpdateCluster(clusterName string, configuration map[string]interface{}) error {
	args := []string{"update-cluster", "--cluster", clusterName}
	
	if config, ok := configuration["configuration"]; ok {
		configJSON, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration: %w", err)
		}
		args = append(args, "--configuration", string(configJSON))
	}
	
	_, err := c.runCommand(args...)
	return err
}

// PutClusterCapacityProviders sets capacity providers on a cluster
func (c *AWSCLIClient) PutClusterCapacityProviders(clusterName string, providers []string, strategy []map[string]interface{}) error {
	args := []string{"put-cluster-capacity-providers", "--cluster", clusterName}
	
	if len(providers) > 0 {
		args = append(args, "--capacity-providers")
		args = append(args, providers...)
	}
	
	if len(strategy) > 0 {
		// Convert strategy to CLI format
		for _, s := range strategy {
			strategyStr := fmt.Sprintf("capacityProvider=%s", s["capacityProvider"])
			if weight, ok := s["weight"]; ok {
				strategyStr += fmt.Sprintf(",weight=%v", weight)
			}
			if base, ok := s["base"]; ok {
				strategyStr += fmt.Sprintf(",base=%v", base)
			}
			args = append(args, "--default-capacity-provider-strategy", strategyStr)
		}
	}
	
	_, err := c.runCommand(args...)
	return err
}

// DescribeClustersWithInclude describes clusters with additional information
func (c *AWSCLIClient) DescribeClustersWithInclude(clusters []string, include []string) ([]Cluster, error) {
	args := []string{"describe-clusters"}
	
	if len(clusters) > 0 {
		args = append(args, "--clusters")
		args = append(args, clusters...)
	}
	
	if len(include) > 0 {
		args = append(args, "--include")
		args = append(args, include...)
	}
	
	output, err := c.runCommand(args...)
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
		return nil, fmt.Errorf("describe clusters failed: %s", result.Failures[0].Reason)
	}
	
	return result.Clusters, nil
}

// ListClustersWithPagination lists clusters with pagination support
func (c *AWSCLIClient) ListClustersWithPagination(maxResults int, nextToken string) (clusters []string, newNextToken string, err error) {
	args := []string{"list-clusters"}
	
	if maxResults > 0 {
		args = append(args, "--max-results", fmt.Sprintf("%d", maxResults))
	}
	
	if nextToken != "" {
		args = append(args, "--next-token", nextToken)
	}
	
	output, err := c.runCommand(args...)
	if err != nil {
		return nil, "", err
	}
	
	var result struct {
		ClusterArns []string `json:"clusterArns"`
		NextToken   string   `json:"nextToken"`
	}
	
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, "", fmt.Errorf("failed to parse response: %w", err)
	}
	
	return result.ClusterArns, result.NextToken, nil
}

// RegisterTaskDefinitionFromJSON registers a task definition from JSON
func (c *AWSCLIClient) RegisterTaskDefinitionFromJSON(jsonDefinition string) (*TaskDefinition, error) {
	// Create temporary file for the JSON definition
	tmpFile, err := os.CreateTemp("", "taskdef-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	
	if _, err := tmpFile.WriteString(jsonDefinition); err != nil {
		return nil, fmt.Errorf("failed to write task definition: %w", err)
	}
	tmpFile.Close()
	
	output, err := c.runCommand("register-task-definition", "--cli-input-json", fmt.Sprintf("file://%s", tmpFile.Name()))
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

// DescribeTask describes a single task
func (c *AWSCLIClient) DescribeTask(clusterName, taskArn string) (*Task, error) {
	tasks, err := c.DescribeTasks(clusterName, []string{taskArn})
	if err != nil {
		return nil, err
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("task not found: %s", taskArn)
	}
	
	return &tasks[0], nil
}