package api

import (
	"context"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// RegisterContainerInstance implements the RegisterContainerInstance operation
func (api *DefaultECSAPI) RegisterContainerInstance(ctx context.Context, req *generated.RegisterContainerInstanceRequest) (*generated.RegisterContainerInstanceResponse, error) {
	// TODO: Implement actual container instance registration logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	containerInstanceArn := req.ContainerInstanceArn
	if containerInstanceArn == nil {
		arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":container-instance/" + cluster + "/i-1234567890abcdef0"
		containerInstanceArn = ptr.String(arn)
	}

	resp := &generated.RegisterContainerInstanceResponse{
		ContainerInstance: &generated.ContainerInstance{
			ContainerInstanceArn: containerInstanceArn,
			Ec2InstanceId:        ptr.String("i-1234567890abcdef0"),
			Version:              ptr.Int64(1),
			Status:               ptr.String("ACTIVE"),
			StatusReason:         ptr.String(""),
			AgentConnected:       ptr.Bool(true),
			RunningTasksCount:    ptr.Int32(0),
			PendingTasksCount:    ptr.Int32(0),
			AgentUpdateStatus:    (*generated.AgentUpdateStatus)(ptr.String("NOT_STAGED")),
			RegisteredAt:         ptr.UnixTime(time.Now()),
			RegisteredResources: []generated.Resource{
				{
					Name:         ptr.String("CPU"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(2048),
				},
				{
					Name:         ptr.String("MEMORY"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(4096),
				},
				{
					Name:           ptr.String("PORTS"),
					Type:           ptr.String("STRINGSET"),
					StringSetValue: []string{"22", "80", "443", "2376", "2375", "51678", "51679"},
				},
				{
					Name:           ptr.String("PORTS_UDP"),
					Type:           ptr.String("STRINGSET"),
					StringSetValue: []string{},
				},
			},
			RemainingResources: []generated.Resource{
				{
					Name:         ptr.String("CPU"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(2048),
				},
				{
					Name:         ptr.String("MEMORY"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(4096),
				},
				{
					Name:           ptr.String("PORTS"),
					Type:           ptr.String("STRINGSET"),
					StringSetValue: []string{"22", "80", "443", "2376", "2375", "51678", "51679"},
				},
				{
					Name:           ptr.String("PORTS_UDP"),
					Type:           ptr.String("STRINGSET"),
					StringSetValue: []string{},
				},
			},
			VersionInfo: req.VersionInfo,
			Attributes:  req.Attributes,
			Tags:        req.Tags,
		},
	}

	return resp, nil
}

// DeregisterContainerInstance implements the DeregisterContainerInstance operation
func (api *DefaultECSAPI) DeregisterContainerInstance(ctx context.Context, req *generated.DeregisterContainerInstanceRequest) (*generated.DeregisterContainerInstanceResponse, error) {
	// TODO: Implement actual container instance deregistration logic
	// For now, return a mock response
	resp := &generated.DeregisterContainerInstanceResponse{
		ContainerInstance: &generated.ContainerInstance{
			ContainerInstanceArn: ptr.String(req.ContainerInstance),
			Status:               ptr.String("INACTIVE"),
			StatusReason:         ptr.String("Instance deregistration forced"),
			AgentConnected:       ptr.Bool(false),
			RunningTasksCount:    ptr.Int32(0),
			PendingTasksCount:    ptr.Int32(0),
		},
	}

	return resp, nil
}

// DescribeContainerInstances implements the DescribeContainerInstances operation
func (api *DefaultECSAPI) DescribeContainerInstances(ctx context.Context, req *generated.DescribeContainerInstancesRequest) (*generated.DescribeContainerInstancesResponse, error) {
	// TODO: Implement actual container instance description logic
	// For now, return mock responses for requested instances
	containerInstances := []generated.ContainerInstance{}
	for i, arn := range req.ContainerInstances {
		containerInstances = append(containerInstances, generated.ContainerInstance{
			ContainerInstanceArn: ptr.String(arn),
			Ec2InstanceId:        ptr.String("i-1234567890abcdef" + string(rune('0'+i))),
			Version:              ptr.Int64(1),
			Status:               ptr.String("ACTIVE"),
			AgentConnected:       ptr.Bool(true),
			RunningTasksCount:    ptr.Int32(0),
			PendingTasksCount:    ptr.Int32(0),
			RegisteredAt:         ptr.UnixTime(time.Now().Add(-24 * time.Hour)),
			RegisteredResources: []generated.Resource{
				{
					Name:         ptr.String("CPU"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(2048),
				},
				{
					Name:         ptr.String("MEMORY"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(4096),
				},
			},
			RemainingResources: []generated.Resource{
				{
					Name:         ptr.String("CPU"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(2048),
				},
				{
					Name:         ptr.String("MEMORY"),
					Type:         ptr.String("INTEGER"),
					IntegerValue: ptr.Int32(4096),
				},
			},
			VersionInfo: &generated.VersionInfo{
				AgentVersion:  ptr.String("1.51.0"),
				AgentHash:     ptr.String("4023248"),
				DockerVersion: ptr.String("20.10.7"),
			},
		})
	}

	resp := &generated.DescribeContainerInstancesResponse{
		ContainerInstances: containerInstances,
		Failures:           []generated.Failure{},
	}

	return resp, nil
}

// ListContainerInstances implements the ListContainerInstances operation
func (api *DefaultECSAPI) ListContainerInstances(ctx context.Context, req *generated.ListContainerInstancesRequest) (*generated.ListContainerInstancesResponse, error) {
	// Get cluster name
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	// Get cluster to validate it exists and get its ARN
	clusterObj, err := api.storage.ClusterStore().Get(ctx, cluster)
	if err != nil {
		// If cluster not found, return empty result
		return &generated.ListContainerInstancesResponse{
			ContainerInstanceArns: []string{},
		}, nil
	}
	clusterARN := clusterObj.ARN

	// Set default limit if not specified
	limit := 100
	if req.MaxResults != nil && *req.MaxResults > 0 {
		limit = int(*req.MaxResults)
		// AWS ECS has a maximum of 100 results per page
		if limit > 100 {
			limit = 100
		}
	}

	// Extract next token
	var nextToken string
	if req.NextToken != nil {
		nextToken = *req.NextToken
	}

	// Build filters
	filters := storage.ContainerInstanceFilters{}
	if req.Status != nil {
		filters.Status = string(*req.Status)
	}
	if req.Filter != nil {
		filters.Filter = *req.Filter
	}

	// Get container instances with pagination
	instances, newNextToken, err := api.storage.ContainerInstanceStore().ListWithPagination(ctx, clusterARN, filters, limit, nextToken)
	if err != nil {
		return nil, fmt.Errorf("failed to list container instances: %w", err)
	}

	// Convert to ARNs
	// Initialize with empty slice to ensure it's not nil when marshaling to JSON
	containerInstanceArns := make([]string, 0, len(instances))
	for _, instance := range instances {
		containerInstanceArns = append(containerInstanceArns, instance.ARN)
	}

	resp := &generated.ListContainerInstancesResponse{
		ContainerInstanceArns: containerInstanceArns,
	}

	// Add next token if there are more results
	if newNextToken != "" {
		resp.NextToken = ptr.String(newNextToken)
	}

	return resp, nil
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

// SubmitContainerStateChange implements the SubmitContainerStateChange operation
func (api *DefaultECSAPI) SubmitContainerStateChange(ctx context.Context, req *generated.SubmitContainerStateChangeRequest) (*generated.SubmitContainerStateChangeResponse, error) {
	// TODO: Implement SubmitContainerStateChange
	return nil, fmt.Errorf("SubmitContainerStateChange not implemented")
}
