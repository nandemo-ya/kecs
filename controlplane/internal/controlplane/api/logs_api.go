package api

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
	return &LogsAPI{
		storage:       storage,
		kubeClient:    kubeClient,
		podLogService: kubernetes.NewPodLogService(kubeClient),
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
	// REST endpoints (paths relative to /api prefix)
	router.HandleFunc("/tasks/{taskArn}/containers/{containerName}/logs", api.HandleGetLogs).Methods("GET")
	router.HandleFunc("/tasks/{taskArn}/containers/{containerName}/logs/stream", api.HandleStreamLogs).Methods("GET")

	// WebSocket endpoint for real-time streaming
	router.HandleFunc("/tasks/{taskArn}/containers/{containerName}/logs/ws", api.HandleWebSocketLogs).Methods("GET")
}

// HandleGetLogs retrieves logs from storage (historical logs)
func (api *LogsAPI) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskArn := vars["taskArn"]
	containerName := vars["containerName"]

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
	logs, err := api.storage.TaskLogStore().GetLogs(r.Context(), filter)
	if err != nil {
		logging.Error("Failed to get logs from storage", "error", err)
		http.Error(w, "Failed to retrieve logs", http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	totalCount, err := api.storage.TaskLogStore().GetLogCount(r.Context(), filter)
	if err != nil {
		logging.Warn("Failed to get log count", "error", err)
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
	taskArn := vars["taskArn"]
	containerName := vars["containerName"]

	// Parse task ARN to get namespace and pod name
	namespace, podName, err := parseTaskArn(taskArn)
	if err != nil {
		http.Error(w, "Invalid task ARN", http.StatusBadRequest)
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

	// Parse since time
	if sinceStr := query.Get("since"); sinceStr != "" {
		if since, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			options.SinceTime = &since
		}
	}

	// Set up SSE (Server-Sent Events) for streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Stream logs
	if options.Follow {
		// Real-time streaming
		logChan, errChan, err := api.podLogService.StreamLogs(r.Context(), namespace, podName, containerName, options)
		if err != nil {
			logging.Error("Failed to start log stream", "error", err)
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return
		}

		for {
			select {
			case log, ok := <-logChan:
				if !ok {
					// Stream closed
					fmt.Fprintf(w, "event: close\ndata: stream ended\n\n")
					flusher.Flush()
					return
				}

				// Send log as SSE event
				data, _ := json.Marshal(log)
				fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(data))
				flusher.Flush()

			case err := <-errChan:
				if err != nil {
					logging.Error("Error in log stream", "error", err)
					fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
					flusher.Flush()
				}
				return

			case <-r.Context().Done():
				// Client disconnected
				return
			}
		}
	} else {
		// One-time fetch
		logs, err := api.podLogService.GetLogs(r.Context(), namespace, podName, containerName, options)
		if err != nil {
			logging.Error("Failed to get logs", "error", err)
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			flusher.Flush()
			return
		}

		// Send all logs
		for _, log := range logs {
			data, _ := json.Marshal(log)
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(data))
			flusher.Flush()
		}

		// Send close event
		fmt.Fprintf(w, "event: close\ndata: complete\n\n")
		flusher.Flush()
	}
}

// HandleWebSocketLogs handles WebSocket connections for real-time log streaming
func (api *LogsAPI) HandleWebSocketLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskArn := vars["taskArn"]
	containerName := vars["containerName"]

	// Parse task ARN to get namespace and pod name
	namespace, podName, err := parseTaskArn(taskArn)
	if err != nil {
		http.Error(w, "Invalid task ARN", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := api.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Error("Failed to upgrade to WebSocket", "error", err)
		return
	}
	defer conn.Close()

	// Set up log streaming options
	options := &kubernetes.LogOptions{
		Follow:    true,
		TailLines: 100, // Default tail lines
	}

	// Create context that cancels when WebSocket closes
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Start goroutine to handle incoming messages (commands from client)
	go func() {
		for {
			var msg map[string]interface{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				// Connection closed
				cancel()
				return
			}

			// Handle client commands
			if cmd, ok := msg["command"].(string); ok {
				switch cmd {
				case "stop":
					cancel()
					return
				case "filter":
					// TODO: Implement log filtering
				}
			}
		}
	}()

	// Stream logs
	logChan, errChan, err := api.podLogService.StreamLogs(ctx, namespace, podName, containerName, options)
	if err != nil {
		logging.Error("Failed to start log stream", "error", err)
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": err.Error(),
		})
		return
	}

	// Send logs to WebSocket
	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				// Stream closed
				conn.WriteJSON(map[string]interface{}{
					"type":    "close",
					"message": "stream ended",
				})
				return
			}

			// Send log to client
			err := conn.WriteJSON(map[string]interface{}{
				"type": "log",
				"data": log,
			})
			if err != nil {
				// Client disconnected
				return
			}

		case err := <-errChan:
			if err != nil {
				logging.Error("Error in log stream", "error", err)
				conn.WriteJSON(map[string]interface{}{
					"type":  "error",
					"error": err.Error(),
				})
			}
			return

		case <-ctx.Done():
			// Context cancelled
			return
		}
	}
}

// parseTaskArn extracts namespace and pod name from task ARN
// Format: arn:aws:ecs:region:account:task/cluster/task-id
func parseTaskArn(taskArn string) (namespace, podName string, err error) {
	parts := strings.Split(taskArn, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid task ARN format")
	}

	// Extract cluster and task ID
	cluster := parts[len(parts)-2]
	taskID := parts[len(parts)-1]

	// Namespace is derived from cluster name
	namespace = cluster + "-us-east-1" // TODO: Make region configurable

	// Pod name is the task ID
	podName = taskID

	// Handle the full ARN format
	if strings.HasPrefix(taskArn, "arn:aws:ecs:") {
		arnParts := strings.Split(taskArn, ":")
		if len(arnParts) >= 6 {
			// Extract region from ARN
			region := arnParts[3]
			namespace = cluster + "-" + region
		}
	}

	return namespace, podName, nil
}

// LogStreamMessage represents a message in the log stream
type LogStreamMessage struct {
	Type      string           `json:"type"` // "log", "error", "close"
	Data      *storage.TaskLog `json:"data,omitempty"`
	Error     string           `json:"error,omitempty"`
	Message   string           `json:"message,omitempty"`
	Timestamp time.Time        `json:"timestamp"`
}
