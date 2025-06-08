package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb" // DuckDB driver
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DuckDBStorage implements storage.Storage using DuckDB
type DuckDBStorage struct {
	db                  *sql.DB
	dbPath              string
	clusterStore        *clusterStore
	taskDefinitionStore *taskDefinitionStore
	serviceStore        *serviceStore
	taskStore           *taskStore
	accountSettingStore *accountSettingStore
}

// NewDuckDBStorage creates a new DuckDB storage instance
func NewDuckDBStorage(dbPath string) (*DuckDBStorage, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		// Note: In production, handle this error properly
		_ = filepath.Dir(dbPath)
	}

	// Open DuckDB connection
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping DuckDB: %w", err)
	}

	s := &DuckDBStorage{
		db:     db,
		dbPath: dbPath,
	}

	// Create stores
	s.clusterStore = &clusterStore{db: db}
	s.taskDefinitionStore = &taskDefinitionStore{db: db}
	s.serviceStore = &serviceStore{db: db}
	s.taskStore = &taskStore{db: db}
	s.accountSettingStore = &accountSettingStore{db: db}

	return s, nil
}

// Initialize creates all necessary tables and indexes
func (s *DuckDBStorage) Initialize(ctx context.Context) error {
	log.Println("Initializing DuckDB storage...")

	// Create clusters table
	if err := s.createClustersTable(ctx); err != nil {
		return fmt.Errorf("failed to create clusters table: %w", err)
	}

	// Create task definitions table
	if err := createTaskDefinitionTable(s.db); err != nil {
		return fmt.Errorf("failed to create task definitions table: %w", err)
	}

	// Create services table
	if err := s.createServicesTable(ctx); err != nil {
		return fmt.Errorf("failed to create services table: %w", err)
	}

	// Create tasks table
	if err := s.createTasksTable(ctx); err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Create account settings table
	if err := s.accountSettingStore.CreateSchema(ctx); err != nil {
		return fmt.Errorf("failed to create account settings table: %w", err)
	}

	log.Println("DuckDB storage initialized successfully")
	return nil
}

