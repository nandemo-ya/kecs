package postgres

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

func TestServiceStore_Create(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create a cluster first
	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("create new service", func(t *testing.T) {
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
		require.NoError(t, err)

		// Verify service was created
		retrieved, err := store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
		require.NoError(t, err)
		assert.Equal(t, service.ServiceName, retrieved.ServiceName)
		assert.Equal(t, service.ARN, retrieved.ARN)
		assert.Equal(t, service.DesiredCount, retrieved.DesiredCount)
	})

	t.Run("create duplicate service", func(t *testing.T) {
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
		require.NoError(t, err)

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
		assert.ErrorIs(t, err, storage.ErrResourceAlreadyExists)
	})
}

func TestServiceStore_Get(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("get existing service", func(t *testing.T) {
		service := createTestService(t, store, cluster.ARN, "test-get")

		retrieved, err := store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
		require.NoError(t, err)
		assert.Equal(t, service.ServiceName, retrieved.ServiceName)
		assert.Equal(t, service.ARN, retrieved.ARN)
	})

	t.Run("get non-existent service", func(t *testing.T) {
		_, err := store.ServiceStore().Get(ctx, cluster.ARN, "non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestServiceStore_Update(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("update existing service", func(t *testing.T) {
		service := createTestService(t, store, cluster.ARN, "test-update")

		// Update service
		service.DesiredCount = 5
		service.RunningCount = 4
		service.Status = "UPDATING"

		err := store.ServiceStore().Update(ctx, service)
		require.NoError(t, err)

		// Verify update
		retrieved, err := store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
		require.NoError(t, err)
		assert.Equal(t, int32(5), retrieved.DesiredCount)
		assert.Equal(t, int32(4), retrieved.RunningCount)
		assert.Equal(t, "UPDATING", retrieved.Status)
	})

	t.Run("update non-existent service", func(t *testing.T) {
		service := &storage.Service{
			ID:          uuid.New().String(),
			ARN:         "arn:aws:ecs:us-east-1:000000000000:service/test-cluster/non-existent",
			ServiceName: "non-existent",
			ClusterARN:  cluster.ARN,
		}

		err := store.ServiceStore().Update(ctx, service)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestServiceStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("delete existing service", func(t *testing.T) {
		service := createTestService(t, store, cluster.ARN, "test-delete")

		err := store.ServiceStore().Delete(ctx, cluster.ARN, service.ServiceName)
		require.NoError(t, err)

		// Verify deletion
		_, err = store.ServiceStore().Get(ctx, cluster.ARN, service.ServiceName)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})

	t.Run("delete non-existent service", func(t *testing.T) {
		err := store.ServiceStore().Delete(ctx, cluster.ARN, "non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestServiceStore_List(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	// Create test services
	for i := 0; i < 5; i++ {
		createTestService(t, store, cluster.ARN, fmt.Sprintf("test-list-%d", i))
	}

	t.Run("list all services", func(t *testing.T) {
		services, nextToken, err := store.ServiceStore().List(ctx, cluster.ARN, "", "", 10, "")
		require.NoError(t, err)
		assert.Len(t, services, 5)
		assert.Empty(t, nextToken) // All services fit in one page
	})

	t.Run("list with pagination", func(t *testing.T) {
		// First page
		services1, nextToken1, err := store.ServiceStore().List(ctx, cluster.ARN, "", "", 2, "")
		require.NoError(t, err)
		assert.Len(t, services1, 2)
		assert.NotEmpty(t, nextToken1)

		// Second page
		services2, nextToken2, err := store.ServiceStore().List(ctx, cluster.ARN, "", "", 2, nextToken1)
		require.NoError(t, err)
		assert.Len(t, services2, 2)
		assert.NotEmpty(t, nextToken2)

		// Ensure no duplicates
		names1 := make(map[string]bool)
		for _, s := range services1 {
			names1[s.ServiceName] = true
		}
		for _, s := range services2 {
			assert.False(t, names1[s.ServiceName], "Found duplicate service: %s", s.ServiceName)
		}
	})

	t.Run("list with launch type filter", func(t *testing.T) {
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
		require.NoError(t, err)

		// List only FARGATE services
		services, _, err := store.ServiceStore().List(ctx, cluster.ARN, "", "FARGATE", 10, "")
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, "fargate-service", services[0].ServiceName)
	})
}

// TestServiceStore_UpdateCounts is skipped as UpdateCounts is not in the interface
