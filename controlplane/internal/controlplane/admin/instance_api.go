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

package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v3"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/instance"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// InstanceAPI handles instance-related API requests
type InstanceAPI struct {
	config  *config.Config
	manager *instance.Manager
	storage storage.Storage
}

// NewInstanceAPI creates a new instance API handler
func NewInstanceAPI(cfg *config.Config, storage storage.Storage) (*InstanceAPI, error) {
	mgr, err := instance.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create instance manager: %w", err)
	}

	return &InstanceAPI{
		config:  cfg,
		manager: mgr,
		storage: storage,
	}, nil
}

// Instance represents a KECS instance in API responses
type Instance struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Clusters   int       `json:"clusters"`
	Services   int       `json:"services"`
	Tasks      int       `json:"tasks"`
	APIPort    int       `json:"apiPort"`
	AdminPort  int       `json:"adminPort"`
	LocalStack bool      `json:"localStack"`
	Traefik    bool      `json:"traefik"`
	DevMode    bool      `json:"devMode"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CreateInstanceRequest represents the request to create a new instance
type CreateInstanceRequest struct {
	Name       string `json:"name"`
	APIPort    int    `json:"apiPort"`
	AdminPort  int    `json:"adminPort"`
	LocalStack bool   `json:"localStack"`
	Traefik    bool   `json:"traefik"`
	DevMode    bool   `json:"devMode"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

// handleListInstances handles GET /api/instances
func (api *InstanceAPI) handleListInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		api.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to get home directory")
		return
	}

	// Read instances from ~/.kecs/instances/
	instancesDir := filepath.Join(homeDir, ".kecs", "instances")
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		// If directory doesn't exist, return empty list
		if os.IsNotExist(err) {
			api.sendJSON(w, []Instance{})
			return
		}
		api.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to read instances directory")
		return
	}

	var instances []Instance
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Read instance config
		configPath := filepath.Join(instancesDir, entry.Name(), "config.yaml")
		configData, err := os.ReadFile(configPath)
		if err != nil {
			logging.Error("Failed to read instance config", "instance", entry.Name(), "error", err)
			continue
		}

		// Parse config
		var config struct {
			Name       string    `yaml:"name"`
			CreatedAt  time.Time `yaml:"createdAt"`
			APIPort    int       `yaml:"apiPort"`
			AdminPort  int       `yaml:"adminPort"`
			LocalStack bool      `yaml:"localStack"`
			Traefik    bool      `yaml:"traefik"`
			DevMode    bool      `yaml:"devMode"`
		}
		if err := yaml.Unmarshal(configData, &config); err != nil {
			logging.Error("Failed to parse instance config", "instance", entry.Name(), "error", err)
			continue
		}

		// Get cluster, service, task counts by calling the instance's API
		clusters, services, tasks := api.getInstanceCounts(config.APIPort)

		instances = append(instances, Instance{
			Name:       config.Name,
			Status:     "running", // TODO: Check actual status
			Clusters:   clusters,
			Services:   services,
			Tasks:      tasks,
			APIPort:    config.APIPort,
			AdminPort:  config.AdminPort,
			LocalStack: config.LocalStack,
			Traefik:    config.Traefik,
			DevMode:    config.DevMode,
			CreatedAt:  config.CreatedAt,
		})
	}

	api.sendJSON(w, instances)
}

// handleGetInstance handles GET /api/instances/{name}
func (api *InstanceAPI) handleGetInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]

	// For now, only support "default" instance
	if name != "default" {
		api.sendError(w, http.StatusNotFound, "InstanceNotFound", fmt.Sprintf("Instance %s not found", name))
		return
	}

	instance := Instance{
		Name:      name,
		Status:    "running",
		Clusters:  api.getClusterCount(),
		Services:  api.getServiceCount(),
		Tasks:     api.getTaskCount(),
		APIPort:   api.config.Server.Port,
		AdminPort: api.config.Server.AdminPort,
		CreatedAt: time.Now().Add(-24 * time.Hour),
	}

	api.sendJSON(w, instance)
}

// handleCreateInstance handles POST /api/instances
func (api *InstanceAPI) handleCreateInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.sendError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	var req CreateInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.sendError(w, http.StatusBadRequest, "InvalidRequest", "Invalid request body")
		return
	}

	// Validate request
	if req.Name == "" {
		api.sendError(w, http.StatusBadRequest, "InvalidParameter", "Instance name is required")
		return
	}

	// For now, return error as multi-instance is not yet supported
	// In the future, this would create a new k3d cluster
	api.sendError(w, http.StatusNotImplemented, "NotImplemented", "Multi-instance support coming soon")
}

// handleDeleteInstance handles DELETE /api/instances/{name}
func (api *InstanceAPI) handleDeleteInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.sendError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]

	// For now, prevent deletion of default instance
	if name == "default" {
		api.sendError(w, http.StatusForbidden, "OperationNotPermitted", "Cannot delete default instance")
		return
	}

	api.sendError(w, http.StatusNotImplemented, "NotImplemented", "Multi-instance support coming soon")
}

// handleInstanceHealth handles GET /api/instances/{name}/health
func (api *InstanceAPI) handleInstanceHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.sendError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]

	if name != "default" {
		api.sendError(w, http.StatusNotFound, "InstanceNotFound", fmt.Sprintf("Instance %s not found", name))
		return
	}

	// Simple health check response
	health := map[string]interface{}{
		"status":  "healthy",
		"version": getVersion(),
		"time":    time.Now(),
	}

	api.sendJSON(w, health)
}

