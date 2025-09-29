package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	postgresStorage "github.com/nandemo-ya/kecs/controlplane/internal/storage/postgres"
)

var (
	postgresContainer *postgres.PostgresContainer
	testDB            storage.Storage
)

// setupPostgresContainer starts a PostgreSQL container for testing
func setupPostgresContainer() {
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("kecs_test"),
		postgres.WithUsername("kecs_test"),
		postgres.WithPassword("kecs_test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		log.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	postgresContainer = container

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to get connection string: %v", err)
	}

	GinkgoWriter.Printf("PostgreSQL container started with connection string: %s\n", connStr)

	// Create storage instance and initialize
	store := postgresStorage.NewPostgresStorage(connStr)
	err = store.Initialize(context.Background())
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	testDB = store
}

// teardownPostgresContainer stops and removes the PostgreSQL container
func teardownPostgresContainer() {
	if postgresContainer != nil {
		ctx := context.Background()
		if err := postgresContainer.Terminate(ctx); err != nil {
			GinkgoWriter.Printf("Failed to terminate PostgreSQL container: %v\n", err)
		}
	}
	if testDB != nil {
		testDB.Close()
	}
}

// setupTestDB returns the shared test database connection
func setupTestDB() storage.Storage {
	if testDB == nil {
		Fail("PostgreSQL container not initialized. Make sure BeforeSuite is called.")
	}

	// Clean up any existing data before each test
	cleanupDatabase()

	return testDB
}

// cleanupDatabase removes all test data
func cleanupDatabase() {
	ctx := context.Background()

	// Get host and port from container
	host, err := postgresContainer.Host(ctx)
	Expect(err).NotTo(HaveOccurred())

	mappedPort, err := postgresContainer.MappedPort(ctx, nat.Port("5432/tcp"))
	Expect(err).NotTo(HaveOccurred())

	// Create direct connection for cleanup
	connStr := fmt.Sprintf("postgres://kecs_test:kecs_test@%s:%s/kecs_test?sslmode=disable",
		host, mappedPort.Port())

	db, err := sql.Open("postgres", connStr)
	Expect(err).NotTo(HaveOccurred())
	defer db.Close()

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
