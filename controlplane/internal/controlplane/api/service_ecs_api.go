package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// getServiceManager returns a ServiceManager using the appropriate cluster manager
func (api *DefaultECSAPI) getServiceManager() (*kubernetes.ServiceManager, error) {
	// In test mode, we can return a ServiceManager with nil cluster manager
	// as the ServiceManager handles test mode internally
	if config.GetBool("features.testMode") {
		return kubernetes.NewServiceManager(api.storage, nil), nil
	}

	// Get cluster manager
	if clusterManager := api.getClusterManager(); clusterManager != nil {
		return kubernetes.NewServiceManager(api.storage, clusterManager), nil
	} else {
		return nil, fmt.Errorf("no cluster manager available")
	}
}

// CreateService implements the CreateService operation
func (api *DefaultECSAPI) CreateService(ctx context.Context, req *generated.CreateServiceRequest) (*generated.CreateServiceResponse, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = *req.Cluster
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Validate required fields
	if req.ServiceName == "" {
		return nil, fmt.Errorf("serviceName is required")
	}
	if req.TaskDefinition == nil {
		return nil, fmt.Errorf("taskDefinition is required")
	}

	// Get task definition
	var taskDef *storage.TaskDefinition
	taskDefArn := *req.TaskDefinition

	log.Printf("DEBUG: Looking for task definition: %s", taskDefArn)

	if !strings.HasPrefix(taskDefArn, "arn:aws:ecs:") {
		// Check if it's family:revision or just family
		if strings.Contains(taskDefArn, ":") {
			// family:revision format
			taskDefArn = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s", api.region, api.accountID, taskDefArn)
			log.Printf("DEBUG: Trying to get task definition by ARN: %s", taskDefArn)
			taskDef, err = api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefArn)
		} else {
			// Just family - get latest
			log.Printf("DEBUG: Trying to get latest task definition for family: %s", taskDefArn)
			taskDef, err = api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefArn)
			if taskDef != nil {
				taskDefArn = taskDef.ARN
				log.Printf("DEBUG: Found latest task definition: %s", taskDefArn)
			}
		}
	} else {
		// Full ARN provided
		log.Printf("DEBUG: Full ARN provided: %s", taskDefArn)
		taskDef, err = api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefArn)
	}

	if err != nil {
		log.Printf("DEBUG: Error getting task definition: %v", err)
		return nil, fmt.Errorf("task definition not found: %s", *req.TaskDefinition)
	}

	if taskDef == nil {
		log.Printf("DEBUG: Task definition is nil")
		return nil, fmt.Errorf("task definition not found: %s", *req.TaskDefinition)
	}

	// Generate ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster.Name, req.ServiceName)
	clusterARN := cluster.ARN

	// Set default values
	launchType := generated.LaunchTypeFARGATE
	if req.LaunchType != nil {
		launchType = *req.LaunchType
	}

	schedulingStrategy := generated.SchedulingStrategyREPLICA
	if req.SchedulingStrategy != nil {
		schedulingStrategy = *req.SchedulingStrategy
	}

	desiredCount := int32(1)
	if req.DesiredCount != nil {
		desiredCount = *req.DesiredCount
	}

	// Convert complex objects to JSON for storage
	loadBalancersJSON, err := json.Marshal(req.LoadBalancers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal load balancers: %w", err)
	}

	serviceRegistriesJSON, err := json.Marshal(req.ServiceRegistries)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service registries: %w", err)
	}

	networkConfigJSON, err := json.Marshal(req.NetworkConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal network configuration: %w", err)
	}

	deploymentConfigJSON, err := json.Marshal(req.DeploymentConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment configuration: %w", err)
	}

	placementConstraintsJSON, err := json.Marshal(req.PlacementConstraints)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal placement constraints: %w", err)
	}

	placementStrategyJSON, err := json.Marshal(req.PlacementStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal placement strategy: %w", err)
	}

	capacityProviderStrategyJSON, err := json.Marshal(req.CapacityProviderStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capacity provider strategy: %w", err)
	}

	tagsJSON, err := json.Marshal(req.Tags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %w", err)
	}

	serviceConnectConfigJSON, err := json.Marshal(req.ServiceConnectConfiguration)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal service connect configuration: %w", err)
	}

	// Check if k3d cluster exists (but don't wait synchronously to avoid API timeout)
	if !config.GetBool("features.testMode") {
		clusterManager := api.getClusterManager()
		if clusterManager != nil {
			exists, err := clusterManager.ClusterExists(ctx, cluster.K8sClusterName)
			if err != nil {
				return nil, fmt.Errorf("failed to check if k3d cluster exists: %w", err)
			}
			if !exists {
				return nil, fmt.Errorf("k3d cluster %s does not exist yet, please wait for cluster creation to complete", cluster.K8sClusterName)
			}
			log.Printf("k3d cluster %s exists, proceeding with service creation", cluster.K8sClusterName)
		}
	}

	// Create service converter and manager
	serviceConverter := converters.NewServiceConverter(api.region, api.accountID)
	serviceManager, err := api.getServiceManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create service manager: %w", err)
	}

	// Convert ECS service to Kubernetes Deployment
	storageServiceTemp := &storage.Service{
		ServiceName:  req.ServiceName,
		DesiredCount: int(desiredCount),
		LaunchType:   string(launchType),
	}
	deployment, kubeService, err := serviceConverter.ConvertServiceToDeploymentWithNetworkConfig(
		storageServiceTemp,
		taskDef,
		cluster,
		req.NetworkConfiguration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to convert service to deployment: %w", err)
	}

	// Create storage service with deployment information
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	deploymentName := fmt.Sprintf("ecs-service-%s", req.ServiceName)

	// Extract optional string values
	var platformVersion, roleARN, propagateTags string
	if req.PlatformVersion != nil {
		platformVersion = *req.PlatformVersion
	}
	if req.Role != nil {
		roleARN = *req.Role
	}
	if req.PropagateTags != nil {
		propagateTags = string(*req.PropagateTags)
	}

	var healthCheckGracePeriod int
	if req.HealthCheckGracePeriodSeconds != nil {
		healthCheckGracePeriod = int(*req.HealthCheckGracePeriodSeconds)
	}

	var enableECSManagedTags, enableExecuteCommand bool
	if req.EnableECSManagedTags != nil {
		enableECSManagedTags = *req.EnableECSManagedTags
	}
	if req.EnableExecuteCommand != nil {
		enableExecuteCommand = *req.EnableExecuteCommand
	}

	storageService := &storage.Service{
		ARN:                           serviceARN,
		ServiceName:                   req.ServiceName,
		ClusterARN:                    clusterARN,
		TaskDefinitionARN:             taskDefArn,
		DesiredCount:                  int(desiredCount),
		RunningCount:                  0,
		PendingCount:                  int(desiredCount),
		LaunchType:                    string(launchType),
		PlatformVersion:               platformVersion,
		Status:                        "PENDING",
		RoleARN:                       roleARN,
		LoadBalancers:                 string(loadBalancersJSON),
		ServiceRegistries:             string(serviceRegistriesJSON),
		NetworkConfiguration:          string(networkConfigJSON),
		DeploymentConfiguration:       string(deploymentConfigJSON),
		PlacementConstraints:          string(placementConstraintsJSON),
		PlacementStrategy:             string(placementStrategyJSON),
		CapacityProviderStrategy:      string(capacityProviderStrategyJSON),
		Tags:                          string(tagsJSON),
		SchedulingStrategy:            string(schedulingStrategy),
		ServiceConnectConfiguration:   string(serviceConnectConfigJSON),
		EnableECSManagedTags:          enableECSManagedTags,
		PropagateTags:                 propagateTags,
		EnableExecuteCommand:          enableExecuteCommand,
		HealthCheckGracePeriodSeconds: healthCheckGracePeriod,
		Region:                        api.region,
		AccountID:                     api.accountID,
		DeploymentName:                deploymentName,
		Namespace:                     namespace,
		CreatedAt:                     time.Now(),
		UpdatedAt:                     time.Now(),
	}

	// Save to storage first
	if err := api.storage.ServiceStore().Create(ctx, storageService); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Increment cluster's active service count
	cluster.ActiveServicesCount++
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		// Log error but don't fail service creation
		log.Printf("Warning: Failed to update cluster service count: %v", err)
	}

	// Create Kubernetes Deployment and Service
	if err := serviceManager.CreateService(ctx, deployment, kubeService, cluster, storageService); err != nil {
		// Service was created in storage but Kubernetes deployment failed
		// Update status to indicate failure - get fresh service data first
		if freshService, getErr := api.storage.ServiceStore().Get(ctx, cluster.ARN, storageService.ServiceName); getErr == nil {
			freshService.Status = "FAILED"
			api.storage.ServiceStore().Update(ctx, freshService)
		}
		return nil, fmt.Errorf("failed to create kubernetes deployment: %w", err)
	}

	// Handle Service Discovery registration if ServiceRegistries are specified
	if len(req.ServiceRegistries) > 0 {
		if err := api.registerServiceWithDiscovery(ctx, storageService, req.ServiceRegistries); err != nil {
			// Log error but don't fail service creation
			log.Printf("Warning: Failed to register service with service discovery: %v", err)
		}
	}

	// In test mode, create tasks immediately for the service
	if config.GetBool("features.testMode") && storageService.DesiredCount > 0 {
		log.Printf("Test mode: Creating %d tasks for service %s", storageService.DesiredCount, storageService.ServiceName)
		if err := api.createTasksForService(ctx, storageService, taskDef, cluster); err != nil {
			log.Printf("Warning: Failed to create tasks for service in test mode: %v", err)
			// Don't fail service creation, tasks will be created by the worker
		}
	}

	// Convert storage service to API response
	responseService := storageServiceToGeneratedService(storageService)

	return &generated.CreateServiceResponse{
		Service: responseService,
	}, nil
}

