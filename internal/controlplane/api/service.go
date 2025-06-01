package api

import (
	"encoding/json"
	"net/http"
)

// Service represents an ECS service
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
	CapacityProviderStrategy     []CapacityStrategy      `json:"capacityProviderStrategy,omitempty"`
	PlatformVersion              string                  `json:"platformVersion,omitempty"`
	PlatformFamily               string                  `json:"platformFamily,omitempty"`
	TaskDefinition               string                  `json:"taskDefinition"`
	DeploymentConfiguration      *DeploymentConfiguration `json:"deploymentConfiguration,omitempty"`
	TaskSets                     []TaskSet               `json:"taskSets,omitempty"`
	Deployments                  []Deployment            `json:"deployments,omitempty"`
	RoleArn                      string                  `json:"roleArn,omitempty"`
	Events                       []ServiceEvent          `json:"events,omitempty"`
	CreatedAt                    string                  `json:"createdAt,omitempty"`
	PlacementConstraints         []TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	PlacementStrategy            []PlacementStrategy     `json:"placementStrategy,omitempty"`
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

// LoadBalancer represents a load balancer configuration for a service
type LoadBalancer struct {
	TargetGroupArn string `json:"targetGroupArn,omitempty"`
	LoadBalancerName string `json:"loadBalancerName,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	ContainerPort int `json:"containerPort,omitempty"`
}

// ServiceRegistry represents a service registry for a service
type ServiceRegistry struct {
	RegistryArn string `json:"registryArn,omitempty"`
	Port int `json:"port,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	ContainerPort int `json:"containerPort,omitempty"`
}

// DeploymentConfiguration represents a deployment configuration for a service
type DeploymentConfiguration struct {
	DeploymentCircuitBreaker *DeploymentCircuitBreaker `json:"deploymentCircuitBreaker,omitempty"`
	MaximumPercent int `json:"maximumPercent,omitempty"`
	MinimumHealthyPercent int `json:"minimumHealthyPercent,omitempty"`
	AlarmConfiguration *AlarmConfiguration `json:"alarmConfiguration,omitempty"`
}

// DeploymentCircuitBreaker represents a deployment circuit breaker for a service
type DeploymentCircuitBreaker struct {
	Enable bool `json:"enable"`
	Rollback bool `json:"rollback"`
}

// AlarmConfiguration represents an alarm configuration for a service
type AlarmConfiguration struct {
	Alarms []Alarm `json:"alarms"`
	Enable bool `json:"enable"`
	RollBack bool `json:"rollBack"`
}

// Alarm represents an alarm for a service
type Alarm struct {
	AlarmName string `json:"alarmName"`
	AlarmArn string `json:"alarmArn,omitempty"`
}

// TaskSet represents a task set for a service
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
	CapacityProviderStrategy []CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
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

// Scale represents a scale configuration for a task set
type Scale struct {
	Value float64 `json:"value"`
	Unit string `json:"unit,omitempty"`
}

// Deployment represents a deployment for a service
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
	CapacityProviderStrategy []CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
	LaunchType string `json:"launchType,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
	PlatformFamily string `json:"platformFamily,omitempty"`
	NetworkConfiguration *NetworkConfiguration `json:"networkConfiguration,omitempty"`
	RolloutState string `json:"rolloutState,omitempty"`
	RolloutStateReason string `json:"rolloutStateReason,omitempty"`
	ServiceConnectConfiguration *ServiceConnectConfiguration `json:"serviceConnectConfiguration,omitempty"`
	ServiceConnectResources *ServiceConnectServiceResource `json:"serviceConnectResources,omitempty"`
}

// ServiceEvent represents an event for a service
type ServiceEvent struct {
	Id string `json:"id,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	Message string `json:"message,omitempty"`
}

// NetworkConfiguration represents a network configuration for a service
type NetworkConfiguration struct {
	AwsvpcConfiguration *AwsVpcConfiguration `json:"awsvpcConfiguration,omitempty"`
}

