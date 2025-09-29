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

func TestTaskDefinitionStore_Register(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	t.Run("register new task definition", func(t *testing.T) {
		td := &storage.TaskDefinition{
			ID:                   uuid.New().String(),
			Family:               "test",
			TaskRoleARN:          "arn:aws:iam::000000000000:role/task-role",
			ExecutionRoleARN:     "arn:aws:iam::000000000000:role/exec-role",
			NetworkMode:          "bridge",
			ContainerDefinitions: `[{"name":"app","image":"nginx:latest"}]`,
			Status:               "ACTIVE",
			Region:               "us-east-1",
			AccountID:            "000000000000",
			RegisteredAt:         time.Now(),
		}

		registered, err := store.TaskDefinitionStore().Register(ctx, td)
		require.NoError(t, err)
		assert.NotNil(t, registered)
		assert.Equal(t, 1, registered.Revision)
		assert.NotEmpty(t, registered.ARN)

		// Verify task definition was created
		retrieved, err := store.TaskDefinitionStore().Get(ctx, td.Family, 1)
		require.NoError(t, err)
		assert.Equal(t, td.Family, retrieved.Family)
		assert.Equal(t, 1, retrieved.Revision)
	})

	t.Run("register new revision of existing family", func(t *testing.T) {
		// Register first revision
		td1 := &storage.TaskDefinition{
			ID:                   uuid.New().String(),
			Family:               "multi-revision",
			ContainerDefinitions: `[{"name":"app","image":"nginx:1.0"}]`,
			Status:               "ACTIVE",
			Region:               "us-east-1",
			AccountID:            "000000000000",
			RegisteredAt:         time.Now(),
		}
		registered1, err := store.TaskDefinitionStore().Register(ctx, td1)
		require.NoError(t, err)
		assert.Equal(t, 1, registered1.Revision)

		// Register second revision
		td2 := &storage.TaskDefinition{
			ID:                   uuid.New().String(),
			Family:               "multi-revision",
			ContainerDefinitions: `[{"name":"app","image":"nginx:2.0"}]`,
			Status:               "ACTIVE",
			Region:               "us-east-1",
			AccountID:            "000000000000",
			RegisteredAt:         time.Now(),
		}
		registered2, err := store.TaskDefinitionStore().Register(ctx, td2)
		require.NoError(t, err)
		assert.Equal(t, 2, registered2.Revision)
	})
}

