package mocks

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockStorage implements storage.Storage interface for testing
type MockStorage struct {
	clusterStore           storage.ClusterStore
	taskDefinitionStore    storage.TaskDefinitionStore
	serviceStore           storage.ServiceStore
	taskStore              storage.TaskStore
	accountSettingStore    storage.AccountSettingStore
	taskSetStore           storage.TaskSetStore
	containerInstanceStore storage.ContainerInstanceStore
	attributeStore         storage.AttributeStore
	elbv2Store             storage.ELBv2Store
	taskLogStore           storage.TaskLogStore
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		taskLogStore: NewMockTaskLogStore(),
	}
}

func (m *MockStorage) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) ClusterStore() storage.ClusterStore {
	return m.clusterStore
}

func (m *MockStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return m.taskDefinitionStore
}

func (m *MockStorage) ServiceStore() storage.ServiceStore {
	return m.serviceStore
}

func (m *MockStorage) TaskStore() storage.TaskStore {
	return m.taskStore
}

func (m *MockStorage) AccountSettingStore() storage.AccountSettingStore {
	return m.accountSettingStore
}

func (m *MockStorage) TaskSetStore() storage.TaskSetStore {
	return m.taskSetStore
}

func (m *MockStorage) ContainerInstanceStore() storage.ContainerInstanceStore {
	return m.containerInstanceStore
}

func (m *MockStorage) AttributeStore() storage.AttributeStore {
	return m.attributeStore
}

func (m *MockStorage) ELBv2Store() storage.ELBv2Store {
	return m.elbv2Store
}

func (m *MockStorage) TaskLogStore() storage.TaskLogStore {
	return m.taskLogStore
}

func (m *MockStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return nil, nil
}

// SetClusterStore sets the cluster store
func (m *MockStorage) SetClusterStore(store storage.ClusterStore) {
	m.clusterStore = store
}

// SetTaskDefinitionStore sets the task definition store
func (m *MockStorage) SetTaskDefinitionStore(store storage.TaskDefinitionStore) {
	m.taskDefinitionStore = store
}

// SetServiceStore sets the service store
func (m *MockStorage) SetServiceStore(store storage.ServiceStore) {
	m.serviceStore = store
}

// SetTaskStore sets the task store
func (m *MockStorage) SetTaskStore(store storage.TaskStore) {
	m.taskStore = store
}

// SetAccountSettingStore sets the account setting store
func (m *MockStorage) SetAccountSettingStore(store storage.AccountSettingStore) {
	m.accountSettingStore = store
}

// SetTaskSetStore sets the task set store
func (m *MockStorage) SetTaskSetStore(store storage.TaskSetStore) {
	m.taskSetStore = store
}

// SetContainerInstanceStore sets the container instance store
func (m *MockStorage) SetContainerInstanceStore(store storage.ContainerInstanceStore) {
	m.containerInstanceStore = store
}

// SetAttributeStore sets the attribute store
func (m *MockStorage) SetAttributeStore(store storage.AttributeStore) {
	m.attributeStore = store
}

// SetELBv2Store sets the ELBv2 store
func (m *MockStorage) SetELBv2Store(store storage.ELBv2Store) {
	m.elbv2Store = store
}

// SetTaskLogStore sets the task log store
func (m *MockStorage) SetTaskLogStore(store storage.TaskLogStore) {
	m.taskLogStore = store
}

// MockClusterStore implements storage.ClusterStore for testing
type MockClusterStore struct {
	clusters map[string]*storage.Cluster
}

func NewMockClusterStore() *MockClusterStore {
	return &MockClusterStore{
		clusters: make(map[string]*storage.Cluster),
	}
}

func (m *MockClusterStore) Create(ctx context.Context, cluster *storage.Cluster) error {
	if m.clusters == nil {
		m.clusters = make(map[string]*storage.Cluster)
	}
	if _, exists := m.clusters[cluster.Name]; exists {
		return errors.New("cluster already exists")
	}
	m.clusters[cluster.Name] = cluster
	return nil
}

func (m *MockClusterStore) Get(ctx context.Context, name string) (*storage.Cluster, error) {
	cluster, exists := m.clusters[name]
	if !exists {
		return nil, errors.New("cluster not found")
	}
	return cluster, nil
}

