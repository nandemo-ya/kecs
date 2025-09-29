package postgres_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("TaskDefinitionStore", func() {
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

	Describe("Register", func() {
		Context("when registering a new task definition", func() {
			It("should register with revision 1", func() {
				td := &storage.TaskDefinition{
					ID:                      uuid.New().String(),
					Family:                  "test-family",
					NetworkMode:             "bridge",
					RequiresCompatibilities: `["EC2"]`,
					CPU:                     "256",
					Memory:                  "512",
					ContainerDefinitions:    `[{"name":"app","image":"nginx:latest"}]`,
					Status:                  "ACTIVE",
					Region:                  "us-east-1",
					AccountID:               "000000000000",
				}

				registered, err := store.TaskDefinitionStore().Register(ctx, td)
				Expect(err).NotTo(HaveOccurred())
				Expect(registered.Revision).To(Equal(1))
				Expect(registered.ARN).To(Equal(fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task-definition/%s:1", td.Family)))
				Expect(registered.Family).To(Equal(td.Family))
			})
		})

		Context("when registering a task definition with existing family", func() {
			It("should increment revision number", func() {
				// Register first revision
				td1 := &storage.TaskDefinition{
					ID:                   uuid.New().String(),
					Family:               "revision-test",
					NetworkMode:          "bridge",
					ContainerDefinitions: `[{"name":"app","image":"nginx:1.0"}]`,
					Status:               "ACTIVE",
					Region:               "us-east-1",
					AccountID:            "000000000000",
				}

				registered1, err := store.TaskDefinitionStore().Register(ctx, td1)
				Expect(err).NotTo(HaveOccurred())
				Expect(registered1.Revision).To(Equal(1))

				// Register second revision
				td2 := &storage.TaskDefinition{
					ID:                   uuid.New().String(),
					Family:               "revision-test",
					NetworkMode:          "bridge",
					ContainerDefinitions: `[{"name":"app","image":"nginx:2.0"}]`,
					Status:               "ACTIVE",
					Region:               "us-east-1",
					AccountID:            "000000000000",
				}

				registered2, err := store.TaskDefinitionStore().Register(ctx, td2)
				Expect(err).NotTo(HaveOccurred())
				Expect(registered2.Revision).To(Equal(2))
				Expect(registered2.ARN).To(Equal("arn:aws:ecs:us-east-1:000000000000:task-definition/revision-test:2"))

				// Register third revision
				td3 := &storage.TaskDefinition{
					ID:                   uuid.New().String(),
					Family:               "revision-test",
					NetworkMode:          "bridge",
					ContainerDefinitions: `[{"name":"app","image":"nginx:3.0"}]`,
					Status:               "ACTIVE",
					Region:               "us-east-1",
					AccountID:            "000000000000",
				}

				registered3, err := store.TaskDefinitionStore().Register(ctx, td3)
				Expect(err).NotTo(HaveOccurred())
				Expect(registered3.Revision).To(Equal(3))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing task definition", func() {
			It("should return the task definition", func() {
				td := createTestTaskDefinition(store, "test-get")

				retrieved, err := store.TaskDefinitionStore().Get(ctx, td.Family, td.Revision)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ARN).To(Equal(td.ARN))
				Expect(retrieved.Family).To(Equal(td.Family))
				Expect(retrieved.Revision).To(Equal(td.Revision))
			})
		})

		Context("when getting a non-existent task definition", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.TaskDefinitionStore().Get(ctx, "non-existent", 1)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("GetLatest", func() {
		Context("when getting the latest revision", func() {
			It("should return the highest revision", func() {
				// Register multiple revisions
				for i := 1; i <= 3; i++ {
					td := &storage.TaskDefinition{
						ID:                   uuid.New().String(),
						Family:               "latest-test",
						NetworkMode:          "bridge",
						ContainerDefinitions: fmt.Sprintf(`[{"name":"app","image":"nginx:%d.0"}]`, i),
						Status:               "ACTIVE",
						Region:               "us-east-1",
						AccountID:            "000000000000",
					}
					_, err := store.TaskDefinitionStore().Register(ctx, td)
					Expect(err).NotTo(HaveOccurred())
				}

				// Get latest
				latest, err := store.TaskDefinitionStore().GetLatest(ctx, "latest-test")
				Expect(err).NotTo(HaveOccurred())
				Expect(latest.Revision).To(Equal(3))
				Expect(latest.ContainerDefinitions).To(ContainSubstring("nginx:3.0"))
			})
		})

		Context("when getting latest for non-existent family", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.TaskDefinitionStore().GetLatest(ctx, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create multiple families with multiple revisions
			families := []string{"family-a", "family-b", "family-c"}
			for _, family := range families {
				for rev := 1; rev <= 2; rev++ {
					td := &storage.TaskDefinition{
						ID:                   uuid.New().String(),
						Family:               family,
						NetworkMode:          "bridge",
						ContainerDefinitions: fmt.Sprintf(`[{"name":"app","image":"nginx:%d.0"}]`, rev),
						Status:               "ACTIVE",
						Region:               "us-east-1",
						AccountID:            "000000000000",
					}
					_, err := store.TaskDefinitionStore().Register(ctx, td)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})

		Context("when listing revisions of a family", func() {
			It("should return all revisions", func() {
				revisions, nextToken, err := store.TaskDefinitionStore().ListRevisions(ctx, "family-a", "ACTIVE", 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(revisions).To(HaveLen(2)) // 2 revisions of family-a
				Expect(nextToken).To(BeEmpty())
				for _, rev := range revisions {
					Expect(rev.Family).To(Equal("family-a"))
				}
			})
		})
	})

	Describe("ListFamilies", func() {
		BeforeEach(func() {
			// Create multiple families
			families := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
			for _, family := range families {
				td := &storage.TaskDefinition{
					ID:                   uuid.New().String(),
					Family:               family,
					NetworkMode:          "bridge",
					ContainerDefinitions: `[{"name":"app","image":"nginx:latest"}]`,
					Status:               "ACTIVE",
					Region:               "us-east-1",
					AccountID:            "000000000000",
				}
				_, err := store.TaskDefinitionStore().Register(ctx, td)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("when listing all families", func() {
			It("should return all unique families", func() {
				families, nextToken, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "ACTIVE", 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(families).To(HaveLen(5))
				Expect(nextToken).To(BeEmpty())
				// Check that each family is a TaskDefinitionFamily struct
				for _, family := range families {
					Expect(family.Family).NotTo(BeEmpty())
				}
			})
		})

		Context("when listing with pagination", func() {
			It("should return paginated results", func() {
				// First page
				families1, nextToken1, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "ACTIVE", 2, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(families1).To(HaveLen(2))
				Expect(nextToken1).NotTo(BeEmpty())

				// Second page
				families2, nextToken2, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "ACTIVE", 2, nextToken1)
				Expect(err).NotTo(HaveOccurred())
				Expect(families2).To(HaveLen(2))
				Expect(nextToken2).NotTo(BeEmpty())

				// Third page
				families3, nextToken3, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "ACTIVE", 2, nextToken2)
				Expect(err).NotTo(HaveOccurred())
				Expect(families3).To(HaveLen(1))
				Expect(nextToken3).To(BeEmpty())
			})
		})

		Context("when listing with family prefix", func() {
			It("should return only matching families", func() {
				// Create additional families with common prefix
				for _, suffix := range []string{"1", "2", "3"} {
					td := &storage.TaskDefinition{
						ID:                   uuid.New().String(),
						Family:               "test-" + suffix,
						NetworkMode:          "bridge",
						ContainerDefinitions: `[{"name":"app","image":"nginx:latest"}]`,
						Status:               "ACTIVE",
						Region:               "us-east-1",
						AccountID:            "000000000000",
					}
					_, err := store.TaskDefinitionStore().Register(ctx, td)
					Expect(err).NotTo(HaveOccurred())
				}

				families, _, err := store.TaskDefinitionStore().ListFamilies(ctx, "test-", "ACTIVE", 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(families).To(HaveLen(3))
				for _, family := range families {
					Expect(family.Family).To(HavePrefix("test-"))
				}
			})
		})
	})

	Describe("Deregister", func() {
		Context("when deregistering an existing task definition", func() {
			It("should mark it as INACTIVE", func() {
				td := createTestTaskDefinition(store, "test-deregister")

				err := store.TaskDefinitionStore().Deregister(ctx, td.Family, td.Revision)
				Expect(err).NotTo(HaveOccurred())

				// Verify task definition is now INACTIVE
				retrieved, err := store.TaskDefinitionStore().Get(ctx, td.Family, td.Revision)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.Status).To(Equal("INACTIVE"))
			})
		})

		Context("when deregistering a non-existent task definition", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.TaskDefinitionStore().Deregister(ctx, "non-existent", 1)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})
})