// DeleteService implements the DeleteService operation
func (api *DefaultECSAPI) DeleteService(ctx context.Context, req *generated.DeleteServiceRequest) (*generated.DeleteServiceResponse, error) {
	// Validate required fields
	if req.Service == "" {
		return nil, fmt.Errorf("service is required")
	}

	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = *req.Cluster
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get existing service to return in response
	existingService, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Check force flag
	forceDelete := false
	if req.Force != nil {
		forceDelete = *req.Force
	}

	// For non-force deletes, check if desired count is 0
	if !forceDelete && existingService.DesiredCount > 0 {
		return nil, fmt.Errorf("service must have desired count of 0 for delete, or use force=true")
	}

	// Update status to DRAINING before deletion
	existingService.Status = "DRAINING"
	existingService.DesiredCount = 0
	existingService.UpdatedAt = time.Now()
	if err := api.storage.ServiceStore().Update(ctx, existingService); err != nil {
		log.Printf("Warning: failed to update service status to DRAINING: %v", err)
	}

	// Delete Kubernetes resources
	serviceManager, err := api.getServiceManager()
	if err != nil {
		log.Printf("Warning: failed to create service manager for deletion: %v", err)
		// Continue with deletion even if service manager creation fails
	} else if err := serviceManager.DeleteService(ctx, cluster, existingService); err != nil {
		log.Printf("Warning: failed to delete Kubernetes resources for service %s: %v",
			existingService.ServiceName, err)
		// Continue with deletion even if Kubernetes deletion fails
		// This matches AWS ECS behavior where the service is marked for deletion
		// even if underlying resources might still exist
	}

	// Delete from storage
	if err := api.storage.ServiceStore().Delete(ctx, cluster.ARN, req.Service); err != nil {
		return nil, fmt.Errorf("failed to delete service: %w", err)
	}

	// Decrement cluster's active service count
	if cluster.ActiveServicesCount > 0 {
		cluster.ActiveServicesCount--
		if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
			// Log error but don't fail service deletion
			log.Printf("Warning: Failed to update cluster service count: %v", err)
		}
	}

	log.Printf("Successfully deleted service %s from cluster %s",
		existingService.ServiceName, clusterName)

	// Convert back to API response
	// The service is returned with DRAINING status as per AWS ECS behavior
	responseService := storageServiceToGeneratedService(existingService)

	return &generated.DeleteServiceResponse{
		Service: responseService,
	}, nil
}

