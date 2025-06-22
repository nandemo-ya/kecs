package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CreateCluster implements the CreateCluster operation
func (api *DefaultECSAPI) CreateCluster(ctx context.Context, req *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
	log.Printf("Creating cluster: %v", req)

	// Default cluster name if not provided
	clusterName := "default"
	if req.ClusterName != nil {
		clusterName = *req.ClusterName
	}

	// Validate cluster name
	if err := ValidateClusterName(clusterName); err != nil {
		return nil, err
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

		// Parse settings, configuration, and tags with caching
		if settings, err := parseClusterSettings(existing.Name, existing.Settings); err == nil {
			cluster.Settings = settings
		}
		if config, err := parseClusterConfiguration(existing.Name, existing.Configuration); err == nil {
			cluster.Configuration = config
		}
		if tags, err := parseTags(fmt.Sprintf("cluster:%s", existing.Name), existing.Tags); err == nil {
			cluster.Tags = tags
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
		ID:                                uuid.New().String(),
		ARN:                               arn,
		Name:                              clusterName,
		Status:                            "ACTIVE",
		Region:                            api.region,
		AccountID:                         api.accountID,
		KindClusterName:                   kindClusterName,
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:                 0,
		PendingTasksCount:                 0,
		ActiveServicesCount:               0,
	}

	// Extract settings and configuration from request
	if req.Settings != nil && len(req.Settings) > 0 {
		// Validate settings
		if err := ValidateClusterSettings(req.Settings); err != nil {
			return nil, err
		}
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster settings: %w", err)
		}
		cluster.Settings = string(settingsJSON)
	}
	if req.Configuration != nil {
		// Validate configuration if execute command config is present
		if req.Configuration.ExecuteCommandConfiguration != nil {
			if err := ValidateExecuteCommandConfiguration(req.Configuration.ExecuteCommandConfiguration); err != nil {
				return nil, err
			}
		}
		configJSON, err := json.Marshal(req.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster configuration: %w", err)
		}
		cluster.Configuration = string(configJSON)
	}
	if req.Tags != nil && len(req.Tags) > 0 {
		tagsJSON, err := json.Marshal(req.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster tags: %w", err)
		}
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
			ClusterArn:    ptr.String(cluster.ARN),
			ClusterName:   ptr.String(cluster.Name),
			Status:        ptr.String(cluster.Status),
			Settings:      req.Settings,
			Configuration: req.Configuration,
			Tags:          req.Tags,
		},
	}

	return response, nil
}

// ListClusters implements the ListClusters operation
func (api *DefaultECSAPI) ListClusters(ctx context.Context, req *generated.ListClustersRequest) (*generated.ListClustersResponse, error) {
	// Debug log the request
	reqJSON, _ := json.Marshal(req)
	log.Printf("ListClusters request: %s", string(reqJSON))

	// Set default limit if not specified
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		// AWS ECS has a maximum of 100 results per page
		if limit > 100 {
			limit = 100
		}
		log.Printf("ListClusters: MaxResults=%d, effective limit=%d", *req.MaxResults, limit)
	} else {
		log.Printf("ListClusters: No MaxResults specified, using default limit=%d", limit)
	}

	// Extract next token
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get clusters with pagination
	clusters, newNextToken, err := api.storage.ClusterStore().ListWithPagination(ctx, limit, nextToken)
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

	// Set next token if there are more results
	if newNextToken != "" {
		response.NextToken = ptr.String(newNextToken)
		log.Printf("ListClusters: Returning %d clusters with nextToken=%s", len(clusterArns), newNextToken)
	} else {
		log.Printf("ListClusters: Returning %d clusters with no nextToken", len(clusterArns))
	}

	return response, nil
}

