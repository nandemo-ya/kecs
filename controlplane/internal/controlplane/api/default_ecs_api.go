package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPI provides the default implementation of ECS API operations
type DefaultECSAPI struct {
	storage     storage.Storage
	kindManager *kubernetes.KindManager
	region      string
	accountID   string
}

// NewDefaultECSAPI creates a new default ECS API implementation with storage and kubernetes manager
func NewDefaultECSAPI(storage storage.Storage, kindManager *kubernetes.KindManager) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		storage:     storage,
		kindManager: kindManager,
		region:      "ap-northeast-1", // Default region
		accountID:   "123456789012",   // Default account ID
	}
}

// CreateCluster implements the CreateCluster operation
func (api *DefaultECSAPI) CreateCluster(ctx context.Context, req *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
	log.Printf("Creating cluster: %v", req)

	// Default cluster name if not provided
	clusterName := "default"
	if req.ClusterName != nil {
		clusterName = *req.ClusterName
	}

	// Check if cluster already exists
	existing, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err == nil && existing != nil {
		// Ensure the kind cluster exists (it might have been deleted manually)
		go api.ensureKindClusterExists(existing)
		
		// Return existing cluster
		cluster := &generated.Cluster{
			ClusterArn:  ptr.String(existing.ARN),
			ClusterName: ptr.String(existing.Name),
			Status:      ptr.String(existing.Status),
		}
		
		// Parse settings, configuration, and tags
		if existing.Settings != "" {
			var settings []generated.ClusterSetting
			if err := json.Unmarshal([]byte(existing.Settings), &settings); err == nil {
				cluster.Settings = settings
			}
		}
		if existing.Configuration != "" {
			var config generated.ClusterConfiguration
			if err := json.Unmarshal([]byte(existing.Configuration), &config); err == nil {
				cluster.Configuration = &config
			}
		}
		if existing.Tags != "" {
			var tags []generated.Tag
			if err := json.Unmarshal([]byte(existing.Tags), &tags); err == nil {
				cluster.Tags = tags
			}
		}

		return &generated.CreateClusterResponse{
			Cluster: cluster,
		}, nil
	}

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, clusterName)

	// Generate a deterministic kind cluster name based on ECS cluster name
	kindClusterName := fmt.Sprintf("kecs-%s", clusterName)

	// Create cluster object
	cluster := &storage.Cluster{
		ID:              uuid.New().String(),
		ARN:             arn,
		Name:            clusterName,
		Status:          "ACTIVE",
		Region:          api.region,
		AccountID:       api.accountID,
		KindClusterName: kindClusterName,
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:                 0,
		PendingTasksCount:                 0,
		ActiveServicesCount:               0,
	}

	// Extract settings and configuration from request
	if req.Settings != nil && len(req.Settings) > 0 {
		settingsJSON, _ := json.Marshal(req.Settings)
		cluster.Settings = string(settingsJSON)
	}
	if req.Configuration != nil {
		configJSON, _ := json.Marshal(req.Configuration)
		cluster.Configuration = string(configJSON)
	}
	if req.Tags != nil && len(req.Tags) > 0 {
		tagsJSON, _ := json.Marshal(req.Tags)
		cluster.Tags = string(tagsJSON)
	}

	// Save to storage
	if err := api.storage.ClusterStore().Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Create kind cluster and namespace asynchronously
	go api.createKindClusterAndNamespace(cluster)

	// Build response
	response := &generated.CreateClusterResponse{
		Cluster: &generated.Cluster{
			ClusterArn:  ptr.String(cluster.ARN),
			ClusterName: ptr.String(cluster.Name),
			Status:      ptr.String(cluster.Status),
			Settings:    req.Settings,
			Configuration: req.Configuration,
			Tags:        req.Tags,
		},
	}

	return response, nil
}

// ListClusters implements the ListClusters operation
func (api *DefaultECSAPI) ListClusters(ctx context.Context, req *generated.ListClustersRequest) (*generated.ListClustersResponse, error) {
	// Get all clusters from storage
	clusters, err := api.storage.ClusterStore().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Build cluster ARNs list
	clusterArns := make([]string, 0, len(clusters))
	for _, cluster := range clusters {
		clusterArns = append(clusterArns, cluster.ARN)
	}

	response := &generated.ListClustersResponse{
		ClusterArns: clusterArns,
	}

	// Handle pagination if requested
	// TODO: Implement proper pagination

	return response, nil
}

