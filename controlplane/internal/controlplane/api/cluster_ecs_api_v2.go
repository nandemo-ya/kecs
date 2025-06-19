package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPIV2 implements ECS API using AWS SDK v2 types
type DefaultECSAPIV2 struct {
	storage     storage.Storage
	kindManager *kubernetes.KindManager
}

// NewDefaultECSAPIV2 creates a new DefaultECSAPIV2 instance
func NewDefaultECSAPIV2(storage storage.Storage, kindManager *kubernetes.KindManager) *DefaultECSAPIV2 {
	return &DefaultECSAPIV2{
		storage:     storage,
		kindManager: kindManager,
	}
}

// ListClustersV2 implements the ListClusters operation using AWS SDK types
func (api *DefaultECSAPIV2) ListClustersV2(ctx context.Context, req *ecs.ListClustersInput) (*ecs.ListClustersOutput, error) {
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

	response := &ecs.ListClustersOutput{
		ClusterArns: clusterArns,
	}

	// Set next token if there are more results
	if newNextToken != "" {
		response.NextToken = aws.String(newNextToken)
		log.Printf("ListClusters: Returning %d clusters with nextToken=%s", len(clusterArns), newNextToken)
	} else {
		log.Printf("ListClusters: Returning %d clusters with no nextToken", len(clusterArns))
	}

	return response, nil
}

// CreateClusterV2 implements the CreateCluster operation using AWS SDK types
func (api *DefaultECSAPIV2) CreateClusterV2(ctx context.Context, req *ecs.CreateClusterInput) (*ecs.CreateClusterOutput, error) {
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
		if api.kindManager != nil {
			if err := api.kindManager.CreateCluster(ctx, fmt.Sprintf("kecs-%s", clusterName)); err != nil {
				log.Printf("Warning: failed to ensure kind cluster: %v", err)
			}
		}

		// Return existing cluster
		return &ecs.CreateClusterOutput{
			Cluster: &ecstypes.Cluster{
				ClusterName:               aws.String(existing.Name),
				ClusterArn:                aws.String(existing.ARN),
				Status:                    aws.String(existing.Status),
				ActiveServicesCount:       int32(existing.ActiveServicesCount),
				RunningTasksCount:         int32(existing.RunningTasksCount),
				PendingTasksCount:         int32(existing.PendingTasksCount),
				RegisteredContainerInstancesCount: int32(existing.RegisteredContainerInstancesCount),
				Tags: convertStorageTagsToSDK(existing.Tags),
			},
		}, nil
	}

	// Generate cluster ARN
	clusterArn := fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:cluster/%s", clusterName)

	// Create kind cluster if manager is available
	if api.kindManager != nil {
		kindClusterName := fmt.Sprintf("kecs-%s", clusterName)
		if err := api.kindManager.CreateCluster(ctx, kindClusterName); err != nil {
			log.Printf("Warning: failed to create kind cluster: %v", err)
		}
	} else {
		log.Printf("Skipping kind cluster creation for %s (kindManager is nil)", clusterName)
	}

	// Create namespace for the cluster
	// TODO: Implement namespace creation when needed

	// Process settings - convert to JSON string
	settingsJSON := ""
	if req.Settings != nil && len(req.Settings) > 0 {
		settingsData := make([]map[string]string, 0)
		for _, s := range req.Settings {
			if s.Value != nil {
				settingsData = append(settingsData, map[string]string{
					"name":  string(s.Name),
					"value": *s.Value,
				})
			}
		}
		if data, err := json.Marshal(settingsData); err == nil {
			settingsJSON = string(data)
		}
	}

	// Process tags - convert to JSON string
	tagsJSON := ""
	if req.Tags != nil && len(req.Tags) > 0 {
		tagsMap := convertFromSDKTags(req.Tags)
		if data, err := json.Marshal(tagsMap); err == nil {
			tagsJSON = string(data)
		}
	}

	// Store cluster in database
	cluster := &storage.Cluster{
		Name:     clusterName,
		ARN:      clusterArn,
		Status:   "ACTIVE",
		Settings: settingsJSON,
		Tags:     tagsJSON,
	}

	if err := api.storage.ClusterStore().Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Return response
	response := &ecs.CreateClusterOutput{
		Cluster: &ecstypes.Cluster{
			ClusterName:               aws.String(clusterName),
			ClusterArn:                aws.String(clusterArn),
			Status:                    aws.String("ACTIVE"),
			ActiveServicesCount:       0,
			RunningTasksCount:         0,
			PendingTasksCount:         0,
			RegisteredContainerInstancesCount: 0,
			Settings: convertSettingsJSONToSDK(settingsJSON),
			Tags: convertStorageTagsToSDK(tagsJSON),
		},
	}

	return response, nil
}

