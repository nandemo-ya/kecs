package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// RunTask implements the RunTask operation
func (api *DefaultECSAPI) RunTask(ctx context.Context, req *generated.RunTaskRequest) (*generated.RunTaskResponse, error) {
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
			parts := strings.SplitN(taskDefIdentifier, ":", 2)
			family := parts[0]
			revision, _ := parseRevision(parts[1])
			taskDef, err = api.storage.TaskDefinitionStore().Get(ctx, family, revision)
		}
	} else {
		// Just family name - get latest
		taskDef, err = api.storage.TaskDefinitionStore().GetLatest(ctx, taskDefIdentifier)
	}

	if err != nil || taskDef == nil {
		return nil, fmt.Errorf("task definition not found: %s", taskDefIdentifier)
	}

	// Determine count
	count := 1
	if req.Count != nil && *req.Count > 0 {
		count = int(*req.Count)
	}

	// Create task manager (skip if kindManager is nil - for tests)
	var taskManager *mockTaskManager
	
	if api.kindManager != nil {
		// For production, we would use the real task manager
		// but for now we'll use the mock since the real one expects *corev1.Pod
		taskManager = &mockTaskManager{storage: api.storage}
	} else {
		// Use mock for tests
		taskManager = &mockTaskManager{storage: api.storage}
	}

	// Create task converter
	taskConverter := converters.NewTaskConverter(api.region, api.accountID)

	// Marshal the request for the converter
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var tasks []generated.Task
	var failures []generated.Failure
	
	// Create requested number of tasks
	for i := 0; i < count; i++ {
		// Generate task ID
		taskID := uuid.New().String()
		
		// Create storage task
		now := time.Now()
		task := &storage.Task{
			ID:                taskID,
			ARN:               fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", api.region, api.accountID, clusterName, taskID),
			ClusterARN:        cluster.ARN,
			TaskDefinitionARN: taskDef.ARN,
			LastStatus:        "PROVISIONING",
			DesiredStatus:     "RUNNING",
			LaunchType:        "EC2", // Default, can be overridden
			Version:           1,
			CreatedAt:         now,
			Region:            api.region,
			AccountID:         api.accountID,
		}

		// Apply overrides if any
		if req.Overrides != nil {
			overridesJSON, err := json.Marshal(req.Overrides)
			if err == nil {
				task.Overrides = string(overridesJSON)
			}
		}

		// Set launch type
		if req.LaunchType != nil {
			task.LaunchType = string(*req.LaunchType)
		}

		// Set started by
		if req.StartedBy != nil {
			task.StartedBy = *req.StartedBy
		}

		// Set group
		if req.Group != nil {
			task.Group = *req.Group
		}

		// Set tags
		if req.Tags != nil && len(req.Tags) > 0 {
			tagsJSON, err := json.Marshal(req.Tags)
			if err == nil {
				task.Tags = string(tagsJSON)
			}
		}

		// Set enable execute command
		if req.EnableExecuteCommand != nil {
			task.EnableExecuteCommand = *req.EnableExecuteCommand
		}

		// Convert to Kubernetes pod
		pod, err := taskConverter.ConvertTaskToPod(taskDef, reqJSON, cluster, taskID)
		if err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(task.ARN),
				Reason: ptr.String("RESOURCE_CREATION_FAILED"),
				Detail: ptr.String(fmt.Sprintf("Failed to convert task to pod: %v", err)),
			})
			continue
		}

		// Extract secrets from the converter
		secrets := extractSecretsFromPod(pod)

		// Create the task
		if err := taskManager.CreateTask(ctx, pod, task, secrets); err != nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(task.ARN),
				Reason: ptr.String("RESOURCE_CREATION_FAILED"),
				Detail: ptr.String(fmt.Sprintf("Failed to create task: %v", err)),
			})
			continue
		}

		// Convert to generated task
		genTask := storageTaskToGenerated(task)
		if genTask != nil {
			tasks = append(tasks, *genTask)
		}
	}

	return &generated.RunTaskResponse{
		Tasks:    tasks,
		Failures: failures,
	}, nil
}

