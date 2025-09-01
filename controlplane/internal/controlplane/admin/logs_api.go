package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	k8sclient "k8s.io/client-go/kubernetes"
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
		if err == nil && task != nil && task.PodName != "" {
			// Use the stored pod name and namespace
			namespace = task.Namespace
			podName = task.PodName
			logging.Debug("Found task with pod mapping",
				"taskId", taskId,
				"podName", podName,
				"namespace", namespace)
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
		// If we didn't get pod info from storage, parse task ARN or use taskId as pod name
		if namespace == "" || podName == "" {
			// First, try to parse as task ARN
			var err error
			namespace, podName, err = parseTaskArn(taskArn)
			if err != nil {
				// If ARN parsing fails, assume taskId is already a pod name
				// For default cluster, use default-us-east-1 namespace
				logging.Debug("Task ARN parsing failed, using taskId as pod name",
					"taskId", taskId,
					"error", err)
				podName = taskId
				namespace = "default-us-east-1" // Default namespace for KECS

				// Check if cluster was specified in query params
				if cluster := r.URL.Query().Get("cluster"); cluster != "" && cluster != "default" {
					namespace = cluster + "-us-east-1"
				}
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
		if err == nil && task != nil && task.PodName != "" {
			// Use the stored pod name and namespace
			namespace = task.Namespace
			podName = task.PodName
			logging.Debug("Found task with pod mapping for streaming",
				"taskId", taskId,
				"podName", podName,
				"namespace", namespace)
		}
	}

	// Fall back to parsing if not found in storage
	if namespace == "" || podName == "" {
		// Already have cluster from above
		taskArn := fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/default/%s", taskId)

		// Parse task ARN to get namespace and pod name
		var err error
		namespace, podName, err = parseTaskArn(taskArn)
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
		if err == nil && task != nil && task.PodName != "" {
			// Use the stored pod name and namespace
			namespace = task.Namespace
			podName = task.PodName
			logging.Debug("Found task with pod mapping for WebSocket",
				"taskId", taskId,
				"podName", podName,
				"namespace", namespace)
		}
	}

	// Fall back to parsing if not found in storage
	if namespace == "" || podName == "" {
		// Already have cluster from above
		taskArn := fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/default/%s", taskId)

		// Parse task ARN to get namespace and pod name
		var err error
		namespace, podName, err = parseTaskArn(taskArn)
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
func parseTaskArn(taskArn string) (namespace, podName string, err error) {
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

	// Namespace is derived from cluster name
	// In KECS, we use default-us-east-1 for the default cluster
	if cluster == "default" {
		namespace = "default-us-east-1"
	} else {
		namespace = cluster + "-us-east-1"
	}

	return namespace, taskID, nil
}

// SetKubeClient sets the Kubernetes client for the LogsAPI
func (api *LogsAPI) SetKubeClient(kubeClient k8sclient.Interface) {
	api.kubeClient = kubeClient
	if kubeClient != nil {
		api.podLogService = kubernetes.NewPodLogService(kubeClient)
	}
}
