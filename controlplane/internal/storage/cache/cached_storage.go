package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// CachedStorage wraps a storage implementation with caching
type CachedStorage struct {
	backend storage.Storage
	cache   *MemoryCache
}

// NewCachedStorage creates a new cached storage wrapper
func NewCachedStorage(backend storage.Storage, maxItems int, ttl time.Duration) *CachedStorage {
	return &CachedStorage{
		backend: backend,
		cache:   NewMemoryCache(maxItems, ttl),
	}
}

// Initialize initializes the backend storage
func (s *CachedStorage) Initialize(ctx context.Context) error {
	return s.backend.Initialize(ctx)
}

// Close closes the backend storage
func (s *CachedStorage) Close() error {
	return s.backend.Close()
}

// ClusterStore returns a cached cluster store
func (s *CachedStorage) ClusterStore() storage.ClusterStore {
	return &cachedClusterStore{
		backend: s.backend.ClusterStore(),
		cache:   s.cache,
	}
}

// TaskDefinitionStore returns a cached task definition store
func (s *CachedStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return &cachedTaskDefinitionStore{
		backend: s.backend.TaskDefinitionStore(),
		cache:   s.cache,
	}
}

// ServiceStore returns a cached service store
func (s *CachedStorage) ServiceStore() storage.ServiceStore {
	return &cachedServiceStore{
		backend: s.backend.ServiceStore(),
		cache:   s.cache,
	}
}

// TaskStore returns a cached task store
func (s *CachedStorage) TaskStore() storage.TaskStore {
	return &cachedTaskStore{
		backend: s.backend.TaskStore(),
		cache:   s.cache,
	}
}

// AccountSettingStore returns the account setting store (no caching)
func (s *CachedStorage) AccountSettingStore() storage.AccountSettingStore {
	return s.backend.AccountSettingStore()
}

// TaskSetStore returns the task set store (no caching)
func (s *CachedStorage) TaskSetStore() storage.TaskSetStore {
	return s.backend.TaskSetStore()
}

// ContainerInstanceStore returns the container instance store (no caching)
func (s *CachedStorage) ContainerInstanceStore() storage.ContainerInstanceStore {
	return s.backend.ContainerInstanceStore()
}

// AttributeStore returns the attribute store (no caching)
func (s *CachedStorage) AttributeStore() storage.AttributeStore {
	return s.backend.AttributeStore()
}

// ELBv2Store returns the ELBv2 store (no caching)
func (s *CachedStorage) ELBv2Store() storage.ELBv2Store {
	return s.backend.ELBv2Store()
}

// TaskLogStore returns the task log store (no caching for logs)
func (s *CachedStorage) TaskLogStore() storage.TaskLogStore {
	return s.backend.TaskLogStore()
}

// BeginTx starts a new transaction
func (s *CachedStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return s.backend.BeginTx(ctx)
}

// Stats returns cache statistics
func (s *CachedStorage) Stats() CacheStats {
	return s.cache.Stats()
}

// cachedClusterStore implements storage.ClusterStore with caching
type cachedClusterStore struct {
	backend storage.ClusterStore
	cache   *MemoryCache
}

func (s *cachedClusterStore) Create(ctx context.Context, cluster *storage.Cluster) error {
	if err := s.backend.Create(ctx, cluster); err != nil {
		return err
	}

	// Cache the created cluster
	s.cache.Set(ctx, clusterKey(cluster.Name), cluster)
	s.cache.Set(ctx, clusterKeyByArn(cluster.ARN), cluster)

	// Invalidate list caches (both simple list and paginated lists)
	s.cache.Delete(ctx, "clusters:list")
	s.cache.DeleteWithPrefix(ctx, "clusters:list:page:")

	return nil
}

func (s *cachedClusterStore) Get(ctx context.Context, name string) (*storage.Cluster, error) {
	key := clusterKey(name)

	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*storage.Cluster), nil
	}

	// Fetch from backend
	cluster, err := s.backend.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, key, cluster)
	s.cache.Set(ctx, clusterKeyByArn(cluster.ARN), cluster)

	return cluster, nil
}

