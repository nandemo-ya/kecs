package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListContainerInstancesPagination(t *testing.T) {
	ctx := context.Background()

	// Setup mock storage
	mockStorage := mocks.NewMockStorage()
	clusterStore := mocks.NewMockClusterStore()
	containerInstanceStore := mocks.NewMockContainerInstanceStore()
	mockStorage.SetClusterStore(clusterStore)
	mockStorage.SetContainerInstanceStore(containerInstanceStore)

	// Create test cluster
	cluster := &storage.Cluster{
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
	require.NoError(t, err)

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
			Region:            "us-east-1",
			AccountID:         "123456789012",
			RegisteredAt:      time.Now(),
			UpdatedAt:         time.Now(),
		}
		err := containerInstanceStore.Register(ctx, instance)
		require.NoError(t, err)
	}

	// Create API instance
	apiInstance := api.NewDefaultECSAPIWithConfig(mockStorage, nil, "us-east-1", "123456789012")

	t.Run("FirstPage", func(t *testing.T) {
		req := &generated.ListContainerInstancesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(5),
		}

		resp, err := apiInstance.ListContainerInstances(ctx, req)
		require.NoError(t, err)
		assert.Len(t, resp.ContainerInstanceArns, 5)
		assert.NotNil(t, resp.NextToken)
	})

	t.Run("SecondPage", func(t *testing.T) {
		// Get first page to get nextToken
		req1 := &generated.ListContainerInstancesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(5),
		}
		resp1, err := apiInstance.ListContainerInstances(ctx, req1)
		require.NoError(t, err)

		// Get second page
		req2 := &generated.ListContainerInstancesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(5),
			NextToken:  resp1.NextToken,
		}
		resp2, err := apiInstance.ListContainerInstances(ctx, req2)
		require.NoError(t, err)
		assert.Len(t, resp2.ContainerInstanceArns, 5)
		assert.NotNil(t, resp2.NextToken)

		// Ensure different results
		assert.NotEqual(t, resp1.ContainerInstanceArns[0], resp2.ContainerInstanceArns[0])
	})

	t.Run("LastPage", func(t *testing.T) {
		// Get to the last page
		req := &generated.ListContainerInstancesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(10),
		}
		resp1, err := apiInstance.ListContainerInstances(ctx, req)
		require.NoError(t, err)

		// Get last page
		req2 := &generated.ListContainerInstancesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(10),
			NextToken:  resp1.NextToken,
		}
		resp2, err := apiInstance.ListContainerInstances(ctx, req2)
		require.NoError(t, err)
		assert.Len(t, resp2.ContainerInstanceArns, 5) // Only 5 remaining
		assert.Nil(t, resp2.NextToken)               // No more pages
	})

	t.Run("StatusFilter", func(t *testing.T) {
		// Add some DRAINING instances
		for i := 15; i < 18; i++ {
			instance := &storage.ContainerInstance{
				ID:                fmt.Sprintf("instance-%02d", i),
				ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:container-instance/test-cluster/i-%02d", i),
				ClusterARN:        cluster.ARN,
				EC2InstanceID:     fmt.Sprintf("i-1234567890abcdef%d", i),
				Status:            "DRAINING",
				AgentConnected:    true,
				RunningTasksCount: 0,
				PendingTasksCount: 0,
				Version:           1,
				Region:            "us-east-1",
				AccountID:         "123456789012",
				RegisteredAt:      time.Now(),
				UpdatedAt:         time.Now(),
			}
			err := containerInstanceStore.Register(ctx, instance)
			require.NoError(t, err)
		}

		// Filter by DRAINING status
		status := generated.ContainerInstanceStatus("DRAINING")
		req := &generated.ListContainerInstancesRequest{
			Cluster:    ptr.String("test-cluster"),
			Status:     &status,
			MaxResults: ptr.Int32(10),
		}

		resp, err := apiInstance.ListContainerInstances(ctx, req)
		require.NoError(t, err)
		assert.Len(t, resp.ContainerInstanceArns, 3)
		assert.Nil(t, resp.NextToken) // All fit in one page
	})

	t.Run("InvalidCluster", func(t *testing.T) {
		req := &generated.ListContainerInstancesRequest{
			Cluster: ptr.String("non-existent-cluster"),
		}

		resp, err := apiInstance.ListContainerInstances(ctx, req)
		require.NoError(t, err)
		assert.Empty(t, resp.ContainerInstanceArns)
		assert.Nil(t, resp.NextToken)
	})
}

