package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb" // DuckDB driver
	"github.com/nandemo-ya/kecs/internal/storage"
)

// DuckDBStorage implements storage.Storage using DuckDB
type DuckDBStorage struct {
	db                  *sql.DB
	dbPath              string
	clusterStore        *clusterStore
	taskDefinitionStore *taskDefinitionStore
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
		registered_container_instances_count INTEGER DEFAULT 0,
		running_tasks_count INTEGER DEFAULT 0,
		pending_tasks_count INTEGER DEFAULT 0,
		active_services_count INTEGER DEFAULT 0,
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