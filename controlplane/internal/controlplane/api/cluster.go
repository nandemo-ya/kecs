package api

import (
	"encoding/json"
	"net/http"
)

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                        string            `json:"clusterArn,omitempty"`
	ClusterName                       string            `json:"clusterName"`
	Status                            string            `json:"status,omitempty"`
	RegisteredContainerInstancesCount int               `json:"registeredContainerInstancesCount,omitempty"`
	RunningTasksCount                 int               `json:"runningTasksCount,omitempty"`
	PendingTasksCount                 int               `json:"pendingTasksCount,omitempty"`
	ActiveServicesCount               int               `json:"activeServicesCount,omitempty"`
	Statistics                        []KeyValuePair    `json:"statistics,omitempty"`
	Tags                              []Tag             `json:"tags,omitempty"`
	Settings                          []ClusterSetting  `json:"settings,omitempty"`
	CapacityProviders                 []string          `json:"capacityProviders,omitempty"`
	DefaultCapacityProviderStrategy   []*CapacityStrategy `json:"defaultCapacityProviderStrategy,omitempty"`
	Attachments                       []Attachment      `json:"attachments,omitempty"`
	AttachmentsStatus                 string            `json:"attachmentsStatus,omitempty"`
}

// ClusterSetting represents a setting for an ECS cluster
type ClusterSetting struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// CapacityStrategy represents a capacity provider strategy
type CapacityStrategy struct {
	CapacityProvider string `json:"capacityProvider"`
	Weight           int    `json:"weight,omitempty"`
	Base             int    `json:"base,omitempty"`
}

// KeyValuePair represents a key-value pair
type KeyValuePair struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// Tag represents a resource tag
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// Attachment represents an attachment to a resource
type Attachment struct {
	Id                 string            `json:"id,omitempty"`
	Type               string            `json:"type,omitempty"`
	Status             string            `json:"status,omitempty"`
	Details            []KeyValuePair    `json:"details,omitempty"`
}

// CreateClusterRequest represents the request to create a cluster
type CreateClusterRequest struct {
	ClusterName                     string            `json:"clusterName,omitempty"`
	Tags                            []Tag             `json:"tags,omitempty"`
	Settings                        []ClusterSetting  `json:"settings,omitempty"`
	CapacityProviders               []string          `json:"capacityProviders,omitempty"`
	DefaultCapacityProviderStrategy []*CapacityStrategy `json:"defaultCapacityProviderStrategy,omitempty"`
}

// CreateClusterResponse represents the response from creating a cluster
type CreateClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// DescribeClustersRequest represents the request to describe clusters
type DescribeClustersRequest struct {
	Clusters []string `json:"clusters,omitempty"`
	Include  []string `json:"include,omitempty"`
}

// DescribeClustersResponse represents the response from describing clusters
type DescribeClustersResponse struct {
	Clusters []Cluster `json:"clusters"`
	Failures []Failure `json:"failures,omitempty"`
}

// ListClustersRequest represents the request to list clusters
type ListClustersRequest struct {
	MaxResults int    `json:"maxResults,omitempty"`
	NextToken  string `json:"nextToken,omitempty"`
}

// ListClustersResponse represents the response from listing clusters
type ListClustersResponse struct {
	ClusterArns []string `json:"clusterArns"`
	NextToken   string   `json:"nextToken,omitempty"`
}

// DeleteClusterRequest represents the request to delete a cluster
type DeleteClusterRequest struct {
	Cluster string `json:"cluster"`
}

// DeleteClusterResponse represents the response from deleting a cluster
type DeleteClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// Failure represents a failure in an API operation
type Failure struct {
	Arn    string `json:"arn,omitempty"`
	Reason string `json:"reason,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// UpdateClusterRequest represents the request to update a cluster
type UpdateClusterRequest struct {
	Cluster                         string            `json:"cluster"`
	Settings                        []ClusterSetting  `json:"settings,omitempty"`
	CapacityProviders               []string          `json:"capacityProviders,omitempty"`
	DefaultCapacityProviderStrategy []*CapacityStrategy `json:"defaultCapacityProviderStrategy,omitempty"`
}

// UpdateClusterResponse represents the response from updating a cluster
type UpdateClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// registerClusterEndpoints registers all cluster-related API endpoints
func (s *Server) registerClusterEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/createcluster", s.handleCreateCluster)
	mux.HandleFunc("/v1/describeclusters", s.handleDescribeClusters)
	mux.HandleFunc("/v1/listclusters", s.handleListClusters)
	mux.HandleFunc("/v1/deletecluster", s.handleDeleteCluster)
	mux.HandleFunc("/v1/updatecluster", s.handleUpdateCluster)
}

// handleCreateCluster handles the CreateCluster API endpoint
func (s *Server) handleCreateCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual cluster creation logic

	// For now, return a mock response
	resp := CreateClusterResponse{
		Cluster: Cluster{
			ClusterArn:  "arn:aws:ecs:region:account:cluster/" + req.ClusterName,
			ClusterName: req.ClusterName,
			Status:      "ACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeClusters handles the DescribeClusters API endpoint
func (s *Server) handleDescribeClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeClustersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual cluster description logic

	// For now, return a mock response
	resp := DescribeClustersResponse{
		Clusters: []Cluster{
			{
				ClusterArn:  "arn:aws:ecs:region:account:cluster/default",
				ClusterName: "default",
				Status:      "ACTIVE",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListClusters handles the ListClusters API endpoint
func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListClustersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual cluster listing logic

	// For now, return a mock response
	resp := ListClustersResponse{
		ClusterArns: []string{"arn:aws:ecs:region:account:cluster/default"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteCluster handles the DeleteCluster API endpoint
func (s *Server) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual cluster deletion logic

	// For now, return a mock response
	resp := DeleteClusterResponse{
		Cluster: Cluster{
			ClusterArn:  "arn:aws:ecs:region:account:cluster/" + req.Cluster,
			ClusterName: req.Cluster,
			Status:      "INACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateCluster handles the UpdateCluster API endpoint
func (s *Server) handleUpdateCluster(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual cluster update logic

	// For now, return a mock response
	resp := UpdateClusterResponse{
		Cluster: Cluster{
			ClusterArn:  "arn:aws:ecs:region:account:cluster/" + req.Cluster,
			ClusterName: req.Cluster,
			Status:      "ACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