// DescribeServices implements the DescribeServices operation
func (api *DefaultECSAPI) DescribeServices(ctx context.Context, req *generated.DescribeServicesRequest) (*generated.DescribeServicesResponse, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = *req.Cluster
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, clusterName)

	var services []generated.Service
	var failures []generated.Failure

	// If no services specified, return empty result
	if len(req.Services) == 0 {
		return &generated.DescribeServicesResponse{
			Services: services,
			Failures: failures,
		}, nil
	}

	for _, serviceName := range req.Services {
		storageService, err := api.storage.ServiceStore().Get(ctx, clusterARN, serviceName)
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, clusterName, serviceName)),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String(err.Error()),
			})
			continue
		}

		service := storageServiceToGeneratedService(storageService)
		if service != nil {
			services = append(services, *service)
		}
	}

	return &generated.DescribeServicesResponse{
		Services: services,
		Failures: failures,
	}, nil
}

// ListServices implements the ListServices operation
func (api *DefaultECSAPI) ListServices(ctx context.Context, req *generated.ListServicesRequest) (*generated.ListServicesResponse, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = *req.Cluster
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, clusterName)

	// Set default limit if not specified
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
	}

	// Extract launch type if specified
	var launchType string
	if req.LaunchType != nil {
		launchType = string(*req.LaunchType)
	}

	// Extract next token if specified
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get services from storage
	storageServices, newNextToken, err := api.storage.ServiceStore().List(ctx, clusterARN, "", launchType, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Extract ARNs
	serviceARNs := make([]string, 0, len(storageServices))
	for _, service := range storageServices {
		serviceARNs = append(serviceARNs, service.ARN)
	}

	response := &generated.ListServicesResponse{
		ServiceArns: serviceARNs,
	}

	// Set next token if there are more results
	if newNextToken != "" {
		response.NextToken = ptr.String(newNextToken)
	}

	return response, nil
}

// ListServicesByNamespace implements the ListServicesByNamespace operation
func (api *DefaultECSAPI) ListServicesByNamespace(ctx context.Context, req *generated.ListServicesByNamespaceRequest) (*generated.ListServicesByNamespaceResponse, error) {
	// Validate required fields
	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}

	// Set default limit if not specified
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
	}

	// Extract next token if specified
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// List services by namespace
	services, newNextToken, err := api.storage.ServiceStore().List(ctx, "", "", "", limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Filter by namespace
	var filteredARNs []string
	for _, service := range services {
		if service.Namespace == req.Namespace {
			filteredARNs = append(filteredARNs, service.ARN)
		}
	}

	response := &generated.ListServicesByNamespaceResponse{
		ServiceArns: filteredARNs,
	}

	// Set next token if there are more results
	if newNextToken != "" {
		response.NextToken = ptr.String(newNextToken)
	}

	return response, nil
}

