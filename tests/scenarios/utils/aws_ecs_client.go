package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ECSClient wraps AWS CLI commands for ECS operations
type ECSClient struct {
	endpoint string
	region   string
}

// NewECSClient creates a new ECS client
func NewECSClient(endpoint string) *ECSClient {
	return &ECSClient{
		endpoint: endpoint,
		region:   "us-east-1", // Default region
	}
}

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                      string `json:"clusterArn"`
	ClusterName                     string `json:"clusterName"`
	Status                          string `json:"status"`
	RegisteredContainerInstancesCount int   `json:"registeredContainerInstancesCount"`
	RunningTasksCount               int    `json:"runningTasksCount"`
	PendingTasksCount               int    `json:"pendingTasksCount"`
	ActiveServicesCount             int    `json:"activeServicesCount"`
}

// CreateCluster creates a new ECS cluster
func (c *ECSClient) CreateCluster(name string) error {
	// Use curl directly as AWS CLI has issues with custom endpoints
	payload := fmt.Sprintf(`{"clusterName": "%s"}`, name)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		fmt.Sprintf("%s/v1/CreateCluster", c.endpoint),
		"-H", "Content-Type: application/x-amz-json-1.1",
		"-H", "X-Amz-Target: AmazonEC2ContainerServiceV20141113.CreateCluster",
		"-d", payload,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w\nOutput: %s", err, output)
	}
	
	// Check if response contains error
	if strings.Contains(string(output), "error") || strings.Contains(string(output), "Error") {
		return fmt.Errorf("API error: %s", output)
	}
	
	return nil
}

// DescribeCluster describes an ECS cluster
func (c *ECSClient) DescribeCluster(name string) (*Cluster, error) {
	// Use curl directly
	payload := fmt.Sprintf(`{"clusters": ["%s"]}`, name)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		fmt.Sprintf("%s/v1/DescribeClusters", c.endpoint),
		"-H", "Content-Type: application/x-amz-json-1.1",
		"-H", "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DescribeClusters",
		"-d", payload,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to describe cluster: %w", err)
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
func (c *ECSClient) ListClusters() ([]string, error) {
	// Use curl directly
	cmd := exec.Command("curl", "-s", "-X", "POST",
		fmt.Sprintf("%s/v1/ListClusters", c.endpoint),
		"-H", "Content-Type: application/x-amz-json-1.1",
		"-H", "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters",
		"-d", "{}",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	var result struct {
		ClusterArns []string `json:"clusterArns"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w\nOutput: %s", err, output)
	}

	// Extract cluster names from ARNs
	names := make([]string, 0, len(result.ClusterArns))
	for _, arn := range result.ClusterArns {
		parts := strings.Split(arn, "/")
		if len(parts) > 0 {
			names = append(names, parts[len(parts)-1])
		}
	}

	return names, nil
}

// DeleteCluster deletes an ECS cluster
func (c *ECSClient) DeleteCluster(name string) error {
	// Use curl directly
	payload := fmt.Sprintf(`{"cluster": "%s"}`, name)
	cmd := exec.Command("curl", "-s", "-X", "POST",
		fmt.Sprintf("%s/v1/DeleteCluster", c.endpoint),
		"-H", "Content-Type: application/x-amz-json-1.1",
		"-H", "X-Amz-Target: AmazonEC2ContainerServiceV20141113.DeleteCluster",
		"-d", payload,
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w\nOutput: %s", err, output)
	}
	return nil
}

// RegisterTaskDefinition registers a new task definition
func (c *ECSClient) RegisterTaskDefinition(jsonFile string) error {
	// Read task definition from file
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read task definition file: %w", err)
	}

	output, err := c.runCommand("register-task-definition",
		"--cli-input-json", string(data),
	)
	if err != nil {
		return fmt.Errorf("failed to register task definition: %w\nOutput: %s", err, output)
	}
	return nil
}

// CreateService creates a new ECS service
func (c *ECSClient) CreateService(cluster, serviceName, taskDef string, desiredCount int) error {
	output, err := c.runCommand("create-service",
		"--cluster", cluster,
		"--service-name", serviceName,
		"--task-definition", taskDef,
		"--desired-count", fmt.Sprintf("%d", desiredCount),
		"--launch-type", "FARGATE",
		"--network-configuration", `awsvpcConfiguration={subnets=[subnet-12345],securityGroups=[sg-12345],assignPublicIp=ENABLED}`,
	)
	if err != nil {
		return fmt.Errorf("failed to create service: %w\nOutput: %s", err, output)
	}
	return nil
}

// runCommand executes an AWS ECS CLI command
func (c *ECSClient) runCommand(args ...string) (string, error) {
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
	return string(output), err
}