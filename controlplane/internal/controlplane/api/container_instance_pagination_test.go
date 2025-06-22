package api_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ContainerInstance Pagination", func() {
	var (
		ctx                    context.Context
		mockStorage            *mocks.MockStorage
		clusterStore           *mocks.MockClusterStore
		containerInstanceStore *mocks.MockContainerInstanceStore
		attributeStore         *mocks.MockAttributeStore
		ecsAPI                 generated.ECSAPIInterface
		cluster                *storage.Cluster
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Setup mock storage
		mockStorage = mocks.NewMockStorage()
		clusterStore = mocks.NewMockClusterStore()
		containerInstanceStore = mocks.NewMockContainerInstanceStore()
		attributeStore = mocks.NewMockAttributeStore()
		mockStorage.SetClusterStore(clusterStore)
		mockStorage.SetContainerInstanceStore(containerInstanceStore)
		mockStorage.SetAttributeStore(attributeStore)

		// Create test cluster
		cluster = &storage.Cluster{
			ID:        uuid.New().String(),
			ARN:       "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster",
			Name:      "test-cluster",
			Status:    "ACTIVE",
			Region:    "us-east-1",
			AccountID: "123456789012",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		err := clusterStore.Create(ctx, cluster)
		Expect(err).NotTo(HaveOccurred())

		// Create test container instances
		for i := 0; i < 15; i++ {
			instance := &storage.ContainerInstance{
				ID:                fmt.Sprintf("instance-%02d", i),
				ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:container-instance/test-cluster/i-%02d", i),
				ClusterARN:        cluster.ARN,
				EC2InstanceID:     fmt.Sprintf("i-1234567890abcdef%d", i),
				Status:            "ACTIVE",
				AgentConnected:    true,
				RunningTasksCount: 0,
				PendingTasksCount: 0,
				Version:           1,
				RegisteredAt:      time.Now(),
				UpdatedAt:         time.Now(),
			}
			if i%3 == 0 {
				instance.Status = "DRAINING"
			}
			err := containerInstanceStore.Register(ctx, instance)
			Expect(err).NotTo(HaveOccurred())
		}

		// Create ECS API instance
		ecsAPI = api.NewDefaultECSAPIWithConfig(mockStorage, nil, "us-east-1", "123456789012")
	})

	Describe("ListContainerInstances", func() {
		Context("with pagination", func() {
			It("should return first page of results", func() {
				req := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("test-cluster"),
					MaxResults: ptr.Int32(5),
				}

				resp, err := ecsAPI.ListContainerInstances(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ContainerInstanceArns).To(HaveLen(5))
				Expect(resp.NextToken).NotTo(BeNil())
			})

			It("should return second page using next token", func() {
				// First page
				req1 := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("test-cluster"),
					MaxResults: ptr.Int32(5),
				}
				resp1, err := ecsAPI.ListContainerInstances(ctx, req1)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp1.NextToken).NotTo(BeNil())

				// Second page
				req2 := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("test-cluster"),
					MaxResults: ptr.Int32(5),
					NextToken:  resp1.NextToken,
				}
				resp2, err := ecsAPI.ListContainerInstances(ctx, req2)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp2).NotTo(BeNil())
				Expect(resp2.ContainerInstanceArns).To(HaveLen(5))
				Expect(resp2.NextToken).NotTo(BeNil())

				// Ensure different results
				Expect(resp2.ContainerInstanceArns[0]).NotTo(Equal(resp1.ContainerInstanceArns[0]))
			})

			It("should return last page without next token", func() {
				// Navigate to last page
				req := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("test-cluster"),
					MaxResults: ptr.Int32(10),
				}
				resp, err := ecsAPI.ListContainerInstances(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Last page
				req2 := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("test-cluster"),
					MaxResults: ptr.Int32(10),
					NextToken:  resp.NextToken,
				}
				resp2, err := ecsAPI.ListContainerInstances(ctx, req2)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp2).NotTo(BeNil())
				Expect(resp2.ContainerInstanceArns).To(HaveLen(5)) // Remaining instances
				Expect(resp2.NextToken).To(BeNil())
			})

			It("should filter by status", func() {
				status := generated.ContainerInstanceStatusDRAINING
				req := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("test-cluster"),
					Status:     &status,
					MaxResults: ptr.Int32(10),
				}

				resp, err := ecsAPI.ListContainerInstances(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ContainerInstanceArns).To(HaveLen(5)) // 15 instances, every 3rd is DRAINING
			})

			It("should return empty list for non-existent cluster", func() {
				req := &generated.ListContainerInstancesRequest{
					Cluster:    ptr.String("non-existent-cluster"),
					MaxResults: ptr.Int32(10),
				}

				resp, err := ecsAPI.ListContainerInstances(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.ContainerInstanceArns).To(BeEmpty())
				Expect(resp.NextToken).To(BeNil())
			})
		})
	})

	Describe("ListAttributes", func() {
		BeforeEach(func() {
			// Create test attributes
			for i := 0; i < 12; i++ {
				attr := &storage.Attribute{
					ID:         fmt.Sprintf("attr-%02d", i),
					Name:       fmt.Sprintf("attribute-%02d", i),
					Value:      fmt.Sprintf("value-%02d", i),
					TargetType: "CONTAINER_INSTANCE",
					TargetID:   fmt.Sprintf("instance-%02d", i%5),
					Cluster:    cluster.Name,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				}
				err := attributeStore.Put(ctx, []*storage.Attribute{attr})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		Context("with pagination", func() {
			It("should return first page of results", func() {
				targetType := generated.TargetTypeCONTAINER_INSTANCE
				req := &generated.ListAttributesRequest{
					Cluster:    ptr.String("test-cluster"),
					TargetType: targetType,
					MaxResults: ptr.Int32(5),
				}

				resp, err := ecsAPI.ListAttributes(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Attributes).To(HaveLen(5))
				Expect(resp.NextToken).NotTo(BeNil())
			})

			It("should return second page using next token", func() {
				targetType := generated.TargetTypeCONTAINER_INSTANCE

				// First page
				req1 := &generated.ListAttributesRequest{
					Cluster:    ptr.String("test-cluster"),
					TargetType: targetType,
					MaxResults: ptr.Int32(5),
				}
				resp1, err := ecsAPI.ListAttributes(ctx, req1)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp1.NextToken).NotTo(BeNil())

				// Second page
				req2 := &generated.ListAttributesRequest{
					Cluster:    ptr.String("test-cluster"),
					TargetType: targetType,
					MaxResults: ptr.Int32(5),
					NextToken:  resp1.NextToken,
				}
				resp2, err := ecsAPI.ListAttributes(ctx, req2)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp2).NotTo(BeNil())
				Expect(resp2.Attributes).To(HaveLen(5))
				Expect(resp2.NextToken).NotTo(BeNil())

				// Ensure different results
				Expect(resp2.Attributes[0].Name).NotTo(Equal(resp1.Attributes[0].Name))
			})

			It("should filter by target type", func() {
				targetType := generated.TargetTypeCONTAINER_INSTANCE
				req := &generated.ListAttributesRequest{
					TargetType: targetType,
					MaxResults: ptr.Int32(20),
				}

				resp, err := ecsAPI.ListAttributes(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Attributes).To(HaveLen(12))
				for _, attr := range resp.Attributes {
					Expect(*attr.TargetType).To(Equal(targetType))
				}
			})
		})
	})
})