// UpdateService implements the UpdateService operation
func (api *DefaultECSAPI) UpdateService(ctx context.Context, req *generated.UpdateServiceRequest) (*generated.UpdateServiceResponse, error) {
	// Validate required fields
	if req.Service == "" {
		return nil, fmt.Errorf("service is required")
	}

	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = *req.Cluster
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get existing service
	existingService, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Track if we need to update Kubernetes resources
	needsKubernetesUpdate := false
	oldDesiredCount := existingService.DesiredCount
	oldTaskDefinitionARN := existingService.TaskDefinitionARN

	// Update fields
	// Note: DesiredCount can be 0 (to scale down to 0 tasks)
	if req.DesiredCount != nil && int(*req.DesiredCount) != existingService.DesiredCount {
		log.Printf("DEBUG: Updating desired count from %d to %d", existingService.DesiredCount, req.DesiredCount)
		existingService.DesiredCount = int(*req.DesiredCount)
		needsKubernetesUpdate = true
	}

	if req.TaskDefinition != nil && *req.TaskDefinition != existingService.TaskDefinitionARN {
		// Convert to ARN if necessary
		var newTaskDefArn string
		if !strings.HasPrefix(*req.TaskDefinition, "arn:aws:ecs:") {
			// Check if it's family:revision or just family
			if strings.Contains(*req.TaskDefinition, ":") {
				// family:revision format
				newTaskDefArn = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s", api.region, api.accountID, *req.TaskDefinition)
			} else {
				// Just family - get latest
				latestTaskDef, err := api.storage.TaskDefinitionStore().GetLatest(ctx, *req.TaskDefinition)
				if err != nil || latestTaskDef == nil {
					return nil, fmt.Errorf("task definition not found: %s", *req.TaskDefinition)
				}
				newTaskDefArn = latestTaskDef.ARN
			}
		} else {
			newTaskDefArn = *req.TaskDefinition
		}

		existingService.TaskDefinitionARN = newTaskDefArn
		needsKubernetesUpdate = true
	}

	if req.PlatformVersion != nil && *req.PlatformVersion != "" {
		existingService.PlatformVersion = *req.PlatformVersion
	}

	// Update complex objects if provided
	if req.NetworkConfiguration != nil {
		networkConfigJSON, err := json.Marshal(req.NetworkConfiguration)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal network configuration: %w", err)
		}
		existingService.NetworkConfiguration = string(networkConfigJSON)
		needsKubernetesUpdate = true
	}
	if req.DeploymentConfiguration != nil {
		deploymentConfigJSON, err := json.Marshal(req.DeploymentConfiguration)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal deployment configuration: %w", err)
		}
		existingService.DeploymentConfiguration = string(deploymentConfigJSON)
	}
	if req.PlacementConstraints != nil {
		placementConstraintsJSON, err := json.Marshal(req.PlacementConstraints)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal placement constraints: %w", err)
		}
		existingService.PlacementConstraints = string(placementConstraintsJSON)
	}
	if req.PlacementStrategy != nil {
		placementStrategyJSON, err := json.Marshal(req.PlacementStrategy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal placement strategy: %w", err)
		}
		existingService.PlacementStrategy = string(placementStrategyJSON)
	}
	if req.CapacityProviderStrategy != nil {
		capacityProviderStrategyJSON, err := json.Marshal(req.CapacityProviderStrategy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal capacity provider strategy: %w", err)
		}
		existingService.CapacityProviderStrategy = string(capacityProviderStrategyJSON)
	}
	if req.LoadBalancers != nil {
		loadBalancersJSON, err := json.Marshal(req.LoadBalancers)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal load balancers: %w", err)
		}
		existingService.LoadBalancers = string(loadBalancersJSON)
		needsKubernetesUpdate = true
	}
	if req.ServiceRegistries != nil {
		serviceRegistriesJSON, err := json.Marshal(req.ServiceRegistries)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal service registries: %w", err)
		}
		existingService.ServiceRegistries = string(serviceRegistriesJSON)
	}
	if req.ServiceConnectConfiguration != nil {
		serviceConnectConfigJSON, err := json.Marshal(req.ServiceConnectConfiguration)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal service connect configuration: %w", err)
		}
		existingService.ServiceConnectConfiguration = string(serviceConnectConfigJSON)
	}

	if req.EnableECSManagedTags != nil {
		existingService.EnableECSManagedTags = *req.EnableECSManagedTags
	}
	if req.EnableExecuteCommand != nil {
		existingService.EnableExecuteCommand = *req.EnableExecuteCommand
	}
	if req.HealthCheckGracePeriodSeconds != nil {
		existingService.HealthCheckGracePeriodSeconds = int(*req.HealthCheckGracePeriodSeconds)
	}

	// Update timestamps
	existingService.UpdatedAt = time.Now()

	// Update Kubernetes resources if needed
	if needsKubernetesUpdate {
		// Update status to show update in progress
		existingService.Status = "PENDING"

		// Get the task definition
		taskDef := existingService.TaskDefinitionARN
		taskDefinition, err := api.storage.TaskDefinitionStore().GetByARN(ctx, taskDef)
		if err != nil {
			log.Printf("Failed to get task definition %s: %v", taskDef, err)
			// Restore old values on failure
			existingService.DesiredCount = oldDesiredCount
			existingService.TaskDefinitionARN = oldTaskDefinitionARN
			return nil, fmt.Errorf("failed to get task definition: %w", err)
		}

		// Create service converter and manager
		converter := converters.NewServiceConverter(api.region, api.accountID)
		deployment, kubeService, err := converter.ConvertServiceToDeployment(existingService, taskDefinition, cluster)
		if err != nil {
			log.Printf("Failed to convert service to deployment: %v", err)
			return nil, fmt.Errorf("failed to convert service: %w", err)
		}

		// Create service manager and update Kubernetes resources
		serviceManager, err := api.getServiceManager()
		if err != nil {
			log.Printf("Failed to create service manager: %v", err)
			return nil, fmt.Errorf("failed to create service manager: %w", err)
		}
		if err := serviceManager.UpdateService(ctx, deployment, kubeService, cluster, existingService); err != nil {
			log.Printf("Failed to update kubernetes deployment: %v", err)
			return nil, fmt.Errorf("failed to update kubernetes deployment: %w", err)
		}

		// Update status to ACTIVE after successful update
		existingService.Status = "ACTIVE"
	}

	// Single update at the end
	if err := api.storage.ServiceStore().Update(ctx, existingService); err != nil {
		return nil, fmt.Errorf("failed to update service: %w", err)
	}

	// Convert back to API response
	responseService := storageServiceToGeneratedService(existingService)

	return &generated.UpdateServiceResponse{
		Service: responseService,
	}, nil
}