// AwsVpcConfiguration represents an AWS VPC configuration for a service
type AwsVpcConfiguration struct {
	Subnets []string `json:"subnets"`
	SecurityGroups []string `json:"securityGroups,omitempty"`
	AssignPublicIp string `json:"assignPublicIp,omitempty"`
}

// DeploymentController represents a deployment controller for a service
type DeploymentController struct {
	Type string `json:"type"`
}

// ServiceConnectConfiguration represents a service connect configuration for a service
type ServiceConnectConfiguration struct {
	Enabled bool `json:"enabled"`
	Namespace string `json:"namespace,omitempty"`
	Services []ServiceConnectService `json:"services,omitempty"`
	LogConfiguration *LogConfiguration `json:"logConfiguration,omitempty"`
}

// ServiceConnectService represents a service connect service
type ServiceConnectService struct {
	PortName string `json:"portName"`
	DiscoveryName string `json:"discoveryName,omitempty"`
	ClientAliases []ServiceConnectClientAlias `json:"clientAliases,omitempty"`
	IngressPortOverride int `json:"ingressPortOverride,omitempty"`
	PortMappingName string `json:"portMappingName,omitempty"`
	DiscoveryArn string `json:"discoveryArn,omitempty"`
	Timeout *TimeoutConfiguration `json:"timeout,omitempty"`
}

// ServiceConnectClientAlias represents a service connect client alias
type ServiceConnectClientAlias struct {
	Port int `json:"port"`
	DnsName string `json:"dnsName,omitempty"`
}

// TimeoutConfiguration represents a timeout configuration for a service connect service
type TimeoutConfiguration struct {
	IdleTimeoutSeconds int `json:"idleTimeoutSeconds,omitempty"`
	PerRequestTimeoutSeconds int `json:"perRequestTimeoutSeconds,omitempty"`
}

// ServiceConnectServiceResource represents a service connect service resource
type ServiceConnectServiceResource struct {
	DiscoveryName string `json:"discoveryName,omitempty"`
	DiscoveryArn string `json:"discoveryArn,omitempty"`
}

// CreateServiceRequest represents the request to create a service
type CreateServiceRequest struct {
	ServiceName string `json:"serviceName"`
	TaskDefinition string `json:"taskDefinition"`
	Cluster string `json:"cluster,omitempty"`
	LoadBalancers []LoadBalancer `json:"loadBalancers,omitempty"`
	ServiceRegistries []ServiceRegistry `json:"serviceRegistries,omitempty"`
	DesiredCount int `json:"desiredCount,omitempty"`
	ClientToken string `json:"clientToken,omitempty"`
	LaunchType string `json:"launchType,omitempty"`
	CapacityProviderStrategy []CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
	PlatformFamily string `json:"platformFamily,omitempty"`
	Role string `json:"role,omitempty"`
	DeploymentConfiguration *DeploymentConfiguration `json:"deploymentConfiguration,omitempty"`
	PlacementConstraints []TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	PlacementStrategy []PlacementStrategy `json:"placementStrategy,omitempty"`
	NetworkConfiguration *NetworkConfiguration `json:"networkConfiguration,omitempty"`
	HealthCheckGracePeriodSeconds int `json:"healthCheckGracePeriodSeconds,omitempty"`
	SchedulingStrategy string `json:"schedulingStrategy,omitempty"`
	DeploymentController *DeploymentController `json:"deploymentController,omitempty"`
	Tags []Tag `json:"tags,omitempty"`
	EnableECSManagedTags bool `json:"enableECSManagedTags,omitempty"`
	PropagateTags string `json:"propagateTags,omitempty"`
	EnableExecuteCommand bool `json:"enableExecuteCommand,omitempty"`
	ServiceConnectConfiguration *ServiceConnectConfiguration `json:"serviceConnectConfiguration,omitempty"`
}

