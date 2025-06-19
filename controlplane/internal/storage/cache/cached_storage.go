package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
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

func (s *cachedClusterStore) Create(ctx context.Context, cluster *generated.Cluster) error {
	if err := s.backend.Create(ctx, cluster); err != nil {
		return err
	}
	
	// Cache the created cluster
	s.cache.Set(ctx, clusterKey(cluster.ClusterName), cluster)
	s.cache.Set(ctx, clusterKeyByArn(cluster.ClusterArn), cluster)
	
	// Invalidate list cache
	s.cache.Delete(ctx, "clusters:list")
	
	return nil
}

func (s *cachedClusterStore) Get(ctx context.Context, name string) (*generated.Cluster, error) {
	key := clusterKey(name)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.Cluster), nil
	}
	
	// Fetch from backend
	cluster, err := s.backend.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, cluster)
	s.cache.Set(ctx, clusterKeyByArn(cluster.ClusterArn), cluster)
	
	return cluster, nil
}

func (s *cachedClusterStore) GetByArn(ctx context.Context, arn string) (*generated.Cluster, error) {
	key := clusterKeyByArn(arn)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.Cluster), nil
	}
	
	// Fetch from backend
	cluster, err := s.backend.GetByArn(ctx, arn)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, cluster)
	s.cache.Set(ctx, clusterKey(cluster.ClusterName), cluster)
	
	return cluster, nil
}

func (s *cachedClusterStore) List(ctx context.Context, region, accountID string) ([]*generated.Cluster, error) {
	key := fmt.Sprintf("clusters:list:%s:%s", region, accountID)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.([]*generated.Cluster), nil
	}
	
	// Fetch from backend
	clusters, err := s.backend.List(ctx, region, accountID)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with shorter TTL
	s.cache.SetWithTTL(ctx, key, clusters, 1*time.Minute)
	
	// Also cache individual clusters
	for _, cluster := range clusters {
		s.cache.Set(ctx, clusterKey(cluster.ClusterName), cluster)
		s.cache.Set(ctx, clusterKeyByArn(cluster.ClusterArn), cluster)
	}
	
	return clusters, nil
}

func (s *cachedClusterStore) Update(ctx context.Context, cluster *generated.Cluster) error {
	if err := s.backend.Update(ctx, cluster); err != nil {
		return err
	}
	
	// Update cache
	s.cache.Set(ctx, clusterKey(cluster.ClusterName), cluster)
	s.cache.Set(ctx, clusterKeyByArn(cluster.ClusterArn), cluster)
	
	// Invalidate list cache
	s.cache.Delete(ctx, "clusters:list")
	
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
		s.cache.Delete(ctx, clusterKeyByArn(cluster.ClusterArn))
	}
	
	// Invalidate list cache
	s.cache.Delete(ctx, "clusters:list")
	
	return nil
}

func (s *cachedClusterStore) DeleteByArn(ctx context.Context, arn string) error {
	// Get cluster first to get name
	cluster, _ := s.GetByArn(ctx, arn)
	
	if err := s.backend.DeleteByArn(ctx, arn); err != nil {
		return err
	}
	
	// Remove from cache
	s.cache.Delete(ctx, clusterKeyByArn(arn))
	if cluster != nil {
		s.cache.Delete(ctx, clusterKey(cluster.ClusterName))
	}
	
	// Invalidate list cache
	s.cache.Delete(ctx, "clusters:list")
	
	return nil
}

// cachedServiceStore implements storage.ServiceStore with caching
type cachedServiceStore struct {
	backend storage.ServiceStore
	cache   *MemoryCache
}

func (s *cachedServiceStore) Create(ctx context.Context, service *generated.Service) error {
	if err := s.backend.Create(ctx, service); err != nil {
		return err
	}
	
	// Cache the created service
	s.cache.Set(ctx, serviceKey(service.ServiceArn), service)
	
	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("services:list:%s", service.ClusterArn))
	
	return nil
}

func (s *cachedServiceStore) Get(ctx context.Context, clusterArn, serviceName string) (*generated.Service, error) {
	// Generate a key based on cluster and service name
	key := fmt.Sprintf("service:%s:%s", clusterArn, serviceName)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.Service), nil
	}
	
	// Fetch from backend
	service, err := s.backend.Get(ctx, clusterArn, serviceName)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, service)
	s.cache.Set(ctx, serviceKey(service.ServiceArn), service)
	
	return service, nil
}

