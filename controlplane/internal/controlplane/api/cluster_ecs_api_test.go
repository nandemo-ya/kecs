package api

import (
	"context"
	"errors"

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

var _ = Describe("Cluster ECS API", func() {
	var (
		server *Server
		ctx    context.Context
	)

	BeforeEach(func() {
		mockStorage := NewMockStorage()
		server = &Server{
			storage:     mockStorage,
			kindManager: nil, // Skip actual kind cluster creation in tests
			ecsAPI:      NewDefaultECSAPI(mockStorage, nil),
		}
		ctx = context.Background()
	})

	Describe("CreateCluster", func() {
		Context("when creating a cluster with random name", func() {
			It("should create cluster with a specific name", func() {
				clusterName := "test-cluster"
				req := &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				}

				resp, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify response
				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(resp.Cluster.ClusterName).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("test-cluster"))

				// Get the cluster from storage to check the kind cluster name
				cluster, err := server.storage.ClusterStore().Get(ctx, "test-cluster")
				Expect(err).NotTo(HaveOccurred())

				// Verify that the kind cluster name follows the expected pattern
				Expect(cluster.KindClusterName).To(HavePrefix("kecs-"))
				
				// Should be kecs-<cluster-name>
				Expect(cluster.KindClusterName).To(Equal("kecs-test-cluster"))
			})

			It("should create different random names for different clusters", func() {
				// Create first cluster
				clusterName1 := "test-cluster-1"
				req1 := &generated.CreateClusterRequest{
					ClusterName: &clusterName1,
				}
				_, err := server.ecsAPI.CreateCluster(ctx, req1)
				Expect(err).NotTo(HaveOccurred())

				cluster1, err := server.storage.ClusterStore().Get(ctx, "test-cluster-1")
				Expect(err).NotTo(HaveOccurred())

				// Create second cluster
				clusterName2 := "test-cluster-2"
				req2 := &generated.CreateClusterRequest{
					ClusterName: &clusterName2,
				}
				_, err = server.ecsAPI.CreateCluster(ctx, req2)
				Expect(err).NotTo(HaveOccurred())

				cluster2, err := server.storage.ClusterStore().Get(ctx, "test-cluster-2")
				Expect(err).NotTo(HaveOccurred())

				// Verify the two clusters have different kind cluster names
				Expect(cluster1.KindClusterName).NotTo(Equal(cluster2.KindClusterName))
			})
		})

		Context("when creating a cluster with idempotency", func() {
			It("should return existing cluster when name already exists", func() {
				clusterName := "idempotent-test"
				req := &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				}

				// First call - should create the cluster
				resp1, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify first response
				Expect(resp1).NotTo(BeNil())
				Expect(resp1.Cluster).NotTo(BeNil())
				Expect(resp1.Cluster.ClusterArn).NotTo(BeNil())
				Expect(resp1.Cluster.ClusterName).NotTo(BeNil())
				Expect(resp1.Cluster.Status).NotTo(BeNil())

				clusterArn1 := *resp1.Cluster.ClusterArn
				clusterName1 := *resp1.Cluster.ClusterName
				status1 := *resp1.Cluster.Status

				Expect(clusterName1).To(Equal("idempotent-test"))
				Expect(status1).To(Equal("ACTIVE"))

				// Second call - should return the existing cluster (idempotent)
				resp2, err := server.ecsAPI.CreateCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify second response
				Expect(resp2).NotTo(BeNil())
				Expect(resp2.Cluster).NotTo(BeNil())
				Expect(resp2.Cluster.ClusterArn).NotTo(BeNil())
				Expect(resp2.Cluster.ClusterName).NotTo(BeNil())
				Expect(resp2.Cluster.Status).NotTo(BeNil())

				clusterArn2 := *resp2.Cluster.ClusterArn
				clusterName2 := *resp2.Cluster.ClusterName
				status2 := *resp2.Cluster.Status

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

	Describe("ListClusters", func() {
		Context("when listing clusters", func() {
			It("should return empty list when no clusters exist", func() {
				req := &generated.ListClustersRequest{}
				
				resp, err := server.ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(BeEmpty())
			})

			It("should return all cluster ARNs", func() {
				// Create test clusters
				clusterNames := []string{"cluster-1", "cluster-2", "cluster-3"}
				for _, name := range clusterNames {
					clusterName := name
					_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
						ClusterName: &clusterName,
					})
					Expect(err).NotTo(HaveOccurred())
				}

				// List clusters
				req := &generated.ListClustersRequest{}
				resp, err := server.ecsAPI.ListClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.ClusterArns).To(HaveLen(3))
				
				// Verify all ARNs are present
				arnMap := make(map[string]bool)
				for _, arn := range resp.ClusterArns {
					arnMap[arn] = true
				}
				
				for _, name := range clusterNames {
					expectedArn := "arn:aws:ecs:ap-northeast-1:123456789012:cluster/" + name
					Expect(arnMap).To(HaveKey(expectedArn))
				}
			})
		})
	})

	Describe("DescribeClusters", func() {
		Context("when describing clusters", func() {
			BeforeEach(func() {
				// Create test clusters
				clusterNames := []string{"describe-test-1", "describe-test-2"}
				for _, name := range clusterNames {
					clusterName := name
					_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
						ClusterName: &clusterName,
					})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should describe all clusters when no specific clusters requested", func() {
				req := &generated.DescribeClustersRequest{}
				
				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(HaveLen(2))
				Expect(resp.Failures).To(BeEmpty())
			})

			It("should describe specific clusters by name", func() {
				req := &generated.DescribeClustersRequest{
					Clusters: []string{"describe-test-1"},
				}
				
				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(HaveLen(1))
				Expect(*resp.Clusters[0].ClusterName).To(Equal("describe-test-1"))
				Expect(resp.Failures).To(BeEmpty())
			})

			It("should return failure for non-existent cluster", func() {
				req := &generated.DescribeClustersRequest{
					Clusters: []string{"non-existent-cluster"},
				}
				
				resp, err := server.ecsAPI.DescribeClusters(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Clusters).To(BeEmpty())
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})
		})
	})

	Describe("DeleteCluster", func() {
		Context("when deleting a cluster", func() {
			It("should delete an existing cluster", func() {
				// Create a cluster first
				clusterName := "delete-test"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Delete the cluster
				req := &generated.DeleteClusterRequest{
					Cluster: &clusterName,
				}
				
				resp, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Cluster).NotTo(BeNil())
				Expect(*resp.Cluster.ClusterName).To(Equal("delete-test"))
				Expect(*resp.Cluster.Status).To(Equal("INACTIVE"))

				// Verify cluster is deleted from storage
				_, err = server.storage.ClusterStore().Get(ctx, "delete-test")
				Expect(err).To(HaveOccurred())
			})

			It("should fail when cluster has active services", func() {
				// Create a cluster with active services count
				clusterName := "cluster-with-services"
				_, err := server.ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
					ClusterName: &clusterName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Simulate active services by updating the cluster
				cluster, err := server.storage.ClusterStore().Get(ctx, clusterName)
				Expect(err).NotTo(HaveOccurred())
				cluster.ActiveServicesCount = 1
				err = server.storage.ClusterStore().Update(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())

				// Try to delete the cluster
				req := &generated.DeleteClusterRequest{
					Cluster: &clusterName,
				}
				
				_, err = server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("active services"))
			})

			It("should fail when cluster does not exist", func() {
				clusterName := "non-existent"
				req := &generated.DeleteClusterRequest{
					Cluster: &clusterName,
				}
				
				_, err := server.ecsAPI.DeleteCluster(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})
})