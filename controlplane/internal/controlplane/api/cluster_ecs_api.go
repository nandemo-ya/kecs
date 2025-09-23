package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CreateCluster implements the CreateCluster operation
func (api *DefaultECSAPI) CreateCluster(ctx context.Context, req *generated.CreateClusterRequest) (*generated.CreateClusterResponse, error) {
	logging.Info("Creating cluster", "request", req)

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
		// Ensure the k8s cluster exists (it might have been deleted manually)
		go api.ensureK8sClusterExists(existing)

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

	// In the new design, all ECS clusters share the same k3d cluster (the KECS instance)
	// We need to determine the KECS instance name
	var k8sClusterName string

	// Get the KECS instance name from environment or configuration
	instanceName := api.getKecsInstanceName()
	if instanceName != "" {
		// The k3d cluster name has "kecs-" prefix
		k8sClusterName = fmt.Sprintf("kecs-%s", instanceName)
	}

	if k8sClusterName == "" {
		// Fallback to a default name if we can't determine the instance name
		k8sClusterName = "kecs-default"
		logging.Warn("Could not determine KECS instance name, using default", "k8sClusterName", k8sClusterName)
	}

	// Create cluster object
	cluster := &storage.Cluster{
		ID:                                uuid.New().String(),
		ARN:                               arn,
		Name:                              clusterName,
		Status:                            "ACTIVE",
		Region:                            api.region,
		AccountID:                         api.accountID,
		K8sClusterName:                    k8sClusterName,
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:                 0,
		PendingTasksCount:                 0,
		ActiveServicesCount:               0,
	}

	// Extract settings and configuration from request
	if len(req.Settings) > 0 {
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
	if len(req.Tags) > 0 {
		tagsJSON, err := json.Marshal(req.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal cluster tags: %w", err)
		}
		cluster.Tags = string(tagsJSON)
	}

	// Save to storage (highest priority for immediate visibility)
	logging.Info("Creating cluster in storage", "clusterName", cluster.Name, "clusterARN", cluster.ARN)
	if err := api.storage.ClusterStore().Create(ctx, cluster); err != nil {
		return nil, toECSError(err, "CreateCluster")
	}
	logging.Info("Cluster created in storage", "clusterName", cluster.Name)

	// Immediately verify it's readable
	verifyCluster, err := api.storage.ClusterStore().Get(ctx, cluster.Name)
	if err != nil {
		logging.Error("Failed to immediately read created cluster", "clusterName", cluster.Name, "error", err)
	} else {
		logging.Info("Verified cluster is immediately readable", "clusterName", verifyCluster.Name, "clusterARN", verifyCluster.ARN)
	}

	// Create k8s namespace asynchronously (won't block cluster visibility)
	go api.createNamespaceForCluster(cluster)

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
	logging.Debug("ListClusters request", "request", string(reqJSON))

	// Set default limit if not specified
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		// AWS ECS has a maximum of 100 results per page
		if limit > 100 {
			limit = 100
		}
		logging.Debug("ListClusters pagination", "maxResults", *req.MaxResults, "effectiveLimit", limit)
	} else {
		logging.Debug("ListClusters pagination", "defaultLimit", limit)
	}

	// Extract next token
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Get clusters with pagination
	logging.Info("ListClusters: Fetching clusters from storage", "limit", limit, "nextToken", nextToken)
	clusters, newNextToken, err := api.storage.ClusterStore().ListWithPagination(ctx, limit, nextToken)
	if err != nil {
		logging.Error("ListClusters storage error", "error", err)
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	logging.Info("ListClusters results", "count", len(clusters), "clusters", func() []string {
		names := make([]string, len(clusters))
		for i, c := range clusters {
			names[i] = c.Name
		}
		return names
	}())

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
		logging.Debug("ListClusters response", "count", len(clusterArns), "nextToken", newNextToken)
	} else {
		logging.Debug("ListClusters response", "count", len(clusterArns), "hasNextToken", false)
	}

	// Debug log the response
	respJSON, _ := json.Marshal(response)
	logging.Debug("ListClusters response", "response", string(respJSON))

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
		// Handle empty identifier
		if identifier == "" {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(""),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String("Empty cluster identifier"),
			})
			continue
		}

		// Validate ARN format if it looks like an ARN
		if strings.HasPrefix(identifier, "arn:") {
			if err := ValidateClusterARN(identifier); err != nil {
				failures = append(failures, generated.Failure{
					Arn:    ptr.String(identifier),
					Reason: ptr.String("MISSING"),
					Detail: ptr.String("Invalid ARN format"),
				})
				continue
			}
		}

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
		return nil, toECSError(err, "DeleteCluster")
	}

	// Delete the cluster
	if err := api.storage.ClusterStore().Delete(ctx, cluster.Name); err != nil {
		return nil, toECSError(err, "DeleteCluster")
	}

	// Invalidate cache for this cluster
	invalidateClusterCache(cluster.Name)

	// Delete k8s cluster and namespace asynchronously
	go api.deleteK8sClusterAndNamespace(cluster)

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
	if len(req.Settings) > 0 {
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
		logging.Warn("ServiceConnectDefaults update requested but not fully implemented yet")
	}

	// Update the cluster
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		return nil, toECSError(err, "UpdateCluster")
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
			logging.Error("Failed to unmarshal existing settings", "error", err)
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
		return nil, toECSError(err, "UpdateCluster")
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
		return nil, toECSError(err, "UpdateCluster")
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

