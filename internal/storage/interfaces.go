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