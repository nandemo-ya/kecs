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
	
	// Timestamps
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}