// DescribeClusters implements the DescribeClusters operation
func (api *DefaultECSAPI) DescribeClusters(ctx context.Context, req *generated.DescribeClustersRequest) (*generated.DescribeClustersResponse, error) {
	// Validate cluster identifiers
	for _, identifier := range req.Clusters {
		if identifier == "" {
			return nil, fmt.Errorf("Invalid parameter: Empty cluster identifier")
		}
		// Validate if it looks like an ARN
		if strings.HasPrefix(identifier, "arn:") {
			if err := ValidateClusterARN(identifier); err != nil {
				return nil, err
			}
		}
	}

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
		// Extract cluster name from ARN if necessary
		clusterName := extractClusterNameFromARN(identifier)

		cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)

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
			Status:                            ptr.String(cluster.Status),
			RegisteredContainerInstancesCount: ptr.Int32(int32(cluster.RegisteredContainerInstancesCount)),
			RunningTasksCount:                 ptr.Int32(int32(cluster.RunningTasksCount)),
			PendingTasksCount:                 ptr.Int32(int32(cluster.PendingTasksCount)),
			ActiveServicesCount:               ptr.Int32(int32(cluster.ActiveServicesCount)),
		}

		// Add settings if requested
		if req.Include != nil {
			for _, include := range req.Include {
				switch include {
				case generated.ClusterFieldSETTINGS:
					if cluster.Settings != "" {
						var settings []generated.ClusterSetting
						if err := json.Unmarshal([]byte(cluster.Settings), &settings); err == nil {
							clusterResp.Settings = settings
						}
					}
				case generated.ClusterFieldCONFIGURATIONS:
					if cluster.Configuration != "" {
						var config generated.ClusterConfiguration
						if err := json.Unmarshal([]byte(cluster.Configuration), &config); err == nil {
							clusterResp.Configuration = &config
						}
					}
				case generated.ClusterFieldTAGS:
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
	if req.Cluster == "" {
		return nil, fmt.Errorf("cluster identifier is required")
	}

	// Validate cluster identifier
	if err := ValidateClusterIdentifier(req.Cluster); err != nil {
		return nil, err
	}

	// Extract cluster name from ARN if necessary
	clusterName := extractClusterNameFromARN(req.Cluster)

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)

	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", req.Cluster)
	}

	// Check if cluster has active resources
	if cluster.ActiveServicesCount > 0 {
		return nil, fmt.Errorf("The cluster cannot be deleted while services are active")
	}
	if cluster.RunningTasksCount > 0 {
		return nil, fmt.Errorf("The cluster cannot be deleted while tasks are active")
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
	
	// Invalidate cache for this cluster
	invalidateClusterCache(cluster.Name)

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

// UpdateCluster implements the UpdateCluster operation
func (api *DefaultECSAPI) UpdateCluster(ctx context.Context, req *generated.UpdateClusterRequest) (*generated.UpdateClusterResponse, error) {
	if req.Cluster == "" {
		return nil, fmt.Errorf("cluster identifier is required")
	}

	// Validate cluster identifier
	if err := ValidateClusterIdentifier(req.Cluster); err != nil {
		return nil, err
	}

	// Extract cluster name from ARN if necessary
	clusterName := extractClusterNameFromARN(req.Cluster)

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", req.Cluster)
	}

	// Update settings if provided
	if req.Settings != nil && len(req.Settings) > 0 {
		// Validate settings
		if err := ValidateClusterSettings(req.Settings); err != nil {
			return nil, err
		}
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster settings: %w", err)
		}
		cluster.Settings = string(settingsJSON)
	}

	// Update configuration if provided
	if req.Configuration != nil {
		// Validate configuration if execute command config is present
		if req.Configuration.ExecuteCommandConfiguration != nil {
			if err := ValidateExecuteCommandConfiguration(req.Configuration.ExecuteCommandConfiguration); err != nil {
				return nil, err
			}
		}
		configJSON, err := json.Marshal(req.Configuration)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster configuration: %w", err)
		}
		cluster.Configuration = string(configJSON)
	}

	// Update service connect defaults if provided
	if req.ServiceConnectDefaults != nil {
		// For now, we'll store this in the Configuration field
		// In a real implementation, this might need a separate field
		log.Printf("ServiceConnectDefaults update requested but not fully implemented yet")
	}

	// Update the cluster
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}
	
	// Invalidate cache for this cluster
	invalidateClusterCache(cluster.Name)

	// Build response
	responseCluster := &generated.Cluster{
		ClusterArn:                        ptr.String(cluster.ARN),
		ClusterName:                       ptr.String(cluster.Name),
		Status:                            ptr.String(cluster.Status),
		RegisteredContainerInstancesCount: ptr.Int32(int32(cluster.RegisteredContainerInstancesCount)),
		RunningTasksCount:                 ptr.Int32(int32(cluster.RunningTasksCount)),
		PendingTasksCount:                 ptr.Int32(int32(cluster.PendingTasksCount)),
		ActiveServicesCount:               ptr.Int32(int32(cluster.ActiveServicesCount)),
	}

	// Add settings if present
	if cluster.Settings != "" {
		var settings []generated.ClusterSetting
		if err := json.Unmarshal([]byte(cluster.Settings), &settings); err == nil {
			responseCluster.Settings = settings
		}
	}

	// Add configuration if present
	if cluster.Configuration != "" {
		var config generated.ClusterConfiguration
		if err := json.Unmarshal([]byte(cluster.Configuration), &config); err == nil {
			responseCluster.Configuration = &config
		}
	}

	// Add tags if present
	if cluster.Tags != "" {
		var tags []generated.Tag
		if err := json.Unmarshal([]byte(cluster.Tags), &tags); err == nil {
			responseCluster.Tags = tags
		}
	}

	return &generated.UpdateClusterResponse{
		Cluster: responseCluster,
	}, nil
}

