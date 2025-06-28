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

import "time"

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                     string    `json:"clusterArn"`
	ClusterName                    string    `json:"clusterName"`
	Status                         string    `json:"status"`
	RegisteredContainerInstancesCount int     `json:"registeredContainerInstancesCount"`
	RunningTasksCount              int       `json:"runningTasksCount"`
	PendingTasksCount              int       `json:"pendingTasksCount"`
	ActiveServicesCount            int       `json:"activeServicesCount"`
	Tags                           []Tag     `json:"tags,omitempty"`
	CreatedAt                      time.Time `json:"createdAt,omitempty"`
}

// Tag represents a key-value tag
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ListClustersRequest represents a request to list clusters
type ListClustersRequest struct {
	NextToken  string `json:"nextToken,omitempty"`
	MaxResults int    `json:"maxResults,omitempty"`
}

// ListClustersResponse represents a response from ListClusters
type ListClustersResponse struct {
	ClusterArns []string `json:"clusterArns"`
	NextToken   string   `json:"nextToken,omitempty"`
}

// DescribeClustersRequest represents a request to describe clusters
type DescribeClustersRequest struct {
	Clusters []string `json:"clusters"`
	Include  []string `json:"include,omitempty"`
}

// DescribeClustersResponse represents a response from DescribeClusters
type DescribeClustersResponse struct {
	Clusters []Cluster `json:"clusters"`
	Failures []Failure `json:"failures,omitempty"`
}

// CreateClusterRequest represents a request to create a cluster
type CreateClusterRequest struct {
	ClusterName string `json:"clusterName"`
	Tags        []Tag  `json:"tags,omitempty"`
}

// CreateClusterResponse represents a response from CreateCluster
type CreateClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// DeleteClusterRequest represents a request to delete a cluster
type DeleteClusterRequest struct {
	Cluster string `json:"cluster"`
}

// DeleteClusterResponse represents a response from DeleteCluster
type DeleteClusterResponse struct {
	Cluster Cluster `json:"cluster"`
}

