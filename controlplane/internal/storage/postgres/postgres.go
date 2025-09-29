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
	serviceStore           *serviceStore
	taskStore              *taskStore
	taskDefinitionStore    *taskDefinitionStore
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

// NewPostgreSQLStorage is an alias for NewPostgresStorage for consistency
func NewPostgreSQLStorage(connString string) storage.Storage {
	return NewPostgresStorage(connString)
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
	s.serviceStore = &serviceStore{db: db}
	s.taskStore = &taskStore{db: db}
	s.taskDefinitionStore = &taskDefinitionStore{db: db}
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
	query := `
	CREATE TABLE IF NOT EXISTS task_definitions (
		id TEXT PRIMARY KEY,
		arn TEXT NOT NULL UNIQUE,
		family TEXT NOT NULL,
		revision INTEGER NOT NULL,
		task_role_arn TEXT,
		execution_role_arn TEXT,
		network_mode TEXT NOT NULL DEFAULT 'bridge',
		container_definitions TEXT NOT NULL,
		volumes TEXT,
		placement_constraints TEXT,
		requires_compatibilities TEXT,
		cpu TEXT,
		memory TEXT,
		tags TEXT,
		pid_mode TEXT,
		ipc_mode TEXT,
		proxy_configuration TEXT,
		inference_accelerators TEXT,
		runtime_platform TEXT,
		status TEXT NOT NULL DEFAULT 'ACTIVE',
		region TEXT,
		account_id TEXT,
		registered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deregistered_at TIMESTAMP,
		UNIQUE(family, revision)
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create task_definitions table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_family ON task_definitions(family)",
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_status ON task_definitions(status)",
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_registered_at ON task_definitions(registered_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

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
	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		arn TEXT NOT NULL UNIQUE,
		cluster_arn TEXT NOT NULL,
		task_definition_arn TEXT NOT NULL,
		container_instance_arn TEXT,
		overrides TEXT,
		last_status TEXT NOT NULL,
		desired_status TEXT NOT NULL,
		cpu TEXT,
		memory TEXT,
		containers TEXT NOT NULL,
		started_by TEXT,
		version BIGINT DEFAULT 0,
		stop_code TEXT,
		stopped_reason TEXT,
		stopping_at TIMESTAMP,
		stopped_at TIMESTAMP,
		connectivity TEXT,
		connectivity_at TIMESTAMP,
		pull_started_at TIMESTAMP,
		pull_stopped_at TIMESTAMP,
		execution_stopped_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		started_at TIMESTAMP,
		launch_type TEXT NOT NULL,
		platform_version TEXT,
		platform_family TEXT,
		task_group TEXT,
		attachments TEXT,
		health_status TEXT,
		tags TEXT,
		attributes TEXT,
		enable_execute_command BOOLEAN DEFAULT FALSE,
		capacity_provider_name TEXT,
		ephemeral_storage TEXT,
		region TEXT,
		account_id TEXT,
		pod_name TEXT,
		namespace TEXT,
		service_registries TEXT
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_tasks_cluster_arn ON tasks(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_task_definition_arn ON tasks(task_definition_arn)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_last_status ON tasks(last_status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_desired_status ON tasks(desired_status)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_started_by ON tasks(started_by)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_cluster_status ON tasks(cluster_arn, last_status)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createAccountSettingsTable creates the account_settings table
func (s *PostgresStorage) createAccountSettingsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS account_settings (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		value TEXT NOT NULL,
		principal_arn TEXT NOT NULL,
		is_default BOOLEAN DEFAULT FALSE,
		region TEXT,
		account_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(principal_arn, name)
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create account_settings table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_account_settings_name ON account_settings(name)",
		"CREATE INDEX IF NOT EXISTS idx_account_settings_principal_arn ON account_settings(principal_arn)",
		"CREATE INDEX IF NOT EXISTS idx_account_settings_is_default ON account_settings(is_default)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createTaskSetsTable creates the task_sets table
func (s *PostgresStorage) createTaskSetsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS task_sets (
		id TEXT PRIMARY KEY,
		arn TEXT NOT NULL UNIQUE,
		service_arn TEXT NOT NULL,
		cluster_arn TEXT NOT NULL,
		external_id TEXT,
		task_definition TEXT NOT NULL,
		launch_type TEXT,
		platform_version TEXT,
		platform_family TEXT,
		network_configuration TEXT,
		load_balancers TEXT,
		service_registries TEXT,
		capacity_provider_strategy TEXT,
		scale TEXT,
		computed_desired_count INTEGER DEFAULT 0,
		pending_count INTEGER DEFAULT 0,
		running_count INTEGER DEFAULT 0,
		status TEXT NOT NULL,
		stability_status TEXT,
		stability_status_at TIMESTAMP,
		started_by TEXT,
		tags TEXT,
		fargate_ephemeral_storage TEXT,
		region TEXT,
		account_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(service_arn, id)
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create task_sets table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_task_sets_service_arn ON task_sets(service_arn)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_cluster_arn ON task_sets(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_status ON task_sets(status)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_created_at ON task_sets(created_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createContainerInstancesTable creates the container_instances table
func (s *PostgresStorage) createContainerInstancesTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS container_instances (
		id TEXT PRIMARY KEY,
		arn TEXT NOT NULL UNIQUE,
		cluster_arn TEXT NOT NULL,
		ec2_instance_id TEXT,
		status TEXT NOT NULL,
		status_reason TEXT,
		agent_connected BOOLEAN DEFAULT FALSE,
		agent_update_status TEXT,
		running_tasks_count INTEGER DEFAULT 0,
		pending_tasks_count INTEGER DEFAULT 0,
		version BIGINT DEFAULT 0,
		version_info TEXT,
		registered_resources TEXT,
		remaining_resources TEXT,
		attributes TEXT,
		attachments TEXT,
		tags TEXT,
		capacity_provider_name TEXT,
		health_status TEXT,
		region TEXT,
		account_id TEXT,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deregistered_at TIMESTAMP
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create container_instances table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_container_instances_cluster_arn ON container_instances(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_status ON container_instances(status)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_ec2_instance_id ON container_instances(ec2_instance_id)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_registered_at ON container_instances(registered_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createAttributesTable creates the attributes table
func (s *PostgresStorage) createAttributesTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS attributes (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		value TEXT,
		target_type TEXT NOT NULL,
		target_id TEXT NOT NULL,
		cluster TEXT NOT NULL,
		region TEXT,
		account_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(name, target_type, target_id, cluster)
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create attributes table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_attributes_cluster ON attributes(cluster)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_target_type ON attributes(target_type)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_target_id ON attributes(target_id)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createELBv2Tables creates the ELBv2 related tables
func (s *PostgresStorage) createELBv2Tables(ctx context.Context) error {
	// Create load balancers table
	lbQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_load_balancers (
		arn TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		dns_name TEXT,
		canonical_hosted_zone_id TEXT,
		state TEXT,
		type TEXT,
		scheme TEXT,
		vpc_id TEXT,
		subnets TEXT,
		availability_zones TEXT,
		security_groups TEXT,
		ip_address_type TEXT,
		tags TEXT,
		region TEXT,
		account_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := s.db.ExecContext(ctx, lbQuery); err != nil {
		return fmt.Errorf("failed to create elbv2_load_balancers table: %w", err)
	}

	// Create target groups table
	tgQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_target_groups (
		arn TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		protocol TEXT,
		port INTEGER,
		vpc_id TEXT,
		target_type TEXT,
		health_check_enabled BOOLEAN DEFAULT TRUE,
		health_check_protocol TEXT,
		health_check_port TEXT,
		health_check_path TEXT,
		health_check_interval_seconds INTEGER,
		health_check_timeout_seconds INTEGER,
		healthy_threshold_count INTEGER,
		unhealthy_threshold_count INTEGER,
		matcher TEXT,
		load_balancer_arns TEXT,
		tags TEXT,
		region TEXT,
		account_id TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := s.db.ExecContext(ctx, tgQuery); err != nil {
		return fmt.Errorf("failed to create elbv2_target_groups table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_elbv2_load_balancers_region ON elbv2_load_balancers(region)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_load_balancers_vpc_id ON elbv2_load_balancers(vpc_id)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_target_groups_region ON elbv2_target_groups(region)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_target_groups_vpc_id ON elbv2_target_groups(vpc_id)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createTaskLogsTable creates the task_logs table
func (s *PostgresStorage) createTaskLogsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS task_logs (
		id TEXT PRIMARY KEY,
		task_arn TEXT NOT NULL,
		container_name TEXT NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		log_line TEXT NOT NULL,
		log_level TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create task_logs table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_task_logs_task_arn ON task_logs(task_arn)",
		"CREATE INDEX IF NOT EXISTS idx_task_logs_container_name ON task_logs(container_name)",
		"CREATE INDEX IF NOT EXISTS idx_task_logs_timestamp ON task_logs(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_task_logs_created_at ON task_logs(created_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
