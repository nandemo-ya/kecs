package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type taskSetStore struct {
	db *sql.DB
}

// Create creates a new task set
func (s *taskSetStore) Create(ctx context.Context, taskSet *storage.TaskSet) error {
	if taskSet.ID == "" {
		taskSet.ID = uuid.New().String()
	}

	now := time.Now()
	if taskSet.CreatedAt.IsZero() {
		taskSet.CreatedAt = now
	}
	taskSet.UpdatedAt = now

	query := `
	INSERT INTO task_sets (
		id, arn, service_arn, cluster_arn, external_id,
		task_definition, launch_type, platform_version, platform_family,
		network_configuration, load_balancers, service_registries,
		capacity_provider_strategy, scale, computed_desired_count,
		pending_count, running_count, status, stability_status,
		stability_status_at, started_by, tags, fargate_ephemeral_storage,
		region, account_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		$21, $22, $23, $24, $25, $26, $27
	)`

	_, err := s.db.ExecContext(ctx, query,
		taskSet.ID, taskSet.ARN, taskSet.ServiceARN, taskSet.ClusterARN,
		toNullString(taskSet.ExternalID), taskSet.TaskDefinition,
		toNullString(taskSet.LaunchType), toNullString(taskSet.PlatformVersion),
		toNullString(taskSet.PlatformFamily), toNullString(taskSet.NetworkConfiguration),
		toNullString(taskSet.LoadBalancers), toNullString(taskSet.ServiceRegistries),
		toNullString(taskSet.CapacityProviderStrategy), toNullString(taskSet.Scale),
		taskSet.ComputedDesiredCount, taskSet.PendingCount, taskSet.RunningCount,
		taskSet.Status, taskSet.StabilityStatus, toNullTime(taskSet.StabilityStatusAt),
		toNullString(taskSet.StartedBy), toNullString(taskSet.Tags),
		toNullString(taskSet.FargateEphemeralStorage),
		toNullString(taskSet.Region), toNullString(taskSet.AccountID),
		taskSet.CreatedAt, taskSet.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
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
	WHERE service_arn = $1 AND id = $2`

	return s.scanTaskSet(ctx, query, serviceARN, taskSetID)
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
	WHERE arn = $1`

	return s.scanTaskSet(ctx, query, arn)
}

// List retrieves task sets for a service
func (s *taskSetStore) List(ctx context.Context, serviceARN string, taskSetIDs []string) ([]*storage.TaskSet, error) {
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
	WHERE service_arn = $1`

	args := []interface{}{serviceARN}

	if len(taskSetIDs) > 0 {
		query += " AND id = ANY($2)"
		args = append(args, pq.Array(taskSetIDs))
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list task sets: %w", err)
	}
	defer rows.Close()

	return s.scanTaskSets(rows)
}

// Update updates a task set
func (s *taskSetStore) Update(ctx context.Context, taskSet *storage.TaskSet) error {
	taskSet.UpdatedAt = time.Now()

	query := `
	UPDATE task_sets SET
		external_id = $1, task_definition = $2, launch_type = $3,
		platform_version = $4, platform_family = $5,
		network_configuration = $6, load_balancers = $7,
		service_registries = $8, capacity_provider_strategy = $9,
		scale = $10, computed_desired_count = $11,
		pending_count = $12, running_count = $13, status = $14,
		stability_status = $15, stability_status_at = $16,
		started_by = $17, tags = $18, fargate_ephemeral_storage = $19,
		updated_at = $20
	WHERE service_arn = $21 AND id = $22`

	result, err := s.db.ExecContext(ctx, query,
		toNullString(taskSet.ExternalID), taskSet.TaskDefinition,
		toNullString(taskSet.LaunchType), toNullString(taskSet.PlatformVersion),
		toNullString(taskSet.PlatformFamily), toNullString(taskSet.NetworkConfiguration),
		toNullString(taskSet.LoadBalancers), toNullString(taskSet.ServiceRegistries),
		toNullString(taskSet.CapacityProviderStrategy), toNullString(taskSet.Scale),
		taskSet.ComputedDesiredCount, taskSet.PendingCount, taskSet.RunningCount,
		taskSet.Status, taskSet.StabilityStatus, toNullTime(taskSet.StabilityStatusAt),
		toNullString(taskSet.StartedBy), toNullString(taskSet.Tags),
		toNullString(taskSet.FargateEphemeralStorage),
		taskSet.UpdatedAt, taskSet.ServiceARN, taskSet.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update task set: %w", err)
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

// Delete deletes a task set
func (s *taskSetStore) Delete(ctx context.Context, serviceARN, taskSetID string) error {
	query := `
	DELETE FROM task_sets
	WHERE service_arn = $1 AND id = $2`

	result, err := s.db.ExecContext(ctx, query, serviceARN, taskSetID)
	if err != nil {
		return fmt.Errorf("failed to delete task set: %w", err)
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

// UpdatePrimary updates the primary task set for a service
func (s *taskSetStore) UpdatePrimary(ctx context.Context, serviceARN, taskSetID string) error {
	// First, unset all task sets for this service as non-primary
	// This would typically involve updating a 'is_primary' field, but since
	// it's not defined in the TaskSet struct, we'll handle it through status
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update all task sets for the service to non-primary status
	updateAllQuery := `
	UPDATE task_sets
	SET status = CASE
		WHEN status = 'PRIMARY' THEN 'ACTIVE'
		ELSE status
	END
	WHERE service_arn = $1`

	if _, err := tx.ExecContext(ctx, updateAllQuery, serviceARN); err != nil {
		return fmt.Errorf("failed to update task sets: %w", err)
	}

	// Set the specified task set as primary
	setPrimaryQuery := `
	UPDATE task_sets
	SET status = 'PRIMARY', updated_at = $1
	WHERE service_arn = $2 AND id = $3`

	result, err := tx.ExecContext(ctx, setPrimaryQuery, time.Now(), serviceARN, taskSetID)
	if err != nil {
		return fmt.Errorf("failed to set primary task set: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return storage.ErrResourceNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteOrphaned deletes task sets that no longer have an associated service
func (s *taskSetStore) DeleteOrphaned(ctx context.Context, clusterARN string) (int, error) {
	query := `
	DELETE FROM task_sets
	WHERE cluster_arn = $1
	AND service_arn NOT IN (
		SELECT arn FROM services WHERE cluster_arn = $1
	)`

	result, err := s.db.ExecContext(ctx, query, clusterARN)
	if err != nil {
		return 0, fmt.Errorf("failed to delete orphaned task sets: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// Helper function to scan a single task set
func (s *taskSetStore) scanTaskSet(ctx context.Context, query string, args ...interface{}) (*storage.TaskSet, error) {
	var ts storage.TaskSet
	var externalID, launchType, platformVersion, platformFamily sql.NullString
	var networkConfig, loadBalancers, serviceRegistries sql.NullString
	var capacityProviderStrategy, scale, startedBy, tags sql.NullString
	var fargateEphemeralStorage, region, accountID sql.NullString
	var stabilityStatusAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&ts.ID, &ts.ARN, &ts.ServiceARN, &ts.ClusterARN, &externalID,
		&ts.TaskDefinition, &launchType, &platformVersion, &platformFamily,
		&networkConfig, &loadBalancers, &serviceRegistries,
		&capacityProviderStrategy, &scale, &ts.ComputedDesiredCount,
		&ts.PendingCount, &ts.RunningCount, &ts.Status, &ts.StabilityStatus,
		&stabilityStatusAt, &startedBy, &tags, &fargateEphemeralStorage,
		&region, &accountID, &ts.CreatedAt, &ts.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to scan task set: %w", err)
	}

	// Convert null values
	ts.ExternalID = fromNullString(externalID)
	ts.LaunchType = fromNullString(launchType)
	ts.PlatformVersion = fromNullString(platformVersion)
	ts.PlatformFamily = fromNullString(platformFamily)
	ts.NetworkConfiguration = fromNullString(networkConfig)
	ts.LoadBalancers = fromNullString(loadBalancers)
	ts.ServiceRegistries = fromNullString(serviceRegistries)
	ts.CapacityProviderStrategy = fromNullString(capacityProviderStrategy)
	ts.Scale = fromNullString(scale)
	ts.StartedBy = fromNullString(startedBy)
	ts.Tags = fromNullString(tags)
	ts.FargateEphemeralStorage = fromNullString(fargateEphemeralStorage)
	ts.Region = fromNullString(region)
	ts.AccountID = fromNullString(accountID)
	ts.StabilityStatusAt = fromNullTime(stabilityStatusAt)

	return &ts, nil
}

// Helper function to scan multiple task sets
func (s *taskSetStore) scanTaskSets(rows *sql.Rows) ([]*storage.TaskSet, error) {
	var taskSets []*storage.TaskSet

	for rows.Next() {
		var ts storage.TaskSet
		var externalID, launchType, platformVersion, platformFamily sql.NullString
		var networkConfig, loadBalancers, serviceRegistries sql.NullString
		var capacityProviderStrategy, scale, startedBy, tags sql.NullString
		var fargateEphemeralStorage, region, accountID sql.NullString
		var stabilityStatusAt sql.NullTime

		err := rows.Scan(
			&ts.ID, &ts.ARN, &ts.ServiceARN, &ts.ClusterARN, &externalID,
			&ts.TaskDefinition, &launchType, &platformVersion, &platformFamily,
			&networkConfig, &loadBalancers, &serviceRegistries,
			&capacityProviderStrategy, &scale, &ts.ComputedDesiredCount,
			&ts.PendingCount, &ts.RunningCount, &ts.Status, &ts.StabilityStatus,
			&stabilityStatusAt, &startedBy, &tags, &fargateEphemeralStorage,
			&region, &accountID, &ts.CreatedAt, &ts.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task set row: %w", err)
		}

		// Convert null values
		ts.ExternalID = fromNullString(externalID)
		ts.LaunchType = fromNullString(launchType)
		ts.PlatformVersion = fromNullString(platformVersion)
		ts.PlatformFamily = fromNullString(platformFamily)
		ts.NetworkConfiguration = fromNullString(networkConfig)
		ts.LoadBalancers = fromNullString(loadBalancers)
		ts.ServiceRegistries = fromNullString(serviceRegistries)
		ts.CapacityProviderStrategy = fromNullString(capacityProviderStrategy)
		ts.Scale = fromNullString(scale)
		ts.StartedBy = fromNullString(startedBy)
		ts.Tags = fromNullString(tags)
		ts.FargateEphemeralStorage = fromNullString(fargateEphemeralStorage)
		ts.Region = fromNullString(region)
		ts.AccountID = fromNullString(accountID)
		ts.StabilityStatusAt = fromNullTime(stabilityStatusAt)

		taskSets = append(taskSets, &ts)
	}

	return taskSets, nil
}
