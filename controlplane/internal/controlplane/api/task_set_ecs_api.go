package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CreateTaskSet implements the CreateTaskSet operation
func (api *DefaultECSAPI) CreateTaskSet(ctx context.Context, req *generated.CreateTaskSetRequest) (*generated.CreateTaskSetResponse, error) {
	// Validate cluster and service
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	service := ""
	if req.Service != nil {
		service = *req.Service
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
		scale = &generated.Scale{
			Value: ptr.Float64(100.0),
			Unit:  (*generated.ScaleUnit)(ptr.String("PERCENT")),
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
		ExternalID:           ptr.ToString(req.ExternalId),
		TaskDefinition:       ptr.ToString(req.TaskDefinition),
		LaunchType:           ptr.ToString((*string)(req.LaunchType)),
		PlatformVersion:      ptr.ToString(req.PlatformVersion),
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

	// Create task set in storage
	if err := api.storage.TaskSetStore().Create(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to create task set: %w", err)
	}

	// Build response
	resp := &generated.CreateTaskSetResponse{
		TaskSet: &generated.TaskSet{
			Id:                       ptr.String(taskSetId),
			TaskSetArn:               ptr.String(taskSetARN),
			ServiceArn:               ptr.String(serviceARN),
			ClusterArn:               ptr.String(clusterARN),
			ExternalId:               req.ExternalId,
			Status:                   ptr.String("ACTIVE"),
			TaskDefinition:           req.TaskDefinition,
			LaunchType:               req.LaunchType,
			Scale:                    scale,
			StabilityStatus:          (*generated.StabilityStatus)(ptr.String("STEADY_STATE")),
			CreatedAt:                ptr.Time(storageTaskSet.CreatedAt),
			LoadBalancers:            req.LoadBalancers,
			ServiceRegistries:        req.ServiceRegistries,
			NetworkConfiguration:     req.NetworkConfiguration,
			CapacityProviderStrategy: req.CapacityProviderStrategy,
			PlatformVersion:          req.PlatformVersion,
			Tags:                     req.Tags,
			ComputedDesiredCount:     ptr.Int32(0),
			PendingCount:             ptr.Int32(0),
			RunningCount:             ptr.Int32(0),
		},
	}

	return resp, nil
}

// DeleteTaskSet implements the DeleteTaskSet operation
func (api *DefaultECSAPI) DeleteTaskSet(ctx context.Context, req *generated.DeleteTaskSetRequest) (*generated.DeleteTaskSetResponse, error) {
	// Validate required fields
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	service := ""
	if req.Service != nil {
		service = *req.Service
	}

	taskSet := ""
	if req.TaskSet != nil {
		taskSet = *req.TaskSet
	}

	if service == "" || taskSet == "" {
		return nil, fmt.Errorf("service and taskSet are required")
	}

	// Build service ARN
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Get task set from storage
	storageTaskSet, err := api.storage.TaskSetStore().Get(ctx, serviceARN, taskSet)
	if err != nil {
		return nil, fmt.Errorf("task set not found: %s", taskSet)
	}

	// Update status to DRAINING
	storageTaskSet.Status = "DRAINING"
	if err := api.storage.TaskSetStore().Update(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to update task set status: %w", err)
	}

	// Build response
	resp := &generated.DeleteTaskSetResponse{
		TaskSet: &generated.TaskSet{
			Id:         ptr.String(storageTaskSet.ID),
			TaskSetArn: ptr.String(storageTaskSet.ARN),
			ServiceArn: ptr.String(storageTaskSet.ServiceARN),
			ClusterArn: ptr.String(storageTaskSet.ClusterARN),
			Status:     ptr.String("DRAINING"),
			UpdatedAt:  ptr.Time(storageTaskSet.UpdatedAt),
		},
	}

	return resp, nil
}

// DescribeTaskSets implements the DescribeTaskSets operation
func (api *DefaultECSAPI) DescribeTaskSets(ctx context.Context, req *generated.DescribeTaskSetsRequest) (*generated.DescribeTaskSetsResponse, error) {
	// Validate required fields
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	service := ""
	if req.Service != nil {
		service = *req.Service
	}

	if service == "" {
		return nil, fmt.Errorf("service is required")
	}

	// Build service ARN
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Get task sets from storage
	storageTaskSets, err := api.storage.TaskSetStore().List(ctx, serviceARN, req.TaskSets)
	if err != nil {
		return nil, fmt.Errorf("failed to list task sets: %w", err)
	}

	// Convert to API response
	taskSets := []generated.TaskSet{}
	failures := []generated.Failure{}

	for _, ts := range storageTaskSets {
		apiTaskSet := generated.TaskSet{
			Id:                   ptr.String(ts.ID),
			TaskSetArn:           ptr.String(ts.ARN),
			ServiceArn:           ptr.String(ts.ServiceARN),
			ClusterArn:           ptr.String(ts.ClusterARN),
			Status:               ptr.String(ts.Status),
			TaskDefinition:       ptr.String(ts.TaskDefinition),
			ComputedDesiredCount: ptr.Int32(ts.ComputedDesiredCount),
			PendingCount:         ptr.Int32(ts.PendingCount),
			RunningCount:         ptr.Int32(ts.RunningCount),
			StabilityStatus:      (*generated.StabilityStatus)(ptr.String(ts.StabilityStatus)),
			CreatedAt:            ptr.Time(ts.CreatedAt),
			UpdatedAt:            ptr.Time(ts.UpdatedAt),
		}

		// Set launch type if specified
		if ts.LaunchType != "" {
			apiTaskSet.LaunchType = (*generated.LaunchType)(ptr.String(ts.LaunchType))
		}

		// Set platform version if specified
		if ts.PlatformVersion != "" {
			apiTaskSet.PlatformVersion = ptr.String(ts.PlatformVersion)
		}

		// Set external ID if specified
		if ts.ExternalID != "" {
			apiTaskSet.ExternalId = ptr.String(ts.ExternalID)
		}

		// Unmarshal complex fields
		if ts.Scale != "" {
			var scale generated.Scale
			if err := json.Unmarshal([]byte(ts.Scale), &scale); err == nil {
				apiTaskSet.Scale = &scale
			}
		}

		if ts.NetworkConfiguration != "" {
			var nc generated.NetworkConfiguration
			if err := json.Unmarshal([]byte(ts.NetworkConfiguration), &nc); err == nil {
				apiTaskSet.NetworkConfiguration = &nc
			}
		}

		if ts.LoadBalancers != "" {
			var lbs []generated.LoadBalancer
			if err := json.Unmarshal([]byte(ts.LoadBalancers), &lbs); err == nil {
				apiTaskSet.LoadBalancers = lbs
			}
		}

		if ts.ServiceRegistries != "" {
			var srs []generated.ServiceRegistry
			if err := json.Unmarshal([]byte(ts.ServiceRegistries), &srs); err == nil {
				apiTaskSet.ServiceRegistries = srs
			}
		}

		if ts.CapacityProviderStrategy != "" {
			var cps []generated.CapacityProviderStrategyItem
			if err := json.Unmarshal([]byte(ts.CapacityProviderStrategy), &cps); err == nil {
				apiTaskSet.CapacityProviderStrategy = cps
			}
		}

		if ts.Tags != "" {
			var tags []generated.Tag
			if err := json.Unmarshal([]byte(ts.Tags), &tags); err == nil {
				apiTaskSet.Tags = tags
			}
		}

		taskSets = append(taskSets, apiTaskSet)
	}

	// If specific task sets were requested but not found, add to failures
	if len(req.TaskSets) > 0 && len(taskSets) < len(req.TaskSets) {
		foundIDs := make(map[string]bool)
		for _, ts := range taskSets {
			foundIDs[*ts.Id] = true
		}
		for _, requestedID := range req.TaskSets {
			if !foundIDs[requestedID] {
				failures = append(failures, generated.Failure{
					Arn:    ptr.String(fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", api.region, api.accountID, cluster, service, requestedID)),
					Reason: ptr.String("MISSING"),
					Detail: ptr.String("Task set not found"),
				})
			}
		}
	}

	resp := &generated.DescribeTaskSetsResponse{
		TaskSets: taskSets,
		Failures: failures,
	}

	return resp, nil
}

// UpdateTaskSet implements the UpdateTaskSet operation
func (api *DefaultECSAPI) UpdateTaskSet(ctx context.Context, req *generated.UpdateTaskSetRequest) (*generated.UpdateTaskSetResponse, error) {
	// Validate required fields
	cluster := "default"
	if req.Cluster != nil {
		cluster = *req.Cluster
	}

	service := ""
	if req.Service != nil {
		service = *req.Service
	}

	taskSet := ""
	if req.TaskSet != nil {
		taskSet = *req.TaskSet
	}

	if service == "" || taskSet == "" || req.Scale == nil {
		return nil, fmt.Errorf("service, taskSet, and scale are required")
	}

	// Build service ARN
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Get task set from storage
	storageTaskSet, err := api.storage.TaskSetStore().Get(ctx, serviceARN, taskSet)
	if err != nil {
		return nil, fmt.Errorf("task set not found: %s", taskSet)
	}

	// Update scale
	if data, err := json.Marshal(req.Scale); err == nil {
		storageTaskSet.Scale = string(data)
	}
	storageTaskSet.StabilityStatus = "STABILIZING"
	if err := api.storage.TaskSetStore().Update(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to update task set: %w", err)
	}

	// Build response
	apiTaskSet := &generated.TaskSet{
		Id:                   ptr.String(storageTaskSet.ID),
		TaskSetArn:           ptr.String(storageTaskSet.ARN),
		ServiceArn:           ptr.String(storageTaskSet.ServiceARN),
		ClusterArn:           ptr.String(storageTaskSet.ClusterARN),
		Status:               ptr.String(storageTaskSet.Status),
		TaskDefinition:       ptr.String(storageTaskSet.TaskDefinition),
		Scale:                req.Scale,
		StabilityStatus:      (*generated.StabilityStatus)(ptr.String("STABILIZING")),
		UpdatedAt:            ptr.Time(storageTaskSet.UpdatedAt),
		ComputedDesiredCount: ptr.Int32(storageTaskSet.ComputedDesiredCount),
		PendingCount:         ptr.Int32(storageTaskSet.PendingCount),
		RunningCount:         ptr.Int32(storageTaskSet.RunningCount),
	}

	// Set launch type if specified
	if storageTaskSet.LaunchType != "" {
		apiTaskSet.LaunchType = (*generated.LaunchType)(ptr.String(storageTaskSet.LaunchType))
	}

	// Set platform version if specified
	if storageTaskSet.PlatformVersion != "" {
		apiTaskSet.PlatformVersion = ptr.String(storageTaskSet.PlatformVersion)
	}

	resp := &generated.UpdateTaskSetResponse{
		TaskSet: apiTaskSet,
	}

	return resp, nil
}
