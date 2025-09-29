package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type clusterStore struct {
	db *sql.DB
}

// Create creates a new cluster
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
			localstack_state,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err := s.db.ExecContext(ctx, query,
		cluster.ID,
		cluster.ARN,
		cluster.Name,
		cluster.Status,
		cluster.Region,
		cluster.AccountID,
		toNullString(cluster.Configuration),
		toNullString(cluster.Settings),
		toNullString(cluster.Tags),
		toNullString(cluster.K8sClusterName),
		cluster.RegisteredContainerInstancesCount,
		cluster.RunningTasksCount,
		cluster.PendingTasksCount,
		cluster.ActiveServicesCount,
		toNullString(cluster.CapacityProviders),
		toNullString(cluster.DefaultCapacityProviderStrategy),
		toNullString(cluster.LocalStackState),
		cluster.CreatedAt,
		cluster.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to create cluster: %w", err)
	}

	return nil
}

// Get retrieves a cluster by ARN or name
func (s *clusterStore) Get(ctx context.Context, identifier string) (*storage.Cluster, error) {
	query := `
		SELECT
			id, arn, name, status, region, account_id,
			configuration, settings, tags, k8s_cluster_name,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			capacity_providers, default_capacity_provider_strategy,
			localstack_state,
			created_at, updated_at
		FROM clusters
		WHERE arn = $1 OR name = $2`

	var cluster storage.Cluster
	var configuration, settings, tags, k8sClusterName sql.NullString
	var capacityProviders, defaultCapacityProviderStrategy, localStackState sql.NullString

	err := s.db.QueryRowContext(ctx, query, identifier, identifier).Scan(
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
		&localStackState,
		&cluster.CreatedAt,
		&cluster.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	cluster.Configuration = fromNullString(configuration)
	cluster.Settings = fromNullString(settings)
	cluster.Tags = fromNullString(tags)
	cluster.K8sClusterName = fromNullString(k8sClusterName)
	cluster.CapacityProviders = fromNullString(capacityProviders)
	cluster.DefaultCapacityProviderStrategy = fromNullString(defaultCapacityProviderStrategy)
	cluster.LocalStackState = fromNullString(localStackState)

	return &cluster, nil
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
			localstack_state,
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
		var cluster storage.Cluster
		var configuration, settings, tags, k8sClusterName sql.NullString
		var capacityProviders, defaultCapacityProviderStrategy, localStackState sql.NullString

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
			&localStackState,
			&cluster.CreatedAt,
			&cluster.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cluster row: %w", err)
		}

		cluster.Configuration = fromNullString(configuration)
		cluster.Settings = fromNullString(settings)
		cluster.Tags = fromNullString(tags)
		cluster.K8sClusterName = fromNullString(k8sClusterName)
		cluster.CapacityProviders = fromNullString(capacityProviders)
		cluster.DefaultCapacityProviderStrategy = fromNullString(defaultCapacityProviderStrategy)
		cluster.LocalStackState = fromNullString(localStackState)

		clusters = append(clusters, &cluster)
	}

	return clusters, nil
}

// ListWithPagination retrieves clusters with pagination
func (s *clusterStore) ListWithPagination(ctx context.Context, maxResults int, nextToken string) ([]*storage.Cluster, string, error) {
	// Parse the next token to get the offset
	offset := 0
	if nextToken != "" {
		if _, err := fmt.Sscanf(nextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	query := `
		SELECT
			id, arn, name, status, region, account_id,
			configuration, settings, tags, k8s_cluster_name,
			registered_container_instances_count, running_tasks_count,
			pending_tasks_count, active_services_count,
			capacity_providers, default_capacity_provider_strategy,
			localstack_state,
			created_at, updated_at
		FROM clusters
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := s.db.QueryContext(ctx, query, maxResults, offset)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list clusters with pagination: %w", err)
	}
	defer rows.Close()

	var clusters []*storage.Cluster
	for rows.Next() {
		var cluster storage.Cluster
		var configuration, settings, tags, k8sClusterName sql.NullString
		var capacityProviders, defaultCapacityProviderStrategy, localStackState sql.NullString

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
			&localStackState,
			&cluster.CreatedAt,
			&cluster.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan cluster row: %w", err)
		}

		cluster.Configuration = fromNullString(configuration)
		cluster.Settings = fromNullString(settings)
		cluster.Tags = fromNullString(tags)
		cluster.K8sClusterName = fromNullString(k8sClusterName)
		cluster.CapacityProviders = fromNullString(capacityProviders)
		cluster.DefaultCapacityProviderStrategy = fromNullString(defaultCapacityProviderStrategy)
		cluster.LocalStackState = fromNullString(localStackState)

		clusters = append(clusters, &cluster)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if len(clusters) == int(maxResults) {
		newNextToken = fmt.Sprintf("%d", offset+int(maxResults))
	}

	return clusters, newNextToken, nil
}

// Update updates an existing cluster
func (s *clusterStore) Update(ctx context.Context, cluster *storage.Cluster) error {
	cluster.UpdatedAt = time.Now()

	query := `
		UPDATE clusters SET
			status = $1, region = $2, account_id = $3,
			configuration = $4, settings = $5, tags = $6,
			k8s_cluster_name = $7,
			registered_container_instances_count = $8,
			running_tasks_count = $9,
			pending_tasks_count = $10,
			active_services_count = $11,
			capacity_providers = $12,
			default_capacity_provider_strategy = $13,
			localstack_state = $14,
			updated_at = $15
		WHERE arn = $16`

	result, err := s.db.ExecContext(ctx, query,
		cluster.Status,
		cluster.Region,
		cluster.AccountID,
		toNullString(cluster.Configuration),
		toNullString(cluster.Settings),
		toNullString(cluster.Tags),
		toNullString(cluster.K8sClusterName),
		cluster.RegisteredContainerInstancesCount,
		cluster.RunningTasksCount,
		cluster.PendingTasksCount,
		cluster.ActiveServicesCount,
		toNullString(cluster.CapacityProviders),
		toNullString(cluster.DefaultCapacityProviderStrategy),
		toNullString(cluster.LocalStackState),
		cluster.UpdatedAt,
		cluster.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update cluster: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	return nil
}

// Delete deletes a cluster
func (s *clusterStore) Delete(ctx context.Context, identifier string) error {
	query := `DELETE FROM clusters WHERE arn = $1 OR name = $2`

	result, err := s.db.ExecContext(ctx, query, identifier, identifier)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	return nil
}

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
