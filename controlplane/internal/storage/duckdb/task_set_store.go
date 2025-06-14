package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// taskSetStore implements storage.TaskSetStore using DuckDB
type taskSetStore struct {
	db *sql.DB
}

// Create creates a new task set
func (s *taskSetStore) Create(ctx context.Context, taskSet *storage.TaskSet) error {
	if taskSet.ID == "" {
		taskSet.ID = "ts-" + uuid.New().String()[:8]
	}

	query := `
		INSERT INTO task_sets (
			id, arn, service_arn, cluster_arn, external_id, 
			task_definition, launch_type, platform_version, platform_family,
			network_configuration, load_balancers, service_registries,
			capacity_provider_strategy, scale, computed_desired_count,
			pending_count, running_count, status, stability_status,
			stability_status_at, started_by, tags, fargate_ephemeral_storage,
			region, account_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	taskSet.CreatedAt = now
	taskSet.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, query,
		taskSet.ID, taskSet.ARN, taskSet.ServiceARN, taskSet.ClusterARN, taskSet.ExternalID,
		taskSet.TaskDefinition, taskSet.LaunchType, taskSet.PlatformVersion, taskSet.PlatformFamily,
		taskSet.NetworkConfiguration, taskSet.LoadBalancers, taskSet.ServiceRegistries,
		taskSet.CapacityProviderStrategy, taskSet.Scale, taskSet.ComputedDesiredCount,
		taskSet.PendingCount, taskSet.RunningCount, taskSet.Status, taskSet.StabilityStatus,
		taskSet.StabilityStatusAt, taskSet.StartedBy, taskSet.Tags, taskSet.FargateEphemeralStorage,
		taskSet.Region, taskSet.AccountID, taskSet.CreatedAt, taskSet.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create task set: %w", err)
	}

	return nil
}

// Get retrieves a task set by service ARN and task set ID
func (s *taskSetStore) Get(ctx context.Context, serviceARN, taskSetID string) (*storage.TaskSet, error) {
	query := `
		SELECT 
			id, arn, service_arn, cluster_arn, external_id,
			task_definition, launch_type, platform_version, platform_family,
			network_configuration, load_balancers, service_registries,
			capacity_provider_strategy, scale, computed_desired_count,
			pending_count, running_count, status, stability_status,
			stability_status_at, started_by, tags, fargate_ephemeral_storage,
			region, account_id, created_at, updated_at
		FROM task_sets
		WHERE service_arn = ? AND id = ?
	`

	taskSet := &storage.TaskSet{}
	err := s.db.QueryRowContext(ctx, query, serviceARN, taskSetID).Scan(
		&taskSet.ID, &taskSet.ARN, &taskSet.ServiceARN, &taskSet.ClusterARN, &taskSet.ExternalID,
		&taskSet.TaskDefinition, &taskSet.LaunchType, &taskSet.PlatformVersion, &taskSet.PlatformFamily,
		&taskSet.NetworkConfiguration, &taskSet.LoadBalancers, &taskSet.ServiceRegistries,
		&taskSet.CapacityProviderStrategy, &taskSet.Scale, &taskSet.ComputedDesiredCount,
		&taskSet.PendingCount, &taskSet.RunningCount, &taskSet.Status, &taskSet.StabilityStatus,
		&taskSet.StabilityStatusAt, &taskSet.StartedBy, &taskSet.Tags, &taskSet.FargateEphemeralStorage,
		&taskSet.Region, &taskSet.AccountID, &taskSet.CreatedAt, &taskSet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task set not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task set: %w", err)
	}

	return taskSet, nil
}

// List retrieves task sets for a service
func (s *taskSetStore) List(ctx context.Context, serviceARN string, taskSetIDs []string) ([]*storage.TaskSet, error) {
	var query string
	var args []interface{}

	if len(taskSetIDs) > 0 {
		// Get specific task sets
		placeholders := make([]string, len(taskSetIDs))
		args = make([]interface{}, len(taskSetIDs)+1)
		args[0] = serviceARN
		for i, id := range taskSetIDs {
			placeholders[i] = "?"
			args[i+1] = id
		}

		query = fmt.Sprintf(`
			SELECT 
				id, arn, service_arn, cluster_arn, external_id,
				task_definition, launch_type, platform_version, platform_family,
				network_configuration, load_balancers, service_registries,
				capacity_provider_strategy, scale, computed_desired_count,
				pending_count, running_count, status, stability_status,
				stability_status_at, started_by, tags, fargate_ephemeral_storage,
				region, account_id, created_at, updated_at
			FROM task_sets
			WHERE service_arn = ? AND id IN (%s)
			ORDER BY created_at DESC
		`, strings.Join(placeholders, ","))
	} else {
		// Get all task sets for the service
		query = `
			SELECT 
				id, arn, service_arn, cluster_arn, external_id,
				task_definition, launch_type, platform_version, platform_family,
				network_configuration, load_balancers, service_registries,
				capacity_provider_strategy, scale, computed_desired_count,
				pending_count, running_count, status, stability_status,
				stability_status_at, started_by, tags, fargate_ephemeral_storage,
				region, account_id, created_at, updated_at
			FROM task_sets
			WHERE service_arn = ?
			ORDER BY created_at DESC
		`
		args = []interface{}{serviceARN}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list task sets: %w", err)
	}
	defer rows.Close()

	var taskSets []*storage.TaskSet
	for rows.Next() {
		taskSet := &storage.TaskSet{}
		err := rows.Scan(
			&taskSet.ID, &taskSet.ARN, &taskSet.ServiceARN, &taskSet.ClusterARN, &taskSet.ExternalID,
			&taskSet.TaskDefinition, &taskSet.LaunchType, &taskSet.PlatformVersion, &taskSet.PlatformFamily,
			&taskSet.NetworkConfiguration, &taskSet.LoadBalancers, &taskSet.ServiceRegistries,
			&taskSet.CapacityProviderStrategy, &taskSet.Scale, &taskSet.ComputedDesiredCount,
			&taskSet.PendingCount, &taskSet.RunningCount, &taskSet.Status, &taskSet.StabilityStatus,
			&taskSet.StabilityStatusAt, &taskSet.StartedBy, &taskSet.Tags, &taskSet.FargateEphemeralStorage,
			&taskSet.Region, &taskSet.AccountID, &taskSet.CreatedAt, &taskSet.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task set: %w", err)
		}
		taskSets = append(taskSets, taskSet)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate task sets: %w", err)
	}

	return taskSets, nil
}

// Update updates a task set
func (s *taskSetStore) Update(ctx context.Context, taskSet *storage.TaskSet) error {
	query := `
		UPDATE task_sets SET
			external_id = ?,
			task_definition = ?,
			launch_type = ?,
			platform_version = ?,
			platform_family = ?,
			network_configuration = ?,
			load_balancers = ?,
			service_registries = ?,
			capacity_provider_strategy = ?,
			scale = ?,
			computed_desired_count = ?,
			pending_count = ?,
			running_count = ?,
			status = ?,
			stability_status = ?,
			stability_status_at = ?,
			started_by = ?,
			tags = ?,
			fargate_ephemeral_storage = ?,
			updated_at = ?
		WHERE service_arn = ? AND id = ?
	`

	taskSet.UpdatedAt = time.Now()

	result, err := s.db.ExecContext(ctx, query,
		taskSet.ExternalID, taskSet.TaskDefinition, taskSet.LaunchType,
		taskSet.PlatformVersion, taskSet.PlatformFamily, taskSet.NetworkConfiguration,
		taskSet.LoadBalancers, taskSet.ServiceRegistries, taskSet.CapacityProviderStrategy,
		taskSet.Scale, taskSet.ComputedDesiredCount, taskSet.PendingCount,
		taskSet.RunningCount, taskSet.Status, taskSet.StabilityStatus,
		taskSet.StabilityStatusAt, taskSet.StartedBy, taskSet.Tags,
		taskSet.FargateEphemeralStorage, taskSet.UpdatedAt,
		taskSet.ServiceARN, taskSet.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update task set: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task set not found")
	}

	return nil
}

// Delete deletes a task set
func (s *taskSetStore) Delete(ctx context.Context, serviceARN, taskSetID string) error {
	query := `DELETE FROM task_sets WHERE service_arn = ? AND id = ?`

	result, err := s.db.ExecContext(ctx, query, serviceARN, taskSetID)
	if err != nil {
		return fmt.Errorf("failed to delete task set: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task set not found")
	}

	return nil
}

// GetByARN retrieves a task set by ARN
func (s *taskSetStore) GetByARN(ctx context.Context, arn string) (*storage.TaskSet, error) {
	query := `
		SELECT 
			id, arn, service_arn, cluster_arn, external_id,
			task_definition, launch_type, platform_version, platform_family,
			network_configuration, load_balancers, service_registries,
			capacity_provider_strategy, scale, computed_desired_count,
			pending_count, running_count, status, stability_status,
			stability_status_at, started_by, tags, fargate_ephemeral_storage,
			region, account_id, created_at, updated_at
		FROM task_sets
		WHERE arn = ?
	`

	taskSet := &storage.TaskSet{}
	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&taskSet.ID, &taskSet.ARN, &taskSet.ServiceARN, &taskSet.ClusterARN, &taskSet.ExternalID,
		&taskSet.TaskDefinition, &taskSet.LaunchType, &taskSet.PlatformVersion, &taskSet.PlatformFamily,
		&taskSet.NetworkConfiguration, &taskSet.LoadBalancers, &taskSet.ServiceRegistries,
		&taskSet.CapacityProviderStrategy, &taskSet.Scale, &taskSet.ComputedDesiredCount,
		&taskSet.PendingCount, &taskSet.RunningCount, &taskSet.Status, &taskSet.StabilityStatus,
		&taskSet.StabilityStatusAt, &taskSet.StartedBy, &taskSet.Tags, &taskSet.FargateEphemeralStorage,
		&taskSet.Region, &taskSet.AccountID, &taskSet.CreatedAt, &taskSet.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task set not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task set by ARN: %w", err)
	}

	return taskSet, nil
}

// UpdatePrimary updates the primary task set for a service
func (s *taskSetStore) UpdatePrimary(ctx context.Context, serviceARN, taskSetID string) error {
	// TODO: This would typically involve updating the service to point to this task set
	// For now, we'll just verify the task set exists
	_, err := s.Get(ctx, serviceARN, taskSetID)
	if err != nil {
		return fmt.Errorf("failed to find task set to make primary: %w", err)
	}

	// In a real implementation, this would update the service's primary task set reference
	// and potentially adjust the scale of other task sets
	return nil
}
