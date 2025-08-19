package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CreateTaskSet implements the CreateTaskSet operation
func (api *DefaultECSAPI) CreateTaskSet(ctx context.Context, req *generated.CreateTaskSetRequest) (*generated.CreateTaskSetResponse, error) {
	// Validate cluster and service
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	service := req.Service

	if service == "" {
		return nil, fmt.Errorf("service name is required")
	}

	// Build ARNs
	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", api.region, api.accountID, cluster)
	serviceARN := fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", api.region, api.accountID, cluster, service)

	// Verify service exists and get desired count
	serviceObj, err := api.storage.ServiceStore().GetByARN(ctx, serviceARN)
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

	// Calculate computed desired count based on service desired count and scale
	computedDesiredCount := int32(0)
	if scale.Value != nil && scale.Unit != nil {
		switch *scale.Unit {
		case generated.ScaleUnit("PERCENT"):
			computedDesiredCount = int32(float64(serviceObj.DesiredCount) * (*scale.Value / 100.0))
		case generated.ScaleUnit("COUNT"):
			computedDesiredCount = int32(*scale.Value)
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
		TaskDefinition:       req.TaskDefinition,
		LaunchType:           ptr.ToString((*string)(req.LaunchType)),
		PlatformVersion:      ptr.ToString(req.PlatformVersion),
		Status:               "ACTIVE",
		StabilityStatus:      "STEADY_STATE",
		ComputedDesiredCount: computedDesiredCount,
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

	// Create TaskSet in Kubernetes if manager is available
	if api.taskSetManager != nil && serviceObj != nil {
		// Get task definition from storage
		taskDefIdentifier := req.TaskDefinition
		var taskDef *storage.TaskDefinition
		
		// Try to get by ARN first
		if strings.HasPrefix(taskDefIdentifier, "arn:") {
			taskDef, err = api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefIdentifier)
		} else {
			// Try to parse as family:revision format
			parts := strings.Split(taskDefIdentifier, ":")
			if len(parts) == 2 {
				family := parts[0]
				revision := 0
				fmt.Sscanf(parts[1], "%d", &revision)
				taskDef, err = api.storage.TaskDefinitionStore().Get(ctx, family, revision)
			} else {
				// Try as family name (get latest)
				taskDef, err = api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefIdentifier)
			}
		}
		
		if err != nil {
			// If not found, log warning
			// TaskSet will be created in storage but not in Kubernetes
			fmt.Printf("Warning: Task definition not found: %s\n", taskDefIdentifier)
		}
		
		if err == nil && taskDef != nil {
			// Create TaskSet in Kubernetes
			if err := api.taskSetManager.CreateTaskSet(ctx, storageTaskSet, serviceObj, taskDef, cluster); err != nil {
				// Log error but don't fail the API call
				// TaskSet is already created in storage
				fmt.Printf("Warning: Failed to create TaskSet in Kubernetes: %v\n", err)
			}
		}
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
			TaskDefinition:           ptr.String(req.TaskDefinition),
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
			ComputedDesiredCount:     ptr.Int32(computedDesiredCount),
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
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	service := req.Service
	taskSet := req.TaskSet

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

	// Delete TaskSet from Kubernetes if manager is available
	if api.taskSetManager != nil {
		// Get service from storage
		serviceObj, err := api.storage.ServiceStore().GetByARN(ctx, serviceARN)
		if err == nil && serviceObj != nil {
			// Delete TaskSet from Kubernetes
			force := req.Force != nil && *req.Force
			if err := api.taskSetManager.DeleteTaskSet(ctx, storageTaskSet, serviceObj, cluster, force); err != nil {
				// Log error but don't fail the API call
				fmt.Printf("Warning: Failed to delete TaskSet from Kubernetes: %v\n", err)
			}
		}
	}

	// Delete from storage after Kubernetes deletion
	if err := api.storage.TaskSetStore().Delete(ctx, serviceARN, taskSet); err != nil {
		// Log warning but don't fail since we already updated status to DRAINING
		fmt.Printf("Warning: Failed to delete TaskSet from storage: %v\n", err)
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
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	service := req.Service

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
		// Get real-time status from Kubernetes if available
		runningCount := ts.RunningCount
		pendingCount := ts.PendingCount
		stabilityStatus := ts.StabilityStatus
		
		if api.taskSetManager != nil {
			// Get service from storage
			serviceObj, err := api.storage.ServiceStore().GetByARN(ctx, serviceARN)
			if err == nil && serviceObj != nil {
				// Get TaskSet status from Kubernetes
				k8sRunning, k8sPending, k8sStability, err := api.taskSetManager.GetTaskSetStatus(ctx, ts, serviceObj, cluster)
				if err == nil {
					runningCount = int32(k8sRunning)
					pendingCount = int32(k8sPending)
					if k8sStability != "" {
						stabilityStatus = k8sStability
					}
				}
			}
		}
		
		apiTaskSet := generated.TaskSet{
			Id:                   ptr.String(ts.ID),
			TaskSetArn:           ptr.String(ts.ARN),
			ServiceArn:           ptr.String(ts.ServiceARN),
			ClusterArn:           ptr.String(ts.ClusterARN),
			Status:               ptr.String(ts.Status),
			TaskDefinition:       ptr.String(ts.TaskDefinition),
			ComputedDesiredCount: ptr.Int32(ts.ComputedDesiredCount),
			PendingCount:         ptr.Int32(pendingCount),
			RunningCount:         ptr.Int32(runningCount),
			StabilityStatus:      (*generated.StabilityStatus)(ptr.String(stabilityStatus)),
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
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	service := req.Service
	taskSet := req.TaskSet

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

	// Get service to recalculate computedDesiredCount
	serviceObj, err := api.storage.ServiceStore().GetByARN(ctx, serviceARN)
	if err != nil {
		return nil, fmt.Errorf("service not found: %s", service)
	}

	// Update scale and recompute desired count
	if data, err := json.Marshal(req.Scale); err == nil {
		storageTaskSet.Scale = string(data)
	}
	
	// Calculate new computed desired count
	if req.Scale.Value != nil && req.Scale.Unit != nil {
		switch *req.Scale.Unit {
		case generated.ScaleUnit("PERCENT"):
			storageTaskSet.ComputedDesiredCount = int32(float64(serviceObj.DesiredCount) * (*req.Scale.Value / 100.0))
		case generated.ScaleUnit("COUNT"):
			storageTaskSet.ComputedDesiredCount = int32(*req.Scale.Value)
		}
	}
	
	storageTaskSet.StabilityStatus = "STABILIZING"
	if err := api.storage.TaskSetStore().Update(ctx, storageTaskSet); err != nil {
		return nil, fmt.Errorf("failed to update task set: %w", err)
	}

	// Update TaskSet in Kubernetes if manager is available
	if api.taskSetManager != nil && serviceObj != nil {
		// Update TaskSet in Kubernetes
		if err := api.taskSetManager.UpdateTaskSet(ctx, storageTaskSet, serviceObj, cluster); err != nil {
			// Log error but don't fail the API call
			fmt.Printf("Warning: Failed to update TaskSet in Kubernetes: %v\n", err)
		}
	}

	// Build response
	apiTaskSet := &generated.TaskSet{
		Id:                   ptr.String(storageTaskSet.ID),
		TaskSetArn:           ptr.String(storageTaskSet.ARN),
		ServiceArn:           ptr.String(storageTaskSet.ServiceARN),
		ClusterArn:           ptr.String(storageTaskSet.ClusterARN),
		Status:               ptr.String(storageTaskSet.Status),
		TaskDefinition:       ptr.String(storageTaskSet.TaskDefinition),
		Scale:                &req.Scale,
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