func (s *cachedClusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	key := "clusters:list"

	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.([]*storage.Cluster), nil
	}

	// Fetch from backend
	clusters, err := s.backend.List(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the result with shorter TTL
	s.cache.SetWithTTL(ctx, key, clusters, 1*time.Minute)

	// Also cache individual clusters
	for _, cluster := range clusters {
		s.cache.Set(ctx, clusterKey(cluster.Name), cluster)
		s.cache.Set(ctx, clusterKeyByArn(cluster.ARN), cluster)
	}

	return clusters, nil
}

func (s *cachedClusterStore) ListWithPagination(ctx context.Context, limit int, nextToken string) ([]*storage.Cluster, string, error) {
	key := fmt.Sprintf("clusters:list:page:%d:%s", limit, nextToken)

	// Check cache for pagination results
	if cached, found := s.cache.Get(ctx, key); found {
		result := cached.(paginatedClustersResult)
		return result.Clusters, result.NextToken, nil
	}

	// Fetch from backend
	clusters, newNextToken, err := s.backend.ListWithPagination(ctx, limit, nextToken)
	if err != nil {
		return nil, "", err
	}

	// Cache the paginated result
	s.cache.SetWithTTL(ctx, key, paginatedClustersResult{
		Clusters:  clusters,
		NextToken: newNextToken,
	}, 1*time.Minute)

	// Also cache individual clusters
	for _, cluster := range clusters {
		s.cache.Set(ctx, clusterKey(cluster.Name), cluster)
		s.cache.Set(ctx, clusterKeyByArn(cluster.ARN), cluster)
	}

	return clusters, newNextToken, nil
}

func (s *cachedClusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	if err := s.backend.Update(ctx, cluster); err != nil {
		return err
	}

	// Update cache
	s.cache.Set(ctx, clusterKey(cluster.Name), cluster)
	s.cache.Set(ctx, clusterKeyByArn(cluster.ARN), cluster)

	// Invalidate list caches (both simple list and paginated lists)
	s.cache.Delete(ctx, "clusters:list")
	s.cache.DeleteWithPrefix(ctx, "clusters:list:page:")

	return nil
}

func (s *cachedClusterStore) Delete(ctx context.Context, name string) error {
	// Get cluster first to get ARN
	cluster, _ := s.Get(ctx, name)

	if err := s.backend.Delete(ctx, name); err != nil {
		return err
	}

	// Remove from cache
	s.cache.Delete(ctx, clusterKey(name))
	if cluster != nil {
		s.cache.Delete(ctx, clusterKeyByArn(cluster.ARN))
	}

	// Invalidate list caches (both simple list and paginated lists)
	s.cache.Delete(ctx, "clusters:list")
	s.cache.DeleteWithPrefix(ctx, "clusters:list:page:")

	return nil
}

// cachedTaskDefinitionStore implements storage.TaskDefinitionStore with caching
type cachedTaskDefinitionStore struct {
	backend storage.TaskDefinitionStore
	cache   *MemoryCache
}

func (s *cachedTaskDefinitionStore) Register(ctx context.Context, taskDef *storage.TaskDefinition) (*storage.TaskDefinition, error) {
	registered, err := s.backend.Register(ctx, taskDef)
	if err != nil {
		return nil, err
	}

	// Cache the task definition
	s.cache.Set(ctx, taskDefKey(registered.Family, registered.Revision), registered)
	s.cache.Set(ctx, taskDefKeyByArn(registered.ARN), registered)

	// Invalidate family cache
	s.cache.Delete(ctx, taskDefFamilyKey(registered.Family))

	return registered, nil
}

func (s *cachedTaskDefinitionStore) Deregister(ctx context.Context, family string, revision int) error {
	if err := s.backend.Deregister(ctx, family, revision); err != nil {
		return err
	}

	// Remove from cache
	s.cache.Delete(ctx, taskDefKey(family, revision))

	// Invalidate family cache
	s.cache.Delete(ctx, taskDefFamilyKey(family))

	return nil
}