// createNamespaceForCluster creates a namespace in the k3d cluster
func (api *DefaultECSAPI) createNamespaceForCluster(cluster *storage.Cluster) {
	ctx := context.Background()

	// Try to create Kubernetes client using in-cluster config
	kubeClient, err := kubernetes.GetInClusterClient()
	if err != nil {
		logging.Error("Failed to get in-cluster kubernetes client", "error", err)
		return
	}

	// Create namespace
	namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
	if err := namespaceManager.CreateNamespace(ctx, cluster.Name, cluster.Region); err != nil {
		logging.Error("Failed to create namespace", "cluster", cluster.Name, "error", err)
		return
	}

	logging.Info("Successfully created namespace", "namespace", cluster.Name, "ecsCluster", cluster.Name, "kecsInstance", cluster.K8sClusterName)
}

// ensureK8sClusterExists ensures that the namespace exists for an existing ECS cluster
func (api *DefaultECSAPI) ensureK8sClusterExists(cluster *storage.Cluster) {
	// In the new design, we only need to ensure the namespace exists
	// The k3d cluster is managed by the KECS instance itself
	logging.Debug("Ensuring namespace exists", "ecsCluster", cluster.Name, "kecsInstance", cluster.K8sClusterName)

	// Just ensure the namespace exists
	api.createNamespaceForCluster(cluster)
}

// deleteK8sClusterAndNamespace deletes the namespace for an ECS cluster
func (api *DefaultECSAPI) deleteK8sClusterAndNamespace(cluster *storage.Cluster) {
	ctx := context.Background()

	// In the new design, we only delete the namespace
	// The k3d cluster is managed by the KECS instance itself
	logging.Info("Deleting namespace", "namespace", cluster.Name, "ecsCluster", cluster.Name, "kecsInstance", cluster.K8sClusterName)

	// Try to create Kubernetes client
	// Get in-cluster kubernetes client
	kubeClient, err := kubernetes.GetInClusterClient()
	if err != nil {
		logging.Error("Failed to get in-cluster kubernetes client", "error", err)
		return
	}

	// Delete namespace
	namespaceManager := kubernetes.NewNamespaceManager(kubeClient)
	if err := namespaceManager.DeleteNamespace(ctx, cluster.Name, cluster.Region); err != nil {
		logging.Error("Failed to delete namespace", "namespace", cluster.Name, "error", err)
		return
	}

	logging.Info("Successfully deleted namespace", "namespace", cluster.Name, "ecsCluster", cluster.Name)
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

// updateLocalStackState updates the LocalStack deployment state in storage
func (api *DefaultECSAPI) updateLocalStackState(cluster *storage.Cluster, status string, errorMsg string) {
	ctx := context.Background()

	// Create LocalStack state
	now := time.Now()
	state := &storage.LocalStackState{
		Deployed:   true,
		Status:     status,
		DeployedAt: &now,
		Namespace:  "kecs-system",
	}

	// Add error message if status is failed
	if status == "failed" && errorMsg != "" {
		state.HealthStatus = errorMsg
	} else if status == "running" && errorMsg != "" {
		// If running but with warnings (e.g., DNS issues), record as warning
		state.HealthStatus = "warning: " + errorMsg
	}

	// Serialize state
	stateJSON, err := storage.SerializeLocalStackState(state)
	if err != nil {
		logging.Error("Failed to serialize LocalStack state", "error", err)
		return
	}

	// Update cluster
	cluster.LocalStackState = stateJSON
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		logging.Error("Failed to update LocalStack state", "cluster", cluster.Name, "error", err)
	}
}

// getKecsInstanceName attempts to determine the KECS instance name (k3d cluster name)
func (api *DefaultECSAPI) getKecsInstanceName() string {
	// In the container-based deployment model, there should be a single k3d cluster
	// that hosts the KECS control plane and all ECS clusters as namespaces

	// For now, we'll use the environment variable if set
	if instanceName := os.Getenv("KECS_INSTANCE_NAME"); instanceName != "" {
		return instanceName
	}

	// Otherwise, try to detect from the current environment
	// When running inside k3d, the hostname typically follows the pattern k3d-<instance>-server-0
	hostname, err := os.Hostname()
	if err == nil && strings.HasPrefix(hostname, "k3d-") && strings.HasSuffix(hostname, "-server-0") {
		// Extract instance name from hostname
		parts := strings.Split(hostname, "-")
		if len(parts) >= 3 {
			// k3d-<instance>-server-0 -> extract <instance>
			instanceName := strings.Join(parts[1:len(parts)-2], "-")
			if instanceName != "" {
				logging.Debug("Detected KECS instance name from hostname", "instanceName", instanceName)
				return instanceName
			}
		}
	}

	// If we still can't determine it, return empty string
	return ""
}
