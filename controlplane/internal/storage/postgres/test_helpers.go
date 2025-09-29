package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// setupTestDB creates a test database connection for PostgreSQL
// It uses environment variables or defaults for connection parameters
func setupTestDB(t *testing.T) storage.Storage {
	// Skip if not in PostgreSQL test mode
	if os.Getenv("TEST_POSTGRES") != "true" {
		t.Skip("Skipping PostgreSQL tests. Set TEST_POSTGRES=true to run")
	}

	// Get connection parameters from environment or use defaults
	host := getEnvOrDefault("POSTGRES_HOST", "localhost")
	port := getEnvOrDefault("POSTGRES_PORT", "5432")
	user := getEnvOrDefault("POSTGRES_USER", "kecs_test")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "kecs_test")
	dbname := getEnvOrDefault("POSTGRES_DB", "kecs_test")

	// Build connection string
	connString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	// Create storage instance
	store := NewPostgresStorage(connString)

	// Initialize the database
	ctx := context.Background()
	err := store.Initialize(ctx)
	require.NoError(t, err)

	// Clean up all tables before tests
	cleanupTestDB(t, store.db)

	// Reinitialize after cleanup to recreate tables
	err = store.Initialize(ctx)
	require.NoError(t, err)

	return store
}

// cleanupTestDB drops all tables for test cleanup
func cleanupTestDB(t *testing.T, db *sql.DB) {
	tables := []string{
		"task_logs",
		"elbv2_target_groups",
		"elbv2_load_balancers",
		"attributes",
		"container_instances",
		"task_sets",
		"account_settings",
		"tasks",
		"services",
		"task_definitions",
		"clusters",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to drop table %s: %v", table, err)
		}
	}
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createTestCluster creates a test cluster for testing
func createTestCluster(t *testing.T, store storage.Storage, name string) *storage.Cluster {
	cluster := &storage.Cluster{
		ID:        fmt.Sprintf("cluster-%s", name),
		ARN:       fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:cluster/%s", name),
		Name:      name,
		Status:    "ACTIVE",
		Region:    "us-east-1",
		AccountID: "000000000000",
	}

	ctx := context.Background()
	err := store.ClusterStore().Create(ctx, cluster)
	require.NoError(t, err)

	return cluster
}

// createTestService creates a test service for testing
func createTestService(t *testing.T, store storage.Storage, clusterARN, serviceName string) *storage.Service {
	service := &storage.Service{
		ID:           fmt.Sprintf("service-%s", serviceName),
		ARN:          fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:service/%s/%s", clusterARN, serviceName),
		ServiceName:  serviceName,
		ClusterARN:   clusterARN,
		Status:       "ACTIVE",
		DesiredCount: 1,
		RunningCount: 1,
		Region:       "us-east-1",
		AccountID:    "000000000000",
	}

	ctx := context.Background()
	err := store.ServiceStore().Create(ctx, service)
	require.NoError(t, err)

	return service
}