func (m *MockClusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	var result []*storage.Cluster
	for _, cluster := range m.clusters {
		result = append(result, cluster)
	}
	return result, nil
}

func (m *MockClusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	if _, exists := m.clusters[cluster.Name]; !exists {
		return errors.New("cluster not found")
	}
	m.clusters[cluster.Name] = cluster
	return nil
}

func (m *MockClusterStore) Delete(ctx context.Context, name string) error {
	if _, exists := m.clusters[name]; !exists {
		return errors.New("cluster not found")
	}
	delete(m.clusters, name)
	return nil
}

func (m *MockClusterStore) ListWithPagination(ctx context.Context, limit int, nextToken string) ([]*storage.Cluster, string, error) {
	// Convert map to slice
	var allClusters []*storage.Cluster
	for _, cluster := range m.clusters {
		allClusters = append(allClusters, cluster)
	}

	// Sort by ID for consistent ordering (matches DuckDB implementation)
	sort.Slice(allClusters, func(i, j int) bool {
		return allClusters[i].ID < allClusters[j].ID
	})

	// Find starting position based on nextToken
	start := 0
	if nextToken != "" {
		// Validate token exists
		tokenExists := false
		for _, cluster := range allClusters {
			if cluster.ID == nextToken {
				tokenExists = true
				break
			}
		}

		// If token doesn't exist, return error (like DuckDB implementation)
		if !tokenExists {
			return nil, "", fmt.Errorf("invalid pagination token")
		}

		for i, cluster := range allClusters {
			if cluster.ID > nextToken {
				start = i
				break
			}
		}
	}

	// Get the requested page
	end := start + limit
	if end > len(allClusters) {
		end = len(allClusters)
	}

	result := allClusters[start:end]

	// Determine next token
	var newNextToken string
	if limit > 0 && end < len(allClusters) {
		// Use the last item's ID as the next token
		if len(result) > 0 {
			newNextToken = result[len(result)-1].ID
		}
	}

	return result, newNextToken, nil
}

// MockTaskDefinitionStore implements storage.TaskDefinitionStore for testing
type MockTaskDefinitionStore struct {
	taskDefs         map[string]*storage.TaskDefinition
	taskDefsByFamily map[string][]*storage.TaskDefinition
}

func NewMockTaskDefinitionStore() *MockTaskDefinitionStore {
	return &MockTaskDefinitionStore{
		taskDefs:         make(map[string]*storage.TaskDefinition),
		taskDefsByFamily: make(map[string][]*storage.TaskDefinition),
	}
}

func (m *MockTaskDefinitionStore) Register(ctx context.Context, taskDef *storage.TaskDefinition) (*storage.TaskDefinition, error) {
	if m.taskDefs == nil {
		m.taskDefs = make(map[string]*storage.TaskDefinition)
		m.taskDefsByFamily = make(map[string][]*storage.TaskDefinition)
	}

	// Assign revision number
	revisions := m.taskDefsByFamily[taskDef.Family]
	taskDef.Revision = len(revisions) + 1

	// Set status to ACTIVE if not set
	if taskDef.Status == "" {
		taskDef.Status = "ACTIVE"
	}

	// Set ARN if not set
	if taskDef.ARN == "" {
		taskDef.ARN = fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/%s:%d", taskDef.Family, taskDef.Revision)
	}

	key := fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision)
	m.taskDefs[key] = taskDef
	m.taskDefsByFamily[taskDef.Family] = append(revisions, taskDef)

	return taskDef, nil
}

func (m *MockTaskDefinitionStore) Get(ctx context.Context, family string, revision int) (*storage.TaskDefinition, error) {
	key := fmt.Sprintf("%s:%d", family, revision)
	taskDef, exists := m.taskDefs[key]
	if !exists {
		return nil, errors.New("task definition not found")
	}
	return taskDef, nil
}

func (m *MockTaskDefinitionStore) GetLatest(ctx context.Context, family string) (*storage.TaskDefinition, error) {
	revisions := m.taskDefsByFamily[family]
	if len(revisions) == 0 {
		return nil, errors.New("task definition family not found")
	}
	return revisions[len(revisions)-1], nil
}

