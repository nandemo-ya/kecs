package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/internal/storage"
)

// clusterStore implements storage.ClusterStore using DuckDB
type clusterStore struct {
	db *sql.DB
}

// Create inserts a new cluster into the database
func (s *clusterStore) Create(ctx context.Context, cluster *storage.Cluster) error {
	// Generate ID if not provided
	if cluster.ID == "" {
		cluster.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	cluster.CreatedAt = now
	cluster.UpdatedAt = now

	query := `
		INSERT INTO clusters (
			id, arn, name, status, region, account_id,
			configuration, settings, tags,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert empty strings to NULL for JSON fields
	configuration := sql.NullString{String: cluster.Configuration, Valid: cluster.Configuration != ""}
	settings := sql.NullString{String: cluster.Settings, Valid: cluster.Settings != ""}
	tags := sql.NullString{String: cluster.Tags, Valid: cluster.Tags != ""}

	_, err := s.db.ExecContext(ctx, query,
		cluster.ID,
		cluster.ARN,
		cluster.Name,
		cluster.Status,
		cluster.Region,
		cluster.AccountID,
		configuration,
		settings,
		tags,
		cluster.RegisteredContainerInstancesCount,
		cluster.RunningTasksCount,
		cluster.PendingTasksCount,
		cluster.ActiveServicesCount,
		cluster.CreatedAt,
		cluster.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	return nil
}

// Get retrieves a cluster by name
func (s *clusterStore) Get(ctx context.Context, name string) (*storage.Cluster, error) {
	query := `
		SELECT 
			id, arn, name, status, region, account_id,
			configuration, settings, tags,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			created_at, updated_at
		FROM clusters
		WHERE name = ?`

	cluster := &storage.Cluster{}
	var configuration, settings, tags sql.NullString

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&cluster.ID,
		&cluster.ARN,
		&cluster.Name,
		&cluster.Status,
		&cluster.Region,
		&cluster.AccountID,
		&configuration,
		&settings,
		&tags,
		&cluster.RegisteredContainerInstancesCount,
		&cluster.RunningTasksCount,
		&cluster.PendingTasksCount,
		&cluster.ActiveServicesCount,
		&cluster.CreatedAt,
		&cluster.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("cluster not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	// Handle nullable fields
	cluster.Configuration = configuration.String
	cluster.Settings = settings.String
	cluster.Tags = tags.String

	return cluster, nil
}

// List retrieves all clusters
func (s *clusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	query := `
		SELECT 
			id, arn, name, status, region, account_id,
			configuration, settings, tags,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			created_at, updated_at
		FROM clusters
		ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer rows.Close()

	var clusters []*storage.Cluster

	for rows.Next() {
		cluster := &storage.Cluster{}
		var configuration, settings, tags sql.NullString

		err := rows.Scan(
			&cluster.ID,
			&cluster.ARN,
			&cluster.Name,
			&cluster.Status,
			&cluster.Region,
			&cluster.AccountID,
			&configuration,
			&settings,
			&tags,
			&cluster.RegisteredContainerInstancesCount,
			&cluster.RunningTasksCount,
			&cluster.PendingTasksCount,
			&cluster.ActiveServicesCount,
			&cluster.CreatedAt,
			&cluster.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cluster row: %w", err)
		}

		// Handle nullable fields
		cluster.Configuration = configuration.String
		cluster.Settings = settings.String
		cluster.Tags = tags.String

		clusters = append(clusters, cluster)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clusters: %w", err)
	}

	return clusters, nil
}

// Update updates an existing cluster
func (s *clusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	// Update timestamp
	cluster.UpdatedAt = time.Now()

	query := `
		UPDATE clusters SET
			arn = ?,
			status = ?,
			region = ?,
			account_id = ?,
			configuration = ?,
			settings = ?,
			tags = ?,
			registered_container_instances_count = ?,
			running_tasks_count = ?,
			pending_tasks_count = ?,
			active_services_count = ?,
			updated_at = ?
		WHERE name = ?`

	// Convert empty strings to NULL for JSON fields
	configuration := sql.NullString{String: cluster.Configuration, Valid: cluster.Configuration != ""}
	settings := sql.NullString{String: cluster.Settings, Valid: cluster.Settings != ""}
	tags := sql.NullString{String: cluster.Tags, Valid: cluster.Tags != ""}

	result, err := s.db.ExecContext(ctx, query,
		cluster.ARN,
		cluster.Status,
		cluster.Region,
		cluster.AccountID,
		configuration,
		settings,
		tags,
		cluster.RegisteredContainerInstancesCount,
		cluster.RunningTasksCount,
		cluster.PendingTasksCount,
		cluster.ActiveServicesCount,
		cluster.UpdatedAt,
		cluster.Name,
	)

	if err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("cluster not found: %s", cluster.Name)
	}

	return nil
}

// Delete removes a cluster by name
func (s *clusterStore) Delete(ctx context.Context, name string) error {
	query := `DELETE FROM clusters WHERE name = ?`

	result, err := s.db.ExecContext(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("cluster not found: %s", name)
	}

	return nil
}

// Helper functions for JSON marshaling/unmarshaling

// marshalJSON converts an interface to JSON string
func marshalJSON(v interface{}) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// unmarshalJSON converts a JSON string to the target interface
func unmarshalJSON(data string, v interface{}) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), v)
}