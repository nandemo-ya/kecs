package api

import (
	"encoding/json"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// storageServiceToAPIService converts a storage.Service to an API Service
// This is kept for compatibility with tests that still use the old Service type
func storageServiceToAPIService(storageService *storage.Service) Service {
	service := Service{
		ServiceArn:                    storageService.ARN,
		ServiceName:                   storageService.ServiceName,
		ClusterArn:                    storageService.ClusterARN,
		Status:                        storageService.Status,
		DesiredCount:                  storageService.DesiredCount,
		RunningCount:                  storageService.RunningCount,
		PendingCount:                  storageService.PendingCount,
		LaunchType:                    storageService.LaunchType,
		PlatformVersion:               storageService.PlatformVersion,
		TaskDefinition:                storageService.TaskDefinitionARN,
		RoleArn:                       storageService.RoleARN,
		SchedulingStrategy:            storageService.SchedulingStrategy,
		EnableECSManagedTags:          storageService.EnableECSManagedTags,
		PropagateTags:                 storageService.PropagateTags,
		EnableExecuteCommand:          storageService.EnableExecuteCommand,
		HealthCheckGracePeriodSeconds: storageService.HealthCheckGracePeriodSeconds,
		CreatedAt:                     storageService.CreatedAt.Format(time.RFC3339),
	}

	// Parse JSON fields
	if storageService.LoadBalancers != "" && storageService.LoadBalancers != "null" {
		json.Unmarshal([]byte(storageService.LoadBalancers), &service.LoadBalancers)
	}
	if storageService.ServiceRegistries != "" && storageService.ServiceRegistries != "null" {
		json.Unmarshal([]byte(storageService.ServiceRegistries), &service.ServiceRegistries)
	}
	if storageService.NetworkConfiguration != "" && storageService.NetworkConfiguration != "null" {
		json.Unmarshal([]byte(storageService.NetworkConfiguration), &service.NetworkConfiguration)
	}
	if storageService.DeploymentConfiguration != "" && storageService.DeploymentConfiguration != "null" {
		json.Unmarshal([]byte(storageService.DeploymentConfiguration), &service.DeploymentConfiguration)
	}
	if storageService.PlacementConstraints != "" && storageService.PlacementConstraints != "null" {
		json.Unmarshal([]byte(storageService.PlacementConstraints), &service.PlacementConstraints)
	}
	if storageService.PlacementStrategy != "" && storageService.PlacementStrategy != "null" {
		json.Unmarshal([]byte(storageService.PlacementStrategy), &service.PlacementStrategy)
	}
	if storageService.CapacityProviderStrategy != "" && storageService.CapacityProviderStrategy != "null" {
		json.Unmarshal([]byte(storageService.CapacityProviderStrategy), &service.CapacityProviderStrategy)
	}
	if storageService.Tags != "" && storageService.Tags != "null" {
		json.Unmarshal([]byte(storageService.Tags), &service.Tags)
	}

	// Add deployment information
	// In AWS ECS, there's always at least one deployment representing the current state
	deployment := Deployment{
		Id:                       "ecs-svc/" + storageService.ServiceName,
		Status:                   "PRIMARY",
		TaskDefinition:           storageService.TaskDefinitionARN,
		DesiredCount:             storageService.DesiredCount,
		RunningCount:             storageService.RunningCount,
		PendingCount:             storageService.PendingCount,
		LaunchType:               storageService.LaunchType,
		PlatformVersion:          storageService.PlatformVersion,
		CreatedAt:                storageService.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                storageService.UpdatedAt.Format(time.RFC3339),
	}
	
	// Copy deployment configuration if it exists
	if service.DeploymentConfiguration != nil {
		// The deployment inherits the service's deployment configuration
	}
	
	service.Deployments = []Deployment{deployment}

	return service
}

// Service represents an ECS service (kept for test compatibility)
type Service struct {
	ServiceArn                   string                  `json:"serviceArn,omitempty"`
	ServiceName                  string                  `json:"serviceName"`
	ClusterArn                   string                  `json:"clusterArn,omitempty"`
	LoadBalancers                []LoadBalancer          `json:"loadBalancers,omitempty"`
	ServiceRegistries            []ServiceRegistry       `json:"serviceRegistries,omitempty"`
	Status                       string                  `json:"status,omitempty"`
	DesiredCount                 int                     `json:"desiredCount"`
	RunningCount                 int                     `json:"runningCount,omitempty"`
	PendingCount                 int                     `json:"pendingCount,omitempty"`
	LaunchType                   string                  `json:"launchType,omitempty"`
	CapacityProviderStrategy     []*CapacityStrategy      `json:"capacityProviderStrategy,omitempty"`
	PlatformVersion              string                  `json:"platformVersion,omitempty"`
	PlatformFamily               string                  `json:"platformFamily,omitempty"`
	TaskDefinition               string                  `json:"taskDefinition"`
	DeploymentConfiguration      *DeploymentConfiguration `json:"deploymentConfiguration,omitempty"`
	TaskSets                     []TaskSet               `json:"taskSets,omitempty"`
	Deployments                  []Deployment            `json:"deployments,omitempty"`
	RoleArn                      string                  `json:"roleArn,omitempty"`
	Events                       []ServiceEvent          `json:"events,omitempty"`
	CreatedAt                    string                  `json:"createdAt,omitempty"`
	PlacementConstraints         []*TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	PlacementStrategy            []*PlacementStrategy     `json:"placementStrategy,omitempty"`
	NetworkConfiguration         *NetworkConfiguration   `json:"networkConfiguration,omitempty"`
	HealthCheckGracePeriodSeconds int                    `json:"healthCheckGracePeriodSeconds,omitempty"`
	SchedulingStrategy           string                  `json:"schedulingStrategy,omitempty"`
	DeploymentController         *DeploymentController   `json:"deploymentController,omitempty"`
	Tags                         []Tag                   `json:"tags,omitempty"`
	CreatedBy                    string                  `json:"createdBy,omitempty"`
	EnableECSManagedTags         bool                    `json:"enableECSManagedTags,omitempty"`
	PropagateTags                string                  `json:"propagateTags,omitempty"`
	EnableExecuteCommand         bool                    `json:"enableExecuteCommand,omitempty"`
}

// Other types kept for test compatibility
type LoadBalancer struct {
	TargetGroupArn string `json:"targetGroupArn,omitempty"`
	LoadBalancerName string `json:"loadBalancerName,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	ContainerPort int `json:"containerPort,omitempty"`
}

type ServiceRegistry struct {
	RegistryArn string `json:"registryArn,omitempty"`
	Port int `json:"port,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	ContainerPort int `json:"containerPort,omitempty"`
}

type DeploymentConfiguration struct {
	DeploymentCircuitBreaker *DeploymentCircuitBreaker `json:"deploymentCircuitBreaker,omitempty"`
	MaximumPercent int `json:"maximumPercent,omitempty"`
	MinimumHealthyPercent int `json:"minimumHealthyPercent,omitempty"`
}

type DeploymentCircuitBreaker struct {
	Enable bool `json:"enable"`
	Rollback bool `json:"rollback"`
}

type NetworkConfiguration struct {
	AwsvpcConfiguration *AwsVpcConfiguration `json:"awsvpcConfiguration,omitempty"`
}

type AwsVpcConfiguration struct {
	Subnets []string `json:"subnets"`
	SecurityGroups []string `json:"securityGroups,omitempty"`
	AssignPublicIp string `json:"assignPublicIp,omitempty"`
}

type Deployment struct {
	Id string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
	TaskDefinition string `json:"taskDefinition,omitempty"`
	DesiredCount int `json:"desiredCount,omitempty"`
	PendingCount int `json:"pendingCount,omitempty"`
	RunningCount int `json:"runningCount,omitempty"`
	FailedTasks int `json:"failedTasks,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	CapacityProviderStrategy []*CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
	LaunchType string `json:"launchType,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
	PlatformFamily string `json:"platformFamily,omitempty"`
	NetworkConfiguration *NetworkConfiguration `json:"networkConfiguration,omitempty"`
	RolloutState string `json:"rolloutState,omitempty"`
	RolloutStateReason string `json:"rolloutStateReason,omitempty"`
}

type TaskSet struct {
	Id string `json:"id,omitempty"`
	TaskSetArn string `json:"taskSetArn,omitempty"`
	ServiceArn string `json:"serviceArn,omitempty"`
	ClusterArn string `json:"clusterArn,omitempty"`
	StartedBy string `json:"startedBy,omitempty"`
	ExternalId string `json:"externalId,omitempty"`
	Status string `json:"status,omitempty"`
	TaskDefinition string `json:"taskDefinition,omitempty"`
	ComputedDesiredCount int `json:"computedDesiredCount,omitempty"`
	PendingCount int `json:"pendingCount,omitempty"`
	RunningCount int `json:"runningCount,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	LaunchType string `json:"launchType,omitempty"`
	CapacityProviderStrategy []*CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
	PlatformFamily string `json:"platformFamily,omitempty"`
	NetworkConfiguration *NetworkConfiguration `json:"networkConfiguration,omitempty"`
	LoadBalancers []LoadBalancer `json:"loadBalancers,omitempty"`
	ServiceRegistries []ServiceRegistry `json:"serviceRegistries,omitempty"`
	Scale *Scale `json:"scale,omitempty"`
	StabilityStatus string `json:"stabilityStatus,omitempty"`
	StabilityStatusAt string `json:"stabilityStatusAt,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
}

type Scale struct {
	Value float64 `json:"value"`
	Unit string `json:"unit,omitempty"`
}

type ServiceEvent struct {
	Id string `json:"id,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	Message string `json:"message,omitempty"`
}

type DeploymentController struct {
	Type string `json:"type"`
}

// Request/Response types kept for test compatibility
type DeleteServiceRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Service string `json:"service"`
	Force bool `json:"force,omitempty"`
}

type DeleteServiceResponse struct {
	Service Service `json:"service"`
}