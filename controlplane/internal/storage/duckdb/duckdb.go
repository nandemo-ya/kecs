package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb/v2" // DuckDB driver

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DuckDBStorage implements storage.Storage using DuckDB
type DuckDBStorage struct {
	db                     *sql.DB
	pool                   *ConnectionPool
	dbPath                 string
	clusterStore           *clusterStore
	taskDefinitionStore    *taskDefinitionStore
	serviceStore           *serviceStore
	taskStore              *taskStore
	accountSettingStore    *accountSettingStore
	taskSetStore           *taskSetStore
	containerInstanceStore *containerInstanceStore
	attributeStore         *attributeStore
	elbv2Store             *elbv2Store
}

// NewDuckDBStorage creates a new DuckDB storage instance
func NewDuckDBStorage(dbPath string) (*DuckDBStorage, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}

	// Create connection pool with optimized settings
	// maxConns: 25 - Allow up to 25 concurrent connections
	// maxIdleConns: 5 - Keep 5 idle connections ready
	pool, err := NewConnectionPool(dbPath, 25, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	db := pool.DB()

	s := &DuckDBStorage{
		db:     db,
		pool:   pool,
		dbPath: dbPath,
	}

	// Create stores with the pooled connection
	s.clusterStore = &clusterStore{db: db, pool: pool}
	s.taskDefinitionStore = &taskDefinitionStore{db: db}
	s.serviceStore = &serviceStore{db: db}
	s.taskStore = &taskStore{db: db}
	s.accountSettingStore = &accountSettingStore{db: db}
	s.taskSetStore = &taskSetStore{db: db}
	s.containerInstanceStore = &containerInstanceStore{db: db}
	s.attributeStore = &attributeStore{db: db}
	s.elbv2Store = &elbv2Store{db: db}

	return s, nil
}

// Initialize creates all necessary tables and indexes
func (s *DuckDBStorage) Initialize(ctx context.Context) error {
	logging.Info("Initializing DuckDB storage")

	// Migrate existing tables if needed
	if err := s.migrateSchema(ctx); err != nil {
		return fmt.Errorf("failed to migrate schema: %w", err)
	}

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

	// Create task sets table
	if err := s.createTaskSetsTable(ctx); err != nil {
		return fmt.Errorf("failed to create task sets table: %w", err)
	}

	// Create container instances table
	if err := s.createContainerInstancesTable(ctx); err != nil {
		return fmt.Errorf("failed to create container instances table: %w", err)
	}

	// Create attributes table
	if err := s.createAttributesTable(ctx); err != nil {
		return fmt.Errorf("failed to create attributes table: %w", err)
	}

	// Create ELBv2 tables
	if err := s.createELBv2Tables(ctx); err != nil {
		return fmt.Errorf("failed to create ELBv2 tables: %w", err)
	}

	// Initialize prepared statements for common queries
	if err := s.pool.InitializeCommonStatements(ctx); err != nil {
		logging.Warn("Failed to initialize prepared statements", "error", err)
		// Non-fatal error - continue initialization
	}

	logging.Info("DuckDB storage initialized successfully")
	return nil
}

