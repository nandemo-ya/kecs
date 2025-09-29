package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

func TestTaskStore_Create(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create a cluster first
	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("create new task", func(t *testing.T) {
		task := &storage.Task{
			ID:                uuid.New().String(),
			ARN:               "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/test-task",
			ClusterARN:        cluster.ARN,
			TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
			DesiredStatus:     "RUNNING",
			LastStatus:        "PENDING",
			LaunchType:        "EC2",
			StartedBy:         "user",
			Group:             "service:test-service",
			Region:            "us-east-1",
			AccountID:         "000000000000",
			CreatedAt:         time.Now(),
		}

		err := store.TaskStore().Create(ctx, task)
		require.NoError(t, err)

		// Verify task was created
		retrieved, err := store.TaskStore().Get(ctx, cluster.ARN, task.ARN)
		require.NoError(t, err)
		assert.Equal(t, task.ARN, retrieved.ARN)
		assert.Equal(t, task.DesiredStatus, retrieved.DesiredStatus)
		assert.Equal(t, task.LaunchType, retrieved.LaunchType)
	})

	t.Run("create duplicate task", func(t *testing.T) {
		task := &storage.Task{
			ID:         uuid.New().String(),
			ARN:        "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/duplicate",
			ClusterARN: cluster.ARN,
			Region:     "us-east-1",
			AccountID:  "000000000000",
			CreatedAt:  time.Now(),
		}

		// Create first task
		err := store.TaskStore().Create(ctx, task)
		require.NoError(t, err)

		// Try to create duplicate
		task2 := &storage.Task{
			ID:         uuid.New().String(),
			ARN:        task.ARN, // Same ARN
			ClusterARN: cluster.ARN,
			Region:     "us-east-1",
			AccountID:  "000000000000",
			CreatedAt:  time.Now(),
		}

		err = store.TaskStore().Create(ctx, task2)
		assert.ErrorIs(t, err, storage.ErrResourceAlreadyExists)
	})
}

func TestTaskStore_Get(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("get existing task", func(t *testing.T) {
		task := createTestTask(t, store, cluster.ARN, "test-get")

		retrieved, err := store.TaskStore().Get(ctx, cluster.ARN, task.ARN)
		require.NoError(t, err)
		assert.Equal(t, task.ARN, retrieved.ARN)
		assert.Equal(t, task.ClusterARN, retrieved.ClusterARN)
	})

	t.Run("get non-existent task", func(t *testing.T) {
		_, err := store.TaskStore().Get(ctx, cluster.ARN, "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskStore_Update(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("update existing task", func(t *testing.T) {
		task := createTestTask(t, store, cluster.ARN, "test-update")

		// Update task
		task.LastStatus = "RUNNING"
		task.DesiredStatus = "RUNNING"
		now := time.Now()
		task.StartedAt = &now

		err := store.TaskStore().Update(ctx, task)
		require.NoError(t, err)

		// Verify update
		retrieved, err := store.TaskStore().Get(ctx, cluster.ARN, task.ARN)
		require.NoError(t, err)
		assert.Equal(t, "RUNNING", retrieved.LastStatus)
		assert.Equal(t, "RUNNING", retrieved.DesiredStatus)
		assert.False(t, retrieved.StartedAt.IsZero())
	})

	t.Run("update non-existent task", func(t *testing.T) {
		task := &storage.Task{
			ID:         uuid.New().String(),
			ARN:        "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/non-existent",
			ClusterARN: cluster.ARN,
		}

		err := store.TaskStore().Update(ctx, task)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	t.Run("delete existing task", func(t *testing.T) {
		task := createTestTask(t, store, cluster.ARN, "test-delete")

		err := store.TaskStore().Delete(ctx, cluster.ARN, task.ARN)
		require.NoError(t, err)

		// Verify deletion
		_, err = store.TaskStore().Get(ctx, cluster.ARN, task.ARN)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})

	t.Run("delete non-existent task", func(t *testing.T) {
		err := store.TaskStore().Delete(ctx, cluster.ARN, "arn:aws:ecs:us-east-1:000000000000:task/test-cluster/non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskStore_List(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	cluster := createTestCluster(t, store, "test-cluster")

	// Create test tasks
	for i := 0; i < 5; i++ {
		createTestTask(t, store, cluster.ARN, fmt.Sprintf("test-list-%d", i))
	}

	t.Run("list all tasks", func(t *testing.T) {
		tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{})
		require.NoError(t, err)
		assert.Len(t, tasks, 5)
	})

	// Note: TaskStore.List doesn't support pagination in current interface
	// This test is commented out until pagination is added to the interface
	/*
		t.Run("list with pagination", func(t *testing.T) {
			// Would test pagination here if interface supported it
		})
	*/

	t.Run("list with status filter", func(t *testing.T) {
		// Create a STOPPED task
		now := time.Now()
		stoppedTask := &storage.Task{
			ID:            uuid.New().String(),
			ARN:           fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/test-cluster/stopped-task"),
			ClusterARN:    cluster.ARN,
			LastStatus:    "STOPPED",
			DesiredStatus: "STOPPED",
			StoppedAt:     &now,
			Region:        "us-east-1",
			AccountID:     "000000000000",
			CreatedAt:     time.Now(),
		}
		err := store.TaskStore().Create(ctx, stoppedTask)
		require.NoError(t, err)

		// List only STOPPED tasks
		tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{
			DesiredStatus: "STOPPED",
		})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "STOPPED", tasks[0].DesiredStatus)
	})

	t.Run("list with service name filter", func(t *testing.T) {
		// Create a task for a specific service
		serviceTask := &storage.Task{
			ID:         uuid.New().String(),
			ARN:        fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/test-cluster/service-task"),
			ClusterARN: cluster.ARN,
			Group:      "service:my-service",
			Region:     "us-east-1",
			AccountID:  "000000000000",
			CreatedAt:  time.Now(),
		}
		err := store.TaskStore().Create(ctx, serviceTask)
		require.NoError(t, err)

		// List tasks for specific service
		tasks, err := store.TaskStore().List(ctx, cluster.ARN, storage.TaskFilters{
			ServiceName: "my-service",
		})
		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "service:my-service", tasks[0].Group)
	})
}

// Note: ListByService is not in the TaskStore interface
// This test function is commented out until the method is added to the interface
/*
func TestTaskStore_ListByService(t *testing.T) {
	// Would test ListByService here if interface had this method
}
*/

// Helper function to create a test task
func createTestTask(t *testing.T, store storage.Storage, clusterARN, taskID string) *storage.Task {
	task := &storage.Task{
		ID:                uuid.New().String(),
		ARN:               fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:task/test-cluster/%s", taskID),
		ClusterARN:        clusterARN,
		TaskDefinitionARN: "arn:aws:ecs:us-east-1:000000000000:task-definition/test:1",
		DesiredStatus:     "RUNNING",
		LastStatus:        "PENDING",
		LaunchType:        "EC2",
		Region:            "us-east-1",
		AccountID:         "000000000000",
		CreatedAt:         time.Now(),
	}
	err := store.TaskStore().Create(context.Background(), task)
	require.NoError(t, err)
	return task
}
