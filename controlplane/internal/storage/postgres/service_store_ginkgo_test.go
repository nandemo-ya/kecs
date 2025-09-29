package postgres_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ServiceStore", func() {
	var (
		store   storage.Storage
		ctx     context.Context
		cluster *storage.Cluster
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
		// Create a cluster for all service tests
		cluster = createTestCluster(store, "test-cluster")
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Create", func() {
		Context("when creating a new service", func() {
			It("should create the service successfully", func() {
				service := &storage.Service{
					ID:                uuid.New().String(),
					ARN:               "arn:aws:ecs:us-east-1:000000000000:service/test-cluster/test-service",
					ServiceName:       "test-service",
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					DesiredCount:      3,
					RunningCount:      2,
					PendingCount:      1,
					Status:            "ACTIVE",
					LaunchType:        "EC2",
					Region:            "us-east-1",
					AccountID:         "000000000000",
				}

				err := store.ServiceStore().Create(ctx, service)
				Expect(err).NotTo(HaveOccurred())

				// Verify service was created
				retrieved, err := store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ServiceName).To(Equal(service.ServiceName))
				Expect(retrieved.ARN).To(Equal(service.ARN))
				Expect(retrieved.DesiredCount).To(Equal(service.DesiredCount))
			})
		})

		Context("when creating a duplicate service", func() {
			It("should return ErrResourceAlreadyExists", func() {
				service := &storage.Service{
					ID:          uuid.New().String(),
					ARN:         "arn:aws:ecs:us-east-1:000000000000:service/test-cluster/duplicate",
					ServiceName: "duplicate",
					ClusterARN:  cluster.ARN,
					Status:      "ACTIVE",
					Region:      "us-east-1",
					AccountID:   "000000000000",
				}

				// Create first service
				err := store.ServiceStore().Create(ctx, service)
				Expect(err).NotTo(HaveOccurred())

				// Try to create duplicate
				service2 := &storage.Service{
					ID:          uuid.New().String(),
					ARN:         service.ARN, // Same ARN
					ServiceName: service.ServiceName,
					ClusterARN:  cluster.ARN,
					Status:      "ACTIVE",
					Region:      "us-east-1",
					AccountID:   "000000000000",
				}

				err = store.ServiceStore().Create(ctx, service2)
				Expect(err).To(MatchError(storage.ErrResourceAlreadyExists))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing service", func() {
			It("should return the service", func() {
				service := createTestService(store, cluster.ARN, "test-get")

				retrieved, err := store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ServiceName).To(Equal(service.ServiceName))
				Expect(retrieved.ARN).To(Equal(service.ARN))
			})
		})

		Context("when getting a non-existent service", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.ServiceStore().Get(ctx, cluster.ARN, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Update", func() {
		Context("when updating an existing service", func() {
			It("should update the service successfully", func() {
				service := createTestService(store, cluster.ARN, "test-update")

				// Update service
				service.DesiredCount = 5
				service.RunningCount = 4
				service.Status = "UPDATING"

				err := store.ServiceStore().Update(ctx, service)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
				Expect(err).NotTo(HaveOccurred())
				Expect(int(retrieved.DesiredCount)).To(Equal(5))
				Expect(int(retrieved.RunningCount)).To(Equal(4))
				Expect(retrieved.Status).To(Equal("UPDATING"))
			})
		})

		Context("when updating a non-existent service", func() {
			It("should return ErrResourceNotFound", func() {
				service := &storage.Service{
					ID:          uuid.New().String(),
					ARN:         "arn:aws:ecs:us-east-1:000000000000:service/test-cluster/non-existent",
					ServiceName: "non-existent",
					ClusterARN:  cluster.ARN,
				}

				err := store.ServiceStore().Update(ctx, service)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing service", func() {
			It("should delete the service successfully", func() {
				service := createTestService(store, cluster.ARN, "test-delete")

				err := store.ServiceStore().Delete(ctx, cluster.ARN, service.ServiceName)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})

		Context("when deleting a non-existent service", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.ServiceStore().Delete(ctx, cluster.ARN, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create test services
			for i := 0; i < 5; i++ {
				createTestService(store, cluster.ARN, fmt.Sprintf("test-list-%d", i))
			}
		})

		Context("when listing all services", func() {
			It("should return all services", func() {
				services, nextToken, err := store.ServiceStore().List(ctx, cluster.ARN, "", "", 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(HaveLen(5))
				Expect(nextToken).To(BeEmpty()) // All services fit in one page
			})
		})

		Context("when listing with pagination", func() {
			It("should return paginated results", func() {
				// First page
				services1, nextToken1, err := store.ServiceStore().List(ctx, cluster.ARN, "", "", 2, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(services1).To(HaveLen(2))
				Expect(nextToken1).NotTo(BeEmpty())

				// Second page
				services2, nextToken2, err := store.ServiceStore().List(ctx, cluster.ARN, "", "", 2, nextToken1)
				Expect(err).NotTo(HaveOccurred())
				Expect(services2).To(HaveLen(2))
				Expect(nextToken2).NotTo(BeEmpty())

				// Ensure no duplicates
				names1 := make(map[string]bool)
				for _, s := range services1 {
					names1[s.ServiceName] = true
				}
				for _, s := range services2 {
					Expect(names1).NotTo(HaveKey(s.ServiceName))
				}
			})
		})

		Context("when listing with launch type filter", func() {
			It("should return only services with matching launch type", func() {
				// Create a FARGATE service
				fargateService := &storage.Service{
					ID:           uuid.New().String(),
					ARN:          fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:service/%s/fargate-service", cluster.ARN),
					ServiceName:  "fargate-service",
					ClusterARN:   cluster.ARN,
					LaunchType:   "FARGATE",
					Status:       "ACTIVE",
					DesiredCount: 1,
					Region:       "us-east-1",
					AccountID:    "000000000000",
				}
				err := store.ServiceStore().Create(ctx, fargateService)
				Expect(err).NotTo(HaveOccurred())

				// List only FARGATE services
				services, _, err := store.ServiceStore().List(ctx, cluster.ARN, "", "FARGATE", 10, "")
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(HaveLen(1))
				Expect(services[0].ServiceName).To(Equal("fargate-service"))
			})
		})
	})
})