// Helper functions to convert between storage and SDK types
func convertToSDKTags(tags map[string]string) []ecstypes.Tag {
	sdkTags := make([]ecstypes.Tag, 0, len(tags))
	for k, v := range tags {
		sdkTags = append(sdkTags, ecstypes.Tag{
			Key:   aws.String(k),
			Value: aws.String(v),
		})
	}
	return sdkTags
}

func convertFromSDKTags(tags []ecstypes.Tag) map[string]string {
	tagMap := make(map[string]string)
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			tagMap[*tag.Key] = *tag.Value
		}
	}
	return tagMap
}

func convertStorageTagsToSDK(tagsJSON string) []ecstypes.Tag {
	if tagsJSON == "" {
		return nil
	}
	
	var tagsMap map[string]string
	if err := json.Unmarshal([]byte(tagsJSON), &tagsMap); err != nil {
		return nil
	}
	
	return convertToSDKTags(tagsMap)
}

func convertSettingsJSONToSDK(settingsJSON string) []ecstypes.ClusterSetting {
	if settingsJSON == "" {
		return nil
	}
	
	var settingsData []map[string]string
	if err := json.Unmarshal([]byte(settingsJSON), &settingsData); err != nil {
		return nil
	}
	
	sdkSettings := make([]ecstypes.ClusterSetting, 0, len(settingsData))
	for _, s := range settingsData {
		if name, ok := s["name"]; ok {
			if value, ok := s["value"]; ok {
				sdkSettings = append(sdkSettings, ecstypes.ClusterSetting{
					Name:  ecstypes.ClusterSettingName(name),
					Value: aws.String(value),
				})
			}
		}
	}
	return sdkSettings
}

// DescribeClustersV2 implements the DescribeClusters operation using AWS SDK types
func (api *DefaultECSAPIV2) DescribeClustersV2(ctx context.Context, req *ecs.DescribeClustersInput) (*ecs.DescribeClustersOutput, error) {
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
	var describedClusters []ecstypes.Cluster
	var failures []ecstypes.Failure

	for _, identifier := range clusterIdentifiers {
		// Extract cluster name from ARN if necessary
		clusterName := extractClusterNameFromARN(identifier)

		cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)

		if err != nil {
			failures = append(failures, ecstypes.Failure{
				Arn:    aws.String(identifier),
				Reason: aws.String("MISSING"),
				Detail: aws.String(fmt.Sprintf("Could not find cluster %s", identifier)),
			})
			continue
		}

		// Build cluster response
		clusterResp := ecstypes.Cluster{
			ClusterArn:                        aws.String(cluster.ARN),
			ClusterName:                       aws.String(cluster.Name),
			Status:                            aws.String(cluster.Status),
			RegisteredContainerInstancesCount: int32(cluster.RegisteredContainerInstancesCount),
			RunningTasksCount:                 int32(cluster.RunningTasksCount),
			PendingTasksCount:                 int32(cluster.PendingTasksCount),
			ActiveServicesCount:               int32(cluster.ActiveServicesCount),
		}

		// Add settings if requested
		if req.Include != nil {
			for _, include := range req.Include {
				switch include {
				case ecstypes.ClusterFieldSettings:
					if cluster.Settings != "" {
						settings := parseClusterSettingsForV2(cluster.Settings)
						if settings != nil {
							clusterResp.Settings = settings
						}
					}
				case ecstypes.ClusterFieldConfigurations:
					if cluster.Configuration != "" {
						config := parseClusterConfigurationForV2(cluster.Configuration)
						if config != nil {
							clusterResp.Configuration = config
						}
					}
				case ecstypes.ClusterFieldTags:
					if cluster.Tags != "" {
						clusterResp.Tags = convertStorageTagsToSDK(cluster.Tags)
					}
				}
			}
		}

		describedClusters = append(describedClusters, clusterResp)
	}

	return &ecs.DescribeClustersOutput{
		Clusters: describedClusters,
		Failures: failures,
	}, nil
}

// DeleteClusterV2 implements the DeleteCluster operation using AWS SDK types
func (api *DefaultECSAPIV2) DeleteClusterV2(ctx context.Context, req *ecs.DeleteClusterInput) (*ecs.DeleteClusterOutput, error) {
	if req.Cluster == nil {
		return nil, fmt.Errorf("cluster identifier is required")
	}

	// Extract cluster name from ARN if necessary
	clusterName := extractClusterNameFromARN(*req.Cluster)

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)

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
	response := &ecs.DeleteClusterOutput{
		Cluster: &ecstypes.Cluster{
			ClusterArn:  aws.String(cluster.ARN),
			ClusterName: aws.String(cluster.Name),
			Status:      aws.String("INACTIVE"),
		},
	}

	return response, nil
}

