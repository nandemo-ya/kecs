package api

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// RegisterContainerInstanceV2 implements the RegisterContainerInstance operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) RegisterContainerInstanceV2(ctx context.Context, req *ecs.RegisterContainerInstanceInput) (*ecs.RegisterContainerInstanceOutput, error) {
	// TODO: Implement actual container instance registration logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	containerInstanceArn := req.ContainerInstanceArn
	if containerInstanceArn == nil {
		arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":container-instance/" + cluster + "/i-1234567890abcdef0"
		containerInstanceArn = aws.String(arn)
	}

	return &ecs.RegisterContainerInstanceOutput{
		ContainerInstance: &types.ContainerInstance{
			ContainerInstanceArn: containerInstanceArn,
			Ec2InstanceId:        aws.String("i-1234567890abcdef0"),
			Version:              1,
			Status:               aws.String("ACTIVE"),
			StatusReason:         aws.String(""),
			AgentConnected:       true,
			RunningTasksCount:    0,
			PendingTasksCount:    0,
			AgentUpdateStatus:    types.AgentUpdateStatusPending,
			RegisteredAt:         aws.Time(time.Now()),
			RegisteredResources: []types.Resource{
				{
					Name:         aws.String("CPU"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 2048,
				},
				{
					Name:         aws.String("MEMORY"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 4096,
				},
				{
					Name:           aws.String("PORTS"),
					Type:           aws.String("STRINGSET"),
					StringSetValue: []string{"22", "80", "443", "2376", "2375", "51678", "51679"},
				},
				{
					Name:           aws.String("PORTS_UDP"),
					Type:           aws.String("STRINGSET"),
					StringSetValue: []string{},
				},
			},
			RemainingResources: []types.Resource{
				{
					Name:         aws.String("CPU"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 2048,
				},
				{
					Name:         aws.String("MEMORY"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 4096,
				},
				{
					Name:           aws.String("PORTS"),
					Type:           aws.String("STRINGSET"),
					StringSetValue: []string{"22", "80", "443", "2376", "2375", "51678", "51679"},
				},
				{
					Name:           aws.String("PORTS_UDP"),
					Type:           aws.String("STRINGSET"),
					StringSetValue: []string{},
				},
			},
			VersionInfo: req.VersionInfo,
			Attributes:  req.Attributes,
			Tags:        req.Tags,
		},
	}, nil
}

// DeregisterContainerInstanceV2 implements the DeregisterContainerInstance operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DeregisterContainerInstanceV2(ctx context.Context, req *ecs.DeregisterContainerInstanceInput) (*ecs.DeregisterContainerInstanceOutput, error) {
	// TODO: Implement actual container instance deregistration logic
	// For now, return a mock response
	return &ecs.DeregisterContainerInstanceOutput{
		ContainerInstance: &types.ContainerInstance{
			ContainerInstanceArn: req.ContainerInstance,
			Status:               aws.String("INACTIVE"),
			StatusReason:         aws.String("Instance deregistration forced"),
			AgentConnected:       false,
			RunningTasksCount:    0,
			PendingTasksCount:    0,
		},
	}, nil
}

// DescribeContainerInstancesV2 implements the DescribeContainerInstances operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DescribeContainerInstancesV2(ctx context.Context, req *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error) {
	// TODO: Implement actual container instance description logic
	// For now, return mock responses for requested instances
	var containerInstances []types.ContainerInstance
	for i, arn := range req.ContainerInstances {
		containerInstances = append(containerInstances, types.ContainerInstance{
			ContainerInstanceArn: aws.String(arn),
			Ec2InstanceId:        aws.String(fmt.Sprintf("i-1234567890abcdef%d", i)),
			Version:              1,
			Status:               aws.String("ACTIVE"),
			AgentConnected:       true,
			RunningTasksCount:    0,
			PendingTasksCount:    0,
			RegisteredAt:         aws.Time(time.Now().Add(-24 * time.Hour)),
			RegisteredResources: []types.Resource{
				{
					Name:         aws.String("CPU"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 2048,
				},
				{
					Name:         aws.String("MEMORY"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 4096,
				},
			},
			RemainingResources: []types.Resource{
				{
					Name:         aws.String("CPU"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 2048,
				},
				{
					Name:         aws.String("MEMORY"),
					Type:         aws.String("INTEGER"),
					IntegerValue: 4096,
				},
			},
			VersionInfo: &types.VersionInfo{
				AgentVersion:  aws.String("1.51.0"),
				AgentHash:     aws.String("4023248"),
				DockerVersion: aws.String("20.10.7"),
			},
		})
	}

	return &ecs.DescribeContainerInstancesOutput{
		ContainerInstances: containerInstances,
		Failures:           []types.Failure{},
	}, nil
}

// ListContainerInstancesV2 implements the ListContainerInstances operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) ListContainerInstancesV2(ctx context.Context, req *ecs.ListContainerInstancesInput) (*ecs.ListContainerInstancesOutput, error) {
	// Get cluster name
	cluster := "default"
	if req.Cluster != nil {
		cluster = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster to validate it exists and get its ARN
	clusterObj, err := api.storage.ClusterStore().Get(ctx, cluster)
	if err != nil {
		// If cluster not found, return empty result
		return &ecs.ListContainerInstancesOutput{
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
	if req.Status != "" {
		filters.Status = string(req.Status)
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

	resp := &ecs.ListContainerInstancesOutput{
		ContainerInstanceArns: containerInstanceArns,
	}

	// Add next token if there are more results
	if newNextToken != "" {
		resp.NextToken = aws.String(newNextToken)
	}

	return resp, nil
}

// UpdateContainerAgentV2 implements the UpdateContainerAgent operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) UpdateContainerAgentV2(ctx context.Context, req *ecs.UpdateContainerAgentInput) (*ecs.UpdateContainerAgentOutput, error) {
	// TODO: Implement UpdateContainerAgent
	return nil, fmt.Errorf("UpdateContainerAgent not implemented")
}

// UpdateContainerInstancesStateV2 implements the UpdateContainerInstancesState operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) UpdateContainerInstancesStateV2(ctx context.Context, req *ecs.UpdateContainerInstancesStateInput) (*ecs.UpdateContainerInstancesStateOutput, error) {
	// TODO: Implement UpdateContainerInstancesState
	return nil, fmt.Errorf("UpdateContainerInstancesState not implemented")
}

// SubmitContainerStateChangeV2 implements the SubmitContainerStateChange operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) SubmitContainerStateChangeV2(ctx context.Context, req *ecs.SubmitContainerStateChangeInput) (*ecs.SubmitContainerStateChangeOutput, error) {
	// TODO: Implement SubmitContainerStateChange
	return nil, fmt.Errorf("SubmitContainerStateChange not implemented")
}