// UpdateClusterSettings implements the UpdateClusterSettings operation
func (api *DefaultECSAPI) UpdateClusterSettings(ctx context.Context, req *generated.UpdateClusterSettingsRequest) (*generated.UpdateClusterSettingsResponse, error) {
	if req.Cluster == "" {
		return nil, fmt.Errorf("cluster identifier is required")
	}
	if req.Settings == nil || len(req.Settings) == 0 {
		return nil, fmt.Errorf("settings are required")
	}

	// Validate cluster identifier
	if err := ValidateClusterIdentifier(req.Cluster); err != nil {
		return nil, err
	}

	// Validate settings
	if err := ValidateClusterSettings(req.Settings); err != nil {
		return nil, err
	}

	// Extract cluster name from ARN if necessary
	clusterName := extractClusterNameFromARN(req.Cluster)

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", req.Cluster)
	}

	// Parse existing settings
	var existingSettings []generated.ClusterSetting
	if cluster.Settings != "" {
		if err := json.Unmarshal([]byte(cluster.Settings), &existingSettings); err != nil {
			log.Printf("Failed to unmarshal existing settings: %v", err)
			existingSettings = []generated.ClusterSetting{}
		}
	}

	// Create a map for easier updates
	settingsMap := make(map[generated.ClusterSettingName]string)
	for _, setting := range existingSettings {
		if setting.Name != nil && setting.Value != nil {
			settingsMap[*setting.Name] = *setting.Value
		}
	}

	// Update with new settings
	for _, setting := range req.Settings {
		if setting.Name != nil && setting.Value != nil {
			settingsMap[*setting.Name] = *setting.Value
		}
	}

	// Convert back to array
	var updatedSettings []generated.ClusterSetting
	for name, value := range settingsMap {
		settingName := name
		settingValue := value
		updatedSettings = append(updatedSettings, generated.ClusterSetting{
			Name:  &settingName,
			Value: &settingValue,
		})
	}

	// Marshal and store updated settings
	settingsJSON, err := json.Marshal(updatedSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cluster settings: %w", err)
	}
	cluster.Settings = string(settingsJSON)

	// Update the cluster
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}
	
	// Invalidate cache for this cluster
	invalidateClusterCache(cluster.Name)

	// Build response
	responseCluster := &generated.Cluster{
		ClusterArn:                        ptr.String(cluster.ARN),
		ClusterName:                       ptr.String(cluster.Name),
		Status:                            ptr.String(cluster.Status),
		Settings:                          updatedSettings,
		RegisteredContainerInstancesCount: ptr.Int32(int32(cluster.RegisteredContainerInstancesCount)),
		RunningTasksCount:                 ptr.Int32(int32(cluster.RunningTasksCount)),
		PendingTasksCount:                 ptr.Int32(int32(cluster.PendingTasksCount)),
		ActiveServicesCount:               ptr.Int32(int32(cluster.ActiveServicesCount)),
	}

	// Add configuration if present
	if cluster.Configuration != "" {
		var config generated.ClusterConfiguration
		if err := json.Unmarshal([]byte(cluster.Configuration), &config); err == nil {
			responseCluster.Configuration = &config
		}
	}

	// Add tags if present
	if cluster.Tags != "" {
		var tags []generated.Tag
		if err := json.Unmarshal([]byte(cluster.Tags), &tags); err == nil {
			responseCluster.Tags = tags
		}
	}

	return &generated.UpdateClusterSettingsResponse{
		Cluster: responseCluster,
	}, nil
}