// UpdateServicePrimaryTaskSet implements the UpdateServicePrimaryTaskSet operation
func (api *DefaultECSAPI) UpdateServicePrimaryTaskSet(ctx context.Context, req *generated.UpdateServicePrimaryTaskSetRequest) (*generated.UpdateServicePrimaryTaskSetResponse, error) {
	// Validate required fields
	if req.Service == "" {
		return nil, fmt.Errorf("service is required")
	}
	if req.PrimaryTaskSet == "" {
		return nil, fmt.Errorf("primaryTaskSet is required")
	}

	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = extractClusterNameFromARN(req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get service
	service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Update primary task set
	err = api.storage.TaskSetStore().UpdatePrimary(ctx, service.ARN, req.PrimaryTaskSet)
	if err != nil {
		return nil, fmt.Errorf("failed to update primary task set: %w", err)
	}

	// Get the updated task set
	taskSet, err := api.storage.TaskSetStore().Get(ctx, service.ARN, req.PrimaryTaskSet)
	if err != nil {
		return nil, fmt.Errorf("task set not found: %s", req.PrimaryTaskSet)
	}

	// Build response
	apiTaskSet := &generated.TaskSet{
		Id:                   ptr.String(taskSet.ID),
		TaskSetArn:           ptr.String(taskSet.ARN),
		ServiceArn:           ptr.String(taskSet.ServiceARN),
		ClusterArn:           ptr.String(taskSet.ClusterARN),
		Status:               ptr.String(taskSet.Status),
		TaskDefinition:       ptr.String(taskSet.TaskDefinition),
		StabilityStatus:      (*generated.StabilityStatus)(ptr.String(taskSet.StabilityStatus)),
		ComputedDesiredCount: ptr.Int32(taskSet.ComputedDesiredCount),
		PendingCount:         ptr.Int32(taskSet.PendingCount),
		RunningCount:         ptr.Int32(taskSet.RunningCount),
		CreatedAt:            ptr.Time(taskSet.CreatedAt),
		UpdatedAt:            ptr.Time(taskSet.UpdatedAt),
	}

	// Set optional fields
	if taskSet.LaunchType != "" {
		apiTaskSet.LaunchType = (*generated.LaunchType)(ptr.String(taskSet.LaunchType))
	}
	if taskSet.PlatformVersion != "" {
		apiTaskSet.PlatformVersion = ptr.String(taskSet.PlatformVersion)
	}
	if taskSet.ExternalID != "" {
		apiTaskSet.ExternalId = ptr.String(taskSet.ExternalID)
	}

	// Unmarshal scale if present
	if taskSet.Scale != "" {
		var scale generated.Scale
		if err := json.Unmarshal([]byte(taskSet.Scale), &scale); err == nil {
			apiTaskSet.Scale = &scale
		}
	}

	return &generated.UpdateServicePrimaryTaskSetResponse{
		TaskSet: apiTaskSet,
	}, nil
}

// DescribeServiceDeployments implements the DescribeServiceDeployments operation
func (api *DefaultECSAPI) DescribeServiceDeployments(ctx context.Context, req *generated.DescribeServiceDeploymentsRequest) (*generated.DescribeServiceDeploymentsResponse, error) {
	// Validate required fields
	if len(req.ServiceDeploymentArns) == 0 {
		return nil, fmt.Errorf("serviceDeploymentArns is required")
	}

	var deployments []generated.ServiceDeployment
	var failures []generated.Failure

	// Process each deployment ARN
	for _, deploymentArn := range req.ServiceDeploymentArns {
		// Parse deployment ARN to extract service information
		// Format: arn:aws:ecs:region:account:service-deployment/cluster/service/deployment-id
		parts := strings.Split(deploymentArn, "/")
		if len(parts) < 4 {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(deploymentArn),
				Reason: ptr.String("INVALID_ARN"),
				Detail: ptr.String("Invalid deployment ARN format"),
			})
			continue
		}

		clusterName := parts[len(parts)-3]
		serviceName := parts[len(parts)-2]
		deploymentID := parts[len(parts)-1]

		// Get cluster
		cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(deploymentArn),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String("Cluster not found"),
			})
			continue
		}

		// Get service
		service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, serviceName)
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(deploymentArn),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String("Service not found"),
			})
			continue
		}

		// Create deployment from current service state
		// In a real implementation, we'd track deployment history
		status := generated.ServiceDeploymentStatusSUCCESSFUL
		deployment := generated.ServiceDeployment{
			ServiceDeploymentArn: ptr.String(deploymentArn),
			ServiceArn:           ptr.String(service.ARN),
			ClusterArn:           ptr.String(cluster.ARN),
			Status:               &status,
			CreatedAt:            ptr.Time(service.CreatedAt),
			UpdatedAt:            ptr.Time(service.UpdatedAt),
		}

		// Set deployment configuration if available
		if service.DeploymentConfiguration != "" && service.DeploymentConfiguration != "null" {
			var deploymentConfig generated.DeploymentConfiguration
			if err := json.Unmarshal([]byte(service.DeploymentConfiguration), &deploymentConfig); err == nil {
				deployment.DeploymentConfiguration = &deploymentConfig
			}
		}

		// Set deployment circuit breaker if available
		circuitBreakerStatus := generated.ServiceDeploymentRollbackMonitorsStatusDISABLED
		deployment.DeploymentCircuitBreaker = &generated.ServiceDeploymentCircuitBreaker{
			Status:       &circuitBreakerStatus,
			FailureCount: ptr.Int32(0),
			Threshold:    ptr.Int32(50),
		}

		// Add deployment ID to deployment
		deployment.SourceServiceRevisions = []generated.ServiceRevisionSummary{
			{
				Arn:                ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:service-revision/%s/%s/%s", api.region, api.accountID, clusterName, serviceName, deploymentID)),
				RequestedTaskCount: ptr.Int32(int32(service.DesiredCount)),
				RunningTaskCount:   ptr.Int32(int32(service.RunningCount)),
				PendingTaskCount:   ptr.Int32(int32(service.PendingCount)),
			},
		}

		deployments = append(deployments, deployment)
	}

	// Note: Include field is not part of the current generated types
	// In a real implementation, we would process include fields if they were available

	return &generated.DescribeServiceDeploymentsResponse{
		ServiceDeployments: deployments,
		Failures:           failures,
	}, nil
}

