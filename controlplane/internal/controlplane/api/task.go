package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

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

	// TODO: Implement actual task stopping logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskArn := req.Task
	reason := req.Reason

	resp := StopTaskResponse{
		Task: Task{
			TaskArn:       taskArn,
			ClusterArn:    "arn:aws:ecs:region:account:cluster/" + cluster,
			LastStatus:    "STOPPING",
			DesiredStatus: "STOPPED",
			StoppedReason: reason,
			StoppingAt:    "2025-05-15T00:40:35+09:00",
		},
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

	// TODO: Implement actual task description logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskArn := ""
	if len(req.Tasks) > 0 {
		taskArn = req.Tasks[0]
	}

	resp := DescribeTasksResponse{
		Tasks: []Task{
			{
				TaskArn:           taskArn,
				ClusterArn:        "arn:aws:ecs:region:account:cluster/" + cluster,
				TaskDefinitionArn: "arn:aws:ecs:region:account:task-definition/family:1",
				LastStatus:        "RUNNING",
				DesiredStatus:     "RUNNING",
				CreatedAt:         "2025-05-15T00:40:35+09:00",
				StartedAt:         "2025-05-15T00:40:40+09:00",
			},
		},
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

	// TODO: Implement actual task description logic

	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskArn := ""
	if len(req.Tasks) > 0 {
		taskArn = req.Tasks[0]
	}

	resp := DescribeTasksResponse{
		Tasks: []Task{
			{
				TaskArn:           taskArn,
				ClusterArn:        "arn:aws:ecs:region:account:cluster/" + cluster,
				TaskDefinitionArn: "arn:aws:ecs:region:account:task-definition/family:1",
				LastStatus:        "RUNNING",
				DesiredStatus:     "RUNNING",
				CreatedAt:         "2025-05-15T00:40:35+09:00",
				StartedAt:         "2025-05-15T00:40:40+09:00",
			},
		},
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