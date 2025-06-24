package api

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("Task ECS API", func() {
	var (
		server           *Server
		ctx              context.Context
		mockStorage      *mocks.MockStorage
		mockTaskStore    *mocks.MockTaskStore
		mockClusterStore *mocks.MockClusterStore
	)

	BeforeEach(func() {
		mockStorage = mocks.NewMockStorage()
		mockTaskStore = mocks.NewMockTaskStore()
		mockClusterStore = mocks.NewMockClusterStore()

		mockStorage.SetTaskStore(mockTaskStore)
		mockStorage.SetClusterStore(mockClusterStore)

		server = &Server{
			storage: mockStorage,
			ecsAPI:  NewDefaultECSAPI(mockStorage),
		}
		ctx = context.Background()

		// Pre-populate with test data
		// Add a default cluster
		cluster := &storage.Cluster{
			Name:      "default",
			ARN:       "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
			Status:    "ACTIVE",
			Region:    "ap-northeast-1",
			AccountID: "123456789012",
		}
		err := mockClusterStore.Create(ctx, cluster)
		Expect(err).To(BeNil())
	})

	Describe("RunTask", func() {
		var mockTaskDefStore *mocks.MockTaskDefinitionStore

		BeforeEach(func() {
			mockTaskDefStore = mocks.NewMockTaskDefinitionStore()
			mockStorage.SetTaskDefinitionStore(mockTaskDefStore)

			// Add a task definition
			taskDef := &storage.TaskDefinition{
				ARN:                  "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
				Family:               "nginx",
				Revision:             1,
				Status:               "ACTIVE",
				ContainerDefinitions: `[{"name":"nginx","image":"nginx:latest","memory":512}]`,
				Region:               "ap-northeast-1",
				AccountID:            "123456789012",
			}
			_, err := mockTaskDefStore.Register(ctx, taskDef)
			Expect(err).To(BeNil())
		})

		Context("when running a task", func() {
			It("should create a new task successfully", func() {
				taskDef := "nginx:1"
				req := &generated.RunTaskRequest{
					TaskDefinition: taskDef,
				}

				resp, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(1))
				Expect(resp.Failures).To(BeEmpty())

				task := resp.Tasks[0]
				Expect(*task.TaskDefinitionArn).To(ContainSubstring("nginx:1"))
				Expect(*task.LastStatus).To(Equal("PROVISIONING"))
				Expect(*task.DesiredStatus).To(Equal("RUNNING"))
			})

			It("should create multiple tasks with count", func() {
				taskDef := "nginx"
				count := int32(3)
				req := &generated.RunTaskRequest{
					TaskDefinition: taskDef,
					Count:          &count,
				}

				resp, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(3))
				Expect(resp.Failures).To(BeEmpty())
			})

			It("should fail when task definition not found", func() {
				taskDef := "non-existent:1"
				req := &generated.RunTaskRequest{
					TaskDefinition: taskDef,
				}

				_, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("task definition not found"))
			})

			It("should fail without task definition", func() {
				req := &generated.RunTaskRequest{}

				_, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("taskDefinition is required"))
			})
		})
	})

	Describe("StopTask", func() {
		Context("when stopping a task", func() {
			BeforeEach(func() {
				// Add a running task
				task := &storage.Task{
					ID:                "task-123",
					ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:task/default/task-123",
					ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
					TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
					LastStatus:        "RUNNING",
					DesiredStatus:     "RUNNING",
					LaunchType:        "EC2",
					Version:           1,
					CreatedAt:         time.Now(),
					Region:            "ap-northeast-1",
					AccountID:         "123456789012",
				}
				err := mockTaskStore.Create(ctx, task)
				Expect(err).To(BeNil())
			})

			It("should stop a running task", func() {
				taskID := "task-123"
				reason := "User requested stop"
				req := &generated.StopTaskRequest{
					Task:   taskID,
					Reason: &reason,
				}

				resp, err := server.ecsAPI.StopTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Task).NotTo(BeNil())
				Expect(*resp.Task.TaskArn).To(ContainSubstring("task-123"))
			})

			It("should be idempotent when task already stopped", func() {
				// Get and update task to stopped
				task, err := mockTaskStore.Get(ctx, "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default", "task-123")
				Expect(err).To(BeNil())
				task.DesiredStatus = "STOPPED"
				err = mockTaskStore.Update(ctx, task)
				Expect(err).To(BeNil())

				taskID := "task-123"
				req := &generated.StopTaskRequest{
					Task: taskID,
				}

				resp, err := server.ecsAPI.StopTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Task).NotTo(BeNil())
			})

			It("should fail when task not found", func() {
				taskID := "non-existent"
				req := &generated.StopTaskRequest{
					Task: taskID,
				}

				_, err := server.ecsAPI.StopTask(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("task not found"))
			})
		})
	})

	Describe("DescribeTasks", func() {
		Context("when describing tasks", func() {
			BeforeEach(func() {
				// Add test tasks
				for i := 1; i <= 3; i++ {
					task := &storage.Task{
						ID:                fmt.Sprintf("task-%d", i),
						ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task/default/task-%d", i),
						ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
						TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
						LastStatus:        "RUNNING",
						DesiredStatus:     "RUNNING",
						LaunchType:        "EC2",
						Version:           1,
						CreatedAt:         time.Now(),
						Region:            "ap-northeast-1",
						AccountID:         "123456789012",
						Tags:              `[{"key":"env","value":"test"}]`,
					}
					err := mockTaskStore.Create(ctx, task)
					Expect(err).To(BeNil())
				}
			})

			It("should describe multiple tasks", func() {
				req := &generated.DescribeTasksRequest{
					Tasks: []string{"task-1", "task-2"},
				}

				resp, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(2))
				Expect(resp.Failures).To(BeEmpty())
			})

			It("should include tags when requested", func() {
				req := &generated.DescribeTasksRequest{
					Tasks:   []string{"task-1"},
					Include: []generated.TaskField{generated.TaskFieldTAGS},
				}

				resp, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(1))
				Expect(resp.Tasks[0].Tags).To(HaveLen(1))
				Expect(string(*resp.Tasks[0].Tags[0].Key)).To(Equal("env"))
			})

			It("should report failures for non-existent tasks", func() {
				req := &generated.DescribeTasksRequest{
					Tasks: []string{"task-1", "non-existent"},
				}

				resp, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(1))
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})

			It("should fail without tasks", func() {
				req := &generated.DescribeTasksRequest{}

				_, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("tasks is required"))
			})
		})
	})

	Describe("ListTasks", func() {
		Context("when listing tasks", func() {
			BeforeEach(func() {
				// Add various test tasks
				tasks := []struct {
					id            string
					family        string
					launchType    string
					serviceName   string
					desiredStatus string
				}{
					{"task-web-1", "web", "FARGATE", "web-service", "RUNNING"},
					{"task-web-2", "web", "FARGATE", "web-service", "RUNNING"},
					{"task-api-1", "api", "EC2", "api-service", "RUNNING"},
					{"task-api-2", "api", "EC2", "api-service", "STOPPED"},
					{"task-batch-1", "batch", "EC2", "", "RUNNING"},
				}

				for _, t := range tasks {
					task := &storage.Task{
						ID:                t.id,
						ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task/default/%s", t.id),
						ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
						TaskDefinitionARN: fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/%s:1", t.family),
						LastStatus:        t.desiredStatus,
						DesiredStatus:     t.desiredStatus,
						LaunchType:        t.launchType,
						Version:           1,
						CreatedAt:         time.Now(),
						Region:            "ap-northeast-1",
						AccountID:         "123456789012",
					}
					if t.serviceName != "" {
						task.StartedBy = fmt.Sprintf("ecs-svc/%s", t.serviceName)
					}
					err := mockTaskStore.Create(ctx, task)
					Expect(err).To(BeNil())
				}
			})

			It("should list all tasks", func() {
				req := &generated.ListTasksRequest{}

				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(5))
			})

			It("should filter by service name", func() {
				serviceName := "web-service"
				req := &generated.ListTasksRequest{
					ServiceName: &serviceName,
				}

				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
				for _, arn := range resp.TaskArns {
					Expect(arn).To(ContainSubstring("task-web"))
				}
			})

			It("should filter by family", func() {
				family := "api"
				req := &generated.ListTasksRequest{
					Family: &family,
				}

				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
				for _, arn := range resp.TaskArns {
					Expect(arn).To(ContainSubstring("task-api"))
				}
			})

			It("should filter by launch type", func() {
				launchType := generated.LaunchTypeFARGATE
				req := &generated.ListTasksRequest{
					LaunchType: &launchType,
				}

				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
			})

			It("should filter by desired status", func() {
				status := generated.DesiredStatusRUNNING
				req := &generated.ListTasksRequest{
					DesiredStatus: &status,
				}

				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(4))
			})

			It("should respect max results", func() {
				maxResults := int32(2)
				req := &generated.ListTasksRequest{
					MaxResults: &maxResults,
				}

				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
				Expect(resp.NextToken).NotTo(BeNil())
			})
		})
	})
})