func (s *cachedServiceStore) GetByArn(ctx context.Context, arn string) (*generated.Service, error) {
	key := serviceKey(arn)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.Service), nil
	}
	
	// Fetch from backend
	service, err := s.backend.GetByArn(ctx, arn)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, service)
	
	return service, nil
}

func (s *cachedServiceStore) List(ctx context.Context, clusterArn string) ([]*generated.Service, error) {
	key := fmt.Sprintf("services:list:%s", clusterArn)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.([]*generated.Service), nil
	}
	
	// Fetch from backend
	services, err := s.backend.List(ctx, clusterArn)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with shorter TTL
	s.cache.SetWithTTL(ctx, key, services, 1*time.Minute)
	
	// Also cache individual services
	for _, service := range services {
		s.cache.Set(ctx, serviceKey(service.ServiceArn), service)
	}
	
	return services, nil
}

func (s *cachedServiceStore) Update(ctx context.Context, service *generated.Service) error {
	if err := s.backend.Update(ctx, service); err != nil {
		return err
	}
	
	// Update cache
	s.cache.Set(ctx, serviceKey(service.ServiceArn), service)
	
	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("services:list:%s", service.ClusterArn))
	
	return nil
}

func (s *cachedServiceStore) Delete(ctx context.Context, clusterArn, serviceName string) error {
	// Get service first to get ARN
	service, _ := s.Get(ctx, clusterArn, serviceName)
	
	if err := s.backend.Delete(ctx, clusterArn, serviceName); err != nil {
		return err
	}
	
	// Remove from cache
	if service != nil {
		s.cache.Delete(ctx, serviceKey(service.ServiceArn))
	}
	s.cache.Delete(ctx, fmt.Sprintf("service:%s:%s", clusterArn, serviceName))
	
	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("services:list:%s", clusterArn))
	
	return nil
}

// cachedTaskDefinitionStore implements storage.TaskDefinitionStore with caching
type cachedTaskDefinitionStore struct {
	backend storage.TaskDefinitionStore
	cache   *MemoryCache
}

func (s *cachedTaskDefinitionStore) Create(ctx context.Context, taskDef *generated.TaskDefinition) error {
	if err := s.backend.Create(ctx, taskDef); err != nil {
		return err
	}
	
	// Cache the created task definition
	s.cache.Set(ctx, taskDefKey(taskDef.TaskDefinitionArn), taskDef)
	s.cache.Set(ctx, taskDefFamilyKey(taskDef.Family), taskDef)
	
	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("taskdefs:list:%s", taskDef.Family))
	
	return nil
}

func (s *cachedTaskDefinitionStore) Get(ctx context.Context, family string, revision int32) (*generated.TaskDefinition, error) {
	key := fmt.Sprintf("taskdef:%s:%d", family, revision)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.TaskDefinition), nil
	}
	
	// Fetch from backend
	taskDef, err := s.backend.Get(ctx, family, revision)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, taskDef)
	s.cache.Set(ctx, taskDefKey(taskDef.TaskDefinitionArn), taskDef)
	
	return taskDef, nil
}

func (s *cachedTaskDefinitionStore) GetByArn(ctx context.Context, arn string) (*generated.TaskDefinition, error) {
	key := taskDefKey(arn)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.TaskDefinition), nil
	}
	
	// Fetch from backend
	taskDef, err := s.backend.GetByArn(ctx, arn)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, taskDef)
	
	return taskDef, nil
}

func (s *cachedTaskDefinitionStore) GetLatest(ctx context.Context, family string) (*generated.TaskDefinition, error) {
	key := taskDefFamilyKey(family)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.TaskDefinition), nil
	}
	
	// Fetch from backend
	taskDef, err := s.backend.GetLatest(ctx, family)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.Set(ctx, key, taskDef)
	s.cache.Set(ctx, taskDefKey(taskDef.TaskDefinitionArn), taskDef)
	
	return taskDef, nil
}

func (s *cachedTaskDefinitionStore) ListByFamily(ctx context.Context, family string) ([]*generated.TaskDefinition, error) {
	key := fmt.Sprintf("taskdefs:list:%s", family)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.([]*generated.TaskDefinition), nil
	}
	
	// Fetch from backend
	taskDefs, err := s.backend.ListByFamily(ctx, family)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with shorter TTL
	s.cache.SetWithTTL(ctx, key, taskDefs, 1*time.Minute)
	
	// Also cache individual task definitions
	for _, taskDef := range taskDefs {
		s.cache.Set(ctx, taskDefKey(taskDef.TaskDefinitionArn), taskDef)
	}
	
	return taskDefs, nil
}

