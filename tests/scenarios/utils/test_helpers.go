package utils

import (
	"fmt"
	"strings"
	"time"
)

// TestConfig holds common test configuration
type TestConfig struct {
	Timeout        time.Duration
	RetryInterval  time.Duration
	CleanupOnError bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		Timeout:        30 * time.Second,
		RetryInterval:  1 * time.Second,
		CleanupOnError: true,
	}
}

// TestingT is an interface that both *testing.T and ginkgo.GinkgoTInterface implement
type TestingT interface {
	Logf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// WaitForCondition waits for a condition to be true
func WaitForCondition(t TestingT, condition func() bool, timeout time.Duration, message string) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(1 * time.Second)
	}
	
	t.Fatalf("Timeout waiting for condition: %s", message)
}

// AssertClusterActive asserts that a cluster is in ACTIVE state
func AssertClusterActive(t TestingT, client *ECSClient, clusterName string) {
	WaitForCondition(t, func() bool {
		cluster, err := client.DescribeCluster(clusterName)
		if err != nil {
			t.Logf("Error describing cluster: %v", err)
			return false
		}
		return cluster.Status == "ACTIVE"
	}, 30*time.Second, fmt.Sprintf("cluster %s to become ACTIVE", clusterName))
}

// AssertClusterDeleted asserts that a cluster has been deleted
func AssertClusterDeleted(t TestingT, client *ECSClient, clusterName string) {
	WaitForCondition(t, func() bool {
		_, err := client.DescribeCluster(clusterName)
		return err != nil && (containsString(err.Error(), "not found") || containsString(err.Error(), "MISSING"))
	}, 30*time.Second, fmt.Sprintf("cluster %s to be deleted", clusterName))
}

// CleanupCluster safely deletes a cluster, ignoring errors if it doesn't exist
func CleanupCluster(t TestingT, client *ECSClient, clusterName string) {
	err := client.DeleteCluster(clusterName)
	if err != nil && !containsString(err.Error(), "not found") {
		t.Logf("Warning: failed to cleanup cluster %s: %v", clusterName, err)
	}
}

// GenerateTestName generates a unique test resource name
func GenerateTestName(prefix string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", prefix, timestamp)
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestLogger provides structured logging for tests
type TestLogger struct {
	t TestingT
}

// NewTestLogger creates a new test logger
func NewTestLogger(t TestingT) *TestLogger {
	return &TestLogger{t: t}
}

// Info logs an info message
func (l *TestLogger) Info(format string, args ...interface{}) {
	l.t.Logf("[INFO] "+format, args...)
}

// Debug logs a debug message
func (l *TestLogger) Debug(format string, args ...interface{}) {
	if getEnvOrDefault("KECS_LOG_LEVEL", "info") == "debug" {
		l.t.Logf("[DEBUG] "+format, args...)
	}
}

// Error logs an error message
func (l *TestLogger) Error(format string, args ...interface{}) {
	l.t.Logf("[ERROR] "+format, args...)
}

// InterfaceSliceToStringSlice converts []interface{} to []string
func InterfaceSliceToStringSlice(slice []interface{}) []string {
	result := make([]string, len(slice))
	for i, v := range slice {
		if str, ok := v.(string); ok {
			result[i] = str
		}
	}
	return result
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		// Add some entropy by sleeping a tiny bit
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

// GetTasksFromResult extracts tasks array from a result map, handling different type formats
func GetTasksFromResult(result map[string]interface{}) ([]map[string]interface{}, error) {
	tasksValue, ok := result["tasks"]
	if !ok {
		return nil, fmt.Errorf("no tasks field in result")
	}

	switch tasks := tasksValue.(type) {
	case []interface{}:
		// Convert []interface{} to []map[string]interface{}
		taskMaps := make([]map[string]interface{}, len(tasks))
		for i, task := range tasks {
			taskMap, ok := task.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("task at index %d is not a map", i)
			}
			taskMaps[i] = taskMap
		}
		return taskMaps, nil
	case []map[string]interface{}:
		// Already in the correct format
		return tasks, nil
	default:
		return nil, fmt.Errorf("unexpected type for tasks: %T", tasks)
	}
}