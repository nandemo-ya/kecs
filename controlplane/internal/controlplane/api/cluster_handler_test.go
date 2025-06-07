package api

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockStorage implements a simple in-memory storage for testing
type MockStorage struct {
	clusters map[string]*storage.Cluster
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		clusters: make(map[string]*storage.Cluster),
	}
}

func (m *MockStorage) ClusterStore() storage.ClusterStore {
	return &MockClusterStore{storage: m}
}

func (m *MockStorage) ServiceStore() storage.ServiceStore {
	return nil // Not needed for this test
}

func (m *MockStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return nil // Not needed for this test
}

func (m *MockStorage) TaskStore() storage.TaskStore {
	return nil // Not needed for this test
}

func (m *MockStorage) AccountSettingStore() storage.AccountSettingStore {
	return nil // Not needed for this test
}

func (m *MockStorage) Initialize(ctx context.Context) error {
	return nil
}

func (m *MockStorage) Close() error {
	return nil
}

func (m *MockStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return &MockTransaction{}, nil
}

// MockTransaction implements storage.Transaction
type MockTransaction struct{}

func (t *MockTransaction) Commit() error   { return nil }
func (t *MockTransaction) Rollback() error { return nil }


type MockClusterStore struct {
	storage *MockStorage
}

func (m *MockClusterStore) Create(ctx context.Context, cluster *storage.Cluster) error {
	if _, exists := m.storage.clusters[cluster.Name]; exists {
		return errors.New("cluster already exists")
	}
	m.storage.clusters[cluster.Name] = cluster
	return nil
}

func (m *MockClusterStore) Get(ctx context.Context, clusterName string) (*storage.Cluster, error) {
	cluster, exists := m.storage.clusters[clusterName]
	if !exists {
		return nil, errors.New("cluster not found")
	}
	return cluster, nil
}

func (m *MockClusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	clusters := make([]*storage.Cluster, 0, len(m.storage.clusters))
	for _, cluster := range m.storage.clusters {
		clusters = append(clusters, cluster)
	}
	return clusters, nil
}

func (m *MockClusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	m.storage.clusters[cluster.Name] = cluster
	return nil
}

func (m *MockClusterStore) Delete(ctx context.Context, clusterName string) error {
	delete(m.storage.clusters, clusterName)
	return nil
}

func TestCreateClusterWithRandomName(t *testing.T) {
	// Create a server with mock storage (kindManager is nil for test)
	server := &Server{
		storage:     NewMockStorage(),
		kindManager: nil, // Skip actual kind cluster creation in tests
	}

	// Test 1: Create cluster with a specific name
	req := &generated.CreateClusterRequest{
		"clusterName": "test-cluster",
	}

	resp, err := server.CreateClusterWithStorage(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateClusterWithStorage() error = %v", err)
	}

	// Verify response
	clusterData, ok := (*resp)["cluster"].(map[string]interface{})
	if !ok {
		t.Fatal("Response doesn't contain cluster data")
	}

	clusterName, _ := clusterData["clusterName"].(string)
	if clusterName != "test-cluster" {
		t.Errorf("Expected cluster name 'test-cluster', got %s", clusterName)
	}

	// Get the cluster from storage to check the kind cluster name
	cluster, err := server.storage.ClusterStore().Get(context.Background(), "test-cluster")
	if err != nil {
		t.Fatalf("Failed to get cluster from storage: %v", err)
	}

	// Verify that the kind cluster name follows the expected pattern
	if !strings.HasPrefix(cluster.KindClusterName, "kecs-") {
		t.Errorf("Kind cluster name should start with 'kecs-', got %s", cluster.KindClusterName)
	}

	// The name should have 3 parts: kecs-adjective-noun
	parts := strings.Split(cluster.KindClusterName, "-")
	if len(parts) != 3 {
		t.Errorf("Kind cluster name should have format 'kecs-adjective-noun', got %s", cluster.KindClusterName)
	}

	t.Logf("Created cluster '%s' with kind cluster name '%s'", clusterName, cluster.KindClusterName)

	// Test 2: Create another cluster and verify it gets a different random name
	req2 := &generated.CreateClusterRequest{
		"clusterName": "another-cluster",
	}

	_, err = server.CreateClusterWithStorage(context.Background(), req2)
	if err != nil {
		t.Fatalf("CreateClusterWithStorage() error = %v", err)
	}

	cluster2, err := server.storage.ClusterStore().Get(context.Background(), "another-cluster")
	if err != nil {
		t.Fatalf("Failed to get second cluster from storage: %v", err)
	}

	// Verify the two clusters have different kind cluster names
	if cluster.KindClusterName == cluster2.KindClusterName {
		t.Errorf("Two different clusters should have different kind cluster names, both got %s", cluster.KindClusterName)
	}

	t.Logf("Created second cluster 'another-cluster' with kind cluster name '%s'", cluster2.KindClusterName)
}

