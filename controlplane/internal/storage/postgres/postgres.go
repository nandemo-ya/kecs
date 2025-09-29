package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// PostgresStorage implements the Storage interface for PostgreSQL
type PostgresStorage struct {
	db                     *sql.DB
	connString             string
	clusterStore           *clusterStore
	taskDefinitionStore    *taskDefinitionStore
	serviceStore           *serviceStore
	taskStore              *taskStore
	accountSettingStore    *accountSettingStore
	taskSetStore           *taskSetStore
	containerInstanceStore *containerInstanceStore
	attributeStore         *attributeStore
	elbv2Store             *elbv2Store
	taskLogStore           *taskLogStore
}

// NewPostgresStorage creates a new PostgreSQL storage instance
func NewPostgresStorage(connString string) *PostgresStorage {
	return &PostgresStorage{
		connString: connString,
	}
}

// Initialize initializes the PostgreSQL connection and creates tables
func (s *PostgresStorage) Initialize(ctx context.Context) error {
	// Open database connection
	db, err := sql.Open("postgres", s.connString)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	s.db = db

	// Initialize stores
	s.clusterStore = &clusterStore{db: db}
	s.taskDefinitionStore = &taskDefinitionStore{db: db}
	s.serviceStore = &serviceStore{db: db}
	s.taskStore = &taskStore{db: db}
	s.accountSettingStore = &accountSettingStore{db: db}
	s.taskSetStore = &taskSetStore{db: db}
	s.containerInstanceStore = &containerInstanceStore{db: db}
	s.attributeStore = &attributeStore{db: db}
	s.elbv2Store = &elbv2Store{db: db}
	s.taskLogStore = &taskLogStore{db: db}

	// Create tables
	if err := s.createTables(ctx); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *PostgresStorage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ClusterStore returns the cluster store
func (s *PostgresStorage) ClusterStore() storage.ClusterStore {
	return s.clusterStore
}

// TaskDefinitionStore returns the task definition store
func (s *PostgresStorage) TaskDefinitionStore() storage.TaskDefinitionStore {
	return s.taskDefinitionStore
}

// ServiceStore returns the service store
func (s *PostgresStorage) ServiceStore() storage.ServiceStore {
	return s.serviceStore
}

// TaskStore returns the task store
func (s *PostgresStorage) TaskStore() storage.TaskStore {
	return s.taskStore
}

// AccountSettingStore returns the account setting store
func (s *PostgresStorage) AccountSettingStore() storage.AccountSettingStore {
	return s.accountSettingStore
}

// TaskSetStore returns the task set store
func (s *PostgresStorage) TaskSetStore() storage.TaskSetStore {
	return s.taskSetStore
}

// ContainerInstanceStore returns the container instance store
func (s *PostgresStorage) ContainerInstanceStore() storage.ContainerInstanceStore {
	return s.containerInstanceStore
}

// AttributeStore returns the attribute store
func (s *PostgresStorage) AttributeStore() storage.AttributeStore {
	return s.attributeStore
}

// ELBv2Store returns the ELBv2 store
func (s *PostgresStorage) ELBv2Store() storage.ELBv2Store {
	return s.elbv2Store
}

// TaskLogStore returns the task log store
func (s *PostgresStorage) TaskLogStore() storage.TaskLogStore {
	return s.taskLogStore
}

// BeginTx starts a new transaction
func (s *PostgresStorage) BeginTx(ctx context.Context) (storage.Transaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &postgresTransaction{tx: tx}, nil
}

// postgresTransaction implements the Transaction interface
type postgresTransaction struct {
	tx *sql.Tx
}

// Commit commits the transaction
func (t *postgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *postgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

// createTables creates all required tables
func (s *PostgresStorage) createTables(ctx context.Context) error {
	// Create tables in order to handle dependencies
	if err := s.createClustersTable(ctx); err != nil {
		return err
	}
	if err := s.createTaskDefinitionsTable(ctx); err != nil {
		return err
	}
	if err := s.createServicesTable(ctx); err != nil {
		return err
	}
	if err := s.createTasksTable(ctx); err != nil {
		return err
	}
	if err := s.createAccountSettingsTable(ctx); err != nil {
		return err
	}
	if err := s.createTaskSetsTable(ctx); err != nil {
		return err
	}
	if err := s.createContainerInstancesTable(ctx); err != nil {
		return err
	}
	if err := s.createAttributesTable(ctx); err != nil {
		return err
	}
	if err := s.createELBv2Tables(ctx); err != nil {
		return err
	}
	if err := s.createTaskLogsTable(ctx); err != nil {
		return err
	}
	return nil
}

// createClustersTable creates the clusters table
func (s *PostgresStorage) createClustersTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS clusters (
		id TEXT PRIMARY KEY,
		arn TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL,
		region TEXT,
		account_id TEXT,
		configuration TEXT,
		settings TEXT,
		tags TEXT,
		k8s_cluster_name TEXT,
		registered_container_instances_count INTEGER DEFAULT 0,
		running_tasks_count INTEGER DEFAULT 0,
		pending_tasks_count INTEGER DEFAULT 0,
		active_services_count INTEGER DEFAULT 0,
		capacity_providers TEXT,
		default_capacity_provider_strategy TEXT,
		localstack_state TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create clusters table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_clusters_arn ON clusters(arn)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_created_at ON clusters(created_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createTaskDefinitionsTable creates the task_definitions table
func (s *PostgresStorage) createTaskDefinitionsTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createServicesTable creates the services table
func (s *PostgresStorage) createServicesTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS services (
		id TEXT PRIMARY KEY,
		arn TEXT NOT NULL UNIQUE,
		service_name TEXT NOT NULL,
		cluster_arn TEXT NOT NULL,
		task_definition_arn TEXT,
		desired_count INTEGER DEFAULT 0,
		running_count INTEGER DEFAULT 0,
		pending_count INTEGER DEFAULT 0,
		launch_type TEXT,
		platform_version TEXT,
		status TEXT NOT NULL,
		role_arn TEXT,
		load_balancers TEXT,
		service_registries TEXT,
		network_configuration TEXT,
		deployment_configuration TEXT,
		deployment_controller TEXT,
		placement_constraints TEXT,
		placement_strategy TEXT,
		capacity_provider_strategy TEXT,
		tags TEXT,
		scheduling_strategy TEXT,
		service_connect_configuration TEXT,
		enable_ecs_managed_tags BOOLEAN DEFAULT FALSE,
		propagate_tags TEXT,
		enable_execute_command BOOLEAN DEFAULT FALSE,
		health_check_grace_period_seconds INTEGER DEFAULT 0,
		region TEXT,
		account_id TEXT,
		deployment_name TEXT,
		namespace TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(cluster_arn, service_name)
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create services table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_services_arn ON services(arn)",
		"CREATE INDEX IF NOT EXISTS idx_services_cluster_arn ON services(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_services_service_name ON services(service_name)",
		"CREATE INDEX IF NOT EXISTS idx_services_status ON services(status)",
		"CREATE INDEX IF NOT EXISTS idx_services_created_at ON services(created_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createTasksTable creates the tasks table
func (s *PostgresStorage) createTasksTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createAccountSettingsTable creates the account_settings table
func (s *PostgresStorage) createAccountSettingsTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createTaskSetsTable creates the task_sets table
func (s *PostgresStorage) createTaskSetsTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createContainerInstancesTable creates the container_instances table
func (s *PostgresStorage) createContainerInstancesTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createAttributesTable creates the attributes table
func (s *PostgresStorage) createAttributesTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createELBv2Tables creates the ELBv2 related tables
func (s *PostgresStorage) createELBv2Tables(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}

// createTaskLogsTable creates the task_logs table
func (s *PostgresStorage) createTaskLogsTable(ctx context.Context) error {
	// Implementation will follow DuckDB schema
	// TODO: Implement
	return nil
}
