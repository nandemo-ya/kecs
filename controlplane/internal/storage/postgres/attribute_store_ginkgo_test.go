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

var _ = Describe("AttributeStore", func() {
	var (
		store   storage.Storage
		ctx     context.Context
		cluster *storage.Cluster
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
		// Create a cluster for attribute tests
		cluster = createTestCluster(store, "test-cluster")
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Put", func() {
		Context("when putting new attributes", func() {
			It("should store attributes successfully", func() {
				attributes := []*storage.Attribute{
					{
						ID:         uuid.New().String(),
						Name:       "ecs.instance-type",
						Value:      "t2.micro",
						TargetType: "container-instance",
						TargetID:   "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/test-instance",
						Cluster:    cluster.Name,
						Region:     "us-east-1",
						AccountID:  "000000000000",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					},
					{
						ID:         uuid.New().String(),
						Name:       "ecs.availability-zone",
						Value:      "us-east-1a",
						TargetType: "container-instance",
						TargetID:   "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/test-instance",
						Cluster:    cluster.Name,
						Region:     "us-east-1",
						AccountID:  "000000000000",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					},
				}

				err := store.AttributeStore().Put(ctx, attributes)
				Expect(err).NotTo(HaveOccurred())

				// Verify attributes were stored by listing them
				storedAttrs, _, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(storedAttrs).To(HaveLen(2))
			})
		})

		Context("when updating existing attributes", func() {
			It("should update attribute values", func() {
				// Create initial attribute
				attr := &storage.Attribute{
					ID:         uuid.New().String(),
					Name:       "ecs.os-type",
					Value:      "linux",
					TargetType: "container-instance",
					TargetID:   "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/update-test",
					Cluster:    cluster.Name,
					Region:     "us-east-1",
					AccountID:  "000000000000",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}

				err := store.AttributeStore().Put(ctx, []*storage.Attribute{attr})
				Expect(err).NotTo(HaveOccurred())

				// Update the attribute value
				attr.Value = "windows"
				attr.UpdatedAt = time.Now()

				err = store.AttributeStore().Put(ctx, []*storage.Attribute{attr})
				Expect(err).NotTo(HaveOccurred())

				// Verify the update
				storedAttrs, _, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 10, "")
				Expect(err).NotTo(HaveOccurred())

				// Find the updated attribute
				var found bool
				for _, stored := range storedAttrs {
					if stored.Name == "ecs.os-type" && stored.TargetID == attr.TargetID {
						Expect(stored.Value).To(Equal("windows"))
						found = true
						break
					}
				}
				Expect(found).To(BeTrue())
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting existing attributes", func() {
			It("should delete attributes successfully", func() {
				// Create attributes to delete
				attributes := []*storage.Attribute{
					{
						ID:         uuid.New().String(),
						Name:       "delete-test-1",
						Value:      "value1",
						TargetType: "container-instance",
						TargetID:   "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/delete-test",
						Cluster:    cluster.Name,
						Region:     "us-east-1",
						AccountID:  "000000000000",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					},
					{
						ID:         uuid.New().String(),
						Name:       "delete-test-2",
						Value:      "value2",
						TargetType: "container-instance",
						TargetID:   "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/delete-test",
						Cluster:    cluster.Name,
						Region:     "us-east-1",
						AccountID:  "000000000000",
						CreatedAt:  time.Now(),
						UpdatedAt:  time.Now(),
					},
				}

				err := store.AttributeStore().Put(ctx, attributes)
				Expect(err).NotTo(HaveOccurred())

				// Delete the attributes
				err = store.AttributeStore().Delete(ctx, cluster.Name, attributes)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				storedAttrs, _, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 10, "")
				Expect(err).NotTo(HaveOccurred())

				// Should not find the deleted attributes
				for _, stored := range storedAttrs {
					Expect(stored.Name).NotTo(Equal("delete-test-1"))
					Expect(stored.Name).NotTo(Equal("delete-test-2"))
				}
			})
		})

		Context("when deleting non-existent attributes", func() {
			It("should not return error", func() {
				// Try to delete non-existent attributes
				attributes := []*storage.Attribute{
					{
						Name:       "non-existent",
						TargetType: "container-instance",
						TargetID:   "arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/non-existent",
						Cluster:    cluster.Name,
					},
				}

				// Should not error on deleting non-existent attributes
				err := store.AttributeStore().Delete(ctx, cluster.Name, attributes)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("ListWithPagination", func() {
		BeforeEach(func() {
			// Create test attributes
			for i := 0; i < 5; i++ {
				attr := &storage.Attribute{
					ID:         uuid.New().String(),
					Name:       fmt.Sprintf("attr-%d", i),
					Value:      fmt.Sprintf("value-%d", i),
					TargetType: "container-instance",
					TargetID:   fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:container-instance/test-cluster/instance-%d", i),
					Cluster:    cluster.Name,
					Region:     "us-east-1",
					AccountID:  "000000000000",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				err := store.AttributeStore().Put(ctx, []*storage.Attribute{attr})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when listing all attributes", func() {
			It("should return all attributes", func() {
				attrs, nextToken, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(attrs).To(HaveLen(5))
				Expect(nextToken).To(BeEmpty())
			})
		})

		Context("when listing with pagination", func() {
			It("should return paginated results", func() {
				// First page
				attrs1, nextToken1, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 2, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(attrs1).To(HaveLen(2))
				Expect(nextToken1).NotTo(BeEmpty())

				// Second page
				attrs2, nextToken2, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 2, nextToken1)
				Expect(err).NotTo(HaveOccurred())
				Expect(attrs2).To(HaveLen(2))
				Expect(nextToken2).NotTo(BeEmpty())

				// Third page
				attrs3, nextToken3, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 2, nextToken2)
				Expect(err).NotTo(HaveOccurred())
				Expect(attrs3).To(HaveLen(1))
				Expect(nextToken3).To(BeEmpty())
			})
		})

		Context("when listing with empty cluster", func() {
			It("should return empty list for non-existent cluster", func() {
				attrs, nextToken, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", "non-existent-cluster", 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(attrs).To(BeEmpty())
				Expect(nextToken).To(BeEmpty())
			})
		})

		Context("when listing with different target types", func() {
			It("should filter by target type", func() {
				// Add an attribute with different target type
				attr := &storage.Attribute{
					ID:         uuid.New().String(),
					Name:       "different-type",
					Value:      "value",
					TargetType: "task", // Different target type
					TargetID:   "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/task-1",
					Cluster:    cluster.Name,
					Region:     "us-east-1",
					AccountID:  "000000000000",
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				err := store.AttributeStore().Put(ctx, []*storage.Attribute{attr})
				Expect(err).NotTo(HaveOccurred())

				// List only container-instance attributes
				attrs, _, err := store.AttributeStore().ListWithPagination(ctx, "container-instance", cluster.Name, 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(attrs).To(HaveLen(5)) // Should not include the task attribute

				// List only task attributes
				taskAttrs, _, err := store.AttributeStore().ListWithPagination(ctx, "task", cluster.Name, 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(taskAttrs).To(HaveLen(1))
				Expect(taskAttrs[0].TargetType).To(Equal("task"))
			})
		})
	})
})
