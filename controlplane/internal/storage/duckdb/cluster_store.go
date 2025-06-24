package duckdb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// clusterStore implements storage.ClusterStore using DuckDB
type clusterStore struct {
	db   *sql.DB
	pool *ConnectionPool
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
			configuration, settings, tags, k8s_cluster_name,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			capacity_providers, default_capacity_provider_strategy,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	// Convert empty strings to NULL for JSON fields
	configuration := sql.NullString{String: cluster.Configuration, Valid: cluster.Configuration != ""}
	settings := sql.NullString{String: cluster.Settings, Valid: cluster.Settings != ""}
	tags := sql.NullString{String: cluster.Tags, Valid: cluster.Tags != ""}
	k8sClusterName := sql.NullString{String: cluster.K8sClusterName, Valid: cluster.K8sClusterName != ""}
	capacityProviders := sql.NullString{String: cluster.CapacityProviders, Valid: cluster.CapacityProviders != ""}
	defaultCapacityProviderStrategy := sql.NullString{String: cluster.DefaultCapacityProviderStrategy, Valid: cluster.DefaultCapacityProviderStrategy != ""}

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
		k8sClusterName,
		cluster.RegisteredContainerInstancesCount,
		cluster.RunningTasksCount,
		cluster.PendingTasksCount,
		cluster.ActiveServicesCount,
		capacityProviders,
		defaultCapacityProviderStrategy,
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
	cluster := &storage.Cluster{}
	var configuration, settings, tags, k8sClusterName, capacityProviders, defaultCapacityProviderStrategy sql.NullString

	query := `
		SELECT 
			id, arn, name, status, region, account_id,
			configuration, settings, tags, k8s_cluster_name,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			capacity_providers, default_capacity_provider_strategy,
			created_at, updated_at
		FROM clusters
		WHERE name = ?`

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
		&k8sClusterName,
		&cluster.RegisteredContainerInstancesCount,
		&cluster.RunningTasksCount,
		&cluster.PendingTasksCount,
		&cluster.ActiveServicesCount,
		&capacityProviders,
		&defaultCapacityProviderStrategy,
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
	cluster.K8sClusterName = k8sClusterName.String
	cluster.CapacityProviders = capacityProviders.String
	cluster.DefaultCapacityProviderStrategy = defaultCapacityProviderStrategy.String

	return cluster, nil
}

// List retrieves all clusters
func (s *clusterStore) List(ctx context.Context) ([]*storage.Cluster, error) {
	query := `
		SELECT 
			id, arn, name, status, region, account_id,
			configuration, settings, tags, k8s_cluster_name,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			capacity_providers, default_capacity_provider_strategy,
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
		var configuration, settings, tags, k8sClusterName, capacityProviders, defaultCapacityProviderStrategy sql.NullString

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
			&k8sClusterName,
			&cluster.RegisteredContainerInstancesCount,
			&cluster.RunningTasksCount,
			&cluster.PendingTasksCount,
			&cluster.ActiveServicesCount,
			&capacityProviders,
			&defaultCapacityProviderStrategy,
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
		cluster.K8sClusterName = k8sClusterName.String
		cluster.CapacityProviders = capacityProviders.String
		cluster.DefaultCapacityProviderStrategy = defaultCapacityProviderStrategy.String

		clusters = append(clusters, cluster)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clusters: %w", err)
	}

	return clusters, nil
}

// ListWithPagination retrieves clusters with pagination support
func (s *clusterStore) ListWithPagination(ctx context.Context, limit int, nextToken string) ([]*storage.Cluster, string, error) {
	var args []interface{}

	baseQuery := `
		SELECT 
			id, arn, name, status, region, account_id,
			configuration, settings, tags, k8s_cluster_name,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			capacity_providers, default_capacity_provider_strategy,
			created_at, updated_at
		FROM clusters`

	// Add token-based pagination
	if nextToken != "" {
		// Validate that the token exists
		var exists bool
		err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM clusters WHERE id = ?)", nextToken).Scan(&exists)
		if err != nil || !exists {
			// Invalid token, start from beginning
			nextToken = ""
		} else {
			baseQuery += " WHERE id > ?"
			args = append(args, nextToken)
		}
	}

	// Add ordering and limit
	baseQuery += " ORDER BY id"
	if limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, limit+1) // Get one extra to determine if there are more results
	}

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list clusters: %w", err)
	}
	defer rows.Close()

	var clusters []*storage.Cluster

	for rows.Next() {
		cluster := &storage.Cluster{}
		var configuration, settings, tags, k8sClusterName, capacityProviders, defaultCapacityProviderStrategy sql.NullString

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
			&k8sClusterName,
			&cluster.RegisteredContainerInstancesCount,
			&cluster.RunningTasksCount,
			&cluster.PendingTasksCount,
			&cluster.ActiveServicesCount,
			&capacityProviders,
			&defaultCapacityProviderStrategy,
			&cluster.CreatedAt,
			&cluster.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan cluster row: %w", err)
		}

		// Handle nullable fields
		cluster.Configuration = configuration.String
		cluster.Settings = settings.String
		cluster.Tags = tags.String
		cluster.K8sClusterName = k8sClusterName.String
		cluster.CapacityProviders = capacityProviders.String
		cluster.DefaultCapacityProviderStrategy = defaultCapacityProviderStrategy.String

		clusters = append(clusters, cluster)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("error iterating clusters: %w", err)
	}

	// Determine next token
	var newNextToken string
	if limit > 0 && len(clusters) > limit {
		// Remove the extra item
		clusters = clusters[:limit]
		// Use the last item's ID as the next token
		if len(clusters) > 0 {
			newNextToken = clusters[len(clusters)-1].ID
		}
	}

	return clusters, newNextToken, nil
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
			k8s_cluster_name = ?,
			registered_container_instances_count = ?,
			running_tasks_count = ?,
			pending_tasks_count = ?,
			active_services_count = ?,
			capacity_providers = ?,
			default_capacity_provider_strategy = ?,
			updated_at = ?
		WHERE name = ?`

	// Convert empty strings to NULL for JSON fields
	configuration := sql.NullString{String: cluster.Configuration, Valid: cluster.Configuration != ""}
	settings := sql.NullString{String: cluster.Settings, Valid: cluster.Settings != ""}
	tags := sql.NullString{String: cluster.Tags, Valid: cluster.Tags != ""}
	k8sClusterName := sql.NullString{String: cluster.K8sClusterName, Valid: cluster.K8sClusterName != ""}
	capacityProviders := sql.NullString{String: cluster.CapacityProviders, Valid: cluster.CapacityProviders != ""}
	defaultCapacityProviderStrategy := sql.NullString{String: cluster.DefaultCapacityProviderStrategy, Valid: cluster.DefaultCapacityProviderStrategy != ""}

	result, err := s.db.ExecContext(ctx, query,
		cluster.ARN,
		cluster.Status,
		cluster.Region,
		cluster.AccountID,
		configuration,
		settings,
		tags,
		k8sClusterName,
		cluster.RegisteredContainerInstancesCount,
		cluster.RunningTasksCount,
		cluster.PendingTasksCount,
		cluster.ActiveServicesCount,
		capacityProviders,
		defaultCapacityProviderStrategy,
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
