package postgres_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ContainerInstanceStore", func() {
	var (
		store   storage.Storage
		ctx     context.Context
		cluster *storage.Cluster
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
		// Create a cluster for container instance tests
		cluster = createTestCluster(store, "test-cluster")
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Register", func() {
		Context("when registering a new container instance", func() {
			It("should register the instance successfully", func() {
				instance := &storage.ContainerInstance{
					ID:                  uuid.New().String(),
					ARN:                 fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:container-instance/%s/%s", cluster.Name, uuid.New().String()),
					ClusterARN:          cluster.ARN,
					EC2InstanceID:       "i-1234567890abcdef0",
					Status:              "ACTIVE",
					StatusReason:        "",
					AgentConnected:      true,
					RunningTasksCount:   0,
					PendingTasksCount:   0,
					RegisteredAt:        time.Now(),
					RegisteredResources: `[{"name":"CPU","type":"INTEGER","integerValue":1024}]`,
					RemainingResources:  `[{"name":"CPU","type":"INTEGER","integerValue":1024}]`,
					VersionInfo:         `{"agentVersion":"1.0.0"}`,
					HealthStatus:        `{"overallStatus":"OK"}`,
					Region:              "us-east-1",
					AccountID:           "000000000000",
				}

				err := store.ContainerInstanceStore().Register(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify instance was registered
				retrieved, err := store.ContainerInstanceStore().Get(ctx, instance.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.EC2InstanceID).To(Equal(instance.EC2InstanceID))
				Expect(retrieved.Status).To(Equal(instance.Status))
			})
		})

		Context("when registering a duplicate container instance", func() {
			It("should return ErrResourceAlreadyExists", func() {
				instance := &storage.ContainerInstance{
					ID:             uuid.New().String(),
					ARN:            "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/duplicate",
					ClusterARN:     cluster.ARN,
					EC2InstanceID:  "i-duplicate",
					Status:         "ACTIVE",
					AgentConnected: true,
					RegisteredAt:   time.Now(),
					Region:         "us-east-1",
					AccountID:      "000000000000",
				}

				// Register first instance
				err := store.ContainerInstanceStore().Register(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Try to register duplicate with same ARN
				instance2 := &storage.ContainerInstance{
					ID:             uuid.New().String(),
					ARN:            instance.ARN, // Same ARN
					ClusterARN:     cluster.ARN,
					EC2InstanceID:  "i-duplicate2",
					Status:         "ACTIVE",
					AgentConnected: true,
					RegisteredAt:   time.Now(),
					Region:         "us-east-1",
					AccountID:      "000000000000",
				}

				err = store.ContainerInstanceStore().Register(ctx, instance2)
				Expect(err).To(MatchError(storage.ErrResourceAlreadyExists))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing container instance", func() {
			It("should return the instance", func() {
				instance := createTestContainerInstance(store, cluster.ARN, "test-get")

				retrieved, err := store.ContainerInstanceStore().Get(ctx, instance.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(instance.ID))
				Expect(retrieved.ARN).To(Equal(instance.ARN))
				Expect(retrieved.EC2InstanceID).To(Equal(instance.EC2InstanceID))
			})
		})

		Context("when getting a non-existent container instance", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.ContainerInstanceStore().Get(ctx, "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Update", func() {
		Context("when updating an existing container instance", func() {
			It("should update the instance successfully", func() {
				instance := createTestContainerInstance(store, cluster.ARN, "test-update")

				// Update instance
				instance.Status = "DRAINING"
				instance.StatusReason = "User requested draining"
				instance.RunningTasksCount = 2
				instance.PendingTasksCount = 1

				err := store.ContainerInstanceStore().Update(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := store.ContainerInstanceStore().Get(ctx, instance.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Status).To(Equal("DRAINING"))
				Expect(retrieved.StatusReason).To(Equal("User requested draining"))
				Expect(retrieved.RunningTasksCount).To(Equal(int32(2)))
				Expect(retrieved.PendingTasksCount).To(Equal(int32(1)))
			})
		})

		Context("when updating a non-existent container instance", func() {
			It("should return ErrResourceNotFound", func() {
				instance := &storage.ContainerInstance{
					ID:         uuid.New().String(),
					ARN:        "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/non-existent",
					ClusterARN: cluster.ARN,
				}

				err := store.ContainerInstanceStore().Update(ctx, instance)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Deregister", func() {
		Context("when deregistering an existing container instance", func() {
			It("should mark the instance as INACTIVE", func() {
				instance := createTestContainerInstance(store, cluster.ARN, "test-deregister")

				err := store.ContainerInstanceStore().Deregister(ctx, instance.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Verify instance is now INACTIVE
				retrieved, err := store.ContainerInstanceStore().Get(ctx, instance.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Status).To(Equal("INACTIVE"))
			})
		})

		Context("when deregistering a non-existent container instance", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.ContainerInstanceStore().Deregister(ctx, "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("ListWithPagination", func() {
		BeforeEach(func() {
			// Create test container instances
			for i := 0; i < 5; i++ {
				createTestContainerInstance(store, cluster.ARN, fmt.Sprintf("test-list-%d", i))
			}
		})

		Context("when listing all container instances", func() {
			It("should return all instances", func() {
				instances, nextToken, err := store.ContainerInstanceStore().ListWithPagination(
					ctx, cluster.ARN, storage.ContainerInstanceFilters{}, 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(HaveLen(5))
				Expect(nextToken).To(BeEmpty())
			})
		})

		Context("when listing with status filter", func() {
			It("should return only active instances", func() {
				// Create a draining instance
				instance := createTestContainerInstance(store, cluster.ARN, "test-draining")
				instance.Status = "DRAINING"
				err := store.ContainerInstanceStore().Update(ctx, instance)
				Expect(err).NotTo(HaveOccurred())

				instances, _, err := store.ContainerInstanceStore().ListWithPagination(
					ctx, cluster.ARN, storage.ContainerInstanceFilters{Status: "ACTIVE"}, 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(HaveLen(5)) // Only the original 5 ACTIVE instances
				for _, inst := range instances {
					Expect(inst.Status).To(Equal("ACTIVE"))
				}
			})
		})

		Context("when listing with pagination", func() {
			It("should return paginated results", func() {
				// First page
				instances1, nextToken1, err := store.ContainerInstanceStore().ListWithPagination(
					ctx, cluster.ARN, storage.ContainerInstanceFilters{}, 2, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(instances1).To(HaveLen(2))
				Expect(nextToken1).NotTo(BeEmpty())

				// Second page
				instances2, nextToken2, err := store.ContainerInstanceStore().ListWithPagination(
					ctx, cluster.ARN, storage.ContainerInstanceFilters{}, 2, nextToken1)
				Expect(err).NotTo(HaveOccurred())
				Expect(instances2).To(HaveLen(2))
				Expect(nextToken2).NotTo(BeEmpty())

				// Third page
				instances3, nextToken3, err := store.ContainerInstanceStore().ListWithPagination(
					ctx, cluster.ARN, storage.ContainerInstanceFilters{}, 2, nextToken2)
				Expect(err).NotTo(HaveOccurred())
				Expect(instances3).To(HaveLen(1))
				Expect(nextToken3).To(BeEmpty())
			})
		})
	})

	Describe("GetByARNs", func() {
		Context("when getting multiple instances by ARNs", func() {
			It("should return all requested instances", func() {
				instance1 := createTestContainerInstance(store, cluster.ARN, "test-batch-1")
				instance2 := createTestContainerInstance(store, cluster.ARN, "test-batch-2")
				instance3 := createTestContainerInstance(store, cluster.ARN, "test-batch-3")

				arns := []string{instance1.ARN, instance2.ARN, instance3.ARN}
				instances, err := store.ContainerInstanceStore().GetByARNs(ctx, arns)
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(HaveLen(3))

				// Verify all ARNs are present
				returnedARNs := make(map[string]bool)
				for _, inst := range instances {
					returnedARNs[inst.ARN] = true
				}
				Expect(returnedARNs).To(HaveKey(instance1.ARN))
				Expect(returnedARNs).To(HaveKey(instance2.ARN))
				Expect(returnedARNs).To(HaveKey(instance3.ARN))
			})
		})

		Context("when some ARNs don't exist", func() {
			It("should return only existing instances", func() {
				instance1 := createTestContainerInstance(store, cluster.ARN, "test-partial")
				nonExistentARN := "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/non-existent"

				arns := []string{instance1.ARN, nonExistentARN}
				instances, err := store.ContainerInstanceStore().GetByARNs(ctx, arns)
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(HaveLen(1))
				Expect(instances[0].ARN).To(Equal(instance1.ARN))
			})
		})

		Context("when ARN list is empty", func() {
			It("should return empty list", func() {
				instances, err := store.ContainerInstanceStore().GetByARNs(ctx, []string{})
				Expect(err).NotTo(HaveOccurred())
				Expect(instances).To(BeEmpty())
			})
		})
	})

	Describe("DeleteStale", func() {
		Context("when deleting stale instances", func() {
			It("should delete instances older than cutoff time", func() {
				// Create instances with different ages
				oldInstance := createTestContainerInstance(store, cluster.ARN, "test-old")
				oldInstance.RegisteredAt = time.Now().Add(-48 * time.Hour)
				err := store.ContainerInstanceStore().Update(ctx, oldInstance)
				Expect(err).NotTo(HaveOccurred())

				recentInstance := createTestContainerInstance(store, cluster.ARN, "test-recent")

				// Delete instances older than 24 hours
				cutoffTime := time.Now().Add(-24 * time.Hour)
				deletedCount, err := store.ContainerInstanceStore().DeleteStale(ctx, cluster.ARN, cutoffTime)
				Expect(err).NotTo(HaveOccurred())
				Expect(deletedCount).To(Equal(1))

				// Verify old instance is deleted
				_, err = store.ContainerInstanceStore().Get(ctx, oldInstance.ARN)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))

				// Verify recent instance still exists
				retrieved, err := store.ContainerInstanceStore().Get(ctx, recentInstance.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ARN).To(Equal(recentInstance.ARN))
			})
		})
	})
})

// Helper function to create a test container instance
func createTestContainerInstance(store storage.Storage, clusterARN, name string) *storage.ContainerInstance {
	instance := &storage.ContainerInstance{
		ID:                  uuid.New().String(),
		ARN:                 fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/%s", name),
		ClusterARN:          clusterARN,
		EC2InstanceID:       fmt.Sprintf("i-%s", name),
		Status:              "ACTIVE",
		AgentConnected:      true,
		RunningTasksCount:   0,
		PendingTasksCount:   0,
		RegisteredAt:        time.Now(),
		RegisteredResources: `[{"name":"CPU","type":"INTEGER","integerValue":1024}]`,
		RemainingResources:  `[{"name":"CPU","type":"INTEGER","integerValue":1024}]`,
		VersionInfo:         `{"agentVersion":"1.0.0"}`,
		HealthStatus:        `{"overallStatus":"OK"}`,
		Region:              "us-east-1",
		AccountID:           "000000000000",
	}
	err := store.ContainerInstanceStore().Register(context.Background(), instance)
	Expect(err).NotTo(HaveOccurred())
	return instance
}
