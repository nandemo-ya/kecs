package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	generated_v2 "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPIGenerated implements ECS API using generated types
type DefaultECSAPIGenerated struct {
	storage     storage.Storage
	kindManager *kubernetes.KindManager
	region      string
	accountID   string
}

// NewDefaultECSAPIGenerated creates a new DefaultECSAPIGenerated instance
func NewDefaultECSAPIGenerated(storage storage.Storage, kindManager *kubernetes.KindManager) *DefaultECSAPIGenerated {
	return &DefaultECSAPIGenerated{
		storage:     storage,
		kindManager: kindManager,
		region:      "ap-northeast-1",
		accountID:   "123456789012",
	}
}

// ListClusters implements the ListClusters operation using generated types
func (api *DefaultECSAPIGenerated) ListClusters(ctx context.Context, req *generated_v2.ListClustersRequest) (*generated_v2.ListClustersResponse, error) {
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

	// Get clusters from storage
	clusterStore := api.storage.ClusterStore()
	clusters, nextToken, err := clusterStore.ListWithPagination(ctx, limit, ptrToString(req.NextToken))
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	// Convert to ARNs
	clusterArns := make([]string, len(clusters))
	for i, cluster := range clusters {
		clusterArns[i] = fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, cluster.Name)
	}

	// Build response
	resp := &generated_v2.ListClustersResponse{
		ClusterArns: clusterArns,
	}

	// Set next token if there are more results
	if nextToken != "" {
		resp.NextToken = &nextToken
	}

	// Debug log the response
	respJSON, _ := json.Marshal(resp)
	log.Printf("ListClusters response: %s", string(respJSON))

	return resp, nil
}

// CreateCluster implements the CreateCluster operation using generated types
func (api *DefaultECSAPIGenerated) CreateCluster(ctx context.Context, req *generated_v2.CreateClusterRequest) (*generated_v2.CreateClusterResponse, error) {
	// Debug log the request
	reqJSON, _ := json.Marshal(req)
	log.Printf("CreateCluster request: %s", string(reqJSON))

	// Validate cluster name
	clusterName := ""
	if req.ClusterName != nil {
		clusterName = *req.ClusterName
	}

	if clusterName == "" {
		// Generate default cluster name
		clusterName = "default"
	}

	// Create cluster in storage
	storageCluster := &storage.Cluster{
		ARN:       fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, clusterName),
		Name:      clusterName,
		Status:    "ACTIVE",
		Region:    api.region,
		AccountID: api.accountID,
		// Store settings, tags, and configuration as JSON
		Settings:      api.convertSettingsToJSON(req.Settings),
		Tags:          api.convertTagsToJSON(req.Tags),
		Configuration: api.convertConfigurationToJSON(req.Configuration),
	}

	// Store cluster
	clusterStore := api.storage.ClusterStore()
	err := clusterStore.Create(ctx, storageCluster)
	if err != nil {
		// Check if cluster already exists
		existingCluster, getErr := clusterStore.Get(ctx, clusterName)
		if getErr == nil && existingCluster != nil {
			// Return existing cluster
			return api.buildCreateClusterResponse(existingCluster), nil
		}
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Get created cluster
	createdCluster, err := clusterStore.Get(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get created cluster: %w", err)
	}

	return api.buildCreateClusterResponse(createdCluster), nil
}