// PutClusterCapacityProviders implements the PutClusterCapacityProviders operation
func (api *DefaultECSAPI) PutClusterCapacityProviders(ctx context.Context, req *generated.PutClusterCapacityProvidersRequest) (*generated.PutClusterCapacityProvidersResponse, error) {
	if req.Cluster == "" {
		return nil, fmt.Errorf("cluster identifier is required")
	}
	if req.CapacityProviders == nil {
		return nil, fmt.Errorf("capacityProviders is required")
	}
	if req.DefaultCapacityProviderStrategy == nil {
		return nil, fmt.Errorf("defaultCapacityProviderStrategy is required")
	}

	// Validate cluster identifier
	if err := ValidateClusterIdentifier(req.Cluster); err != nil {
		return nil, err
	}

	// Validate capacity providers and strategy
	if err := ValidateCapacityProviders(req.CapacityProviders, req.DefaultCapacityProviderStrategy); err != nil {
		return nil, err
	}

	// Extract cluster name from ARN if necessary
	clusterName := extractClusterNameFromARN(req.Cluster)

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", req.Cluster)
	}

	// Marshal capacity providers
	capacityProvidersJSON, err := json.Marshal(req.CapacityProviders)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capacity providers: %w", err)
	}
	cluster.CapacityProviders = string(capacityProvidersJSON)

	// Marshal default capacity provider strategy
	strategyJSON, err := json.Marshal(req.DefaultCapacityProviderStrategy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default capacity provider strategy: %w", err)
	}
	cluster.DefaultCapacityProviderStrategy = string(strategyJSON)

	// Update the cluster
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}
	
	// Invalidate cache for this cluster
	invalidateClusterCache(cluster.Name)

	// Build response
	responseCluster := &generated.Cluster{
		ClusterArn:                        ptr.String(cluster.ARN),
		ClusterName:                       ptr.String(cluster.Name),
		Status:                            ptr.String(cluster.Status),
		CapacityProviders:                 req.CapacityProviders,
		DefaultCapacityProviderStrategy:   req.DefaultCapacityProviderStrategy,
		RegisteredContainerInstancesCount: ptr.Int32(int32(cluster.RegisteredContainerInstancesCount)),
		RunningTasksCount:                 ptr.Int32(int32(cluster.RunningTasksCount)),
		PendingTasksCount:                 ptr.Int32(int32(cluster.PendingTasksCount)),
		ActiveServicesCount:               ptr.Int32(int32(cluster.ActiveServicesCount)),
	}

	// Add settings if present
	if cluster.Settings != "" {
		var settings []generated.ClusterSetting
		if err := json.Unmarshal([]byte(cluster.Settings), &settings); err == nil {
			responseCluster.Settings = settings
		}
	}

	// Add configuration if present
	if cluster.Configuration != "" {
		var config generated.ClusterConfiguration
		if err := json.Unmarshal([]byte(cluster.Configuration), &config); err == nil {
			responseCluster.Configuration = &config
		}
	}

	// Add tags if present
	if cluster.Tags != "" {
		var tags []generated.Tag
		if err := json.Unmarshal([]byte(cluster.Tags), &tags); err == nil {
			responseCluster.Tags = tags
		}
	}

	return &generated.PutClusterCapacityProvidersResponse{
		Cluster: responseCluster,
	}, nil
}