func (m *MockTaskDefinitionStore) ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionFamily, string, error) {
	var families []*storage.TaskDefinitionFamily
	for family, revisions := range m.taskDefsByFamily {
		// Filter by family prefix
		if familyPrefix == "" || strings.HasPrefix(family, familyPrefix) {
			families = append(families, &storage.TaskDefinitionFamily{
				Family:         family,
				LatestRevision: len(revisions),
			})
		}
	}

	// Apply limit if specified
	var newNextToken string
	if limit > 0 && len(families) > limit {
		families = families[:limit]
		newNextToken = "next-token"
	}

	return families, newNextToken, nil
}

func (m *MockTaskDefinitionStore) ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionRevision, string, error) {
	revisions := m.taskDefsByFamily[family]
	var result []*storage.TaskDefinitionRevision
	for _, td := range revisions {
		result = append(result, &storage.TaskDefinitionRevision{
			ARN:      td.ARN,
			Family:   td.Family,
			Revision: td.Revision,
			Status:   td.Status,
		})
	}
	return result, "", nil
}

func (m *MockTaskDefinitionStore) Deregister(ctx context.Context, family string, revision int) error {
	key := fmt.Sprintf("%s:%d", family, revision)
	taskDef, exists := m.taskDefs[key]
	if !exists {
		return errors.New("task definition not found")
	}
	taskDef.Status = "INACTIVE"
	return nil
}

func (m *MockTaskDefinitionStore) GetByARN(ctx context.Context, arn string) (*storage.TaskDefinition, error) {
	for _, td := range m.taskDefs {
		if td.ARN == arn {
			return td, nil
		}
	}
	return nil, errors.New("task definition not found")
}

// MockServiceStore implements storage.ServiceStore for testing
type MockServiceStore struct {
	services map[string]*storage.Service
}

func NewMockServiceStore() *MockServiceStore {
	return &MockServiceStore{
		services: make(map[string]*storage.Service),
	}
}

func (m *MockServiceStore) Create(ctx context.Context, service *storage.Service) error {
	if m.services == nil {
		m.services = make(map[string]*storage.Service)
	}
	key := fmt.Sprintf("%s:%s", service.ClusterARN, service.ServiceName)
	if _, exists := m.services[key]; exists {
		return errors.New("service already exists")
	}
	m.services[key] = service
	return nil
}

func (m *MockServiceStore) Get(ctx context.Context, cluster, serviceName string) (*storage.Service, error) {
	key := fmt.Sprintf("%s:%s", cluster, serviceName)
	service, exists := m.services[key]
	if !exists {
		return nil, errors.New("service not found")
	}
	return service, nil
}

func (m *MockServiceStore) List(ctx context.Context, cluster string, serviceName string, launchType string, limit int, nextToken string) ([]*storage.Service, string, error) {
	var results []*storage.Service
	for _, svc := range m.services {
		// Apply filters
		if cluster != "" && svc.ClusterARN != cluster {
			continue
		}
		if serviceName != "" && svc.ServiceName != serviceName {
			continue
		}
		if launchType != "" && svc.LaunchType != launchType {
			continue
		}
		results = append(results, svc)
	}
	return results, "", nil
}

func (m *MockServiceStore) Update(ctx context.Context, service *storage.Service) error {
	key := fmt.Sprintf("%s:%s", service.ClusterARN, service.ServiceName)
	if _, exists := m.services[key]; !exists {
		return errors.New("service not found")
	}
	service.UpdatedAt = time.Now()
	m.services[key] = service
	return nil
}

func (m *MockServiceStore) Delete(ctx context.Context, cluster, serviceName string) error {
	key := fmt.Sprintf("%s:%s", cluster, serviceName)
	if _, exists := m.services[key]; !exists {
		return errors.New("service not found")
	}
	delete(m.services, key)
	return nil
}

func (m *MockServiceStore) GetByARN(ctx context.Context, arn string) (*storage.Service, error) {
	for _, svc := range m.services {
		if svc.ARN == arn {
			return svc, nil
		}
	}
	return nil, errors.New("service not found")
}

func (m *MockServiceStore) DeleteMarkedForDeletion(ctx context.Context, clusterARN string, before time.Time) (int, error) {
	// Mock implementation - just return 0 for tests
	return 0, nil
}

// MockTaskStore implements storage.TaskStore for testing
type MockTaskStore struct {
	tasks map[string]*storage.Task
}

func NewMockTaskStore() *MockTaskStore {
	return &MockTaskStore{
		tasks: make(map[string]*storage.Task),
	}
}