// DeleteCluster implements the DeleteCluster operation using generated types
func (api *DefaultECSAPIGenerated) DeleteCluster(ctx context.Context, req *generated_v2.DeleteClusterRequest) (*generated_v2.DeleteClusterResponse, error) {
	// Debug log the request
	reqJSON, _ := json.Marshal(req)
	log.Printf("DeleteCluster request: %s", string(reqJSON))

	// Validate cluster name
	if req.Cluster == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	// Get cluster before deletion
	clusterStore := api.storage.ClusterStore()
	cluster, err := clusterStore.Get(ctx, req.Cluster)
	if err != nil {
		return nil, &ClusterNotFoundException{
			Message: fmt.Sprintf("Cluster not found: %s", req.Cluster),
		}
	}

	// Check if cluster has services
	serviceStore := api.storage.ServiceStore()
	services, _, err := serviceStore.List(ctx, req.Cluster, "", "", 1, "")
	if err != nil {
		return nil, fmt.Errorf("failed to check services: %w", err)
	}
	if len(services) > 0 {
		return nil, &ClusterContainsServicesException{
			Message: fmt.Sprintf("Cluster contains %d services", len(services)),
		}
	}

	// Check if cluster has tasks
	taskStore := api.storage.TaskStore()
	tasks, err := taskStore.List(ctx, req.Cluster, storage.TaskFilters{})
	if err != nil {
		return nil, fmt.Errorf("failed to check tasks: %w", err)
	}
	if len(tasks) > 0 {
		return nil, &ClusterContainsTasksException{
			Message: fmt.Sprintf("Cluster contains %d tasks", len(tasks)),
		}
	}

	// Delete cluster
	err = clusterStore.Delete(ctx, req.Cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to delete cluster: %w", err)
	}

	// Update cluster status
	cluster.Status = "INACTIVE"

	return &generated_v2.DeleteClusterResponse{
		Cluster: api.convertClusterToGenerated(cluster),
	}, nil
}

// DescribeClusters implements the DescribeClusters operation using generated types
func (api *DefaultECSAPIGenerated) DescribeClusters(ctx context.Context, req *generated_v2.DescribeClustersRequest) (*generated_v2.DescribeClustersResponse, error) {
	// Debug log the request
	reqJSON, _ := json.Marshal(req)
	log.Printf("DescribeClusters request: %s", string(reqJSON))

	clusterStore := api.storage.ClusterStore()
	
	// If no clusters specified, describe all clusters
	if req.Clusters == nil || len(req.Clusters) == 0 {
		clusters, err := clusterStore.List(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list clusters: %w", err)
		}

		// Convert all clusters
		result := make([]generated_v2.Cluster, len(clusters))
		for i, cluster := range clusters {
			result[i] = *api.convertClusterToGenerated(cluster)
		}

		return &generated_v2.DescribeClustersResponse{
			Clusters: result,
		}, nil
	}

	// Describe specific clusters
	result := make([]generated_v2.Cluster, 0, len(req.Clusters))
	failures := make([]generated_v2.Failure, 0)

	for _, clusterName := range req.Clusters {
		cluster, err := clusterStore.Get(ctx, clusterName)
		if err != nil {
			// Add to failures
			reason := "MISSING"
			failures = append(failures, generated_v2.Failure{
				Arn:    &clusterName,
				Reason: &reason,
			})
			continue
		}

		result = append(result, *api.convertClusterToGenerated(cluster))
	}

	resp := &generated_v2.DescribeClustersResponse{
		Clusters: result,
	}

	if len(failures) > 0 {
		resp.Failures = failures
	}

	return resp, nil
}

// UpdateCluster implements the UpdateCluster operation using generated types
func (api *DefaultECSAPIGenerated) UpdateCluster(ctx context.Context, req *generated_v2.UpdateClusterRequest) (*generated_v2.UpdateClusterResponse, error) {
	// Debug log the request
	reqJSON, _ := json.Marshal(req)
	log.Printf("UpdateCluster request: %s", string(reqJSON))

	// Validate cluster name
	if req.Cluster == "" {
		return nil, fmt.Errorf("cluster name is required")
	}

	// Get existing cluster
	clusterStore := api.storage.ClusterStore()
	cluster, err := clusterStore.Get(ctx, req.Cluster)
	if err != nil {
		return nil, &ClusterNotFoundException{
			Message: fmt.Sprintf("Cluster not found: %s", req.Cluster),
		}
	}

	// Update cluster configuration
	if req.Configuration != nil {
		cluster.Configuration = api.convertConfigurationToJSON(req.Configuration)
	}

	// Update settings
	if req.Settings != nil {
		cluster.Settings = api.convertSettingsToJSON(req.Settings)
	}

	// Save updated cluster
	err = clusterStore.Update(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}

	return &generated_v2.UpdateClusterResponse{
		Cluster: api.convertClusterToGenerated(cluster),
	}, nil
}

