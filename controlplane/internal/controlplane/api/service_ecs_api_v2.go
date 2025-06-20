package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CreateServiceV2 implements the CreateService operation using AWS SDK types
func (api *DefaultECSAPIV2) CreateServiceV2(ctx context.Context, req *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
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
			taskDefArn = fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/%s", taskDefArn)
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

	if err != nil || taskDef == nil {
		log.Printf("DEBUG: Error getting task definition: %v", err)
		return nil, fmt.Errorf("task definition not found: %s", *req.TaskDefinition)
	}

	// Generate ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:service/%s/%s", cluster.Name, *req.ServiceName)
	clusterARN := cluster.ARN

	// Set default values
	launchType := types.LaunchTypeFargate
	if req.LaunchType != "" {
		launchType = req.LaunchType
	}

	schedulingStrategy := types.SchedulingStrategyReplica
	if req.SchedulingStrategy != "" {
		schedulingStrategy = req.SchedulingStrategy
	}

	desiredCount := int32(1)
	if req.DesiredCount != nil {
		desiredCount = *req.DesiredCount
	}

	// Convert complex objects to JSON for storage
	loadBalancersJSON, _ := json.Marshal(req.LoadBalancers)
	serviceRegistriesJSON, _ := json.Marshal(req.ServiceRegistries)
	networkConfigJSON, _ := json.Marshal(req.NetworkConfiguration)
	deploymentConfigJSON, _ := json.Marshal(req.DeploymentConfiguration)
	placementConstraintsJSON, _ := json.Marshal(req.PlacementConstraints)
	placementStrategyJSON, _ := json.Marshal(req.PlacementStrategy)
	serviceConnectConfigJSON, _ := json.Marshal(req.ServiceConnectConfiguration)
	// volumeConfigsJSON, _ := json.Marshal(req.VolumeConfigurations)
	capacityProviderStrategyJSON, _ := json.Marshal(req.CapacityProviderStrategy)
	tagsJSON, _ := json.Marshal(req.Tags)

	// Create service in storage
	service := &storage.Service{
		ServiceName:                   *req.ServiceName,
		ARN:                           serviceARN,
		ClusterARN:                    clusterARN,
		TaskDefinitionARN:             taskDefArn,
		DesiredCount:                  int(desiredCount),
		RunningCount:                  0,
		PendingCount:                  0,
		LaunchType:                    string(launchType),
		SchedulingStrategy:            string(schedulingStrategy),
		Status:                        "ACTIVE",
		LoadBalancers:                 string(loadBalancersJSON),
		ServiceRegistries:             string(serviceRegistriesJSON),
		NetworkConfiguration:          string(networkConfigJSON),
		DeploymentConfiguration:       string(deploymentConfigJSON),
		PlacementConstraints:          string(placementConstraintsJSON),
		PlacementStrategy:             string(placementStrategyJSON),
		ServiceConnectConfiguration:   string(serviceConnectConfigJSON),
		CapacityProviderStrategy:      string(capacityProviderStrategyJSON),
		Tags:                          string(tagsJSON),
		PlatformVersion:               aws.ToString(req.PlatformVersion),
		RoleARN:                       aws.ToString(req.Role),
		HealthCheckGracePeriodSeconds: int(aws.ToInt32(req.HealthCheckGracePeriodSeconds)),
		EnableECSManagedTags:          req.EnableECSManagedTags,
		EnableExecuteCommand:          req.EnableExecuteCommand,
		PropagateTags:                 string(req.PropagateTags),
		Region:                        "ap-northeast-1",
		AccountID:                     "123456789012",
		CreatedAt:                     time.Now(),
		UpdatedAt:                     time.Now(),
	}

	if err := api.storage.ServiceStore().Create(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	// Create deployment
	deployment := types.Deployment{
		Id:                       aws.String(fmt.Sprintf("ecs-svc/%d", time.Now().Unix())),
		Status:                   aws.String("PRIMARY"),
		TaskDefinition:           aws.String(taskDefArn),
		DesiredCount:             desiredCount,
		PendingCount:             0,
		RunningCount:             0,
		FailedTasks:              0,
		CreatedAt:                aws.Time(time.Now()),
		UpdatedAt:                aws.Time(time.Now()),
		LaunchType:               launchType,
		PlatformVersion:          req.PlatformVersion,
		NetworkConfiguration:     req.NetworkConfiguration,
		RolloutState:             types.DeploymentRolloutStateInProgress,
		RolloutStateReason:       aws.String("ECS deployment in progress"),
		ServiceConnectConfiguration: req.ServiceConnectConfiguration,
		VolumeConfigurations:     req.VolumeConfigurations,
		CapacityProviderStrategy: req.CapacityProviderStrategy,
	}

	// Update cluster service count
	cluster.ActiveServicesCount++
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		log.Printf("Failed to update cluster service count: %v", err)
	}

	// Create tasks if Kubernetes manager is available
	if api.kindManager != nil {
		go api.createServiceTasks(ctx, cluster, service, taskDef, int(desiredCount))
	}

	// Build response
	response := &ecs.CreateServiceOutput{
		Service: &types.Service{
			ServiceArn:               aws.String(serviceARN),
			ServiceName:              aws.String(*req.ServiceName),
			ClusterArn:               aws.String(clusterARN),
			TaskDefinition:           aws.String(taskDefArn),
			DesiredCount:             desiredCount,
			RunningCount:             0,
			PendingCount:             0,
			LaunchType:               launchType,
			SchedulingStrategy:       schedulingStrategy,
			Status:                   aws.String("ACTIVE"),
			LoadBalancers:            req.LoadBalancers,
			ServiceRegistries:        req.ServiceRegistries,
			NetworkConfiguration:     req.NetworkConfiguration,
			DeploymentConfiguration:  req.DeploymentConfiguration,
			PlacementConstraints:     req.PlacementConstraints,
			PlacementStrategy:        req.PlacementStrategy,
			Deployments:              []types.Deployment{deployment},
			RoleArn:                  req.Role,
			CreatedAt:                aws.Time(time.Now()),
			CreatedBy:                aws.String("kecs"),
			HealthCheckGracePeriodSeconds: req.HealthCheckGracePeriodSeconds,
			EnableECSManagedTags:     req.EnableECSManagedTags,
			EnableExecuteCommand:     req.EnableExecuteCommand,
			PropagateTags:            req.PropagateTags,
			Tags:                     req.Tags,
			CapacityProviderStrategy: req.CapacityProviderStrategy,
		},
	}

	return response, nil
}