func (m *MockTaskStore) Create(ctx context.Context, task *storage.Task) error {
	if m.tasks == nil {
		m.tasks = make(map[string]*storage.Task)
	}
	key := fmt.Sprintf("%s:%s", task.ClusterARN, task.ID)
	if _, exists := m.tasks[key]; exists {
		return errors.New("task already exists")
	}
	m.tasks[key] = task
	return nil
}

func (m *MockTaskStore) Get(ctx context.Context, cluster, taskID string) (*storage.Task, error) {
	// Handle both short task ID and full ARN (like DuckDB implementation)
	if strings.Contains(taskID, "arn:aws:ecs:") {
		// Full ARN provided - search by ARN
		for _, task := range m.tasks {
			if task.ARN == taskID {
				return task, nil
			}
		}
		return nil, errors.New("task not found")
	} else {
		// Short ID provided - need cluster context
		key := fmt.Sprintf("%s:%s", cluster, taskID)
		task, exists := m.tasks[key]
		if !exists {
			// Also check if taskID matches the ID field
			for k, t := range m.tasks {
				if strings.HasPrefix(k, cluster+":") && t.ID == taskID {
					return t, nil
				}
			}
			return nil, errors.New("task not found")
		}
		return task, nil
	}
}

func (m *MockTaskStore) List(ctx context.Context, cluster string, filters storage.TaskFilters) ([]*storage.Task, error) {
	var results []*storage.Task
	for _, task := range m.tasks {
		if task.ClusterARN != cluster {
			continue
		}
		// Apply filters
		if filters.ServiceName != "" && task.StartedBy != fmt.Sprintf("ecs-svc/%s", filters.ServiceName) {
			continue
		}
		if filters.DesiredStatus != "" && task.DesiredStatus != filters.DesiredStatus {
			continue
		}
		if filters.LaunchType != "" && task.LaunchType != filters.LaunchType {
			continue
		}
		if filters.Family != "" && !hasTaskFamily(task.TaskDefinitionARN, filters.Family) {
			continue
		}
		results = append(results, task)
	}

	// Apply MaxResults limit
	if filters.MaxResults > 0 && len(results) > filters.MaxResults {
		results = results[:filters.MaxResults]
	}

	return results, nil
}

func (m *MockTaskStore) Update(ctx context.Context, task *storage.Task) error {
	key := fmt.Sprintf("%s:%s", task.ClusterARN, task.ID)
	if _, exists := m.tasks[key]; !exists {
		return errors.New("task not found")
	}
	m.tasks[key] = task
	return nil
}

func (m *MockTaskStore) Delete(ctx context.Context, cluster, taskID string) error {
	key := fmt.Sprintf("%s:%s", cluster, taskID)
	if _, exists := m.tasks[key]; !exists {
		return errors.New("task not found")
	}
	delete(m.tasks, key)
	return nil
}

func (m *MockTaskStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.Task, error) {
	var results []*storage.Task
	for _, arn := range arns {
		for _, task := range m.tasks {
			if task.ARN == arn {
				results = append(results, task)
				break
			}
		}
	}
	return results, nil
}

func (m *MockTaskStore) CreateOrUpdate(ctx context.Context, task *storage.Task) error {
	if m.tasks == nil {
		m.tasks = make(map[string]*storage.Task)
	}
	key := fmt.Sprintf("%s:%s", task.ClusterARN, task.ID)
	m.tasks[key] = task
	return nil
}

func (m *MockTaskStore) DeleteOlderThan(ctx context.Context, clusterARN string, before time.Time, status string) (int, error) {
	count := 0
	toDelete := []string{}
	for key, task := range m.tasks {
		if task.ClusterARN == clusterARN && task.LastStatus == status && task.StoppedAt != nil && task.StoppedAt.Before(before) {
			toDelete = append(toDelete, key)
			count++
		}
	}
	for _, key := range toDelete {
		delete(m.tasks, key)
	}
	return count, nil
}

// MockTaskSetStore implements storage.TaskSetStore for testing
type MockTaskSetStore struct {
	taskSets map[string]*storage.TaskSet
}

func NewMockTaskSetStore() *MockTaskSetStore {
	return &MockTaskSetStore{
		taskSets: make(map[string]*storage.TaskSet),
	}
}

