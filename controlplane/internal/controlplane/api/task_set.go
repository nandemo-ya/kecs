package api

import (
	"encoding/json"
	"net/http"
)

// CreateTaskSetRequest represents the request to create a task set
type CreateTaskSetRequest struct {
	Service           string                `json:"service"`
	Cluster           string                `json:"cluster,omitempty"`
	ExternalId        string                `json:"externalId,omitempty"`
	TaskDefinition    string                `json:"taskDefinition"`
	NetworkConfiguration *NetworkConfiguration `json:"networkConfiguration,omitempty"`
	LoadBalancers     []LoadBalancer        `json:"loadBalancers,omitempty"`
	ServiceRegistries []ServiceRegistry     `json:"serviceRegistries,omitempty"`
	LaunchType        string                `json:"launchType,omitempty"`
	CapacityProviderStrategy []*CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
	PlatformVersion   string                `json:"platformVersion,omitempty"`
	Scale             *Scale                `json:"scale,omitempty"`
	ClientToken       string                `json:"clientToken,omitempty"`
	Tags              []Tag                 `json:"tags,omitempty"`
}

// CreateTaskSetResponse represents the response from creating a task set
type CreateTaskSetResponse struct {
	TaskSet TaskSet `json:"taskSet"`
}

// DeleteTaskSetRequest represents the request to delete a task set
type DeleteTaskSetRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Service string `json:"service"`
	TaskSet string `json:"taskSet"`
	Force   bool   `json:"force,omitempty"`
}

// DeleteTaskSetResponse represents the response from deleting a task set
type DeleteTaskSetResponse struct {
	TaskSet TaskSet `json:"taskSet"`
}

// DescribeTaskSetsRequest represents the request to describe task sets
type DescribeTaskSetsRequest struct {
	Cluster  string   `json:"cluster,omitempty"`
	Service  string   `json:"service"`
	TaskSets []string `json:"taskSets,omitempty"`
	Include  []string `json:"include,omitempty"`
}

// DescribeTaskSetsResponse represents the response from describing task sets
type DescribeTaskSetsResponse struct {
	TaskSets []TaskSet `json:"taskSets"`
	Failures []Failure `json:"failures,omitempty"`
}

// UpdateTaskSetRequest represents the request to update a task set
type UpdateTaskSetRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Service string `json:"service"`
	TaskSet string `json:"taskSet"`
	Scale   *Scale `json:"scale"`
}

// UpdateTaskSetResponse represents the response from updating a task set
type UpdateTaskSetResponse struct {
	TaskSet TaskSet `json:"taskSet"`
}

// registerTaskSetEndpoints registers all task set-related API endpoints
func (s *Server) registerTaskSetEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/createtaskset", s.handleCreateTaskSet)
	mux.HandleFunc("/v1/deletetaskset", s.handleDeleteTaskSet)
	mux.HandleFunc("/v1/describetasksets", s.handleDescribeTaskSets)
	mux.HandleFunc("/v1/updatetaskset", s.handleUpdateTaskSet)
}

// handleCreateTaskSet handles the CreateTaskSet API endpoint
func (s *Server) handleCreateTaskSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateTaskSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set creation logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := CreateTaskSetResponse{
		TaskSet: TaskSet{
			Id:             "ts-1234567890",
			TaskSetArn:     "arn:aws:ecs:region:account:task-set/" + cluster + "/" + req.Service + "/ts-1234567890",
			ServiceArn:     "arn:aws:ecs:region:account:service/" + cluster + "/" + req.Service,
			ClusterArn:     "arn:aws:ecs:region:account:cluster/" + cluster,
			ExternalId:     req.ExternalId,
			Status:         "ACTIVE",
			TaskDefinition: req.TaskDefinition,
			Scale: &Scale{
				Value: 100.0,
				Unit:  "PERCENT",
			},
			StabilityStatus: "STEADY_STATE",
			CreatedAt:       "2025-05-15T00:50:29+09:00",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteTaskSet handles the DeleteTaskSet API endpoint
func (s *Server) handleDeleteTaskSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteTaskSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set deletion logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := DeleteTaskSetResponse{
		TaskSet: TaskSet{
			Id:             req.TaskSet,
			TaskSetArn:     "arn:aws:ecs:region:account:task-set/" + cluster + "/" + req.Service + "/" + req.TaskSet,
			ServiceArn:     "arn:aws:ecs:region:account:service/" + cluster + "/" + req.Service,
			ClusterArn:     "arn:aws:ecs:region:account:cluster/" + cluster,
			Status:         "DRAINING",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeTaskSets handles the DescribeTaskSets API endpoint
func (s *Server) handleDescribeTaskSets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeTaskSetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set description logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskSets := []TaskSet{}
	if len(req.TaskSets) > 0 {
		for _, taskSetId := range req.TaskSets {
			taskSets = append(taskSets, TaskSet{
				Id:             taskSetId,
				TaskSetArn:     "arn:aws:ecs:region:account:task-set/" + cluster + "/" + req.Service + "/" + taskSetId,
				ServiceArn:     "arn:aws:ecs:region:account:service/" + cluster + "/" + req.Service,
				ClusterArn:     "arn:aws:ecs:region:account:cluster/" + cluster,
				Status:         "ACTIVE",
				TaskDefinition: "arn:aws:ecs:region:account:task-definition/family:1",
				Scale: &Scale{
					Value: 100.0,
					Unit:  "PERCENT",
				},
				StabilityStatus: "STEADY_STATE",
				CreatedAt:       "2025-05-15T00:50:29+09:00",
			})
		}
	} else {
		// Return default task set if none specified
		taskSets = append(taskSets, TaskSet{
			Id:             "ts-1234567890",
			TaskSetArn:     "arn:aws:ecs:region:account:task-set/" + cluster + "/" + req.Service + "/ts-1234567890",
			ServiceArn:     "arn:aws:ecs:region:account:service/" + cluster + "/" + req.Service,
			ClusterArn:     "arn:aws:ecs:region:account:cluster/" + cluster,
			Status:         "ACTIVE",
			TaskDefinition: "arn:aws:ecs:region:account:task-definition/family:1",
			Scale: &Scale{
				Value: 100.0,
				Unit:  "PERCENT",
			},
			StabilityStatus: "STEADY_STATE",
			CreatedAt:       "2025-05-15T00:50:29+09:00",
		})
	}

	resp := DescribeTaskSetsResponse{
		TaskSets: taskSets,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateTaskSet handles the UpdateTaskSet API endpoint
func (s *Server) handleUpdateTaskSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateTaskSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set update logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := UpdateTaskSetResponse{
		TaskSet: TaskSet{
			Id:             req.TaskSet,
			TaskSetArn:     "arn:aws:ecs:region:account:task-set/" + cluster + "/" + req.Service + "/" + req.TaskSet,
			ServiceArn:     "arn:aws:ecs:region:account:service/" + cluster + "/" + req.Service,
			ClusterArn:     "arn:aws:ecs:region:account:cluster/" + cluster,
			Status:         "ACTIVE",
			Scale:          req.Scale,
			StabilityStatus: "UPDATING",
			UpdatedAt:       "2025-05-15T00:50:29+09:00",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
