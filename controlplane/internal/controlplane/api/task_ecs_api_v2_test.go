package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("Task ECS API V2", func() {
	var (
		ecsAPIV2    *api.DefaultECSAPIV2
		testStorage storage.Storage
		ctx         context.Context
	)

	BeforeEach(func() {
		// Create test storage
		var err error
		testStorage, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).ToNot(HaveOccurred())

		// Initialize tables
		err = testStorage.Initialize(context.Background())
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()

		// Initialize V2 API
		ecsAPIV2 = api.NewDefaultECSAPIV2(testStorage, nil)

		// Create a test cluster
		cluster := &storage.Cluster{
			Name:   "test-cluster",
			ARN:    "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
			Status: "ACTIVE",
		}
		err = testStorage.ClusterStore().Create(ctx, cluster)
		Expect(err).ToNot(HaveOccurred())

		// Create a test task definition
		taskDef := &storage.TaskDefinition{
			Family:               "test-task",
			Revision:             1,
			ARN:                  "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
			ContainerDefinitions: `[{"name":"app","image":"nginx:latest","memory":512,"essential":true}]`,
			Status:               "ACTIVE",
			Region:               "ap-northeast-1",
			AccountID:            "123456789012",
		}
		_, err = testStorage.TaskDefinitionStore().Register(ctx, taskDef)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if testStorage != nil {
			testStorage.Close()
		}
	})

	Describe("RunTaskV2", func() {
		It("should run a single task", func() {
			req := &ecs.RunTaskInput{
				Cluster:        aws.String("test-cluster"),
				TaskDefinition: aws.String("test-task:1"),
			}
			resp, err := ecsAPIV2.RunTaskV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Tasks).To(HaveLen(1))
			Expect(*resp.Tasks[0].TaskDefinitionArn).To(ContainSubstring("test-task:1"))
			Expect(*resp.Tasks[0].LastStatus).To(Equal("PENDING"))
			Expect(*resp.Tasks[0].DesiredStatus).To(Equal("RUNNING"))
		})

		It("should run multiple tasks", func() {
			req := &ecs.RunTaskInput{
				Cluster:        aws.String("test-cluster"),
				TaskDefinition: aws.String("test-task"),
				Count:          aws.Int32(3),
			}
			resp, err := ecsAPIV2.RunTaskV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Tasks).To(HaveLen(3))
			for _, task := range resp.Tasks {
				Expect(*task.TaskDefinitionArn).To(ContainSubstring("test-task:1"))
			}
		})

		It("should handle task overrides", func() {
			req := &ecs.RunTaskInput{
				Cluster:        aws.String("test-cluster"),
				TaskDefinition: aws.String("test-task:1"),
				Overrides: &ecstypes.TaskOverride{
					ContainerOverrides: []ecstypes.ContainerOverride{
						{
							Name: aws.String("app"),
							Environment: []ecstypes.KeyValuePair{
								{
									Name:  aws.String("ENV"),
									Value: aws.String("test"),
								},
							},
						},
					},
				},
			}
			resp, err := ecsAPIV2.RunTaskV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Tasks).To(HaveLen(1))
			Expect(resp.Tasks[0].Overrides).ToNot(BeNil())
			Expect(resp.Tasks[0].Overrides.ContainerOverrides).To(HaveLen(1))
		})

		It("should handle launch type and other options", func() {
			req := &ecs.RunTaskInput{
				Cluster:              aws.String("test-cluster"),
				TaskDefinition:       aws.String("test-task:1"),
				LaunchType:           ecstypes.LaunchTypeFargate,
				PlatformVersion:      aws.String("1.4.0"),
				EnableExecuteCommand: true,
				StartedBy:            aws.String("test-user"),
				Group:                aws.String("test-group"),
				Tags: []ecstypes.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				},
			}
			resp, err := ecsAPIV2.RunTaskV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Tasks).To(HaveLen(1))
			
			task := resp.Tasks[0]
			Expect(task.LaunchType).To(Equal(ecstypes.LaunchTypeFargate))
			Expect(*task.PlatformVersion).To(Equal("1.4.0"))
			Expect(task.EnableExecuteCommand).To(BeTrue())
			Expect(*task.StartedBy).To(Equal("test-user"))
			Expect(*task.Group).To(Equal("test-group"))
			Expect(task.Tags).To(HaveLen(1))
		})

		It("should fail when task definition not found", func() {
			req := &ecs.RunTaskInput{
				Cluster:        aws.String("test-cluster"),
				TaskDefinition: aws.String("non-existent:1"),
			}
			_, err := ecsAPIV2.RunTaskV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("task definition not found"))
		})

		It("should fail when cluster not found", func() {
			req := &ecs.RunTaskInput{
				Cluster:        aws.String("non-existent"),
				TaskDefinition: aws.String("test-task:1"),
			}
			_, err := ecsAPIV2.RunTaskV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster not found"))
		})
	})

	Describe("StopTaskV2", func() {
		var testTask *storage.Task

		BeforeEach(func() {
			// Create a test task
			testTask = &storage.Task{
				ID:                "test-task-id",
				ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:task/test-cluster/test-task-id",
				ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				DesiredStatus:     "RUNNING",
				LastStatus:        "RUNNING",
				LaunchType:        "FARGATE",
				CreatedAt:         time.Now(),
				Version:           1,
				Region:            "ap-northeast-1",
				AccountID:         "123456789012",
			}
			err := testStorage.TaskStore().Create(ctx, testTask)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should stop a task", func() {
			req := &ecs.StopTaskInput{
				Cluster: aws.String("test-cluster"),
				Task:    aws.String(testTask.ARN),
				Reason:  aws.String("Test stop"),
			}
			resp, err := ecsAPIV2.StopTaskV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.Task.TaskArn).To(Equal(testTask.ARN))
			Expect(*resp.Task.DesiredStatus).To(Equal("STOPPED"))
			Expect(*resp.Task.StoppedReason).To(Equal("Test stop"))
			Expect(resp.Task.StoppedAt).ToNot(BeNil())
		})

		It("should fail when task not found", func() {
			req := &ecs.StopTaskInput{
				Cluster: aws.String("test-cluster"),
				Task:    aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task/test-cluster/non-existent"),
			}
			_, err := ecsAPIV2.StopTaskV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("task not found"))
		})
	})

	Describe("DescribeTasksV2", func() {
		var testTasks []*storage.Task

		BeforeEach(func() {
			// Create test tasks
			for i := 1; i <= 3; i++ {
				task := &storage.Task{
					ID:                fmt.Sprintf("task-%d", i),
					ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task/test-cluster/task-%d", i),
					ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
					TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
					DesiredStatus:     "RUNNING",
					LastStatus:        "RUNNING",
					LaunchType:        "FARGATE",
					StartedBy:         "test-user",
					Group:             "test-group",
					CreatedAt:         time.Now(),
					Version:           1,
					Region:            "ap-northeast-1",
					AccountID:         "123456789012",
				}
				
				// Add containers info
				containers := []map[string]interface{}{
					{
						"name":         "app",
						"image":        "nginx:latest",
						"lastStatus":   "RUNNING",
						"containerArn": fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:container/%s", task.ID),
					},
				}
				containersJSON, _ := json.Marshal(containers)
				task.Containers = string(containersJSON)
				
				err := testStorage.TaskStore().Create(ctx, task)
				Expect(err).ToNot(HaveOccurred())
				testTasks = append(testTasks, task)
			}
		})

		It("should describe specific tasks", func() {
			req := &ecs.DescribeTasksInput{
				Cluster: aws.String("test-cluster"),
				Tasks: []string{
					testTasks[0].ARN,
					testTasks[2].ARN,
				},
			}
			resp, err := ecsAPIV2.DescribeTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Tasks).To(HaveLen(2))
			
			taskArns := []string{*resp.Tasks[0].TaskArn, *resp.Tasks[1].TaskArn}
			Expect(taskArns).To(ContainElements(testTasks[0].ARN, testTasks[2].ARN))
			
			for _, task := range resp.Tasks {
				Expect(*task.LastStatus).To(Equal("RUNNING"))
				Expect(*task.StartedBy).To(Equal("test-user"))
				Expect(task.Containers).To(HaveLen(1))
			}
		})

		It("should handle non-existent tasks", func() {
			req := &ecs.DescribeTasksInput{
				Cluster: aws.String("test-cluster"),
				Tasks: []string{
					testTasks[0].ARN,
					"arn:aws:ecs:ap-northeast-1:123456789012:task/test-cluster/non-existent",
				},
			}
			resp, err := ecsAPIV2.DescribeTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Tasks).To(HaveLen(1))
			Expect(resp.Failures).To(HaveLen(1))
			Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
		})
	})

	Describe("ListTasksV2", func() {
		BeforeEach(func() {
			// Create test tasks with different attributes
			tasks := []struct {
				id            string
				serviceName   string
				launchType    string
				desiredStatus string
				startedBy     string
			}{
				{"task-1", "service-a", "FARGATE", "RUNNING", "service-a"},
				{"task-2", "service-a", "EC2", "RUNNING", "service-a"},
				{"task-3", "service-b", "FARGATE", "RUNNING", "service-b"},
				{"task-4", "", "FARGATE", "STOPPED", "user"},
			}

			for _, t := range tasks {
				task := &storage.Task{
					ID:                t.id,
					ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task/test-cluster/%s", t.id),
					ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
					TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
					DesiredStatus:     t.desiredStatus,
					LastStatus:        t.desiredStatus,
					LaunchType:        t.launchType,
					StartedBy:         t.startedBy,
					CreatedAt:         time.Now(),
					Version:           1,
					Region:            "ap-northeast-1",
					AccountID:         "123456789012",
				}
				
				if t.serviceName != "" {
					task.Group = fmt.Sprintf("service:%s", t.serviceName)
					task.StartedBy = fmt.Sprintf("ecs-svc/%s", t.serviceName)
				}
				
				err := testStorage.TaskStore().Create(ctx, task)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should list all tasks in cluster", func() {
			req := &ecs.ListTasksInput{
				Cluster: aws.String("test-cluster"),
			}
			resp, err := ecsAPIV2.ListTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskArns).To(HaveLen(4))
		})

		It("should filter by service name", func() {
			req := &ecs.ListTasksInput{
				Cluster:     aws.String("test-cluster"),
				ServiceName: aws.String("service-a"),
			}
			resp, err := ecsAPIV2.ListTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskArns).To(HaveLen(2))
		})

		It("should filter by launch type", func() {
			req := &ecs.ListTasksInput{
				Cluster:    aws.String("test-cluster"),
				LaunchType: ecstypes.LaunchTypeEc2,
			}
			resp, err := ecsAPIV2.ListTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskArns).To(HaveLen(1))
			Expect(resp.TaskArns[0]).To(ContainSubstring("task-2"))
		})

		It("should filter by desired status", func() {
			req := &ecs.ListTasksInput{
				Cluster:       aws.String("test-cluster"),
				DesiredStatus: ecstypes.DesiredStatusStopped,
			}
			resp, err := ecsAPIV2.ListTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskArns).To(HaveLen(1))
			Expect(resp.TaskArns[0]).To(ContainSubstring("task-4"))
		})

		It("should handle pagination", func() {
			req := &ecs.ListTasksInput{
				Cluster:    aws.String("test-cluster"),
				MaxResults: aws.Int32(2),
			}
			resp, err := ecsAPIV2.ListTasksV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskArns).To(HaveLen(2))
			Expect(resp.NextToken).ToNot(BeNil())
		})
	})
})

func TestTaskECSAPIV2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Task ECS API V2 Suite")
}