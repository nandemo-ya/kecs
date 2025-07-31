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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// ECSProxy proxies ECS API requests to the main API server
type ECSProxy struct {
	config *config.Config
}

// NewECSProxy creates a new ECS API proxy
func NewECSProxy(cfg *config.Config) *ECSProxy {
	return &ECSProxy{
		config: cfg,
	}
}

// ProxyRequest represents a generic proxy request
type ProxyRequest struct {
	Action  string          `json:"Action"`
	Version string          `json:"Version"`
	Params  json.RawMessage `json:"Params,omitempty"`
}

// handleECSProxy handles all ECS API proxy requests
func (p *ECSProxy) handleECSProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		p.sendError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "Method not allowed")
		return
	}

	// Get instance name from URL
	vars := mux.Vars(r)
	instanceName := vars["name"]
	endpoint := vars["endpoint"]

	// For now, only support default instance
	if instanceName != "default" {
		p.sendError(w, http.StatusNotFound, "InstanceNotFound", fmt.Sprintf("Instance %s not found", instanceName))
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.sendError(w, http.StatusBadRequest, "InvalidRequest", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Map endpoint to ECS action
	action := p.mapEndpointToAction(endpoint)
	if action == "" {
		p.sendError(w, http.StatusNotFound, "InvalidEndpoint", fmt.Sprintf("Unknown endpoint: %s", endpoint))
		return
	}

	// Forward to main API server
	apiURL := fmt.Sprintf("http://localhost:%d/v1/%s", p.config.Server.Port, action)
	
	// Create new request
	proxyReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		logging.Error("Failed to create proxy request", "error", err)
		p.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to create proxy request")
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Perform request
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		logging.Error("Failed to proxy request", "error", err)
		p.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to proxy request")
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

// mapEndpointToAction maps API endpoints to ECS actions
func (p *ECSProxy) mapEndpointToAction(endpoint string) string {
	switch endpoint {
	// Cluster operations
	case "clusters":
		return "ListClusters"
	case "clusters/describe":
		return "DescribeClusters"
		
	// Service operations
	case "services":
		return "ListServices"
	case "services/describe":
		return "DescribeServices"
		
	// Task operations
	case "tasks":
		return "ListTasks"
	case "tasks/describe":
		return "DescribeTasks"
	case "tasks/run":
		return "RunTask"
		
	// Task definition operations
	case "task-definitions":
		return "ListTaskDefinitions"
	case "task-definitions/register":
		return "RegisterTaskDefinition"
		
	default:
		return ""
	}
}

// handleCreateCluster handles cluster creation
func (p *ECSProxy) handleCreateCluster(w http.ResponseWriter, r *http.Request) {
	p.proxySimpleAction(w, r, "CreateCluster")
}

// handleDeleteCluster handles cluster deletion
func (p *ECSProxy) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterName := vars["cluster"]
	
	body := map[string]string{
		"cluster": clusterName,
	}
	
	p.proxyWithBody(w, r, "DeleteCluster", body)
}

// handleDeleteService handles service deletion
func (p *ECSProxy) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	p.proxySimpleAction(w, r, "DeleteService")
}

// handleStopTask handles task stopping
func (p *ECSProxy) handleStopTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskArn := vars["task"]
	
	// Read request body for cluster info
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.sendError(w, http.StatusBadRequest, "InvalidRequest", "Invalid request body")
		return
	}
	
	body := map[string]string{
		"task":    taskArn,
		"cluster": req["cluster"],
	}
	
	p.proxyWithBody(w, r, "StopTask", body)
}

// proxySimpleAction proxies a simple action with the request body as-is
func (p *ECSProxy) proxySimpleAction(w http.ResponseWriter, r *http.Request, action string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		p.sendError(w, http.StatusBadRequest, "InvalidRequest", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	p.proxyRequest(w, r, action, body)
}

// proxyWithBody proxies a request with a custom body
func (p *ECSProxy) proxyWithBody(w http.ResponseWriter, r *http.Request, action string, body interface{}) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		p.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to marshal request")
		return
	}

	p.proxyRequest(w, r, action, bodyBytes)
}

// proxyRequest performs the actual proxy request
func (p *ECSProxy) proxyRequest(w http.ResponseWriter, r *http.Request, action string, body []byte) {
	apiURL := fmt.Sprintf("http://localhost:%d/v1/%s", p.config.Server.Port, action)
	
	proxyReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		logging.Error("Failed to create proxy request", "error", err)
		p.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to create proxy request")
		return
	}

	// Copy relevant headers
	proxyReq.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		logging.Error("Failed to proxy request", "error", err)
		p.sendError(w, http.StatusInternalServerError, "InternalError", "Failed to proxy request")
		return
	}
	defer resp.Body.Close()

	// Copy response
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *ECSProxy) sendError(w http.ResponseWriter, status int, errType, message string) {
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

// RegisterRoutes registers ECS proxy routes
func (p *ECSProxy) RegisterRoutes(router *mux.Router) {
	// Generic ECS proxy endpoints
	router.HandleFunc("/api/instances/{name}/{endpoint:.*}", p.handleECSProxy).Methods("POST")
	
	// Specific endpoints that need custom handling
	router.HandleFunc("/api/instances/{name}/clusters", p.handleCreateCluster).Methods("POST")
	router.HandleFunc("/api/instances/{name}/clusters/{cluster}", p.handleDeleteCluster).Methods("DELETE")
	router.HandleFunc("/api/instances/{name}/services/{service}", p.handleDeleteService).Methods("DELETE")
	router.HandleFunc("/api/instances/{name}/tasks/{task}", p.handleStopTask).Methods("DELETE")
}