package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// registerTaskEndpoints registers all task-related API endpoints
func (s *Server) registerTaskEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/runtask", s.handleRunTask)
	mux.HandleFunc("/v1/starttask", s.handleStartTask)
	mux.HandleFunc("/v1/stoptask", s.handleStopTask)
	mux.HandleFunc("/v1/describetasks", s.handleDescribeTasks)
	mux.HandleFunc("/v1/listtasks", s.handleListTasks)
	mux.HandleFunc("/v1/gettaskprotection", s.handleGetTaskProtection)
}

// handleRunTask handles the RunTask API endpoint
func (s *Server) handleRunTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RunTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task running logic with the implementations from task_handler.go

	// For now, return a simple mock response that matches the local types
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskDefinitionArn := req.TaskDefinition

	resp := RunTaskResponse{
		Tasks: []Task{
			{
				TaskArn:           "arn:aws:ecs:region:account:task/" + cluster + "/task-id",
				ClusterArn:        "arn:aws:ecs:region:account:cluster/" + cluster,
				TaskDefinitionArn: taskDefinitionArn,
				LastStatus:        "PENDING",
				DesiredStatus:     "RUNNING",
				CreatedAt:         "2025-05-15T00:40:35+09:00",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStartTask handles the StartTask API endpoint
func (s *Server) handleStartTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StartTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task starting logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskDefinitionArn := req.TaskDefinition

	containerInstance := ""
	if len(req.ContainerInstances) > 0 {
		containerInstance = req.ContainerInstances[0]
	}

	resp := StartTaskResponse{
		Tasks: []Task{
			{
				TaskArn:              "arn:aws:ecs:region:account:task/" + cluster + "/task-id",
				ClusterArn:           "arn:aws:ecs:region:account:cluster/" + cluster,
				TaskDefinitionArn:    taskDefinitionArn,
				ContainerInstanceArn: "arn:aws:ecs:region:account:container-instance/" + cluster + "/" + containerInstance,
				LastStatus:           "PENDING",
				DesiredStatus:        "RUNNING",
				CreatedAt:            "2025-05-15T00:40:35+09:00",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStopTask handles the StopTask API endpoint
func (s *Server) handleStopTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req StopTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.StopTaskWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error stopping task: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeTasks handles the DescribeTasks API endpoint
func (s *Server) handleDescribeTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.DescribeTasksWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error describing tasks: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListTasks handles the ListTasks API endpoint
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListTasksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task listing logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := ListTasksResponse{
		TaskArns: []string{"arn:aws:ecs:region:account:task/" + cluster + "/task-id"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleGetTaskProtection handles the GetTaskProtection API endpoint
func (s *Server) handleGetTaskProtection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetTaskProtectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task protection logic

	// For now, return a mock response
	taskArn := ""
	if len(req.Tasks) > 0 {
		taskArn = req.Tasks[0]
	}

	resp := GetTaskProtectionResponse{
		ProtectedTasks: []ProtectionResult{
			{
				TaskArn:           taskArn,
				ProtectionEnabled: false,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ECS API handlers (called from ecs_handler.go)

// handleRunTaskECS handles the RunTask operation for ECS API
func (s *Server) handleRunTaskECS(w http.ResponseWriter, body []byte) {
	var req RunTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Step 1: Validate and get cluster
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	cluster, err := s.storage.ClusterStore().Get(context.Background(), clusterName)
	if err != nil || cluster == nil {
		http.Error(w, fmt.Sprintf("Cluster not found: %s", clusterName), http.StatusBadRequest)
		return
	}

	// Step 2: Get task definition
	taskDefArn := req.TaskDefinition
	if !strings.HasPrefix(taskDefArn, "arn:aws:ecs:") {
		// Convert family:revision to ARN
		taskDefArn = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s", s.region, s.accountID, req.TaskDefinition)
	}

	log.Printf("Looking for task definition with ARN: %s", taskDefArn)
	taskDef, err := s.storage.TaskDefinitionStore().GetByARN(context.Background(), taskDefArn)
	if err != nil || taskDef == nil {
		log.Printf("Task definition not found. Error: %v, TaskDef: %v", err, taskDef)
		http.Error(w, fmt.Sprintf("Task definition not found: %s (searched ARN: %s)", req.TaskDefinition, taskDefArn), http.StatusBadRequest)
		return
	}

	// Step 3: Create a simple Kubernetes pod
	taskID := generateSimpleTaskID()
	taskArn := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", s.region, s.accountID, cluster.Name, taskID)

	// Create basic pod
	pod, err := s.createBasicPod(taskDef, cluster, taskID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create pod: %v", err), http.StatusInternalServerError)
		return
	}

	// Step 4: Store task in database
	now := time.Now()
	task := &storage.Task{
		ID:                taskID,
		ARN:               taskArn,
		ClusterARN:        cluster.ARN,
		TaskDefinitionARN: taskDef.ARN,
		LastStatus:        "PENDING",
		DesiredStatus:     "RUNNING",
		LaunchType:        "FARGATE",
		Version:           1,
		CreatedAt:         now,
		Region:            s.region,
		AccountID:         s.accountID,
		Containers:        "[]",
		PodName:           pod.Name,
		Namespace:         pod.Namespace,
	}

	if err := s.storage.TaskStore().Create(context.Background(), task); err != nil {
		log.Printf("Failed to store task: %v", err)
		// Continue anyway for now
	}

	// Step 5: Return response
	resp := RunTaskResponse{
		Tasks: []Task{
			{
				TaskArn:           taskArn,
				ClusterArn:        cluster.ARN,
				TaskDefinitionArn: taskDef.ARN,
				LastStatus:        "PENDING",
				DesiredStatus:     "RUNNING",
				CreatedAt:         now.Format(time.RFC3339),
				LaunchType:        "FARGATE",
				Version:           1,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeTasksECS handles the DescribeTasks operation for ECS API
func (s *Server) handleDescribeTasksECS(w http.ResponseWriter, body []byte) {
	var req DescribeTasksRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.DescribeTasksWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error describing tasks: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListTasksECS handles the ListTasks operation for ECS API
func (s *Server) handleListTasksECS(w http.ResponseWriter, body []byte) {
	var req ListTasksRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task listing logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := ListTasksResponse{
		TaskArns: []string{"arn:aws:ecs:region:account:task/" + cluster + "/task-id"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStopTaskECS handles the StopTask operation for ECS API
func (s *Server) handleStopTaskECS(w http.ResponseWriter, body []byte) {
	var req StopTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	resp, err := s.StopTaskWithStorage(ctx, req)
	if err != nil {
		log.Printf("Error stopping task: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// DescribeTasksWithStorage describes tasks using storage and Kubernetes status
func (s *Server) DescribeTasksWithStorage(ctx context.Context, req DescribeTasksRequest) (*DescribeTasksResponse, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	// Get cluster from storage
	cluster, err := s.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	var tasks []Task
	var failures []Failure

	for _, taskIdentifier := range req.Tasks {
		// taskIdentifier can be either task ARN or task ID
		var taskARN string
		if strings.HasPrefix(taskIdentifier, "arn:aws:ecs:") {
			taskARN = taskIdentifier
		} else {
			// Convert task ID to ARN
			taskARN = fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", s.region, s.accountID, cluster.Name, taskIdentifier)
		}

		// Get task from storage
		storageTask, err := s.storage.TaskStore().Get(ctx, cluster.ARN, taskIdentifier)
		if err != nil {
			failures = append(failures, Failure{
				Arn:    taskARN,
				Reason: "MISSING",
				Detail: err.Error(),
			})
			continue
		}

		if storageTask == nil {
			failures = append(failures, Failure{
				Arn:    taskARN,
				Reason: "MISSING", 
				Detail: "Task not found",
			})
			continue
		}

		// Get current status from Kubernetes if pod exists
		if storageTask.PodName != "" && storageTask.Namespace != "" {
			if err := s.updateTaskStatusFromKubernetes(ctx, storageTask, cluster); err != nil {
				log.Printf("Warning: failed to update task status from Kubernetes: %v", err)
				// Continue with stored status
			}
		}

		// Convert storage task to API task
		task := s.storageTaskToAPITask(storageTask)
		tasks = append(tasks, task)
	}

	return &DescribeTasksResponse{
		Tasks:    tasks,
		Failures: failures,
	}, nil
}

// updateTaskStatusFromKubernetes updates task status by checking Kubernetes pod
func (s *Server) updateTaskStatusFromKubernetes(ctx context.Context, task *storage.Task, cluster *storage.Cluster) error {
	// Get Kubernetes client for the cluster
	kubeClient, err := s.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Get Pod status
	pod, err := kubeClient.CoreV1().Pods(task.Namespace).Get(
		ctx, task.PodName, metav1.GetOptions{})
	if err != nil {
		// Pod not found - task is stopped
		task.LastStatus = "STOPPED"
		task.DesiredStatus = "STOPPED"
		if task.StoppedReason == "" {
			task.StoppedReason = "Pod not found"
		}
		return nil
	}

	// Map Pod phase to ECS status
	previousStatus := task.LastStatus
	switch pod.Status.Phase {
	case corev1.PodPending:
		task.LastStatus = "PENDING"
	case corev1.PodRunning:
		task.LastStatus = "RUNNING"
		if task.StartedAt == nil && pod.Status.StartTime != nil {
			startTime := pod.Status.StartTime.Time
			task.StartedAt = &startTime
		}
	case corev1.PodSucceeded:
		task.LastStatus = "STOPPED"
		task.DesiredStatus = "STOPPED"
		if task.StoppedReason == "" {
			task.StoppedReason = "Task completed successfully"
		}
	case corev1.PodFailed:
		task.LastStatus = "STOPPED"
		task.DesiredStatus = "STOPPED"
		if task.StoppedReason == "" {
			task.StoppedReason = "Task failed"
		}
	}

	// Update task in storage if status changed
	if previousStatus != task.LastStatus {
		task.Version++
		now := time.Now()
		if task.LastStatus == "STOPPED" && task.StoppedAt == nil {
			task.StoppedAt = &now
			task.ExecutionStoppedAt = &now
		}

		if err := s.storage.TaskStore().Update(ctx, task); err != nil {
			log.Printf("Warning: failed to update task in storage: %v", err)
		}
	}

	return nil
}

// storageTaskToAPITask converts a storage.Task to an API Task
func (s *Server) storageTaskToAPITask(storageTask *storage.Task) Task {
	task := Task{
		TaskArn:           storageTask.ARN,
		ClusterArn:        storageTask.ClusterARN,
		TaskDefinitionArn: storageTask.TaskDefinitionARN,
		LastStatus:        storageTask.LastStatus,
		DesiredStatus:     storageTask.DesiredStatus,
		CreatedAt:         storageTask.CreatedAt.Format(time.RFC3339),
		LaunchType:        storageTask.LaunchType,
		PlatformVersion:   storageTask.PlatformVersion,
		Group:             storageTask.Group,
		StoppedReason:     storageTask.StoppedReason,
		HealthStatus:      storageTask.HealthStatus,
		Cpu:               storageTask.CPU,
		Memory:            storageTask.Memory,
	}

	// Add timestamps if available
	if storageTask.StartedAt != nil {
		task.StartedAt = storageTask.StartedAt.Format(time.RFC3339)
	}
	if storageTask.StoppedAt != nil {
		task.StoppedAt = storageTask.StoppedAt.Format(time.RFC3339)
	}
	if storageTask.PullStartedAt != nil {
		task.PullStartedAt = storageTask.PullStartedAt.Format(time.RFC3339)
	}
	if storageTask.PullStoppedAt != nil {
		task.PullStoppedAt = storageTask.PullStoppedAt.Format(time.RFC3339)
	}

	// Parse containers JSON
	if storageTask.Containers != "" && storageTask.Containers != "[]" {
		var containers []Container
		if err := json.Unmarshal([]byte(storageTask.Containers), &containers); err == nil {
			task.Containers = containers
		}
	}

	// Parse overrides if available
	if storageTask.Overrides != "" {
		var overrides TaskOverride
		if err := json.Unmarshal([]byte(storageTask.Overrides), &overrides); err == nil {
			task.Overrides = &overrides
		}
	}

	return task
}

// StopTaskWithStorage stops a running task by deleting its Kubernetes pod
func (s *Server) StopTaskWithStorage(ctx context.Context, req StopTaskRequest) (*StopTaskResponse, error) {
	// Default cluster if not specified
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	// Get cluster from storage
	cluster, err := s.storage.ClusterStore().Get(ctx, clusterName)
	if err != nil || cluster == nil {
		return nil, fmt.Errorf("cluster not found: %s", clusterName)
	}

	// Get task from storage (task identifier can be ARN or ID)
	taskIdentifier := req.Task
	storageTask, err := s.storage.TaskStore().Get(ctx, cluster.ARN, taskIdentifier)
	if err != nil || storageTask == nil {
		return nil, fmt.Errorf("task not found: %s", taskIdentifier)
	}

	// Update task status to STOPPING
	now := time.Now()
	storageTask.DesiredStatus = "STOPPED"
	storageTask.LastStatus = "STOPPING"
	storageTask.StoppedReason = req.Reason
	if storageTask.StoppedReason == "" {
		storageTask.StoppedReason = "Task stopped by user"
	}
	storageTask.StoppingAt = &now
	storageTask.Version++

	// Update task in storage
	if err := s.storage.TaskStore().Update(ctx, storageTask); err != nil {
		log.Printf("Warning: failed to update task status to STOPPING: %v", err)
	}

	// Delete the Kubernetes pod if it exists
	if storageTask.PodName != "" && storageTask.Namespace != "" {
		if err := s.deleteTaskPod(ctx, storageTask, cluster); err != nil {
			log.Printf("Warning: failed to delete pod for task %s: %v", storageTask.ARN, err)
			// Continue anyway - the pod might already be gone
		}
	}

	// Update final task status
	storageTask.LastStatus = "STOPPED"
	storageTask.StoppedAt = &now
	storageTask.ExecutionStoppedAt = &now
	storageTask.Version++

	// Update task in storage with final status
	if err := s.storage.TaskStore().Update(ctx, storageTask); err != nil {
		log.Printf("Warning: failed to update task status to STOPPED: %v", err)
	}

	// Convert to API response
	task := s.storageTaskToAPITask(storageTask)

	return &StopTaskResponse{
		Task: task,
	}, nil
}

// deleteTaskPod deletes the Kubernetes pod for a task
func (s *Server) deleteTaskPod(ctx context.Context, task *storage.Task, cluster *storage.Cluster) error {
	// Get Kubernetes client for the cluster
	kubeClient, err := s.kindManager.GetKubeClient(cluster.KindClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Delete the pod
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	err = kubeClient.CoreV1().Pods(task.Namespace).Delete(ctx, task.PodName, deleteOptions)
	if err != nil {
		// Check if the pod is already gone
		if !strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("failed to delete pod: %w", err)
		}
		// Pod not found is OK - it might have already been deleted
		log.Printf("Pod %s in namespace %s was already deleted", task.PodName, task.Namespace)
	} else {
		log.Printf("Successfully deleted pod %s in namespace %s for task %s", 
			task.PodName, task.Namespace, task.ARN)
	}

	return nil
}

// handleStartTaskECS handles the StartTask operation for ECS API
func (s *Server) handleStartTaskECS(w http.ResponseWriter, body []byte) {
	var req StartTaskRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// StartTask is used to start tasks on specific container instances
	// For now, we'll treat it similarly to RunTask but with container instance constraints
	
	// Step 1: Validate and get cluster
	clusterName := "default"
	if req.Cluster != "" {
		clusterName = req.Cluster
	}

	cluster, err := s.storage.ClusterStore().Get(context.Background(), clusterName)
	if err != nil || cluster == nil {
		http.Error(w, fmt.Sprintf("Cluster not found: %s", clusterName), http.StatusBadRequest)
		return
	}

	// Step 2: Get task definition
	taskDefArn := req.TaskDefinition
	if !strings.HasPrefix(taskDefArn, "arn:aws:ecs:") {
		// Convert family:revision to ARN
		taskDefArn = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s", s.region, s.accountID, req.TaskDefinition)
	}

	taskDef, err := s.storage.TaskDefinitionStore().GetByARN(context.Background(), taskDefArn)
	if err != nil || taskDef == nil {
		http.Error(w, fmt.Sprintf("Task definition not found: %s", req.TaskDefinition), http.StatusBadRequest)
		return
	}

	// Step 3: Create tasks for each container instance
	tasks := []Task{}
	failures := []Failure{}

	for _, containerInstance := range req.ContainerInstances {
		// Create task ID and ARN
		taskID := generateSimpleTaskID()
		taskArn := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", s.region, s.accountID, cluster.Name, taskID)

		// Create basic pod
		pod, err := s.createBasicPod(taskDef, cluster, taskID)
		if err != nil {
			failures = append(failures, Failure{
				Arn:    containerInstance,
				Reason: fmt.Sprintf("Failed to create pod: %v", err),
			})
			continue
		}

		// Store task in database
		now := time.Now()
		storageTask := &storage.Task{
			ARN:                  taskArn,
			ClusterARN:           cluster.ARN,
			TaskDefinitionARN:    taskDef.ARN,
			ContainerInstanceARN: containerInstance,
			DesiredStatus:        "RUNNING",
			LastStatus:           "PENDING",
			LaunchType:           "EC2", // StartTask always uses EC2 launch type
			Version:              1,
			CreatedAt:            now,
			PodName:              pod.Name,
			Namespace:            pod.Namespace,
			Group:                req.Group,
			StartedBy:            req.StartedBy,
		}

		// Store task
		if err := s.storage.TaskStore().Create(context.Background(), storageTask); err != nil {
			failures = append(failures, Failure{
				Arn:    containerInstance,
				Reason: fmt.Sprintf("Failed to store task: %v", err),
			})
			continue
		}

		// Convert to API task
		apiTask := s.storageTaskToAPITask(storageTask)
		tasks = append(tasks, apiTask)
	}

	// Return response
	resp := StartTaskResponse{
		Tasks:    tasks,
		Failures: failures,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