// StartTask implements the StartTask operation
func (api *DefaultECSAPI) StartTask(ctx context.Context, req *generated.StartTaskRequest) (*generated.StartTaskResponse, error) {
	// TODO: Implement StartTask
	return nil, fmt.Errorf("StartTask not implemented")
}

// StopTask implements the StopTask operation
func (api *DefaultECSAPI) StopTask(ctx context.Context, req *generated.StopTaskRequest) (*generated.StopTaskResponse, error) {
	// Validate required fields
	if req.Task == nil || *req.Task == "" {
		return nil, fmt.Errorf("task is required")
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

	// Get the task
	taskIdentifier := *req.Task
	task, err := api.storage.TaskStore().Get(ctx, cluster.ARN, taskIdentifier)
	if err != nil || task == nil {
		return nil, fmt.Errorf("task not found: %s", taskIdentifier)
	}

	// Check if task is already stopped
	if task.DesiredStatus == "STOPPED" {
		// Return current state
		genTask := storageTaskToGenerated(task)
		if genTask != nil {
			return &generated.StopTaskResponse{
				Task: genTask,
			}, nil
		}
		return nil, fmt.Errorf("failed to convert task")
	}

	// Create task manager (skip if kindManager is nil - for tests)
	var taskManager *mockTaskManager
	
	if api.kindManager != nil {
		// For production, we would use the real task manager
		// but for now we'll use the mock
		taskManager = &mockTaskManager{storage: api.storage}
	} else {
		// Use mock for tests
		taskManager = &mockTaskManager{storage: api.storage}
	}

	// Set the reason
	reason := "Task stopped by user"
	if req.Reason != nil && *req.Reason != "" {
		reason = *req.Reason
	}

	// Stop the task
	if err := taskManager.StopTask(ctx, cluster.ARN, task.ID, reason); err != nil {
		return nil, fmt.Errorf("failed to stop task: %w", err)
	}

	// Get updated task
	task, err = api.storage.TaskStore().Get(ctx, cluster.ARN, taskIdentifier)
	if err != nil || task == nil {
		return nil, fmt.Errorf("failed to get updated task")
	}

	// Convert to generated task
	genTask := storageTaskToGenerated(task)
	if genTask == nil {
		return nil, fmt.Errorf("failed to convert task")
	}

	return &generated.StopTaskResponse{
		Task: genTask,
	}, nil
}

// DescribeTasks implements the DescribeTasks operation
func (api *DefaultECSAPI) DescribeTasks(ctx context.Context, req *generated.DescribeTasksRequest) (*generated.DescribeTasksResponse, error) {
	// Validate required fields
	if req.Tasks == nil || len(req.Tasks) == 0 {
		return nil, fmt.Errorf("tasks is required")
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

	var tasks []generated.Task
	var failures []generated.Failure

	// Process each task identifier
	for _, taskIdentifier := range req.Tasks {
		task, err := api.storage.TaskStore().Get(ctx, cluster.ARN, taskIdentifier)
		if err != nil || task == nil {
			failures = append(failures, generated.Failure{
				Arn:    ptr.String(taskIdentifier),
				Reason: ptr.String("MISSING"),
				Detail: ptr.String("Task not found"),
			})
			continue
		}

		// Convert to generated task
		genTask := storageTaskToGenerated(task)
		if genTask != nil {
			// Include tags if requested
			if req.Include != nil {
				for _, include := range req.Include {
					if include == generated.TaskFieldTags && task.Tags != "" {
						var tags []generated.Tag
						if err := json.Unmarshal([]byte(task.Tags), &tags); err == nil {
							genTask.Tags = tags
						}
					}
				}
			}
			tasks = append(tasks, *genTask)
		}
	}

	return &generated.DescribeTasksResponse{
		Tasks:    tasks,
		Failures: failures,
	}, nil
}

// ListTasks implements the ListTasks operation
func (api *DefaultECSAPI) ListTasks(ctx context.Context, req *generated.ListTasksRequest) (*generated.ListTasksResponse, error) {
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

	// Build filters
	filters := storage.TaskFilters{}

	if req.ServiceName != nil {
		filters.ServiceName = *req.ServiceName
	}

	if req.Family != nil {
		filters.Family = *req.Family
	}

	if req.ContainerInstance != nil {
		filters.ContainerInstance = *req.ContainerInstance
	}

	if req.LaunchType != nil {
		filters.LaunchType = string(*req.LaunchType)
	}

	if req.DesiredStatus != nil {
		filters.DesiredStatus = string(*req.DesiredStatus)
	}

	if req.StartedBy != nil {
		filters.StartedBy = *req.StartedBy
	}

	if req.MaxResults != nil && *req.MaxResults > 0 {
		filters.MaxResults = int(*req.MaxResults)
	} else {
		filters.MaxResults = 100 // Default limit
	}

	if req.NextToken != nil {
		filters.NextToken = *req.NextToken
	}

	// List tasks
	tasks, err := api.storage.TaskStore().List(ctx, cluster.ARN, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	// Convert to ARNs
	taskArns := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskArns = append(taskArns, task.ARN)
	}

	response := &generated.ListTasksResponse{
		TaskArns: taskArns,
	}

	// Set next token if there might be more results
	if len(taskArns) == filters.MaxResults && len(taskArns) > 0 {
		response.NextToken = ptr.String(taskArns[len(taskArns)-1])
	}

	return response, nil
}

// GetTaskProtection implements the GetTaskProtection operation
func (api *DefaultECSAPI) GetTaskProtection(ctx context.Context, req *generated.GetTaskProtectionRequest) (*generated.GetTaskProtectionResponse, error) {
	// TODO: Implement GetTaskProtection
	return nil, fmt.Errorf("GetTaskProtection not implemented")
}

// UpdateTaskProtection implements the UpdateTaskProtection operation
func (api *DefaultECSAPI) UpdateTaskProtection(ctx context.Context, req *generated.UpdateTaskProtectionRequest) (*generated.UpdateTaskProtectionResponse, error) {
	// TODO: Implement UpdateTaskProtection
	return nil, fmt.Errorf("UpdateTaskProtection not implemented")
}

// SubmitTaskStateChange implements the SubmitTaskStateChange operation
func (api *DefaultECSAPI) SubmitTaskStateChange(ctx context.Context, req *generated.SubmitTaskStateChangeRequest) (*generated.SubmitTaskStateChangeResponse, error) {
	// TODO: Implement SubmitTaskStateChange
	return nil, fmt.Errorf("SubmitTaskStateChange not implemented")
}

// Helper function to convert storage.Task to generated.Task
func storageTaskToGenerated(task *storage.Task) *generated.Task {
	if task == nil {
		return nil
	}

	genTask := &generated.Task{
		TaskArn:           ptr.String(task.ARN),
		ClusterArn:        ptr.String(task.ClusterARN),
		TaskDefinitionArn: ptr.String(task.TaskDefinitionARN),
		LastStatus:        ptr.String(task.LastStatus),
		DesiredStatus:     ptr.String(task.DesiredStatus),
		LaunchType:        (*generated.LaunchType)(ptr.String(task.LaunchType)),
		Version:           ptr.Int64(task.Version),
		CreatedAt:         ptr.Time(task.CreatedAt),
		Cpu:               ptr.String(task.CPU),
		Memory:            ptr.String(task.Memory),
	}

	// Set optional fields
	if task.ContainerInstanceARN != "" {
		genTask.ContainerInstanceArn = ptr.String(task.ContainerInstanceARN)
	}
	if task.StartedBy != "" {
		genTask.StartedBy = ptr.String(task.StartedBy)
	}
	if task.StopCode != "" {
		genTask.StopCode = (*generated.TaskStopCode)(ptr.String(task.StopCode))
	}
	if task.StoppedReason != "" {
		genTask.StoppedReason = ptr.String(task.StoppedReason)
	}
	if task.StoppingAt != nil {
		genTask.StoppingAt = ptr.Time(*task.StoppingAt)
	}
	if task.StoppedAt != nil {
		genTask.StoppedAt = ptr.Time(*task.StoppedAt)
	}
	if task.StartedAt != nil {
		genTask.StartedAt = ptr.Time(*task.StartedAt)
	}
	if task.Connectivity != "" {
		genTask.Connectivity = (*generated.Connectivity)(ptr.String(task.Connectivity))
	}
	if task.ConnectivityAt != nil {
		genTask.ConnectivityAt = ptr.Time(*task.ConnectivityAt)
	}
	if task.PullStartedAt != nil {
		genTask.PullStartedAt = ptr.Time(*task.PullStartedAt)
	}
	if task.PullStoppedAt != nil {
		genTask.PullStoppedAt = ptr.Time(*task.PullStoppedAt)
	}
	if task.ExecutionStoppedAt != nil {
		genTask.ExecutionStoppedAt = ptr.Time(*task.ExecutionStoppedAt)
	}
	if task.PlatformVersion != "" {
		genTask.PlatformVersion = ptr.String(task.PlatformVersion)
	}
	if task.PlatformFamily != "" {
		genTask.PlatformFamily = ptr.String(task.PlatformFamily)
	}
	if task.Group != "" {
		genTask.Group = ptr.String(task.Group)
	}
	if task.HealthStatus != "" {
		genTask.HealthStatus = (*generated.HealthStatus)(ptr.String(task.HealthStatus))
	}
	if task.CapacityProviderName != "" {
		genTask.CapacityProviderName = ptr.String(task.CapacityProviderName)
	}

	genTask.EnableExecuteCommand = ptr.Bool(task.EnableExecuteCommand)

	// Parse JSON fields
	if task.Overrides != "" {
		var overrides generated.TaskOverride
		if err := json.Unmarshal([]byte(task.Overrides), &overrides); err == nil {
			genTask.Overrides = &overrides
		}
	}
	if task.Containers != "" {
		var containers []generated.Container
		if err := json.Unmarshal([]byte(task.Containers), &containers); err == nil {
			genTask.Containers = containers
		}
	}
	if task.Attachments != "" && task.Attachments != "[]" {
		var attachments []generated.Attachment
		if err := json.Unmarshal([]byte(task.Attachments), &attachments); err == nil {
			genTask.Attachments = attachments
		}
	}
	if task.Attributes != "" && task.Attributes != "[]" {
		var attributes []generated.Attribute
		if err := json.Unmarshal([]byte(task.Attributes), &attributes); err == nil {
			genTask.Attributes = attributes
		}
	}
	if task.EphemeralStorage != "" {
		var ephemeralStorage generated.EphemeralStorage
		if err := json.Unmarshal([]byte(task.EphemeralStorage), &ephemeralStorage); err == nil {
			genTask.EphemeralStorage = &ephemeralStorage
		}
	}

	return genTask
}

// Helper function to parse revision number from string
func parseRevision(revisionStr string) (int, error) {
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		return 0, fmt.Errorf("invalid revision number: %s", revisionStr)
	}
	return revision, nil
}

// Helper function to extract secrets from pod (placeholder - actual implementation would analyze pod spec)
func extractSecretsFromPod(pod interface{}) map[string]*converters.SecretInfo {
	// TODO: Extract secrets from pod spec if needed
	return make(map[string]*converters.SecretInfo)
}

// mockTaskManager is a simple mock for testing
type mockTaskManager struct {
	storage storage.Storage
}

func (m *mockTaskManager) CreateTask(ctx context.Context, pod interface{}, task *storage.Task, secrets map[string]*converters.SecretInfo) error {
	// Simply store the task
	// Keep the LastStatus that was set by RunTask (PROVISIONING)
	task.Connectivity = "CONNECTED"
	now := time.Now()
	task.ConnectivityAt = &now
	return m.storage.TaskStore().Create(ctx, task)
}

func (m *mockTaskManager) StopTask(ctx context.Context, cluster, taskID, reason string) error {
	task, err := m.storage.TaskStore().Get(ctx, cluster, taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	task.DesiredStatus = "STOPPED"
	task.StoppedReason = reason
	task.StoppingAt = &now
	return m.storage.TaskStore().Update(ctx, task)
}