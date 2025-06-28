// cluster_persistence_test.go
// This test verifies that KECS properly recovers k3d clusters after restart.

package phase1

import (
	"os"
	"testing"
	"time"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClusterPersistenceAfterRestart verifies that ECS clusters and their
// corresponding k3d clusters are properly recovered after KECS restart
func TestClusterPersistenceAfterRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cluster persistence test in short mode")
	}

	// Ensure we're not in test mode
	os.Unsetenv("KECS_TEST_MODE")

	// Start KECS container with persistent data directory
	kecs := utils.StartKECSWithPersistence(t)
	defer kecs.Cleanup()

	// Create ECS client
	client := utils.NewECSClientInterface(kecs.Endpoint())

	// Test data
	clusterName := "persistence-test-cluster"

	// Step 1: Create a cluster
	t.Log("Creating ECS cluster...")
	err := client.CreateCluster(clusterName)
	require.NoError(t, err, "Failed to create cluster")

	// Wait for cluster to be active
	utils.AssertClusterActive(t, client, clusterName)

	// Verify k3d cluster exists
	k3dName := "kecs-" + clusterName
	k3dExists, err := utils.K3dClusterExists(k3dName)
	require.NoError(t, err, "Failed to check k3d cluster")
	assert.True(t, k3dExists, "k3d cluster should exist after creation")

	// Step 2: Stop KECS
	t.Log("Stopping KECS...")
	err = kecs.Stop()
	require.NoError(t, err, "Failed to stop KECS container")

	// Wait a bit for container to fully stop
	time.Sleep(2 * time.Second)

	// Step 3: Start KECS again with the same data directory
	t.Log("Restarting KECS...")
	kecs2 := utils.RestartKECSWithPersistence(t, kecs.DataDir)
	defer kecs2.Cleanup()

	// Create new client with new endpoint
	client2 := utils.NewECSClientInterface(kecs2.Endpoint())

	// Step 4: Verify cluster is recovered
	t.Log("Verifying cluster recovery...")
	clusters, err := client2.ListClusters()
	require.NoError(t, err, "Failed to list clusters after restart")
	assert.Contains(t, clusters, clusterName, "Cluster should be recovered from storage")

	// Step 5: Verify k3d cluster is recreated
	t.Log("Verifying k3d cluster recreation...")
	// Give some time for the recovery process to complete
	time.Sleep(10 * time.Second)

	k3dExists, err = utils.K3dClusterExists(k3dName)
	require.NoError(t, err, "Failed to check k3d cluster after restart")
	assert.True(t, k3dExists, "k3d cluster should be recreated after KECS restart")

	// Step 6: Verify cluster is functional
	t.Log("Verifying cluster functionality...")
	cluster, err := client2.DescribeCluster(clusterName)
	require.NoError(t, err, "Failed to describe cluster after restart")
	assert.Equal(t, "ACTIVE", cluster.Status, "Cluster should be active after restart")

	// Clean up
	err = client2.DeleteCluster(clusterName)
	assert.NoError(t, err, "Failed to delete cluster during cleanup")
}

// TestMultipleClusterPersistence tests recovery of multiple clusters
func TestMultipleClusterPersistence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multiple cluster persistence test in short mode")
	}

	// Ensure we're not in test mode
	os.Unsetenv("KECS_TEST_MODE")

	// Start KECS
	kecs := utils.StartKECSWithPersistence(t)
	defer kecs.Cleanup()

	client := utils.NewECSClientInterface(kecs.Endpoint())

	// Create multiple clusters
	clusterNames := []string{
		"persistence-cluster-1",
		"persistence-cluster-2",
		"persistence-cluster-3",
	}

	t.Log("Creating multiple clusters...")
	for _, name := range clusterNames {
		err := client.CreateCluster(name)
		require.NoError(t, err, "Failed to create cluster %s", name)
		utils.AssertClusterActive(t, client, name)
	}

	// Verify all k3d clusters exist
	for _, name := range clusterNames {
		k3dName := "kecs-" + name
		exists, err := utils.K3dClusterExists(k3dName)
		require.NoError(t, err)
		assert.True(t, exists, "k3d cluster %s should exist", k3dName)
	}

	// Restart KECS
	t.Log("Restarting KECS...")
	err := kecs.Stop()
	require.NoError(t, err)
	time.Sleep(2 * time.Second)

	kecs2 := utils.RestartKECSWithPersistence(t, kecs.DataDir)
	defer kecs2.Cleanup()

	client2 := utils.NewECSClientInterface(kecs2.Endpoint())

	// Wait for recovery
	time.Sleep(15 * time.Second)

	// Verify all clusters are recovered
	t.Log("Verifying all clusters are recovered...")
	clusters, err := client2.ListClusters()
	require.NoError(t, err)

	for _, name := range clusterNames {
		assert.Contains(t, clusters, name, "Cluster %s should be recovered", name)

		// Verify k3d cluster is recreated
		k3dName := "kecs-" + name
		exists, err := utils.K3dClusterExists(k3dName)
		require.NoError(t, err)
		assert.True(t, exists, "k3d cluster %s should be recreated", k3dName)
	}

	// Clean up
	for _, name := range clusterNames {
		err := client2.DeleteCluster(name)
		assert.NoError(t, err, "Failed to delete cluster %s", name)
	}
}