// DescribeClusters implements the DescribeClusters operation
func (api *DefaultECSAPI) DescribeClusters(ctx context.Context, req *generated.DescribeClustersRequest) (*generated.DescribeClustersResponse, error) {
	// If no clusters specified, describe all clusters
	clusterIdentifiers := req.Clusters
	if len(clusterIdentifiers) == 0 {
		clusters, err := api.storage.ClusterStore().List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}
		for _, cluster := range clusters {
			clusterIdentifiers = append(clusterIdentifiers, cluster.Name)
		}
	}

	// Fetch details for each cluster
	var describedClusters []generated.Cluster
	var failures []generated.Failure

	for _, identifier := range clusterIdentifiers {
		cluster, err := api.storage.ClusterStore().Get(ctx, identifier)
		// Storage only supports lookup by name currently
		// TODO: Add ARN lookup support
		
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(identifier),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String(fmt.Sprintf("Could not find cluster %s", identifier)),
			})
			continue
		}

		// Build cluster response
		clusterResp := generated.Cluster{
			ClusterArn:                        ptr.String(cluster.ARN),
			ClusterName:                       ptr.String(cluster.Name),
			Status:                           ptr.String(cluster.Status),
			RegisteredContainerInstancesCount: ptr.Int32(int32(cluster.RegisteredContainerInstancesCount)),
			RunningTasksCount:                ptr.Int32(int32(cluster.RunningTasksCount)),
			PendingTasksCount:                ptr.Int32(int32(cluster.PendingTasksCount)),
			ActiveServicesCount:              ptr.Int32(int32(cluster.ActiveServicesCount)),
		}

		// Add settings if requested
		if req.Include != nil {
			for _, include := range req.Include {
				switch include {
				case generated.ClusterFieldSettings:
					if cluster.Settings != "" {
						var settings []generated.ClusterSetting
						if err := json.Unmarshal([]byte(cluster.Settings), &settings); err == nil {
							clusterResp.Settings = settings
						}
					}
				case generated.ClusterFieldConfigurations:
					if cluster.Configuration != "" {
						var config generated.ClusterConfiguration
						if err := json.Unmarshal([]byte(cluster.Configuration), &config); err == nil {
							clusterResp.Configuration = &config
						}
					}
				case generated.ClusterFieldTags:
					if cluster.Tags != "" {
						var tags []generated.Tag
						if err := json.Unmarshal([]byte(cluster.Tags), &tags); err == nil {
							clusterResp.Tags = tags
						}
					}
				}
			}
		}

		describedClusters = append(describedClusters, clusterResp)
	}

	return &generated.DescribeClustersResponse{
		Clusters: describedClusters,
		Failures: failures,
	}, nil
}

// DeleteCluster implements the DeleteCluster operation
func (api *DefaultECSAPI) DeleteCluster(ctx context.Context, req *generated.DeleteClusterRequest) (*generated.DeleteClusterResponse, error) {
	if req.Cluster == nil {
		return nil, fmt.Errorf("cluster identifier is required")
	}

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, *req.Cluster)
	// TODO: Add ARN lookup support
	
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", *req.Cluster)
	}

	// Check if cluster has active resources
	if cluster.ActiveServicesCount > 0 || cluster.RunningTasksCount > 0 {
		return nil, fmt.Errorf("cluster has active services or tasks")
	}

	// Update status to INACTIVE
	cluster.Status = "INACTIVE"
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to update cluster status: %w", err)
	}

	// Delete the cluster
	if err := api.storage.ClusterStore().Delete(ctx, cluster.Name); err != nil {
		return nil, fmt.Errorf("failed to delete cluster: %w", err)
	}

	// Delete kind cluster and namespace asynchronously
	go api.deleteKindClusterAndNamespace(cluster)

	// Build response with the deleted cluster info
	response := &generated.DeleteClusterResponse{
		Cluster: &generated.Cluster{
			ClusterArn:  ptr.String(cluster.ARN),
			ClusterName: ptr.String(cluster.Name),
			Status:      ptr.String("INACTIVE"),
		},
	}

	return response, nil
}