// Failure represents an API failure
type Failure struct {
	Arn    string `json:"arn"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// Service represents an ECS service
type Service struct {
	ServiceArn        string       `json:"serviceArn"`
	ServiceName       string       `json:"serviceName"`
	ClusterArn        string       `json:"clusterArn"`
	Status            string       `json:"status"`
	DesiredCount      int          `json:"desiredCount"`
	RunningCount      int          `json:"runningCount"`
	PendingCount      int          `json:"pendingCount"`
	TaskDefinition    string       `json:"taskDefinition"`
	LaunchType        string       `json:"launchType,omitempty"`
	PlatformVersion   string       `json:"platformVersion,omitempty"`
	PlatformFamily    string       `json:"platformFamily,omitempty"`
	SchedulingStrategy string      `json:"schedulingStrategy,omitempty"`
	Tags              []Tag        `json:"tags,omitempty"`
	CreatedAt         time.Time    `json:"createdAt,omitempty"`
	CreatedBy         string       `json:"createdBy,omitempty"`
	HealthCheckGracePeriodSeconds int `json:"healthCheckGracePeriodSeconds,omitempty"`
}

// ListServicesRequest represents a request to list services
type ListServicesRequest struct {
	Cluster       string `json:"cluster"`
	NextToken     string `json:"nextToken,omitempty"`
	MaxResults    int    `json:"maxResults,omitempty"`
	LaunchType    string `json:"launchType,omitempty"`
	SchedulingStrategy string `json:"schedulingStrategy,omitempty"`
}

// ListServicesResponse represents a response from ListServices
type ListServicesResponse struct {
	ServiceArns []string `json:"serviceArns"`
	NextToken   string   `json:"nextToken,omitempty"`
}

// DescribeServicesRequest represents a request to describe services
type DescribeServicesRequest struct {
	Cluster  string   `json:"cluster"`
	Services []string `json:"services"`
	Include  []string `json:"include,omitempty"`
}

// DescribeServicesResponse represents a response from DescribeServices
type DescribeServicesResponse struct {
	Services []Service `json:"services"`
	Failures []Failure `json:"failures,omitempty"`
}

// CreateServiceRequest represents a request to create a service
type CreateServiceRequest struct {
	Cluster             string `json:"cluster"`
	ServiceName         string `json:"serviceName"`
	TaskDefinition      string `json:"taskDefinition"`
	DesiredCount        int    `json:"desiredCount"`
	LaunchType          string `json:"launchType,omitempty"`
	Tags                []Tag  `json:"tags,omitempty"`
}

// CreateServiceResponse represents a response from CreateService
type CreateServiceResponse struct {
	Service Service `json:"service"`
}

// UpdateServiceRequest represents a request to update a service
type UpdateServiceRequest struct {
	Cluster        string `json:"cluster"`
	Service        string `json:"service"`
	DesiredCount   *int   `json:"desiredCount,omitempty"`
	TaskDefinition string `json:"taskDefinition,omitempty"`
}

// UpdateServiceResponse represents a response from UpdateService
type UpdateServiceResponse struct {
	Service Service `json:"service"`
}

// DeleteServiceRequest represents a request to delete a service
type DeleteServiceRequest struct {
	Cluster string `json:"cluster"`
	Service string `json:"service"`
	Force   bool   `json:"force,omitempty"`
}

// DeleteServiceResponse represents a response from DeleteService
type DeleteServiceResponse struct {
	Service Service `json:"service"`
}

// Task represents an ECS task
type Task struct {
	TaskArn           string          `json:"taskArn"`
	ClusterArn        string          `json:"clusterArn"`
	TaskDefinitionArn string          `json:"taskDefinitionArn"`
	ContainerInstanceArn string       `json:"containerInstanceArn,omitempty"`
	Overrides         interface{}     `json:"overrides,omitempty"`
	LastStatus        string          `json:"lastStatus"`
	DesiredStatus     string          `json:"desiredStatus"`
	Cpu               string          `json:"cpu,omitempty"`
	Memory            string          `json:"memory,omitempty"`
	Containers        []Container     `json:"containers,omitempty"`
	StartedBy         string          `json:"startedBy,omitempty"`
	Version           int64           `json:"version"`
	StoppedReason     string          `json:"stoppedReason,omitempty"`
	StopCode          string          `json:"stopCode,omitempty"`
	Connectivity      string          `json:"connectivity,omitempty"`
	ConnectivityAt    *time.Time      `json:"connectivityAt,omitempty"`
	PullStartedAt     *time.Time      `json:"pullStartedAt,omitempty"`
	PullStoppedAt     *time.Time      `json:"pullStoppedAt,omitempty"`
	ExecutionStoppedAt *time.Time     `json:"executionStoppedAt,omitempty"`
	CreatedAt         *time.Time      `json:"createdAt,omitempty"`
	StartedAt         *time.Time      `json:"startedAt,omitempty"`
	StoppingAt        *time.Time      `json:"stoppingAt,omitempty"`
	StoppedAt         *time.Time      `json:"stoppedAt,omitempty"`
	Group             string          `json:"group,omitempty"`
	LaunchType        string          `json:"launchType,omitempty"`
	PlatformVersion   string          `json:"platformVersion,omitempty"`
	Tags              []Tag           `json:"tags,omitempty"`
}

// Container represents a container within a task
type Container struct {
	ContainerArn    string          `json:"containerArn"`
	TaskArn         string          `json:"taskArn"`
	Name            string          `json:"name"`
	Image           string          `json:"image,omitempty"`
	ImageDigest     string          `json:"imageDigest,omitempty"`
	RuntimeId       string          `json:"runtimeId,omitempty"`
	LastStatus      string          `json:"lastStatus"`
	ExitCode        *int            `json:"exitCode,omitempty"`
	Reason          string          `json:"reason,omitempty"`
	NetworkBindings []interface{}   `json:"networkBindings,omitempty"`
	NetworkInterfaces []interface{} `json:"networkInterfaces,omitempty"`
	HealthStatus    string          `json:"healthStatus,omitempty"`
	Cpu             string          `json:"cpu,omitempty"`
	Memory          string          `json:"memory,omitempty"`
	MemoryReservation string        `json:"memoryReservation,omitempty"`
}

// ListTasksRequest represents a request to list tasks
type ListTasksRequest struct {
	Cluster           string `json:"cluster,omitempty"`
	ContainerInstance string `json:"containerInstance,omitempty"`
	Family            string `json:"family,omitempty"`
	NextToken         string `json:"nextToken,omitempty"`
	MaxResults        int    `json:"maxResults,omitempty"`
	StartedBy         string `json:"startedBy,omitempty"`
	ServiceName       string `json:"serviceName,omitempty"`
	DesiredStatus     string `json:"desiredStatus,omitempty"`
	LaunchType        string `json:"launchType,omitempty"`
}

// ListTasksResponse represents a response from ListTasks
type ListTasksResponse struct {
	TaskArns  []string `json:"taskArns"`
	NextToken string   `json:"nextToken,omitempty"`
}

// DescribeTasksRequest represents a request to describe tasks
type DescribeTasksRequest struct {
	Cluster string   `json:"cluster,omitempty"`
	Tasks   []string `json:"tasks"`
	Include []string `json:"include,omitempty"`
}

// DescribeTasksResponse represents a response from DescribeTasks
type DescribeTasksResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures,omitempty"`
}

// StopTaskRequest represents a request to stop a task
type StopTaskRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Task    string `json:"task"`
	Reason  string `json:"reason,omitempty"`
}

// StopTaskResponse represents a response from StopTask
type StopTaskResponse struct {
	Task Task `json:"task"`
}

