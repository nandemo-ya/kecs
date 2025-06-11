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
	client        ECSClientInterface
	statusHistory map[string][]TaskStatus
}

// NewTaskStatusChecker creates a new task status checker
func NewTaskStatusChecker(client ECSClientInterface) *TaskStatusChecker {
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

// WaitForTaskStatus waits for a task to reach a specific status
func (c *TaskStatusChecker) WaitForTaskStatus(cluster, taskArn, expectedStatus string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Describe task to get current status
			tasks, err := c.client.DescribeTasks(cluster, []string{taskArn})
			if err != nil {
				return fmt.Errorf("failed to describe task: %w", err)
			}

			if len(tasks) == 0 {
				return fmt.Errorf("no task found with arn: %s", taskArn)
			}

			task := tasks[0]
			currentStatus := task.LastStatus
			
			// Record status in history
			c.recordStatus(taskArn, currentStatus, "")

			if currentStatus == expectedStatus {
				return nil
			}

			// Check for terminal states
			if currentStatus == "STOPPED" && expectedStatus != "STOPPED" {
				return fmt.Errorf("task stopped unexpectedly: %s", task.StoppedReason)
			}

			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for task status %s, current status: %s", expectedStatus, currentStatus)
			}

		case <-time.After(timeout):
			return fmt.Errorf("timeout waiting for task status: %s", expectedStatus)
		}
	}
}

// WaitForTaskRunning waits for a task to reach RUNNING status
func (c *TaskStatusChecker) WaitForTaskRunning(cluster, taskArn string) error {
	return c.WaitForTaskStatus(cluster, taskArn, "RUNNING", 2*time.Minute)
}

// WaitForTaskStopped waits for a task to reach STOPPED status
func (c *TaskStatusChecker) WaitForTaskStopped(cluster, taskArn string) error {
	return c.WaitForTaskStatus(cluster, taskArn, "STOPPED", 2*time.Minute)
}

// ValidateTaskTransition validates that task status transitions are valid
func (c *TaskStatusChecker) ValidateTaskTransition(taskArn, fromStatus, toStatus string) error {
	// Valid transitions based on ECS task lifecycle
	validTransitions := map[string][]string{
		"PROVISIONING": {"PENDING", "STOPPED"},
		"PENDING":      {"ACTIVATING", "STOPPED"},
		"ACTIVATING":   {"RUNNING", "STOPPED"},
		"RUNNING":      {"DEACTIVATING", "STOPPED", "STOPPING"},
		"DEACTIVATING": {"STOPPING", "STOPPED"},
		"STOPPING":     {"DEPROVISIONING", "STOPPED"},
		"DEPROVISIONING": {"STOPPED"},
		"STOPPED":      {}, // Terminal state
	}

	allowed, exists := validTransitions[fromStatus]
	if !exists {
		return fmt.Errorf("unknown status: %s", fromStatus)
	}

	for _, status := range allowed {
		if status == toStatus {
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", fromStatus, toStatus)
}

// GetStatusHistory returns the status history for a task
func (c *TaskStatusChecker) GetStatusHistory(taskArn string) []TaskStatus {
	return c.statusHistory[taskArn]
}

// ClearHistory clears the status history for a task
func (c *TaskStatusChecker) ClearHistory(taskArn string) {
	delete(c.statusHistory, taskArn)
}

// ClearAllHistory clears all status history
func (c *TaskStatusChecker) ClearAllHistory() {
	c.statusHistory = make(map[string][]TaskStatus)
}

// CheckTaskHealth checks if a task is healthy based on its containers
func (c *TaskStatusChecker) CheckTaskHealth(cluster, taskArn string) (bool, error) {
	tasks, err := c.client.DescribeTasks(cluster, []string{taskArn})
	if err != nil {
		return false, fmt.Errorf("failed to describe task: %w", err)
	}

	if len(tasks) == 0 {
		return false, fmt.Errorf("no task found with arn: %s", taskArn)
	}

	task := tasks[0]
	
	// Check if task is running
	if task.LastStatus != "RUNNING" {
		return false, nil
	}

	// For now, if task is running, consider it healthy
	// TODO: Add container health checks when container details are available
	return true, nil
}

// GetCurrentStatus returns the current status of a task
func (c *TaskStatusChecker) GetCurrentStatus(cluster, taskArn string) (*TaskStatus, error) {
	tasks, err := c.client.DescribeTasks(cluster, []string{taskArn})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no task found with arn: %s", taskArn)
	}

	task := tasks[0]
	
	status := &TaskStatus{
		Status:    task.LastStatus,
		Timestamp: time.Now(),
	}
	
	// Track the status in history
	c.trackStatus(taskArn, status.Status)
	
	return status, nil
}

// GetTaskExitCode returns the exit code of the task's essential container
func (c *TaskStatusChecker) GetTaskExitCode(cluster, taskArn string) (*int, error) {
	tasks, err := c.client.DescribeTasks(cluster, []string{taskArn})
	if err != nil {
		return nil, fmt.Errorf("failed to describe task: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no task found with arn: %s", taskArn)
	}

	// TODO: Implement exit code extraction when container details are available in Task struct
	// For now, return nil
	return nil, fmt.Errorf("exit code extraction not yet implemented")
}

// recordStatus is a helper to record status with optional reason
func (c *TaskStatusChecker) recordStatus(taskArn, status, reason string) {
	if c.statusHistory == nil {
		c.statusHistory = make(map[string][]TaskStatus)
	}
	
	c.statusHistory[taskArn] = append(c.statusHistory[taskArn], TaskStatus{
		Status:    status,
		Timestamp: time.Now(),
		Reason:    reason,
	})
}

// WaitForServiceStable waits for a service to become stable
func (c *TaskStatusChecker) WaitForServiceStable(cluster, service string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			svc, err := c.client.DescribeService(cluster, service)
			if err != nil {
				return fmt.Errorf("failed to describe service: %w", err)
			}

			if svc == nil {
				return fmt.Errorf("no service found: %s", service)
			}
			
			// Check if desired count matches running count
			desiredCount := svc.DesiredCount
			runningCount := svc.RunningCount
			pendingCount := svc.PendingCount

			if runningCount == desiredCount && pendingCount == 0 {
				// Check deployments
				if len(svc.Deployments) > 0 {
					primaryDeployment := svc.Deployments[0]
					if primaryDeployment.Status == "PRIMARY" &&
						primaryDeployment.RunningCount == desiredCount &&
						primaryDeployment.PendingCount == 0 {
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

// CheckMultipleTasksStatus checks if all tasks have the expected status
func (c *TaskStatusChecker) CheckMultipleTasksStatus(cluster string, taskArns []string, expectedStatus string) error {
	if len(taskArns) == 0 {
		return errors.New("no task ARNs provided")
	}

	tasks, err := c.client.DescribeTasks(cluster, taskArns)
	if err != nil {
		return fmt.Errorf("failed to describe tasks: %w", err)
	}

	if len(tasks) != len(taskArns) {
		return fmt.Errorf("expected %d tasks, but got %d", len(taskArns), len(tasks))
	}

	for _, task := range tasks {
		if task.LastStatus != expectedStatus {
			return fmt.Errorf("task %s has status %s, expected %s", 
				task.TaskArn, task.LastStatus, expectedStatus)
		}
	}

	return nil
}