func (s *cachedTaskDefinitionStore) Get(ctx context.Context, family string, revision int) (*storage.TaskDefinition, error) {
	key := taskDefKey(family, revision)

	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*storage.TaskDefinition), nil
	}

	// Fetch from backend
	taskDef, err := s.backend.Get(ctx, family, revision)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, key, taskDef)
	s.cache.Set(ctx, taskDefKeyByArn(taskDef.ARN), taskDef)

	return taskDef, nil
}

func (s *cachedTaskDefinitionStore) GetByARN(ctx context.Context, arn string) (*storage.TaskDefinition, error) {
	key := taskDefKeyByArn(arn)

	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*storage.TaskDefinition), nil
	}

	// Fetch from backend
	taskDef, err := s.backend.GetByARN(ctx, arn)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, key, taskDef)
	s.cache.Set(ctx, taskDefKey(taskDef.Family, taskDef.Revision), taskDef)

	return taskDef, nil
}

func (s *cachedTaskDefinitionStore) GetLatest(ctx context.Context, family string) (*storage.TaskDefinition, error) {
	// Don't cache latest queries as they change frequently
	return s.backend.GetLatest(ctx, family)
}

func (s *cachedTaskDefinitionStore) ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionFamily, string, error) {
	// Don't cache list operations as they change frequently
	return s.backend.ListFamilies(ctx, familyPrefix, status, limit, nextToken)
}

func (s *cachedTaskDefinitionStore) ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionRevision, string, error) {
	// Don't cache list operations as they change frequently
	return s.backend.ListRevisions(ctx, family, status, limit, nextToken)
}

// cachedServiceStore implements storage.ServiceStore with caching
type cachedServiceStore struct {
	backend storage.ServiceStore
	cache   *MemoryCache
}

func (s *cachedServiceStore) Create(ctx context.Context, service *storage.Service) error {
	if err := s.backend.Create(ctx, service); err != nil {
		return err
	}

	// Cache the service
	s.cache.Set(ctx, serviceKey(service.ClusterARN, service.ServiceName), service)

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("services:list:%s", service.ClusterARN))

	return nil
}

func (s *cachedServiceStore) Get(ctx context.Context, cluster, serviceName string) (*storage.Service, error) {
	key := serviceKey(cluster, serviceName)

	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*storage.Service), nil
	}

	// Fetch from backend
	service, err := s.backend.Get(ctx, cluster, serviceName)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, key, service)

	return service, nil
}

func (s *cachedServiceStore) List(ctx context.Context, cluster string, serviceName string, launchType string, limit int, nextToken string) ([]*storage.Service, string, error) {
	// Don't cache list operations as they change frequently and have complex filtering
	return s.backend.List(ctx, cluster, serviceName, launchType, limit, nextToken)
}

func (s *cachedServiceStore) Update(ctx context.Context, service *storage.Service) error {
	if err := s.backend.Update(ctx, service); err != nil {
		return err
	}

	// Update cache
	s.cache.Set(ctx, serviceKey(service.ClusterARN, service.ServiceName), service)

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("services:list:%s", service.ClusterARN))

	return nil
}

func (s *cachedServiceStore) Delete(ctx context.Context, cluster, serviceName string) error {
	if err := s.backend.Delete(ctx, cluster, serviceName); err != nil {
		return err
	}

	// Remove from cache
	s.cache.Delete(ctx, serviceKey(cluster, serviceName))

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("services:list:%s", cluster))

	return nil
}

func (s *cachedServiceStore) GetByARN(ctx context.Context, arn string) (*storage.Service, error) {
	// For now, just fetch from backend
	// TODO: extract cluster and service name from ARN to use cache
	return s.backend.GetByARN(ctx, arn)
}

// cachedTaskStore implements storage.TaskStore with caching
type cachedTaskStore struct {
	backend storage.TaskStore
	cache   *MemoryCache
}

