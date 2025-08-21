package storage

import (
	"context"
	"time"
)

// TaskLog represents a single log entry for a task container
type TaskLog struct {
	ID            string    `json:"id"`
	TaskArn       string    `json:"task_arn"`
	ContainerName string    `json:"container_name"`
	Timestamp     time.Time `json:"timestamp"`
	LogLine       string    `json:"log_line"`
	LogLevel      string    `json:"log_level,omitempty"` // Optional: INFO, WARN, ERROR, DEBUG
	CreatedAt     time.Time `json:"created_at"`
}

// TaskLogFilter represents filter criteria for querying logs
type TaskLogFilter struct {
	TaskArn       string
	ContainerName string
	From          *time.Time
	To            *time.Time
	LogLevel      string
	SearchText    string
	Limit         int
	Offset        int
}

// TaskLogStore defines the interface for task log storage operations
type TaskLogStore interface {
	// SaveLogs saves multiple log entries for a task container
	SaveLogs(ctx context.Context, logs []TaskLog) error

	// GetLogs retrieves logs based on filter criteria
	GetLogs(ctx context.Context, filter TaskLogFilter) ([]TaskLog, error)

	// GetLogCount returns the count of logs matching the filter
	GetLogCount(ctx context.Context, filter TaskLogFilter) (int64, error)

	// DeleteOldLogs removes logs older than the specified retention period
	DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error)

	// DeleteTaskLogs removes all logs for a specific task
	DeleteTaskLogs(ctx context.Context, taskArn string) error
}