package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/postgres"
)

// setupTestDB creates a test database connection
func setupTestDB() storage.Storage {
	if os.Getenv("TEST_POSTGRES") != "true" {
		Skip("Skipping PostgreSQL tests. Set TEST_POSTGRES=true to run")
	}

	// Get database configuration from environment
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "kecs_test"
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "kecs_test"
	}

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "kecs_test"
	}

	// Create connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbName)

	// Create storage instance
	store := postgres.NewPostgresStorage(connStr)

	// Initialize storage (opens connection and creates tables)
	err := store.Initialize(context.Background())
	Expect(err).NotTo(HaveOccurred())

	// Get database connection for cleanup
	// Note: We need to add a getter for this in the actual implementation
	// For now, we'll open a separate connection for cleanup
	db, err := sql.Open("postgres", connStr)
	Expect(err).NotTo(HaveOccurred())
	defer db.Close()

	// Clean up any existing data
	cleanupDatabase(db)

	return store
}

// cleanupDatabase removes all test data
func cleanupDatabase(db *sql.DB) {
	ctx := context.Background()

	// Tables to clean in order (respecting foreign key constraints)
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
		_, err := db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			// Table might not exist yet, ignore error
			GinkgoWriter.Printf("Warning: Could not clean table %s: %v\n", table, err)
		}
	}
}

// createTestCluster creates a test cluster
func createTestCluster(store storage.Storage, name string) *storage.Cluster {
	cluster := &storage.Cluster{
		ID:        uuid.New().String(),
		ARN:       fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:cluster/%s", name),
		Name:      name,
		Status:    "ACTIVE",
		Region:    "us-east-1",
		AccountID: "000000000000",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := store.ClusterStore().Create(context.Background(), cluster)
	Expect(err).NotTo(HaveOccurred())
	return cluster
}

// createTestService creates a test service
func createTestService(store storage.Storage, clusterARN, serviceName string) *storage.Service {
	service := &storage.Service{
		ID:           uuid.New().String(),
		ARN:          fmt.Sprintf("arn:aws:ecs:us-east-1:000000000000:service/%s/%s", clusterARN, serviceName),
		ServiceName:  serviceName,
		ClusterARN:   clusterARN,
		DesiredCount: 1,
		RunningCount: 0,
		PendingCount: 0,
		Status:       "ACTIVE",
		LaunchType:   "EC2",
		Region:       "us-east-1",
		AccountID:    "000000000000",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := store.ServiceStore().Create(context.Background(), service)
	Expect(err).NotTo(HaveOccurred())
	return service
}

// createTestTask creates a test task
func createTestTask(store storage.Storage, clusterARN, taskID string) *storage.Task {
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
	Expect(err).NotTo(HaveOccurred())
	return task
}

// createTestTaskDefinition creates and registers a test task definition
func createTestTaskDefinition(store storage.Storage, family string) *storage.TaskDefinition {
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
	Expect(err).NotTo(HaveOccurred())
	return registered
}
