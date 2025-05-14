package api

import (
	"encoding/json"
	"net/http"
)

// ContainerInstance represents an ECS container instance
type ContainerInstance struct {
	ContainerInstanceArn       string                 `json:"containerInstanceArn,omitempty"`
	Ec2InstanceId              string                 `json:"ec2InstanceId,omitempty"`
	Version                    int                    `json:"version,omitempty"`
	VersionInfo                *VersionInfo           `json:"versionInfo,omitempty"`
	RemainingResources         []Resource             `json:"remainingResources,omitempty"`
	RegisteredResources        []Resource             `json:"registeredResources,omitempty"`
	Status                     string                 `json:"status,omitempty"`
	StatusReason               string                 `json:"statusReason,omitempty"`
	AgentConnected             bool                   `json:"agentConnected,omitempty"`
	RunningTasksCount          int                    `json:"runningTasksCount,omitempty"`
	PendingTasksCount          int                    `json:"pendingTasksCount,omitempty"`
	AgentUpdateStatus          string                 `json:"agentUpdateStatus,omitempty"`
	Attributes                 []Attribute            `json:"attributes,omitempty"`
	RegisteredAt               string                 `json:"registeredAt,omitempty"`
	Attachments                []Attachment           `json:"attachments,omitempty"`
	Tags                       []Tag                  `json:"tags,omitempty"`
	HealthStatus               string                 `json:"healthStatus,omitempty"`
	CapacityProviderName       string                 `json:"capacityProviderName,omitempty"`
}

// VersionInfo represents version information for a container instance
type VersionInfo struct {
	AgentVersion  string `json:"agentVersion,omitempty"`
	AgentHash     string `json:"agentHash,omitempty"`
	DockerVersion string `json:"dockerVersion,omitempty"`
}

// Resource represents a resource for a container instance
type Resource struct {
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	DoubleValue    float64        `json:"doubleValue,omitempty"`
	LongValue      int64          `json:"longValue,omitempty"`
	IntegerValue   int            `json:"integerValue,omitempty"`
	StringSetValue []string       `json:"stringSetValue,omitempty"`
}

// RegisterContainerInstanceRequest represents the request to register a container instance
type RegisterContainerInstanceRequest struct {
	Cluster                       string      `json:"cluster,omitempty"`
	InstanceIdentityDocumentAndSignature string `json:"instanceIdentityDocumentAndSignature,omitempty"`
	InstanceIdentityDocument      string      `json:"instanceIdentityDocument,omitempty"`
	TotalResources                []Resource  `json:"totalResources,omitempty"`
	VersionInfo                   *VersionInfo `json:"versionInfo,omitempty"`
	ContainerInstanceArn          string      `json:"containerInstanceArn,omitempty"`
	Attributes                    []Attribute `json:"attributes,omitempty"`
	Tags                          []Tag       `json:"tags,omitempty"`
}

// RegisterContainerInstanceResponse represents the response from registering a container instance
type RegisterContainerInstanceResponse struct {
	ContainerInstance ContainerInstance `json:"containerInstance"`
}

// DeregisterContainerInstanceRequest represents the request to deregister a container instance
type DeregisterContainerInstanceRequest struct {
	Cluster           string `json:"cluster,omitempty"`
	ContainerInstance string `json:"containerInstance"`
	Force             bool   `json:"force,omitempty"`
}

// DeregisterContainerInstanceResponse represents the response from deregistering a container instance
type DeregisterContainerInstanceResponse struct {
	ContainerInstance ContainerInstance `json:"containerInstance"`
}

// DescribeContainerInstancesRequest represents the request to describe container instances
type DescribeContainerInstancesRequest struct {
	Cluster            string   `json:"cluster,omitempty"`
	ContainerInstances []string `json:"containerInstances"`
	Include            []string `json:"include,omitempty"`
}

// DescribeContainerInstancesResponse represents the response from describing container instances
type DescribeContainerInstancesResponse struct {
	ContainerInstances []ContainerInstance `json:"containerInstances"`
	Failures           []Failure           `json:"failures,omitempty"`
}

// ListContainerInstancesRequest represents the request to list container instances
type ListContainerInstancesRequest struct {
	Cluster    string `json:"cluster,omitempty"`
	Filter     string `json:"filter,omitempty"`
	NextToken  string `json:"nextToken,omitempty"`
	MaxResults int    `json:"maxResults,omitempty"`
	Status     string `json:"status,omitempty"`
}

// ListContainerInstancesResponse represents the response from listing container instances
type ListContainerInstancesResponse struct {
	ContainerInstanceArns []string `json:"containerInstanceArns"`
	NextToken             string   `json:"nextToken,omitempty"`
}