func (s *cachedTaskStore) Create(ctx context.Context, task *storage.Task) error {
	if err := s.backend.Create(ctx, task); err != nil {
		return err
	}

	// Cache the task
	s.cache.Set(ctx, taskKey(task.ClusterARN, task.ID), task)

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", task.ClusterARN))

	return nil
}

func (s *cachedTaskStore) Get(ctx context.Context, clusterArn, taskID string) (*storage.Task, error) {
	key := taskKey(clusterArn, taskID)

	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*storage.Task), nil
	}

	// Fetch from backend
	task, err := s.backend.Get(ctx, clusterArn, taskID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	s.cache.Set(ctx, key, task)

	return task, nil
}

func (s *cachedTaskStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.Task, error) {
	// For batch operations, fetch from backend
	return s.backend.GetByARNs(ctx, arns)
}

func (s *cachedTaskStore) List(ctx context.Context, cluster string, filters storage.TaskFilters) ([]*storage.Task, error) {
	// Don't cache list operations as they change frequently and have complex filtering
	return s.backend.List(ctx, cluster, filters)
}

func (s *cachedTaskStore) Update(ctx context.Context, task *storage.Task) error {
	if err := s.backend.Update(ctx, task); err != nil {
		return err
	}

	// Update cache
	s.cache.Set(ctx, taskKey(task.ClusterARN, task.ID), task)

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", task.ClusterARN))

	return nil
}

func (s *cachedTaskStore) Delete(ctx context.Context, clusterArn, taskID string) error {
	if err := s.backend.Delete(ctx, clusterArn, taskID); err != nil {
		return err
	}

	// Remove from cache
	s.cache.Delete(ctx, taskKey(clusterArn, taskID))

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", clusterArn))

	return nil
}

func (s *cachedTaskStore) CreateOrUpdate(ctx context.Context, task *storage.Task) error {
	if err := s.backend.CreateOrUpdate(ctx, task); err != nil {
		return err
	}

	// Update cache
	s.cache.Set(ctx, taskKey(task.ClusterARN, task.ID), task)

	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", task.ClusterARN))

	return nil
}

// Helper functions for cache keys
func clusterKey(name string) string {
	return fmt.Sprintf("cluster:name:%s", name)
}

func clusterKeyByArn(arn string) string {
	return fmt.Sprintf("cluster:arn:%s", arn)
}

func taskDefKey(family string, revision int) string {
	return fmt.Sprintf("taskdef:%s:%d", family, revision)
}

func taskDefKeyByArn(arn string) string {
	return fmt.Sprintf("taskdef:arn:%s", arn)
}

func taskDefFamilyKey(family string) string {
	return fmt.Sprintf("taskdef:family:%s", family)
}

func serviceKey(cluster, serviceName string) string {
	return fmt.Sprintf("service:%s:%s", cluster, serviceName)
}

func taskKey(cluster, taskID string) string {
	return fmt.Sprintf("task:%s:%s", cluster, taskID)
}

// paginatedClustersResult holds paginated cluster results
type paginatedClustersResult struct {
	Clusters  []*storage.Cluster
	NextToken string
}

// DeleteOlderThan deletes tasks older than the specified time with the given status
func (s *cachedTaskStore) DeleteOlderThan(ctx context.Context, clusterARN string, before time.Time, status string) (int, error) {
	// Clear cache for the cluster
	s.cache.Delete(ctx, clusterARN)
	// Pass through to underlying store
	return s.backend.DeleteOlderThan(ctx, clusterARN, before, status)
}

// DeleteMarkedForDeletion deletes services marked for deletion before the specified time
func (s *cachedServiceStore) DeleteMarkedForDeletion(ctx context.Context, clusterARN string, before time.Time) (int, error) {
	// Clear cache for the cluster
	s.cache.Delete(ctx, clusterARN)
	// Pass through to underlying store
	return s.backend.DeleteMarkedForDeletion(ctx, clusterARN, before)
}