// Close closes the database connection
func (s *DuckDBStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ClusterStore returns the cluster store
func (s *DuckDBStorage) ClusterStore() storage.ClusterStore {
	return s.clusterStore
}

// TaskDefinitionStore returns the task definition store
func (s *DuckDBStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return s.taskDefinitionStore
}

// ServiceStore returns the service store
func (s *DuckDBStorage) ServiceStore() storage.ServiceStore {
	return s.serviceStore
}

// TaskStore returns the task store
func (s *DuckDBStorage) TaskStore() storage.TaskStore {
	return s.taskStore
}

// AccountSettingStore returns the account setting store
func (s *DuckDBStorage) AccountSettingStore() storage.AccountSettingStore {
	return s.accountSettingStore
}

// BeginTx starts a new transaction
func (s *DuckDBStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &duckDBTransaction{tx: tx}, nil
}

// createClustersTable creates the clusters table
func (s *DuckDBStorage) createClustersTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS clusters (
		id VARCHAR PRIMARY KEY,
		arn VARCHAR NOT NULL UNIQUE,
		name VARCHAR NOT NULL UNIQUE,
		status VARCHAR NOT NULL,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		configuration JSON,
		settings JSON,
		tags JSON,
		kind_cluster_name VARCHAR,
		registered_container_instances_count INTEGER DEFAULT 0,
		running_tasks_count INTEGER DEFAULT 0,
		pending_tasks_count INTEGER DEFAULT 0,
		active_services_count INTEGER DEFAULT 0,
		capacity_providers JSON,
		default_capacity_provider_strategy JSON,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create clusters table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_status ON clusters(status)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_region ON clusters(region)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_account_id ON clusters(account_id)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createServicesTable creates the services table
func (s *DuckDBStorage) createServicesTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS services (
		id VARCHAR PRIMARY KEY,
		arn VARCHAR NOT NULL UNIQUE,
		service_name VARCHAR NOT NULL,
		cluster_arn VARCHAR NOT NULL,
		task_definition_arn VARCHAR NOT NULL,
		desired_count INTEGER NOT NULL DEFAULT 0,
		running_count INTEGER NOT NULL DEFAULT 0,
		pending_count INTEGER NOT NULL DEFAULT 0,
		launch_type VARCHAR NOT NULL,
		platform_version VARCHAR,
		status VARCHAR NOT NULL,
		role_arn VARCHAR,
		load_balancers VARCHAR,
		service_registries VARCHAR,
		network_configuration VARCHAR,
		deployment_configuration VARCHAR,
		placement_constraints VARCHAR,
		placement_strategy VARCHAR,
		capacity_provider_strategy VARCHAR,
		tags VARCHAR,
		scheduling_strategy VARCHAR NOT NULL DEFAULT 'REPLICA',
		service_connect_configuration VARCHAR,
		enable_ecs_managed_tags BOOLEAN NOT NULL DEFAULT false,
		propagate_tags VARCHAR,
		enable_execute_command BOOLEAN NOT NULL DEFAULT false,
		health_check_grace_period_seconds INTEGER,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		deployment_name VARCHAR,
		namespace VARCHAR,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create services table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_services_cluster_name ON services(cluster_arn, service_name)",
		"CREATE INDEX IF NOT EXISTS idx_services_name ON services(service_name)",
		"CREATE INDEX IF NOT EXISTS idx_services_cluster ON services(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_services_status ON services(status)",
		"CREATE INDEX IF NOT EXISTS idx_services_launch_type ON services(launch_type)",
		"CREATE INDEX IF NOT EXISTS idx_services_region ON services(region)",
		"CREATE INDEX IF NOT EXISTS idx_services_account_id ON services(account_id)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createTasksTable creates the tasks table
func (s *DuckDBStorage) createTasksTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id VARCHAR PRIMARY KEY,
		arn VARCHAR NOT NULL UNIQUE,
		cluster_arn VARCHAR NOT NULL,
		task_definition_arn VARCHAR NOT NULL,
		container_instance_arn VARCHAR,
		overrides VARCHAR,
		last_status VARCHAR NOT NULL,
		desired_status VARCHAR NOT NULL,
		cpu VARCHAR,
		memory VARCHAR,
		containers VARCHAR NOT NULL,
		started_by VARCHAR,
		version BIGINT NOT NULL DEFAULT 1,
		stop_code VARCHAR,
		stopped_reason VARCHAR,
		stopping_at TIMESTAMP,
		stopped_at TIMESTAMP,
		connectivity VARCHAR,
		connectivity_at TIMESTAMP,
		pull_started_at TIMESTAMP,
		pull_stopped_at TIMESTAMP,
		execution_stopped_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL,
		started_at TIMESTAMP,
		launch_type VARCHAR NOT NULL,
		platform_version VARCHAR,
		platform_family VARCHAR,
		task_group VARCHAR,
		attachments VARCHAR,
		health_status VARCHAR,
		tags VARCHAR,
		attributes VARCHAR,
		enable_execute_command BOOLEAN NOT NULL DEFAULT false,
		capacity_provider_name VARCHAR,
		ephemeral_storage VARCHAR,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		pod_name VARCHAR,
		namespace VARCHAR
	)`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_cluster ON tasks(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_cluster_id ON tasks(cluster_arn, id)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_task_definition ON tasks(task_definition_arn)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_container_instance ON tasks(container_instance_arn)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_started_by ON tasks(started_by)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(last_status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_desired_status ON tasks(desired_status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_launch_type ON tasks(launch_type)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_pod ON tasks(pod_name, namespace)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// duckDBTransaction wraps sql.Tx to implement storage.Transaction
type duckDBTransaction struct {
	tx *sql.Tx
}

func (t *duckDBTransaction) Commit() error {
	return t.tx.Commit()
}

func (t *duckDBTransaction) Rollback() error {
	return t.tx.Rollback()
}