// UpdateClusterV2 implements the UpdateCluster operation using AWS SDK types
func (api *DefaultECSAPIV2) UpdateClusterV2(ctx context.Context, req *ecs.UpdateClusterInput) (*ecs.UpdateClusterOutput, error) {
	if req.Cluster == nil {
		return nil, fmt.Errorf("cluster identifier is required")
	}

	// Extract cluster name from ARN if necessary
	clusterName := extractClusterNameFromARN(*req.Cluster)

	// Look up cluster
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %s", *req.Cluster)
	}

	// Update settings if provided
	if req.Settings != nil && len(req.Settings) > 0 {
		settingsData := make([]map[string]string, 0)
		for _, s := range req.Settings {
			if s.Value != nil {
				settingsData = append(settingsData, map[string]string{
					"name":  string(s.Name),
					"value": *s.Value,
				})
			}
		}
		if data, err := json.Marshal(settingsData); err == nil {
			cluster.Settings = string(data)
		}
	}

	// Update configuration if provided
	if req.Configuration != nil {
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

	// Build response
	responseCluster := &ecstypes.Cluster{
		ClusterArn:                        aws.String(cluster.ARN),
		ClusterName:                       aws.String(cluster.Name),
		Status:                            aws.String(cluster.Status),
		RegisteredContainerInstancesCount: int32(cluster.RegisteredContainerInstancesCount),
		RunningTasksCount:                 int32(cluster.RunningTasksCount),
		PendingTasksCount:                 int32(cluster.PendingTasksCount),
		ActiveServicesCount:               int32(cluster.ActiveServicesCount),
	}

	// Add settings if present
	if cluster.Settings != "" {
		settings := parseClusterSettingsForV2(cluster.Settings)
		if settings != nil {
			responseCluster.Settings = settings
		}
	}

	// Add configuration if present
	if cluster.Configuration != "" {
		config := parseClusterConfigurationForV2(cluster.Configuration)
		if config != nil {
			responseCluster.Configuration = config
		}
	}

	// Add tags if present
	if cluster.Tags != "" {
		responseCluster.Tags = convertStorageTagsToSDK(cluster.Tags)
	}

	return &ecs.UpdateClusterOutput{
		Cluster: responseCluster,
	}, nil
}

// Helper functions for V2 type conversions
func parseClusterSettingsForV2(settingsJSON string) []ecstypes.ClusterSetting {
	if settingsJSON == "" {
		return nil
	}
	
	var settingsData []map[string]string
	if err := json.Unmarshal([]byte(settingsJSON), &settingsData); err != nil {
		// Try parsing as generated type format
		var generatedSettings []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(settingsJSON), &generatedSettings); err != nil {
			return nil
		}
		// Convert to expected format
		for _, s := range generatedSettings {
			settingsData = append(settingsData, map[string]string{
				"name":  s.Name,
				"value": s.Value,
			})
		}
	}
	
	sdkSettings := make([]ecstypes.ClusterSetting, 0, len(settingsData))
	for _, s := range settingsData {
		if name, ok := s["name"]; ok {
			if value, ok := s["value"]; ok {
				sdkSettings = append(sdkSettings, ecstypes.ClusterSetting{
					Name:  ecstypes.ClusterSettingName(name),
					Value: aws.String(value),
				})
			}
		}
	}
	return sdkSettings
}

func parseClusterConfigurationForV2(configJSON string) *ecstypes.ClusterConfiguration {
	if configJSON == "" {
		return nil
	}
	
	var config ecstypes.ClusterConfiguration
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil
	}
	return &config
}

// deleteKindClusterAndNamespace deletes the Kind cluster and namespace for an ECS cluster
func (api *DefaultECSAPIV2) deleteKindClusterAndNamespace(cluster *storage.Cluster) {
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

	// Delete the kind cluster
	if err := api.kindManager.DeleteCluster(ctx, cluster.KindClusterName); err != nil {
		log.Printf("Failed to delete kind cluster %s for ECS cluster %s: %v", cluster.KindClusterName, cluster.Name, err)
		return
	}

	log.Printf("Successfully deleted kind cluster %s and namespace for ECS cluster %s", cluster.KindClusterName, cluster.Name)
}