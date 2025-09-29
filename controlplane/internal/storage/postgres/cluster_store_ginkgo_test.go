package postgres_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ClusterStore", func() {
	var (
		store storage.Storage
		ctx   context.Context
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Create", func() {
		Context("when creating a new cluster", func() {
			It("should create the cluster successfully", func() {
				cluster := &storage.Cluster{
					ID:        uuid.New().String(),
					ARN:       "arn:aws:ecs:us-east-1:000000000000:cluster/test-cluster",
					Name:      "test-cluster",
					Status:    "ACTIVE",
					Region:    "us-east-1",
					AccountID: "000000000000",
				}

				err := store.ClusterStore().Create(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())

				// Verify cluster was created
				retrieved, err := store.ClusterStore().Get(ctx, cluster.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal(cluster.Name))
				Expect(retrieved.ARN).To(Equal(cluster.ARN))
				Expect(retrieved.Status).To(Equal(cluster.Status))
			})
		})

		Context("when creating a duplicate cluster", func() {
			It("should return ErrResourceAlreadyExists", func() {
				cluster := &storage.Cluster{
					ID:        uuid.New().String(),
					ARN:       "arn:aws:ecs:us-east-1:000000000000:cluster/duplicate",
					Name:      "duplicate",
					Status:    "ACTIVE",
					Region:    "us-east-1",
					AccountID: "000000000000",
				}

				// Create first cluster
				err := store.ClusterStore().Create(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())

				// Try to create duplicate
				cluster2 := &storage.Cluster{
					ID:        uuid.New().String(),
					ARN:       cluster.ARN, // Same ARN
					Name:      cluster.Name,
					Status:    "ACTIVE",
					Region:    "us-east-1",
					AccountID: "000000000000",
				}

				err = store.ClusterStore().Create(ctx, cluster2)
				Expect(err).To(MatchError(storage.ErrResourceAlreadyExists))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing cluster", func() {
			It("should return the cluster", func() {
				cluster := createTestCluster(store, "test-get")

				retrieved, err := store.ClusterStore().Get(ctx, cluster.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Name).To(Equal(cluster.Name))
				Expect(retrieved.ARN).To(Equal(cluster.ARN))
			})
		})

		Context("when getting a non-existent cluster", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.ClusterStore().Get(ctx, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Update", func() {
		Context("when updating an existing cluster", func() {
			It("should update the cluster successfully", func() {
				cluster := createTestCluster(store, "test-update")

				// Update cluster
				cluster.Status = "PROVISIONING"
				cluster.Tags = `{"Environment": "test"}`

				err := store.ClusterStore().Update(ctx, cluster)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := store.ClusterStore().Get(ctx, cluster.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Status).To(Equal("PROVISIONING"))
				Expect(retrieved.Tags).To(Equal(`{"Environment": "test"}`))
			})
		})

		Context("when updating a non-existent cluster", func() {
			It("should return ErrResourceNotFound", func() {
				cluster := &storage.Cluster{
					ID:   uuid.New().String(),
					ARN:  "arn:aws:ecs:us-east-1:000000000000:cluster/non-existent",
					Name: "non-existent",
				}

				err := store.ClusterStore().Update(ctx, cluster)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing cluster", func() {
			It("should delete the cluster successfully", func() {
				cluster := createTestCluster(store, "test-delete")

				err := store.ClusterStore().Delete(ctx, cluster.Name)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = store.ClusterStore().Get(ctx, cluster.Name)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})

		Context("when deleting a non-existent cluster", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.ClusterStore().Delete(ctx, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("List", func() {
		Context("when listing clusters", func() {
			It("should return all clusters", func() {
				// Create test clusters
				for i := 0; i < 5; i++ {
					createTestCluster(store, fmt.Sprintf("test-list-%d", i))
				}

				clusters, err := store.ClusterStore().List(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(clusters).To(HaveLen(5))
			})
		})
	})

	Describe("ListWithPagination", func() {
		Context("when listing with pagination", func() {
			BeforeEach(func() {
				// Create test clusters
				for i := 0; i < 10; i++ {
					createTestCluster(store, fmt.Sprintf("test-page-%02d", i))
				}
			})

			It("should return the first page", func() {
				result, nextToken, err := store.ClusterStore().ListWithPagination(ctx, 3, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(HaveLen(3))
				Expect(nextToken).NotTo(BeEmpty())
			})

			It("should return subsequent pages", func() {
				// Get first page
				result1, nextToken1, err := store.ClusterStore().ListWithPagination(ctx, 3, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(nextToken1).NotTo(BeEmpty())

				// Get second page
				result2, nextToken2, err := store.ClusterStore().ListWithPagination(ctx, 3, nextToken1)
				Expect(err).NotTo(HaveOccurred())
				Expect(result2).To(HaveLen(3))
				Expect(nextToken2).NotTo(BeEmpty())

				// Ensure no duplicates between pages
				ids1 := make(map[string]bool)
				for _, c := range result1 {
					ids1[c.ID] = true
				}
				for _, c := range result2 {
					Expect(ids1).NotTo(HaveKey(c.ID))
				}
			})

			It("should handle the last page correctly", func() {
				var lastToken string
				pageCount := 0

				for {
					_, nextToken, err := store.ClusterStore().ListWithPagination(ctx, 3, lastToken)
					Expect(err).NotTo(HaveOccurred())
					pageCount++
					if nextToken == "" {
						break
					}
					lastToken = nextToken
				}

				// 10 items with page size 3 = 4 pages (3+3+3+1)
				Expect(pageCount).To(Equal(4))
			})

			It("should return error for invalid next token", func() {
				_, _, err := store.ClusterStore().ListWithPagination(ctx, 3, "invalid-token")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