// UpdateContainerInstancesStateRequest represents the request to update container instances state
type UpdateContainerInstancesStateRequest struct {
	Cluster            string   `json:"cluster,omitempty"`
	ContainerInstances []string `json:"containerInstances"`
	Status             string   `json:"status"`
}

// UpdateContainerInstancesStateResponse represents the response from updating container instances state
type UpdateContainerInstancesStateResponse struct {
	ContainerInstances []ContainerInstance `json:"containerInstances"`
	Failures           []Failure           `json:"failures,omitempty"`
}

// UpdateContainerAgentRequest represents the request to update a container agent
type UpdateContainerAgentRequest struct {
	Cluster           string `json:"cluster,omitempty"`
	ContainerInstance string `json:"containerInstance"`
}

// UpdateContainerAgentResponse represents the response from updating a container agent
type UpdateContainerAgentResponse struct {
	ContainerInstance ContainerInstance `json:"containerInstance"`
}

// registerContainerInstanceEndpoints registers all container instance-related API endpoints
func (s *Server) registerContainerInstanceEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/registercontainerinstance", s.handleRegisterContainerInstance)
	mux.HandleFunc("/v1/deregistercontainerinstance", s.handleDeregisterContainerInstance)
	mux.HandleFunc("/v1/describecontainerinstances", s.handleDescribeContainerInstances)
	mux.HandleFunc("/v1/listcontainerinstances", s.handleListContainerInstances)
	mux.HandleFunc("/v1/updatecontainerinstancesstate", s.handleUpdateContainerInstancesState)
	mux.HandleFunc("/v1/updatecontaineragent", s.handleUpdateContainerAgent)
}

// handleRegisterContainerInstance handles the RegisterContainerInstance API endpoint
func (s *Server) handleRegisterContainerInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterContainerInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance registration logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	containerInstanceArn := req.ContainerInstanceArn
	if containerInstanceArn == "" {
		containerInstanceArn = "arn:aws:ecs:region:account:container-instance/" + cluster + "/container-instance-id"
	}

	resp := RegisterContainerInstanceResponse{
		ContainerInstance: ContainerInstance{
			ContainerInstanceArn: containerInstanceArn,
			Status:              "ACTIVE",
			AgentConnected:      true,
			RegisteredAt:        "2025-05-15T00:40:35+09:00",
			VersionInfo:         req.VersionInfo,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeregisterContainerInstance handles the DeregisterContainerInstance API endpoint
func (s *Server) handleDeregisterContainerInstance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeregisterContainerInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance deregistration logic

	// For now, return a mock response
	// Note: In a real implementation, we would use the cluster information

	resp := DeregisterContainerInstanceResponse{
		ContainerInstance: ContainerInstance{
			ContainerInstanceArn: req.ContainerInstance,
			Status:              "INACTIVE",
			AgentConnected:      false,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeContainerInstances handles the DescribeContainerInstances API endpoint
func (s *Server) handleDescribeContainerInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeContainerInstancesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance description logic

	// For now, return a mock response
	// Note: In a real implementation, we would use the cluster information from the request

	resp := DescribeContainerInstancesResponse{
		ContainerInstances: []ContainerInstance{
			{
				ContainerInstanceArn: req.ContainerInstances[0],
				Status:              "ACTIVE",
				AgentConnected:      true,
				RunningTasksCount:   0,
				PendingTasksCount:   0,
				RegisteredAt:        "2025-05-15T00:40:35+09:00",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListContainerInstances handles the ListContainerInstances API endpoint
func (s *Server) handleListContainerInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListContainerInstancesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance listing logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := ListContainerInstancesResponse{
		ContainerInstanceArns: []string{"arn:aws:ecs:region:account:container-instance/" + cluster + "/container-instance-id"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateContainerInstancesState handles the UpdateContainerInstancesState API endpoint
func (s *Server) handleUpdateContainerInstancesState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateContainerInstancesStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance state update logic

	// For now, return a mock response
	resp := UpdateContainerInstancesStateResponse{
		ContainerInstances: []ContainerInstance{
			{
				ContainerInstanceArn: req.ContainerInstances[0],
				Status:              req.Status,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateContainerAgent handles the UpdateContainerAgent API endpoint
func (s *Server) handleUpdateContainerAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateContainerAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container agent update logic

	// For now, return a mock response
	resp := UpdateContainerAgentResponse{
		ContainerInstance: ContainerInstance{
			ContainerInstanceArn: req.ContainerInstance,
			Status:              "ACTIVE",
			AgentConnected:      true,
			AgentUpdateStatus:   "UPDATED",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
