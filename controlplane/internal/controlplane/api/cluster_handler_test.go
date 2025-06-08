package api

import (
	"context"
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var _ = Describe("ClusterHandler", func() {
	var (
		server *Server
		ctx    context.Context
	)

	BeforeEach(func() {
		server = &Server{
			storage:     NewMockStorage(),
			kindManager: nil, // Skip actual kind cluster creation in tests
		}
		ctx = context.Background()
	})

	Describe("CreateClusterWithStorage", func() {
		Context("when creating a cluster with random name", func() {
			It("should create cluster with a specific name", func() {
				req := &generated.CreateClusterRequest{
					"clusterName": "test-cluster",
				}

				resp, err := server.CreateClusterWithStorage(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify response
				clusterData, ok := (*resp)["cluster"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				clusterName, _ := clusterData["clusterName"].(string)
				Expect(clusterName).To(Equal("test-cluster"))

				// Get the cluster from storage to check the kind cluster name
				cluster, err := server.storage.ClusterStore().Get(ctx, "test-cluster")
				Expect(err).NotTo(HaveOccurred())

				// Verify that the kind cluster name follows the expected pattern
				Expect(cluster.KindClusterName).To(HavePrefix("kecs-"))

				// The name should have 3 parts: kecs-adjective-noun
				parts := strings.Split(cluster.KindClusterName, "-")
				Expect(parts).To(HaveLen(3))
			})

			It("should create different random names for different clusters", func() {
				// Create first cluster
				req1 := &generated.CreateClusterRequest{
					"clusterName": "test-cluster-1",
				}
				_, err := server.CreateClusterWithStorage(ctx, req1)
				Expect(err).NotTo(HaveOccurred())

				cluster1, err := server.storage.ClusterStore().Get(ctx, "test-cluster-1")
				Expect(err).NotTo(HaveOccurred())

				// Create second cluster
				req2 := &generated.CreateClusterRequest{
					"clusterName": "test-cluster-2",
				}
				_, err = server.CreateClusterWithStorage(ctx, req2)
				Expect(err).NotTo(HaveOccurred())

				cluster2, err := server.storage.ClusterStore().Get(ctx, "test-cluster-2")
				Expect(err).NotTo(HaveOccurred())

				// Verify the two clusters have different kind cluster names
				Expect(cluster1.KindClusterName).NotTo(Equal(cluster2.KindClusterName))
			})
		})

		Context("when creating a cluster with idempotency", func() {
			It("should return existing cluster when name already exists", func() {
				req := &generated.CreateClusterRequest{
					"clusterName": "idempotent-test",
				}

				// First call - should create the cluster
				resp1, err := server.CreateClusterWithStorage(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify first response
				clusterData1, ok := (*resp1)["cluster"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				clusterArn1, _ := clusterData1["clusterArn"].(string)
				clusterName1, _ := clusterData1["clusterName"].(string)
				status1, _ := clusterData1["status"].(string)

				Expect(clusterName1).To(Equal("idempotent-test"))
				Expect(status1).To(Equal("ACTIVE"))

				// Second call - should return the existing cluster (idempotent)
				resp2, err := server.CreateClusterWithStorage(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify second response
				clusterData2, ok := (*resp2)["cluster"].(map[string]interface{})
				Expect(ok).To(BeTrue())

				clusterArn2, _ := clusterData2["clusterArn"].(string)
				clusterName2, _ := clusterData2["clusterName"].(string)
				status2, _ := clusterData2["status"].(string)

				// Verify both responses are identical
				Expect(clusterArn1).To(Equal(clusterArn2))
				Expect(clusterName1).To(Equal(clusterName2))
				Expect(status1).To(Equal(status2))

				// Verify only one cluster exists in storage
				clusters, err := server.storage.ClusterStore().List(ctx)
				Expect(err).NotTo(HaveOccurred())

				clusterCount := 0
				for _, cluster := range clusters {
					if cluster.Name == "idempotent-test" {
						clusterCount++
					}
				}

				Expect(clusterCount).To(Equal(1))
			})
		})
	})
})