func (m *MockTaskSetStore) Create(ctx context.Context, taskSet *storage.TaskSet) error {
	if m.taskSets == nil {
		m.taskSets = make(map[string]*storage.TaskSet)
	}
	key := fmt.Sprintf("%s:%s", taskSet.ServiceARN, taskSet.ID)
	if _, exists := m.taskSets[key]; exists {
		return errors.New("task set already exists")
	}
	m.taskSets[key] = taskSet
	return nil
}

func (m *MockTaskSetStore) Get(ctx context.Context, serviceARN, taskSetID string) (*storage.TaskSet, error) {
	key := fmt.Sprintf("%s:%s", serviceARN, taskSetID)
	taskSet, exists := m.taskSets[key]
	if !exists {
		return nil, errors.New("task set not found")
	}
	return taskSet, nil
}

func (m *MockTaskSetStore) List(ctx context.Context, serviceARN string, taskSetIDs []string) ([]*storage.TaskSet, error) {
	var results []*storage.TaskSet
	// First collect all matching task sets
	for _, ts := range m.taskSets {
		if ts.ServiceARN == serviceARN {
			if len(taskSetIDs) == 0 {
				results = append(results, ts)
			} else {
				for _, id := range taskSetIDs {
					if ts.ID == id {
						results = append(results, ts)
						break
					}
				}
			}
		}
	}

	// Sort by ID to ensure consistent ordering
	if len(results) > 1 {
		for i := 0; i < len(results)-1; i++ {
			for j := i + 1; j < len(results); j++ {
				if results[i].ID > results[j].ID {
					results[i], results[j] = results[j], results[i]
				}
			}
		}
	}

	return results, nil
}

func (m *MockTaskSetStore) Update(ctx context.Context, taskSet *storage.TaskSet) error {
	key := fmt.Sprintf("%s:%s", taskSet.ServiceARN, taskSet.ID)
	if _, exists := m.taskSets[key]; !exists {
		return errors.New("task set not found")
	}
	taskSet.UpdatedAt = time.Now()
	m.taskSets[key] = taskSet
	return nil
}

func (m *MockTaskSetStore) Delete(ctx context.Context, serviceARN, taskSetID string) error {
	key := fmt.Sprintf("%s:%s", serviceARN, taskSetID)
	if _, exists := m.taskSets[key]; !exists {
		return errors.New("task set not found")
	}
	delete(m.taskSets, key)
	return nil
}

func (m *MockTaskSetStore) GetByARN(ctx context.Context, arn string) (*storage.TaskSet, error) {
	for _, ts := range m.taskSets {
		if ts.ARN == arn {
			return ts, nil
		}
	}
	return nil, errors.New("task set not found")
}

func (m *MockTaskSetStore) UpdatePrimary(ctx context.Context, serviceARN, taskSetID string) error {
	key := fmt.Sprintf("%s:%s", serviceARN, taskSetID)
	if _, exists := m.taskSets[key]; !exists {
		return errors.New("task set not found")
	}
	// In a real implementation, this would update the service to point to this task set
	return nil
}

// GetTaskSets returns the internal task sets map for testing
func (m *MockTaskSetStore) DeleteOrphaned(ctx context.Context, clusterARN string) (int, error) {
	// For mock, we'll just return 0 deleted since we don't track service associations
	return 0, nil
}

func (m *MockTaskSetStore) GetTaskSets() map[string]*storage.TaskSet {
	return m.taskSets
}

// hasTaskFamily checks if task definition ARN contains the family name
func hasTaskFamily(taskDefArn, family string) bool {
	// Check if task definition ARN contains the family name
	if taskDefArn == "" || family == "" {
		return false
	}
	// Check various formats
	return taskDefArn == family || // exact match
		strings.Contains(taskDefArn, fmt.Sprintf("task-definition/%s:", family)) || // ARN format
		strings.Contains(taskDefArn, fmt.Sprintf(":%s:", family)) || // contains family with colons
		strings.Contains(taskDefArn, fmt.Sprintf("/%s:", family)) // ARN format
}

// MockContainerInstanceStore implements storage.ContainerInstanceStore for testing
type MockContainerInstanceStore struct {
	instances map[string]*storage.ContainerInstance
}

func NewMockContainerInstanceStore() *MockContainerInstanceStore {
	return &MockContainerInstanceStore{
		instances: make(map[string]*storage.ContainerInstance),
	}
}