// ListServicesV2 implements the ListServices operation using AWS SDK types
func (api *DefaultECSAPIV2) ListServicesV2(ctx context.Context, req *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Set default limit
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		if limit > 100 {
			limit = 100
		}
	}

	// Extract next token
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get services with optional filtering
	var services []*storage.Service
	var newNextToken string

	// Use the List method with appropriate filters
	launchTypeFilter := ""
	if req.LaunchType != "" {
		launchTypeFilter = string(req.LaunchType)
	}

	services, newNextToken, err = api.storage.ServiceStore().List(ctx, cluster.ARN, "", launchTypeFilter, limit, nextToken)

	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	// Build service ARN list
	serviceArns := make([]string, 0, len(services))
	for _, service := range services {
		serviceArns = append(serviceArns, service.ARN)
	}

	response := &ecs.ListServicesOutput{
		ServiceArns: serviceArns,
	}

	if newNextToken != "" {
		response.NextToken = aws.String(newNextToken)
	}

	return response, nil
}

// DescribeServicesV2 implements the DescribeServices operation using AWS SDK types
func (api *DefaultECSAPIV2) DescribeServicesV2(ctx context.Context, req *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// If no services specified, describe all services in cluster
	serviceIdentifiers := req.Services
	if len(serviceIdentifiers) == 0 {
		services, _, err := api.storage.ServiceStore().List(ctx, cluster.ARN, "", "", 100, "")
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}
		for _, service := range services {
			serviceIdentifiers = append(serviceIdentifiers, service.ServiceName)
		}
	}

	// Fetch details for each service
	var describedServices []types.Service
	var failures []types.Failure

	for _, identifier := range serviceIdentifiers {
		// Extract service name from ARN if necessary
		serviceName := extractServiceNameFromARN(identifier)

		service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, serviceName)
		if err != nil {
			failures = append(failures, types.Failure{
				Arn:    aws.String(identifier),
				Reason: aws.String("MISSING"),
				Detail: aws.String(fmt.Sprintf("Could not find service %s", identifier)),
			})
			continue
		}

		// Parse JSON fields
		var loadBalancers []types.LoadBalancer
		if service.LoadBalancers != "" {
			json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers)
		}

		var serviceRegistries []types.ServiceRegistry
		if service.ServiceRegistries != "" {
			json.Unmarshal([]byte(service.ServiceRegistries), &serviceRegistries)
		}

		var networkConfig *types.NetworkConfiguration
		if service.NetworkConfiguration != "" {
			json.Unmarshal([]byte(service.NetworkConfiguration), &networkConfig)
		}

		var deploymentConfig *types.DeploymentConfiguration
		if service.DeploymentConfiguration != "" {
			json.Unmarshal([]byte(service.DeploymentConfiguration), &deploymentConfig)
		}

		var placementConstraints []types.PlacementConstraint
		if service.PlacementConstraints != "" {
			json.Unmarshal([]byte(service.PlacementConstraints), &placementConstraints)
		}

		var placementStrategy []types.PlacementStrategy
		if service.PlacementStrategy != "" {
			json.Unmarshal([]byte(service.PlacementStrategy), &placementStrategy)
		}

		var serviceConnectConfig *types.ServiceConnectConfiguration
		if service.ServiceConnectConfiguration != "" {
			json.Unmarshal([]byte(service.ServiceConnectConfiguration), &serviceConnectConfig)
		}

		// VolumeConfigurations not currently stored
		// var volumeConfigs []types.ServiceVolumeConfiguration

		var capacityProviderStrategy []types.CapacityProviderStrategyItem
		if service.CapacityProviderStrategy != "" {
			json.Unmarshal([]byte(service.CapacityProviderStrategy), &capacityProviderStrategy)
		}

		var tags []types.Tag
		if service.Tags != "" {
			json.Unmarshal([]byte(service.Tags), &tags)
		}

		// Create deployment
		deployment := types.Deployment{
			Id:                       aws.String(fmt.Sprintf("ecs-svc/%d", service.CreatedAt.Unix())),
			Status:                   aws.String("PRIMARY"),
			TaskDefinition:           aws.String(service.TaskDefinitionARN),
			DesiredCount:             int32(service.DesiredCount),
			PendingCount:             int32(service.PendingCount),
			RunningCount:             int32(service.RunningCount),
			FailedTasks:              0,
			CreatedAt:                aws.Time(service.CreatedAt),
			UpdatedAt:                aws.Time(service.UpdatedAt),
			LaunchType:               types.LaunchType(service.LaunchType),
			PlatformVersion:          aws.String(service.PlatformVersion),
			// PlatformFamily not stored in service
			NetworkConfiguration:     networkConfig,
			RolloutState:             types.DeploymentRolloutStateCompleted,
			RolloutStateReason:       aws.String("ECS deployment completed"),
			CapacityProviderStrategy: capacityProviderStrategy,
		}

		// Build service response
		serviceResp := types.Service{
			ServiceArn:               aws.String(service.ARN),
			ServiceName:              aws.String(service.ServiceName),
			ClusterArn:               aws.String(service.ClusterARN),
			TaskDefinition:           aws.String(service.TaskDefinitionARN),
			DesiredCount:             int32(service.DesiredCount),
			RunningCount:             int32(service.RunningCount),
			PendingCount:             int32(service.PendingCount),
			LaunchType:               types.LaunchType(service.LaunchType),
			SchedulingStrategy:       types.SchedulingStrategy(service.SchedulingStrategy),
			Status:                   aws.String(service.Status),
			LoadBalancers:            loadBalancers,
			ServiceRegistries:        serviceRegistries,
			NetworkConfiguration:     networkConfig,
			DeploymentConfiguration:  deploymentConfig,
			PlacementConstraints:     placementConstraints,
			PlacementStrategy:        placementStrategy,
			Deployments:              []types.Deployment{deployment},
			RoleArn:                  aws.String(service.RoleARN),
			CreatedAt:                aws.Time(service.CreatedAt),
			CreatedBy:                aws.String("kecs"),
			HealthCheckGracePeriodSeconds: aws.Int32(int32(service.HealthCheckGracePeriodSeconds)),
			EnableECSManagedTags:     service.EnableECSManagedTags,
			EnableExecuteCommand:     service.EnableExecuteCommand,
			PropagateTags:            types.PropagateTags(service.PropagateTags),
			Tags:                     tags,
			CapacityProviderStrategy: capacityProviderStrategy,
		}

		// Add events if requested
		if req.Include != nil {
			for _, include := range req.Include {
				if include == types.ServiceFieldTags {
					// Tags already included
				}
			}
		}

		describedServices = append(describedServices, serviceResp)
	}

	return &ecs.DescribeServicesOutput{
		Services: describedServices,
		Failures: failures,
	}, nil
}

