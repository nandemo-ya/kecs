package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CreateTaskSetV2 implements the CreateTaskSet operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) CreateTaskSetV2(ctx context.Context, req *ecs.CreateTaskSetInput) (*ecs.CreateTaskSetOutput, error) {
	// Validate cluster and service
	cluster := "default"
	if req.Cluster != nil {
		cluster = extractClusterNameFromARN(*req.Cluster)
	}

	service := ""
	if req.Service != nil {
		service = extractServiceNameFromARN(*req.Service)
	}

	if service == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Build ARNs
	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, cluster)
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Verify service exists
	_, err := api.storage.ServiceStore().GetByARN(ctx, serviceARN)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", service)
	}

	// Default scale if not provided
	scale := req.Scale
	if scale == nil {
		scale = &types.Scale{
			Value: 100.0,
			Unit:  types.ScaleUnitPercent,
		}
	}

	// Generate task set ID
	taskSetId := "ts-" + uuid.New().String()[:8]
	taskSetARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", api.region, api.accountID, cluster, service, taskSetId)

	// Create storage object
	storageTaskSet := &storage.TaskSet{
		ID:                   taskSetId,
		ARN:                  taskSetARN,
		ServiceARN:           serviceARN,
		ClusterARN:           clusterARN,
		ExternalID:           aws.ToString(req.ExternalId),
		TaskDefinition:       aws.ToString(req.TaskDefinition),
		LaunchType:           string(req.LaunchType),
		PlatformVersion:      aws.ToString(req.PlatformVersion),
		Status:               "ACTIVE",
		StabilityStatus:      "STEADY_STATE",
		ComputedDesiredCount: 0, // Will be computed based on service desired count and scale
		PendingCount:         0,
		RunningCount:         0,
		Region:               api.region,
		AccountID:            api.accountID,
	}

	// Marshal complex fields to JSON
	if req.NetworkConfiguration != nil {
		if data, err := json.Marshal(req.NetworkConfiguration); err == nil {
			storageTaskSet.NetworkConfiguration = string(data)
		}
	}
	if len(req.LoadBalancers) > 0 {
		if data, err := json.Marshal(req.LoadBalancers); err == nil {
			storageTaskSet.LoadBalancers = string(data)
		}
	}
	if len(req.ServiceRegistries) > 0 {
		if data, err := json.Marshal(req.ServiceRegistries); err == nil {
			storageTaskSet.ServiceRegistries = string(data)
		}
	}
	if len(req.CapacityProviderStrategy) > 0 {
		if data, err := json.Marshal(req.CapacityProviderStrategy); err == nil {
			storageTaskSet.CapacityProviderStrategy = string(data)
		}
	}
	if scale != nil {
		if data, err := json.Marshal(scale); err == nil {
			storageTaskSet.Scale = string(data)
		}
	}
	if len(req.Tags) > 0 {
		if data, err := json.Marshal(req.Tags); err == nil {
			storageTaskSet.Tags = string(data)
		}
	}

	// Store task set
	if err := api.storage.TaskSetStore().Create(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to create task set: %w", err)
	}

	// Convert storage object to AWS SDK type
	taskSet := storageTaskSetToSDK(storageTaskSet)

	return &ecs.CreateTaskSetOutput{
		TaskSet: taskSet,
	}, nil
}