func (m *MockContainerInstanceStore) Register(ctx context.Context, instance *storage.ContainerInstance) error {
	if m.instances == nil {
		m.instances = make(map[string]*storage.ContainerInstance)
	}
	if _, exists := m.instances[instance.ARN]; exists {
		return errors.New("container instance already exists")
	}
	m.instances[instance.ARN] = instance
	return nil
}

func (m *MockContainerInstanceStore) Get(ctx context.Context, arn string) (*storage.ContainerInstance, error) {
	instance, exists := m.instances[arn]
	if !exists {
		return nil, errors.New("container instance not found")
	}
	return instance, nil
}

func (m *MockContainerInstanceStore) ListWithPagination(ctx context.Context, cluster string, filters storage.ContainerInstanceFilters, limit int, nextToken string) ([]*storage.ContainerInstance, string, error) {
	// Convert map to slice
	var allInstances []*storage.ContainerInstance
	for _, instance := range m.instances {
		if instance.ClusterARN == cluster {
			// Apply status filter
			if filters.Status != "" && instance.Status != filters.Status {
				continue
			}
			allInstances = append(allInstances, instance)
		}
	}

	// Sort by ID for consistent ordering
	sort.Slice(allInstances, func(i, j int) bool {
		return allInstances[i].ID < allInstances[j].ID
	})

	// Find starting position based on nextToken
	start := 0
	if nextToken != "" {
		for i, instance := range allInstances {
			if instance.ID > nextToken {
				start = i
				break
			}
		}
	}

	// Get the requested page
	end := start + limit
	if end > len(allInstances) {
		end = len(allInstances)
	}

	result := allInstances[start:end]

	// Determine next token
	var newNextToken string
	if limit > 0 && end < len(allInstances) {
		// Use the last item's ID as the next token
		if len(result) > 0 {
			newNextToken = result[len(result)-1].ID
		}
	}

	return result, newNextToken, nil
}

func (m *MockContainerInstanceStore) Update(ctx context.Context, instance *storage.ContainerInstance) error {
	if _, exists := m.instances[instance.ARN]; !exists {
		return errors.New("container instance not found")
	}
	instance.UpdatedAt = time.Now()
	m.instances[instance.ARN] = instance
	return nil
}

func (m *MockContainerInstanceStore) Deregister(ctx context.Context, arn string) error {
	instance, exists := m.instances[arn]
	if !exists {
		return errors.New("container instance not found")
	}
	instance.Status = "INACTIVE"
	now := time.Now()
	instance.DeregisteredAt = &now
	instance.UpdatedAt = now
	return nil
}

func (m *MockContainerInstanceStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.ContainerInstance, error) {
	var results []*storage.ContainerInstance
	for _, arn := range arns {
		if instance, exists := m.instances[arn]; exists {
			results = append(results, instance)
		}
	}
	return results, nil
}

func (m *MockContainerInstanceStore) DeleteStale(ctx context.Context, clusterARN string, before time.Time) (int, error) {
	count := 0
	toDelete := []string{}
	for arn, instance := range m.instances {
		if instance.ClusterARN == clusterARN && instance.Status == "INACTIVE" && instance.UpdatedAt.Before(before) {
			toDelete = append(toDelete, arn)
			count++
		}
	}
	for _, arn := range toDelete {
		delete(m.instances, arn)
	}
	return count, nil
}

// MockAttributeStore implements storage.AttributeStore for testing
type MockAttributeStore struct {
	attributes map[string]*storage.Attribute
}

func NewMockAttributeStore() *MockAttributeStore {
	return &MockAttributeStore{
		attributes: make(map[string]*storage.Attribute),
	}
}

func (m *MockAttributeStore) Put(ctx context.Context, attributes []*storage.Attribute) error {
	if m.attributes == nil {
		m.attributes = make(map[string]*storage.Attribute)
	}
	for _, attr := range attributes {
		key := fmt.Sprintf("%s:%s:%s:%s", attr.Cluster, attr.TargetType, attr.TargetID, attr.Name)
		m.attributes[key] = attr
	}
	return nil
}

func (m *MockAttributeStore) Delete(ctx context.Context, cluster string, attributes []*storage.Attribute) error {
	for _, attr := range attributes {
		key := fmt.Sprintf("%s:%s:%s:%s", cluster, attr.TargetType, attr.TargetID, attr.Name)
		delete(m.attributes, key)
	}
	return nil
}

