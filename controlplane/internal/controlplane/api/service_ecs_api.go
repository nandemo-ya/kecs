package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

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
	if req.ServiceName == nil {
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
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster.Name, *req.ServiceName)
	clusterARN := cluster.ARN

	// Set default values
	launchType := generated.LaunchTypeFargate
	if req.LaunchType != nil {
		launchType = *req.LaunchType
	}
	
	schedulingStrategy := generated.SchedulingStrategyReplica
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

	// Create service converter and manager
	serviceConverter := converters.NewServiceConverter(api.region, api.accountID)
	serviceManager := kubernetes.NewServiceManager(api.storage, api.kindManager)

	// Convert ECS service to Kubernetes Deployment
	storageServiceTemp := &storage.Service{
		ServiceName:  *req.ServiceName,
		DesiredCount: int(desiredCount),
		LaunchType:   string(launchType),
	}
	deployment, kubeService, err := serviceConverter.ConvertServiceToDeployment(
		storageServiceTemp,
		taskDef,
		cluster,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to convert service to deployment: %w", err)
	}

	// Create storage service with deployment information
	namespace := fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
	deploymentName := fmt.Sprintf("ecs-service-%s", *req.ServiceName)
	
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
		ServiceName:                   *req.ServiceName,
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

	// Convert storage service to API response
	responseService := storageServiceToGeneratedService(storageService)

	return &generated.CreateServiceResponse{
		Service: responseService,
	}, nil
}

// DeleteService implements the DeleteService operation
func (api *DefaultECSAPI) DeleteService(ctx context.Context, req *generated.DeleteServiceRequest) (*generated.DeleteServiceResponse, error) {
	// Validate required fields
	if req.Service == nil {
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
	existingService, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, *req.Service)
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
	serviceManager := kubernetes.NewServiceManager(api.storage, api.kindManager)
	if err := serviceManager.DeleteService(ctx, cluster, existingService); err != nil {
		log.Printf("Warning: failed to delete Kubernetes resources for service %s: %v", 
			existingService.ServiceName, err)
		// Continue with deletion even if Kubernetes deletion fails
		// This matches AWS ECS behavior where the service is marked for deletion
		// even if underlying resources might still exist
	}

	// Delete from storage
	if err := api.storage.ServiceStore().Delete(ctx, cluster.ARN, *req.Service); err != nil {
		return nil, fmt.Errorf("failed to delete service: %w", err)
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
	// TODO: Implement ListServicesByNamespace
	return nil, fmt.Errorf("ListServicesByNamespace not implemented")
}

// UpdateService implements the UpdateService operation
func (api *DefaultECSAPI) UpdateService(ctx context.Context, req *generated.UpdateServiceRequest) (*generated.UpdateServiceResponse, error) {
	// Validate required fields
	if req.Service == nil {
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
	existingService, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, *req.Service)
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
		log.Printf("DEBUG: Updating desired count from %d to %d", existingService.DesiredCount, *req.DesiredCount)
		existingService.DesiredCount = int(*req.DesiredCount)
		needsKubernetesUpdate = true
	}
	
	if req.TaskDefinition != nil && *req.TaskDefinition != "" && *req.TaskDefinition != existingService.TaskDefinitionARN {
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
	if req.HealthCheckGracePeriodSeconds != nil && *req.HealthCheckGracePeriodSeconds > 0 {
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
		serviceManager := kubernetes.NewServiceManager(api.storage, api.kindManager)
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
	// TODO: Implement UpdateServicePrimaryTaskSet
	return nil, fmt.Errorf("UpdateServicePrimaryTaskSet not implemented")
}

// DescribeServiceDeployments implements the DescribeServiceDeployments operation
func (api *DefaultECSAPI) DescribeServiceDeployments(ctx context.Context, req *generated.DescribeServiceDeploymentsRequest) (*generated.DescribeServiceDeploymentsResponse, error) {
	// TODO: Implement DescribeServiceDeployments
	return nil, fmt.Errorf("DescribeServiceDeployments not implemented")
}

// DescribeServiceRevisions implements the DescribeServiceRevisions operation
func (api *DefaultECSAPI) DescribeServiceRevisions(ctx context.Context, req *generated.DescribeServiceRevisionsRequest) (*generated.DescribeServiceRevisionsResponse, error) {
	// TODO: Implement DescribeServiceRevisions
	return nil, fmt.Errorf("DescribeServiceRevisions not implemented")
}

// ListServiceDeployments implements the ListServiceDeployments operation  
func (api *DefaultECSAPI) ListServiceDeployments(ctx context.Context, req *generated.ListServiceDeploymentsRequest) (*generated.ListServiceDeploymentsResponse, error) {
	// TODO: Implement ListServiceDeployments
	return nil, fmt.Errorf("ListServiceDeployments not implemented")
}

// StopServiceDeployment implements the StopServiceDeployment operation
func (api *DefaultECSAPI) StopServiceDeployment(ctx context.Context, req *generated.StopServiceDeploymentRequest) (*generated.StopServiceDeploymentResponse, error) {
	// TODO: Implement StopServiceDeployment
	return nil, fmt.Errorf("StopServiceDeployment not implemented")
}

// storageServiceToGeneratedService converts a storage.Service to generated.Service
func storageServiceToGeneratedService(storageService *storage.Service) *generated.Service {
	if storageService == nil {
		return nil
	}

	service := &generated.Service{
		ServiceArn:                   ptr.String(storageService.ARN),
		ServiceName:                  ptr.String(storageService.ServiceName),
		ClusterArn:                   ptr.String(storageService.ClusterARN),
		Status:                       ptr.String(storageService.Status),
		DesiredCount:                 ptr.Int32(int32(storageService.DesiredCount)),
		RunningCount:                 ptr.Int32(int32(storageService.RunningCount)),
		PendingCount:                 ptr.Int32(int32(storageService.PendingCount)),
		TaskDefinition:               ptr.String(storageService.TaskDefinitionARN),
		SchedulingStrategy:           (*generated.SchedulingStrategy)(ptr.String(storageService.SchedulingStrategy)),
		EnableECSManagedTags:         ptr.Bool(storageService.EnableECSManagedTags),
		EnableExecuteCommand:         ptr.Bool(storageService.EnableExecuteCommand),
		HealthCheckGracePeriodSeconds: ptr.Int32(int32(storageService.HealthCheckGracePeriodSeconds)),
		CreatedAt:                    ptr.Time(storageService.CreatedAt),
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
	if storageService.DeploymentConfiguration != "" && storageService.DeploymentConfiguration != "null" {
		var deploymentConfig generated.DeploymentConfiguration
		if err := json.Unmarshal([]byte(storageService.DeploymentConfiguration), &deploymentConfig); err == nil {
			service.DeploymentConfiguration = &deploymentConfig
		}
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
		Id:              ptr.String(fmt.Sprintf("ecs-svc/%s", storageService.ServiceName)),
		Status:          ptr.String("PRIMARY"),
		TaskDefinition:  ptr.String(storageService.TaskDefinitionARN),
		DesiredCount:    ptr.Int32(int32(storageService.DesiredCount)),
		RunningCount:    ptr.Int32(int32(storageService.RunningCount)),
		PendingCount:    ptr.Int32(int32(storageService.PendingCount)),
		CreatedAt:       ptr.Time(storageService.CreatedAt),
		UpdatedAt:       ptr.Time(storageService.UpdatedAt),
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