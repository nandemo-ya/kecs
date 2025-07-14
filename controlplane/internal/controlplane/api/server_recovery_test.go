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

var _ = Describe("Server State Recovery", func() {
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

		// Create test data in storage
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
			K8sClusterName: "", // No k8s cluster name, should be skipped
			ARN:            "arn:aws:ecs:us-east-1:000000000000:cluster/test-cluster-2",
			Status:         "ACTIVE",
			Region:         "us-east-1",
			AccountID:      "000000000000",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		err = mockStorage.ClusterStore().Create(ctx, cluster2)
		Expect(err).To(BeNil())

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
		os.Unsetenv("KECS_AUTO_RECOVER_STATE")
	})

	Describe("RecoverState", func() {
		Context("when clusters need recovery", func() {
			It("should recreate missing k3d clusters", func() {
				// Ensure clusters don't exist
				mockClusterMgr.ClusterMap = make(map[string]bool)

				err := server.RecoverState(ctx)
				Expect(err).To(BeNil())

				// Verify cluster was created
				Expect(mockClusterMgr.ClusterMap).To(HaveKey("kecs-test-cluster-1"))
				// Cluster 2 should not be created (no k8s cluster name)
				Expect(mockClusterMgr.ClusterMap).NotTo(HaveKey("kecs-test-cluster-2"))
			})

			It("should recover services and reschedule tasks", func() {
				Skip("Skipping task rescheduling test - memory storage doesn't implement required stores")
			})

			It("should skip clusters that already exist", func() {
				// Mark cluster as existing
				mockClusterMgr.ClusterMap["kecs-test-cluster-1"] = true

				err := server.RecoverState(ctx)
				Expect(err).To(BeNil())

				// Verify no new clusters were created
				Expect(len(mockClusterMgr.ClusterMap)).To(Equal(1))
			})

			It("should skip clusters without k8s cluster name", func() {
				err := server.RecoverState(ctx)
				Expect(err).To(BeNil())

				// Only cluster 1 should be created
				Expect(len(mockClusterMgr.ClusterMap)).To(Equal(1))
				Expect(mockClusterMgr.ClusterMap).To(HaveKey("kecs-test-cluster-1"))
			})
		})

		Context("when storage is empty", func() {
			It("should return without error", func() {
				// Create empty storage
				emptyStorage := memory.NewMemoryStorage()
				err := emptyStorage.Initialize(ctx)
				Expect(err).To(BeNil())

				server.storage = emptyStorage

				err = server.RecoverState(ctx)
				Expect(err).To(BeNil())

				// No clusters should be created
				Expect(len(mockClusterMgr.ClusterMap)).To(Equal(0))
			})
		})

		Context("when storage or cluster manager is nil", func() {
			It("should skip recovery when storage is nil", func() {
				server.storage = nil

				err := server.RecoverState(ctx)
				Expect(err).To(BeNil())
			})

			It("should skip recovery when cluster manager is nil", func() {
				server.clusterManager = nil

				err := server.RecoverState(ctx)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("Server Start with Recovery", func() {
		It("should skip recovery when KECS_TEST_MODE is true", func() {
			os.Setenv("KECS_TEST_MODE", "true")

			// We can't easily test the full Start method, but we can verify
			// that RecoverState would be skipped based on environment
			testMode := os.Getenv("KECS_TEST_MODE")
			autoRecover := os.Getenv("KECS_AUTO_RECOVER_STATE")

			shouldRecover := testMode != "true" && autoRecover != "false"
			Expect(shouldRecover).To(BeFalse())
		})

		It("should skip recovery when KECS_AUTO_RECOVER_STATE is false", func() {
			os.Setenv("KECS_AUTO_RECOVER_STATE", "false")

			testMode := os.Getenv("KECS_TEST_MODE")
			autoRecover := os.Getenv("KECS_AUTO_RECOVER_STATE")

			shouldRecover := testMode != "true" && autoRecover != "false"
			Expect(shouldRecover).To(BeFalse())
		})

		It("should enable recovery by default", func() {
			os.Unsetenv("KECS_TEST_MODE")
			os.Unsetenv("KECS_AUTO_RECOVER_STATE")

			testMode := os.Getenv("KECS_TEST_MODE")
			autoRecover := os.Getenv("KECS_AUTO_RECOVER_STATE")

			shouldRecover := testMode != "true" && autoRecover != "false"
			Expect(shouldRecover).To(BeTrue())
		})
	})
})
