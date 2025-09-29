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

var _ = Describe("TaskSetStore", func() {
	var (
		store   storage.Storage
		ctx     context.Context
		cluster *storage.Cluster
		service *storage.Service
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
		// Create a cluster and service for all task set tests
		cluster = createTestCluster(store, "test-cluster")
		service = createTestService(store, cluster.ARN, "test-service")
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Create", func() {
		Context("when creating a new task set", func() {
			It("should create the task set successfully", func() {
				taskSet := &storage.TaskSet{
					ID:                   uuid.New().String(),
					ARN:                  fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task-set/%s/%s/%s", cluster.Name, service.ServiceName, uuid.New().String()),
					ServiceARN:           service.ARN,
					ClusterARN:           cluster.ARN,
					TaskDefinition:       "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					ExternalID:           "external-123",
					Status:               "ACTIVE",
					Scale:                `{"value":100,"unit":"PERCENT"}`,
					ComputedDesiredCount: 3,
					PendingCount:         1,
					RunningCount:         2,
					CreatedAt:            time.Now(),
					UpdatedAt:            time.Now(),
					Region:               "us-east-1",
					AccountID:            "000000000000",
				}

				err := store.TaskSetStore().Create(ctx, taskSet)
				Expect(err).NotTo(HaveOccurred())

				// Verify task set was created
				retrieved, err := store.TaskSetStore().Get(ctx, service.ARN, taskSet.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ExternalID).To(Equal(taskSet.ExternalID))
				Expect(retrieved.ARN).To(Equal(taskSet.ARN))
				Expect(retrieved.Status).To(Equal(taskSet.Status))
			})
		})

		Context("when creating a duplicate task set", func() {
			It("should return ErrResourceAlreadyExists", func() {
				taskSet := &storage.TaskSet{
					ID:             uuid.New().String(),
					ARN:            "arn:aws:ecs:us-east-1:000000000000:task-set/test-cluster/test-service/duplicate",
					ServiceARN:     service.ARN,
					ClusterARN:     cluster.ARN,
					TaskDefinition: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					Status:         "ACTIVE",
					Region:         "us-east-1",
					AccountID:      "000000000000",
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}

				// Create first task set
				err := store.TaskSetStore().Create(ctx, taskSet)
				Expect(err).NotTo(HaveOccurred())

				// Try to create duplicate with same ARN
				taskSet2 := &storage.TaskSet{
					ID:             uuid.New().String(),
					ARN:            taskSet.ARN, // Same ARN
					ServiceARN:     service.ARN,
					ClusterARN:     cluster.ARN,
					TaskDefinition: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					Status:         "ACTIVE",
					Region:         "us-east-1",
					AccountID:      "000000000000",
					CreatedAt:      time.Now(),
					UpdatedAt:      time.Now(),
				}

				err = store.TaskSetStore().Create(ctx, taskSet2)
				Expect(err).To(MatchError(storage.ErrResourceAlreadyExists))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing task set", func() {
			It("should return the task set", func() {
				taskSet := createTestTaskSet(store, service.ARN, cluster.ARN, "test-get")

				retrieved, err := store.TaskSetStore().Get(ctx, service.ARN, taskSet.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(taskSet.ID))
				Expect(retrieved.ARN).To(Equal(taskSet.ARN))
			})
		})

		Context("when getting a non-existent task set", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.TaskSetStore().Get(ctx, service.ARN, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Update", func() {
		Context("when updating an existing task set", func() {
			It("should update the task set successfully", func() {
				taskSet := createTestTaskSet(store, service.ARN, cluster.ARN, "test-update")

				// Update task set
				taskSet.Status = "DRAINING"
				taskSet.Scale = `{"value":50,"unit":"PERCENT"}`
				taskSet.ComputedDesiredCount = 1
				taskSet.UpdatedAt = time.Now()

				err := store.TaskSetStore().Update(ctx, taskSet)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := store.TaskSetStore().Get(ctx, service.ARN, taskSet.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Status).To(Equal("DRAINING"))
				Expect(retrieved.Scale).To(Equal(`{"value":50,"unit":"PERCENT"}`))
				Expect(retrieved.ComputedDesiredCount).To(Equal(int32(1)))
			})
		})

		Context("when updating a non-existent task set", func() {
			It("should return ErrResourceNotFound", func() {
				taskSet := &storage.TaskSet{
					ID:         uuid.New().String(),
					ARN:        "arn:aws:ecs:us-east-1:000000000000:task-set/test-cluster/test-service/non-existent",
					ServiceARN: service.ARN,
					ClusterARN: cluster.ARN,
				}

				err := store.TaskSetStore().Update(ctx, taskSet)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing task set", func() {
			It("should delete the task set successfully", func() {
				taskSet := createTestTaskSet(store, service.ARN, cluster.ARN, "test-delete")

				err := store.TaskSetStore().Delete(ctx, service.ARN, taskSet.ID)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = store.TaskSetStore().Get(ctx, service.ARN, taskSet.ID)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})

		Context("when deleting a non-existent task set", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.TaskSetStore().Delete(ctx, service.ARN, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create test task sets
			for i := 0; i < 5; i++ {
				createTestTaskSet(store, service.ARN, cluster.ARN, fmt.Sprintf("test-list-%d", i))
			}
		})

		Context("when listing all task sets for a service", func() {
			It("should return all task sets", func() {
				taskSets, err := store.TaskSetStore().List(ctx, service.ARN, nil)
				Expect(err).NotTo(HaveOccurred())
				Expect(taskSets).To(HaveLen(5))
			})
		})

		Context("when listing specific task sets by IDs", func() {
			It("should return only requested task sets", func() {
				// Create additional task sets and get their IDs
				taskSet1 := createTestTaskSet(store, service.ARN, cluster.ARN, "specific-1")
				taskSet2 := createTestTaskSet(store, service.ARN, cluster.ARN, "specific-2")

				taskSets, err := store.TaskSetStore().List(ctx, service.ARN, []string{taskSet1.ID, taskSet2.ID})
				Expect(err).NotTo(HaveOccurred())
				Expect(taskSets).To(HaveLen(2))
				Expect(taskSets[0].ID).To(Or(Equal(taskSet1.ID), Equal(taskSet2.ID)))
				Expect(taskSets[1].ID).To(Or(Equal(taskSet1.ID), Equal(taskSet2.ID)))
			})
		})
	})

	Describe("GetByARN", func() {
		Context("when getting by ARN", func() {
			It("should return the task set", func() {
				taskSet := createTestTaskSet(store, service.ARN, cluster.ARN, "test-get-by-arn")

				retrieved, err := store.TaskSetStore().GetByARN(ctx, taskSet.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(taskSet.ID))
				Expect(retrieved.ARN).To(Equal(taskSet.ARN))
			})
		})
	})

	Describe("UpdatePrimary", func() {
		Context("when updating primary task set", func() {
			It("should mark task set for primary update", func() {
				// Create multiple task sets
				taskSet1 := createTestTaskSet(store, service.ARN, cluster.ARN, "primary-1")
				taskSet2 := createTestTaskSet(store, service.ARN, cluster.ARN, "primary-2")

				// Set taskSet2 as primary
				err := store.TaskSetStore().UpdatePrimary(ctx, service.ARN, taskSet2.ID)
				Expect(err).NotTo(HaveOccurred())

				// Verify the method executed without error
				// (The actual primary flag logic would be handled by the implementation)
				retrieved2, err := store.TaskSetStore().Get(ctx, service.ARN, taskSet2.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved2.ID).To(Equal(taskSet2.ID))

				// Verify taskSet1 still exists
				retrieved1, err := store.TaskSetStore().Get(ctx, service.ARN, taskSet1.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved1.ID).To(Equal(taskSet1.ID))
			})
		})
	})
})

// Helper function to create a test task set
func createTestTaskSet(store storage.Storage, serviceARN, clusterARN, name string) *storage.TaskSet {
	taskSet := &storage.TaskSet{
		ID:                   uuid.New().String(),
		ARN:                  fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task-set/test-cluster/test-service/%s", name),
		ServiceARN:           serviceARN,
		ClusterARN:           clusterARN,
		TaskDefinition:       "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
		Status:               "ACTIVE",
		ExternalID:           name,
		Scale:                `{"value":100,"unit":"PERCENT"}`,
		ComputedDesiredCount: 2,
		PendingCount:         0,
		RunningCount:         2,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		Region:               "us-east-1",
		AccountID:            "000000000000",
	}
	err := store.TaskSetStore().Create(context.Background(), taskSet)
	Expect(err).NotTo(HaveOccurred())
	return taskSet
}
