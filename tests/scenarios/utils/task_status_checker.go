package utils

import (
	"errors"
	"fmt"
	"time"
)

// TaskStatus represents a task status with timestamp
type TaskStatus struct {
	Status    string
	Timestamp time.Time
	Reason    string
}

// TaskStatusChecker tracks and validates task status transitions
type TaskStatusChecker struct {
	client        *ECSClient
	statusHistory map[string][]TaskStatus
}

// NewTaskStatusChecker creates a new task status checker
func NewTaskStatusChecker(client *ECSClient) *TaskStatusChecker {
	return &TaskStatusChecker{
		client:        client,
		statusHistory: make(map[string][]TaskStatus),
	}
}

// trackStatus adds a status to the history for a task
func (c *TaskStatusChecker) trackStatus(taskArn, status string) {
	c.statusHistory[taskArn] = append(c.statusHistory[taskArn], TaskStatus{
		Status:    status,
		Timestamp: time.Now(),
	})
}

// WaitForStatus waits for a task to reach a specific status
func (c *TaskStatusChecker) WaitForStatus(cluster, taskArn string, expectedStatus string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Describe task to get current status
			result, err := c.client.DescribeTasks(cluster, []string{taskArn})
			if err != nil {
				return fmt.Errorf("failed to describe task: %w", err)
			}

			tasks, ok := result["tasks"].([]interface{})
			if !ok || len(tasks) == 0 {
				return fmt.Errorf("no task found with arn: %s", taskArn)
			}

			task := tasks[0].(map[string]interface{})
			currentStatus := task["lastStatus"].(string)
			
			// Record status in history
			c.recordStatus(taskArn, currentStatus, "")

			if currentStatus == expectedStatus {
				return nil
			}

			// Check for terminal states
			if currentStatus == "STOPPED" && expectedStatus != "STOPPED" {
				stoppedReason := ""
				if reason, ok := task["stoppedReason"].(string); ok {
					stoppedReason = reason
				}
				return fmt.Errorf("task stopped unexpectedly: %s", stoppedReason)
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for task status %s, current status: %s", expectedStatus, currentStatus)
			}

		case <-time.After(timeout):
			return fmt.Errorf("timeout waiting for task status: %s", expectedStatus)
		}
	}
}

// GetStatusHistory returns the status history for a task
func (c *TaskStatusChecker) GetStatusHistory(taskArn string) []TaskStatus {
	history, exists := c.statusHistory[taskArn]
	if !exists {
		return []TaskStatus{}
	}
	return history
}

// ValidateTransitions validates that task status transitions are valid
func (c *TaskStatusChecker) ValidateTransitions(taskArn string) error {
	history := c.GetStatusHistory(taskArn)
	if len(history) == 0 {
		return errors.New("no status history for task")
	}

	validTransitions := map[string][]string{
		"PROVISIONING":     {"PENDING", "STOPPED"},
		"PENDING":          {"ACTIVATING", "RUNNING", "STOPPED"},
		"ACTIVATING":       {"RUNNING", "STOPPED"},
		"RUNNING":          {"DEACTIVATING", "STOPPING", "STOPPED"},
		"DEACTIVATING":     {"STOPPING", "STOPPED"},
		"STOPPING":         {"DEPROVISIONING", "STOPPED"},
		"DEPROVISIONING":   {"STOPPED"},
		"STOPPED":          {}, // Terminal state
	}

	for i := 0; i < len(history)-1; i++ {
		currentStatus := history[i].Status
		nextStatus := history[i+1].Status

		validNext, exists := validTransitions[currentStatus]
		if !exists {
			return fmt.Errorf("unknown status: %s", currentStatus)
		}

		isValid := false
		for _, valid := range validNext {
			if valid == nextStatus {
				isValid = true
				break
			}
		}

		if !isValid {
			return fmt.Errorf("invalid transition: %s -> %s", currentStatus, nextStatus)
		}
	}

	return nil
}

