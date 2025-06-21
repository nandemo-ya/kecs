package cloudwatch

import (
	"context"
	"fmt"
	"strings"

	cloudwatchlogsapi "github.com/nandemo-ya/kecs/controlplane/internal/cloudwatchlogs/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
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

	// TODO: Create CloudWatch Logs client using generated types
	// For now, return an error indicating migration is in progress
	return nil, fmt.Errorf("CloudWatch integration migration to generated types in progress")
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
			klog.V(2).Infof("Log group %s already exists", groupName)
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
			klog.Warningf("Failed to set retention policy for log group %s: %v", groupName, err)
		}
	}

	klog.Infof("Created CloudWatch log group: %s", groupName)
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
			klog.V(2).Infof("Log stream %s already exists in group %s", streamName, groupName)
			return nil
		}
		return fmt.Errorf("failed to create log stream: %w", err)
	}

	klog.Infof("Created CloudWatch log stream: %s/%s", groupName, streamName)
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
			klog.V(2).Infof("Log group %s not found", groupName)
			return nil
		}
		return fmt.Errorf("failed to delete log group: %w", err)
	}

	klog.Infof("Deleted CloudWatch log group: %s", groupName)
	return nil
}

// GetLogGroupForTask returns the log group name for a task
func (i *integration) GetLogGroupForTask(taskArn string) string {
	// Extract task family from ARN
	// Example: arn:aws:ecs:us-east-1:123456789012:task/default/1234567890
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

	// Create FluentBit configuration for CloudWatch
	fluentBitConfig := i.generateFluentBitConfig(logGroupName, logStreamName, options)

	return &LogConfiguration{
		LogGroupName:           logGroupName,
		LogStreamName:          logStreamName,
		LogDriver:              logDriver,
		Options:                options,
		FluentBitConfigMapName: fmt.Sprintf("fluent-bit-config-%s", taskArn),
		FluentBitConfig:        fluentBitConfig,
	}, nil
}

// generateFluentBitConfig generates FluentBit configuration for CloudWatch
func (i *integration) generateFluentBitConfig(logGroupName, logStreamPrefix string, options map[string]string) string {
	region := options["awslogs-region"]
	if region == "" {
		region = "us-east-1"
	}

	endpoint := i.config.LocalStackEndpoint
	if endpoint == "" {
		endpoint = "http://localstack.aws-services.svc.cluster.local:4566"
	}

	// FluentBit configuration for CloudWatch
	return fmt.Sprintf(`[SERVICE]
    Flush        1
    Daemon       Off
    Log_Level    info

[INPUT]
    Name              tail
    Path              /var/log/containers/*.log
    Parser            docker
    Tag               kube.*
    Refresh_Interval  5
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On

[FILTER]
    Name                kubernetes
    Match               kube.*
    Kube_URL            https://kubernetes.default.svc:443
    Kube_CA_File        /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    Kube_Token_File     /var/run/secrets/kubernetes.io/serviceaccount/token
    Merge_Log           On
    K8S-Logging.Parser  On
    K8S-Logging.Exclude On

[OUTPUT]
    Name                cloudwatch_logs
    Match               *
    region              %s
    log_group_name      %s
    log_stream_prefix   %s
    auto_create_group   true
    endpoint            %s
    port                4566
    tls                 Off
    net.keepalive       Off
`, region, logGroupName, logStreamPrefix, endpoint)
}