// Helper methods

// getInstanceCounts retrieves cluster, service, and task counts from an instance's API
func (api *InstanceAPI) getInstanceCounts(apiPort int) (clusters, services, tasks int) {
	// Call ListClusters API
	clustersResp, err := api.callInstanceAPI(apiPort, "ListClusters", map[string]interface{}{})
	if err != nil {
		logging.Debug("Failed to get cluster count from instance", "port", apiPort, "error", err)
		return 0, 0, 0
	}

	// Parse cluster response
	if clusterArns, ok := clustersResp["clusterArns"].([]interface{}); ok {
		clusters = len(clusterArns)

		// For each cluster, get services and tasks
		for _, arnInterface := range clusterArns {
			if arn, ok := arnInterface.(string); ok {
				// Extract cluster name from ARN
				clusterName := extractClusterName(arn)

				// Get services count
				servicesResp, err := api.callInstanceAPI(apiPort, "ListServices", map[string]interface{}{
					"cluster": clusterName,
				})
				if err == nil {
					if serviceArns, ok := servicesResp["serviceArns"].([]interface{}); ok {
						services += len(serviceArns)
					}
				}

				// Get tasks count
				tasksResp, err := api.callInstanceAPI(apiPort, "ListTasks", map[string]interface{}{
					"cluster": clusterName,
				})
				if err == nil {
					if taskArns, ok := tasksResp["taskArns"].([]interface{}); ok {
						tasks += len(taskArns)
					}
				}
			}
		}
	}

	return clusters, services, tasks
}

// callInstanceAPI makes an API call to a specific instance
func (api *InstanceAPI) callInstanceAPI(apiPort int, action string, params map[string]interface{}) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://localhost:%d/v1/%s", apiPort, action)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// extractClusterName extracts the cluster name from an ARN
func extractClusterName(arn string) string {
	// ARN format: arn:aws:ecs:region:account:cluster/name
	parts := strings.Split(arn, "/")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	// If not in ARN format, assume it's already the cluster name
	return arn
}

func (api *InstanceAPI) getClusterCount() int {
	if api.storage == nil {
		return 0
	}

	ctx := context.Background()
	clusters, err := api.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Failed to count clusters", "error", err)
		return 0
	}
	return len(clusters)
}

func (api *InstanceAPI) getServiceCount() int {
	if api.storage == nil {
		return 0
	}

	ctx := context.Background()
	// Count services across all clusters
	count := 0
	clusters, err := api.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Failed to list clusters for service count", "error", err)
		return 0
	}

	for _, cluster := range clusters {
		// List all services in the cluster (no filtering)
		services, _, err := api.storage.ServiceStore().List(ctx, cluster.Name, "", "", 1000, "")
		if err != nil {
			logging.Error("Failed to count services", "cluster", cluster.Name, "error", err)
			continue
		}
		count += len(services)
	}
	return count
}

func (api *InstanceAPI) getTaskCount() int {
	if api.storage == nil {
		return 0
	}

	ctx := context.Background()
	// Count tasks across all clusters
	count := 0
	clusters, err := api.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Error("Failed to list clusters for task count", "error", err)
		return 0
	}

	for _, cluster := range clusters {
		// List all tasks in the cluster
		tasks, err := api.storage.TaskStore().List(ctx, cluster.Name, storage.TaskFilters{
			MaxResults: 1000, // Get up to 1000 tasks per cluster
		})
		if err != nil {
			logging.Error("Failed to count tasks", "cluster", cluster.Name, "error", err)
			continue
		}
		count += len(tasks)
	}
	return count
}

func (api *InstanceAPI) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logging.Error("Failed to encode JSON response", "error", err)
	}
}

// handleGetCreationStatus handles GET /api/instances/{name}/creation-status
func (api *InstanceAPI) handleGetCreationStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instanceName := vars["name"]

	// Get creation status from manager
	status := api.manager.GetCreationStatus(instanceName)
	if status == nil {
		// No status means creation is complete or not started
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Return status as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		logging.Error("Failed to encode creation status", "error", err)
		api.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to encode response")
	}
}

func (api *InstanceAPI) sendError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := ErrorResponse{
		Type:    errType,
		Message: message,
	}
	if encErr := json.NewEncoder(w).Encode(err); encErr != nil {
		logging.Error("Failed to encode error response", "error", encErr)
	}
}

// RegisterRoutes registers instance API routes
func (api *InstanceAPI) RegisterRoutes(router *mux.Router) {
	// Instance endpoints
	router.HandleFunc("/api/instances", api.handleListInstances).Methods("GET")
	router.HandleFunc("/api/instances", api.handleCreateInstance).Methods("POST")
	router.HandleFunc("/api/instances/{name}", api.handleGetInstance).Methods("GET")
	router.HandleFunc("/api/instances/{name}", api.handleDeleteInstance).Methods("DELETE")
	router.HandleFunc("/api/instances/{name}/health", api.handleInstanceHealth).Methods("GET")
	router.HandleFunc("/api/instances/{name}/creation-status", api.handleGetCreationStatus).Methods("GET")
}