// CheckTaskHealth checks if a task is healthy based on its containers
func (c *TaskStatusChecker) CheckTaskHealth(cluster, taskArn string) (bool, error) {
	result, err := c.client.DescribeTasks(cluster, []string{taskArn})
	if err != nil {
		return false, fmt.Errorf("failed to describe task: %w", err)
	}

	tasks, ok := result["tasks"].([]interface{})
	if !ok || len(tasks) == 0 {
		return false, fmt.Errorf("no task found with arn: %s", taskArn)
	}

	task := tasks[0].(map[string]interface{})
	
	// Check if task is running
	if task["lastStatus"] != "RUNNING" {
		return false, nil
	}

	// Check container health
	containers, ok := task["containers"].([]interface{})
	if !ok {
		return false, fmt.Errorf("no containers found in task")
	}

	for _, c := range containers {
		container := c.(map[string]interface{})
		
		// Check container status
		if lastStatus, ok := container["lastStatus"].(string); ok && lastStatus != "RUNNING" {
			return false, nil
		}

		// Check health status if available
		if healthStatus, ok := container["healthStatus"].(string); ok && healthStatus == "UNHEALTHY" {
			return false, nil
		}
	}

	return true, nil
}

// GetCurrentStatus returns the current status of a task
func (c *TaskStatusChecker) GetCurrentStatus(cluster, taskArn string) (*TaskStatus, error) {
	result, err := c.client.DescribeTasks(cluster, []string{taskArn})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task: %w", err)
	}

	tasks, ok := result["tasks"].([]interface{})
	if !ok || len(tasks) == 0 {
		return nil, fmt.Errorf("no task found with arn: %s", taskArn)
	}

	task := tasks[0].(map[string]interface{})
	
	status := &TaskStatus{
		Status:    task["lastStatus"].(string),
		Timestamp: time.Now(),
	}
	
	// Track the status in history
	c.trackStatus(taskArn, status.Status)
	
	return status, nil
}

// GetTaskExitCode returns the exit code of the task's essential container
func (c *TaskStatusChecker) GetTaskExitCode(cluster, taskArn string) (*int, error) {
	result, err := c.client.DescribeTasks(cluster, []string{taskArn})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task: %w", err)
	}

	tasks, ok := result["tasks"].([]interface{})
	if !ok || len(tasks) == 0 {
		return nil, fmt.Errorf("no task found with arn: %s", taskArn)
	}

	task := tasks[0].(map[string]interface{})
	containers, ok := task["containers"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no containers found in task")
	}

	// Find essential container's exit code
	for _, c := range containers {
		container := c.(map[string]interface{})
		
		// Check if container is essential
		essential, ok := container["essential"].(bool)
		if !ok || !essential {
			continue
		}

		// Get exit code
		if exitCode, ok := container["exitCode"].(float64); ok {
			code := int(exitCode)
			return &code, nil
		}
	}

	return nil, fmt.Errorf("no exit code found for essential container")
}

// recordStatus records a status in the history
func (c *TaskStatusChecker) recordStatus(taskArn, status, reason string) {
	if _, exists := c.statusHistory[taskArn]; !exists {
		c.statusHistory[taskArn] = []TaskStatus{}
	}

	// Only record if status changed
	history := c.statusHistory[taskArn]
	if len(history) == 0 || history[len(history)-1].Status != status {
		c.statusHistory[taskArn] = append(history, TaskStatus{
			Status:    status,
			Timestamp: time.Now(),
			Reason:    reason,
		})
	}
}

// WaitForServiceStable waits for a service to become stable
func (c *TaskStatusChecker) WaitForServiceStable(cluster, service string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := c.client.DescribeServices(cluster, []string{service})
			if err != nil {
				return fmt.Errorf("failed to describe service: %w", err)
			}

			services, ok := result["services"].([]interface{})
			if !ok || len(services) == 0 {
				return fmt.Errorf("no service found: %s", service)
			}

			svc := services[0].(map[string]interface{})
			
			// Check if desired count matches running count
			desiredCount := int(svc["desiredCount"].(float64))
			runningCount := int(svc["runningCount"].(float64))
			pendingCount := int(svc["pendingCount"].(float64))

			if runningCount == desiredCount && pendingCount == 0 {
				// Check deployments
				deployments := svc["deployments"].([]interface{})
				if len(deployments) == 1 {
					deployment := deployments[0].(map[string]interface{})
					if deployment["status"] == "PRIMARY" &&
						int(deployment["runningCount"].(float64)) == desiredCount &&
						int(deployment["pendingCount"].(float64)) == 0 {
						return nil
					}
				}
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for service to stabilize: desired=%d, running=%d, pending=%d",
					desiredCount, runningCount, pendingCount)
			}

		case <-time.After(timeout):
			return fmt.Errorf("timeout waiting for service to stabilize")
		}
	}
}