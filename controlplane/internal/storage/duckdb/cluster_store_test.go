package duckdb

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

func TestClusterStore_ListWithPagination(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create test clusters
	clusters := make([]*storage.Cluster, 10)
	for i := 0; i < 10; i++ {
		clusters[i] = &storage.Cluster{
			ID:        uuid.New().String(),
			ARN:       fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-%02d", i),
			Name:      fmt.Sprintf("test-cluster-%02d", i),
			Status:    "ACTIVE",
			Region:    "us-east-1",
			AccountID: "123456789012",
		}
		err := store.ClusterStore().Create(ctx, clusters[i])
		require.NoError(t, err)
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
		// Invalid token should start from beginning
		result, nextToken, err := store.ClusterStore().ListWithPagination(ctx, 3, "invalid-token")
		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.NotEmpty(t, nextToken)
	})

	t.Run("large page size", func(t *testing.T) {
		result, nextToken, err := store.ClusterStore().ListWithPagination(ctx, 100, "")
		require.NoError(t, err)
		assert.Len(t, result, 10)  // Should return all 10 items
		assert.Empty(t, nextToken) // No more pages
	})

	t.Run("consistent ordering", func(t *testing.T) {
		// Get all items in one page
		allItems, _, err := store.ClusterStore().ListWithPagination(ctx, 100, "")
		require.NoError(t, err)

		// Get items page by page
		var pagedItems []*storage.Cluster
		var nextToken string
		for {
			page, newToken, err := store.ClusterStore().ListWithPagination(ctx, 3, nextToken)
			require.NoError(t, err)
			pagedItems = append(pagedItems, page...)
			if newToken == "" {
				break
			}
			nextToken = newToken
		}

		// Should have same items in same order
		require.Equal(t, len(allItems), len(pagedItems))
		for i := range allItems {
			assert.Equal(t, allItems[i].ID, pagedItems[i].ID)
		}
	})
}

func setupTestDB(t *testing.T) storage.Storage {
	store, err := NewDuckDBStorage(":memory:")
	require.NoError(t, err)

	// Initialize the database schema
	ctx := context.Background()
	err = store.Initialize(ctx)
	require.NoError(t, err)

	return store
}
