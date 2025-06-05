package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// handleECSCreateTaskSet handles the CreateTaskSet API endpoint in AWS ECS format
func (s *Server) handleECSCreateTaskSet(w http.ResponseWriter, body []byte) {
	var req CreateTaskSetRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set creation logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskSetId := "ts-" + uuid.New().String()[:8]
	
	// Default scale if not provided
	scale := req.Scale
	if scale == nil {
		scale = &Scale{
			Value: 100.0,
			Unit:  "PERCENT",
		}
	}

	resp := CreateTaskSetResponse{
		TaskSet: TaskSet{
			Id:               taskSetId,
			TaskSetArn:       fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", s.region, s.accountID, cluster, req.Service, taskSetId),
			ServiceArn:       fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster, req.Service),
			ClusterArn:       fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster),
			ExternalId:       req.ExternalId,
			Status:           "ACTIVE",
			TaskDefinition:   req.TaskDefinition,
			LaunchType:       req.LaunchType,
			Scale:            scale,
			StabilityStatus:  "STEADY_STATE",
			CreatedAt:        time.Now().Format(time.RFC3339),
			LoadBalancers:    req.LoadBalancers,
			ServiceRegistries: req.ServiceRegistries,
			NetworkConfiguration: req.NetworkConfiguration,
			CapacityProviderStrategy: req.CapacityProviderStrategy,
			PlatformVersion:  req.PlatformVersion,
			Tags:             req.Tags,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDeleteTaskSet handles the DeleteTaskSet API endpoint in AWS ECS format
func (s *Server) handleECSDeleteTaskSet(w http.ResponseWriter, body []byte) {
	var req DeleteTaskSetRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set deletion logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := DeleteTaskSetResponse{
		TaskSet: TaskSet{
			Id:             req.TaskSet,
			TaskSetArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", s.region, s.accountID, cluster, req.Service, req.TaskSet),
			ServiceArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster, req.Service),
			ClusterArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster),
			Status:         "DRAINING",
			UpdatedAt:      time.Now().Format(time.RFC3339),
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDescribeTaskSets handles the DescribeTaskSets API endpoint in AWS ECS format
func (s *Server) handleECSDescribeTaskSets(w http.ResponseWriter, body []byte) {
	var req DescribeTaskSetsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set description logic
	// For now, return mock task sets
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	taskSets := []TaskSet{}
	
	if len(req.TaskSets) > 0 {
		// Return specific task sets
		for _, taskSetId := range req.TaskSets {
			taskSets = append(taskSets, TaskSet{
				Id:             taskSetId,
				TaskSetArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", s.region, s.accountID, cluster, req.Service, taskSetId),
				ServiceArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster, req.Service),
				ClusterArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster),
				Status:         "ACTIVE",
				TaskDefinition: fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/sample-app:1", s.region, s.accountID),
				ComputedDesiredCount: 3,
				PendingCount:   0,
				RunningCount:   3,
				LaunchType:     "EC2",
				Scale: &Scale{
					Value: 100.0,
					Unit:  "PERCENT",
				},
				StabilityStatus: "STEADY_STATE",
				CreatedAt:       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				UpdatedAt:       time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			})
		}
	} else {
		// Return all task sets for the service (mock data)
		taskSets = append(taskSets, TaskSet{
			Id:             "ts-12345678",
			TaskSetArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/ts-12345678", s.region, s.accountID, cluster, req.Service),
			ServiceArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster, req.Service),
			ClusterArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster),
			Status:         "ACTIVE",
			TaskDefinition: fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/sample-app:1", s.region, s.accountID),
			ComputedDesiredCount: 3,
			PendingCount:   0,
			RunningCount:   3,
			LaunchType:     "EC2",
			Scale: &Scale{
				Value: 100.0,
				Unit:  "PERCENT",
			},
			StabilityStatus: "STEADY_STATE",
			CreatedAt:       time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			UpdatedAt:       time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		})
	}

	resp := DescribeTaskSetsResponse{
		TaskSets: taskSets,
		Failures: []Failure{},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSUpdateTaskSet handles the UpdateTaskSet API endpoint in AWS ECS format
func (s *Server) handleECSUpdateTaskSet(w http.ResponseWriter, body []byte) {
	var req UpdateTaskSetRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task set update logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	resp := UpdateTaskSetResponse{
		TaskSet: TaskSet{
			Id:             req.TaskSet,
			TaskSetArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", s.region, s.accountID, cluster, req.Service, req.TaskSet),
			ServiceArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", s.region, s.accountID, cluster, req.Service),
			ClusterArn:     fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", s.region, s.accountID, cluster),
			Status:         "ACTIVE",
			TaskDefinition: fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/sample-app:1", s.region, s.accountID),
			Scale:          req.Scale,
			StabilityStatus: "STABILIZING",
			UpdatedAt:       time.Now().Format(time.RFC3339),
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}