// TaskDefinition represents an ECS task definition
type TaskDefinition struct {
	TaskDefinitionArn    string                 `json:"taskDefinitionArn"`
	Family               string                 `json:"family"`
	Revision             int                    `json:"revision"`
	TaskRoleArn          string                 `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn     string                 `json:"executionRoleArn,omitempty"`
	NetworkMode          string                 `json:"networkMode,omitempty"`
	ContainerDefinitions []ContainerDefinition  `json:"containerDefinitions"`
	Volumes              []interface{}          `json:"volumes,omitempty"`
	PlacementConstraints []interface{}          `json:"placementConstraints,omitempty"`
	RequiresCompatibilities []string            `json:"requiresCompatibilities,omitempty"`
	Cpu                  string                 `json:"cpu,omitempty"`
	Memory               string                 `json:"memory,omitempty"`
	Tags                 []Tag                  `json:"tags,omitempty"`
	Status               string                 `json:"status"`
	RegisteredAt         *time.Time             `json:"registeredAt,omitempty"`
	DeregisteredAt       *time.Time             `json:"deregisteredAt,omitempty"`
	RegisteredBy         string                 `json:"registeredBy,omitempty"`
}

// ContainerDefinition represents a container definition within a task definition
type ContainerDefinition struct {
	Name                 string                 `json:"name"`
	Image                string                 `json:"image"`
	Cpu                  int                    `json:"cpu,omitempty"`
	Memory               int                    `json:"memory,omitempty"`
	MemoryReservation    int                    `json:"memoryReservation,omitempty"`
	Links                []string               `json:"links,omitempty"`
	PortMappings         []PortMapping          `json:"portMappings,omitempty"`
	Essential            bool                   `json:"essential"`
	EntryPoint           []string               `json:"entryPoint,omitempty"`
	Command              []string               `json:"command,omitempty"`
	Environment          []EnvironmentVariable  `json:"environment,omitempty"`
	MountPoints          []interface{}          `json:"mountPoints,omitempty"`
	VolumesFrom          []interface{}          `json:"volumesFrom,omitempty"`
	Secrets              []interface{}          `json:"secrets,omitempty"`
	LogConfiguration     interface{}            `json:"logConfiguration,omitempty"`
	HealthCheck          interface{}            `json:"healthCheck,omitempty"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ListTaskDefinitionsRequest represents a request to list task definitions
type ListTaskDefinitionsRequest struct {
	FamilyPrefix string `json:"familyPrefix,omitempty"`
	Status       string `json:"status,omitempty"`
	Sort         string `json:"sort,omitempty"`
	NextToken    string `json:"nextToken,omitempty"`
	MaxResults   int    `json:"maxResults,omitempty"`
}

// ListTaskDefinitionsResponse represents a response from ListTaskDefinitions
type ListTaskDefinitionsResponse struct {
	TaskDefinitionArns []string `json:"taskDefinitionArns"`
	NextToken          string   `json:"nextToken,omitempty"`
}

// DescribeTaskDefinitionRequest represents a request to describe a task definition
type DescribeTaskDefinitionRequest struct {
	TaskDefinition string   `json:"taskDefinition"`
	Include        []string `json:"include,omitempty"`
}

// DescribeTaskDefinitionResponse represents a response from DescribeTaskDefinition
type DescribeTaskDefinitionResponse struct {
	TaskDefinition TaskDefinition `json:"taskDefinition"`
	Tags           []Tag          `json:"tags,omitempty"`
}

// RegisterTaskDefinitionRequest represents a request to register a task definition
type RegisterTaskDefinitionRequest struct {
	Family                  string                `json:"family"`
	TaskRoleArn             string                `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn        string                `json:"executionRoleArn,omitempty"`
	NetworkMode             string                `json:"networkMode,omitempty"`
	ContainerDefinitions    []ContainerDefinition `json:"containerDefinitions"`
	Volumes                 []interface{}         `json:"volumes,omitempty"`
	PlacementConstraints    []interface{}         `json:"placementConstraints,omitempty"`
	RequiresCompatibilities []string              `json:"requiresCompatibilities,omitempty"`
	Cpu                     string                `json:"cpu,omitempty"`
	Memory                  string                `json:"memory,omitempty"`
	Tags                    []Tag                 `json:"tags,omitempty"`
}

// RegisterTaskDefinitionResponse represents a response from RegisterTaskDefinition
type RegisterTaskDefinitionResponse struct {
	TaskDefinition TaskDefinition `json:"taskDefinition"`
}

// DeregisterTaskDefinitionRequest represents a request to deregister a task definition
type DeregisterTaskDefinitionRequest struct {
	TaskDefinition string `json:"taskDefinition"`
}

// DeregisterTaskDefinitionResponse represents a response from DeregisterTaskDefinition
type DeregisterTaskDefinitionResponse struct {
	TaskDefinition TaskDefinition `json:"taskDefinition"`
}