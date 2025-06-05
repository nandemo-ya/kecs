package api

import (
	"encoding/json"
	"net/http"
)

// handleECSCreateCapacityProvider handles the CreateCapacityProvider API endpoint in AWS ECS format
func (s *Server) handleECSCreateCapacityProvider(w http.ResponseWriter, body []byte) {
	var req CreateCapacityProviderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider creation logic
	// For now, return a mock response
	resp := CreateCapacityProviderResponse{
		CapacityProvider: CapacityProvider{
			CapacityProviderArn:      "arn:aws:ecs:" + s.region + ":" + s.accountID + ":capacity-provider/" + req.Name,
			Name:                     req.Name,
			Status:                   "ACTIVE",
			AutoScalingGroupProvider: req.AutoScalingGroupProvider,
			Tags:                     req.Tags,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSUpdateCapacityProvider handles the UpdateCapacityProvider API endpoint in AWS ECS format
func (s *Server) handleECSUpdateCapacityProvider(w http.ResponseWriter, body []byte) {
	var req UpdateCapacityProviderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider update logic
	// For now, return a mock response
	resp := UpdateCapacityProviderResponse{
		CapacityProvider: CapacityProvider{
			CapacityProviderArn:      "arn:aws:ecs:" + s.region + ":" + s.accountID + ":capacity-provider/" + req.Name,
			Name:                     req.Name,
			Status:                   "ACTIVE",
			AutoScalingGroupProvider: req.AutoScalingGroupProvider,
			UpdateStatus:             "UPDATE_COMPLETE",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDeleteCapacityProvider handles the DeleteCapacityProvider API endpoint in AWS ECS format
func (s *Server) handleECSDeleteCapacityProvider(w http.ResponseWriter, body []byte) {
	var req DeleteCapacityProviderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider deletion logic
	// For now, return a mock response
	resp := DeleteCapacityProviderResponse{
		CapacityProvider: CapacityProvider{
			CapacityProviderArn: "arn:aws:ecs:" + s.region + ":" + s.accountID + ":capacity-provider/" + req.CapacityProvider,
			Name:               req.CapacityProvider,
			Status:             "INACTIVE",
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDescribeCapacityProviders handles the DescribeCapacityProviders API endpoint in AWS ECS format
func (s *Server) handleECSDescribeCapacityProviders(w http.ResponseWriter, body []byte) {
	var req DescribeCapacityProvidersRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual capacity provider description logic
	// For now, return a mock response
	capacityProviders := []CapacityProvider{}
	
	if len(req.CapacityProviders) > 0 {
		for _, name := range req.CapacityProviders {
			capacityProviders = append(capacityProviders, CapacityProvider{
				CapacityProviderArn: "arn:aws:ecs:" + s.region + ":" + s.accountID + ":capacity-provider/" + name,
				Name:               name,
				Status:             "ACTIVE",
			})
		}
	} else {
		// Return default capacity providers if none specified
		capacityProviders = append(capacityProviders, CapacityProvider{
			CapacityProviderArn: "arn:aws:ecs:" + s.region + ":" + s.accountID + ":capacity-provider/FARGATE",
			Name:               "FARGATE",
			Status:             "ACTIVE",
		})
		capacityProviders = append(capacityProviders, CapacityProvider{
			CapacityProviderArn: "arn:aws:ecs:" + s.region + ":" + s.accountID + ":capacity-provider/FARGATE_SPOT",
			Name:               "FARGATE_SPOT",
			Status:             "ACTIVE",
		})
	}

	resp := DescribeCapacityProvidersResponse{
		CapacityProviders: capacityProviders,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}