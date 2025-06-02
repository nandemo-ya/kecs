package storage

import (
	"context"
	"time"
)

// Storage defines the interface for all storage operations
type Storage interface {
	// Initialize the storage backend
	Initialize(ctx context.Context) error
	
	// Close the storage connection
	Close() error
	
	// Cluster operations
	ClusterStore() ClusterStore
	
	// Task Definition operations
	TaskDefinitionStore() TaskDefinitionStore
	
	// Service operations
	ServiceStore() ServiceStore
	
	// Task operations
	TaskStore() TaskStore
	
	// Transaction support
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error
}

// ClusterStore defines cluster-specific storage operations
type ClusterStore interface {
	// Create a new cluster
	Create(ctx context.Context, cluster *Cluster) error
	
	// Get a cluster by name
	Get(ctx context.Context, name string) (*Cluster, error)
	
	// List all clusters
	List(ctx context.Context) ([]*Cluster, error)
	
	// Update a cluster
	Update(ctx context.Context, cluster *Cluster) error
	
	// Delete a cluster
	Delete(ctx context.Context, name string) error
}

// Cluster represents an ECS cluster in storage
type Cluster struct {
	// Unique identifier
	ID string `json:"id"`
	
	// Cluster ARN
	ARN string `json:"arn"`
	
	// Cluster name
	Name string `json:"name"`
	
	// Cluster status (ACTIVE, INACTIVE, etc.)
	Status string `json:"status"`
	
	// Region
	Region string `json:"region"`
	
	// Account ID
	AccountID string `json:"accountId"`
	
	// Configuration as JSON
	Configuration string `json:"configuration,omitempty"`
	
	// Settings as JSON
	Settings string `json:"settings,omitempty"`
	
	// Tags as JSON
	Tags string `json:"tags,omitempty"`
	
	// Kind cluster name (kecs-<cluster-name>)
	KindClusterName string `json:"kindClusterName,omitempty"`
	
	// Statistics
	RegisteredContainerInstancesCount int `json:"registeredContainerInstancesCount"`
	RunningTasksCount                 int `json:"runningTasksCount"`
	PendingTasksCount                 int `json:"pendingTasksCount"`
	ActiveServicesCount               int `json:"activeServicesCount"`
	
	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TaskDefinitionStore defines task definition-specific storage operations
type TaskDefinitionStore interface {
	// Register a new task definition (creates a new revision)
	Register(ctx context.Context, taskDef *TaskDefinition) (*TaskDefinition, error)
	
	// Get a specific task definition revision
	Get(ctx context.Context, family string, revision int) (*TaskDefinition, error)
	
	// Get the latest revision of a task definition family
	GetLatest(ctx context.Context, family string) (*TaskDefinition, error)
	
	// List task definition families with pagination
	ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*TaskDefinitionFamily, string, error)
	
	// List revisions of a specific task definition family
	ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*TaskDefinitionRevision, string, error)
	
	// Deregister a task definition revision
	Deregister(ctx context.Context, family string, revision int) error
	
	// Get task definition by ARN
	GetByARN(ctx context.Context, arn string) (*TaskDefinition, error)
}

// TaskDefinition represents a task definition with its full configuration
type TaskDefinition struct {
	// Unique identifier
	ID string `json:"id"`
	
	// Task definition ARN
	ARN string `json:"arn"`
	
	// Task definition family
	Family string `json:"family"`
	
	// Task definition revision
	Revision int `json:"revision"`
	
	// Task role ARN
	TaskRoleARN string `json:"taskRoleArn,omitempty"`
	
	// Execution role ARN
	ExecutionRoleARN string `json:"executionRoleArn,omitempty"`
	
	// Network mode (bridge, host, awsvpc, none)
	NetworkMode string `json:"networkMode"`
	
	// Container definitions as JSON
	ContainerDefinitions string `json:"containerDefinitions"`
	
	// Volumes as JSON
	Volumes string `json:"volumes,omitempty"`
	
	// Placement constraints as JSON
	PlacementConstraints string `json:"placementConstraints,omitempty"`
	
	// Required compatibility (EC2, FARGATE, etc.)
	RequiresCompatibilities string `json:"requiresCompatibilities,omitempty"`
	
	// CPU value (in CPU units or vCPU)
	CPU string `json:"cpu,omitempty"`
	
	// Memory value (in MiB)
	Memory string `json:"memory,omitempty"`
	
	// Tags as JSON
	Tags string `json:"tags,omitempty"`
	
	// PID mode
	PidMode string `json:"pidMode,omitempty"`
	
	// IPC mode
	IpcMode string `json:"ipcMode,omitempty"`
	
	// Proxy configuration as JSON
	ProxyConfiguration string `json:"proxyConfiguration,omitempty"`
	
	// Inference accelerators as JSON
	InferenceAccelerators string `json:"inferenceAccelerators,omitempty"`
	
	// Runtime platform as JSON
	RuntimePlatform string `json:"runtimePlatform,omitempty"`
	
	// Status (ACTIVE, INACTIVE)
	Status string `json:"status"`
	
	// Region
	Region string `json:"region"`
	
	// Account ID
	AccountID string `json:"accountId"`
	
	// Timestamps
	RegisteredAt    time.Time  `json:"registeredAt"`
	DeregisteredAt  *time.Time `json:"deregisteredAt,omitempty"`
}

