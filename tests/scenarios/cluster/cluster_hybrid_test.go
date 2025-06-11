package cluster_test

import (
	"testing"
	"time"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClusterLifecycleHybrid tests the complete cluster lifecycle with both curl and AWS CLI
func TestClusterLifecycleHybrid(t *testing.T) {
	utils.TestWithBothClients(t, "ClusterLifecycle", func(t *testing.T, client utils.ECSClientInterface, mode utils.ClientMode) {
		t.Logf("Testing cluster lifecycle with %s client", mode)
		
		// Test data
		clusterName := "test-cluster-lifecycle"
		
		// Step 1: Create cluster
		t.Log("Creating cluster...")
		err := client.CreateCluster(clusterName)
		require.NoError(t, err, "Failed to create cluster")
		
		// Step 2: Verify cluster is created and active
		t.Log("Verifying cluster status...")
		cluster, err := client.DescribeCluster(clusterName)
		require.NoError(t, err, "Failed to describe cluster")
		assert.Equal(t, clusterName, cluster.ClusterName, "Cluster name mismatch")
		assert.Equal(t, "ACTIVE", cluster.Status, "Cluster should be active immediately")
		assert.Equal(t, 0, cluster.RegisteredContainerInstancesCount, "Should have no container instances")
		assert.Equal(t, 0, cluster.RunningTasksCount, "Should have no running tasks")
		assert.Equal(t, 0, cluster.PendingTasksCount, "Should have no pending tasks")
		assert.Equal(t, 0, cluster.ActiveServicesCount, "Should have no active services")
		
		// Step 3: List clusters and verify our cluster is included
		t.Log("Listing clusters...")
		clusters, err := client.ListClusters()
		require.NoError(t, err, "Failed to list clusters")
		assert.Contains(t, clusters, cluster.ClusterArn, "Cluster should be in the list")
		
		// Step 4: Create the same cluster again (idempotency test)
		t.Log("Testing idempotency...")
		err = client.CreateCluster(clusterName)
		require.NoError(t, err, "Creating existing cluster should not fail")
		
		// Verify it's still the same cluster
		cluster2, err := client.DescribeCluster(clusterName)
		require.NoError(t, err, "Failed to describe cluster after idempotent create")
		assert.Equal(t, cluster.ClusterArn, cluster2.ClusterArn, "Cluster ARN should remain the same")
		
		// Step 5: Delete cluster
		t.Log("Deleting cluster...")
		err = client.DeleteCluster(clusterName)
		require.NoError(t, err, "Failed to delete cluster")
		
		// Step 6: Verify cluster is deleted
		t.Log("Verifying cluster deletion...")
		// Give it a moment to process deletion
		time.Sleep(100 * time.Millisecond)
		
		_, err = client.DescribeCluster(clusterName)
		assert.Error(t, err, "Describing deleted cluster should return an error")
		
		// Step 7: List clusters and verify our cluster is not included
		clusters, err = client.ListClusters()
		require.NoError(t, err, "Failed to list clusters after deletion")
		assert.NotContains(t, clusters, cluster.ClusterArn, "Deleted cluster should not be in the list")
	})
}

// TestMultipleClusterManagementHybrid tests managing multiple clusters simultaneously
func TestMultipleClusterManagementHybrid(t *testing.T) {
	utils.TestWithBothClients(t, "MultipleClusterManagement", func(t *testing.T, client utils.ECSClientInterface, mode utils.ClientMode) {
		t.Logf("Testing multiple cluster management with %s client", mode)
		
		// Create multiple clusters
		clusterNames := []string{
			"test-cluster-1",
			"test-cluster-2",
			"test-cluster-3",
		}
		
		// Create all clusters
		for _, name := range clusterNames {
			err := client.CreateCluster(name)
			require.NoError(t, err, "Failed to create cluster %s", name)
		}
		
		// List and verify all clusters exist
		clusters, err := client.ListClusters()
		require.NoError(t, err, "Failed to list clusters")
		
		// Verify each cluster exists in the list
		for _, name := range clusterNames {
			found := false
			for _, arn := range clusters {
				if contains(arn, name) {
					found = true
					break
				}
			}
			assert.True(t, found, "Cluster %s should be in the list", name)
		}
		
		// Verify each cluster individually
		for _, name := range clusterNames {
			cluster, err := client.DescribeCluster(name)
			require.NoError(t, err, "Failed to describe cluster %s", name)
			assert.Equal(t, name, cluster.ClusterName)
			assert.Equal(t, "ACTIVE", cluster.Status)
		}
		
		// Delete all clusters
		for _, name := range clusterNames {
			err := client.DeleteCluster(name)
			require.NoError(t, err, "Failed to delete cluster %s", name)
		}
		
		// Verify all clusters are deleted
		time.Sleep(100 * time.Millisecond)
		for _, name := range clusterNames {
			_, err := client.DescribeCluster(name)
			assert.Error(t, err, "Cluster %s should be deleted", name)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || (len(s) > len(substr) && s[len(s)-len(substr):] == substr) || (len(s) > len(substr) && s[:len(substr)] == substr) || (len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}