// DescribeServiceRevisions implements the DescribeServiceRevisions operation
func (api *DefaultECSAPI) DescribeServiceRevisions(ctx context.Context, req *generated.DescribeServiceRevisionsRequest) (*generated.DescribeServiceRevisionsResponse, error) {
	// Validate required fields
	if len(req.ServiceRevisionArns) == 0 {
		return nil, fmt.Errorf("serviceRevisionArns is required")
	}

	var revisions []generated.ServiceRevision
	var failures []generated.Failure

	// Process each revision ARN
	for _, revisionArn := range req.ServiceRevisionArns {
		// Parse revision ARN to extract service information
		// Format: arn:aws:ecs:region:account:service-revision/cluster/service/revision-id
		parts := strings.Split(revisionArn, "/")
		if len(parts) < 4 {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(revisionArn),
				Reason: ptr.String("INVALID_ARN"),
				Detail: ptr.String("Invalid revision ARN format"),
			})
			continue
		}

		clusterName := parts[len(parts)-3]
		serviceName := parts[len(parts)-2]
		// revisionID := parts[len(parts)-1] // For future use when we track revision history

		// Get cluster
		cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(revisionArn),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String("Cluster not found"),
			})
			continue
		}

		// Get service
		service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, serviceName)
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(revisionArn),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String("Service not found"),
			})
			continue
		}

		// Create revision from current service state
		// In a real implementation, we'd track revision history
		revision := generated.ServiceRevision{
			ServiceRevisionArn: ptr.String(revisionArn),
			ServiceArn:         ptr.String(service.ARN),
			ClusterArn:         ptr.String(cluster.ARN),
			TaskDefinition:     ptr.String(service.TaskDefinitionARN),
			CreatedAt:          ptr.Time(service.CreatedAt),
		}

		// Set capacity provider strategy if available
		if service.CapacityProviderStrategy != "" && service.CapacityProviderStrategy != "null" {
			var capacityProviderStrategy []generated.CapacityProviderStrategyItem
			if err := json.Unmarshal([]byte(service.CapacityProviderStrategy), &capacityProviderStrategy); err == nil {
				revision.CapacityProviderStrategy = capacityProviderStrategy
			}
		}

		// Set launch type
		if service.LaunchType != "" {
			launchType := generated.LaunchType(service.LaunchType)
			revision.LaunchType = &launchType
		}

		// Set platform version
		if service.PlatformVersion != "" {
			revision.PlatformVersion = ptr.String(service.PlatformVersion)
		}

		// Note: PlacementConstraints and PlacementStrategy are not part of ServiceRevision
		// They would be handled at the service level, not revision level

		// Set network configuration if available
		if service.NetworkConfiguration != "" && service.NetworkConfiguration != "null" {
			var networkConfig generated.NetworkConfiguration
			if err := json.Unmarshal([]byte(service.NetworkConfiguration), &networkConfig); err == nil {
				revision.NetworkConfiguration = &networkConfig
			}
		}

		// Set load balancers if available
		if service.LoadBalancers != "" && service.LoadBalancers != "null" {
			var loadBalancers []generated.LoadBalancer
			if err := json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers); err == nil {
				revision.LoadBalancers = loadBalancers
			}
		}

		// Set service registries if available
		if service.ServiceRegistries != "" && service.ServiceRegistries != "null" {
			var serviceRegistries []generated.ServiceRegistry
			if err := json.Unmarshal([]byte(service.ServiceRegistries), &serviceRegistries); err == nil {
				revision.ServiceRegistries = serviceRegistries
			}
		}

		// Container insights
		revision.ContainerImages = []generated.ContainerImage{
			{
				ContainerName: ptr.String("main"),
				Image:         ptr.String("nginx:latest"), // Placeholder
			},
		}

		// Add guard rails - these would be based on deployment configuration
		revision.GuardDutyEnabled = ptr.Bool(false)
		revision.ServiceConnectConfiguration = &generated.ServiceConnectConfiguration{
			Enabled: false,
		}

		// Add revision-specific metadata
		revision.VolumeConfigurations = []generated.ServiceVolumeConfiguration{}

		revisions = append(revisions, revision)
	}

	return &generated.DescribeServiceRevisionsResponse{
		ServiceRevisions: revisions,
		Failures:         failures,
	}, nil
}

// ListServiceDeployments implements the ListServiceDeployments operation
func (api *DefaultECSAPI) ListServiceDeployments(ctx context.Context, req *generated.ListServiceDeploymentsRequest) (*generated.ListServiceDeploymentsResponse, error) {
	// Validate required fields
	if req.Service == "" {
		return nil, fmt.Errorf("service is required")
	}

	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get service
	service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, req.Service)
	if err != nil {
		return nil, fmt.Errorf("service not found: %w", err)
	}

	// Create deployments
	// In a real implementation, we'd track deployment history
	var deployments []generated.ServiceDeploymentBrief

	// Current deployment
	currentStatus := generated.ServiceDeploymentStatusSUCCESSFUL
	currentDeployment := generated.ServiceDeploymentBrief{
		ServiceDeploymentArn:     ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:service-deployment/%s/%s/current", api.region, api.accountID, clusterName, service.ServiceName)),
		ServiceArn:               ptr.String(service.ARN),
		ClusterArn:               ptr.String(cluster.ARN),
		Status:                   &currentStatus,
		CreatedAt:                ptr.Time(service.UpdatedAt),
		StartedAt:                ptr.Time(service.UpdatedAt),
		TargetServiceRevisionArn: ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:service-revision/%s/%s/current", api.region, api.accountID, clusterName, service.ServiceName)),
	}
	deployments = append(deployments, currentDeployment)

	// Add historical deployments if they exist
	// For now, we'll simulate one previous deployment
	if service.UpdatedAt.After(service.CreatedAt) {
		prevStatus := generated.ServiceDeploymentStatusSUCCESSFUL
		prevDeployment := generated.ServiceDeploymentBrief{
			ServiceDeploymentArn:     ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:service-deployment/%s/%s/previous-1", api.region, api.accountID, clusterName, service.ServiceName)),
			ServiceArn:               ptr.String(service.ARN),
			ClusterArn:               ptr.String(cluster.ARN),
			Status:                   &prevStatus,
			CreatedAt:                ptr.Time(service.CreatedAt),
			StartedAt:                ptr.Time(service.CreatedAt),
			FinishedAt:               ptr.Time(service.UpdatedAt.Add(-1 * time.Hour)), // Simulate finished 1 hour before update
			TargetServiceRevisionArn: ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:service-revision/%s/%s/previous-1", api.region, api.accountID, clusterName, service.ServiceName)),
		}
		deployments = append(deployments, prevDeployment)
	}

	// Apply status filter if specified
	if len(req.Status) > 0 {
		// Filter deployments by status
		filteredDeployments := []generated.ServiceDeploymentBrief{}
		for _, deployment := range deployments {
			for _, status := range req.Status {
				if deployment.Status != nil && *deployment.Status == status {
					filteredDeployments = append(filteredDeployments, deployment)
					break
				}
			}
		}
		deployments = filteredDeployments
	}

	// Apply pagination
	maxResults := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		maxResults = int(*req.MaxResults)
	}

	var nextToken *string
	if len(deployments) > maxResults {
		deployments = deployments[:maxResults]
		nextToken = ptr.String(*deployments[maxResults-1].ServiceDeploymentArn)
	}

	response := &generated.ListServiceDeploymentsResponse{
		ServiceDeployments: deployments,
	}

	if nextToken != nil {
		response.NextToken = nextToken
	}

	return response, nil
}

