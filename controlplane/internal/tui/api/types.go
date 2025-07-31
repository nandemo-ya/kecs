// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"time"
)

// Instance represents a KECS instance from the API
type Instance struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Clusters  int       `json:"clusters"`
	Services  int       `json:"services"`
	Tasks     int       `json:"tasks"`
	APIPort   int       `json:"apiPort"`
	AdminPort int       `json:"adminPort"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateInstanceOptions contains options for creating a new instance
type CreateInstanceOptions struct {
	Name        string `json:"name"`
	APIPort     int    `json:"apiPort"`
	AdminPort   int    `json:"adminPort"`
	LocalStack  bool   `json:"localStack"`
	Traefik     bool   `json:"traefik"`
	DevMode     bool   `json:"devMode"`
}

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                string `json:"clusterArn"`
	ClusterName               string `json:"clusterName"`
	Status                    string `json:"status"`
	RegisteredContainerInstancesCount int `json:"registeredContainerInstancesCount"`
	RunningTasksCount         int    `json:"runningTasksCount"`
	PendingTasksCount         int    `json:"pendingTasksCount"`
	ActiveServicesCount       int    `json:"activeServicesCount"`
}

// Service represents an ECS service
type Service struct {
	ServiceArn     string    `json:"serviceArn"`
	ServiceName    string    `json:"serviceName"`
	ClusterArn     string    `json:"clusterArn"`
	Status         string    `json:"status"`
	DesiredCount   int       `json:"desiredCount"`
	RunningCount   int       `json:"runningCount"`
	PendingCount   int       `json:"pendingCount"`
	TaskDefinition string    `json:"taskDefinition"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Task represents an ECS task
type Task struct {
	TaskArn           string    `json:"taskArn"`
	ClusterArn        string    `json:"clusterArn"`
	TaskDefinitionArn string    `json:"taskDefinitionArn"`
	ServiceName       string    `json:"serviceName,omitempty"`
	LastStatus        string    `json:"lastStatus"`
	DesiredStatus     string    `json:"desiredStatus"`
	HealthStatus      string    `json:"healthStatus,omitempty"`
	Cpu               string    `json:"cpu,omitempty"`
	Memory            string    `json:"memory,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	StartedAt         *time.Time `json:"startedAt,omitempty"`
	StoppedAt         *time.Time `json:"stoppedAt,omitempty"`
	Containers        []Container `json:"containers,omitempty"`
}

// Container represents a container within a task
type Container struct {
	ContainerArn string `json:"containerArn"`
	Name         string `json:"name"`
	LastStatus   string `json:"lastStatus"`
	ExitCode     *int   `json:"exitCode,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

// ECS API Request/Response types

// ListClustersRequest represents the request for ListClusters
type ListClustersRequest struct {
	NextToken  *string `json:"nextToken,omitempty"`
	MaxResults *int    `json:"maxResults,omitempty"`
}

// ListClustersResponse represents the response from ListClusters
type ListClustersResponse struct {
	ClusterArns []string `json:"clusterArns"`
	NextToken   *string  `json:"nextToken,omitempty"`
}

// DescribeClustersRequest represents the request for DescribeClusters
type DescribeClustersRequest struct {
	Clusters []string `json:"clusters"`
	Include  []string `json:"include,omitempty"`
}

// DescribeClustersResponse represents the response from DescribeClusters
type DescribeClustersResponse struct {
	Clusters []Cluster `json:"clusters"`
	Failures []Failure `json:"failures,omitempty"`
}

// ListServicesRequest represents the request for ListServices
type ListServicesRequest struct {
	Cluster     string  `json:"cluster"`
	NextToken   *string `json:"nextToken,omitempty"`
	MaxResults  *int    `json:"maxResults,omitempty"`
	LaunchType  string  `json:"launchType,omitempty"`
	SchedulingStrategy string `json:"schedulingStrategy,omitempty"`
}

// ListServicesResponse represents the response from ListServices
type ListServicesResponse struct {
	ServiceArns []string `json:"serviceArns"`
	NextToken   *string  `json:"nextToken,omitempty"`
}

// DescribeServicesRequest represents the request for DescribeServices
type DescribeServicesRequest struct {
	Cluster  string   `json:"cluster"`
	Services []string `json:"services"`
	Include  []string `json:"include,omitempty"`
}

// DescribeServicesResponse represents the response from DescribeServices
type DescribeServicesResponse struct {
	Services []Service `json:"services"`
	Failures []Failure `json:"failures,omitempty"`
}

// ListTasksRequest represents the request for ListTasks
type ListTasksRequest struct {
	Cluster       string  `json:"cluster"`
	ServiceName   string  `json:"serviceName,omitempty"`
	DesiredStatus string  `json:"desiredStatus,omitempty"`
	NextToken     *string `json:"nextToken,omitempty"`
	MaxResults    *int    `json:"maxResults,omitempty"`
}

// ListTasksResponse represents the response from ListTasks
type ListTasksResponse struct {
	TaskArns  []string `json:"taskArns"`
	NextToken *string  `json:"nextToken,omitempty"`
}

// DescribeTasksRequest represents the request for DescribeTasks
type DescribeTasksRequest struct {
	Cluster string   `json:"cluster"`
	Tasks   []string `json:"tasks"`
	Include []string `json:"include,omitempty"`
}

// DescribeTasksResponse represents the response from DescribeTasks
type DescribeTasksResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures,omitempty"`
}

// Failure represents an API failure
type Failure struct {
	Arn    string `json:"arn"`
	Reason string `json:"reason"`
}

// Error response
type ErrorResponse struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}