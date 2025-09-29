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

var _ = Describe("TaskStore", func() {
	var (
		store   storage.Storage
		ctx     context.Context
		cluster *storage.Cluster
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
		// Create a cluster for all task tests
		cluster = createTestCluster(store, "test-cluster")
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Create", func() {
		Context("when creating a new task", func() {
			It("should create the task successfully", func() {
				task := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/%s/%s", cluster.Name, uuid.New().String()),
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					DesiredStatus:     "RUNNING",
					LastStatus:        "PENDING",
					LaunchType:        "EC2",
					PlatformVersion:   "LATEST",
					CPU:               "256",
					Memory:            "512",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}

				err := store.TaskStore().Create(ctx, task)
				Expect(err).NotTo(HaveOccurred())

				// Verify task was created
				retrieved, err := store.TaskStore().Get(ctx, cluster.ARN, task.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ARN).To(Equal(task.ARN))
				Expect(retrieved.TaskDefinitionARN).To(Equal(task.TaskDefinitionARN))
				Expect(retrieved.LastStatus).To(Equal(task.LastStatus))
			})
		})

		Context("when creating a duplicate task", func() {
			It("should return ErrResourceAlreadyExists", func() {
				task := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/duplicate",
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					DesiredStatus:     "RUNNING",
					LastStatus:        "PENDING",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}

				// Create first task
				err := store.TaskStore().Create(ctx, task)
				Expect(err).NotTo(HaveOccurred())

				// Try to create duplicate with same ARN
				task2 := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               task.ARN, // Same ARN
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: task.TaskDefinitionARN,
					DesiredStatus:     "RUNNING",
					LastStatus:        "PENDING",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}

				err = store.TaskStore().Create(ctx, task2)
				Expect(err).To(MatchError(storage.ErrResourceAlreadyExists))
			})
		})
	})

	Describe("Get", func() {
		Context("when getting an existing task", func() {
			It("should return the task", func() {
				task := createTestTask(store, cluster.ARN, uuid.New().String())

				retrieved, err := store.TaskStore().Get(ctx, cluster.ARN, task.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.ID).To(Equal(task.ID))
				Expect(retrieved.ARN).To(Equal(task.ARN))
			})
		})

		Context("when getting a non-existent task", func() {
			It("should return ErrResourceNotFound", func() {
				_, err := store.TaskStore().Get(ctx, cluster.ARN, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Update", func() {
		Context("when updating an existing task", func() {
			It("should update the task successfully", func() {
				task := createTestTask(store, cluster.ARN, uuid.New().String())

				// Update task status
				task.LastStatus = "RUNNING"
				task.DesiredStatus = "RUNNING"
				now := time.Now()
				task.StartedAt = &now

				err := store.TaskStore().Update(ctx, task)
				Expect(err).NotTo(HaveOccurred())

				// Verify update
				retrieved, err := store.TaskStore().Get(ctx, cluster.ARN, task.ID)
				Expect(err).NotTo(HaveOccurred())
				Expect(retrieved.LastStatus).To(Equal("RUNNING"))
				Expect(retrieved.DesiredStatus).To(Equal("RUNNING"))
				Expect(retrieved.StartedAt).NotTo(BeZero())
			})
		})

		Context("when updating a non-existent task", func() {
			It("should return ErrResourceNotFound", func() {
				task := &storage.Task{
					ID:         uuid.New().String(),
					ARN:        "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/non-existent",
					ClusterARN: cluster.ARN,
				}

				err := store.TaskStore().Update(ctx, task)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("Delete", func() {
		Context("when deleting an existing task", func() {
			It("should delete the task successfully", func() {
				task := createTestTask(store, cluster.ARN, uuid.New().String())

				err := store.TaskStore().Delete(ctx, cluster.ARN, task.ID)
				Expect(err).NotTo(HaveOccurred())

				// Verify deletion
				_, err = store.TaskStore().Get(ctx, cluster.ARN, task.ID)
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})

		Context("when deleting a non-existent task", func() {
			It("should return ErrResourceNotFound", func() {
				err := store.TaskStore().Delete(ctx, cluster.ARN, "non-existent")
				Expect(err).To(MatchError(storage.ErrResourceNotFound))
			})
		})
	})

	Describe("List", func() {
		BeforeEach(func() {
			// Create test tasks
			for i := 0; i < 5; i++ {
				createTestTask(store, cluster.ARN, fmt.Sprintf("task-%d", i))
			}
		})

		Context("when listing all tasks", func() {
			It("should return all tasks", func() {
				tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{})
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(HaveLen(5))
			})
		})

		Context("when listing with family filter", func() {
			It("should return only tasks with matching family", func() {
				// Create a task with specific task definition
				task := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/%s/family-task", cluster.Name),
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/special-family:1",
					DesiredStatus:     "RUNNING",
					LastStatus:        "RUNNING",
					LaunchType:        "EC2",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}
				err := store.TaskStore().Create(ctx, task)
				Expect(err).NotTo(HaveOccurred())

				// List with family filter
				tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{
					Family: "special-family",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(HaveLen(1))
				Expect(tasks[0].TaskDefinitionARN).To(ContainSubstring("special-family"))
			})
		})

		Context("when listing with status filter", func() {
			It("should return only tasks with matching status", func() {
				// Create a stopped task
				task := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/%s/stopped-task", cluster.Name),
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					DesiredStatus:     "STOPPED",
					LastStatus:        "STOPPED",
					LaunchType:        "EC2",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}
				stoppedTime := time.Now()
				task.StoppedAt = &stoppedTime
				err := store.TaskStore().Create(ctx, task)
				Expect(err).NotTo(HaveOccurred())

				// List only stopped tasks
				tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{
					DesiredStatus: "STOPPED",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(tasks).To(HaveLen(1))
				Expect(tasks[0].DesiredStatus).To(Equal("STOPPED"))
			})
		})
	})

	Describe("ListByService", func() {
		var service *storage.Service

		BeforeEach(func() {
			// Create a service
			service = createTestService(store, cluster.ARN, "test-service")

			// Create tasks for the service
			for i := 0; i < 3; i++ {
				task := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/%s/service-task-%d", cluster.Name, i),
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					Group:             "service:" + service.ServiceName,
					DesiredStatus:     "RUNNING",
					LastStatus:        "RUNNING",
					LaunchType:        "EC2",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}
				err := store.TaskStore().Create(ctx, task)
				Expect(err).NotTo(HaveOccurred())
			}

			// Create tasks for other services
			for i := 0; i < 2; i++ {
				task := &storage.Task{
					ID:                uuid.New().String(),
					ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/%s/other-task-%d", cluster.Name, i),
					ClusterARN:        cluster.ARN,
					TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
					Group:             "service:other-service",
					DesiredStatus:     "RUNNING",
					LastStatus:        "RUNNING",
					LaunchType:        "EC2",
					Region:            "us-east-1",
					AccountID:         "000000000000",
					CreatedAt:         time.Now(),
				}
				err := store.TaskStore().Create(ctx, task)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should return only tasks for the specified service", func() {
			tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{
				ServiceName: service.ServiceName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(3))
			for _, task := range tasks {
				Expect(task.Group).To(Equal("service:" + service.ServiceName))
			}
		})
	})
})
