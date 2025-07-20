package api

import (
	"context"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/memory"
)

var _ = Describe("Server Shutdown", func() {
	var (
		server           *Server
		mockStorage      storage.Storage
		mockClusterMgr   *MockClusterManager
		ctx              context.Context
		origTestMode     string
		origKeepClusters string
	)

	BeforeEach(func() {
		// Save original environment variables
		origTestMode = os.Getenv("KECS_TEST_MODE")
		origKeepClusters = os.Getenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN")

		// Clear environment variables for tests
		os.Unsetenv("KECS_TEST_MODE")
		os.Unsetenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN")
		ctx = context.Background()
		mockStorage = memory.NewMemoryStorage()
		mockClusterMgr = NewMockClusterManager()

		// Reset mock state
		mockClusterMgr.DeletedClusters = []string{}

		// Initialize storage
		err := mockStorage.Initialize(ctx)
		Expect(err).To(BeNil())

		// Create test clusters in storage
		cluster1 := &storage.Cluster{
			ID:             "1",
			Name:           "test-cluster-1",
			K8sClusterName: "kecs-test-cluster-1",
			ARN:            "arn:aws:ecs:us-east-1:000000000000:cluster/test-cluster-1",
			Status:         "ACTIVE",
			Region:         "us-east-1",
			AccountID:      "000000000000",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = mockStorage.ClusterStore().Create(ctx, cluster1)
		Expect(err).To(BeNil())

		cluster2 := &storage.Cluster{
			ID:             "2",
			Name:           "test-cluster-2",
			K8sClusterName: "kecs-test-cluster-2",
			ARN:            "arn:aws:ecs:us-east-1:000000000000:cluster/test-cluster-2",
			Status:         "ACTIVE",
			Region:         "us-east-1",
			AccountID:      "000000000000",
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
		// Restore original environment variables
		if origTestMode != "" {
			os.Setenv("KECS_TEST_MODE", origTestMode)
		} else {
			os.Unsetenv("KECS_TEST_MODE")
		}
		if origKeepClusters != "" {
			os.Setenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN", origKeepClusters)
		} else {
			os.Unsetenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN")
		}
	})

	Context("when KECS_TEST_MODE is not set", func() {
		Context("and KECS_KEEP_CLUSTERS_ON_SHUTDOWN is not set", func() {
			It("should not delete k3d clusters on shutdown (new architecture)", func() {
				// Ensure environment variables are not set
				os.Unsetenv("KECS_TEST_MODE")
				os.Unsetenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN")

				// Debug: Check environment variables
				testMode := os.Getenv("KECS_TEST_MODE")
				keepClusters := os.Getenv("KECS_KEEP_CLUSTERS_ON_SHUTDOWN")
				GinkgoWriter.Printf("Before test - KECS_TEST_MODE: '%s', KECS_KEEP_CLUSTERS_ON_SHUTDOWN: '%s'\n", testMode, keepClusters)

				// Call Stop
				shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				err := server.Stop(shutdownCtx)
				Expect(err).To(BeNil())

				// Debug: Print what was deleted
				GinkgoWriter.Printf("Deleted clusters: %v\n", mockClusterMgr.DeletedClusters)

				// In the new architecture, k3d clusters are not deleted by the API server
				// They are managed by the CLI
				Expect(len(mockClusterMgr.DeletedClusters)).To(Equal(0))
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
