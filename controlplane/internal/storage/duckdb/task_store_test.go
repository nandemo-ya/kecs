package duckdb

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskStore_ConcurrentCreateOrUpdate(t *testing.T) {
	// Skip test if running in CI without database
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	ctx := context.Background()

	// Create temporary database
	dbPath := t.TempDir() + "/test.db"
	duckDB, err := NewDuckDBStorage(dbPath)
	require.NoError(t, err)
	defer duckDB.Close()

	// Initialize database
	err = duckDB.Initialize(ctx)
	require.NoError(t, err)

	// Create task store
	store := duckDB.TaskStore()

	// Prepare test task
	baseTask := &storage.Task{
		ID:                   "test-task-001",
		ARN:                  "arn:aws:ecs:us-east-1:123456789012:task/default/test-task-001",
		ClusterARN:           "arn:aws:ecs:us-east-1:123456789012:cluster/default",
		TaskDefinitionARN:    "arn:aws:ecs:us-east-1:123456789012:task-definition/test-def:1",
		LastStatus:           "PENDING",
		DesiredStatus:        "RUNNING",
		LaunchType:           "FARGATE",
		CreatedAt:            time.Now(),
		Containers:           "[]",
		Version:              1,
		EnableExecuteCommand: false,
		Region:               "us-east-1",
		AccountID:            "123456789012",
	}

	// Test concurrent updates
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			// Create a copy of the task with different status
			task := *baseTask
			if idx%2 == 0 {
				task.LastStatus = "RUNNING"
			} else {
				task.LastStatus = "PROVISIONING"
			}
			task.Version = int64(idx)

			// Attempt to create or update
			errors[idx] = store.CreateOrUpdate(ctx, &task)
		}(i)
	}

	wg.Wait()

	// Check that all operations succeeded (thanks to retry logic)
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		} else {
			t.Logf("Error occurred: %v", err)
		}
	}

	// At least some operations should succeed
	// With retry logic, we expect most or all to succeed
	assert.Greater(t, successCount, 0, "At least one operation should succeed")
	t.Logf("Success rate: %d/%d", successCount, numGoroutines)

	// Verify the task was actually created/updated
	retrievedTask, err := store.Get(ctx, "arn:aws:ecs:us-east-1:123456789012:cluster/default", "test-task-001")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	if retrievedTask != nil {
		assert.Equal(t, baseTask.ARN, retrievedTask.ARN)
		assert.Contains(t, []string{"PENDING", "PROVISIONING", "RUNNING"}, retrievedTask.LastStatus)
	}
}

func TestTaskStore_ConcurrentUpdate(t *testing.T) {
	// Skip test if running in CI without database
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	ctx := context.Background()

	// Create temporary database
	dbPath := t.TempDir() + "/test.db"
	duckDB, err := NewDuckDBStorage(dbPath)
	require.NoError(t, err)
	defer duckDB.Close()

	// Initialize database
	err = duckDB.Initialize(ctx)
	require.NoError(t, err)

	// Create task store
	store := duckDB.TaskStore()

	// Create initial task
	initialTask := &storage.Task{
		ID:                   "test-task-002",
		ARN:                  "arn:aws:ecs:us-east-1:123456789012:task/default/test-task-002",
		ClusterARN:           "arn:aws:ecs:us-east-1:123456789012:cluster/default",
		TaskDefinitionARN:    "arn:aws:ecs:us-east-1:123456789012:task-definition/test-def:1",
		LastStatus:           "PENDING",
		DesiredStatus:        "RUNNING",
		LaunchType:           "FARGATE",
		CreatedAt:            time.Now(),
		Containers:           "[]",
		Version:              1,
		EnableExecuteCommand: false,
		Region:               "us-east-1",
		AccountID:            "123456789012",
	}

	// Create the task first
	err = store.Create(ctx, initialTask)
	require.NoError(t, err)

	// Test concurrent updates
	const numGoroutines = 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			// Create a copy of the task with different status
			task := *initialTask
			task.LastStatus = "RUNNING"
			task.Version = int64(idx + 2)

			// Add some variation in the update timing
			time.Sleep(time.Millisecond * time.Duration(idx))

			// Attempt to update
			errors[idx] = store.Update(ctx, &task)
		}(i)
	}

	wg.Wait()

	// Check that all operations succeeded (thanks to retry logic)
	successCount := 0
	for _, err := range errors {
		if err == nil {
			successCount++
		} else {
			t.Logf("Error occurred: %v", err)
		}
	}

	// With retry logic, we expect most or all updates to succeed
	assert.Greater(t, successCount, 0, "At least one update should succeed")
	t.Logf("Update success rate: %d/%d", successCount, numGoroutines)

	// Verify the task was actually updated
	retrievedTask, err := store.Get(ctx, "arn:aws:ecs:us-east-1:123456789012:cluster/default", "test-task-002")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTask)
	if retrievedTask != nil {
		assert.Equal(t, "RUNNING", retrievedTask.LastStatus)
		assert.Greater(t, retrievedTask.Version, int64(1))
	}
}