func (s *cachedTaskDefinitionStore) ListFamilies(ctx context.Context, region, accountID string) ([]string, error) {
	key := fmt.Sprintf("taskdefs:families:%s:%s", region, accountID)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.([]string), nil
	}
	
	// Fetch from backend
	families, err := s.backend.ListFamilies(ctx, region, accountID)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	s.cache.SetWithTTL(ctx, key, families, 2*time.Minute)
	
	return families, nil
}

func (s *cachedTaskDefinitionStore) Delete(ctx context.Context, family string, revision int32) error {
	// Get task definition first to get ARN
	taskDef, _ := s.Get(ctx, family, revision)
	
	if err := s.backend.Delete(ctx, family, revision); err != nil {
		return err
	}
	
	// Remove from cache
	s.cache.Delete(ctx, fmt.Sprintf("taskdef:%s:%d", family, revision))
	if taskDef != nil {
		s.cache.Delete(ctx, taskDefKey(taskDef.TaskDefinitionArn))
	}
	
	// Invalidate family cache
	s.cache.Delete(ctx, taskDefFamilyKey(family))
	s.cache.Delete(ctx, fmt.Sprintf("taskdefs:list:%s", family))
	
	return nil
}

// cachedTaskStore implements storage.TaskStore with caching
type cachedTaskStore struct {
	backend storage.TaskStore
	cache   *MemoryCache
}

func (s *cachedTaskStore) Create(ctx context.Context, task *generated.Task) error {
	if err := s.backend.Create(ctx, task); err != nil {
		return err
	}
	
	// Cache the created task
	s.cache.Set(ctx, taskKey(task.TaskArn), task)
	
	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", task.ClusterArn))
	
	return nil
}

func (s *cachedTaskStore) Get(ctx context.Context, taskArn string) (*generated.Task, error) {
	key := taskKey(taskArn)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.(*generated.Task), nil
	}
	
	// Fetch from backend
	task, err := s.backend.Get(ctx, taskArn)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with shorter TTL (tasks change frequently)
	s.cache.SetWithTTL(ctx, key, task, 30*time.Second)
	
	return task, nil
}

func (s *cachedTaskStore) List(ctx context.Context, clusterArn, serviceName, status string) ([]*generated.Task, error) {
	key := fmt.Sprintf("tasks:list:%s:%s:%s", clusterArn, serviceName, status)
	
	// Check cache
	if cached, found := s.cache.Get(ctx, key); found {
		return cached.([]*generated.Task), nil
	}
	
	// Fetch from backend
	tasks, err := s.backend.List(ctx, clusterArn, serviceName, status)
	if err != nil {
		return nil, err
	}
	
	// Cache the result with very short TTL (tasks change frequently)
	s.cache.SetWithTTL(ctx, key, tasks, 10*time.Second)
	
	// Also cache individual tasks
	for _, task := range tasks {
		s.cache.SetWithTTL(ctx, taskKey(task.TaskArn), task, 30*time.Second)
	}
	
	return tasks, nil
}

func (s *cachedTaskStore) Update(ctx context.Context, task *generated.Task) error {
	if err := s.backend.Update(ctx, task); err != nil {
		return err
	}
	
	// Update cache
	s.cache.SetWithTTL(ctx, taskKey(task.TaskArn), task, 30*time.Second)
	
	// Invalidate list cache
	s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", task.ClusterArn))
	
	return nil
}

func (s *cachedTaskStore) Delete(ctx context.Context, taskArn string) error {
	// Get task first to get cluster ARN
	task, _ := s.Get(ctx, taskArn)
	
	if err := s.backend.Delete(ctx, taskArn); err != nil {
		return err
	}
	
	// Remove from cache
	s.cache.Delete(ctx, taskKey(taskArn))
	
	// Invalidate list cache
	if task != nil {
		s.cache.Delete(ctx, fmt.Sprintf("tasks:list:%s", task.ClusterArn))
	}
	
	return nil
}

// Helper functions for cache keys
func clusterKey(name string) string {
	return fmt.Sprintf("cluster:name:%s", name)
}

func clusterKeyByArn(arn string) string {
	return fmt.Sprintf("cluster:arn:%s", arn)
}

func serviceKey(arn string) string {
	return fmt.Sprintf("service:arn:%s", arn)
}

func taskDefKey(arn string) string {
	return fmt.Sprintf("taskdef:arn:%s", arn)
}

func taskDefFamilyKey(family string) string {
	return fmt.Sprintf("taskdef:family:latest:%s", family)
}

func taskKey(arn string) string {
	return fmt.Sprintf("task:arn:%s", arn)
}