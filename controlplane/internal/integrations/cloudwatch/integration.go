package cloudwatch

import (
	"context"
	"fmt"
	"strings"

	cloudwatchlogsapi "github.com/nandemo-ya/kecs/controlplane/internal/cloudwatchlogs/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"k8s.io/client-go/kubernetes"
)

// integration implements the CloudWatch Integration interface
type integration struct {
	logsClient        CloudWatchLogsClient
	kubeClient        kubernetes.Interface
	localstackManager localstack.Manager
	config            *Config
}

// NewIntegration creates a new CloudWatch integration instance
func NewIntegration(kubeClient kubernetes.Interface, localstackManager localstack.Manager, config *Config) (Integration, error) {
	if config == nil {
		config = &Config{
			LogGroupPrefix: "/ecs/",
			RetentionDays:  7,
			KubeNamespace:  "default",
		}
	}

	// Create CloudWatch Logs client configured for LocalStack
	endpoint := config.LocalStackEndpoint
	if endpoint == "" {
		// Use cluster-internal LocalStack service endpoint
		endpoint = "http://localstack.kecs-system.svc.cluster.local:4566"
	}

	logsClient := newCloudWatchLogsClient(endpoint)

	return &integration{
		logsClient:        logsClient,
		kubeClient:        kubeClient,
		localstackManager: localstackManager,
		config:            config,
	}, nil
}

// NewIntegrationWithClient creates a new CloudWatch integration with custom client (for testing)
func NewIntegrationWithClient(kubeClient kubernetes.Interface, localstackManager localstack.Manager, config *Config, logsClient CloudWatchLogsClient) Integration {
	if config == nil {
		config = &Config{
			LogGroupPrefix: "/ecs/",
			RetentionDays:  7,
			KubeNamespace:  "default",
		}
	}

	return &integration{
		logsClient:        logsClient,
		kubeClient:        kubeClient,
		localstackManager: localstackManager,
		config:            config,
	}
}

// CreateLogGroup creates a log group in LocalStack CloudWatch
func (i *integration) CreateLogGroup(groupName string) error {
	ctx := context.Background()

	// Ensure log group name has prefix
	if !strings.HasPrefix(groupName, i.config.LogGroupPrefix) {
		groupName = i.config.LogGroupPrefix + groupName
	}

	// Create log group
	_, err := i.logsClient.CreateLogGroup(ctx, &cloudwatchlogsapi.CreateLogGroupRequest{
		LogGroupName: groupName,
	})
	if err != nil {
		// Check if already exists
		if strings.Contains(err.Error(), "ResourceAlreadyExistsException") {
			logging.Debug("Log group already exists", "groupName", groupName)
			return nil
		}
		return fmt.Errorf("failed to create log group: %w", err)
	}

	// Set retention policy
	if i.config.RetentionDays > 0 {
		_, err = i.logsClient.PutRetentionPolicy(ctx, &cloudwatchlogsapi.PutRetentionPolicyRequest{
			LogGroupName:    groupName,
			RetentionInDays: i.config.RetentionDays,
		})
		if err != nil {
			logging.Warn("Failed to set retention policy for log group", "groupName", groupName, "error", err)
		}
	}

	logging.Info("Created CloudWatch log group", "groupName", groupName)
	return nil
}

// CreateLogStream creates a log stream for a container
func (i *integration) CreateLogStream(groupName, streamName string) error {
	ctx := context.Background()

	// Ensure log group name has prefix
	if !strings.HasPrefix(groupName, i.config.LogGroupPrefix) {
		groupName = i.config.LogGroupPrefix + groupName
	}

	_, err := i.logsClient.CreateLogStream(ctx, &cloudwatchlogsapi.CreateLogStreamRequest{
		LogGroupName:  groupName,
		LogStreamName: streamName,
	})
	if err != nil {
		// Check if already exists
		if strings.Contains(err.Error(), "ResourceAlreadyExistsException") {
			logging.Debug("Log stream already exists", "streamName", streamName, "groupName", groupName)
			return nil
		}
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	logging.Info("Created CloudWatch log stream", "groupName", groupName, "streamName", streamName)
	return nil
}

// DeleteLogGroup deletes a log group
func (i *integration) DeleteLogGroup(groupName string) error {
	ctx := context.Background()

	// Ensure log group name has prefix
	if !strings.HasPrefix(groupName, i.config.LogGroupPrefix) {
		groupName = i.config.LogGroupPrefix + groupName
	}

	_, err := i.logsClient.DeleteLogGroup(ctx, &cloudwatchlogsapi.DeleteLogGroupRequest{
		LogGroupName: groupName,
	})
	if err != nil {
		// Ignore if not found
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			logging.Debug("Log group not found", "groupName", groupName)
			return nil
		}
		return fmt.Errorf("failed to delete log group: %w", err)
	}

	logging.Info("Deleted CloudWatch log group", "groupName", groupName)
	return nil
}

// GetLogGroupForTask returns the log group name for a task
func (i *integration) GetLogGroupForTask(taskArn string) string {
	// Extract task family from ARN
	// Example: arn:aws:ecs:us-east-1:000000000000:task/default/1234567890
	parts := strings.Split(taskArn, "/")
	if len(parts) >= 2 {
		// For ECS tasks, we typically use the task definition family name
		// For now, we'll use a simplified approach
		return i.config.LogGroupPrefix + "kecs-tasks"
	}
	return i.config.LogGroupPrefix + "kecs-tasks"
}

// GetLogStreamForContainer returns the log stream name for a container
func (i *integration) GetLogStreamForContainer(taskArn, containerName string) string {
	// Extract task ID from ARN
	parts := strings.Split(taskArn, "/")
	taskID := "unknown"
	if len(parts) >= 2 {
		taskID = parts[len(parts)-1]
	}

	// Format: <container-name>/<task-id>
	return fmt.Sprintf("%s/%s", containerName, taskID)
}

// ConfigureContainerLogging configures logging for a container in pod spec
func (i *integration) ConfigureContainerLogging(taskArn string, containerName string, logDriver string, options map[string]string) (*LogConfiguration, error) {
	if logDriver != "awslogs" {
		return nil, fmt.Errorf("unsupported log driver: %s (only 'awslogs' is supported)", logDriver)
	}

	// Get or use provided log group and stream names
	logGroupName := options["awslogs-group"]
	if logGroupName == "" {
		logGroupName = i.GetLogGroupForTask(taskArn)
	}

	logStreamName := options["awslogs-stream-prefix"]
	if logStreamName == "" {
		logStreamName = containerName
	}

	// Ensure log group exists
	if err := i.CreateLogGroup(logGroupName); err != nil {
		return nil, fmt.Errorf("failed to create log group: %w", err)
	}

	return &LogConfiguration{
		LogGroupName:  logGroupName,
		LogStreamName: logStreamName,
		LogDriver:     logDriver,
		Options:       options,
	}, nil
}
