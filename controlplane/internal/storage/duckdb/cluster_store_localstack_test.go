package duckdb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

func TestClusterStore_LocalStackState(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "kecs-test-localstack-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create DuckDB storage
	dbPath := filepath.Join(tmpDir, "test.db")
	duckDB, err := NewDuckDBStorage(dbPath)
	require.NoError(t, err)
	defer duckDB.Close()

	// Initialize storage
	ctx := context.Background()
	err = duckDB.Initialize(ctx)
	require.NoError(t, err)

	store := duckDB.ClusterStore()

	t.Run("create_cluster_with_localstack_state", func(t *testing.T) {
		// Create LocalStack state
		now := time.Now()
		localStackState := &storage.LocalStackState{
			Deployed:    true,
			Status:      "running",
			Version:     "2.3.0",
			Namespace:   "aws-services",
			PodName:     "localstack-0",
			Endpoint:    "http://localstack.aws-services.svc.cluster.local:4566",
			DeployedAt:  &now,
			HealthStatus: "healthy",
		}

		// Serialize state
		stateJSON, err := storage.SerializeLocalStackState(localStackState)
		require.NoError(t, err)

		// Create cluster with LocalStack state
		cluster := &storage.Cluster{
			ARN:                               "arn:aws:ecs:us-east-1:000000000000:cluster/test-localstack",
			Name:                              "test-localstack",
			Status:                            "ACTIVE",
			Region:                            "us-east-1",
			AccountID:                         "000000000000",
			K8sClusterName:                    "kecs-test-localstack",
			RegisteredContainerInstancesCount: 0,
			RunningTasksCount:                 0,
			PendingTasksCount:                 0,
			ActiveServicesCount:               0,
			LocalStackState:                   stateJSON,
		}

		err = store.Create(ctx, cluster)
		require.NoError(t, err)

		// Retrieve cluster
		retrieved, err := store.Get(ctx, "test-localstack")
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, cluster.Name, retrieved.Name)
		assert.NotEmpty(t, retrieved.LocalStackState)

		// Deserialize and verify LocalStack state
		retrievedState, err := storage.DeserializeLocalStackState(retrieved.LocalStackState)
		require.NoError(t, err)
		assert.NotNil(t, retrievedState)
		assert.True(t, retrievedState.Deployed)
		assert.Equal(t, "running", retrievedState.Status)
		assert.Equal(t, "2.3.0", retrievedState.Version)
		assert.Equal(t, "aws-services", retrievedState.Namespace)
		assert.Equal(t, "localstack-0", retrievedState.PodName)
		assert.Equal(t, "http://localstack.aws-services.svc.cluster.local:4566", retrievedState.Endpoint)
		assert.Equal(t, "healthy", retrievedState.HealthStatus)
	})

	t.Run("update_localstack_state", func(t *testing.T) {
		// Get existing cluster
		cluster, err := store.Get(ctx, "test-localstack")
		require.NoError(t, err)

		// Update LocalStack state to failed
		now := time.Now()
		newState := &storage.LocalStackState{
			Deployed:    true,
			Status:      "failed",
			DeployedAt:  &now,
			HealthStatus: "connection refused",
		}

		// Serialize new state
		stateJSON, err := storage.SerializeLocalStackState(newState)
		require.NoError(t, err)

		// Update cluster
		cluster.LocalStackState = stateJSON
		err = store.Update(ctx, cluster)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := store.Get(ctx, "test-localstack")
		require.NoError(t, err)
		assert.NotNil(t, retrieved)

		// Deserialize and verify updated state
		retrievedState, err := storage.DeserializeLocalStackState(retrieved.LocalStackState)
		require.NoError(t, err)
		assert.NotNil(t, retrievedState)
		assert.True(t, retrievedState.Deployed)
		assert.Equal(t, "failed", retrievedState.Status)
		assert.Equal(t, "connection refused", retrievedState.HealthStatus)
	})

	t.Run("list_clusters_with_localstack_state", func(t *testing.T) {
		// Create another cluster without LocalStack
		cluster2 := &storage.Cluster{
			ARN:                               "arn:aws:ecs:us-east-1:000000000000:cluster/test-no-localstack",
			Name:                              "test-no-localstack",
			Status:                            "ACTIVE",
			Region:                            "us-east-1",
			AccountID:                         "000000000000",
			K8sClusterName:                    "kecs-test-no-localstack",
			RegisteredContainerInstancesCount: 0,
			RunningTasksCount:                 0,
			PendingTasksCount:                 0,
			ActiveServicesCount:               0,
		}

		err = store.Create(ctx, cluster2)
		require.NoError(t, err)

		// List all clusters
		clusters, err := store.List(ctx)
		require.NoError(t, err)
		assert.Len(t, clusters, 2)

		// Check LocalStack states
		localStackCount := 0
		for _, cluster := range clusters {
			if cluster.LocalStackState != "" {
				localStackCount++
				state, err := storage.DeserializeLocalStackState(cluster.LocalStackState)
				require.NoError(t, err)
				assert.NotNil(t, state)
			}
		}
		assert.Equal(t, 1, localStackCount)
	})

	t.Run("empty_localstack_state", func(t *testing.T) {
		// Test deserialization of empty state
		state, err := storage.DeserializeLocalStackState("")
		require.NoError(t, err)
		assert.Nil(t, state)

		// Test serialization of nil state
		stateJSON, err := storage.SerializeLocalStackState(nil)
		require.NoError(t, err)
		assert.Empty(t, stateJSON)
	})
}

