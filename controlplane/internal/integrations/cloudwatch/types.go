package cloudwatch

import (
	"context"
	cloudwatchlogsapi "github.com/nandemo-ya/kecs/controlplane/internal/cloudwatchlogs/generated"
)

// Integration represents the CloudWatch-Kubernetes integration
type Integration interface {
	// CreateLogGroup creates a log group in LocalStack CloudWatch
	CreateLogGroup(groupName string) error
	
	// CreateLogStream creates a log stream for a container
	CreateLogStream(groupName, streamName string) error
	
	// DeleteLogGroup deletes a log group
	DeleteLogGroup(groupName string) error
	
	// GetLogGroupForTask returns the log group name for a task
	GetLogGroupForTask(taskArn string) string
	
	// GetLogStreamForContainer returns the log stream name for a container
	GetLogStreamForContainer(taskArn, containerName string) string
	
	// ConfigureContainerLogging configures logging for a container in pod spec
	ConfigureContainerLogging(taskArn string, containerName string, logDriver string, options map[string]string) (*LogConfiguration, error)
}

// LogConfiguration represents CloudWatch logging configuration for a container
type LogConfiguration struct {
	LogGroupName  string
	LogStreamName string
	LogDriver     string
	Options       map[string]string
	
	// Kubernetes logging configuration
	FluentBitConfigMapName string
	FluentBitConfig        string
}

// Config represents CloudWatch integration configuration
type Config struct {
	LocalStackEndpoint string
	LogGroupPrefix     string // Prefix for created log groups (e.g., "/ecs/")
	RetentionDays      int32  // Log retention in days
	KubeNamespace      string
}

// Constants for annotations and labels
var LogAnnotations = struct {
	LogGroupName  string
	LogStreamName string
	LogDriver     string
}{
	LogGroupName:  "kecs.io/cloudwatch-log-group",
	LogStreamName: "kecs.io/cloudwatch-log-stream",
	LogDriver:     "kecs.io/log-driver",
}

// CloudWatchLogsClient interface for CloudWatch Logs operations (for testing)
type CloudWatchLogsClient interface {
	CreateLogGroup(ctx context.Context, params *cloudwatchlogsapi.CreateLogGroupRequest) (*cloudwatchlogsapi.Unit, error)
	DeleteLogGroup(ctx context.Context, params *cloudwatchlogsapi.DeleteLogGroupRequest) (*cloudwatchlogsapi.Unit, error)
	CreateLogStream(ctx context.Context, params *cloudwatchlogsapi.CreateLogStreamRequest) (*cloudwatchlogsapi.Unit, error)
	PutRetentionPolicy(ctx context.Context, params *cloudwatchlogsapi.PutRetentionPolicyRequest) (*cloudwatchlogsapi.Unit, error)
}