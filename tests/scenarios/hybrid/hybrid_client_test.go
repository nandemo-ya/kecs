package hybrid

import (
	"testing"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHybridClientCompatibility verifies that both curl and AWS CLI clients work correctly
func TestHybridClientCompatibility(t *testing.T) {
	utils.TestWithBothClients(t, "ClusterOperations", func(t *testing.T, client utils.ECSClientInterface, mode utils.ClientMode) {
		t.Logf("Testing with client mode: %s", mode)
		
		// Test cluster creation
		clusterName := "test-cluster-hybrid"
		err := client.CreateCluster(clusterName)
		require.NoError(t, err, "Failed to create cluster")
		
		// Verify cluster exists
		cluster, err := client.DescribeCluster(clusterName)
		require.NoError(t, err, "Failed to describe cluster")
		assert.Equal(t, clusterName, cluster.ClusterName)
		assert.Equal(t, "ACTIVE", cluster.Status)
		
		// List clusters
		clusters, err := client.ListClusters()
		require.NoError(t, err, "Failed to list clusters")
		assert.Contains(t, clusters, cluster.ClusterArn)
		
		// Delete cluster
		err = client.DeleteCluster(clusterName)
		require.NoError(t, err, "Failed to delete cluster")
		
		// Verify cluster is deleted
		_, err = client.DescribeCluster(clusterName)
		assert.Error(t, err, "Expected error when describing deleted cluster")
	})
}

// TestTaskDefinitionCompatibility tests task definition operations with both clients
func TestTaskDefinitionCompatibility(t *testing.T) {
	utils.TestWithBothClients(t, "TaskDefinitionOperations", func(t *testing.T, client utils.ECSClientInterface, mode utils.ClientMode) {
		t.Logf("Testing task definitions with client mode: %s", mode)
		
		// Register task definition
		taskDefJSON := `{
			"family": "test-task-hybrid",
			"containerDefinitions": [{
				"name": "test-container",
				"image": "nginx:latest",
				"memory": 512,
				"essential": true
			}]
		}`
		
		taskDef, err := client.RegisterTaskDefinition("test-task-hybrid", taskDefJSON)
		require.NoError(t, err, "Failed to register task definition")
		assert.Equal(t, "test-task-hybrid", taskDef.Family)
		assert.Equal(t, 1, taskDef.Revision)
		
		// Describe task definition
		described, err := client.DescribeTaskDefinition(taskDef.TaskDefinitionArn)
		require.NoError(t, err, "Failed to describe task definition")
		assert.Equal(t, taskDef.TaskDefinitionArn, described.TaskDefinitionArn)
		
		// List task definitions
		taskDefs, err := client.ListTaskDefinitions()
		require.NoError(t, err, "Failed to list task definitions")
		assert.Contains(t, taskDefs, taskDef.TaskDefinitionArn)
		
		// Deregister task definition
		err = client.DeregisterTaskDefinition(taskDef.TaskDefinitionArn)
		require.NoError(t, err, "Failed to deregister task definition")
	})
}

// TestServiceCompatibility tests service operations with both clients
func TestServiceCompatibility(t *testing.T) {
	utils.TestWithBothClients(t, "ServiceOperations", func(t *testing.T, client utils.ECSClientInterface, mode utils.ClientMode) {
		t.Logf("Testing services with client mode: %s", mode)
		
		// Create cluster first
		clusterName := "test-cluster-service"
		err := client.CreateCluster(clusterName)
		require.NoError(t, err, "Failed to create cluster")
		defer client.DeleteCluster(clusterName)
		
		// Register task definition
		taskDefJSON := `{
			"family": "test-service-task",
			"containerDefinitions": [{
				"name": "test-container",
				"image": "nginx:latest",
				"memory": 512,
				"essential": true
			}]
		}`
		
		taskDef, err := client.RegisterTaskDefinition("test-service-task", taskDefJSON)
		require.NoError(t, err, "Failed to register task definition")
		defer client.DeregisterTaskDefinition(taskDef.TaskDefinitionArn)
		
		// Create service
		serviceName := "test-service-hybrid"
		err = client.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 2)
		require.NoError(t, err, "Failed to create service")
		
		// Describe service
		service, err := client.DescribeService(clusterName, serviceName)
		require.NoError(t, err, "Failed to describe service")
		assert.Equal(t, serviceName, service.ServiceName)
		assert.Equal(t, 2, service.DesiredCount)
		
		// Update service
		newCount := 3
		err = client.UpdateService(clusterName, serviceName, &newCount, "")
		require.NoError(t, err, "Failed to update service")
		
		// Verify update
		service, err = client.DescribeService(clusterName, serviceName)
		require.NoError(t, err, "Failed to describe updated service")
		assert.Equal(t, 3, service.DesiredCount)
		
		// Delete service
		err = client.DeleteService(clusterName, serviceName)
		require.NoError(t, err, "Failed to delete service")
	})
}

// TestClientModeSelection tests that the correct client is selected based on mode
func TestClientModeSelection(t *testing.T) {
	// Test default mode (curl)
	client := utils.NewECSClientInterface("http://localhost:8080")
	_, isCurl := client.(*utils.CurlClient)
	assert.True(t, isCurl, "Default client should be CurlClient")
	
	// Test explicit curl mode
	curlClient := utils.NewECSClientInterface("http://localhost:8080", utils.CurlMode)
	_, isCurl = curlClient.(*utils.CurlClient)
	assert.True(t, isCurl, "Explicit curl mode should return CurlClient")
	
	// Test AWS CLI mode
	awsClient := utils.NewECSClientInterface("http://localhost:8080", utils.AWSCLIMode)
	_, isAWSCLI := awsClient.(*utils.AWSCLIClient)
	assert.True(t, isAWSCLI, "AWS CLI mode should return AWSCLIClient")
}