// UpdateServiceV2 implements the UpdateService operation using AWS SDK types
func (api *DefaultECSAPIV2) UpdateServiceV2(ctx context.Context, req *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Validate required fields
	if req.Service == nil {
		return nil, fmt.Errorf("service is required")
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Extract service name from ARN if necessary
	serviceName := extractServiceNameFromARN(*req.Service)

	// Get service from storage
	service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", *req.Service)
	}

	// Update fields if provided
	updated := false

	if req.DesiredCount != nil {
		service.DesiredCount = int(*req.DesiredCount)
		updated = true
	}

	if req.TaskDefinition != nil {
		// Validate task definition exists
		taskDefArn := *req.TaskDefinition
		if !strings.HasPrefix(taskDefArn, "arn:aws:ecs:") {
			if strings.Contains(taskDefArn, ":") {
				taskDefArn = fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/%s", taskDefArn)
			} else {
				taskDef, err := api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefArn)
				if err != nil || taskDef == nil {
					return nil, fmt.Errorf("task definition not found: %s", *req.TaskDefinition)
				}
				taskDefArn = taskDef.ARN
			}
		}
		service.TaskDefinitionARN = taskDefArn
		updated = true
	}

	if req.CapacityProviderStrategy != nil {
		strategyJSON, _ := json.Marshal(req.CapacityProviderStrategy)
		service.CapacityProviderStrategy = string(strategyJSON)
		updated = true
	}

	if req.NetworkConfiguration != nil {
		networkJSON, _ := json.Marshal(req.NetworkConfiguration)
		service.NetworkConfiguration = string(networkJSON)
		updated = true
	}

	if req.PlacementConstraints != nil {
		constraintsJSON, _ := json.Marshal(req.PlacementConstraints)
		service.PlacementConstraints = string(constraintsJSON)
		updated = true
	}

	if req.PlacementStrategy != nil {
		strategyJSON, _ := json.Marshal(req.PlacementStrategy)
		service.PlacementStrategy = string(strategyJSON)
		updated = true
	}

	if req.ServiceConnectConfiguration != nil {
		configJSON, _ := json.Marshal(req.ServiceConnectConfiguration)
		service.ServiceConnectConfiguration = string(configJSON)
		updated = true
	}

	if req.VolumeConfigurations != nil {
		// volumesJSON, _ := json.Marshal(req.VolumeConfigurations)
		// Note: VolumeConfigurations is not stored in the Service struct currently
		// This would need to be added to the storage model
		updated = true
	}

	if req.DeploymentConfiguration != nil {
		deploymentJSON, _ := json.Marshal(req.DeploymentConfiguration)
		service.DeploymentConfiguration = string(deploymentJSON)
		updated = true
	}

	if req.PlatformVersion != nil {
		service.PlatformVersion = *req.PlatformVersion
		updated = true
	}

	if req.HealthCheckGracePeriodSeconds != nil {
		service.HealthCheckGracePeriodSeconds = int(*req.HealthCheckGracePeriodSeconds)
		updated = true
	}

	if req.EnableExecuteCommand != nil {
		service.EnableExecuteCommand = *req.EnableExecuteCommand
		updated = true
	}

	if req.EnableECSManagedTags != nil {
		service.EnableECSManagedTags = *req.EnableECSManagedTags
		updated = true
	}

	if req.PropagateTags != "" {
		service.PropagateTags = string(req.PropagateTags)
		updated = true
	}

	if updated {
		service.UpdatedAt = time.Now()
		if err := api.storage.ServiceStore().Update(ctx, service); err != nil {
			return nil, fmt.Errorf("failed to update service: %w", err)
		}
	}

	// Build response with updated service details
	return api.describeServiceForResponse(ctx, service)
}

