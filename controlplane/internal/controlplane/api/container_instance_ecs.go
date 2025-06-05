package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleECSRegisterContainerInstance handles the RegisterContainerInstance API endpoint in AWS ECS format
func (s *Server) handleECSRegisterContainerInstance(w http.ResponseWriter, body []byte) {
	var req RegisterContainerInstanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance registration logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	containerInstanceArn := req.ContainerInstanceArn
	if containerInstanceArn == "" {
		containerInstanceArn = "arn:aws:ecs:" + s.region + ":" + s.accountID + ":container-instance/" + cluster + "/i-1234567890abcdef0"
	}

	resp := RegisterContainerInstanceResponse{
		ContainerInstance: ContainerInstance{
			ContainerInstanceArn: containerInstanceArn,
			Ec2InstanceId:        "i-1234567890abcdef0",
			Version:              1,
			Status:               "ACTIVE",
			StatusReason:         "",
			AgentConnected:       true,
			RunningTasksCount:    0,
			PendingTasksCount:    0,
			AgentUpdateStatus:    "NOT_STAGED",
			RegisteredAt:         time.Now().Format(time.RFC3339),
			RegisteredResources: []Resource{
				{
					Name:         "CPU",
					Type:         "INTEGER",
					IntegerValue: 2048,
				},
				{
					Name:         "MEMORY",
					Type:         "INTEGER",
					IntegerValue: 4096,
				},
				{
					Name:           "PORTS",
					Type:           "STRINGSET",
					StringSetValue: []string{"22", "80", "443", "2376", "2375", "51678", "51679"},
				},
				{
					Name:           "PORTS_UDP",
					Type:           "STRINGSET",
					StringSetValue: []string{},
				},
			},
			RemainingResources: []Resource{
				{
					Name:         "CPU",
					Type:         "INTEGER",
					IntegerValue: 2048,
				},
				{
					Name:         "MEMORY",
					Type:         "INTEGER",
					IntegerValue: 4096,
				},
				{
					Name:           "PORTS",
					Type:           "STRINGSET",
					StringSetValue: []string{"22", "80", "443", "2376", "2375", "51678", "51679"},
				},
				{
					Name:           "PORTS_UDP",
					Type:           "STRINGSET",
					StringSetValue: []string{},
				},
			},
			VersionInfo:     req.VersionInfo,
			Attributes:      req.Attributes,
			Tags:            req.Tags,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDeregisterContainerInstance handles the DeregisterContainerInstance API endpoint in AWS ECS format
func (s *Server) handleECSDeregisterContainerInstance(w http.ResponseWriter, body []byte) {
	var req DeregisterContainerInstanceRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance deregistration logic
	// For now, return a mock response
	resp := DeregisterContainerInstanceResponse{
		ContainerInstance: ContainerInstance{
			ContainerInstanceArn: req.ContainerInstance,
			Status:               "INACTIVE",
			StatusReason:         "Instance deregistration forced",
			AgentConnected:       false,
			RunningTasksCount:    0,
			PendingTasksCount:    0,
		},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSDescribeContainerInstances handles the DescribeContainerInstances API endpoint in AWS ECS format
func (s *Server) handleECSDescribeContainerInstances(w http.ResponseWriter, body []byte) {
	var req DescribeContainerInstancesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance description logic
	// For now, return mock responses for requested instances
	containerInstances := []ContainerInstance{}
	for i, arn := range req.ContainerInstances {
		containerInstances = append(containerInstances, ContainerInstance{
			ContainerInstanceArn: arn,
			Ec2InstanceId:        "i-1234567890abcdef" + string(rune('0'+i)),
			Version:              1,
			Status:               "ACTIVE",
			AgentConnected:       true,
			RunningTasksCount:    0,
			PendingTasksCount:    0,
			RegisteredAt:         time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
			RegisteredResources: []Resource{
				{
					Name:         "CPU",
					Type:         "INTEGER",
					IntegerValue: 2048,
				},
				{
					Name:         "MEMORY",
					Type:         "INTEGER",
					IntegerValue: 4096,
				},
			},
			RemainingResources: []Resource{
				{
					Name:         "CPU",
					Type:         "INTEGER",
					IntegerValue: 2048,
				},
				{
					Name:         "MEMORY",
					Type:         "INTEGER",
					IntegerValue: 4096,
				},
			},
			VersionInfo: &VersionInfo{
				AgentVersion:  "1.51.0",
				AgentHash:     "4023248",
				DockerVersion: "20.10.7",
			},
		})
	}

	resp := DescribeContainerInstancesResponse{
		ContainerInstances: containerInstances,
		Failures:           []Failure{},
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// handleECSListContainerInstances handles the ListContainerInstances API endpoint in AWS ECS format
func (s *Server) handleECSListContainerInstances(w http.ResponseWriter, body []byte) {
	var req ListContainerInstancesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual container instance listing logic
	// For now, return a mock response
	cluster := "default"
	if req.Cluster != "" {
		cluster = req.Cluster
	}

	// Mock response with sample container instance ARNs
	containerInstanceArns := []string{
		"arn:aws:ecs:" + s.region + ":" + s.accountID + ":container-instance/" + cluster + "/i-1234567890abcdef0",
		"arn:aws:ecs:" + s.region + ":" + s.accountID + ":container-instance/" + cluster + "/i-1234567890abcdef1",
		"arn:aws:ecs:" + s.region + ":" + s.accountID + ":container-instance/" + cluster + "/i-1234567890abcdef2",
	}

	// Apply filtering if status is specified
	if req.Status != "" && req.Status != "ACTIVE" {
		// If filtering for non-ACTIVE status, return empty list
		containerInstanceArns = []string{}
	}

	// Apply pagination if requested
	maxResults := 100
	if req.MaxResults > 0 && req.MaxResults < len(containerInstanceArns) {
		maxResults = req.MaxResults
		containerInstanceArns = containerInstanceArns[:maxResults]
	}

	resp := ListContainerInstancesResponse{
		ContainerInstanceArns: containerInstanceArns,
	}

	// Add next token if there are more results
	if req.MaxResults > 0 && req.MaxResults < 3 {
		resp.NextToken = "next-page-token"
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}