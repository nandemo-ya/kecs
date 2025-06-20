package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// RunTaskV2 implements the RunTask operation using AWS SDK types
func (api *DefaultECSAPIV2) RunTaskV2(ctx context.Context, req *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	// Validate required fields
	if req.TaskDefinition == nil || *req.TaskDefinition == "" {
		return nil, fmt.Errorf("taskDefinition is required")
	}

	// Get cluster name (default to "default" if not specified)
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get task definition
	taskDefIdentifier := *req.TaskDefinition
	var taskDef *storage.TaskDefinition

	if strings.Contains(taskDefIdentifier, ":") {
		// family:revision format or ARN
		if strings.HasPrefix(taskDefIdentifier, "arn:aws:ecs:") {
			taskDef, err = api.storage.TaskDefinitionStore().GetByARN(ctx, taskDefIdentifier)
		} else {
			// family:revision format
			parts := strings.Split(taskDefIdentifier, ":")
			revision, _ := strconv.Atoi(parts[1])
			taskDef, err = api.storage.TaskDefinitionStore().Get(ctx, parts[0], revision)
		}
	} else {
		// Just family - get latest
		taskDef, err = api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefIdentifier)
	}

	if err != nil || taskDef == nil {
		return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
	}

	// Set default count
	count := int32(1)
	if req.Count != nil {
		count = *req.Count
	}

	// Create tasks
	var tasks []types.Task
	var taskArns []string
	failures := []types.Failure{}

	for i := int32(0); i < count; i++ {
		taskID := uuid.New().String()
		taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", api.region, api.accountID, cluster.Name, taskID)

		// Create storage task
		storageTask := &storage.Task{
			ID:                taskID,
			ARN:               taskARN,
			ClusterARN:        cluster.ARN,
			TaskDefinitionARN: taskDef.ARN,
			DesiredStatus:     "RUNNING",
			LastStatus:        "PENDING",
			LaunchType:        string(req.LaunchType),
			StartedBy:         aws.ToString(req.StartedBy),
			Group:             aws.ToString(req.Group),
			CreatedAt:         time.Now(),
			Version:           1,
			Region:            api.region,
			AccountID:         api.accountID,
		}

		// Handle overrides
		if req.Overrides != nil {
			overridesJSON, _ := json.Marshal(req.Overrides)
			storageTask.Overrides = string(overridesJSON)
		}

		// Handle platform version
		if req.PlatformVersion != nil {
			storageTask.PlatformVersion = *req.PlatformVersion
		}

		// Handle capacity provider
		if len(req.CapacityProviderStrategy) > 0 {
			storageTask.CapacityProviderName = *req.CapacityProviderStrategy[0].CapacityProvider
		}

		// Handle enable execute command
		if req.EnableExecuteCommand {
			storageTask.EnableExecuteCommand = req.EnableExecuteCommand
		}

		// Handle tags
		if req.Tags != nil {
			tagsJSON, _ := json.Marshal(req.Tags)
			storageTask.Tags = string(tagsJSON)
		}

		// Store task
		if err := api.storage.TaskStore().Create(ctx, storageTask); err != nil {
			log.Printf("Failed to create task in storage: %v", err)
			failures = append(failures, types.Failure{
				Arn:    aws.String(taskARN),
				Reason: aws.String("INTERNAL_ERROR"),
				Detail: aws.String(fmt.Sprintf("Failed to create task: %v", err)),
			})
			continue
		}

		// Create Kubernetes pod if manager is available
		// TODO: Implement Kubernetes pod creation
		// if api.kindManager != nil {
		// 	go api.createTaskPod(ctx, cluster, storageTask, taskDef, req)
		// }

		// Build task response
		task := types.Task{
			TaskArn:           aws.String(taskARN),
			ClusterArn:        aws.String(cluster.ARN),
			TaskDefinitionArn: aws.String(taskDef.ARN),
			DesiredStatus:     aws.String("RUNNING"),
			LastStatus:        aws.String("PENDING"),
			LaunchType:        req.LaunchType,
			StartedBy:         req.StartedBy,
			Group:             req.Group,
			Overrides:         req.Overrides,
			PlatformVersion:   req.PlatformVersion,
			CreatedAt:         aws.Time(storageTask.CreatedAt),
			Version:           1,
			EnableExecuteCommand: req.EnableExecuteCommand,
			Tags:              req.Tags,
		}

		tasks = append(tasks, task)
		taskArns = append(taskArns, taskARN)
	}

	// Update cluster task count
	cluster.PendingTasksCount += len(tasks)
	if err := api.storage.ClusterStore().Update(ctx, cluster); err != nil {
		log.Printf("Failed to update cluster task count: %v", err)
	}

	return &ecs.RunTaskOutput{
		Tasks:    tasks,
		Failures: failures,
	}, nil
}

