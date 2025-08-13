// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// sendError sends an error response in AWS ECS API format
func sendError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"__type":  errorType,
		"message": message,
	})
}

// GetTaskLogsRequest represents the request for getting task logs
type GetTaskLogsRequest struct {
	Cluster    string `json:"cluster"`
	TaskArn    string `json:"taskArn"`
	Follow     bool   `json:"follow,omitempty"`
	Timestamps bool   `json:"timestamps,omitempty"`
	Since      string `json:"since,omitempty"`
	Tail       *int64 `json:"tail,omitempty"`
}

// GetTaskLogsResponse represents the response for getting task logs
type GetTaskLogsResponse struct {
	Logs []LogEntry `json:"logs"`
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Container string    `json:"container,omitempty"`
}

// HandleGetTaskLogs handles the GetTaskLogs API request
func (api *DefaultECSAPI) HandleGetTaskLogs(w http.ResponseWriter, r *http.Request) {
	var req GetTaskLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "InvalidParameterValue", "Invalid request body")
		return
	}

	// Extract task ID from ARN
	taskID := extractTaskIDFromArn(req.TaskArn)
	if taskID == "" {
		sendError(w, http.StatusBadRequest, "InvalidParameterValue", "Invalid task ARN")
		return
	}

	// Get task from storage to validate it exists
	ctx := r.Context()
	_, err := api.storage.TaskStore().Get(ctx, req.Cluster, taskID)
	if err != nil {
		sendError(w, http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf("Task not found: %s", taskID))
		return
	}

	// Get Kubernetes client
	clientset, err := api.getKubernetesClient()
	if err != nil {
		logging.Error("Failed to get Kubernetes client", "error", err)
		sendError(w, http.StatusInternalServerError, "InternalError", "Failed to connect to Kubernetes")
		return
	}

	// Extract namespace and pod name from task ARN
	// Task ARN format: arn:aws:ecs:region:account:task/namespace/pod-name
	// or just namespace/pod-name
	namespace, podName := extractNamespaceAndPodName(req.TaskArn)
	
	logging.Debug("Fetching logs for task", 
		"taskArn", req.TaskArn,
		"namespace", namespace, 
		"podName", podName)

	// Get pod logs
	logs, err := getPodLogs(ctx, clientset, namespace, podName, req)
	if err != nil {
		logging.Error("Failed to get pod logs", "pod", podName, "error", err)
		sendError(w, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Failed to get logs: %v", err))
		return
	}

	// Send response
	response := GetTaskLogsResponse{
		Logs: logs,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error("Failed to encode response", "error", err)
	}
}

// getPodLogs retrieves logs from a Kubernetes pod
func getPodLogs(ctx context.Context, clientset kubernetes.Interface, namespace, podName string, req GetTaskLogsRequest) ([]LogEntry, error) {
	// First try to get the pod directly by name
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		// If not found, try to find by task-id label
		logging.Debug("Pod not found by name, trying to find by task-id label", 
			"namespace", namespace, 
			"podName", podName)
		
		labelSelector := fmt.Sprintf("kecs.dev/task-id=%s", podName)
		pods, listErr := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		
		if listErr != nil || len(pods.Items) == 0 {
			// Last attempt: find pod with matching name pattern
			// For service tasks, the pod name might be a deployment pod
			pods, listErr = clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if listErr != nil {
				return nil, fmt.Errorf("failed to list pods: %w", listErr)
			}
			
			// Find pod that contains the task ID or matches the pattern
			for _, p := range pods.Items {
				if p.Name == podName || strings.Contains(p.Name, podName) {
					pod = &p
					break
				}
			}
			
			if pod == nil {
				return nil, fmt.Errorf("pod not found: %s in namespace %s", podName, namespace)
			}
		} else {
			pod = &pods.Items[0]
		}
	}
	
	logging.Debug("Found pod for logs", 
		"podName", pod.Name,
		"namespace", pod.Namespace,
		"containerCount", len(pod.Spec.Containers))

	// Build pod log options
	opts := &corev1.PodLogOptions{
		Timestamps: req.Timestamps,
		Follow:     req.Follow,
	}

	if req.Tail != nil {
		opts.TailLines = req.Tail
	}

	if req.Since != "" {
		// Parse duration (e.g., "5m", "1h")
		duration, err := time.ParseDuration(req.Since)
		if err == nil {
			sinceSeconds := int64(duration.Seconds())
			opts.SinceSeconds = &sinceSeconds
		}
	}

	var allLogs []LogEntry

	// Get logs from each container
	for _, container := range pod.Spec.Containers {
		opts.Container = container.Name

		// Get log stream
		logReq := clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
		stream, err := logReq.Stream(ctx)
		if err != nil {
			logging.Warn("Failed to get logs for container", "container", container.Name, "error", err)
			continue
		}
		defer stream.Close()

		// Parse logs
		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			line := scanner.Text()
			
			// Parse log entry
			entry := parseLogLine(line, container.Name, req.Timestamps)
			allLogs = append(allLogs, entry)

			// For follow mode, we'd need to handle this differently (e.g., SSE or WebSocket)
			if req.Follow {
				// TODO: Implement streaming logs
				break
			}
		}

		if err := scanner.Err(); err != nil {
			logging.Warn("Error reading logs", "container", container.Name, "error", err)
		}
	}

	// Sort logs by timestamp for multi-container tasks
	sortLogsByTimestamp(allLogs)

	return allLogs, nil
}

