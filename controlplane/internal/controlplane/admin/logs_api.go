package admin

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// LogsAPI handles log-related endpoints
type LogsAPI struct {
	storage       storage.Storage
	kubeClient    k8sclient.Interface
	podLogService *kubernetes.PodLogService
	upgrader      websocket.Upgrader
}

// NewLogsAPI creates a new logs API handler
func NewLogsAPI(storage storage.Storage, kubeClient k8sclient.Interface) *LogsAPI {
	var podLogService *kubernetes.PodLogService
	if kubeClient != nil {
		podLogService = kubernetes.NewPodLogService(kubeClient)
	}

	return &LogsAPI{
		storage:       storage,
		kubeClient:    kubeClient,
		podLogService: podLogService,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin for now
				// TODO: Implement proper CORS check
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// RegisterRoutes registers log API routes
func (api *LogsAPI) RegisterRoutes(router *mux.Router) {
	// REST endpoints using task ID instead of full ARN for simpler URL handling
	logging.Info("Registering log API endpoints")
	router.HandleFunc("/api/tasks/{taskId}/containers/{containerName}/logs", api.HandleGetLogs).Methods("GET")
	router.HandleFunc("/api/tasks/{taskId}/containers/{containerName}/logs/stream", api.HandleStreamLogs).Methods("GET")
	router.HandleFunc("/api/tasks/{taskId}/containers/{containerName}/logs/ws", api.HandleWebSocketLogs).Methods("GET")
	// Add /v1/GetTaskLogs endpoint for KECS TUI
	router.HandleFunc("/v1/GetTaskLogs", api.HandleGetTaskLogs).Methods("POST")
	logging.Info("Log API endpoints registered successfully")
}

// HandleGetLogs retrieves logs from storage (historical logs)
func (api *LogsAPI) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["taskId"]
	containerName := vars["containerName"]

	logging.Info("HandleGetLogs called",
		"taskId", taskId,
		"containerName", containerName,
		"url", r.URL.String())

	// Try to get the actual pod name and namespace from task storage
	var namespace, podName string
	if api.storage != nil && api.storage.TaskStore() != nil {
		// Get cluster from query params or default
		cluster := r.URL.Query().Get("cluster")
		if cluster == "" {
			cluster = "default"
		}
		clusterArn := fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:cluster/%s", cluster)

		// First try to find the task by ID
		task, err := api.storage.TaskStore().Get(r.Context(), clusterArn, taskId)
		if err == nil && task != nil {
			// First try to get pod info from Attributes
			attrNamespace, attrPodName := extractPodInfoFromTaskAttributes(task)
			if attrNamespace != "" && attrPodName != "" {
				namespace = attrNamespace
				podName = attrPodName
				logging.Debug("Found task with pod mapping from attributes",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			} else if task.PodName != "" {
				// Fallback to the stored pod name and namespace fields
				namespace = task.Namespace
				podName = task.PodName
				logging.Debug("Found task with pod mapping from fields",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			}
		}
	}

	// Convert task ID to full ARN for storage lookup
	// Assume default cluster and region for now
	taskArn := fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/default/%s", taskId)

	// Parse query parameters
	query := r.URL.Query()
	filter := storage.TaskLogFilter{
		TaskArn:       taskArn,
		ContainerName: containerName,
	}

	// Parse from timestamp
	if fromStr := query.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			filter.From = &from
		}
	}

	// Parse to timestamp
	if toStr := query.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			filter.To = &to
		}
	}

	// Parse log level filter
	if level := query.Get("level"); level != "" {
		filter.LogLevel = strings.ToUpper(level)
	}

	// Parse search text
	if search := query.Get("search"); search != "" {
		filter.SearchText = search
	}

	// Parse pagination
	if limitStr := query.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 100 // Default limit
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	// Get logs from storage
	var logs []storage.TaskLog
	var err error

	if api.storage != nil && api.storage.TaskLogStore() != nil {
		logs, err = api.storage.TaskLogStore().GetLogs(r.Context(), filter)
		if err != nil {
			logging.Error("Failed to get logs from storage", "error", err)
			// Continue without storage logs
			logs = []storage.TaskLog{}
		}
	}

	// If no logs in storage, try to get from Kubernetes directly
	logging.Info("Checking Kubernetes for logs",
		"storageLogsCount", len(logs),
		"hasKubeClient", api.kubeClient != nil,
		"hasPodLogService", api.podLogService != nil)
	if len(logs) == 0 && api.kubeClient != nil && api.podLogService != nil {
		// If we didn't get pod info from storage, try to find pod by task ID label
		if namespace == "" || podName == "" {
			// Get cluster and region from query parameters
			cluster := r.URL.Query().Get("cluster")
			region := r.URL.Query().Get("region")

			// Both cluster and region are required
			if cluster == "" || region == "" {
				logging.Warn("Missing required query parameters",
					"cluster", cluster,
					"region", region)
				http.Error(w, "Both 'cluster' and 'region' query parameters are required", http.StatusBadRequest)
				return
			}

			// Construct namespace from cluster and region
			namespace = fmt.Sprintf("%s-%s", cluster, region)

			// Try to find pod by task ID label
			logging.Debug("Looking for pod by task ID label",
				"taskId", taskId,
				"namespace", namespace)

			// List pods with the task ID label
			labelSelector := fmt.Sprintf("kecs.dev/task-id=%s", taskId)
			pods, err := api.kubeClient.CoreV1().Pods(namespace).List(r.Context(), metav1.ListOptions{
				LabelSelector: labelSelector,
			})

			if err == nil && len(pods.Items) > 0 {
				// Use the first matching pod
				podName = pods.Items[0].Name
				logging.Debug("Found pod by task ID label",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			} else {
				// Fallback: try using task ID as pod name (for RunTask created pods)
				logging.Debug("No pod found with task ID label, using taskId as pod name",
					"taskId", taskId,
					"error", err)
				podName = taskId
			}
		}

		if namespace != "" && podName != "" {
			// Try to get logs from Kubernetes
			logging.Debug("Attempting to fetch logs from Kubernetes",
				"namespace", namespace,
				"podName", podName,
				"container", containerName)

			options := &kubernetes.LogOptions{
				Follow:    false,
				TailLines: filter.Limit,
			}

			podLogs, err := api.podLogService.GetLogs(r.Context(), namespace, podName, containerName, options)
			if err == nil && len(podLogs) > 0 {
				// The logs are already in TaskLog format
				logs = podLogs
				logging.Info("Successfully retrieved logs from Kubernetes",
					"count", len(podLogs),
					"namespace", namespace,
					"pod", podName,
					"container", containerName)
			} else if err != nil {
				logging.Warn("Failed to get logs from Kubernetes",
					"error", err,
					"namespace", namespace,
					"pod", podName,
					"container", containerName)
			}
		} else {
			logging.Warn("Cannot fetch logs: missing namespace or pod name",
				"taskId", taskId)
		}
	}

	// Get total count for pagination
	var totalCount int64
	if api.storage != nil && api.storage.TaskLogStore() != nil {
		totalCount, err = api.storage.TaskLogStore().GetLogCount(r.Context(), filter)
		if err != nil {
			logging.Warn("Failed to get log count", "error", err)
			totalCount = int64(len(logs))
		}
	} else {
		totalCount = int64(len(logs))
	}

	// Prepare response
	response := map[string]interface{}{
		"logs":       logs,
		"totalCount": totalCount,
		"limit":      filter.Limit,
		"offset":     filter.Offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleStreamLogs streams logs from Kubernetes (live logs)
func (api *LogsAPI) HandleStreamLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["taskId"]
	containerName := vars["containerName"]

	// Get cluster and region from query params
	cluster := r.URL.Query().Get("cluster")
	region := r.URL.Query().Get("region")
	if cluster == "" || region == "" {
		http.Error(w, "Both 'cluster' and 'region' query parameters are required", http.StatusBadRequest)
		return
	}

	// Try to get the actual pod name and namespace from task storage
	var namespace, podName string
	if api.storage != nil && api.storage.TaskStore() != nil {
		clusterArn := fmt.Sprintf("arn:aws:ecs:%s:000000000000:cluster/%s", region, cluster)

		// First try to find the task by ID
		task, err := api.storage.TaskStore().Get(r.Context(), clusterArn, taskId)
		if err == nil && task != nil {
			// First try to get pod info from Attributes
			attrNamespace, attrPodName := extractPodInfoFromTaskAttributes(task)
			if attrNamespace != "" && attrPodName != "" {
				namespace = attrNamespace
				podName = attrPodName
				logging.Debug("Found task with pod mapping from attributes for streaming",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			} else if task.PodName != "" {
				// Fallback to the stored pod name and namespace fields
				namespace = task.Namespace
				podName = task.PodName
				logging.Debug("Found task with pod mapping from fields for streaming",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			}
		}
	}

	// Fall back to parsing if not found in storage
	if namespace == "" || podName == "" {
		// Use cluster and region from query params
		taskArn := fmt.Sprintf("arn:aws:ecs:%s:000000000000:task/%s/%s", region, cluster, taskId)

		// Parse task ARN to get namespace and pod name
		var err error
		namespace, podName, err = parseTaskArn(taskArn, region)
		if err != nil {
			http.Error(w, "Invalid task ARN", http.StatusBadRequest)
			return
		}
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Parse query parameters
	query := r.URL.Query()
	options := &kubernetes.LogOptions{
		Follow: query.Get("follow") == "true",
	}

	// Parse tail lines
	if tailStr := query.Get("tail"); tailStr != "" {
		if tail, err := strconv.Atoi(tailStr); err == nil && tail > 0 {
			options.TailLines = tail
		}
	}

	// Get logs stream
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	logStream, errChan, err := api.podLogService.StreamLogs(ctx, namespace, podName, containerName, options)
	if err != nil {
		logging.Error("Failed to stream logs", "error", err)
		http.Error(w, "Failed to stream logs", http.StatusInternalServerError)
		return
	}

	// Stream logs to client
	for {
		select {
		case log, ok := <-logStream:
			if !ok {
				// Stream closed
				fmt.Fprintf(w, "event: close\ndata: stream ended\n\n")
				flusher.Flush()
				return
			}

			// Send log as SSE event
			data, _ := json.Marshal(log)
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			flusher.Flush()

		case err := <-errChan:
			// Error occurred
			if err != nil {
				logging.Error("Error streaming logs", "error", err)
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
			}
			return

		case <-ctx.Done():
			// Client disconnected
			return
		}
	}
}

// HandleWebSocketLogs handles WebSocket log streaming
func (api *LogsAPI) HandleWebSocketLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId := vars["taskId"]
	containerName := vars["containerName"]

	// Get cluster and region from query params
	cluster := r.URL.Query().Get("cluster")
	region := r.URL.Query().Get("region")
	if cluster == "" || region == "" {
		http.Error(w, "Both 'cluster' and 'region' query parameters are required", http.StatusBadRequest)
		return
	}

	// Try to get the actual pod name and namespace from task storage
	var namespace, podName string
	if api.storage != nil && api.storage.TaskStore() != nil {
		clusterArn := fmt.Sprintf("arn:aws:ecs:%s:000000000000:cluster/%s", region, cluster)

		// First try to find the task by ID
		task, err := api.storage.TaskStore().Get(r.Context(), clusterArn, taskId)
		if err == nil && task != nil {
			// First try to get pod info from Attributes
			attrNamespace, attrPodName := extractPodInfoFromTaskAttributes(task)
			if attrNamespace != "" && attrPodName != "" {
				namespace = attrNamespace
				podName = attrPodName
				logging.Debug("Found task with pod mapping from attributes for WebSocket",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			} else if task.PodName != "" {
				// Fallback to the stored pod name and namespace fields
				namespace = task.Namespace
				podName = task.PodName
				logging.Debug("Found task with pod mapping from fields for WebSocket",
					"taskId", taskId,
					"podName", podName,
					"namespace", namespace)
			}
		}
	}

	// Fall back to parsing if not found in storage
	if namespace == "" || podName == "" {
		// Use cluster and region from query params
		taskArn := fmt.Sprintf("arn:aws:ecs:%s:000000000000:task/%s/%s", region, cluster, taskId)

		// Parse task ARN to get namespace and pod name
		var err error
		namespace, podName, err = parseTaskArn(taskArn, region)
		if err != nil {
			http.Error(w, "Invalid task ARN", http.StatusBadRequest)
			return
		}
	}

	// Upgrade to WebSocket
	conn, err := api.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Failed to upgrade to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	// Parse query parameters
	query := r.URL.Query()
	options := &kubernetes.LogOptions{
		Follow: true, // Always follow for WebSocket
	}

	// Parse tail lines
	if tailStr := query.Get("tail"); tailStr != "" {
		if tail, err := strconv.Atoi(tailStr); err == nil && tail > 0 {
			options.TailLines = tail
		}
	}

	// Get logs stream
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logStream, errChan, err := api.podLogService.StreamLogs(ctx, namespace, podName, containerName, options)
	if err != nil {
		logging.Error("Failed to stream logs", "error", err)
		conn.WriteJSON(map[string]string{"error": "Failed to stream logs"})
		return
	}

	// Handle client messages (for closing connection)
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
		}
	}()

	// Stream logs to WebSocket
	for {
		select {
		case log, ok := <-logStream:
			if !ok {
				// Stream closed
				conn.WriteJSON(map[string]string{"event": "close"})
				return
			}

			// Send log via WebSocket
			if err := conn.WriteJSON(log); err != nil {
				logging.Error("Failed to write to WebSocket", "error", err)
				return
			}

		case err := <-errChan:
			// Error occurred
			if err != nil {
				logging.Error("Error streaming logs", "error", err)
				conn.WriteJSON(map[string]string{"event": "error", "error": err.Error()})
			}
			return

		case <-ctx.Done():
			// Context cancelled
			return
		}
	}
}

// parseTaskArn parses a task ARN to extract namespace and pod name
func parseTaskArn(taskArn string, region string) (namespace, podName string, err error) {
	// Format: arn:aws:ecs:region:account:task/cluster/task-id
	// Example: arn:aws:ecs:us-east-1:000000000000:task/default/multi-container-webapp-66dcddbdd8-x7tqc

	// Check if this is actually an ARN format
	if !strings.HasPrefix(taskArn, "arn:aws:ecs:") {
		return "", "", fmt.Errorf("not a valid task ARN format: %s", taskArn)
	}

	parts := strings.Split(taskArn, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid task ARN format: %s", taskArn)
	}

	cluster := parts[len(parts)-2]
	taskID := parts[len(parts)-1]

	// Namespace is derived from cluster name and region
	// In KECS, we use cluster-region format for namespaces
	namespace = fmt.Sprintf("%s-%s", cluster, region)

	return namespace, taskID, nil
}

// SetKubeClient sets the Kubernetes client for the LogsAPI
func (api *LogsAPI) SetKubeClient(kubeClient k8sclient.Interface) {
	api.kubeClient = kubeClient
	if kubeClient != nil {
		api.podLogService = kubernetes.NewPodLogService(kubeClient)
	}
}

// extractPodInfoFromTaskAttributes extracts pod name and namespace from task attributes
func extractPodInfoFromTaskAttributes(task *storage.Task) (namespace, podName string) {
	if task.Attributes == "" || task.Attributes == "[]" {
		return "", ""
	}

	// Parse attributes
	var attributes []map[string]interface{}
	if err := json.Unmarshal([]byte(task.Attributes), &attributes); err != nil {
		logging.Warn("Failed to unmarshal task attributes", "task", task.ARN, "error", err)
		return "", ""
	}

	// Look for pod name and namespace attributes
	for _, attr := range attributes {
		name, nameOk := attr["name"].(string)
		value, valueOk := attr["value"].(string)

		if !nameOk || !valueOk {
			continue
		}

		switch name {
		case "kecs.dev/pod-name":
			podName = value
		case "kecs.dev/pod-namespace":
			namespace = value
		}
	}

	return namespace, podName
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
	Logs []TaskLogEntry `json:"logs"`
}

// TaskLogEntry represents a single log entry
type TaskLogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Container string    `json:"container,omitempty"`
}

// HandleGetTaskLogs handles the /v1/GetTaskLogs API request
func (api *LogsAPI) HandleGetTaskLogs(w http.ResponseWriter, r *http.Request) {
	var req GetTaskLogsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "InvalidParameterValue", "Invalid request body")
		return
	}

	// Extract task ID from ARN
	taskID := extractTaskIDFromArn(req.TaskArn)
	if taskID == "" {
		api.sendError(w, http.StatusBadRequest, "InvalidParameterValue", "Invalid task ARN")
		return
	}

	// Get task from storage to validate it exists
	ctx := r.Context()
	if api.storage != nil && api.storage.TaskStore() != nil {
		clusterArn := fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:cluster/%s", req.Cluster)
		_, err := api.storage.TaskStore().Get(ctx, clusterArn, taskID)
		if err != nil {
			api.sendError(w, http.StatusNotFound, "ResourceNotFoundException", fmt.Sprintf("Task not found: %s", taskID))
			return
		}
	}

	// Get Kubernetes client
	if api.kubeClient == nil {
		logging.Error("Kubernetes client not initialized")
		api.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to connect to Kubernetes")
		return
	}

	// Extract namespace and pod name from task ARN
	namespace, podName := extractNamespaceAndPodName(req.TaskArn)

	logging.Debug("Fetching logs for task",
		"taskArn", req.TaskArn,
		"namespace", namespace,
		"podName", podName)

	// Get pod logs
	logs, err := api.getPodLogs(ctx, namespace, podName, req)
	if err != nil {
		logging.Error("Failed to get pod logs", "pod", podName, "error", err)
		api.sendError(w, http.StatusInternalServerError, "InternalError", fmt.Sprintf("Failed to get logs: %v", err))
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
func (api *LogsAPI) getPodLogs(ctx context.Context, namespace, podName string, req GetTaskLogsRequest) ([]TaskLogEntry, error) {
	// In KECS, the podName passed here is actually the task ID
	// We need to find the pod by the kecs.dev/task-id label
	var pod *corev1.Pod

	// Try to find by task-id label first (most common case)
	logging.Debug("Looking for pod by task-id label",
		"namespace", namespace,
		"taskId", podName)

	labelSelector := fmt.Sprintf("kecs.dev/task-id=%s", podName)
	pods, err := api.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})

	if err == nil && len(pods.Items) > 0 {
		pod = &pods.Items[0]
		logging.Debug("Found pod by task-id label",
			"podName", pod.Name,
			"taskId", podName)
	} else {
		// Fallback: try to get the pod directly by name
		directPod, directErr := api.kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if directErr == nil {
			pod = directPod
			logging.Debug("Found pod by direct name",
				"podName", pod.Name)
		} else {
			// Last attempt: find pod with matching name pattern
			allPods, listErr := api.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
			if listErr != nil {
				return nil, fmt.Errorf("failed to list pods: %w", listErr)
			}

			// Find pod that contains the task ID or matches the pattern
			for _, p := range allPods.Items {
				if p.Name == podName || strings.Contains(p.Name, podName) {
					pod = &p
					logging.Debug("Found pod by name pattern",
						"podName", p.Name,
						"pattern", podName)
					break
				}
			}

			if pod == nil {
				return nil, fmt.Errorf("pod not found for task: %s in namespace %s", podName, namespace)
			}
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

	var allLogs []TaskLogEntry

	// Get logs from each container
	for _, container := range pod.Spec.Containers {
		opts.Container = container.Name

		// Get log stream
		logReq := api.kubeClient.CoreV1().Pods(namespace).GetLogs(pod.Name, opts)
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
			entry := api.parseLogLine(line, container.Name, req.Timestamps)
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
	api.sortLogsByTimestamp(allLogs)

	return allLogs, nil
}

// parseLogLine parses a log line into a TaskLogEntry
func (api *LogsAPI) parseLogLine(line, containerName string, hasTimestamp bool) TaskLogEntry {
	entry := TaskLogEntry{
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
	// KECS Task ARN format: arn:aws:ecs:region:account:task/cluster/task-id
	// Example: arn:aws:ecs:us-east-1:000000000000:task/default/83ac405d374b412b97f64facff832a33
	//
	// In KECS, namespace is derived from cluster name and region
	// namespace = "{cluster}-{region}" (e.g., "default-us-east-1")

	// Extract region from ARN
	var region string
	if strings.HasPrefix(arn, "arn:aws:ecs:") {
		arnParts := strings.Split(arn, ":")
		if len(arnParts) >= 4 {
			region = arnParts[3] // region is the 4th part
		}
	}

	// Default region if not found
	if region == "" {
		region = "us-east-1"
	}

	// Extract the part after "task/"
	parts := strings.Split(arn, "task/")
	if len(parts) < 2 {
		// Fallback to default namespace and use the whole ARN as task ID
		return fmt.Sprintf("default-%s", region), arn
	}

	// Split by "/" to get cluster and task ID
	taskParts := strings.Split(parts[1], "/")
	if len(taskParts) >= 2 {
		// taskParts[0] is the cluster name (e.g., "default")
		// taskParts[1] is the task ID (e.g., "83ac405d374b412b97f64facff832a33")
		cluster := taskParts[0]
		taskID := taskParts[1]
		// Construct namespace from cluster and region
		namespace = fmt.Sprintf("%s-%s", cluster, region)
		return namespace, taskID
	}

	// If only one part, assume default cluster
	if len(taskParts) == 1 {
		return fmt.Sprintf("default-%s", region), taskParts[0]
	}

	return fmt.Sprintf("default-%s", region), arn
}

// sortLogsByTimestamp sorts log entries by timestamp
func (api *LogsAPI) sortLogsByTimestamp(logs []TaskLogEntry) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.Before(logs[j].Timestamp)
	})
}

// sendError sends an error response in AWS ECS API format
func (api *LogsAPI) sendError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"__type":  errorType,
		"message": message,
	})
}