// StopTaskV2 implements the StopTask operation using AWS SDK types
func (api *DefaultECSAPIV2) StopTaskV2(ctx context.Context, req *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	// Validate required fields
	if req.Task == nil {
		return nil, fmt.Errorf("task is required")
	}

	// Get cluster name
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get task from storage
	taskID := extractTaskIDFromARN(*req.Task)
	task, err := api.storage.TaskStore().Get(ctx, cluster.ARN, taskID)
	if err != nil || task == nil {
		return nil, fmt.Errorf("task not found: %s", *req.Task)
	}

	// Update task status
	task.DesiredStatus = "STOPPED"
	task.StoppedReason = aws.ToString(req.Reason)
	stoppedAt := time.Now()
	task.StoppedAt = &stoppedAt
	
	if err := api.storage.TaskStore().Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Stop Kubernetes pod if manager is available
	// TODO: Implement Kubernetes pod deletion
	// if api.kindManager != nil {
	// 	go api.stopTaskPod(ctx, cluster, task)
	// }

	// Build response
	return &ecs.StopTaskOutput{
		Task: &types.Task{
			TaskArn:           aws.String(task.ARN),
			ClusterArn:        aws.String(task.ClusterARN),
			TaskDefinitionArn: aws.String(task.TaskDefinitionARN),
			DesiredStatus:     aws.String(task.DesiredStatus),
			LastStatus:        aws.String(task.LastStatus),
			StoppedReason:     aws.String(task.StoppedReason),
			StoppedAt:         task.StoppedAt,
			StartedBy:         aws.String(task.StartedBy),
			Group:             aws.String(task.Group),
			LaunchType:        types.LaunchType(task.LaunchType),
			Version:           task.Version,
			CreatedAt:         aws.Time(task.CreatedAt),
		},
	}, nil
}

// DescribeTasksV2 implements the DescribeTasks operation using AWS SDK types
func (api *DefaultECSAPIV2) DescribeTasksV2(ctx context.Context, req *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	// Get cluster name
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get tasks
	var tasks []types.Task
	var failures []types.Failure

	// Extract task IDs from ARNs
	var taskIDs []string
	for _, taskArn := range req.Tasks {
		taskID := extractTaskIDFromARN(taskArn)
		taskIDs = append(taskIDs, taskID)
	}

	// Get tasks from storage
	storageTasks, err := api.storage.TaskStore().GetByARNs(ctx, req.Tasks)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Convert storage tasks to response
	for _, storageTask := range storageTasks {
		task := convertStorageTaskToAWSTask(storageTask)
		tasks = append(tasks, task)
	}

	// Check for missing tasks
	foundTaskIDs := make(map[string]bool)
	for _, task := range storageTasks {
		foundTaskIDs[task.ID] = true
	}

	for i, taskID := range taskIDs {
		if !foundTaskIDs[taskID] {
			failures = append(failures, types.Failure{
				Arn:    aws.String(req.Tasks[i]),
				Reason: aws.String("MISSING"),
				Detail: aws.String(fmt.Sprintf("Could not find task %s", req.Tasks[i])),
			})
		}
	}

	return &ecs.DescribeTasksOutput{
		Tasks:    tasks,
		Failures: failures,
	}, nil
}

