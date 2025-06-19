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