// StopServiceDeployment implements the StopServiceDeployment operation
func (api *DefaultECSAPI) StopServiceDeployment(ctx context.Context, req *generated.StopServiceDeploymentRequest) (*generated.StopServiceDeploymentResponse, error) {
	// Validate required fields
	if req.ServiceDeploymentArn == "" {
		return nil, fmt.Errorf("serviceDeploymentArn is required")
	}

	// Parse deployment ARN to extract service information
	// Format: arn:aws:ecs:region:account:service-deployment/cluster/service/deployment-id
	parts := strings.Split(req.ServiceDeploymentArn, "/")
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid deployment ARN format")
	}

	clusterName := parts[len(parts)-3]
	serviceName := parts[len(parts)-2]
	// deploymentID := parts[len(parts)-1] // For future use

	// Get cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get service to verify it exists
	_, err = api.storage.ServiceStore().Get(ctx, cluster.ARN, serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", serviceName)
	}

	// In a real implementation, we'd actually stop the deployment
	// For now, we just validate the request and return success

	return &generated.StopServiceDeploymentResponse{
		ServiceDeploymentArn: ptr.String(req.ServiceDeploymentArn),
	}, nil
}

// storageServiceToGeneratedService converts a storage.Service to generated.Service
func storageServiceToGeneratedService(storageService *storage.Service) *generated.Service {
	if storageService == nil {
		return nil
	}

	service := &generated.Service{
		ServiceArn:                    ptr.String(storageService.ARN),
		ServiceName:                   ptr.String(storageService.ServiceName),
		ClusterArn:                    ptr.String(storageService.ClusterARN),
		Status:                        ptr.String(storageService.Status),
		DesiredCount:                  ptr.Int32(int32(storageService.DesiredCount)),
		RunningCount:                  ptr.Int32(int32(storageService.RunningCount)),
		PendingCount:                  ptr.Int32(int32(storageService.PendingCount)),
		TaskDefinition:                ptr.String(storageService.TaskDefinitionARN),
		SchedulingStrategy:            (*generated.SchedulingStrategy)(ptr.String(storageService.SchedulingStrategy)),
		EnableECSManagedTags:          ptr.Bool(storageService.EnableECSManagedTags),
		EnableExecuteCommand:          ptr.Bool(storageService.EnableExecuteCommand),
		HealthCheckGracePeriodSeconds: ptr.Int32(int32(storageService.HealthCheckGracePeriodSeconds)),
		CreatedAt:                     ptr.Time(storageService.CreatedAt),
	}

	// Set optional fields
	if storageService.LaunchType != "" {
		launchType := generated.LaunchType(storageService.LaunchType)
		service.LaunchType = &launchType
	}
	if storageService.PlatformVersion != "" {
		service.PlatformVersion = ptr.String(storageService.PlatformVersion)
	}
	if storageService.RoleARN != "" {
		service.RoleArn = ptr.String(storageService.RoleARN)
	}
	if storageService.PropagateTags != "" {
		propagateTags := generated.PropagateTags(storageService.PropagateTags)
		service.PropagateTags = &propagateTags
	}

	// Parse JSON fields
	if storageService.LoadBalancers != "" && storageService.LoadBalancers != "null" {
		var loadBalancers []generated.LoadBalancer
		if err := json.Unmarshal([]byte(storageService.LoadBalancers), &loadBalancers); err == nil {
			service.LoadBalancers = loadBalancers
		}
	}
	if storageService.ServiceRegistries != "" && storageService.ServiceRegistries != "null" {
		var serviceRegistries []generated.ServiceRegistry
		if err := json.Unmarshal([]byte(storageService.ServiceRegistries), &serviceRegistries); err == nil {
			service.ServiceRegistries = serviceRegistries
		}
	}
	if storageService.NetworkConfiguration != "" && storageService.NetworkConfiguration != "null" {
		var networkConfig generated.NetworkConfiguration
		if err := json.Unmarshal([]byte(storageService.NetworkConfiguration), &networkConfig); err == nil {
			service.NetworkConfiguration = &networkConfig
		}
	}
	// Always set deployment configuration with defaults if not provided
	deploymentConfig := &generated.DeploymentConfiguration{
		MaximumPercent:        ptr.Int32(200),
		MinimumHealthyPercent: ptr.Int32(100),
	}

	if storageService.DeploymentConfiguration != "" && storageService.DeploymentConfiguration != "null" {
		// Override defaults with stored configuration
		if err := json.Unmarshal([]byte(storageService.DeploymentConfiguration), deploymentConfig); err == nil {
			service.DeploymentConfiguration = deploymentConfig
		} else {
			// If unmarshal fails, use defaults
			service.DeploymentConfiguration = deploymentConfig
		}
	} else {
		// No configuration stored, use defaults
		service.DeploymentConfiguration = deploymentConfig
	}
	if storageService.PlacementConstraints != "" && storageService.PlacementConstraints != "null" {
		var placementConstraints []generated.PlacementConstraint
		if err := json.Unmarshal([]byte(storageService.PlacementConstraints), &placementConstraints); err == nil {
			service.PlacementConstraints = placementConstraints
		}
	}
	if storageService.PlacementStrategy != "" && storageService.PlacementStrategy != "null" {
		var placementStrategy []generated.PlacementStrategy
		if err := json.Unmarshal([]byte(storageService.PlacementStrategy), &placementStrategy); err == nil {
			service.PlacementStrategy = placementStrategy
		}
	}
	if storageService.CapacityProviderStrategy != "" && storageService.CapacityProviderStrategy != "null" {
		var capacityProviderStrategy []generated.CapacityProviderStrategyItem
		if err := json.Unmarshal([]byte(storageService.CapacityProviderStrategy), &capacityProviderStrategy); err == nil {
			service.CapacityProviderStrategy = capacityProviderStrategy
		}
	}
	if storageService.Tags != "" && storageService.Tags != "null" {
		var tags []generated.Tag
		if err := json.Unmarshal([]byte(storageService.Tags), &tags); err == nil {
			service.Tags = tags
		}
	}

	// Add deployment information
	// In AWS ECS, there's always at least one deployment representing the current state
	deployment := generated.Deployment{
		Id:             ptr.String(fmt.Sprintf("ecs-svc/%s", storageService.ServiceName)),
		Status:         ptr.String("PRIMARY"),
		TaskDefinition: ptr.String(storageService.TaskDefinitionARN),
		DesiredCount:   ptr.Int32(int32(storageService.DesiredCount)),
		RunningCount:   ptr.Int32(int32(storageService.RunningCount)),
		PendingCount:   ptr.Int32(int32(storageService.PendingCount)),
		CreatedAt:      ptr.Time(storageService.CreatedAt),
		UpdatedAt:      ptr.Time(storageService.UpdatedAt),
	}

	if storageService.LaunchType != "" {
		launchType := generated.LaunchType(storageService.LaunchType)
		deployment.LaunchType = &launchType
	}
	if storageService.PlatformVersion != "" {
		deployment.PlatformVersion = ptr.String(storageService.PlatformVersion)
	}

	service.Deployments = []generated.Deployment{deployment}

	return service
}