// createKindClusterAndNamespace creates a Kind cluster and namespace for the ECS cluster
func (api *DefaultECSAPI) createKindClusterAndNamespace(cluster *storage.Cluster) {
	ctx := context.Background()

	// Skip kind cluster creation if kindManager is nil (test mode)
	if api.kindManager == nil {
		log.Printf("Skipping kind cluster creation for %s (kindManager is nil)", cluster.Name)
		return
	}

	// Use async operations if available
	if asyncOps := api.getAsyncKindOperations(); asyncOps != nil {
		// Create cluster asynchronously
		operationID := asyncOps.CreateClusterAsync(ctx, cluster.KindClusterName, func(err error) {
			if err != nil {
				log.Printf("Async: Failed to create kind cluster %s for ECS cluster %s: %v", 
					cluster.KindClusterName, cluster.Name, err)
				return
			}
			
			// Cluster created successfully, now create namespace
			api.createNamespaceForCluster(cluster)
		})
		
		log.Printf("Queued async creation of kind cluster %s for ECS cluster %s (operation: %s)", 
			cluster.KindClusterName, cluster.Name, operationID)
		return
	}

	// Fallback to synchronous operation
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

	// Create namespace
	api.createNamespaceForCluster(cluster)
}

// createNamespaceForCluster creates a namespace in the Kind cluster
func (api *DefaultECSAPI) createNamespaceForCluster(cluster *storage.Cluster) {
	ctx := context.Background()
	
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

	log.Printf("Successfully created namespace for ECS cluster %s in kind cluster %s", cluster.Name, cluster.KindClusterName)
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

	// Delete namespace first (while we still have access to the cluster)
	kubeClient, err := api.kindManager.GetKubeClient(cluster.KindClusterName)
	if err == nil {
		namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
		if err := namespaceManager.DeleteNamespace(ctx, cluster.Name, cluster.Region); err != nil {
			log.Printf("Failed to delete namespace for %s: %v", cluster.Name, err)
		}
	}

	// Use async operations if available
	if asyncOps := api.getAsyncKindOperations(); asyncOps != nil {
		// Delete cluster asynchronously
		operationID := asyncOps.DeleteClusterAsync(ctx, cluster.KindClusterName, func(err error) {
			if err != nil {
				log.Printf("Async: Failed to delete kind cluster %s for ECS cluster %s: %v", 
					cluster.KindClusterName, cluster.Name, err)
			} else {
				log.Printf("Async: Successfully deleted kind cluster %s for ECS cluster %s", 
					cluster.KindClusterName, cluster.Name)
			}
		})
		
		log.Printf("Queued async deletion of kind cluster %s for ECS cluster %s (operation: %s)", 
			cluster.KindClusterName, cluster.Name, operationID)
		return
	}

	// Fallback to synchronous operation
	if err := api.kindManager.DeleteCluster(ctx, cluster.KindClusterName); err != nil {
		log.Printf("Failed to delete kind cluster %s for ECS cluster %s: %v", cluster.KindClusterName, cluster.Name, err)
		return
	}

	log.Printf("Successfully deleted kind cluster %s and namespace for ECS cluster %s", cluster.KindClusterName, cluster.Name)
}

// extractClusterNameFromARN extracts cluster name from ARN or returns the input if it's not an ARN
// ARN format: arn:aws:ecs:region:account-id:cluster/cluster-name
func extractClusterNameFromARN(identifier string) string {
	if strings.HasPrefix(identifier, "arn:aws:ecs:") {
		parts := strings.Split(identifier, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[1]
		}
	}
	return identifier
}