// ListTasksV2 implements the ListTasks operation using AWS SDK types
func (api *DefaultECSAPIV2) ListTasksV2(ctx context.Context, req *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	// Get cluster name
	clusterName := "default"
	if req.Cluster != nil && *req.Cluster != "" {
		clusterName = extractClusterNameFromARN(*req.Cluster)
	}

	// Get cluster from storage
	cluster, err := api.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Build filters
	filters := storage.TaskFilters{
		ServiceName:       aws.ToString(req.ServiceName),
		Family:            aws.ToString(req.Family),
		ContainerInstance: aws.ToString(req.ContainerInstance),
		LaunchType:        string(req.LaunchType),
		DesiredStatus:     string(req.DesiredStatus),
		StartedBy:         aws.ToString(req.StartedBy),
	}

	// Set limit
	if req.MaxResults != nil {
		filters.MaxResults = int(*req.MaxResults)
	} else {
		filters.MaxResults = 100
	}

	// Set next token
	if req.NextToken != nil {
		filters.NextToken = *req.NextToken
	}

	// List tasks
	tasks, err := api.storage.TaskStore().List(ctx, cluster.ARN, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Build response
	taskArns := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskArns = append(taskArns, task.ARN)
	}

	response := &ecs.ListTasksOutput{
		TaskArns: taskArns,
	}

	// Set next token if there are more results
	if len(tasks) == filters.MaxResults {
		response.NextToken = aws.String(fmt.Sprintf("offset:%d", len(taskArns)))
	}

	return response, nil
}

// Helper function to convert storage task to AWS task type
func convertStorageTaskToAWSTask(storageTask *storage.Task) types.Task {
	task := types.Task{
		TaskArn:           aws.String(storageTask.ARN),
		ClusterArn:        aws.String(storageTask.ClusterARN),
		TaskDefinitionArn: aws.String(storageTask.TaskDefinitionARN),
		DesiredStatus:     aws.String(storageTask.DesiredStatus),
		LastStatus:        aws.String(storageTask.LastStatus),
		LaunchType:        types.LaunchType(storageTask.LaunchType),
		StartedBy:         aws.String(storageTask.StartedBy),
		Group:             aws.String(storageTask.Group),
		Version:           storageTask.Version,
		CreatedAt:         aws.Time(storageTask.CreatedAt),
		PlatformVersion:   aws.String(storageTask.PlatformVersion),
		Cpu:               aws.String(storageTask.CPU),
		Memory:            aws.String(storageTask.Memory),
		EnableExecuteCommand: storageTask.EnableExecuteCommand,
	}

	// Parse JSON fields
	if storageTask.Overrides != "" {
		var overrides types.TaskOverride
		if err := json.Unmarshal([]byte(storageTask.Overrides), &overrides); err == nil {
			task.Overrides = &overrides
		}
	}

	if storageTask.Containers != "" {
		var containers []types.Container
		if err := json.Unmarshal([]byte(storageTask.Containers), &containers); err == nil {
			task.Containers = containers
		}
	}

	if storageTask.Attachments != "" {
		var attachments []types.Attachment
		if err := json.Unmarshal([]byte(storageTask.Attachments), &attachments); err == nil {
			task.Attachments = attachments
		}
	}

	if storageTask.Tags != "" {
		var tags []types.Tag
		if err := json.Unmarshal([]byte(storageTask.Tags), &tags); err == nil {
			task.Tags = tags
		}
	}

	// Set timestamps
	if storageTask.StartedAt != nil {
		task.StartedAt = storageTask.StartedAt
	}
	if storageTask.StoppedAt != nil {
		task.StoppedAt = storageTask.StoppedAt
		task.StoppedReason = aws.String(storageTask.StoppedReason)
	}
	if storageTask.PullStartedAt != nil {
		task.PullStartedAt = storageTask.PullStartedAt
	}
	if storageTask.PullStoppedAt != nil {
		task.PullStoppedAt = storageTask.PullStoppedAt
	}

	return task
}

// TODO: Implement these methods when KindManager supports pod operations
// // createTaskPod creates a Kubernetes pod for the task
// func (api *DefaultECSAPIV2) createTaskPod(ctx context.Context, cluster *storage.Cluster, task *storage.Task, taskDef *storage.TaskDefinition, req *ecs.RunTaskInput) {
// }
//
// // stopTaskPod stops a Kubernetes pod for the task  
// func (api *DefaultECSAPIV2) stopTaskPod(ctx context.Context, cluster *storage.Cluster, task *storage.Task) {
// }


// extractTaskIDFromARN extracts task ID from ARN
func extractTaskIDFromARN(arn string) string {
	// ARN format: arn:aws:ecs:region:account:task/cluster-name/task-id
	parts := strings.Split(arn, "/")
	if len(parts) >= 3 {
		return parts[len(parts)-1]
	}
	return arn
}