// Helper methods for type conversion

func (api *DefaultECSAPIGenerated) buildCreateClusterResponse(cluster *storage.Cluster) *generated_v2.CreateClusterResponse {
	return &generated_v2.CreateClusterResponse{
		Cluster: api.convertClusterToGenerated(cluster),
	}
}

func (api *DefaultECSAPIGenerated) convertClusterToGenerated(cluster *storage.Cluster) *generated_v2.Cluster {
	if cluster == nil {
		return nil
	}

	result := &generated_v2.Cluster{
		ClusterArn:  &cluster.ARN,
		ClusterName: &cluster.Name,
		Status:      &cluster.Status,
		// Convert other fields as needed
		RegisteredContainerInstancesCount: ptrInt32(0),
		RunningTasksCount:                 ptrInt32(0),
		PendingTasksCount:                 ptrInt32(0),
		ActiveServicesCount:               ptrInt32(0),
	}

	// Parse and convert settings from JSON
	if cluster.Settings != "" {
		var settings []generated_v2.ClusterSetting
		if err := json.Unmarshal([]byte(cluster.Settings), &settings); err == nil {
			result.Settings = settings
		}
	}

	// Parse and convert tags from JSON
	if cluster.Tags != "" {
		var tags []generated_v2.Tag
		if err := json.Unmarshal([]byte(cluster.Tags), &tags); err == nil {
			result.Tags = tags
		}
	}

	return result
}

// JSON conversion helpers

func (api *DefaultECSAPIGenerated) convertSettingsToJSON(settings []generated_v2.ClusterSetting) string {
	if settings == nil || len(settings) == 0 {
		return ""
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return ""
	}
	return string(data)
}

func (api *DefaultECSAPIGenerated) convertTagsToJSON(tags []generated_v2.Tag) string {
	if tags == nil || len(tags) == 0 {
		return ""
	}
	data, err := json.Marshal(tags)
	if err != nil {
		return ""
	}
	return string(data)
}

func (api *DefaultECSAPIGenerated) convertConfigurationToJSON(config *generated_v2.ClusterConfiguration) string {
	if config == nil {
		return ""
	}
	data, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	return string(data)
}

// Helper functions
func ptrToString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func ptrToInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

func ptrToBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func ptrInt32(v int32) *int32 {
	return &v
}

// Error types
type ClusterNotFoundException struct {
	Message string
}

func (e *ClusterNotFoundException) Error() string {
	return e.Message
}

type ClusterContainsServicesException struct {
	Message string
}

func (e *ClusterContainsServicesException) Error() string {
	return e.Message
}

type ClusterContainsTasksException struct {
	Message string
}

func (e *ClusterContainsTasksException) Error() string {
	return e.Message
}

// Service operations stubs - TODO: Implement these

func (api *DefaultECSAPIGenerated) CreateService(ctx context.Context, req *generated_v2.CreateServiceRequest) (*generated_v2.CreateServiceResponse, error) {
	return &generated_v2.CreateServiceResponse{}, nil
}

func (api *DefaultECSAPIGenerated) ListServices(ctx context.Context, req *generated_v2.ListServicesRequest) (*generated_v2.ListServicesResponse, error) {
	return &generated_v2.ListServicesResponse{}, nil
}

func (api *DefaultECSAPIGenerated) DescribeServices(ctx context.Context, req *generated_v2.DescribeServicesRequest) (*generated_v2.DescribeServicesResponse, error) {
	return &generated_v2.DescribeServicesResponse{}, nil
}

