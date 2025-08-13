package api

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("TaskSetEcsApi", func() {
	var (
		ctx              context.Context
		ecsAPI           generated.ECSAPIInterface
		mockStorage      *mocks.MockStorage
		mockTaskSetStore *mocks.MockTaskSetStore
		mockServiceStore *mocks.MockServiceStore
		region           string
		accountID        string
		clusterName      string
		serviceName      string
		serviceARN       string
		clusterARN       string
	)

	BeforeEach(func() {
		ctx = context.Background()
		region = "us-east-1"
		accountID = "000000000000"
		clusterName = "test-cluster"
		serviceName = "test-service"
		clusterARN = fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, accountID, clusterName)
		serviceARN = fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s", region, accountID, clusterName, serviceName)

		mockStorage = mocks.NewMockStorage()
		mockTaskSetStore = mocks.NewMockTaskSetStore()
		mockServiceStore = mocks.NewMockServiceStore()

		mockStorage.SetTaskSetStore(mockTaskSetStore)
		mockStorage.SetServiceStore(mockServiceStore)

		ecsAPI = NewDefaultECSAPI(mockStorage)
		// Set region and accountID on the underlying DefaultECSAPI
		if defaultAPI, ok := ecsAPI.(*DefaultECSAPI); ok {
			defaultAPI.region = region
			defaultAPI.accountID = accountID
		}
	})

	Describe("CreateTaskSet", func() {
		var req *generated.CreateTaskSetRequest

		BeforeEach(func() {
			req = &generated.CreateTaskSetRequest{
				Cluster:        clusterName,
				Service:        serviceName,
				TaskDefinition: "arn:aws:ecs:us-east-1:000000000000:task-definition/my-app:1",
				LaunchType:     (*generated.LaunchType)(ptr.String("EC2")),
				Scale: &generated.Scale{
					Value: ptr.Float64(100.0),
					Unit:  (*generated.ScaleUnit)(ptr.String("PERCENT")),
				},
				Tags: []generated.Tag{
					{Key: ptr.String("Environment"), Value: ptr.String("test")},
				},
			}

			// Mock service exists
			err := mockServiceStore.Create(ctx, &storage.Service{
				ARN:         serviceARN,
				ServiceName: serviceName,
				ClusterARN:  clusterARN,
			})
			Expect(err).To(BeNil())
		})

		It("should create a task set successfully", func() {
			resp, err := ecsAPI.CreateTaskSet(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskSet).ToNot(BeNil())
			Expect(*resp.TaskSet.Status).To(Equal("ACTIVE"))
			Expect(*resp.TaskSet.StabilityStatus).To(Equal(generated.StabilityStatus("STEADY_STATE")))
			Expect(resp.TaskSet.Scale).ToNot(BeNil())
			Expect(*resp.TaskSet.Scale.Value).To(Equal(100.0))
			Expect(resp.TaskSet.Tags).To(HaveLen(1))

			// Verify storage was called - check that task set was created
			Expect(len(mockTaskSetStore.GetTaskSets())).To(Equal(1))
		})

		It("should return error when service is missing", func() {
			req.Service = ""
			resp, err := ecsAPI.CreateTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service name is required"))
			Expect(resp).To(BeNil())
		})

		It("should return error when service not found", func() {
			// Clear services to simulate not found - no services created
			mockServiceStore = mocks.NewMockServiceStore()
			mockStorage.SetServiceStore(mockServiceStore)

			resp, err := ecsAPI.CreateTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service not found"))
			Expect(resp).To(BeNil())
		})

		It("should use default scale when not provided", func() {
			req.Scale = nil
			resp, err := ecsAPI.CreateTaskSet(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp.TaskSet.Scale).ToNot(BeNil())
			Expect(*resp.TaskSet.Scale.Value).To(Equal(100.0))
			Expect(*resp.TaskSet.Scale.Unit).To(Equal(generated.ScaleUnit("PERCENT")))
		})
	})

	Describe("DeleteTaskSet", func() {
		var req *generated.DeleteTaskSetRequest
		var taskSetID string

		BeforeEach(func() {
			taskSetID = "ts-12345678"
			req = &generated.DeleteTaskSetRequest{
				Cluster: clusterName,
				Service: serviceName,
				TaskSet: taskSetID,
			}

			// Mock task set exists
			err := mockTaskSetStore.Create(ctx, &storage.TaskSet{
				ID:         taskSetID,
				ARN:        fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", region, accountID, clusterName, serviceName, taskSetID),
				ServiceARN: serviceARN,
				ClusterARN: clusterARN,
				Status:     "ACTIVE",
			})
			Expect(err).To(BeNil())
		})

		It("should delete a task set successfully", func() {
			resp, err := ecsAPI.DeleteTaskSet(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskSet).ToNot(BeNil())
			Expect(*resp.TaskSet.Status).To(Equal("DRAINING"))
			Expect(*resp.TaskSet.Id).To(Equal(taskSetID))

			// Verify status was updated to DRAINING
			taskSet, err := mockTaskSetStore.Get(ctx, serviceARN, taskSetID)
			Expect(err).To(BeNil())
			Expect(taskSet.Status).To(Equal("DRAINING"))
		})

		It("should return error when service or taskSet is missing", func() {
			req.Service = ""
			resp, err := ecsAPI.DeleteTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service and taskSet are required"))
			Expect(resp).To(BeNil())

			req.Service = serviceName
			req.TaskSet = ""
			resp, err = ecsAPI.DeleteTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service and taskSet are required"))
			Expect(resp).To(BeNil())
		})

		It("should return error when task set not found", func() {
			// Clear task sets to simulate not found - no task sets created
			mockTaskSetStore = mocks.NewMockTaskSetStore()
			mockStorage.SetTaskSetStore(mockTaskSetStore)

			resp, err := ecsAPI.DeleteTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("task set not found"))
			Expect(resp).To(BeNil())
		})
	})

	Describe("DescribeTaskSets", func() {
		var req *generated.DescribeTaskSetsRequest

		BeforeEach(func() {
			req = &generated.DescribeTaskSetsRequest{
				Cluster: clusterName,
				Service: serviceName,
			}

			// Mock task sets
			err := mockTaskSetStore.Create(ctx, &storage.TaskSet{
				ID:                   "ts-12345678",
				ARN:                  fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/ts-12345678", region, accountID, clusterName, serviceName),
				ServiceARN:           serviceARN,
				ClusterARN:           clusterARN,
				Status:               "ACTIVE",
				TaskDefinition:       "arn:aws:ecs:us-east-1:000000000000:task-definition/my-app:1",
				LaunchType:           "EC2",
				StabilityStatus:      "STEADY_STATE",
				ComputedDesiredCount: 3,
				RunningCount:         3,
				PendingCount:         0,
				Scale:                `{"value":100.0,"unit":"PERCENT"}`,
				Tags:                 `[{"key":"Environment","value":"test"}]`,
			})
			Expect(err).To(BeNil())

			err = mockTaskSetStore.Create(ctx, &storage.TaskSet{
				ID:                   "ts-87654321",
				ARN:                  fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/ts-87654321", region, accountID, clusterName, serviceName),
				ServiceARN:           serviceARN,
				ClusterARN:           clusterARN,
				Status:               "DRAINING",
				TaskDefinition:       "arn:aws:ecs:us-east-1:000000000000:task-definition/my-app:2",
				LaunchType:           "FARGATE",
				StabilityStatus:      "STABILIZING",
				ComputedDesiredCount: 0,
				RunningCount:         1,
				PendingCount:         0,
				Scale:                `{"value":0.0,"unit":"PERCENT"}`,
			})
			Expect(err).To(BeNil())
		})

		It("should describe all task sets for a service", func() {
			resp, err := ecsAPI.DescribeTaskSets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskSets).To(HaveLen(2))
			Expect(resp.Failures).To(BeEmpty())

			// Create a map for easier lookup
			taskSetMap := make(map[string]generated.TaskSet)
			for _, ts := range resp.TaskSets {
				taskSetMap[*ts.Id] = ts
			}

			// Verify task set "ts-12345678"
			ts1, exists := taskSetMap["ts-12345678"]
			Expect(exists).To(BeTrue())
			Expect(*ts1.Status).To(Equal("ACTIVE"))
			Expect(*ts1.StabilityStatus).To(Equal(generated.StabilityStatus("STEADY_STATE")))
			Expect(*ts1.ComputedDesiredCount).To(Equal(int32(3)))
			Expect(ts1.Scale).ToNot(BeNil())
			Expect(*ts1.Scale.Value).To(Equal(100.0))
			Expect(ts1.Tags).To(HaveLen(1))

			// Verify task set "ts-87654321"
			ts2, exists := taskSetMap["ts-87654321"]
			Expect(exists).To(BeTrue())
			Expect(*ts2.Status).To(Equal("DRAINING"))
			Expect(*ts2.StabilityStatus).To(Equal(generated.StabilityStatus("STABILIZING")))
			Expect(*ts2.ComputedDesiredCount).To(Equal(int32(0)))
		})

		It("should describe specific task sets", func() {
			req.TaskSets = []string{"ts-12345678"}
			resp, err := ecsAPI.DescribeTaskSets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskSets).To(HaveLen(1))
			Expect(*resp.TaskSets[0].Id).To(Equal("ts-12345678"))
		})

		It("should return failures for missing task sets", func() {
			req.TaskSets = []string{"ts-12345678", "ts-missing"}
			resp, err := ecsAPI.DescribeTaskSets(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskSets).To(HaveLen(1))
			Expect(resp.Failures).To(HaveLen(1))
			Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
		})

		It("should return error when service is missing", func() {
			req.Service = ""
			resp, err := ecsAPI.DescribeTaskSets(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service is required"))
			Expect(resp).To(BeNil())
		})
	})

	Describe("UpdateTaskSet", func() {
		var req *generated.UpdateTaskSetRequest
		var taskSetID string

		BeforeEach(func() {
			taskSetID = "ts-12345678"
			req = &generated.UpdateTaskSetRequest{
				Cluster: clusterName,
				Service: serviceName,
				TaskSet: taskSetID,
				Scale: generated.Scale{
					Value: ptr.Float64(50.0),
					Unit:  (*generated.ScaleUnit)(ptr.String("PERCENT")),
				},
			}

			// Mock task set exists
			err := mockTaskSetStore.Create(ctx, &storage.TaskSet{
				ID:                   taskSetID,
				ARN:                  fmt.Sprintf("arn:aws:ecs:%s:%s:task-set/%s/%s/%s", region, accountID, clusterName, serviceName, taskSetID),
				ServiceARN:           serviceARN,
				ClusterARN:           clusterARN,
				Status:               "ACTIVE",
				TaskDefinition:       "arn:aws:ecs:us-east-1:000000000000:task-definition/my-app:1",
				LaunchType:           "EC2",
				StabilityStatus:      "STEADY_STATE",
				ComputedDesiredCount: 3,
				RunningCount:         3,
				PendingCount:         0,
			})
			Expect(err).To(BeNil())
		})

		It("should update a task set scale successfully", func() {
			resp, err := ecsAPI.UpdateTaskSet(ctx, req)
			Expect(err).To(BeNil())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskSet).ToNot(BeNil())
			Expect(*resp.TaskSet.Id).To(Equal(taskSetID))
			Expect(*resp.TaskSet.StabilityStatus).To(Equal(generated.StabilityStatus("STABILIZING")))
			Expect(resp.TaskSet.Scale).ToNot(BeNil())
			Expect(*resp.TaskSet.Scale.Value).To(Equal(50.0))

			// Verify scale was updated
			taskSet, err := mockTaskSetStore.Get(ctx, serviceARN, taskSetID)
			Expect(err).To(BeNil())
			var scale generated.Scale
			err = json.Unmarshal([]byte(taskSet.Scale), &scale)
			Expect(err).To(BeNil())
			Expect(*scale.Value).To(Equal(50.0))
			Expect(taskSet.StabilityStatus).To(Equal("STABILIZING"))
		})

		It("should return error when required fields are missing", func() {
			req.Service = ""
			resp, err := ecsAPI.UpdateTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service and taskSet are required"))
			Expect(resp).To(BeNil())

			req.Service = serviceName
			req.TaskSet = ""
			resp, err = ecsAPI.UpdateTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("service and taskSet are required"))
			Expect(resp).To(BeNil())
		})

		It("should return error when task set not found", func() {
			// Clear task sets to simulate not found - no task sets created
			mockTaskSetStore = mocks.NewMockTaskSetStore()
			mockStorage.SetTaskSetStore(mockTaskSetStore)

			resp, err := ecsAPI.UpdateTaskSet(ctx, req)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("task set not found"))
			Expect(resp).To(BeNil())
		})
	})
})
