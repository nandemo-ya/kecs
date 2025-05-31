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