func TestLocalStackState_Serialization(t *testing.T) {
	t.Run("full_state", func(t *testing.T) {
		now := time.Now()
		lastCheck := now.Add(5 * time.Minute)
		
		state := &storage.LocalStackState{
			Deployed:         true,
			Status:          "running",
			Version:         "2.3.0",
			Namespace:       "aws-services",
			PodName:         "localstack-0",
			Endpoint:        "http://localstack.aws-services.svc.cluster.local:4566",
			DeployedAt:      &now,
			LastHealthCheck: &lastCheck,
			HealthStatus:    "healthy",
		}

		// Serialize
		jsonStr, err := storage.SerializeLocalStackState(state)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonStr)

		// Verify JSON structure
		var jsonMap map[string]interface{}
		err = json.Unmarshal([]byte(jsonStr), &jsonMap)
		require.NoError(t, err)
		assert.Equal(t, true, jsonMap["deployed"])
		assert.Equal(t, "running", jsonMap["status"])
		assert.Equal(t, "2.3.0", jsonMap["version"])

		// Deserialize
		deserialized, err := storage.DeserializeLocalStackState(jsonStr)
		require.NoError(t, err)
		assert.NotNil(t, deserialized)
		assert.Equal(t, state.Deployed, deserialized.Deployed)
		assert.Equal(t, state.Status, deserialized.Status)
		assert.Equal(t, state.Version, deserialized.Version)
		assert.Equal(t, state.Namespace, deserialized.Namespace)
		assert.Equal(t, state.PodName, deserialized.PodName)
		assert.Equal(t, state.Endpoint, deserialized.Endpoint)
		assert.Equal(t, state.HealthStatus, deserialized.HealthStatus)
		assert.NotNil(t, deserialized.DeployedAt)
		assert.NotNil(t, deserialized.LastHealthCheck)
	})

	t.Run("minimal_state", func(t *testing.T) {
		state := &storage.LocalStackState{
			Deployed: false,
			Status:   "not_deployed",
		}

		// Serialize
		jsonStr, err := storage.SerializeLocalStackState(state)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonStr)

		// Deserialize
		deserialized, err := storage.DeserializeLocalStackState(jsonStr)
		require.NoError(t, err)
		assert.NotNil(t, deserialized)
		assert.False(t, deserialized.Deployed)
		assert.Equal(t, "not_deployed", deserialized.Status)
		assert.Empty(t, deserialized.Version)
		assert.Nil(t, deserialized.DeployedAt)
	})

	t.Run("invalid_json", func(t *testing.T) {
		_, err := storage.DeserializeLocalStackState("invalid json")
		assert.Error(t, err)
	})
}