// Close closes the database connection
func (s *DuckDBStorage) Close() error {
	if s.pool != nil {
		return s.pool.Close()
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

// TaskSetStore returns the task set store
func (s *DuckDBStorage) TaskSetStore() storage.TaskSetStore {
	return s.taskSetStore
}

// ContainerInstanceStore returns the container instance store
func (s *DuckDBStorage) ContainerInstanceStore() storage.ContainerInstanceStore {
	return s.containerInstanceStore
}

// AttributeStore returns the attribute store
func (s *DuckDBStorage) AttributeStore() storage.AttributeStore {
	return s.attributeStore
}

// ELBv2Store returns the ELBv2 store
func (s *DuckDBStorage) ELBv2Store() storage.ELBv2Store {
	return s.elbv2Store
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
		configuration VARCHAR,
		settings VARCHAR,
		tags VARCHAR,
		k8s_cluster_name VARCHAR,
		registered_container_instances_count INTEGER DEFAULT 0,
		running_tasks_count INTEGER DEFAULT 0,
		pending_tasks_count INTEGER DEFAULT 0,
		active_services_count INTEGER DEFAULT 0,
		capacity_providers VARCHAR,
		default_capacity_provider_strategy VARCHAR,
		localstack_state VARCHAR,
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

// migrateSchema migrates existing database schema to new format
func (s *DuckDBStorage) migrateSchema(ctx context.Context) error {
	// Check if clusters table exists
	var tableExists bool
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'clusters'
		)
	`).Scan(&tableExists)
	if err != nil || !tableExists {
		// Table doesn't exist, no migration needed
		return nil
	}

	// Check if tags column is JSON type (old schema)
	var dataType string
	err = s.db.QueryRowContext(ctx, `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'clusters' AND column_name = 'tags'
	`).Scan(&dataType)
	if err != nil {
		// Column doesn't exist or error checking, skip migration
		return nil
	}

	if dataType != "JSON" {
		// Already migrated
		return nil
	}

	logging.Info("Migrating clusters table from JSON to VARCHAR columns")

	// Create a transaction for the migration
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create temporary table with new schema
	_, err = tx.ExecContext(ctx, `
		CREATE TABLE clusters_new (
			id VARCHAR PRIMARY KEY,
			arn VARCHAR NOT NULL UNIQUE,
			name VARCHAR NOT NULL UNIQUE,
			status VARCHAR NOT NULL,
			region VARCHAR NOT NULL,
			account_id VARCHAR NOT NULL,
			configuration VARCHAR,
			settings VARCHAR,
			tags VARCHAR,
			k8s_cluster_name VARCHAR,
			registered_container_instances_count INTEGER DEFAULT 0,
			running_tasks_count INTEGER DEFAULT 0,
			pending_tasks_count INTEGER DEFAULT 0,
			active_services_count INTEGER DEFAULT 0,
			capacity_providers VARCHAR,
			default_capacity_provider_strategy VARCHAR,
			localstack_state VARCHAR,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// Copy data with JSON to VARCHAR conversion
	_, err = tx.ExecContext(ctx, `
		INSERT INTO clusters_new
		SELECT 
			id, arn, name, status, region, account_id,
			CAST(configuration AS VARCHAR),
			CAST(settings AS VARCHAR),
			CAST(tags AS VARCHAR),
			k8s_cluster_name,
			registered_container_instances_count,
			running_tasks_count,
			pending_tasks_count,
			active_services_count,
			CAST(capacity_providers AS VARCHAR),
			CAST(default_capacity_provider_strategy AS VARCHAR),
			NULL as localstack_state,
			created_at, updated_at
		FROM clusters
	`)
	if err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	// Drop old table
	_, err = tx.ExecContext(ctx, "DROP TABLE clusters")
	if err != nil {
		return fmt.Errorf("failed to drop old table: %w", err)
	}

	// Rename new table
	_, err = tx.ExecContext(ctx, "ALTER TABLE clusters_new RENAME TO clusters")
	if err != nil {
		return fmt.Errorf("failed to rename table: %w", err)
	}

	// Recreate indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_clusters_name ON clusters(name)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_status ON clusters(status)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_region ON clusters(region)",
		"CREATE INDEX IF NOT EXISTS idx_clusters_account_id ON clusters(account_id)",
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	logging.Info("Successfully migrated clusters table")
	return nil
}

// createTaskSetsTable creates the task_sets table
func (s *DuckDBStorage) createTaskSetsTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS task_sets (
		id VARCHAR PRIMARY KEY,
		arn VARCHAR NOT NULL UNIQUE,
		service_arn VARCHAR NOT NULL,
		cluster_arn VARCHAR NOT NULL,
		external_id VARCHAR,
		task_definition VARCHAR NOT NULL,
		launch_type VARCHAR,
		platform_version VARCHAR,
		platform_family VARCHAR,
		network_configuration VARCHAR,
		load_balancers VARCHAR,
		service_registries VARCHAR,
		capacity_provider_strategy VARCHAR,
		scale VARCHAR,
		computed_desired_count INTEGER NOT NULL DEFAULT 0,
		pending_count INTEGER NOT NULL DEFAULT 0,
		running_count INTEGER NOT NULL DEFAULT 0,
		status VARCHAR NOT NULL,
		stability_status VARCHAR NOT NULL,
		stability_status_at TIMESTAMP,
		started_by VARCHAR,
		tags VARCHAR,
		fargate_ephemeral_storage VARCHAR,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		UNIQUE(service_arn, id)
	)`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create task_sets table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_task_sets_service ON task_sets(service_arn)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_cluster ON task_sets(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_status ON task_sets(status)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_region ON task_sets(region)",
		"CREATE INDEX IF NOT EXISTS idx_task_sets_account_id ON task_sets(account_id)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createContainerInstancesTable creates the container_instances table
func (s *DuckDBStorage) createContainerInstancesTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS container_instances (
		id VARCHAR PRIMARY KEY,
		arn VARCHAR NOT NULL UNIQUE,
		cluster_arn VARCHAR NOT NULL,
		ec2_instance_id VARCHAR NOT NULL,
		status VARCHAR NOT NULL,
		status_reason VARCHAR,
		agent_connected BOOLEAN NOT NULL DEFAULT false,
		agent_update_status VARCHAR,
		running_tasks_count INTEGER NOT NULL DEFAULT 0,
		pending_tasks_count INTEGER NOT NULL DEFAULT 0,
		version BIGINT NOT NULL DEFAULT 1,
		version_info VARCHAR,
		registered_resources VARCHAR,
		remaining_resources VARCHAR,
		attributes VARCHAR,
		attachments VARCHAR,
		tags VARCHAR,
		capacity_provider_name VARCHAR,
		health_status VARCHAR,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		registered_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		deregistered_at TIMESTAMP
	)`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create container_instances table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_container_instances_cluster ON container_instances(cluster_arn)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_status ON container_instances(status)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_ec2_instance ON container_instances(ec2_instance_id)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_region ON container_instances(region)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_account_id ON container_instances(account_id)",
		"CREATE INDEX IF NOT EXISTS idx_container_instances_capacity_provider ON container_instances(capacity_provider_name)",
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
func (s *DuckDBStorage) createAttributesTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS attributes (
		id VARCHAR PRIMARY KEY,
		name VARCHAR NOT NULL,
		value VARCHAR,
		target_type VARCHAR NOT NULL,
		target_id VARCHAR NOT NULL,
		cluster VARCHAR NOT NULL,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		UNIQUE(name, target_type, target_id, cluster)
	)`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to create attributes table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_attributes_name ON attributes(name)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_target ON attributes(target_type, target_id)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_cluster ON attributes(cluster)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_region ON attributes(region)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_account_id ON attributes(account_id)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_created_at ON attributes(created_at)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}


// createELBv2Tables creates all ELBv2-related tables
func (s *DuckDBStorage) createELBv2Tables(ctx context.Context) error {
	// Create load balancers table
	lbQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_load_balancers (
		arn VARCHAR PRIMARY KEY,
		name VARCHAR NOT NULL UNIQUE,
		dns_name VARCHAR NOT NULL,
		canonical_hosted_zone_id VARCHAR,
		state VARCHAR NOT NULL,
		type VARCHAR NOT NULL,
		scheme VARCHAR NOT NULL,
		vpc_id VARCHAR,
		subnets TEXT,
		availability_zones TEXT,
		security_groups TEXT,
		ip_address_type VARCHAR,
		tags TEXT,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	if _, err := s.db.ExecContext(ctx, lbQuery); err != nil {
		return fmt.Errorf("failed to create load balancers table: %w", err)
	}

	// Create target groups table
	tgQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_target_groups (
		arn VARCHAR PRIMARY KEY,
		name VARCHAR NOT NULL UNIQUE,
		protocol VARCHAR NOT NULL,
		port INTEGER NOT NULL,
		vpc_id VARCHAR,
		target_type VARCHAR NOT NULL,
		health_check_enabled BOOLEAN NOT NULL,
		health_check_protocol VARCHAR,
		health_check_port VARCHAR,
		health_check_path VARCHAR,
		health_check_interval_seconds INTEGER,
		health_check_timeout_seconds INTEGER,
		healthy_threshold_count INTEGER,
		unhealthy_threshold_count INTEGER,
		matcher VARCHAR,
		load_balancer_arns TEXT,
		tags TEXT,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	if _, err := s.db.ExecContext(ctx, tgQuery); err != nil {
		return fmt.Errorf("failed to create target groups table: %w", err)
	}

	// Create listeners table
	listenerQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_listeners (
		arn VARCHAR PRIMARY KEY,
		load_balancer_arn VARCHAR NOT NULL,
		port INTEGER NOT NULL,
		protocol VARCHAR NOT NULL,
		default_actions TEXT,
		ssl_policy VARCHAR,
		certificates TEXT,
		alpn_policy TEXT,
		tags TEXT,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	if _, err := s.db.ExecContext(ctx, listenerQuery); err != nil {
		return fmt.Errorf("failed to create listeners table: %w", err)
	}

	// Create targets table
	targetsQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_targets (
		target_group_arn VARCHAR NOT NULL,
		id VARCHAR NOT NULL,
		port INTEGER NOT NULL,
		availability_zone VARCHAR,
		health_state VARCHAR NOT NULL,
		health_reason VARCHAR,
		health_description VARCHAR,
		registered_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		PRIMARY KEY (target_group_arn, id)
	)`

	if _, err := s.db.ExecContext(ctx, targetsQuery); err != nil {
		return fmt.Errorf("failed to create targets table: %w", err)
	}

	// Create ELBv2 rules table
	rulesQuery := `
	CREATE TABLE IF NOT EXISTS elbv2_rules (
		arn VARCHAR PRIMARY KEY,
		listener_arn VARCHAR NOT NULL,
		priority INTEGER NOT NULL,
		conditions TEXT NOT NULL,
		actions TEXT NOT NULL,
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		tags TEXT,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`

	if _, err := s.db.ExecContext(ctx, rulesQuery); err != nil {
		return fmt.Errorf("failed to create rules table: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_elbv2_lb_name ON elbv2_load_balancers(name)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_lb_region ON elbv2_load_balancers(region)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_lb_state ON elbv2_load_balancers(state)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_tg_name ON elbv2_target_groups(name)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_tg_region ON elbv2_target_groups(region)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_listener_lb ON elbv2_listeners(load_balancer_arn)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_targets_tg ON elbv2_targets(target_group_arn)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_targets_health ON elbv2_targets(health_state)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_rules_listener ON elbv2_rules(listener_arn)",
		"CREATE INDEX IF NOT EXISTS idx_elbv2_rules_priority ON elbv2_rules(listener_arn, priority)",
	}

	for _, idx := range indexes {
		if _, err := s.db.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}
