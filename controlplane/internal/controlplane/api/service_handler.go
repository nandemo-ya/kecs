package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// HTTP Handlers for ECS Service operations

// handleECSCreateService handles the CreateService operation
func (s *Server) handleECSCreateService(w http.ResponseWriter, body []byte) {
	var req CreateServiceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	ctx := context.Background()
	resp, err := s.CreateServiceWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error creating service: %v", err)
		errorResponse := map[string]interface{}{
			"__type": "ServiceException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
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
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	ctx := context.Background()
	resp, err := s.DescribeServicesWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error describing services: %v", err)
		errorResponse := map[string]interface{}{
			"__type": "ServiceException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
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
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	ctx := context.Background()
	resp, err := s.ListServicesWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error listing services: %v", err)
		errorResponse := map[string]interface{}{
			"__type": "ServiceException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
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
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	ctx := context.Background()
	resp, err := s.UpdateServiceWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error updating service: %v", err)
		errorResponse := map[string]interface{}{
			"__type": "ServiceException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
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
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request body",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	ctx := context.Background()
	resp, err := s.DeleteServiceWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error deleting service: %v", err)
		errorResponse := map[string]interface{}{
			"__type": "ServiceException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// CreateServiceWithStorage creates a new service using storage
func (s *Server) CreateServiceWithStorage(ctx context.Context, req CreateServiceRequest) (*CreateServiceResponse, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	// Get cluster from storage
	cluster, err := s.storage.ClusterStore().Get(context.Background(), clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get task definition
	var taskDef *storage.TaskDefinition
	taskDefArn := req.TaskDefinition
	
	log.Printf("DEBUG: Looking for task definition: %s", req.TaskDefinition)
	
	if !strings.HasPrefix(taskDefArn, "arn:aws:ecs:") {
		// Check if it's family:revision or just family
		if strings.Contains(req.TaskDefinition, ":") {
			// family:revision format
			taskDefArn = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s", s.region, s.accountID, req.TaskDefinition)
			log.Printf("DEBUG: Trying to get task definition by ARN: %s", taskDefArn)
			taskDef, err = s.storage.TaskDefinitionStore().GetByARN(context.Background(), taskDefArn)
		} else {
			// Just family - get latest
			log.Printf("DEBUG: Trying to get latest task definition for family: %s", req.TaskDefinition)
			taskDef, err = s.storage.TaskDefinitionStore().GetLatest(context.Background(), req.TaskDefinition)
			if taskDef != nil {
				taskDefArn = taskDef.ARN
				log.Printf("DEBUG: Found latest task definition: %s", taskDefArn)
			}
		}
	} else {
		// Full ARN provided
		log.Printf("DEBUG: Full ARN provided: %s", taskDefArn)
		taskDef, err = s.storage.TaskDefinitionStore().GetByARN(context.Background(), taskDefArn)
	}
	
	if err != nil {
		log.Printf("DEBUG: Error getting task definition: %v", err)
		return nil, fmt.Errorf("task definition not found: %s", req.TaskDefinition)
	}
	
	if taskDef == nil {
		log.Printf("DEBUG: Task definition is nil")
		return nil, fmt.Errorf("task definition not found: %s", req.TaskDefinition)
	}

	// Generate ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster.Name, req.ServiceName)
	clusterARN := cluster.ARN

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

	// Create service converter and manager
	serviceConverter := converters.NewServiceConverter(s.region, s.accountID)
	serviceManager := kubernetes.NewServiceManager(s.storage, s.kindManager)

	// Convert ECS service to Kubernetes Deployment
	deployment, kubeService, err := serviceConverter.ConvertServiceToDeployment(
		&storage.Service{ServiceName: req.ServiceName, DesiredCount: req.DesiredCount, LaunchType: req.LaunchType},
		taskDef,
		cluster,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to convert service to deployment: %w", err)
	}

	// Create storage service with deployment information
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	deploymentName := fmt.Sprintf("ecs-service-%s", req.ServiceName)
	
	storageService := &storage.Service{
		ARN:                           serviceARN,
		ServiceName:                   req.ServiceName,
		ClusterARN:                    clusterARN,
		TaskDefinitionARN:             taskDefArn,
		DesiredCount:                  req.DesiredCount,
		RunningCount:                  0,
		PendingCount:                  req.DesiredCount,
		LaunchType:                    req.LaunchType,
		PlatformVersion:               req.PlatformVersion,
		Status:                        "PENDING",
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
		Region:                        s.region,
		AccountID:                     s.accountID,
		DeploymentName:                deploymentName,
		Namespace:                     namespace,
	}

	// Save to storage first
	if err := s.storage.ServiceStore().Create(ctx, storageService); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Create Kubernetes Deployment and Service
	if err := serviceManager.CreateService(ctx, deployment, kubeService, cluster, storageService); err != nil {
		// Service was created in storage but Kubernetes deployment failed
		// Update status to indicate failure - get fresh service data first
		if freshService, getErr := s.storage.ServiceStore().Get(ctx, cluster.ARN, storageService.ServiceName); getErr == nil {
			freshService.Status = "FAILED"
			s.storage.ServiceStore().Update(ctx, freshService)
		}
		return nil, fmt.Errorf("failed to create kubernetes deployment: %w", err)
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
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	// Get cluster from storage
	cluster, err := s.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get existing service
	existingService, err := s.storage.ServiceStore().Get(ctx, cluster.ARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Track if we need to update Kubernetes resources
	needsKubernetesUpdate := false
	oldDesiredCount := existingService.DesiredCount
	oldTaskDefinitionARN := existingService.TaskDefinitionARN

	// Update fields
	// Note: DesiredCount can be 0 (to scale down to 0 tasks)
	if req.DesiredCount >= 0 && req.DesiredCount != existingService.DesiredCount {
		log.Printf("DEBUG: Updating desired count from %d to %d", existingService.DesiredCount, req.DesiredCount)
		existingService.DesiredCount = req.DesiredCount
		needsKubernetesUpdate = true
	}
	if req.TaskDefinition != "" && req.TaskDefinition != existingService.TaskDefinitionARN {
		// Convert to ARN if necessary
		var newTaskDefArn string
		if !strings.HasPrefix(req.TaskDefinition, "arn:aws:ecs:") {
			// Check if it's family:revision or just family
			if strings.Contains(req.TaskDefinition, ":") {
				// family:revision format
				newTaskDefArn = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s", s.region, s.accountID, req.TaskDefinition)
			} else {
				// Just family - get latest
				latestTaskDef, err := s.storage.TaskDefinitionStore().GetLatest(context.Background(), req.TaskDefinition)
				if err != nil || latestTaskDef == nil {
					return nil, fmt.Errorf("task definition not found: %s", req.TaskDefinition)
				}
				newTaskDefArn = latestTaskDef.ARN
			}
		} else {
			newTaskDefArn = req.TaskDefinition
		}
		
		existingService.TaskDefinitionARN = newTaskDefArn
		needsKubernetesUpdate = true
	}
	if req.PlatformVersion != "" {
		existingService.PlatformVersion = req.PlatformVersion
	}

	// Update complex objects if provided
	if req.NetworkConfiguration != nil {
		networkConfigJSON, _ := json.Marshal(req.NetworkConfiguration)
		existingService.NetworkConfiguration = string(networkConfigJSON)
		needsKubernetesUpdate = true
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
		needsKubernetesUpdate = true
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

	// Update Kubernetes resources if needed
	if needsKubernetesUpdate {
		// Update status to show update in progress
		existingService.Status = "PENDING"
		// Get the task definition if it was updated
		taskDef := existingService.TaskDefinitionARN
		taskDefinition, err := s.storage.TaskDefinitionStore().GetByARN(ctx, taskDef)
		if err != nil {
			log.Printf("Failed to get task definition %s: %v", taskDef, err)
			// Restore old values on failure
			existingService.DesiredCount = oldDesiredCount
			existingService.TaskDefinitionARN = oldTaskDefinitionARN
			return nil, fmt.Errorf("failed to get task definition: %w", err)
		}

		// Create service converter and manager
		converter := converters.NewServiceConverter(s.region, s.accountID)
		deployment, kubeService, err := converter.ConvertServiceToDeployment(existingService, taskDefinition, cluster)
		if err != nil {
			log.Printf("Failed to convert service to deployment: %v", err)
			return nil, fmt.Errorf("failed to convert service: %w", err)
		}

		// Create service manager and update Kubernetes resources
		serviceManager := kubernetes.NewServiceManager(s.storage, s.kindManager)
		if err := serviceManager.UpdateService(ctx, deployment, kubeService, cluster, existingService); err != nil {
			log.Printf("Failed to update kubernetes deployment: %v", err)
			return nil, fmt.Errorf("failed to update kubernetes deployment: %w", err)
		}

		// Update status to ACTIVE after successful update
		existingService.Status = "ACTIVE"
	}
	
	// Single update at the end
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
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	// Get cluster from storage
	cluster, err := s.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get existing service to return in response
	existingService, err := s.storage.ServiceStore().Get(ctx, cluster.ARN, req.Service)
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

	// Delete Kubernetes resources
	serviceManager := kubernetes.NewServiceManager(s.storage, s.kindManager)
	if err := serviceManager.DeleteService(ctx, cluster, existingService); err != nil {
		log.Printf("Warning: failed to delete Kubernetes resources for service %s: %v", 
			existingService.ServiceName, err)
		// Continue with deletion even if Kubernetes deletion fails
		// This matches AWS ECS behavior where the service is marked for deletion
		// even if underlying resources might still exist
	}

	// Delete from storage
	if err := s.storage.ServiceStore().Delete(ctx, cluster.ARN, req.Service); err != nil {
		return nil, fmt.Errorf("failed to delete service: %w", err)
	}

	log.Printf("Successfully deleted service %s from cluster %s", 
		existingService.ServiceName, clusterName)

	// Convert back to API response
	// The service is returned with DRAINING status as per AWS ECS behavior
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

	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster)

	var services []Service
	var failures []Failure

	for _, serviceName := range req.Services {
		storageService, err := s.storage.ServiceStore().Get(ctx, clusterARN, serviceName)
		if err != nil {
			failures = append(failures, Failure{
				Arn:    fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster, serviceName),
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

	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster)

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

	// Add deployment information
	// In AWS ECS, there's always at least one deployment representing the current state
	deployment := Deployment{
		Id:                       fmt.Sprintf("ecs-svc/%s", storageService.ServiceName),
		Status:                   "PRIMARY",
		TaskDefinition:           storageService.TaskDefinitionARN,
		DesiredCount:             storageService.DesiredCount,
		RunningCount:             storageService.RunningCount,
		PendingCount:             storageService.PendingCount,
		LaunchType:               storageService.LaunchType,
		PlatformVersion:          storageService.PlatformVersion,
		CreatedAt:                storageService.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                storageService.UpdatedAt.Format(time.RFC3339),
	}
	
	// Copy deployment configuration if it exists
	if service.DeploymentConfiguration != nil {
		// The deployment inherits the service's deployment configuration
	}
	
	service.Deployments = []Deployment{deployment}

	return service
}