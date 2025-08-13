package duckdb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

func TestTaskDefinitionStore_GetLatest(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)

	taskDefStore := store.TaskDefinitionStore()

	// Test case 1: No task definitions exist
	_, err := taskDefStore.GetLatest(ctx, "test-family")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active task definition found")

	// Test case 2: Register multiple revisions
	taskDef1 := &storage.TaskDefinition{
		Family:               "test-family",
		TaskRoleARN:          "arn:aws:iam::123456789012:role/test-role",
		ExecutionRoleARN:     "arn:aws:iam::123456789012:role/test-exec-role",
		NetworkMode:          "bridge",
		ContainerDefinitions: `[{"name":"test","image":"nginx"}]`,
		Region:               "us-east-1",
		AccountID:            "123456789012",
	}

	// Register first revision
	registered1, err := taskDefStore.Register(ctx, taskDef1)
	require.NoError(t, err)
	assert.Equal(t, 1, registered1.Revision)

	// Register second revision
	taskDef2 := &storage.TaskDefinition{
		Family:               "test-family",
		TaskRoleARN:          "arn:aws:iam::123456789012:role/test-role-v2",
		ExecutionRoleARN:     "arn:aws:iam::123456789012:role/test-exec-role",
		NetworkMode:          "awsvpc",
		ContainerDefinitions: `[{"name":"test","image":"nginx:1.19"}]`,
		Region:               "us-east-1",
		AccountID:            "123456789012",
	}
	registered2, err := taskDefStore.Register(ctx, taskDef2)
	require.NoError(t, err)
	assert.Equal(t, 2, registered2.Revision)

	// Test case 3: GetLatest returns the highest revision
	latest, err := taskDefStore.GetLatest(ctx, "test-family")
	require.NoError(t, err)
	assert.Equal(t, 2, latest.Revision)
	assert.Equal(t, "awsvpc", latest.NetworkMode)
	assert.Equal(t, "ACTIVE", latest.Status)

	// Test case 4: Deregister latest revision
	err = taskDefStore.Deregister(ctx, "test-family", 2)
	require.NoError(t, err)

	// GetLatest should now return revision 1
	latest, err = taskDefStore.GetLatest(ctx, "test-family")
	require.NoError(t, err)
	assert.Equal(t, 1, latest.Revision)
	assert.Equal(t, "bridge", latest.NetworkMode)

	// Test case 5: Deregister all revisions
	err = taskDefStore.Deregister(ctx, "test-family", 1)
	require.NoError(t, err)

	// No active revisions left
	_, err = taskDefStore.GetLatest(ctx, "test-family")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active task definition found")

	// Test case 6: Multiple families
	taskDefOther := &storage.TaskDefinition{
		Family:               "other-family",
		NetworkMode:          "host",
		ContainerDefinitions: `[{"name":"other","image":"redis"}]`,
		Region:               "us-east-1",
		AccountID:            "123456789012",
	}
	_, err = taskDefStore.Register(ctx, taskDefOther)
	require.NoError(t, err)

	// GetLatest for other family
	latestOther, err := taskDefStore.GetLatest(ctx, "other-family")
	require.NoError(t, err)
	assert.Equal(t, 1, latestOther.Revision)
	assert.Equal(t, "host", latestOther.NetworkMode)

	// Original family still has no active revisions
	_, err = taskDefStore.GetLatest(ctx, "test-family")
	assert.Error(t, err)
}
