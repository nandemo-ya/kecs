package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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

// PutClusterCapacityProviders implements the PutClusterCapacityProviders operation
func (api *DefaultECSAPI) PutClusterCapacityProviders(ctx context.Context, req *generated.PutClusterCapacityProvidersRequest) (*generated.PutClusterCapacityProvidersResponse, error) {
	// TODO: Implement PutClusterCapacityProviders
	return nil, fmt.Errorf("PutClusterCapacityProviders not implemented")
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