// DeleteTaskSetV2 implements the DeleteTaskSet operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DeleteTaskSetV2(ctx context.Context, req *ecs.DeleteTaskSetInput) (*ecs.DeleteTaskSetOutput, error) {
	// Extract identifiers
	cluster := "default"
	if req.Cluster != nil {
		cluster = extractClusterNameFromARN(*req.Cluster)
	}

	service := ""
	if req.Service != nil {
		service = extractServiceNameFromARN(*req.Service)
	}

	taskSet := ""
	if req.TaskSet != nil {
		taskSet = *req.TaskSet
	}

	if service == "" || taskSet == "" {
		return nil, fmt.Errorf("service and taskSet are required")
	}

	// Build ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Get task set
	storageTaskSet, err := api.storage.TaskSetStore().Get(ctx, serviceARN, taskSet)
	if err != nil {
		return nil, fmt.Errorf("task set not found: %s", taskSet)
	}

	// Update status to DRAINING
	storageTaskSet.Status = "DRAINING"
	if err := api.storage.TaskSetStore().Update(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to update task set status: %w", err)
	}

	// Delete if force is true
	if req.Force != nil && *req.Force {
		if err := api.storage.TaskSetStore().Delete(ctx, serviceARN, taskSet); err != nil {
			return nil, fmt.Errorf("failed to delete task set: %w", err)
		}
		storageTaskSet.Status = "INACTIVE"
	}

	// Convert to SDK type
	taskSetResp := storageTaskSetToSDK(storageTaskSet)

	return &ecs.DeleteTaskSetOutput{
		TaskSet: taskSetResp,
	}, nil
}

// DescribeTaskSetsV2 implements the DescribeTaskSets operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DescribeTaskSetsV2(ctx context.Context, req *ecs.DescribeTaskSetsInput) (*ecs.DescribeTaskSetsOutput, error) {
	// Extract identifiers
	cluster := "default"
	if req.Cluster != nil {
		cluster = extractClusterNameFromARN(*req.Cluster)
	}

	service := ""
	if req.Service != nil {
		service = extractServiceNameFromARN(*req.Service)
	}

	if service == "" {
		return nil, fmt.Errorf("service is required")
	}

	// Build ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	var taskSets []types.TaskSet
	var failures []types.Failure

	if len(req.TaskSets) > 0 {
		// Get specific task sets
		for _, taskSetId := range req.TaskSets {
			taskSet, err := api.storage.TaskSetStore().Get(ctx, serviceARN, taskSetId)
			if err != nil {
				failures = append(failures, types.Failure{
					Arn:    aws.String(taskSetId),
					Reason: aws.String("MISSING"),
					Detail: aws.String("Task set not found"),
				})
				continue
			}
			taskSets = append(taskSets, *storageTaskSetToSDK(taskSet))
		}
	} else {
		// List all task sets for the service
		storageTaskSets, err := api.storage.TaskSetStore().List(ctx, serviceARN, []string{})
		if err != nil {
			return nil, fmt.Errorf("failed to list task sets: %w", err)
		}
		for _, taskSet := range storageTaskSets {
			taskSets = append(taskSets, *storageTaskSetToSDK(taskSet))
		}
	}

	return &ecs.DescribeTaskSetsOutput{
		TaskSets: taskSets,
		Failures: failures,
	}, nil
}

// UpdateTaskSetV2 implements the UpdateTaskSet operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) UpdateTaskSetV2(ctx context.Context, req *ecs.UpdateTaskSetInput) (*ecs.UpdateTaskSetOutput, error) {
	// Extract identifiers
	cluster := "default"
	if req.Cluster != nil {
		cluster = extractClusterNameFromARN(*req.Cluster)
	}

	service := ""
	if req.Service != nil {
		service = extractServiceNameFromARN(*req.Service)
	}

	taskSet := ""
	if req.TaskSet != nil {
		taskSet = *req.TaskSet
	}

	if service == "" || taskSet == "" {
		return nil, fmt.Errorf("service and taskSet are required")
	}

	// Build ARNs
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Get task set
	storageTaskSet, err := api.storage.TaskSetStore().Get(ctx, serviceARN, taskSet)
	if err != nil {
		return nil, fmt.Errorf("task set not found: %s", taskSet)
	}

	// Update scale if provided
	if req.Scale != nil {
		if data, err := json.Marshal(req.Scale); err == nil {
			storageTaskSet.Scale = string(data)
		}
	}

	// Update the task set
	if err := api.storage.TaskSetStore().Update(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to update task set: %w", err)
	}

	// Convert to SDK type
	taskSetResp := storageTaskSetToSDK(storageTaskSet)

	return &ecs.UpdateTaskSetOutput{
		TaskSet: taskSetResp,
	}, nil
}

