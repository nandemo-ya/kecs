package api

import (
	"context"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/memory"
	k8s "k8s.io/client-go/kubernetes"
)

// MockClusterManager is a mock implementation of ClusterManager for testing
type MockClusterManager struct {
	DeletedClusters []string
	ClusterMap      map[string]bool
}

func NewMockClusterManager() *MockClusterManager {
	return &MockClusterManager{
		DeletedClusters: []string{},
		ClusterMap:      make(map[string]bool),
	}
}

func (m *MockClusterManager) CreateCluster(ctx context.Context, clusterName string) error {
	m.ClusterMap[clusterName] = true
	return nil
}

func (m *MockClusterManager) DeleteCluster(ctx context.Context, clusterName string) error {
	m.DeletedClusters = append(m.DeletedClusters, clusterName)
	delete(m.ClusterMap, clusterName)
	return nil
}

func (m *MockClusterManager) ClusterExists(ctx context.Context, clusterName string) (bool, error) {
	return m.ClusterMap[clusterName], nil
}

func (m *MockClusterManager) GetKubeClient(clusterName string) (k8s.Interface, error) {
	return nil, nil
}

func (m *MockClusterManager) WaitForClusterReady(clusterName string, timeout time.Duration) error {
	return nil
}

func (m *MockClusterManager) GetKubeconfigPath(clusterName string) string {
	return ""
}

func (m *MockClusterManager) GetClusterInfo(ctx context.Context, clusterName string) (*kubernetes.ClusterInfo, error) {
	return nil, nil
}

var _ = Describe("Server Shutdown", func() {
	var (
		server         *Server
		mockStorage    storage.Storage
		mockClusterMgr *MockClusterManager
		ctx            context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockStorage = memory.NewMemoryStorage()
		mockClusterMgr = NewMockClusterManager()

		// Initialize storage
		err := mockStorage.Initialize(ctx)
		Expect(err).To(BeNil())

		// Create test clusters in storage
		cluster1 := &storage.Cluster{
			ID:             "1",
			Name:           "test-cluster-1",
			K8sClusterName: "kecs-test-cluster-1",
			ARN:            "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-1",
			Status:         "ACTIVE",
			Region:         "us-east-1",
			AccountID:      "123456789012",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = mockStorage.ClusterStore().Create(ctx, cluster1)
		Expect(err).To(BeNil())

		cluster2 := &storage.Cluster{
			ID:             "2",
			Name:           "test-cluster-2",
			K8sClusterName: "kecs-test-cluster-2",
			ARN:            "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-2",
			Status:         "ACTIVE",
			Region:         "us-east-1",
			AccountID:      "123456789012",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = mockStorage.ClusterStore().Create(ctx, cluster2)
		Expect(err).To(BeNil())

		// Mark clusters as existing in mock cluster manager
		mockClusterMgr.ClusterMap["kecs-test-cluster-1"] = true
		mockClusterMgr.ClusterMap["kecs-test-cluster-2"] = true

		// Create server with mocks
		server = &Server{
			storage:        mockStorage,
			clusterManager: mockClusterMgr,
			httpServer: &http.Server{
				Addr: ":8080",
			},
		}
	})

	AfterEach(func() {
		// Clean up environment variables
		os.Unsetenv("KECS_TEST_MODE")
		os.Unsetenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN")
	})

	Context("when KECS_TEST_MODE is not set", func() {
		Context("and KECS_KEEP_CLUSTERS_ON_SHUTDOWN is not set", func() {
			It("should delete all k3d clusters on shutdown", func() {
				// Call Stop
				shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				
				err := server.Stop(shutdownCtx)
				Expect(err).To(BeNil())

				// Verify clusters were deleted
				Expect(mockClusterMgr.DeletedClusters).To(ConsistOf(
					"kecs-test-cluster-1",
					"kecs-test-cluster-2",
				))
			})
		})

		Context("and KECS_KEEP_CLUSTERS_ON_SHUTDOWN is true", func() {
			It("should not delete k3d clusters on shutdown", func() {
				// Set environment variable
				os.Setenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN", "true")

				// Call Stop
				shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				
				err := server.Stop(shutdownCtx)
				Expect(err).To(BeNil())

				// Verify no clusters were deleted
				Expect(mockClusterMgr.DeletedClusters).To(BeEmpty())
			})
		})
	})

	Context("when KECS_TEST_MODE is true", func() {
		It("should not delete k3d clusters on shutdown", func() {
			// Set environment variable
			os.Setenv("KECS_TEST_MODE", "true")

			// Call Stop
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			
			err := server.Stop(shutdownCtx)
			Expect(err).To(BeNil())

			// Verify no clusters were deleted
			Expect(mockClusterMgr.DeletedClusters).To(BeEmpty())
		})
	})

	Context("when clusterManager is nil", func() {
		It("should gracefully handle shutdown without errors", func() {
			// Create server without cluster manager
			serverNoMgr := &Server{
				storage:        mockStorage,
				clusterManager: nil,
				httpServer: &http.Server{
					Addr: ":8080",
				},
			}

			// Call Stop
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			
			err := serverNoMgr.Stop(shutdownCtx)
			Expect(err).To(BeNil())
		})
	})

	Context("when storage is nil", func() {
		It("should gracefully handle shutdown without errors", func() {
			// Create server without storage
			serverNoStorage := &Server{
				storage:        nil,
				clusterManager: mockClusterMgr,
				httpServer: &http.Server{
					Addr: ":8080",
				},
			}

			// Call Stop
			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			
			err := serverNoStorage.Stop(shutdownCtx)
			Expect(err).To(BeNil())

			// Verify no clusters were deleted (can't list from storage)
			Expect(mockClusterMgr.DeletedClusters).To(BeEmpty())
		})
	})
})