// DeleteServiceV2 implements the DeleteService operation using AWS SDK types
func (api *DefaultECSAPIV2) DeleteServiceV2(ctx context.Context, req *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Validate required fields
	if req.Service == nil {
		return nil, fmt.Errorf("service is required")
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Extract service name from ARN if necessary
	serviceName := extractServiceNameFromARN(*req.Service)

	// Get service from storage
	service, err := api.storage.ServiceStore().Get(ctx, cluster.ARN, serviceName)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", *req.Service)
	}

	// Force delete if requested
	if req.Force != nil && *req.Force {
		// Stop all tasks immediately
		service.DesiredCount = 0
		service.Status = "DRAINING"
	} else {
		// Check if service can be deleted
		if service.DesiredCount > 0 {
			return nil, fmt.Errorf("service has desired count > 0, set desiredCount to 0 or use force=true")
		}
		if service.RunningCount > 0 {
			return nil, fmt.Errorf("service has running tasks")
		}
	}

	// Update service status
	service.Status = "DRAINING"
	service.UpdatedAt = time.Now()
	if err := api.storage.ServiceStore().Update(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to update service status: %w", err)
	}

	// Delete service from storage
	if err := api.storage.ServiceStore().Delete(ctx, cluster.ARN, service.ServiceName); err != nil {
		return nil, fmt.Errorf("failed to delete service: %w", err)
	}

	// Update cluster service count
	cluster.ActiveServicesCount--
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		log.Printf("Failed to update cluster service count: %v", err)
	}

	// Stop tasks if Kubernetes manager is available
	if api.kindManager != nil {
		go api.stopServiceTasks(ctx, cluster, service)
	}

	// Build response with deleted service info
	service.Status = "INACTIVE"
	resp, err := api.describeServiceForResponse(ctx, service)
	if err != nil {
		return nil, err
	}
	
	// Convert UpdateServiceOutput to DeleteServiceOutput
	return &ecs.DeleteServiceOutput{
		Service: resp.Service,
	}, nil
}

