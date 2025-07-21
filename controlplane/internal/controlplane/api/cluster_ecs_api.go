package api

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apiconfig "github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
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
	
	// Try to get the KECS instance name from the existing k3d clusters
	if api.clusterManager != nil {
		// For now, we'll use a simple approach - look for a k3d cluster
		// In a real implementation, this should be passed from the server configuration
		k8sClusterName = api.getKecsInstanceName()
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

	// Save to storage
	if err := api.storage.ClusterStore().Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Create k8s cluster and namespace asynchronously
	go api.createK8sClusterAndNamespace(cluster)

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
	clusters, newNextToken, err := api.storage.ClusterStore().ListWithPagination(ctx, limit, nextToken)
	if err != nil {
		logging.Error("ListClusters storage error", "error", err)
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	logging.Debug("ListClusters results", "count", len(clusters))

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
		return nil, fmt.Errorf("failed to update cluster status: %w", err)
	}

	// Delete the cluster
	if err := api.storage.ClusterStore().Delete(ctx, cluster.Name); err != nil {
		return nil, fmt.Errorf("failed to delete cluster: %w", err)
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

// createK8sClusterAndNamespace creates a namespace for the ECS cluster in the existing KECS instance
func (api *DefaultECSAPI) createK8sClusterAndNamespace(cluster *storage.Cluster) {
	// In the new design, we use the existing KECS instance's k3d cluster
	// ECS clusters are represented as Kubernetes namespaces
	logging.Info("Creating namespace for ECS cluster", "cluster", cluster.Name, "k8sCluster", cluster.K8sClusterName)

	// In the new architecture, the KECS instance (k3d cluster) should already exist
	// We only need to create namespaces for ECS clusters
	// The k3d cluster name in storage is just for reference to the KECS instance

	// Create namespace
	api.createNamespaceForCluster(cluster)

	// Deploy LocalStack if enabled
	api.deployLocalStackIfEnabled(cluster)
}

// createNamespaceForCluster creates a namespace in the k3d cluster
func (api *DefaultECSAPI) createNamespaceForCluster(cluster *storage.Cluster) {
	ctx := context.Background()

	// Try to create Kubernetes client
	// First, try in-cluster config (when running inside Kubernetes)
	kubeClient, err := kubernetes.GetInClusterClient()
	if err != nil {
		// If in-cluster fails, try using cluster manager (for local development)
		logging.Debug("In-cluster config failed (expected in local development)", "error", err)
		clusterManager := api.getClusterManager()
		if clusterManager == nil {
			logging.Error("Cannot create namespace: no Kubernetes client available (neither in-cluster nor cluster manager)")
			return
		}

		// Get Kubernetes client for the KECS instance
		client, err := clusterManager.GetKubeClient(cluster.K8sClusterName)
		if err != nil {
			logging.Error("Failed to get kubernetes client", "error", err)
			return
		}
		kubeClient = client.(*k8s.Clientset)
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
	// First, try in-cluster config (when running inside Kubernetes)
	kubeClient, err := kubernetes.GetInClusterClient()
	if err != nil {
		// If in-cluster fails, try using cluster manager (for local development)
		logging.Debug("In-cluster config failed (expected in local development)", "error", err)
		clusterManager := api.getClusterManager()
		if clusterManager == nil {
			logging.Error("Cannot delete namespace: no Kubernetes client available (neither in-cluster nor cluster manager)")
			return
		}

		// Get Kubernetes client for the KECS instance
		client, err := clusterManager.GetKubeClient(cluster.K8sClusterName)
		if err != nil {
			logging.Error("Failed to get kubernetes client", "error", err)
			return
		}
		kubeClient = client.(*k8s.Clientset)
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

// deployLocalStackIfEnabled deploys LocalStack to the k3d cluster if enabled
func (api *DefaultECSAPI) deployLocalStackIfEnabled(cluster *storage.Cluster) {
	logging.Debug("deployLocalStackIfEnabled called", "cluster", cluster.Name)
	
	// Skip if cluster manager is not available
	if api.clusterManager == nil {
		logging.Warn("Cluster manager not available, cannot deploy LocalStack", "cluster", cluster.Name)
		return
	}

	ctx := context.Background()

	// Get the LocalStack configuration
	var config *localstack.Config
	if api.localStackConfig != nil {
		// Create a copy of the config to avoid modifying the shared instance
		configCopy := *api.localStackConfig
		config = &configCopy
		// Ensure container mode is set if running in container
		if apiconfig.GetBool("features.containerMode") {
			config.ContainerMode = true
		}
		logging.Debug("Using LocalStack config from API", "enabled", config.Enabled, "useTraefik", config.UseTraefik, "containerMode", config.ContainerMode)
	} else if api.localStackManager != nil {
		config = api.localStackManager.GetConfig()
	} else {
		// Use default config and check if enabled via environment
		config = localstack.DefaultConfig()
		// Use Viper config which handles environment variables
		appConfig := apiconfig.GetConfig()
		if appConfig.LocalStack.Enabled {
			config.Enabled = true
		}
		// Check features.traefik configuration
		if appConfig.Features.Traefik {
			config.UseTraefik = true
			logging.Debug("Traefik is enabled for LocalStack via features.traefik")
		}
		// Set container mode
		if appConfig.Features.ContainerMode {
			config.ContainerMode = true
			logging.Debug("Container mode is enabled for LocalStack")
		}
	}
	
	if config == nil || !config.Enabled {
		logging.Debug("LocalStack is not enabled in configuration")
		return
	}
	
	// If Traefik is enabled, get the dynamic port from cluster manager
	if config.UseTraefik && api.clusterManager != nil {
		if port, exists := api.clusterManager.GetTraefikPort(cluster.K8sClusterName); exists {
			// In container mode, use k3d node hostname with NodePort
			if config.ContainerMode {
				k3dNodeName := fmt.Sprintf("k3d-%s-server-0", cluster.K8sClusterName)
				config.ProxyEndpoint = fmt.Sprintf("http://%s:30890", k3dNodeName)
				logging.Info("Container mode: Using k3d node for LocalStack proxy", "node", k3dNodeName, "port", 30890, "endpoint", config.ProxyEndpoint)
			} else {
				config.ProxyEndpoint = fmt.Sprintf("http://localhost:%d", port)
				logging.Info("Using dynamic Traefik port for LocalStack proxy", "port", port, "endpoint", config.ProxyEndpoint)
			}
		} else {
			logging.Warn("Traefik is enabled but no port found", "cluster", cluster.K8sClusterName)
		}
	} else {
		logging.Debug("Traefik disabled or cluster manager not available", "useTraefik", config.UseTraefik, "hasClusterManager", api.clusterManager != nil)
	}

	// Set lazy loading for faster startup in container mode
	if config.ContainerMode {
		// Use lazy loading to avoid timeout issues
		if config.Environment == nil {
			config.Environment = make(map[string]string)
		}
		config.Environment["EAGER_SERVICE_LOADING"] = "0"
		logging.Debug("Container mode: Disabled eager service loading for faster LocalStack startup")
	}
	
	// Try to create Kubernetes client
	// First, try in-cluster config (when running inside Kubernetes)
	var kubeClient k8s.Interface
	var kubeConfig *rest.Config
	
	// Try in-cluster config first
	inClusterClient, err := kubernetes.GetInClusterClient()
	if err == nil {
		kubeClient = inClusterClient
		// For in-cluster config, we need to get the config separately
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			logging.Debug("Failed to get in-cluster config", "error", err)
			return
		}
	} else {
		// If in-cluster fails, try using cluster manager (for local development)
		logging.Debug("In-cluster config failed (expected in local development)", "error", err)
		
		// Get Kubernetes client for the specific k3d cluster
		client, err := api.clusterManager.GetKubeClient(cluster.K8sClusterName)
		if err != nil {
			logging.Error("Failed to get Kubernetes client", "error", err)
			return
		}
		kubeClient = client

		// Get kube config
		kubeConfig, err = api.clusterManager.GetKubeConfig(cluster.K8sClusterName)
		if err != nil {
			logging.Error("Failed to get kube config", "error", err)
			return
		}
	}

	// Create a new LocalStack manager with the cluster-specific client
	clusterLocalStackManager, err := localstack.NewManager(config, kubeClient.(*k8s.Clientset), kubeConfig)
	if err != nil {
		logging.Error("Failed to create LocalStack manager", "cluster", cluster.Name, "error", err)
		return
	}

	// Check if LocalStack is already running in this cluster
	if clusterLocalStackManager.IsRunning() {
		logging.Info("LocalStack is already running", "cluster", cluster.Name)
		return
	}

	// Start LocalStack in the cluster
	logging.Info("Starting LocalStack", "cluster", cluster.Name)
	// Update LocalStack state to deploying
	api.updateLocalStackState(cluster, "deploying", "")
	
	if err := clusterLocalStackManager.Start(ctx); err != nil {
		logging.Error("Failed to start LocalStack", "cluster", cluster.Name, "error", err)
		// Update LocalStack state to failed
		api.updateLocalStackState(cluster, "failed", err.Error())
		return
	}

	// Wait for LocalStack to be ready (monitoring logs for "Ready." message)
	logging.Info("Waiting for LocalStack to be ready (monitoring logs)", "cluster", cluster.Name)
	
	// Check LocalStack status - the manager now monitors logs for "Ready."
	status, err := clusterLocalStackManager.GetStatus()
	if err != nil {
		logging.Error("Failed to get LocalStack status", "cluster", cluster.Name, "error", err)
		api.updateLocalStackState(cluster, "failed", err.Error())
		return
	}
	
	if status.Running && status.Healthy {
		logging.Info("LocalStack successfully deployed and ready", "cluster", cluster.Name)
		api.updateLocalStackState(cluster, "running", "")
	} else if status.Running && !status.Healthy {
		logging.Info("LocalStack is running but not yet fully ready", "cluster", cluster.Name)
		// Still mark as running since it can handle requests
		api.updateLocalStackState(cluster, "running", "Services still initializing")
	} else {
		logging.Error("LocalStack failed to start", "cluster", cluster.Name)
		api.updateLocalStackState(cluster, "failed", "Failed to start")
	}
	
	// Update the global LocalStack manager reference and notify server to re-initialize AWS proxy router
	api.localStackManager = clusterLocalStackManager
	
	// Call the update callback if set
	if api.localStackUpdateCallback != nil {
		logging.Debug("Notifying server about LocalStack manager update")
		api.localStackUpdateCallback(clusterLocalStackManager)
	}
}

// updateLocalStackState updates the LocalStack deployment state in storage
func (api *DefaultECSAPI) updateLocalStackState(cluster *storage.Cluster, status string, errorMsg string) {
	ctx := context.Background()
	
	// Create LocalStack state
	now := time.Now()
	state := &storage.LocalStackState{
		Deployed: true,
		Status:   status,
		DeployedAt: &now,
		Namespace: "kecs-system",
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