// parseLogLine parses a log line into a LogEntry
func parseLogLine(line, containerName string, hasTimestamp bool) LogEntry {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   line,
		Container: containerName,
	}

	// If timestamps are included, parse them
	if hasTimestamp {
		// Kubernetes log format with timestamp: "2025-01-08T10:30:45.123456789Z message"
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 {
			if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				entry.Timestamp = t
				entry.Message = parts[1]
			}
		}
	}

	// Determine log level from message content
	lowerMsg := strings.ToLower(entry.Message)
	if strings.Contains(lowerMsg, "error") || strings.Contains(lowerMsg, "fatal") || strings.Contains(lowerMsg, "panic") {
		entry.Level = "ERROR"
	} else if strings.Contains(lowerMsg, "warn") || strings.Contains(lowerMsg, "warning") {
		entry.Level = "WARN"
	} else if strings.Contains(lowerMsg, "debug") || strings.Contains(lowerMsg, "trace") {
		entry.Level = "DEBUG"
	}

	return entry
}

// extractTaskIDFromArn extracts the task ID from a task ARN
func extractTaskIDFromArn(arn string) string {
	// ARN format: arn:aws:ecs:region:account:task/cluster-name/task-id
	// or just task-id
	parts := strings.Split(arn, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return arn
}

// extractNamespaceAndPodName extracts namespace and pod name from a task ARN
func extractNamespaceAndPodName(arn string) (namespace, podName string) {
	// KECS Task ARN format: arn:aws:ecs:region:account:task/namespace/pod-name
	// where namespace is typically "{cluster-name}-{region}" format
	// e.g., arn:aws:ecs:us-east-1:000000000000:task/default-us-east-1/ecs-service-single-task-nginx-6b48b86448-4czhg
	
	// Extract the part after "task/"
	parts := strings.Split(arn, "task/")
	if len(parts) < 2 {
		// Fallback to default namespace and use the whole ARN as pod name
		return "default", arn
	}
	
	// Split by "/" to get namespace and pod name
	// In KECS, the namespace contains both cluster name and region
	taskParts := strings.Split(parts[1], "/")
	if len(taskParts) >= 2 {
		// taskParts[0] is the namespace (e.g., "default-us-east-1")
		// taskParts[1] is the pod name
		return taskParts[0], taskParts[1]
	}
	
	// If only one part, assume default namespace
	if len(taskParts) == 1 {
		return "default", taskParts[0]
	}
	
	return "default", arn
}

// sortLogsByTimestamp sorts log entries by timestamp
func sortLogsByTimestamp(logs []LogEntry) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Before(logs[j].Timestamp)
	})
}

// getKubernetesClient returns a Kubernetes client
func (api *DefaultECSAPI) getKubernetesClient() (kubernetes.Interface, error) {
	// Get the Kubernetes client from task manager
	if api.taskManagerInstance == nil {
		return nil, fmt.Errorf("task manager not initialized")
	}
	
	if api.taskManagerInstance.Clientset == nil {
		return nil, fmt.Errorf("kubernetes client not initialized")
	}
	
	return api.taskManagerInstance.Clientset, nil
}