// CreateServiceResponse represents the response from creating a service
type CreateServiceResponse struct {
	Service Service `json:"service"`
}

// UpdateServiceRequest represents the request to update a service
type UpdateServiceRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Service string `json:"service"`
	DesiredCount int `json:"desiredCount,omitempty"`
	TaskDefinition string `json:"taskDefinition,omitempty"`
	CapacityProviderStrategy []CapacityStrategy `json:"capacityProviderStrategy,omitempty"`
	DeploymentConfiguration *DeploymentConfiguration `json:"deploymentConfiguration,omitempty"`
	NetworkConfiguration *NetworkConfiguration `json:"networkConfiguration,omitempty"`
	PlacementConstraints []TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	PlacementStrategy []PlacementStrategy `json:"placementStrategy,omitempty"`
	PlatformVersion string `json:"platformVersion,omitempty"`
	ForceNewDeployment bool `json:"forceNewDeployment,omitempty"`
	HealthCheckGracePeriodSeconds int `json:"healthCheckGracePeriodSeconds,omitempty"`
	EnableExecuteCommand bool `json:"enableExecuteCommand,omitempty"`
	EnableECSManagedTags bool `json:"enableECSManagedTags,omitempty"`
	LoadBalancers []LoadBalancer `json:"loadBalancers,omitempty"`
	ServiceRegistries []ServiceRegistry `json:"serviceRegistries,omitempty"`
	ServiceConnectConfiguration *ServiceConnectConfiguration `json:"serviceConnectConfiguration,omitempty"`
}

// UpdateServiceResponse represents the response from updating a service
type UpdateServiceResponse struct {
	Service Service `json:"service"`
}

// DeleteServiceRequest represents the request to delete a service
type DeleteServiceRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Service string `json:"service"`
	Force bool `json:"force,omitempty"`
}

// DeleteServiceResponse represents the response from deleting a service
type DeleteServiceResponse struct {
	Service Service `json:"service"`
}

// DescribeServicesRequest represents the request to describe services
type DescribeServicesRequest struct {
	Cluster string `json:"cluster,omitempty"`
	Services []string `json:"services"`
	Include []string `json:"include,omitempty"`
}

// DescribeServicesResponse represents the response from describing services
type DescribeServicesResponse struct {
	Services []Service `json:"services"`
	Failures []Failure `json:"failures,omitempty"`
}

// ListServicesRequest represents the request to list services
type ListServicesRequest struct {
	Cluster string `json:"cluster,omitempty"`
	NextToken string `json:"nextToken,omitempty"`
	MaxResults int `json:"maxResults,omitempty"`
	LaunchType string `json:"launchType,omitempty"`
	SchedulingStrategy string `json:"schedulingStrategy,omitempty"`
}

// ListServicesResponse represents the response from listing services
type ListServicesResponse struct {
	ServiceArns []string `json:"serviceArns"`
	NextToken string `json:"nextToken,omitempty"`
}

// registerServiceEndpoints registers all service-related API endpoints
func (s *Server) registerServiceEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/createservice", s.handleCreateService)
	mux.HandleFunc("/v1/updateservice", s.handleUpdateService)
	mux.HandleFunc("/v1/deleteservice", s.handleDeleteService)
	mux.HandleFunc("/v1/describeservices", s.handleDescribeServices)
	mux.HandleFunc("/v1/listservices", s.handleListServices)
}

// handleCreateService handles the CreateService API endpoint
func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := s.CreateServiceWithStorage(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleUpdateService handles the UpdateService API endpoint
func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := s.UpdateServiceWithStorage(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteService handles the DeleteService API endpoint
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := s.DeleteServiceWithStorage(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeServices handles the DescribeServices API endpoint
func (s *Server) handleDescribeServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeServicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := s.DescribeServicesWithStorage(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListServices handles the ListServices API endpoint
func (s *Server) handleListServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListServicesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	resp, err := s.ListServicesWithStorage(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
