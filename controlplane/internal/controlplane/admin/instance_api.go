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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Clusters  int       `json:"clusters"`
	Services  int       `json:"services"`
	Tasks     int       `json:"tasks"`
	APIPort   int       `json:"apiPort"`
	AdminPort int       `json:"adminPort"`
	CreatedAt time.Time `json:"createdAt"`
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

	// For now, return the current instance info
	// In multi-instance mode, this would list all instances
	instances := []Instance{
		{
			Name:      "default", // Current instance
			Status:    "running",
			Clusters:  api.getClusterCount(),
			Services:  api.getServiceCount(),
			Tasks:     api.getTaskCount(),
			APIPort:   api.config.Server.Port,
			AdminPort: api.config.Server.AdminPort,
			CreatedAt: time.Now().Add(-24 * time.Hour), // Mock creation time
		},
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
}