// CreateCapacityProvider implements the CreateCapacityProvider operation
func (api *DefaultECSAPI) CreateCapacityProvider(ctx context.Context, req *generated.CreateCapacityProviderRequest) (*generated.CreateCapacityProviderResponse, error) {
	// TODO: Implement CreateCapacityProvider
	return nil, fmt.Errorf("CreateCapacityProvider not implemented")
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

// CreateTaskSet implements the CreateTaskSet operation
func (api *DefaultECSAPI) CreateTaskSet(ctx context.Context, req *generated.CreateTaskSetRequest) (*generated.CreateTaskSetResponse, error) {
	// TODO: Implement CreateTaskSet
	return nil, fmt.Errorf("CreateTaskSet not implemented")
}

// DeleteAccountSetting implements the DeleteAccountSetting operation
func (api *DefaultECSAPI) DeleteAccountSetting(ctx context.Context, req *generated.DeleteAccountSettingRequest) (*generated.DeleteAccountSettingResponse, error) {
	// TODO: Implement DeleteAccountSetting
	return nil, fmt.Errorf("DeleteAccountSetting not implemented")
}

// DeleteAttributes implements the DeleteAttributes operation
func (api *DefaultECSAPI) DeleteAttributes(ctx context.Context, req *generated.DeleteAttributesRequest) (*generated.DeleteAttributesResponse, error) {
	// TODO: Implement DeleteAttributes
	return nil, fmt.Errorf("DeleteAttributes not implemented")
}

// DeleteCapacityProvider implements the DeleteCapacityProvider operation
func (api *DefaultECSAPI) DeleteCapacityProvider(ctx context.Context, req *generated.DeleteCapacityProviderRequest) (*generated.DeleteCapacityProviderResponse, error) {
	// TODO: Implement DeleteCapacityProvider
	return nil, fmt.Errorf("DeleteCapacityProvider not implemented")
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

// DeleteTaskDefinitions implements the DeleteTaskDefinitions operation
func (api *DefaultECSAPI) DeleteTaskDefinitions(ctx context.Context, req *generated.DeleteTaskDefinitionsRequest) (*generated.DeleteTaskDefinitionsResponse, error) {
	// TODO: Implement DeleteTaskDefinitions
	return nil, fmt.Errorf("DeleteTaskDefinitions not implemented")
}

// DeleteTaskSet implements the DeleteTaskSet operation
func (api *DefaultECSAPI) DeleteTaskSet(ctx context.Context, req *generated.DeleteTaskSetRequest) (*generated.DeleteTaskSetResponse, error) {
	// TODO: Implement DeleteTaskSet
	return nil, fmt.Errorf("DeleteTaskSet not implemented")
}

// DeregisterContainerInstance implements the DeregisterContainerInstance operation
func (api *DefaultECSAPI) DeregisterContainerInstance(ctx context.Context, req *generated.DeregisterContainerInstanceRequest) (*generated.DeregisterContainerInstanceResponse, error) {
	// TODO: Implement DeregisterContainerInstance
	return nil, fmt.Errorf("DeregisterContainerInstance not implemented")
}

// DeregisterTaskDefinition implements the DeregisterTaskDefinition operation
func (api *DefaultECSAPI) DeregisterTaskDefinition(ctx context.Context, req *generated.DeregisterTaskDefinitionRequest) (*generated.DeregisterTaskDefinitionResponse, error) {
	// TODO: Implement DeregisterTaskDefinition
	return nil, fmt.Errorf("DeregisterTaskDefinition not implemented")
}

// DescribeCapacityProviders implements the DescribeCapacityProviders operation
func (api *DefaultECSAPI) DescribeCapacityProviders(ctx context.Context, req *generated.DescribeCapacityProvidersRequest) (*generated.DescribeCapacityProvidersResponse, error) {
	// TODO: Implement DescribeCapacityProviders
	return nil, fmt.Errorf("DescribeCapacityProviders not implemented")
}

// DescribeContainerInstances implements the DescribeContainerInstances operation
func (api *DefaultECSAPI) DescribeContainerInstances(ctx context.Context, req *generated.DescribeContainerInstancesRequest) (*generated.DescribeContainerInstancesResponse, error) {
	// TODO: Implement DescribeContainerInstances
	return nil, fmt.Errorf("DescribeContainerInstances not implemented")
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

// DescribeTaskDefinition implements the DescribeTaskDefinition operation
func (api *DefaultECSAPI) DescribeTaskDefinition(ctx context.Context, req *generated.DescribeTaskDefinitionRequest) (*generated.DescribeTaskDefinitionResponse, error) {
	// TODO: Implement DescribeTaskDefinition
	return nil, fmt.Errorf("DescribeTaskDefinition not implemented")
}

// DescribeTaskSets implements the DescribeTaskSets operation
func (api *DefaultECSAPI) DescribeTaskSets(ctx context.Context, req *generated.DescribeTaskSetsRequest) (*generated.DescribeTaskSetsResponse, error) {
	// TODO: Implement DescribeTaskSets
	return nil, fmt.Errorf("DescribeTaskSets not implemented")
}

// DescribeTasks implements the DescribeTasks operation
func (api *DefaultECSAPI) DescribeTasks(ctx context.Context, req *generated.DescribeTasksRequest) (*generated.DescribeTasksResponse, error) {
	// TODO: Implement DescribeTasks
	return nil, fmt.Errorf("DescribeTasks not implemented")
}

// DiscoverPollEndpoint implements the DiscoverPollEndpoint operation
func (api *DefaultECSAPI) DiscoverPollEndpoint(ctx context.Context, req *generated.DiscoverPollEndpointRequest) (*generated.DiscoverPollEndpointResponse, error) {
	// TODO: Implement DiscoverPollEndpoint
	return nil, fmt.Errorf("DiscoverPollEndpoint not implemented")
}

// ExecuteCommand implements the ExecuteCommand operation
func (api *DefaultECSAPI) ExecuteCommand(ctx context.Context, req *generated.ExecuteCommandRequest) (*generated.ExecuteCommandResponse, error) {
	// TODO: Implement ExecuteCommand
	return nil, fmt.Errorf("ExecuteCommand not implemented")
}

// GetTaskProtection implements the GetTaskProtection operation
func (api *DefaultECSAPI) GetTaskProtection(ctx context.Context, req *generated.GetTaskProtectionRequest) (*generated.GetTaskProtectionResponse, error) {
	// TODO: Implement GetTaskProtection
	return nil, fmt.Errorf("GetTaskProtection not implemented")
}

// ListAccountSettings implements the ListAccountSettings operation
func (api *DefaultECSAPI) ListAccountSettings(ctx context.Context, req *generated.ListAccountSettingsRequest) (*generated.ListAccountSettingsResponse, error) {
	// TODO: Implement ListAccountSettings
	return nil, fmt.Errorf("ListAccountSettings not implemented")
}

// ListAttributes implements the ListAttributes operation
func (api *DefaultECSAPI) ListAttributes(ctx context.Context, req *generated.ListAttributesRequest) (*generated.ListAttributesResponse, error) {
	// TODO: Implement ListAttributes
	return nil, fmt.Errorf("ListAttributes not implemented")
}

// ListContainerInstances implements the ListContainerInstances operation
func (api *DefaultECSAPI) ListContainerInstances(ctx context.Context, req *generated.ListContainerInstancesRequest) (*generated.ListContainerInstancesResponse, error) {
	// TODO: Implement ListContainerInstances
	return nil, fmt.Errorf("ListContainerInstances not implemented")
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

// ListTagsForResource implements the ListTagsForResource operation
func (api *DefaultECSAPI) ListTagsForResource(ctx context.Context, req *generated.ListTagsForResourceRequest) (*generated.ListTagsForResourceResponse, error) {
	// TODO: Implement ListTagsForResource
	return nil, fmt.Errorf("ListTagsForResource not implemented")
}

// ListTaskDefinitionFamilies implements the ListTaskDefinitionFamilies operation
func (api *DefaultECSAPI) ListTaskDefinitionFamilies(ctx context.Context, req *generated.ListTaskDefinitionFamiliesRequest) (*generated.ListTaskDefinitionFamiliesResponse, error) {
	// TODO: Implement ListTaskDefinitionFamilies
	return nil, fmt.Errorf("ListTaskDefinitionFamilies not implemented")
}

// ListTaskDefinitions implements the ListTaskDefinitions operation
func (api *DefaultECSAPI) ListTaskDefinitions(ctx context.Context, req *generated.ListTaskDefinitionsRequest) (*generated.ListTaskDefinitionsResponse, error) {
	// TODO: Implement ListTaskDefinitions
	return nil, fmt.Errorf("ListTaskDefinitions not implemented")
}

// ListTasks implements the ListTasks operation
func (api *DefaultECSAPI) ListTasks(ctx context.Context, req *generated.ListTasksRequest) (*generated.ListTasksResponse, error) {
	// TODO: Implement ListTasks
	return nil, fmt.Errorf("ListTasks not implemented")
}

// PutAccountSetting implements the PutAccountSetting operation
func (api *DefaultECSAPI) PutAccountSetting(ctx context.Context, req *generated.PutAccountSettingRequest) (*generated.PutAccountSettingResponse, error) {
	// TODO: Implement PutAccountSetting
	return nil, fmt.Errorf("PutAccountSetting not implemented")
}

// PutAccountSettingDefault implements the PutAccountSettingDefault operation
func (api *DefaultECSAPI) PutAccountSettingDefault(ctx context.Context, req *generated.PutAccountSettingDefaultRequest) (*generated.PutAccountSettingDefaultResponse, error) {
	// TODO: Implement PutAccountSettingDefault
	return nil, fmt.Errorf("PutAccountSettingDefault not implemented")
}

// PutAttributes implements the PutAttributes operation
func (api *DefaultECSAPI) PutAttributes(ctx context.Context, req *generated.PutAttributesRequest) (*generated.PutAttributesResponse, error) {
	// TODO: Implement PutAttributes
	return nil, fmt.Errorf("PutAttributes not implemented")
}

// PutClusterCapacityProviders implements the PutClusterCapacityProviders operation
func (api *DefaultECSAPI) PutClusterCapacityProviders(ctx context.Context, req *generated.PutClusterCapacityProvidersRequest) (*generated.PutClusterCapacityProvidersResponse, error) {
	// TODO: Implement PutClusterCapacityProviders
	return nil, fmt.Errorf("PutClusterCapacityProviders not implemented")
}

// RegisterContainerInstance implements the RegisterContainerInstance operation
func (api *DefaultECSAPI) RegisterContainerInstance(ctx context.Context, req *generated.RegisterContainerInstanceRequest) (*generated.RegisterContainerInstanceResponse, error) {
	// TODO: Implement RegisterContainerInstance
	return nil, fmt.Errorf("RegisterContainerInstance not implemented")
}

// RegisterTaskDefinition implements the RegisterTaskDefinition operation
func (api *DefaultECSAPI) RegisterTaskDefinition(ctx context.Context, req *generated.RegisterTaskDefinitionRequest) (*generated.RegisterTaskDefinitionResponse, error) {
	// TODO: Implement RegisterTaskDefinition
	return nil, fmt.Errorf("RegisterTaskDefinition not implemented")
}

// RunTask implements the RunTask operation
func (api *DefaultECSAPI) RunTask(ctx context.Context, req *generated.RunTaskRequest) (*generated.RunTaskResponse, error) {
	// TODO: Implement RunTask
	return nil, fmt.Errorf("RunTask not implemented")
}

// StartTask implements the StartTask operation
func (api *DefaultECSAPI) StartTask(ctx context.Context, req *generated.StartTaskRequest) (*generated.StartTaskResponse, error) {
	// TODO: Implement StartTask
	return nil, fmt.Errorf("StartTask not implemented")
}

// StopTask implements the StopTask operation
func (api *DefaultECSAPI) StopTask(ctx context.Context, req *generated.StopTaskRequest) (*generated.StopTaskResponse, error) {
	// TODO: Implement StopTask
	return nil, fmt.Errorf("StopTask not implemented")
}

// SubmitAttachmentStateChanges implements the SubmitAttachmentStateChanges operation
func (api *DefaultECSAPI) SubmitAttachmentStateChanges(ctx context.Context, req *generated.SubmitAttachmentStateChangesRequest) (*generated.SubmitAttachmentStateChangesResponse, error) {
	// TODO: Implement SubmitAttachmentStateChanges
	return nil, fmt.Errorf("SubmitAttachmentStateChanges not implemented")
}

// SubmitContainerStateChange implements the SubmitContainerStateChange operation
func (api *DefaultECSAPI) SubmitContainerStateChange(ctx context.Context, req *generated.SubmitContainerStateChangeRequest) (*generated.SubmitContainerStateChangeResponse, error) {
	// TODO: Implement SubmitContainerStateChange
	return nil, fmt.Errorf("SubmitContainerStateChange not implemented")
}

// SubmitTaskStateChange implements the SubmitTaskStateChange operation
func (api *DefaultECSAPI) SubmitTaskStateChange(ctx context.Context, req *generated.SubmitTaskStateChangeRequest) (*generated.SubmitTaskStateChangeResponse, error) {
	// TODO: Implement SubmitTaskStateChange
	return nil, fmt.Errorf("SubmitTaskStateChange not implemented")
}

// TagResource implements the TagResource operation
func (api *DefaultECSAPI) TagResource(ctx context.Context, req *generated.TagResourceRequest) (*generated.TagResourceResponse, error) {
	// TODO: Implement TagResource
	return nil, fmt.Errorf("TagResource not implemented")
}

// UntagResource implements the UntagResource operation
func (api *DefaultECSAPI) UntagResource(ctx context.Context, req *generated.UntagResourceRequest) (*generated.UntagResourceResponse, error) {
	// TODO: Implement UntagResource
	return nil, fmt.Errorf("UntagResource not implemented")
}

// UpdateCapacityProvider implements the UpdateCapacityProvider operation
func (api *DefaultECSAPI) UpdateCapacityProvider(ctx context.Context, req *generated.UpdateCapacityProviderRequest) (*generated.UpdateCapacityProviderResponse, error) {
	// TODO: Implement UpdateCapacityProvider
	return nil, fmt.Errorf("UpdateCapacityProvider not implemented")
}

// UpdateCluster implements the UpdateCluster operation
func (api *DefaultECSAPI) UpdateCluster(ctx context.Context, req *generated.UpdateClusterRequest) (*generated.UpdateClusterResponse, error) {
	// TODO: Implement UpdateCluster
	return nil, fmt.Errorf("UpdateCluster not implemented")
}

// UpdateClusterSettings implements the UpdateClusterSettings operation
func (api *DefaultECSAPI) UpdateClusterSettings(ctx context.Context, req *generated.UpdateClusterSettingsRequest) (*generated.UpdateClusterSettingsResponse, error) {
	// TODO: Implement UpdateClusterSettings
	return nil, fmt.Errorf("UpdateClusterSettings not implemented")
}

// UpdateContainerAgent implements the UpdateContainerAgent operation
func (api *DefaultECSAPI) UpdateContainerAgent(ctx context.Context, req *generated.UpdateContainerAgentRequest) (*generated.UpdateContainerAgentResponse, error) {
	// TODO: Implement UpdateContainerAgent
	return nil, fmt.Errorf("UpdateContainerAgent not implemented")
}

// UpdateContainerInstancesState implements the UpdateContainerInstancesState operation
func (api *DefaultECSAPI) UpdateContainerInstancesState(ctx context.Context, req *generated.UpdateContainerInstancesStateRequest) (*generated.UpdateContainerInstancesStateResponse, error) {
	// TODO: Implement UpdateContainerInstancesState
	return nil, fmt.Errorf("UpdateContainerInstancesState not implemented")
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

// UpdateTaskProtection implements the UpdateTaskProtection operation
func (api *DefaultECSAPI) UpdateTaskProtection(ctx context.Context, req *generated.UpdateTaskProtectionRequest) (*generated.UpdateTaskProtectionResponse, error) {
	// TODO: Implement UpdateTaskProtection
	return nil, fmt.Errorf("UpdateTaskProtection not implemented")
}

// UpdateTaskSet implements the UpdateTaskSet operation
func (api *DefaultECSAPI) UpdateTaskSet(ctx context.Context, req *generated.UpdateTaskSetRequest) (*generated.UpdateTaskSetResponse, error) {
	// TODO: Implement UpdateTaskSet
	return nil, fmt.Errorf("UpdateTaskSet not implemented")
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

// createKindClusterAndNamespace creates a Kind cluster and namespace for the ECS cluster
func (api *DefaultECSAPI) createKindClusterAndNamespace(cluster *storage.Cluster) {
	ctx := context.Background()
	
	// Skip kind cluster creation if kindManager is nil (test mode)
	if api.kindManager == nil {
		log.Printf("Skipping kind cluster creation for %s (kindManager is nil)", cluster.Name)
		return
	}
	
	// Check if kind cluster already exists
	if _, err := api.kindManager.GetKubeClient(cluster.KindClusterName); err != nil {
		// Cluster doesn't exist, create it
		log.Printf("Kind cluster %s doesn't exist, creating...", cluster.KindClusterName)
		if err := api.kindManager.CreateCluster(ctx, cluster.KindClusterName); err != nil {
			log.Printf("Failed to create kind cluster %s for ECS cluster %s: %v", cluster.KindClusterName, cluster.Name, err)
			return
		}
	} else {
		log.Printf("Reusing existing kind cluster %s for ECS cluster %s", cluster.KindClusterName, cluster.Name)
	}
	
	// Get Kubernetes client
	kubeClient, err := api.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		log.Printf("Failed to get kubernetes client for %s: %v", cluster.KindClusterName, err)
		return
	}
	
	// Create namespace
	namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
	if err := namespaceManager.CreateNamespace(ctx, cluster.Name, cluster.Region); err != nil {
		log.Printf("Failed to create namespace for %s: %v", cluster.Name, err)
		return
	}
	
	log.Printf("Successfully created kind cluster %s and namespace for ECS cluster %s", cluster.KindClusterName, cluster.Name)
}

// ensureKindClusterExists ensures that a Kind cluster exists for an existing ECS cluster
func (api *DefaultECSAPI) ensureKindClusterExists(cluster *storage.Cluster) {
	ctx := context.Background()
	
	// Skip kind cluster creation if kindManager is nil (test mode)
	if api.kindManager == nil {
		return
	}
	
	// Check if kind cluster exists, create if it doesn't
	if _, err := api.kindManager.GetKubeClient(cluster.KindClusterName); err != nil {
		log.Printf("Kind cluster %s for existing ECS cluster %s is missing, recreating...", cluster.KindClusterName, cluster.Name)
		if err := api.kindManager.CreateCluster(ctx, cluster.KindClusterName); err != nil {
			log.Printf("Failed to recreate kind cluster %s: %v", cluster.KindClusterName, err)
			return
		}
		
		// Get Kubernetes client and create namespace
		kubeClient, err := api.kindManager.GetKubeClient(cluster.KindClusterName)
		if err != nil {
			log.Printf("Failed to get kubernetes client for %s: %v", cluster.KindClusterName, err)
			return
		}
		
		namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
		if err := namespaceManager.CreateNamespace(ctx, cluster.Name, cluster.Region); err != nil {
			log.Printf("Failed to create namespace for %s: %v", cluster.Name, err)
			return
		}
		log.Printf("Successfully recreated kind cluster %s for ECS cluster %s", cluster.KindClusterName, cluster.Name)
	}
}

// deleteKindClusterAndNamespace deletes the Kind cluster and namespace for an ECS cluster
func (api *DefaultECSAPI) deleteKindClusterAndNamespace(cluster *storage.Cluster) {
	ctx := context.Background()
	
	// Skip kind cluster deletion if kindManager is nil (test mode)
	if api.kindManager == nil {
		log.Printf("Skipping kind cluster deletion for %s (kindManager is nil)", cluster.Name)
		return
	}
	
	// Get Kubernetes client before deleting cluster
	kubeClient, err := api.kindManager.GetKubeClient(cluster.KindClusterName)
	if err == nil {
		// Delete namespace
		namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
		if err := namespaceManager.DeleteNamespace(ctx, cluster.Name, cluster.Region); err != nil {
			log.Printf("Failed to delete namespace for %s: %v", cluster.Name, err)
		}
	}
	
	// Delete kind cluster
	if err := api.kindManager.DeleteCluster(ctx, cluster.KindClusterName); err != nil {
		log.Printf("Failed to delete kind cluster %s for ECS cluster %s: %v", cluster.KindClusterName, cluster.Name, err)
		return
	}
	
	log.Printf("Successfully deleted kind cluster %s and namespace for ECS cluster %s", cluster.KindClusterName, cluster.Name)
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
