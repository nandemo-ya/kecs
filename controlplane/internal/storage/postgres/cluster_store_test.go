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

func TestClusterStore_Create(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	t.Run("create new cluster", func(t *testing.T) {
		cluster := &storage.Cluster{
			ID:        uuid.New().String(),
			ARN:       "arn:aws:ecs:us-east-1:000000000000:cluster/test-cluster",
			Name:      "test-cluster",
			Status:    "ACTIVE",
			Region:    "us-east-1",
			AccountID: "000000000000",
		}

		err := store.ClusterStore().Create(ctx, cluster)
		require.NoError(t, err)

		// Verify cluster was created
		retrieved, err := store.ClusterStore().Get(ctx, cluster.Name)
		require.NoError(t, err)
		assert.Equal(t, cluster.Name, retrieved.Name)
		assert.Equal(t, cluster.ARN, retrieved.ARN)
		assert.Equal(t, cluster.Status, retrieved.Status)
	})

	t.Run("create duplicate cluster", func(t *testing.T) {
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
		require.NoError(t, err)

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
		assert.ErrorIs(t, err, storage.ErrResourceAlreadyExists)
	})
}

func TestClusterStore_Get(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	t.Run("get existing cluster", func(t *testing.T) {
		cluster := createTestCluster(t, store, "test-get")

		retrieved, err := store.ClusterStore().Get(ctx, cluster.Name)
		require.NoError(t, err)
		assert.Equal(t, cluster.Name, retrieved.Name)
		assert.Equal(t, cluster.ARN, retrieved.ARN)
	})

	t.Run("get non-existent cluster", func(t *testing.T) {
		_, err := store.ClusterStore().Get(ctx, "non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

// TestClusterStore_GetByARN is skipped as GetByARN is not in the interface
// but exists in the implementation for internal use

func TestClusterStore_Update(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	t.Run("update existing cluster", func(t *testing.T) {
		cluster := createTestCluster(t, store, "test-update")

		// Update cluster
		cluster.Status = "PROVISIONING"
		cluster.Tags = `{"Environment": "test"}`

		err := store.ClusterStore().Update(ctx, cluster)
		require.NoError(t, err)

		// Verify update
		retrieved, err := store.ClusterStore().Get(ctx, cluster.Name)
		require.NoError(t, err)
		assert.Equal(t, "PROVISIONING", retrieved.Status)
		assert.Equal(t, `{"Environment": "test"}`, retrieved.Tags)
	})

	t.Run("update non-existent cluster", func(t *testing.T) {
		cluster := &storage.Cluster{
			ID:   uuid.New().String(),
			ARN:  "arn:aws:ecs:us-east-1:000000000000:cluster/non-existent",
			Name: "non-existent",
		}

		err := store.ClusterStore().Update(ctx, cluster)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestClusterStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	t.Run("delete existing cluster", func(t *testing.T) {
		cluster := createTestCluster(t, store, "test-delete")

		err := store.ClusterStore().Delete(ctx, cluster.Name)
		require.NoError(t, err)

		// Verify deletion
		_, err = store.ClusterStore().Get(ctx, cluster.Name)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})

	t.Run("delete non-existent cluster", func(t *testing.T) {
		err := store.ClusterStore().Delete(ctx, "non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestClusterStore_List(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create test clusters
	for i := 0; i < 5; i++ {
		createTestCluster(t, store, fmt.Sprintf("test-list-%d", i))
	}

	t.Run("list all clusters", func(t *testing.T) {
		clusters, err := store.ClusterStore().List(ctx)
		require.NoError(t, err)
		assert.Len(t, clusters, 5)
	})
}

func TestClusterStore_ListWithPagination(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create test clusters
	clusters := make([]*storage.Cluster, 10)
	for i := 0; i < 10; i++ {
		clusters[i] = createTestCluster(t, store, fmt.Sprintf("test-page-%02d", i))
	}

	t.Run("list first page", func(t *testing.T) {
		result, nextToken, err := store.ClusterStore().ListWithPagination(ctx, 3, "")
		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.NotEmpty(t, nextToken)
	})

	t.Run("list second page", func(t *testing.T) {
		// Get first page to get nextToken
		result1, nextToken1, err := store.ClusterStore().ListWithPagination(ctx, 3, "")
		require.NoError(t, err)
		require.NotEmpty(t, nextToken1)

		// Get second page
		result2, nextToken2, err := store.ClusterStore().ListWithPagination(ctx, 3, nextToken1)
		require.NoError(t, err)
		assert.Len(t, result2, 3)
		assert.NotEmpty(t, nextToken2)

		// Ensure no duplicates between pages
		ids1 := make(map[string]bool)
		for _, c := range result1 {
			ids1[c.ID] = true
		}
		for _, c := range result2 {
			assert.False(t, ids1[c.ID], "Found duplicate cluster ID: %s", c.ID)
		}
	})

	t.Run("list last page", func(t *testing.T) {
		// Navigate to last page
		var lastToken string
		pageCount := 0
		for {
			_, nextToken, err := store.ClusterStore().ListWithPagination(ctx, 3, lastToken)
			require.NoError(t, err)
			pageCount++
			if nextToken == "" {
				break
			}
			lastToken = nextToken
		}
		assert.Equal(t, 4, pageCount) // 10 items with page size 3 = 4 pages (3+3+3+1)
	})

	t.Run("invalid next token", func(t *testing.T) {
		_, _, err := store.ClusterStore().ListWithPagination(ctx, 3, "invalid-token")
		assert.Error(t, err)
	})
}

// TestClusterStore_UpdateStatistics and TestClusterStore_DeleteOlderThan
// are skipped as these methods are not in the interface