// Helper function to build service response
func (api *DefaultECSAPIV2) describeServiceForResponse(ctx context.Context, service *storage.Service) (*ecs.UpdateServiceOutput, error) {
	// Parse JSON fields
	var loadBalancers []types.LoadBalancer
	if service.LoadBalancers != "" {
		json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers)
	}

	var serviceRegistries []types.ServiceRegistry
	if service.ServiceRegistries != "" {
		json.Unmarshal([]byte(service.ServiceRegistries), &serviceRegistries)
	}

	var networkConfig *types.NetworkConfiguration
	if service.NetworkConfiguration != "" {
		json.Unmarshal([]byte(service.NetworkConfiguration), &networkConfig)
	}

	var deploymentConfig *types.DeploymentConfiguration
	if service.DeploymentConfiguration != "" {
		json.Unmarshal([]byte(service.DeploymentConfiguration), &deploymentConfig)
	}

	var placementConstraints []types.PlacementConstraint
	if service.PlacementConstraints != "" {
		json.Unmarshal([]byte(service.PlacementConstraints), &placementConstraints)
	}

	var placementStrategy []types.PlacementStrategy
	if service.PlacementStrategy != "" {
		json.Unmarshal([]byte(service.PlacementStrategy), &placementStrategy)
	}

	var serviceConnectConfig *types.ServiceConnectConfiguration
	if service.ServiceConnectConfiguration != "" {
		json.Unmarshal([]byte(service.ServiceConnectConfiguration), &serviceConnectConfig)
	}

	// VolumeConfigurations not currently stored  
	var volumeConfigs []types.ServiceVolumeConfiguration

	var capacityProviderStrategy []types.CapacityProviderStrategyItem
	if service.CapacityProviderStrategy != "" {
		json.Unmarshal([]byte(service.CapacityProviderStrategy), &capacityProviderStrategy)
	}

	var tags []types.Tag
	if service.Tags != "" {
		json.Unmarshal([]byte(service.Tags), &tags)
	}

	// Create deployment
	deployment := types.Deployment{
		Id:                       aws.String(fmt.Sprintf("ecs-svc/%d", service.UpdatedAt.Unix())),
		Status:                   aws.String("PRIMARY"),
		TaskDefinition:           aws.String(service.TaskDefinitionARN),
		DesiredCount:             int32(service.DesiredCount),
		PendingCount:             int32(service.PendingCount),
		RunningCount:             int32(service.RunningCount),
		FailedTasks:              0,
		CreatedAt:                aws.Time(service.CreatedAt),
		UpdatedAt:                aws.Time(service.UpdatedAt),
		LaunchType:               types.LaunchType(service.LaunchType),
		PlatformVersion:          aws.String(service.PlatformVersion),
		// PlatformFamily not stored in service
		NetworkConfiguration:     networkConfig,
		RolloutState:             types.DeploymentRolloutStateCompleted,
		RolloutStateReason:       aws.String("ECS deployment completed"),
		ServiceConnectConfiguration: serviceConnectConfig,
		VolumeConfigurations:     volumeConfigs,
		CapacityProviderStrategy: capacityProviderStrategy,
	}

	return &ecs.UpdateServiceOutput{
		Service: &types.Service{
			ServiceArn:               aws.String(service.ARN),
			ServiceName:              aws.String(service.ServiceName),
			ClusterArn:               aws.String(service.ClusterARN),
			TaskDefinition:           aws.String(service.TaskDefinitionARN),
			DesiredCount:             int32(service.DesiredCount),
			RunningCount:             int32(service.RunningCount),
			PendingCount:             int32(service.PendingCount),
			LaunchType:               types.LaunchType(service.LaunchType),
			SchedulingStrategy:       types.SchedulingStrategy(service.SchedulingStrategy),
			Status:                   aws.String(service.Status),
			LoadBalancers:            loadBalancers,
			ServiceRegistries:        serviceRegistries,
			NetworkConfiguration:     networkConfig,
			DeploymentConfiguration:  deploymentConfig,
			PlacementConstraints:     placementConstraints,
			PlacementStrategy:        placementStrategy,
			Deployments:              []types.Deployment{deployment},
			RoleArn:                  aws.String(service.RoleARN),
			CreatedAt:                aws.Time(service.CreatedAt),
			CreatedBy:                aws.String("kecs"),
			HealthCheckGracePeriodSeconds: aws.Int32(int32(service.HealthCheckGracePeriodSeconds)),
			EnableECSManagedTags:     service.EnableECSManagedTags,
			EnableExecuteCommand:     service.EnableExecuteCommand,
			PropagateTags:            types.PropagateTags(service.PropagateTags),
			Tags:                     tags,
			CapacityProviderStrategy: capacityProviderStrategy,
		},
	}, nil
}

// createServiceTasks creates tasks for a service
func (api *DefaultECSAPIV2) createServiceTasks(ctx context.Context, cluster *storage.Cluster, service *storage.Service, taskDef *storage.TaskDefinition, count int) {
	// Implementation would create tasks in Kubernetes
	// This is a placeholder for the actual implementation
	log.Printf("Would create %d tasks for service %s", count, service.ServiceName)
}

// stopServiceTasks stops all tasks for a service
func (api *DefaultECSAPIV2) stopServiceTasks(ctx context.Context, cluster *storage.Cluster, service *storage.Service) {
	// Implementation would stop tasks in Kubernetes
	// This is a placeholder for the actual implementation
	log.Printf("Would stop tasks for service %s", service.ServiceName)
}

// extractServiceNameFromARN extracts service name from ARN or returns the input if it's not an ARN
func extractServiceNameFromARN(identifier string) string {
	if strings.HasPrefix(identifier, "arn:aws:ecs:") {
		parts := strings.Split(identifier, "/")
		if len(parts) >= 3 {
			return parts[len(parts)-1]
		}
	}
	return identifier
}