func TestTaskDefinitionStore_Get(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Register a task definition
	td := createTestTaskDefinition(t, store, "test-get")

	t.Run("get existing task definition", func(t *testing.T) {
		retrieved, err := store.TaskDefinitionStore().Get(ctx, td.Family, td.Revision)
		require.NoError(t, err)
		assert.Equal(t, td.Family, retrieved.Family)
		assert.Equal(t, td.Revision, retrieved.Revision)
	})

	t.Run("get non-existent task definition", func(t *testing.T) {
		_, err := store.TaskDefinitionStore().Get(ctx, "non-existent", 1)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskDefinitionStore_GetLatest(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Register multiple revisions
	family := "test-latest"
	for i := 0; i < 3; i++ {
		td := &storage.TaskDefinition{
			ID:                   uuid.New().String(),
			Family:               family,
			ContainerDefinitions: fmt.Sprintf(`[{"name":"app","image":"nginx:%d.0"}]`, i+1),
			Status:               "ACTIVE",
			Region:               "us-east-1",
			AccountID:            "000000000000",
			RegisteredAt:         time.Now(),
		}
		_, err := store.TaskDefinitionStore().Register(ctx, td)
		require.NoError(t, err)
	}

	t.Run("get latest revision", func(t *testing.T) {
		latest, err := store.TaskDefinitionStore().GetLatest(ctx, family)
		require.NoError(t, err)
		assert.Equal(t, family, latest.Family)
		assert.Equal(t, 3, latest.Revision)
	})

	t.Run("get latest of non-existent family", func(t *testing.T) {
		_, err := store.TaskDefinitionStore().GetLatest(ctx, "non-existent")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskDefinitionStore_GetByARN(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Register a task definition
	td := createTestTaskDefinition(t, store, "test-arn")

	t.Run("get by ARN", func(t *testing.T) {
		retrieved, err := store.TaskDefinitionStore().GetByARN(ctx, td.ARN)
		require.NoError(t, err)
		assert.Equal(t, td.Family, retrieved.Family)
		assert.Equal(t, td.ARN, retrieved.ARN)
	})

	t.Run("get by non-existent ARN", func(t *testing.T) {
		_, err := store.TaskDefinitionStore().GetByARN(ctx, "arn:aws:ecs:us-east-1:000000000000:task-definition/non-existent:1")
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskDefinitionStore_Deregister(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Register a task definition
	td := createTestTaskDefinition(t, store, "test-deregister")

	t.Run("deregister existing task definition", func(t *testing.T) {
		err := store.TaskDefinitionStore().Deregister(ctx, td.Family, td.Revision)
		require.NoError(t, err)

		// Verify status changed to INACTIVE
		retrieved, err := store.TaskDefinitionStore().Get(ctx, td.Family, td.Revision)
		require.NoError(t, err)
		assert.Equal(t, "INACTIVE", retrieved.Status)
		assert.NotNil(t, retrieved.DeregisteredAt)
	})

	t.Run("deregister non-existent task definition", func(t *testing.T) {
		err := store.TaskDefinitionStore().Deregister(ctx, "non-existent", 1)
		assert.ErrorIs(t, err, storage.ErrResourceNotFound)
	})
}

func TestTaskDefinitionStore_ListFamilies(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create test task definitions with multiple revisions
	families := []string{"app", "api", "worker", "database", "cache"}
	for _, family := range families {
		for i := 0; i < 3; i++ {
			td := &storage.TaskDefinition{
				ID:                   uuid.New().String(),
				Family:               family,
				ContainerDefinitions: fmt.Sprintf(`[{"name":"app","image":"%s:%d.0"}]`, family, i+1),
				Status:               "ACTIVE",
				Region:               "us-east-1",
				AccountID:            "000000000000",
				RegisteredAt:         time.Now(),
			}
			_, err := store.TaskDefinitionStore().Register(ctx, td)
			require.NoError(t, err)
		}
	}

	t.Run("list all families", func(t *testing.T) {
		result, nextToken, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "", 100, "")
		require.NoError(t, err)
		assert.Len(t, result, 5)
		assert.Empty(t, nextToken)

		// Check each family has correct revision counts
		for _, fam := range result {
			assert.Equal(t, 3, fam.LatestRevision)
			assert.Equal(t, 3, fam.ActiveRevisions)
		}
	})

	t.Run("list families with prefix", func(t *testing.T) {
		result, nextToken, err := store.TaskDefinitionStore().ListFamilies(ctx, "a", "", 100, "")
		require.NoError(t, err)
		assert.Len(t, result, 2) // "app" and "api"
		assert.Empty(t, nextToken)
		for _, fam := range result {
			assert.True(t, fam.Family == "app" || fam.Family == "api")
		}
	})

	t.Run("list families with pagination", func(t *testing.T) {
		// First page
		result1, nextToken1, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "", 2, "")
		require.NoError(t, err)
		assert.Len(t, result1, 2)
		assert.NotEmpty(t, nextToken1)

		// Second page
		result2, _, err := store.TaskDefinitionStore().ListFamilies(ctx, "", "", 2, nextToken1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(result2), 1)

		// Ensure no duplicates
		families1 := make(map[string]bool)
		for _, f := range result1 {
			families1[f.Family] = true
		}
		for _, f := range result2 {
			assert.False(t, families1[f.Family], "Found duplicate family: %s", f.Family)
		}
	})
}

func TestTaskDefinitionStore_ListRevisions(t *testing.T) {
	ctx := context.Background()
	store := setupTestDB(t)
	defer store.Close()

	// Create multiple revisions for a family
	family := "test-revisions"
	for i := 0; i < 5; i++ {
		td := &storage.TaskDefinition{
			ID:                   uuid.New().String(),
			Family:               family,
			ContainerDefinitions: fmt.Sprintf(`[{"name":"app","image":"nginx:%d.0"}]`, i+1),
			Status:               "ACTIVE",
			Region:               "us-east-1",
			AccountID:            "000000000000",
			RegisteredAt:         time.Now(),
		}
		_, err := store.TaskDefinitionStore().Register(ctx, td)
		require.NoError(t, err)
	}

	// Deregister one revision
	err := store.TaskDefinitionStore().Deregister(ctx, family, 2)
	require.NoError(t, err)

	t.Run("list all revisions", func(t *testing.T) {
		revisions, nextToken, err := store.TaskDefinitionStore().ListRevisions(ctx, family, "", 100, "")
		require.NoError(t, err)
		assert.Len(t, revisions, 5)
		assert.Empty(t, nextToken)
	})

	t.Run("list active revisions only", func(t *testing.T) {
		revisions, nextToken, err := store.TaskDefinitionStore().ListRevisions(ctx, family, "ACTIVE", 100, "")
		require.NoError(t, err)
		assert.Len(t, revisions, 4) // One was deregistered
		assert.Empty(t, nextToken)
		for _, rev := range revisions {
			assert.Equal(t, "ACTIVE", rev.Status)
		}
	})

	t.Run("list inactive revisions only", func(t *testing.T) {
		revisions, nextToken, err := store.TaskDefinitionStore().ListRevisions(ctx, family, "INACTIVE", 100, "")
		require.NoError(t, err)
		assert.Len(t, revisions, 1) // One was deregistered
		assert.Empty(t, nextToken)
		assert.Equal(t, "INACTIVE", revisions[0].Status)
	})

	t.Run("list with pagination", func(t *testing.T) {
		// First page
		revisions1, nextToken1, err := store.TaskDefinitionStore().ListRevisions(ctx, family, "", 2, "")
		require.NoError(t, err)
		assert.Len(t, revisions1, 2)
		assert.NotEmpty(t, nextToken1)

		// Second page
		revisions2, _, err := store.TaskDefinitionStore().ListRevisions(ctx, family, "", 2, nextToken1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(revisions2), 1)
	})
}

// Helper function to create and register a test task definition
func createTestTaskDefinition(t *testing.T, store storage.Storage, family string) *storage.TaskDefinition {
	td := &storage.TaskDefinition{
		ID:                   uuid.New().String(),
		Family:               family,
		NetworkMode:          "bridge",
		ContainerDefinitions: `[{"name":"app","image":"nginx:latest"}]`,
		Status:               "ACTIVE",
		Region:               "us-east-1",
		AccountID:            "000000000000",
		RegisteredAt:         time.Now(),
	}
	registered, err := store.TaskDefinitionStore().Register(context.Background(), td)
	require.NoError(t, err)
	return registered
}
