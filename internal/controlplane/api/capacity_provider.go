package api

import (
	"encoding/json"
	"net/http"
)

// CapacityProvider represents an ECS capacity provider
type CapacityProvider struct {
	CapacityProviderArn          string                       `json:"capacityProviderArn,omitempty"`
	Name                         string                       `json:"name"`
	Status                       string                       `json:"status,omitempty"`
	AutoScalingGroupProvider     *AutoScalingGroupProvider    `json:"autoScalingGroupProvider,omitempty"`
	UpdateStatus                 string                       `json:"updateStatus,omitempty"`
	UpdateStatusReason           string                       `json:"updateStatusReason,omitempty"`
	Tags                         []Tag                        `json:"tags,omitempty"`
}

// AutoScalingGroupProvider represents an auto scaling group provider for a capacity provider
type AutoScalingGroupProvider struct {
	AutoScalingGroupArn            string                     `json:"autoScalingGroupArn"`
	ManagedScaling                 *ManagedScaling            `json:"managedScaling,omitempty"`
	ManagedTerminationProtection   string                     `json:"managedTerminationProtection,omitempty"`
	ManagedDraining                string                     `json:"managedDraining,omitempty"`
}

// ManagedScaling represents managed scaling for an auto scaling group provider
type ManagedScaling struct {
	Status                          string                    `json:"status,omitempty"`
	TargetCapacity                  int                       `json:"targetCapacity,omitempty"`
	MinimumScalingStepSize          int                       `json:"minimumScalingStepSize,omitempty"`
	MaximumScalingStepSize          int                       `json:"maximumScalingStepSize,omitempty"`
	InstanceWarmupPeriod            int                       `json:"instanceWarmupPeriod,omitempty"`
}

// CreateCapacityProviderRequest represents the request to create a capacity provider
type CreateCapacityProviderRequest struct {
	Name                         string                       `json:"name"`
	AutoScalingGroupProvider     *AutoScalingGroupProvider    `json:"autoScalingGroupProvider"`
	Tags                         []Tag                        `json:"tags,omitempty"`
}

// CreateCapacityProviderResponse represents the response from creating a capacity provider
type CreateCapacityProviderResponse struct {
	CapacityProvider CapacityProvider `json:"capacityProvider"`
}

// UpdateCapacityProviderRequest represents the request to update a capacity provider
type UpdateCapacityProviderRequest struct {
	Name                         string                       `json:"name"`
	AutoScalingGroupProvider     *AutoScalingGroupProvider    `json:"autoScalingGroupProvider"`
}

// UpdateCapacityProviderResponse represents the response from updating a capacity provider
type UpdateCapacityProviderResponse struct {
	CapacityProvider CapacityProvider `json:"capacityProvider"`
}

// DeleteCapacityProviderRequest represents the request to delete a capacity provider
type DeleteCapacityProviderRequest struct {
	CapacityProvider string `json:"capacityProvider"`
}

// DeleteCapacityProviderResponse represents the response from deleting a capacity provider
type DeleteCapacityProviderResponse struct {
	CapacityProvider CapacityProvider `json:"capacityProvider"`
}

// DescribeCapacityProvidersRequest represents the request to describe capacity providers
type DescribeCapacityProvidersRequest struct {
	CapacityProviders []string `json:"capacityProviders,omitempty"`
	Include           []string `json:"include,omitempty"`
}

// DescribeCapacityProvidersResponse represents the response from describing capacity providers
type DescribeCapacityProvidersResponse struct {
	CapacityProviders []CapacityProvider `json:"capacityProviders"`
	Failures          []Failure          `json:"failures,omitempty"`
}

// PutClusterCapacityProvidersRequest represents the request to put cluster capacity providers
type PutClusterCapacityProvidersRequest struct {
	Cluster                       string             `json:"cluster"`
	CapacityProviders             []string           `json:"capacityProviders"`
	DefaultCapacityProviderStrategy []CapacityStrategy `json:"defaultCapacityProviderStrategy"`
}

// PutClusterCapacityProvidersResponse represents the response from putting cluster capacity providers
type PutClusterCapacityProvidersResponse struct {
	Cluster Cluster `json:"cluster"`
}

