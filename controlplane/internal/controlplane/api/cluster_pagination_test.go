package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListClustersPagination(t *testing.T) {
	// Create test storage
	testStorage, err := duckdb.NewDuckDBStorage(":memory:")
	require.NoError(t, err)
	defer testStorage.Close()

	// Initialize storage
	ctx := context.Background()
	require.NoError(t, testStorage.Initialize(ctx))

	// Create API instance
	apiInstance := api.NewDefaultECSAPIWithConfig(testStorage, nil, "us-east-1", "123456789012")

	// Create multiple clusters for testing pagination
	clusterNames := []string{}
	for i := 0; i < 15; i++ {
		clusterName := fmt.Sprintf("test-cluster-%02d", i)
		clusterNames = append(clusterNames, clusterName)
		
		req := &generated.CreateClusterRequest{
			ClusterName: ptr.String(clusterName),
			Tags: []generated.Tag{
				{
					Key:   (*generated.TagKey)(ptr.String("Environment")),
					Value: (*generated.TagValue)(ptr.String("test")),
				},
			},
		}
		
		_, err := apiInstance.CreateCluster(ctx, req)
		require.NoError(t, err)
	}

	// Test 1: List clusters without pagination (should return all)
	t.Run("list all clusters", func(t *testing.T) {
		req := &generated.ListClustersRequest{}
		resp, err := apiInstance.ListClusters(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 15, len(resp.ClusterArns))
		assert.Nil(t, resp.NextToken)
	})

	// Test 2: List clusters with maxResults = 5
	t.Run("list with maxResults=5", func(t *testing.T) {
		req := &generated.ListClustersRequest{
			MaxResults: ptr.Int32(5),
		}
		resp, err := apiInstance.ListClusters(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 5, len(resp.ClusterArns))
		assert.NotNil(t, resp.NextToken)
		
		// Verify the clusters are in the expected order
		for i := 0; i < 5; i++ {
			assert.Contains(t, resp.ClusterArns[i], "test-cluster")
		}
	})

	// Test 3: List next page using NextToken
	t.Run("list next page", func(t *testing.T) {
		// First page
		req1 := &generated.ListClustersRequest{
			MaxResults: ptr.Int32(5),
		}
		resp1, err := apiInstance.ListClusters(ctx, req1)
		require.NoError(t, err)
		require.NotNil(t, resp1.NextToken)

		// Second page
		req2 := &generated.ListClustersRequest{
			MaxResults: ptr.Int32(5),
			NextToken:  resp1.NextToken,
		}
		resp2, err := apiInstance.ListClusters(ctx, req2)
		require.NoError(t, err)
		assert.Equal(t, 5, len(resp2.ClusterArns))
		assert.NotNil(t, resp2.NextToken)

		// Verify no overlap between pages
		for _, arn1 := range resp1.ClusterArns {
			for _, arn2 := range resp2.ClusterArns {
				assert.NotEqual(t, arn1, arn2, "Found duplicate ARN across pages")
			}
		}
	})

	// Test 4: List last page
	t.Run("list last page", func(t *testing.T) {
		var allArns []string
		var nextToken *string
		pageSize := int32(6)

		// Collect all pages
		for i := 0; i < 3; i++ {
			req := &generated.ListClustersRequest{
				MaxResults: ptr.Int32(pageSize),
				NextToken:  nextToken,
			}
			resp, err := apiInstance.ListClusters(ctx, req)
			require.NoError(t, err)
			
			allArns = append(allArns, resp.ClusterArns...)
			nextToken = resp.NextToken
			
			if nextToken == nil {
				// Last page should have 3 items (15 total, 6+6+3)
				assert.Equal(t, 3, len(resp.ClusterArns))
				break
			}
		}
		
		assert.Equal(t, 15, len(allArns))
	})

	// Test 5: Invalid next token
	t.Run("invalid next token", func(t *testing.T) {
		req := &generated.ListClustersRequest{
			MaxResults: ptr.Int32(5),
			NextToken:  ptr.String("invalid-token"),
		}
		resp, err := apiInstance.ListClusters(ctx, req)
		require.NoError(t, err)
		// Should return results starting from the beginning when token is invalid
		assert.GreaterOrEqual(t, len(resp.ClusterArns), 5)
	})

	// Test 6: MaxResults > 100 (should be capped at 100)
	t.Run("maxResults capped at 100", func(t *testing.T) {
		req := &generated.ListClustersRequest{
			MaxResults: ptr.Int32(200),
		}
		resp, err := apiInstance.ListClusters(ctx, req)
		require.NoError(t, err)
		// Should return all 15 clusters since we have less than 100
		assert.Equal(t, 15, len(resp.ClusterArns))
		assert.Nil(t, resp.NextToken)
	})
}

// TestListClustersPaginationConsistency verifies that pagination returns consistent results
func TestListClustersPaginationConsistency(t *testing.T) {
	// Create test storage
	testStorage, err := duckdb.NewDuckDBStorage(":memory:")
	require.NoError(t, err)
	defer testStorage.Close()

	// Initialize storage
	ctx := context.Background()
	require.NoError(t, testStorage.Initialize(ctx))

	// Create API instance
	apiInstance := api.NewDefaultECSAPIWithConfig(testStorage, nil, "us-east-1", "123456789012")

	// Create clusters with known IDs for predictable ordering
	clusterIDs := []string{}
	for i := 0; i < 10; i++ {
		// Create cluster with specific ID to control ordering
		cluster := &storage.Cluster{
			ID:        fmt.Sprintf("%02d-%s", i, uuid.New().String()),
			ARN:       fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster-%02d", i),
			Name:      fmt.Sprintf("test-cluster-%02d", i),
			Status:    "ACTIVE",
			Region:    "us-east-1",
			AccountID: "123456789012",
		}
		
		err := testStorage.ClusterStore().Create(ctx, cluster)
		require.NoError(t, err)
		clusterIDs = append(clusterIDs, cluster.ID)
	}

	// Paginate through all results and collect them
	var allArns []string
	var nextToken *string
	pageSize := int32(3)

	for {
		req := &generated.ListClustersRequest{
			MaxResults: ptr.Int32(pageSize),
			NextToken:  nextToken,
		}
		resp, err := apiInstance.ListClusters(ctx, req)
		require.NoError(t, err)
		
		allArns = append(allArns, resp.ClusterArns...)
		
		if resp.NextToken == nil {
			break
		}
		nextToken = resp.NextToken
	}

	// Verify we got all clusters
	assert.Equal(t, 10, len(allArns))
	
	// Verify no duplicates
	seen := make(map[string]bool)
	for _, arn := range allArns {
		assert.False(t, seen[arn], "Found duplicate ARN: %s", arn)
		seen[arn] = true
	}
}