// registerServiceWithDiscovery registers the service with service discovery
func (api *DefaultECSAPI) registerServiceWithDiscovery(ctx context.Context, service *storage.Service, serviceRegistries []generated.ServiceRegistry) error {
	// Check if service discovery manager is available
	if api.serviceDiscoveryManager == nil {
		return fmt.Errorf("service discovery not configured")
	}

	// For each service registry
	for _, registry := range serviceRegistries {
		if registry.RegistryArn == nil {
			continue
		}

		// Parse the registry ARN to extract service ID
		// Format: arn:aws:servicediscovery:region:account-id:service/srv-xxxxx
		arnParts := strings.Split(*registry.RegistryArn, ":")
		if len(arnParts) < 6 {
			log.Printf("Invalid service registry ARN: %s", *registry.RegistryArn)
			continue
		}

		resourceParts := strings.Split(arnParts[5], "/")
		if len(resourceParts) < 2 || resourceParts[0] != "service" {
			log.Printf("Invalid service registry resource: %s", arnParts[5])
			continue
		}

		serviceID := resourceParts[1]

		// Register service instances (tasks will register themselves when they start)
		// For now, we'll just log the registration intent
		log.Printf("Service %s registered with service discovery service %s", service.ServiceName, serviceID)

		// Store the service registry information in service metadata
		// This will be used by tasks when they start
		if service.ServiceRegistryMetadata == nil {
			service.ServiceRegistryMetadata = make(map[string]string)
		}

		containerName := ""
		if registry.ContainerName != nil {
			containerName = *registry.ContainerName
		}
		containerPort := int32(0)
		if registry.ContainerPort != nil {
			containerPort = *registry.ContainerPort
		}
		service.ServiceRegistryMetadata[serviceID] = fmt.Sprintf("{\"containerName\":\"%s\",\"containerPort\":%d}",
			containerName, containerPort)
	}

	// Update service in storage with registry metadata
	if err := api.storage.ServiceStore().Update(ctx, service); err != nil {
		return fmt.Errorf("failed to update service with registry metadata: %w", err)
	}

	return nil
}

// createTasksForService creates tasks for a service in test mode
func (api *DefaultECSAPI) createTasksForService(ctx context.Context, service *storage.Service, taskDef *storage.TaskDefinition, cluster *storage.Cluster) error {
	// In test mode, we create tasks directly in storage without kubernetes resources
	for i := 0; i < service.DesiredCount; i++ {
		// Generate task ID
		taskID := uuid.New().String()
		taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", api.region, api.accountID, cluster.Name, taskID)
		
		// Create storage task
		now := time.Now()
		task := &storage.Task{
			ID:                 taskID,
			ARN:                taskARN,
			ClusterARN:         cluster.ARN,
			TaskDefinitionARN:  taskDef.ARN,
			LastStatus:         "PROVISIONING",
			DesiredStatus:      "RUNNING",
			LaunchType:         service.LaunchType,
			StartedBy:          fmt.Sprintf("ecs-svc/%s", service.ServiceName),
			CreatedAt:          now,
			Version:            1,
			CPU:                taskDef.CPU,
			Memory:             taskDef.Memory,
			ContainerInstanceARN: "", // Empty in test mode
			Group:              fmt.Sprintf("service:%s", service.ServiceName),
			Containers:         "[]", // Empty containers JSON initially
		}
		
		// Save task to storage
		if err := api.storage.TaskStore().Create(ctx, task); err != nil {
			return fmt.Errorf("failed to create task %d: %w", i, err)
		}
		
		log.Printf("Created task %s for service %s in test mode", taskID, service.ServiceName)
	}
	
	// Update service counts
	service.RunningCount = service.DesiredCount
	service.PendingCount = 0
	service.Status = "ACTIVE"
	service.UpdatedAt = time.Now()
	
	if err := api.storage.ServiceStore().Update(ctx, service); err != nil {
		return fmt.Errorf("failed to update service counts: %w", err)
	}
	
	return nil
}