// registerCapacityProviderEndpoints registers all capacity provider-related API endpoints
func (s *Server) registerCapacityProviderEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/createcapacityprovider", s.handleCreateCapacityProvider)
	mux.HandleFunc("/v1/updatecapacityprovider", s.handleUpdateCapacityProvider)
	mux.HandleFunc("/v1/deletecapacityprovider", s.handleDeleteCapacityProvider)
	mux.HandleFunc("/v1/describecapacityproviders", s.handleDescribeCapacityProviders)
	mux.HandleFunc("/v1/putclustercapacityproviders", s.handlePutClusterCapacityProviders)
}

// handleCreateCapacityProvider handles the CreateCapacityProvider API endpoint
func (s *Server) handleCreateCapacityProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateCapacityProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider creation logic

	// For now, return a mock response
	resp := CreateCapacityProviderResponse{
		CapacityProvider: CapacityProvider{
			CapacityProviderArn:      "arn:aws:ecs:region:account:capacity-provider/" + req.Name,
			Name:                     req.Name,
			Status:                   "ACTIVE",
			AutoScalingGroupProvider: req.AutoScalingGroupProvider,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateCapacityProvider handles the UpdateCapacityProvider API endpoint
func (s *Server) handleUpdateCapacityProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateCapacityProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider update logic

	// For now, return a mock response
	resp := UpdateCapacityProviderResponse{
		CapacityProvider: CapacityProvider{
			CapacityProviderArn:      "arn:aws:ecs:region:account:capacity-provider/" + req.Name,
			Name:                     req.Name,
			Status:                   "ACTIVE",
			AutoScalingGroupProvider: req.AutoScalingGroupProvider,
			UpdateStatus:             "UPDATE_COMPLETE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteCapacityProvider handles the DeleteCapacityProvider API endpoint
func (s *Server) handleDeleteCapacityProvider(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteCapacityProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider deletion logic

	// For now, return a mock response
	resp := DeleteCapacityProviderResponse{
		CapacityProvider: CapacityProvider{
			CapacityProviderArn: "arn:aws:ecs:region:account:capacity-provider/" + req.CapacityProvider,
			Name:               req.CapacityProvider,
			Status:             "INACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeCapacityProviders handles the DescribeCapacityProviders API endpoint
func (s *Server) handleDescribeCapacityProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeCapacityProvidersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider description logic

	// For now, return a mock response
	capacityProviders := []CapacityProvider{}
	
	if len(req.CapacityProviders) > 0 {
		for _, name := range req.CapacityProviders {
			capacityProviders = append(capacityProviders, CapacityProvider{
				CapacityProviderArn: "arn:aws:ecs:region:account:capacity-provider/" + name,
				Name:               name,
				Status:             "ACTIVE",
			})
		}
	} else {
		// Return default capacity providers if none specified
		capacityProviders = append(capacityProviders, CapacityProvider{
			CapacityProviderArn: "arn:aws:ecs:region:account:capacity-provider/FARGATE",
			Name:               "FARGATE",
			Status:             "ACTIVE",
		})
		capacityProviders = append(capacityProviders, CapacityProvider{
			CapacityProviderArn: "arn:aws:ecs:region:account:capacity-provider/FARGATE_SPOT",
			Name:               "FARGATE_SPOT",
			Status:             "ACTIVE",
		})
	}

	resp := DescribeCapacityProvidersResponse{
		CapacityProviders: capacityProviders,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handlePutClusterCapacityProviders handles the PutClusterCapacityProviders API endpoint
func (s *Server) handlePutClusterCapacityProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PutClusterCapacityProvidersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual cluster capacity provider update logic

	// For now, return a mock response
	resp := PutClusterCapacityProvidersResponse{
		Cluster: Cluster{
			ClusterArn:                    "arn:aws:ecs:region:account:cluster/" + req.Cluster,
			ClusterName:                   req.Cluster,
			Status:                        "ACTIVE",
			CapacityProviders:             req.CapacityProviders,
			DefaultCapacityProviderStrategy: req.DefaultCapacityProviderStrategy,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