func TestCreateClusterIdempotency(t *testing.T) {
	// Create a server with mock storage
	server := &Server{
		storage:     NewMockStorage(),
		kindManager: nil, // Skip actual kind cluster creation in tests
	}

	// Test: Create cluster twice with the same name should return same cluster
	req := &generated.CreateClusterRequest{
		"clusterName": "idempotent-test",
	}

	// First call - should create the cluster
	resp1, err := server.CreateClusterWithStorage(context.Background(), req)
	if err != nil {
		t.Fatalf("First CreateClusterWithStorage() error = %v", err)
	}

	// Verify first response
	clusterData1, ok := (*resp1)["cluster"].(map[string]interface{})
	if !ok {
		t.Fatal("First response doesn't contain cluster data")
	}

	clusterArn1, _ := clusterData1["clusterArn"].(string)
	clusterName1, _ := clusterData1["clusterName"].(string)
	status1, _ := clusterData1["status"].(string)

	if clusterName1 != "idempotent-test" {
		t.Errorf("Expected cluster name 'idempotent-test', got %s", clusterName1)
	}
	if status1 != "ACTIVE" {
		t.Errorf("Expected status 'ACTIVE', got %s", status1)
	}

	// Second call - should return the existing cluster (idempotent)
	resp2, err := server.CreateClusterWithStorage(context.Background(), req)
	if err != nil {
		t.Fatalf("Second CreateClusterWithStorage() error = %v", err)
	}

	// Verify second response
	clusterData2, ok := (*resp2)["cluster"].(map[string]interface{})
	if !ok {
		t.Fatal("Second response doesn't contain cluster data")
	}

	clusterArn2, _ := clusterData2["clusterArn"].(string)
	clusterName2, _ := clusterData2["clusterName"].(string)
	status2, _ := clusterData2["status"].(string)

	// Verify both responses are identical
	if clusterArn1 != clusterArn2 {
		t.Errorf("ClusterArn mismatch: first=%s, second=%s", clusterArn1, clusterArn2)
	}
	if clusterName1 != clusterName2 {
		t.Errorf("ClusterName mismatch: first=%s, second=%s", clusterName1, clusterName2)
	}
	if status1 != status2 {
		t.Errorf("Status mismatch: first=%s, second=%s", status1, status2)
	}

	// Verify only one cluster exists in storage
	clusters, err := server.storage.ClusterStore().List(context.Background())
	if err != nil {
		t.Fatalf("Failed to list clusters: %v", err)
	}

	clusterCount := 0
	for _, cluster := range clusters {
		if cluster.Name == "idempotent-test" {
			clusterCount++
		}
	}

	if clusterCount != 1 {
		t.Errorf("Expected exactly 1 cluster with name 'idempotent-test', found %d", clusterCount)
	}

	t.Logf("Idempotent CreateCluster test passed - second call returned existing cluster")
}