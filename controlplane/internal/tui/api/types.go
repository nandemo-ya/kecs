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
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Clusters   int       `json:"clusters"`
	Services   int       `json:"services"`
	Tasks      int       `json:"tasks"`
	APIPort    int       `json:"apiPort"`
	AdminPort  int       `json:"adminPort"`
	LocalStack bool      `json:"localStack"`
	Traefik    bool      `json:"traefik"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CreateInstanceOptions contains options for creating a new instance
type CreateInstanceOptions struct {
	Name               string `json:"name"`
	APIPort            int    `json:"apiPort"`
	AdminPort          int    `json:"adminPort"`
	LocalStack         bool   `json:"localStack"`
	AdditionalServices string `json:"additionalServices,omitempty"`
}

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                        string `json:"clusterArn"`
	ClusterName                       string `json:"clusterName"`
	Status                            string `json:"status"`
	RegisteredContainerInstancesCount int    `json:"registeredContainerInstancesCount"`
	RunningTasksCount                 int    `json:"runningTasksCount"`
	PendingTasksCount                 int    `json:"pendingTasksCount"`
	ActiveServicesCount               int    `json:"activeServicesCount"`
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
	TaskArn           string      `json:"taskArn"`
	ClusterArn        string      `json:"clusterArn"`
	TaskDefinitionArn string      `json:"taskDefinitionArn"`
	ServiceName       string      `json:"serviceName,omitempty"`
	LastStatus        string      `json:"lastStatus"`
	DesiredStatus     string      `json:"desiredStatus"`
	HealthStatus      string      `json:"healthStatus,omitempty"`
	Cpu               string      `json:"cpu,omitempty"`
	Memory            string      `json:"memory,omitempty"`
	CreatedAt         time.Time   `json:"createdAt"`
	StartedAt         *time.Time  `json:"startedAt,omitempty"`
	StoppedAt         *time.Time  `json:"stoppedAt,omitempty"`
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

// CreateClusterRequest represents the request for CreateCluster
type CreateClusterRequest struct {
	ClusterName string           `json:"clusterName"`
	Tags        []Tag            `json:"tags,omitempty"`
	Settings    []ClusterSetting `json:"settings,omitempty"`
}

// CreateClusterResponse represents the response from CreateCluster
type CreateClusterResponse struct {
	Cluster *Cluster `json:"cluster,omitempty"`
}

// Tag represents a resource tag
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ClusterSetting represents a cluster setting
type ClusterSetting struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

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
	Cluster            string  `json:"cluster"`
	NextToken          *string `json:"nextToken,omitempty"`
	MaxResults         *int    `json:"maxResults,omitempty"`
	LaunchType         string  `json:"launchType,omitempty"`
	SchedulingStrategy string  `json:"schedulingStrategy,omitempty"`
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

// TaskDefinition represents an ECS task definition
type TaskDefinition struct {
	TaskDefinitionArn       string                `json:"taskDefinitionArn"`
	Family                  string                `json:"family"`
	Revision                int                   `json:"revision"`
	Status                  string                `json:"status"`
	TaskRoleArn             string                `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn        string                `json:"executionRoleArn,omitempty"`
	NetworkMode             string                `json:"networkMode"`
	ContainerDefinitions    []ContainerDefinition `json:"containerDefinitions"`
	RequiresCompatibilities []string              `json:"requiresCompatibilities"`
	Cpu                     string                `json:"cpu,omitempty"`
	Memory                  string                `json:"memory,omitempty"`
	RegisteredAt            time.Time             `json:"registeredAt"`
}

// KeyValuePair represents a key-value pair for environment variables
type KeyValuePair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ContainerDefinition represents a container within a task definition
type ContainerDefinition struct {
	Name              string                `json:"name"`
	Image             string                `json:"image"`
	Cpu               int                   `json:"cpu,omitempty"`
	Memory            int                   `json:"memory,omitempty"`
	MemoryReservation int                   `json:"memoryReservation,omitempty"`
	PortMappings      []PortMapping         `json:"portMappings,omitempty"`
	Essential         bool                  `json:"essential"`
	Environment       []KeyValuePair        `json:"environment,omitempty"`
	MountPoints       []MountPoint          `json:"mountPoints,omitempty"`
	VolumesFrom       []VolumeFrom          `json:"volumesFrom,omitempty"`
	DependsOn         []ContainerDependency `json:"dependsOn,omitempty"`
	HealthCheck       *HealthCheck          `json:"healthCheck,omitempty"`
	LogConfiguration  *LogConfiguration     `json:"logConfiguration,omitempty"`
	Secrets           []Secret              `json:"secrets,omitempty"`
	Command           []string              `json:"command,omitempty"`
	EntryPoint        []string              `json:"entryPoint,omitempty"`
	WorkingDirectory  string                `json:"workingDirectory,omitempty"`
	User              string                `json:"user,omitempty"`
}

// PortMapping represents a port mapping for a container
type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol"`
}

// MountPoint represents a mount point
type MountPoint struct {
	SourceVolume  string `json:"sourceVolume"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
}

// VolumeFrom represents a volume to mount from another container
type VolumeFrom struct {
	SourceContainer string `json:"sourceContainer"`
	ReadOnly        bool   `json:"readOnly,omitempty"`
}

// ContainerDependency represents a dependency between containers
type ContainerDependency struct {
	ContainerName string `json:"containerName"`
	Condition     string `json:"condition"` // START, COMPLETE, SUCCESS, HEALTHY
}

// HealthCheck represents container health check configuration
type HealthCheck struct {
	Command     []string `json:"command"`
	Interval    int      `json:"interval,omitempty"`
	Timeout     int      `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod int      `json:"startPeriod,omitempty"`
}

// LogConfiguration represents container logging configuration
type LogConfiguration struct {
	LogDriver string            `json:"logDriver"`
	Options   map[string]string `json:"options,omitempty"`
}

// Secret represents a secret to inject into the container
type Secret struct {
	Name      string `json:"name"`
	ValueFrom string `json:"valueFrom"`
}

// TaskDefinitionRevision represents summary info for a task definition revision
type TaskDefinitionRevision struct {
	Family    string    `json:"family"`
	Revision  int       `json:"revision"`
	Status    string    `json:"status"`
	Cpu       string    `json:"cpu"`
	Memory    string    `json:"memory"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreationStatus represents the status of instance creation
type CreationStatus struct {
	Step    string `json:"step"`    // Current step name
	Status  string `json:"status"`  // "pending", "running", "done", "failed"
	Message string `json:"message"` // Optional message
}
