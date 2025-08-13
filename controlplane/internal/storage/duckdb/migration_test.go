package duckdb

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration_LocalStackStateColumn(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "kecs-test-migration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	ctx := context.Background()

	t.Run("new_table_has_localstack_state_column", func(t *testing.T) {
		// Create DuckDB storage
		duckDB, err := NewDuckDBStorage(dbPath)
		require.NoError(t, err)
		defer duckDB.Close()

		// Initialize storage
		err = duckDB.Initialize(ctx)
		require.NoError(t, err)

		// Check that localstack_state column exists
		var columnExists bool
		err = duckDB.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'clusters' AND column_name = 'localstack_state'
			)
		`).Scan(&columnExists)
		require.NoError(t, err)
		assert.True(t, columnExists, "localstack_state column should exist in new clusters table")
	})

	t.Run("migration_adds_localstack_state_column", func(t *testing.T) {
		// Create a new database
		tmpDir2, err := os.MkdirTemp("", "kecs-test-migration2-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir2)

		dbPath2 := filepath.Join(tmpDir2, "test2.db")

		// Create old schema without localstack_state
		db, err := sql.Open("duckdb", dbPath2)
		require.NoError(t, err)

		// Create old clusters table without localstack_state
		_, err = db.ExecContext(ctx, `
			CREATE TABLE clusters (
				id VARCHAR PRIMARY KEY,
				arn VARCHAR NOT NULL UNIQUE,
				name VARCHAR NOT NULL UNIQUE,
				status VARCHAR NOT NULL,
				region VARCHAR NOT NULL,
				account_id VARCHAR NOT NULL,
				configuration JSON,
				settings JSON,
				tags JSON,
				k8s_cluster_name VARCHAR,
				registered_container_instances_count INTEGER DEFAULT 0,
				running_tasks_count INTEGER DEFAULT 0,
				pending_tasks_count INTEGER DEFAULT 0,
				active_services_count INTEGER DEFAULT 0,
				capacity_providers JSON,
				default_capacity_provider_strategy JSON,
				created_at TIMESTAMP NOT NULL,
				updated_at TIMESTAMP NOT NULL
			)
		`)
		require.NoError(t, err)

		// Insert test data
		_, err = db.ExecContext(ctx, `
			INSERT INTO clusters (
				id, arn, name, status, region, account_id,
				created_at, updated_at
			) VALUES (
				'test-id', 'arn:aws:ecs:us-east-1:000000000000:cluster/test',
				'test', 'ACTIVE', 'us-east-1', '000000000000',
				CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
			)
		`)
		require.NoError(t, err)

		db.Close()

		// Now open with our storage which should trigger migration
		duckDB, err := NewDuckDBStorage(dbPath2)
		require.NoError(t, err)
		defer duckDB.Close()

		// Initialize storage (this should trigger migration)
		err = duckDB.Initialize(ctx)
		require.NoError(t, err)

		// Check that localstack_state column exists
		var columnExists bool
		err = duckDB.db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'clusters' AND column_name = 'localstack_state'
			)
		`).Scan(&columnExists)
		require.NoError(t, err)
		assert.True(t, columnExists, "localstack_state column should exist after migration")

		// Verify data type is VARCHAR
		var dataType string
		err = duckDB.db.QueryRowContext(ctx, `
			SELECT data_type 
			FROM information_schema.columns 
			WHERE table_name = 'clusters' AND column_name = 'localstack_state'
		`).Scan(&dataType)
		require.NoError(t, err)
		assert.Equal(t, "VARCHAR", dataType)

		// Verify existing data is preserved
		store := duckDB.ClusterStore()
		cluster, err := store.Get(ctx, "test")
		require.NoError(t, err)
		assert.NotNil(t, cluster)
		assert.Equal(t, "test", cluster.Name)
		assert.Equal(t, "ACTIVE", cluster.Status)
		assert.Empty(t, cluster.LocalStackState) // Should be empty for migrated data
	})
}