func (api *DefaultECSAPIGenerated) UpdateService(ctx context.Context, req *generated_v2.UpdateServiceRequest) (*generated_v2.UpdateServiceResponse, error) {
	return &generated_v2.UpdateServiceResponse{}, nil
}

func (api *DefaultECSAPIGenerated) DeleteService(ctx context.Context, req *generated_v2.DeleteServiceRequest) (*generated_v2.DeleteServiceResponse, error) {
	return &generated_v2.DeleteServiceResponse{}, nil
}

// Task operations stubs - TODO: Implement these

func (api *DefaultECSAPIGenerated) RunTask(ctx context.Context, req *generated_v2.RunTaskRequest) (*generated_v2.RunTaskResponse, error) {
	return &generated_v2.RunTaskResponse{}, nil
}

func (api *DefaultECSAPIGenerated) StopTask(ctx context.Context, req *generated_v2.StopTaskRequest) (*generated_v2.StopTaskResponse, error) {
	return &generated_v2.StopTaskResponse{}, nil
}

func (api *DefaultECSAPIGenerated) DescribeTasks(ctx context.Context, req *generated_v2.DescribeTasksRequest) (*generated_v2.DescribeTasksResponse, error) {
	return &generated_v2.DescribeTasksResponse{}, nil
}

func (api *DefaultECSAPIGenerated) ListTasks(ctx context.Context, req *generated_v2.ListTasksRequest) (*generated_v2.ListTasksResponse, error) {
	return &generated_v2.ListTasksResponse{}, nil
}

// TaskDefinition operations stubs - TODO: Implement these

func (api *DefaultECSAPIGenerated) RegisterTaskDefinition(ctx context.Context, req *generated_v2.RegisterTaskDefinitionRequest) (*generated_v2.RegisterTaskDefinitionResponse, error) {
	return &generated_v2.RegisterTaskDefinitionResponse{}, nil
}

func (api *DefaultECSAPIGenerated) DeregisterTaskDefinition(ctx context.Context, req *generated_v2.DeregisterTaskDefinitionRequest) (*generated_v2.DeregisterTaskDefinitionResponse, error) {
	return &generated_v2.DeregisterTaskDefinitionResponse{}, nil
}

func (api *DefaultECSAPIGenerated) DescribeTaskDefinition(ctx context.Context, req *generated_v2.DescribeTaskDefinitionRequest) (*generated_v2.DescribeTaskDefinitionResponse, error) {
	return &generated_v2.DescribeTaskDefinitionResponse{}, nil
}

func (api *DefaultECSAPIGenerated) ListTaskDefinitionFamilies(ctx context.Context, req *generated_v2.ListTaskDefinitionFamiliesRequest) (*generated_v2.ListTaskDefinitionFamiliesResponse, error) {
	return &generated_v2.ListTaskDefinitionFamiliesResponse{}, nil
}

func (api *DefaultECSAPIGenerated) ListTaskDefinitions(ctx context.Context, req *generated_v2.ListTaskDefinitionsRequest) (*generated_v2.ListTaskDefinitionsResponse, error) {
	return &generated_v2.ListTaskDefinitionsResponse{}, nil
}

// Tag operations stubs - TODO: Implement these

func (api *DefaultECSAPIGenerated) TagResource(ctx context.Context, req *generated_v2.TagResourceRequest) (*generated_v2.TagResourceResponse, error) {
	return &generated_v2.TagResourceResponse{}, nil
}

func (api *DefaultECSAPIGenerated) UntagResource(ctx context.Context, req *generated_v2.UntagResourceRequest) (*generated_v2.UntagResourceResponse, error) {
	return &generated_v2.UntagResourceResponse{}, nil
}

func (api *DefaultECSAPIGenerated) ListTagsForResource(ctx context.Context, req *generated_v2.ListTagsForResourceRequest) (*generated_v2.ListTagsForResourceResponse, error) {
	return &generated_v2.ListTagsForResourceResponse{}, nil
}