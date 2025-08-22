package kubernetes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// LogCollector collects logs from terminating pods
type LogCollector struct {
	clientset kubernetes.Interface
	storage   storage.Storage
}

// NewLogCollector creates a new log collector
func NewLogCollector(clientset kubernetes.Interface, storage storage.Storage) *LogCollector {
	return &LogCollector{
		clientset: clientset,
		storage:   storage,
	}
}

// CollectTaskLogs collects logs from all containers in a task before termination
func (lc *LogCollector) CollectTaskLogs(ctx context.Context, taskArn, namespace, podName string) error {
	if lc.clientset == nil || lc.storage == nil {
		logging.Debug("Skipping log collection - no kubernetes client or storage available")
		return nil
	}

	// Check if log store is available
	logStore := lc.storage.TaskLogStore()
	if logStore == nil {
		logging.Debug("Skipping log collection - TaskLogStore not available")
		return nil
	}

	logging.Info("Collecting logs for terminating task", "taskArn", taskArn, "pod", podName)

	// Get pod to retrieve container names
	pod, err := lc.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		logging.Warn("Failed to get pod for log collection", "error", err, "pod", podName)
		return fmt.Errorf("failed to get pod: %w", err)
	}

	// Collect logs from all containers
	var allLogs []storage.TaskLog
	for _, container := range pod.Spec.Containers {
		logs, err := lc.collectContainerLogs(ctx, taskArn, namespace, podName, container.Name)
		if err != nil {
			logging.Warn("Failed to collect logs from container",
				"container", container.Name,
				"error", err)
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	// Also collect logs from init containers if any
	for _, container := range pod.Spec.InitContainers {
		logs, err := lc.collectContainerLogs(ctx, taskArn, namespace, podName, container.Name)
		if err != nil {
			logging.Warn("Failed to collect logs from init container",
				"container", container.Name,
				"error", err)
			continue
		}
		allLogs = append(allLogs, logs...)
	}

	// Save collected logs to storage
	if len(allLogs) > 0 {
		if err := logStore.SaveLogs(ctx, allLogs); err != nil {
			logging.Error("Failed to save collected logs", "error", err, "count", len(allLogs))
			return fmt.Errorf("failed to save logs: %w", err)
		}
		logging.Info("Successfully collected and saved task logs",
			"taskArn", taskArn,
			"logCount", len(allLogs))
	} else {
		logging.Info("No logs collected for task", "taskArn", taskArn)
	}

	return nil
}

// collectContainerLogs collects logs from a specific container
func (lc *LogCollector) collectContainerLogs(ctx context.Context, taskArn, namespace, podName, containerName string) ([]storage.TaskLog, error) {
	// Get logs from the container
	req := lc.clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container:  containerName,
		Timestamps: true,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get log stream: %w", err)
	}
	defer stream.Close()

	var logs []storage.TaskLog
	scanner := bufio.NewScanner(stream)
	now := time.Now()

	for scanner.Scan() {
		line := scanner.Text()

		// Parse timestamp and log line
		// Format: 2024-01-20T10:30:45.123456789Z Log message here
		timestamp, logLine := parseTimestampedLogLine(line)

		// Determine log level from content
		logLevel := detectLogLevel(logLine)

		log := storage.TaskLog{
			TaskArn:       taskArn,
			ContainerName: containerName,
			Timestamp:     timestamp,
			LogLine:       logLine,
			LogLevel:      logLevel,
			CreatedAt:     now,
		}
		logs = append(logs, log)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return logs, fmt.Errorf("error reading logs: %w", err)
	}

	return logs, nil
}

// parseTimestampedLogLine parses a log line with timestamp
func parseTimestampedLogLine(line string) (time.Time, string) {
	// Try to parse RFC3339Nano timestamp at the beginning
	parts := strings.SplitN(line, " ", 2)
	if len(parts) >= 2 {
		if timestamp, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
			return timestamp, parts[1]
		}
	}
	// If no timestamp found, use current time
	return time.Now(), line
}

// detectLogLevel attempts to detect the log level from the log content
func detectLogLevel(logLine string) string {
	logLineUpper := strings.ToUpper(logLine)

	// Check for common log level patterns
	switch {
	case strings.Contains(logLineUpper, "ERROR") || strings.Contains(logLineUpper, "FATAL"):
		return "ERROR"
	case strings.Contains(logLineUpper, "WARN") || strings.Contains(logLineUpper, "WARNING"):
		return "WARN"
	case strings.Contains(logLineUpper, "DEBUG"):
		return "DEBUG"
	case strings.Contains(logLineUpper, "INFO"):
		return "INFO"
	default:
		return "INFO" // Default to INFO if no pattern matches
	}
}

// CollectLogsBeforeDeletion is a convenience method that collects logs before pod deletion
func (lc *LogCollector) CollectLogsBeforeDeletion(ctx context.Context, taskArn, namespace, podName string) {
	// Run collection in a goroutine with timeout to avoid blocking deletion
	go func() {
		collectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := lc.CollectTaskLogs(collectCtx, taskArn, namespace, podName); err != nil {
			logging.Error("Failed to collect logs before deletion",
				"error", err,
				"taskArn", taskArn,
				"pod", podName)
		}
	}()

	// Give log collection a small head start before deletion proceeds
	time.Sleep(100 * time.Millisecond)
}