// UpdateServicePrimaryTaskSetV2 implements the UpdateServicePrimaryTaskSet operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) UpdateServicePrimaryTaskSetV2(ctx context.Context, req *ecs.UpdateServicePrimaryTaskSetInput) (*ecs.UpdateServicePrimaryTaskSetOutput, error) {
	// TODO: Implement UpdateServicePrimaryTaskSet
	return nil, fmt.Errorf("UpdateServicePrimaryTaskSet not implemented")
}

// Helper function to convert storage TaskSet to SDK type
func storageTaskSetToSDK(taskSet *storage.TaskSet) *types.TaskSet {
	if taskSet == nil {
		return nil
	}

	sdkTaskSet := &types.TaskSet{
		Id:                    aws.String(taskSet.ID),
		TaskSetArn:            aws.String(taskSet.ARN),
		ServiceArn:            aws.String(taskSet.ServiceARN),
		ClusterArn:            aws.String(taskSet.ClusterARN),
		ExternalId:            aws.String(taskSet.ExternalID),
		TaskDefinition:        aws.String(taskSet.TaskDefinition),
		Status:                aws.String(taskSet.Status),
		StabilityStatus:       types.StabilityStatus(taskSet.StabilityStatus),
		ComputedDesiredCount:  int32(taskSet.ComputedDesiredCount),
		PendingCount:          int32(taskSet.PendingCount),
		RunningCount:          int32(taskSet.RunningCount),
		CreatedAt:             aws.Time(taskSet.CreatedAt),
		UpdatedAt:             aws.Time(taskSet.UpdatedAt),
	}

	// Set launch type if not empty
	if taskSet.LaunchType != "" {
		sdkTaskSet.LaunchType = types.LaunchType(taskSet.LaunchType)
	}

	// Set platform version if not empty
	if taskSet.PlatformVersion != "" {
		sdkTaskSet.PlatformVersion = aws.String(taskSet.PlatformVersion)
	}

	// Unmarshal complex fields
	if taskSet.NetworkConfiguration != "" {
		var networkConfig types.NetworkConfiguration
		if err := json.Unmarshal([]byte(taskSet.NetworkConfiguration), &networkConfig); err == nil {
			sdkTaskSet.NetworkConfiguration = &networkConfig
		}
	}
	if taskSet.LoadBalancers != "" {
		var loadBalancers []types.LoadBalancer
		if err := json.Unmarshal([]byte(taskSet.LoadBalancers), &loadBalancers); err == nil {
			sdkTaskSet.LoadBalancers = loadBalancers
		}
	}
	if taskSet.ServiceRegistries != "" {
		var serviceRegistries []types.ServiceRegistry
		if err := json.Unmarshal([]byte(taskSet.ServiceRegistries), &serviceRegistries); err == nil {
			sdkTaskSet.ServiceRegistries = serviceRegistries
		}
	}
	if taskSet.CapacityProviderStrategy != "" {
		var strategy []types.CapacityProviderStrategyItem
		if err := json.Unmarshal([]byte(taskSet.CapacityProviderStrategy), &strategy); err == nil {
			sdkTaskSet.CapacityProviderStrategy = strategy
		}
	}
	if taskSet.Scale != "" {
		var scale types.Scale
		if err := json.Unmarshal([]byte(taskSet.Scale), &scale); err == nil {
			sdkTaskSet.Scale = &scale
		}
	}
	if taskSet.Tags != "" {
		var tags []types.Tag
		if err := json.Unmarshal([]byte(taskSet.Tags), &tags); err == nil {
			sdkTaskSet.Tags = tags
		}
	}

	return sdkTaskSet
}