func TestListAttributesPagination(t *testing.T) {
	ctx := context.Background()

	// Setup mock storage
	mockStorage := mocks.NewMockStorage()
	attributeStore := mocks.NewMockAttributeStore()
	mockStorage.SetAttributeStore(attributeStore)

	// Create test attributes
	for i := 0; i < 15; i++ {
		attr := &storage.Attribute{
			ID:         fmt.Sprintf("attr-%02d", i),
			Name:       fmt.Sprintf("attribute-%02d", i),
			Value:      fmt.Sprintf("value-%02d", i),
			TargetType: "container-instance",
			TargetID:   fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:container-instance/test-cluster/i-%02d", i),
			Cluster:    "test-cluster",
			Region:     "us-east-1",
			AccountID:  "123456789012",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		err := attributeStore.Put(ctx, []*storage.Attribute{attr})
		require.NoError(t, err)
	}

	// Create API instance
	apiInstance := api.NewDefaultECSAPIWithConfig(mockStorage, nil, "us-east-1", "123456789012")

	t.Run("FirstPage", func(t *testing.T) {
		req := &generated.ListAttributesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(5),
		}

		resp, err := apiInstance.ListAttributes(ctx, req)
		require.NoError(t, err)
		assert.Len(t, resp.Attributes, 5)
		assert.NotNil(t, resp.NextToken)
	})

	t.Run("SecondPage", func(t *testing.T) {
		// Get first page to get nextToken
		req1 := &generated.ListAttributesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(5),
		}
		resp1, err := apiInstance.ListAttributes(ctx, req1)
		require.NoError(t, err)

		// Get second page
		req2 := &generated.ListAttributesRequest{
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(5),
			NextToken:  resp1.NextToken,
		}
		resp2, err := apiInstance.ListAttributes(ctx, req2)
		require.NoError(t, err)
		assert.Len(t, resp2.Attributes, 5)
		assert.NotNil(t, resp2.NextToken)

		// Ensure different results
		assert.NotEqual(t, *resp1.Attributes[0].Name, *resp2.Attributes[0].Name)
	})

	t.Run("TargetTypeFilter", func(t *testing.T) {
		// Add some attributes with different target type
		for i := 15; i < 18; i++ {
			attr := &storage.Attribute{
				ID:         fmt.Sprintf("task-attr-%02d", i),
				Name:       fmt.Sprintf("task-attribute-%02d", i),
				Value:      fmt.Sprintf("task-value-%02d", i),
				TargetType: "task",
				TargetID:   fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task/test-cluster/task-%02d", i),
				Cluster:    "test-cluster",
				Region:     "us-east-1",
				AccountID:  "123456789012",
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			err := attributeStore.Put(ctx, []*storage.Attribute{attr})
			require.NoError(t, err)
		}

		// Filter by task target type
		targetType := generated.TargetType("task")
		req := &generated.ListAttributesRequest{
			TargetType: &targetType,
			Cluster:    ptr.String("test-cluster"),
			MaxResults: ptr.Int32(10),
		}

		resp, err := apiInstance.ListAttributes(ctx, req)
		require.NoError(t, err)
		assert.Len(t, resp.Attributes, 3)
		assert.Nil(t, resp.NextToken) // All fit in one page
		
		// Verify all are task type
		for _, attr := range resp.Attributes {
			assert.Equal(t, "task", string(*attr.TargetType))
		}
	})
}