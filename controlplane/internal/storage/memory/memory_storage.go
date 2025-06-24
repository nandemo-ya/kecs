package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MemoryStorage implements storage.Storage interface using in-memory maps
type MemoryStorage struct {
	clusters map[string]*storage.Cluster
	mu       sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage instance
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		clusters: make(map[string]*storage.Cluster),
	}
}

// Initialize initializes the storage (no-op for memory storage)
func (m *MemoryStorage) Initialize(ctx context.Context) error {
	return nil
}

// Close closes the storage connection (no-op for memory storage)
func (m *MemoryStorage) Close() error {
	return nil
}

// ClusterStore returns the cluster store
func (m *MemoryStorage) ClusterStore() storage.ClusterStore {
	return &memoryClusterStore{storage: m}
}

// TaskDefinitionStore returns the task definition store
func (m *MemoryStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return nil // Not implemented for tests
}

// ServiceStore returns the service store
func (m *MemoryStorage) ServiceStore() storage.ServiceStore {
	return nil // Not implemented for tests
}

// TaskStore returns the task store
func (m *MemoryStorage) TaskStore() storage.TaskStore {
	return nil // Not implemented for tests
}

// AccountSettingStore returns the account setting store
func (m *MemoryStorage) AccountSettingStore() storage.AccountSettingStore {
	return nil // Not implemented for tests
}

// TaskSetStore returns the task set store
func (m *MemoryStorage) TaskSetStore() storage.TaskSetStore {
	return nil // Not implemented for tests
}

// ContainerInstanceStore returns the container instance store
func (m *MemoryStorage) ContainerInstanceStore() storage.ContainerInstanceStore {
	return nil // Not implemented for tests
}

// AttributeStore returns the attribute store
func (m *MemoryStorage) AttributeStore() storage.AttributeStore {
	return nil // Not implemented for tests
}

// BeginTx begins a transaction
func (m *MemoryStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return &memoryTransaction{}, nil
}

// memoryClusterStore implements storage.ClusterStore
type memoryClusterStore struct {
	storage *MemoryStorage
}

// Create creates a new cluster
func (s *memoryClusterStore) Create(ctx context.Context, cluster *storage.Cluster) error {
	s.storage.mu.Lock()
	defer s.storage.mu.Unlock()

	if _, exists := s.storage.clusters[cluster.Name]; exists {
		return fmt.Errorf("cluster %s already exists", cluster.Name)
	}

	s.storage.clusters[cluster.Name] = cluster
	return nil
}

// Get retrieves a cluster by name
func (s *memoryClusterStore) Get(ctx context.Context, name string) (*storage.Cluster, error) {
	s.storage.mu.RLock()
	defer s.storage.mu.RUnlock()

	cluster, exists := s.storage.clusters[name]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	return cluster, nil
}

// List lists all clusters
func (s *memoryClusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	s.storage.mu.RLock()
	defer s.storage.mu.RUnlock()

	clusters := make([]*storage.Cluster, 0, len(s.storage.clusters))
	for _, cluster := range s.storage.clusters {
		clusters = append(clusters, cluster)
	}

	return clusters, nil
}

// ListWithPagination lists clusters with pagination
func (s *memoryClusterStore) ListWithPagination(ctx context.Context, limit int, nextToken string) ([]*storage.Cluster, string, error) {
	clusters, err := s.List(ctx)
	return clusters, "", err
}

// Update updates a cluster
func (s *memoryClusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	s.storage.mu.Lock()
	defer s.storage.mu.Unlock()

	if _, exists := s.storage.clusters[cluster.Name]; !exists {
		return fmt.Errorf("cluster %s not found", cluster.Name)
	}

	cluster.UpdatedAt = time.Now()
	s.storage.clusters[cluster.Name] = cluster
	return nil
}

// Delete deletes a cluster
func (s *memoryClusterStore) Delete(ctx context.Context, name string) error {
	s.storage.mu.Lock()
	defer s.storage.mu.Unlock()

	if _, exists := s.storage.clusters[name]; !exists {
		return fmt.Errorf("cluster %s not found", name)
	}

	delete(s.storage.clusters, name)
	return nil
}

// memoryTransaction implements storage.Transaction
type memoryTransaction struct{}

func (t *memoryTransaction) Commit() error {
	return nil
}

func (t *memoryTransaction) Rollback() error {
	return nil
}