func (m *MockAttributeStore) ListWithPagination(ctx context.Context, targetType, cluster string, limit int, nextToken string) ([]*storage.Attribute, string, error) {
	// Convert map to slice
	var allAttributes []*storage.Attribute
	for _, attr := range m.attributes {
		// Apply filters
		if targetType != "" && attr.TargetType != targetType {
			continue
		}
		if cluster != "" && attr.Cluster != cluster {
			continue
		}
		allAttributes = append(allAttributes, attr)
	}

	// Sort by ID for consistent ordering
	sort.Slice(allAttributes, func(i, j int) bool {
		return allAttributes[i].ID < allAttributes[j].ID
	})

	// Find starting position based on nextToken
	start := 0
	if nextToken != "" {
		for i, attr := range allAttributes {
			if attr.ID > nextToken {
				start = i
				break
			}
		}
	}

	// Get the requested page
	end := start + limit
	if end > len(allAttributes) {
		end = len(allAttributes)
	}

	result := allAttributes[start:end]

	// Determine next token
	var newNextToken string
	if limit > 0 && end < len(allAttributes) {
		// Use the last item's ID as the next token
		if len(result) > 0 {
			newNextToken = result[len(result)-1].ID
		}
	}

	return result, newNextToken, nil
}

// MockTaskLogStore implements storage.TaskLogStore for testing
type MockTaskLogStore struct {
	logs []storage.TaskLog
}

// NewMockTaskLogStore creates a new mock task log store
func NewMockTaskLogStore() *MockTaskLogStore {
	return &MockTaskLogStore{
		logs: []storage.TaskLog{},
	}
}

// SaveLogs implements TaskLogStore
func (m *MockTaskLogStore) SaveLogs(ctx context.Context, logs []storage.TaskLog) error {
	m.logs = append(m.logs, logs...)
	return nil
}

// GetLogs implements TaskLogStore
func (m *MockTaskLogStore) GetLogs(ctx context.Context, filter storage.TaskLogFilter) ([]storage.TaskLog, error) {
	var result []storage.TaskLog

	for _, log := range m.logs {
		// Filter by task ARN
		if filter.TaskArn != "" && log.TaskArn != filter.TaskArn {
			continue
		}

		// Filter by container name
		if filter.ContainerName != "" && log.ContainerName != filter.ContainerName {
			continue
		}

		// Filter by log level
		if filter.LogLevel != "" && log.LogLevel != filter.LogLevel {
			continue
		}

		// Filter by time range
		if filter.From != nil && log.Timestamp.Before(*filter.From) {
			continue
		}
		if filter.To != nil && log.Timestamp.After(*filter.To) {
			continue
		}

		// Filter by search text (simple substring search)
		if filter.SearchText != "" {
			if !strings.Contains(log.LogLine, filter.SearchText) {
				continue
			}
		}

		result = append(result, log)
	}

	// Apply pagination
	start := filter.Offset
	end := filter.Offset + filter.Limit
	if end > len(result) {
		end = len(result)
	}
	if start > len(result) {
		start = len(result)
	}

	return result[start:end], nil
}

// GetLogCount implements TaskLogStore
func (m *MockTaskLogStore) GetLogCount(ctx context.Context, filter storage.TaskLogFilter) (int64, error) {
	// For simplicity, we'll reuse GetLogs without pagination
	filter.Offset = 0
	filter.Limit = len(m.logs) + 1
	logs, err := m.GetLogs(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int64(len(logs)), nil
}

// DeleteOldLogs implements TaskLogStore
func (m *MockTaskLogStore) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	var kept []storage.TaskLog
	var deletedCount int64

	for _, log := range m.logs {
		if log.CreatedAt.Before(olderThan) {
			deletedCount++
		} else {
			kept = append(kept, log)
		}
	}

	m.logs = kept
	return deletedCount, nil
}

// DeleteTaskLogs implements TaskLogStore
func (m *MockTaskLogStore) DeleteTaskLogs(ctx context.Context, taskArn string) error {
	var kept []storage.TaskLog

	for _, log := range m.logs {
		if log.TaskArn != taskArn {
			kept = append(kept, log)
		}
	}

	m.logs = kept
	return nil
}
