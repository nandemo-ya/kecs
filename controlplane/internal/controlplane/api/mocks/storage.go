package mocks

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockStorage implements storage.Storage interface for testing
type MockStorage struct {
	clusterStore        storage.ClusterStore
	taskDefinitionStore storage.TaskDefinitionStore
	serviceStore        storage.ServiceStore
	taskStore           storage.TaskStore
	accountSettingStore storage.AccountSettingStore
	taskSetStore        storage.TaskSetStore
}

func NewMockStorage() *MockStorage {
	return &MockStorage{}
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
		if familyPrefix == "" || family == familyPrefix {
			families = append(families, &storage.TaskDefinitionFamily{
				Family:         family,
				LatestRevision: len(revisions),
			})
		}
	}
	return families, "", nil
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
	key := fmt.Sprintf("%s:%s", cluster, taskID)
	task, exists := m.tasks[key]
	if !exists {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (m *MockTaskStore) List(ctx context.Context, cluster string, filters storage.TaskFilters) ([]*storage.Task, error) {
	var results []*storage.Task
	for _, task := range m.tasks {
		if task.ClusterARN != cluster {
			continue
		}
		// Apply filters
		if filters.ServiceName != "" && task.StartedBy != filters.ServiceName {
			continue
		}
		if filters.DesiredStatus != "" && task.DesiredStatus != filters.DesiredStatus {
			continue
		}
		if filters.LaunchType != "" && task.LaunchType != filters.LaunchType {
			continue
		}
		results = append(results, task)
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
func (m *MockTaskSetStore) GetTaskSets() map[string]*storage.TaskSet {
	return m.taskSets
}