// TaskDefinitionFamily represents a task definition family summary
type TaskDefinitionFamily struct {
	Family           string `json:"family"`
	LatestRevision   int    `json:"latestRevision"`
	ActiveRevisions  int    `json:"activeRevisions"`
}

// TaskDefinitionRevision represents a task definition revision summary
type TaskDefinitionRevision struct {
	ARN          string    `json:"arn"`
	Family       string    `json:"family"`
	Revision     int       `json:"revision"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// ServiceStore defines service-specific storage operations
type ServiceStore interface {
	// Create a new service
	Create(ctx context.Context, service *Service) error
	
	// Get a service by cluster and service name
	Get(ctx context.Context, cluster, serviceName string) (*Service, error)
	
	// List services with filtering
	List(ctx context.Context, cluster string, serviceName string, launchType string, limit int, nextToken string) ([]*Service, string, error)
	
	// Update a service
	Update(ctx context.Context, service *Service) error
	
	// Delete a service
	Delete(ctx context.Context, cluster, serviceName string) error
	
	// Get service by ARN
	GetByARN(ctx context.Context, arn string) (*Service, error)
}

// Service represents an ECS service in storage
type Service struct {
	// Unique identifier
	ID string `json:"id"`
	
	// Service ARN
	ARN string `json:"arn"`
	
	// Service name
	ServiceName string `json:"serviceName"`
	
	// Cluster ARN
	ClusterARN string `json:"clusterArn"`
	
	// Task definition ARN
	TaskDefinitionARN string `json:"taskDefinitionArn"`
	
	// Desired count
	DesiredCount int `json:"desiredCount"`
	
	// Running count
	RunningCount int `json:"runningCount"`
	
	// Pending count
	PendingCount int `json:"pendingCount"`
	
	// Launch type (EC2, FARGATE, EXTERNAL)
	LaunchType string `json:"launchType"`
	
	// Platform version
	PlatformVersion string `json:"platformVersion,omitempty"`
	
	// Status
	Status string `json:"status"`
	
	// Role ARN
	RoleARN string `json:"roleArn,omitempty"`
	
	// Load balancers as JSON
	LoadBalancers string `json:"loadBalancers,omitempty"`
	
	// Service registries as JSON
	ServiceRegistries string `json:"serviceRegistries,omitempty"`
	
	// Network configuration as JSON
	NetworkConfiguration string `json:"networkConfiguration,omitempty"`
	
	// Deployment configuration as JSON
	DeploymentConfiguration string `json:"deploymentConfiguration,omitempty"`
	
	// Placement constraints as JSON
	PlacementConstraints string `json:"placementConstraints,omitempty"`
	
	// Placement strategy as JSON
	PlacementStrategy string `json:"placementStrategy,omitempty"`
	
	// Capacity provider strategy as JSON
	CapacityProviderStrategy string `json:"capacityProviderStrategy,omitempty"`
	
	// Tags as JSON
	Tags string `json:"tags,omitempty"`
	
	// Scheduling strategy (REPLICA, DAEMON)
	SchedulingStrategy string `json:"schedulingStrategy"`
	
	// Service connect configuration as JSON
	ServiceConnectConfiguration string `json:"serviceConnectConfiguration,omitempty"`
	
	// Enable ECS managed tags
	EnableECSManagedTags bool `json:"enableECSManagedTags"`
	
	// Propagate tags (TASK_DEFINITION, SERVICE, NONE)
	PropagateTags string `json:"propagateTags,omitempty"`
	
	// Enable execute command
	EnableExecuteCommand bool `json:"enableExecuteCommand"`
	
	// Health check grace period
	HealthCheckGracePeriodSeconds int `json:"healthCheckGracePeriodSeconds,omitempty"`
	
	// Region
	Region string `json:"region"`
	
	// Account ID
	AccountID string `json:"accountId"`
	
	// Kubernetes Deployment information (for tracking)
	DeploymentName string `json:"deploymentName,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	
	// Timestamps
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// TaskStore defines task-specific storage operations
type TaskStore interface {
	// Create a new task
	Create(ctx context.Context, task *Task) error
	
	// Get a task by cluster and task ID/ARN
	Get(ctx context.Context, cluster, taskID string) (*Task, error)
	
	// List tasks with filtering
	List(ctx context.Context, cluster string, filters TaskFilters) ([]*Task, error)
	
	// Update a task (status, etc.)
	Update(ctx context.Context, task *Task) error
	
	// Delete a task
	Delete(ctx context.Context, cluster, taskID string) error
	
	// Get tasks by ARNs
	GetByARNs(ctx context.Context, arns []string) ([]*Task, error)
}

// TaskFilters defines filters for listing tasks
type TaskFilters struct {
	// Filter by service name
	ServiceName string
	
	// Filter by task definition family
	Family string
	
	// Filter by container instance
	ContainerInstance string
	
	// Filter by launch type
	LaunchType string
	
	// Filter by status
	DesiredStatus string
	
	// Filter by started by
	StartedBy string
	
	// Maximum results
	MaxResults int
	
	// Next token for pagination
	NextToken string
}

// Task represents an ECS task in storage
type Task struct {
	// Unique identifier
	ID string `json:"id"`
	
	// Task ARN
	ARN string `json:"arn"`
	
	// Cluster ARN
	ClusterARN string `json:"clusterArn"`
	
	// Task definition ARN
	TaskDefinitionARN string `json:"taskDefinitionArn"`
	
	// Container instance ARN (for EC2 launch type)
	ContainerInstanceARN string `json:"containerInstanceArn,omitempty"`
	
	// Overrides as JSON
	Overrides string `json:"overrides,omitempty"`
	
	// Last status
	LastStatus string `json:"lastStatus"`
	
	// Desired status
	DesiredStatus string `json:"desiredStatus"`
	
	// CPU
	CPU string `json:"cpu,omitempty"`
	
	// Memory
	Memory string `json:"memory,omitempty"`
	
	// Containers as JSON (status information)
	Containers string `json:"containers"`
	
	// Started by (service name, user, etc.)
	StartedBy string `json:"startedBy,omitempty"`
	
	// Version
	Version int64 `json:"version"`
	
	// Stop code
	StopCode string `json:"stopCode,omitempty"`
	
	// Stop reason
	StoppedReason string `json:"stoppedReason,omitempty"`
	
	// Stopping at
	StoppingAt *time.Time `json:"stoppingAt,omitempty"`
	
	// Stopped at
	StoppedAt *time.Time `json:"stoppedAt,omitempty"`
	
	// Connectivity
	Connectivity string `json:"connectivity,omitempty"`
	
	// Connectivity at
	ConnectivityAt *time.Time `json:"connectivityAt,omitempty"`
	
	// Pull started at
	PullStartedAt *time.Time `json:"pullStartedAt,omitempty"`
	
	// Pull stopped at
	PullStoppedAt *time.Time `json:"pullStoppedAt,omitempty"`
	
	// Execution stopped at
	ExecutionStoppedAt *time.Time `json:"executionStoppedAt,omitempty"`
	
	// Created at
	CreatedAt time.Time `json:"createdAt"`
	
	// Started at
	StartedAt *time.Time `json:"startedAt,omitempty"`
	
	// Launch type
	LaunchType string `json:"launchType"`
	
	// Platform version
	PlatformVersion string `json:"platformVersion,omitempty"`
	
	// Platform family
	PlatformFamily string `json:"platformFamily,omitempty"`
	
	// Group
	Group string `json:"group,omitempty"`
	
	// Attachments as JSON
	Attachments string `json:"attachments,omitempty"`
	
	// Health status
	HealthStatus string `json:"healthStatus,omitempty"`
	
	// Tags as JSON
	Tags string `json:"tags,omitempty"`
	
	// Attributes as JSON
	Attributes string `json:"attributes,omitempty"`
	
	// Enable execute command
	EnableExecuteCommand bool `json:"enableExecuteCommand"`
	
	// Capacity provider name
	CapacityProviderName string `json:"capacityProviderName,omitempty"`
	
	// Ephemeral storage as JSON
	EphemeralStorage string `json:"ephemeralStorage,omitempty"`
	
	// Region
	Region string `json:"region"`
	
	// Account ID
	AccountID string `json:"accountId"`
	
	// Kubernetes Pod name
	PodName string `json:"podName,omitempty"`
	
	// Kubernetes namespace
	Namespace string `json:"namespace,omitempty"`
}