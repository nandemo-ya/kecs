package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nandemo-ya/kecs/internal/storage"
)

// HTTP Handlers for ECS Service operations

// handleECSCreateService handles the CreateService operation
func (s *Server) handleECSCreateService(w http.ResponseWriter, body []byte) {
	var req CreateServiceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.CreateServiceWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error creating service: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDescribeServices handles the DescribeServices operation
func (s *Server) handleECSDescribeServices(w http.ResponseWriter, body []byte) {
	var req DescribeServicesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.DescribeServicesWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error describing services: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSListServices handles the ListServices operation
func (s *Server) handleECSListServices(w http.ResponseWriter, body []byte) {
	var req ListServicesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.ListServicesWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error listing services: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSUpdateService handles the UpdateService operation
func (s *Server) handleECSUpdateService(w http.ResponseWriter, body []byte) {
	var req UpdateServiceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.UpdateServiceWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error updating service: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDeleteService handles the DeleteService operation
func (s *Server) handleECSDeleteService(w http.ResponseWriter, body []byte) {
	var req DeleteServiceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.DeleteServiceWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error deleting service: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// CreateServiceWithStorage creates a new service using storage
func (s *Server) CreateServiceWithStorage(ctx context.Context, req CreateServiceRequest) (*CreateServiceResponse, error) {
	// Default cluster if not specified
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	// Generate ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:service/%s/%s", cluster, req.ServiceName)
	clusterARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", cluster)

	// Set default values
	if req.LaunchType == "" {
		req.LaunchType = "FARGATE"
	}
	if req.SchedulingStrategy == "" {
		req.SchedulingStrategy = "REPLICA"
	}

	// Convert complex objects to JSON
	loadBalancersJSON, _ := json.Marshal(req.LoadBalancers)
	serviceRegistriesJSON, _ := json.Marshal(req.ServiceRegistries)
	networkConfigJSON, _ := json.Marshal(req.NetworkConfiguration)
	deploymentConfigJSON, _ := json.Marshal(req.DeploymentConfiguration)
	placementConstraintsJSON, _ := json.Marshal(req.PlacementConstraints)
	placementStrategyJSON, _ := json.Marshal(req.PlacementStrategy)
	capacityProviderStrategyJSON, _ := json.Marshal(req.CapacityProviderStrategy)
	tagsJSON, _ := json.Marshal(req.Tags)
	serviceConnectConfigJSON, _ := json.Marshal(req.ServiceConnectConfiguration)

	// Create storage service
	storageService := &storage.Service{
		ARN:                           serviceARN,
		ServiceName:                   req.ServiceName,
		ClusterARN:                    clusterARN,
		TaskDefinitionARN:             req.TaskDefinition,
		DesiredCount:                  req.DesiredCount,
		RunningCount:                  0,
		PendingCount:                  req.DesiredCount,
		LaunchType:                    req.LaunchType,
		PlatformVersion:               req.PlatformVersion,
		Status:                        "ACTIVE",
		RoleARN:                       req.Role,
		LoadBalancers:                 string(loadBalancersJSON),
		ServiceRegistries:             string(serviceRegistriesJSON),
		NetworkConfiguration:          string(networkConfigJSON),
		DeploymentConfiguration:       string(deploymentConfigJSON),
		PlacementConstraints:          string(placementConstraintsJSON),
		PlacementStrategy:             string(placementStrategyJSON),
		CapacityProviderStrategy:      string(capacityProviderStrategyJSON),
		Tags:                          string(tagsJSON),
		SchedulingStrategy:            req.SchedulingStrategy,
		ServiceConnectConfiguration:  string(serviceConnectConfigJSON),
		EnableECSManagedTags:          req.EnableECSManagedTags,
		PropagateTags:                 req.PropagateTags,
		EnableExecuteCommand:          req.EnableExecuteCommand,
		HealthCheckGracePeriodSeconds: req.HealthCheckGracePeriodSeconds,
		Region:                        "us-east-1",
		AccountID:                     "123456789012",
	}

	// Save to storage
	if err := s.storage.ServiceStore().Create(ctx, storageService); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Convert back to API response
	service := storageServiceToAPIService(storageService)

	return &CreateServiceResponse{
		Service: service,
	}, nil
}

// UpdateServiceWithStorage updates an existing service using storage
func (s *Server) UpdateServiceWithStorage(ctx context.Context, req UpdateServiceRequest) (*UpdateServiceResponse, error) {
	// Default cluster if not specified
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", cluster)

	// Get existing service
	existingService, err := s.storage.ServiceStore().Get(ctx, clusterARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Update fields
	if req.DesiredCount > 0 {
		existingService.DesiredCount = req.DesiredCount
	}
	if req.TaskDefinition != "" {
		existingService.TaskDefinitionARN = req.TaskDefinition
	}
	if req.PlatformVersion != "" {
		existingService.PlatformVersion = req.PlatformVersion
	}

	// Update complex objects if provided
	if req.NetworkConfiguration != nil {
		networkConfigJSON, _ := json.Marshal(req.NetworkConfiguration)
		existingService.NetworkConfiguration = string(networkConfigJSON)
	}
	if req.DeploymentConfiguration != nil {
		deploymentConfigJSON, _ := json.Marshal(req.DeploymentConfiguration)
		existingService.DeploymentConfiguration = string(deploymentConfigJSON)
	}
	if req.PlacementConstraints != nil {
		placementConstraintsJSON, _ := json.Marshal(req.PlacementConstraints)
		existingService.PlacementConstraints = string(placementConstraintsJSON)
	}
	if req.PlacementStrategy != nil {
		placementStrategyJSON, _ := json.Marshal(req.PlacementStrategy)
		existingService.PlacementStrategy = string(placementStrategyJSON)
	}
	if req.CapacityProviderStrategy != nil {
		capacityProviderStrategyJSON, _ := json.Marshal(req.CapacityProviderStrategy)
		existingService.CapacityProviderStrategy = string(capacityProviderStrategyJSON)
	}
	if req.LoadBalancers != nil {
		loadBalancersJSON, _ := json.Marshal(req.LoadBalancers)
		existingService.LoadBalancers = string(loadBalancersJSON)
	}
	if req.ServiceRegistries != nil {
		serviceRegistriesJSON, _ := json.Marshal(req.ServiceRegistries)
		existingService.ServiceRegistries = string(serviceRegistriesJSON)
	}
	if req.ServiceConnectConfiguration != nil {
		serviceConnectConfigJSON, _ := json.Marshal(req.ServiceConnectConfiguration)
		existingService.ServiceConnectConfiguration = string(serviceConnectConfigJSON)
	}

	existingService.EnableECSManagedTags = req.EnableECSManagedTags
	existingService.EnableExecuteCommand = req.EnableExecuteCommand
	if req.HealthCheckGracePeriodSeconds > 0 {
		existingService.HealthCheckGracePeriodSeconds = req.HealthCheckGracePeriodSeconds
	}

	// Update in storage
	if err := s.storage.ServiceStore().Update(ctx, existingService); err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	// Convert back to API response
	service := storageServiceToAPIService(existingService)

	return &UpdateServiceResponse{
		Service: service,
	}, nil
}

// DeleteServiceWithStorage deletes a service using storage
func (s *Server) DeleteServiceWithStorage(ctx context.Context, req DeleteServiceRequest) (*DeleteServiceResponse, error) {
	// Default cluster if not specified
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", cluster)

	// Get existing service to return in response
	existingService, err := s.storage.ServiceStore().Get(ctx, clusterARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// For non-force deletes, check if desired count is 0
	if !req.Force && existingService.DesiredCount > 0 {
		return nil, fmt.Errorf("service must have desired count of 0 for delete, or use force=true")
	}

	// Update status to DRAINING before deletion
	existingService.Status = "DRAINING"
	existingService.DesiredCount = 0
	if err := s.storage.ServiceStore().Update(ctx, existingService); err != nil {
		log.Printf("Warning: failed to update service status to DRAINING: %v", err)
	}

	// Delete from storage
	if err := s.storage.ServiceStore().Delete(ctx, clusterARN, req.Service); err != nil {
		return nil, fmt.Errorf("failed to delete service: %w", err)
	}

	// Convert back to API response
	service := storageServiceToAPIService(existingService)

	return &DeleteServiceResponse{
		Service: service,
	}, nil
}

// DescribeServicesWithStorage describes services using storage
func (s *Server) DescribeServicesWithStorage(ctx context.Context, req DescribeServicesRequest) (*DescribeServicesResponse, error) {
	// Default cluster if not specified
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", cluster)

	var services []Service
	var failures []Failure

	for _, serviceName := range req.Services {
		storageService, err := s.storage.ServiceStore().Get(ctx, clusterARN, serviceName)
		if err != nil {
			failures = append(failures, Failure{
				Arn:    fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:service/%s/%s", cluster, serviceName),
				Reason: "MISSING",
				Detail: err.Error(),
			})
			continue
		}

		service := storageServiceToAPIService(storageService)
		services = append(services, service)
	}

	return &DescribeServicesResponse{
		Services: services,
		Failures: failures,
	}, nil
}

// ListServicesWithStorage lists services using storage
func (s *Server) ListServicesWithStorage(ctx context.Context, req ListServicesRequest) (*ListServicesResponse, error) {
	// Default cluster if not specified
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", cluster)

	// Set default limit if not specified
	limit := req.MaxResults
	if limit <= 0 {
		limit = 100
	}

	// Get services from storage
	storageServices, nextToken, err := s.storage.ServiceStore().List(ctx, clusterARN, "", req.LaunchType, limit, req.NextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Extract ARNs
	var serviceARNs []string
	for _, service := range storageServices {
		serviceARNs = append(serviceARNs, service.ARN)
	}

	return &ListServicesResponse{
		ServiceArns: serviceARNs,
		NextToken:   nextToken,
	}, nil
}

// storageServiceToAPIService converts a storage.Service to an API Service
func storageServiceToAPIService(storageService *storage.Service) Service {
	service := Service{
		ServiceArn:                    storageService.ARN,
		ServiceName:                   storageService.ServiceName,
		ClusterArn:                    storageService.ClusterARN,
		Status:                        storageService.Status,
		DesiredCount:                  storageService.DesiredCount,
		RunningCount:                  storageService.RunningCount,
		PendingCount:                  storageService.PendingCount,
		LaunchType:                    storageService.LaunchType,
		PlatformVersion:               storageService.PlatformVersion,
		TaskDefinition:                storageService.TaskDefinitionARN,
		RoleArn:                       storageService.RoleARN,
		SchedulingStrategy:            storageService.SchedulingStrategy,
		EnableECSManagedTags:          storageService.EnableECSManagedTags,
		PropagateTags:                 storageService.PropagateTags,
		EnableExecuteCommand:          storageService.EnableExecuteCommand,
		HealthCheckGracePeriodSeconds: storageService.HealthCheckGracePeriodSeconds,
		CreatedAt:                     storageService.CreatedAt.Format(time.RFC3339),
	}

	// Parse JSON fields
	if storageService.LoadBalancers != "" && storageService.LoadBalancers != "null" {
		json.Unmarshal([]byte(storageService.LoadBalancers), &service.LoadBalancers)
	}
	if storageService.ServiceRegistries != "" && storageService.ServiceRegistries != "null" {
		json.Unmarshal([]byte(storageService.ServiceRegistries), &service.ServiceRegistries)
	}
	if storageService.NetworkConfiguration != "" && storageService.NetworkConfiguration != "null" {
		json.Unmarshal([]byte(storageService.NetworkConfiguration), &service.NetworkConfiguration)
	}
	if storageService.DeploymentConfiguration != "" && storageService.DeploymentConfiguration != "null" {
		json.Unmarshal([]byte(storageService.DeploymentConfiguration), &service.DeploymentConfiguration)
	}
	if storageService.PlacementConstraints != "" && storageService.PlacementConstraints != "null" {
		json.Unmarshal([]byte(storageService.PlacementConstraints), &service.PlacementConstraints)
	}
	if storageService.PlacementStrategy != "" && storageService.PlacementStrategy != "null" {
		json.Unmarshal([]byte(storageService.PlacementStrategy), &service.PlacementStrategy)
	}
	if storageService.CapacityProviderStrategy != "" && storageService.CapacityProviderStrategy != "null" {
		json.Unmarshal([]byte(storageService.CapacityProviderStrategy), &service.CapacityProviderStrategy)
	}
	if storageService.Tags != "" && storageService.Tags != "null" {
		json.Unmarshal([]byte(storageService.Tags), &service.Tags)
	}

	return service
}