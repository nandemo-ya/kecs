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

type containerInstanceStore struct {
	db *sql.DB
}

// Register registers a new container instance
func (s *containerInstanceStore) Register(ctx context.Context, instance *storage.ContainerInstance) error {
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}

	now := time.Now()
	if instance.RegisteredAt.IsZero() {
		instance.RegisteredAt = now
	}
	instance.UpdatedAt = now

	query := `
	INSERT INTO container_instances (
		id, arn, cluster_arn, ec2_instance_id, status, status_reason,
		agent_connected, agent_update_status, running_tasks_count,
		pending_tasks_count, version, version_info, registered_resources,
		remaining_resources, attributes, attachments, tags,
		capacity_provider_name, health_status, region, account_id,
		registered_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19,
		$20, $21, $22, $23
	)`

	_, err := s.db.ExecContext(ctx, query,
		instance.ID, instance.ARN, instance.ClusterARN, instance.EC2InstanceID,
		instance.Status, toNullString(instance.StatusReason),
		instance.AgentConnected, toNullString(instance.AgentUpdateStatus),
		instance.RunningTasksCount, instance.PendingTasksCount,
		instance.Version, toNullString(instance.VersionInfo),
		toNullString(instance.RegisteredResources), toNullString(instance.RemainingResources),
		toNullString(instance.Attributes), toNullString(instance.Attachments),
		toNullString(instance.Tags), toNullString(instance.CapacityProviderName),
		toNullString(instance.HealthStatus), toNullString(instance.Region),
		toNullString(instance.AccountID), instance.RegisteredAt, instance.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to register container instance: %w", err)
	}

	return nil
}

// Get retrieves a container instance by ARN
func (s *containerInstanceStore) Get(ctx context.Context, arn string) (*storage.ContainerInstance, error) {
	query := `
	SELECT
		id, arn, cluster_arn, ec2_instance_id, status, status_reason,
		agent_connected, agent_update_status, running_tasks_count,
		pending_tasks_count, version, version_info, registered_resources,
		remaining_resources, attributes, attachments, tags,
		capacity_provider_name, health_status, region, account_id,
		registered_at, updated_at, deregistered_at
	FROM container_instances
	WHERE arn = $1`

	var instance storage.ContainerInstance
	var statusReason, agentUpdateStatus, versionInfo sql.NullString
	var registeredResources, remainingResources, attributes, attachments sql.NullString
	var tags, capacityProviderName, healthStatus, region, accountID sql.NullString
	var deregisteredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&instance.ID, &instance.ARN, &instance.ClusterARN, &instance.EC2InstanceID,
		&instance.Status, &statusReason, &instance.AgentConnected, &agentUpdateStatus,
		&instance.RunningTasksCount, &instance.PendingTasksCount,
		&instance.Version, &versionInfo, &registeredResources, &remainingResources,
		&attributes, &attachments, &tags, &capacityProviderName, &healthStatus,
		&region, &accountID, &instance.RegisteredAt, &instance.UpdatedAt, &deregisteredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get container instance: %w", err)
	}

	// Convert null values
	instance.StatusReason = fromNullString(statusReason)
	instance.AgentUpdateStatus = fromNullString(agentUpdateStatus)
	instance.VersionInfo = fromNullString(versionInfo)
	instance.RegisteredResources = fromNullString(registeredResources)
	instance.RemainingResources = fromNullString(remainingResources)
	instance.Attributes = fromNullString(attributes)
	instance.Attachments = fromNullString(attachments)
	instance.Tags = fromNullString(tags)
	instance.CapacityProviderName = fromNullString(capacityProviderName)
	instance.HealthStatus = fromNullString(healthStatus)
	instance.Region = fromNullString(region)
	instance.AccountID = fromNullString(accountID)
	instance.DeregisteredAt = fromNullTime(deregisteredAt)

	return &instance, nil
}

// ListWithPagination retrieves container instances with filtering and pagination
func (s *containerInstanceStore) ListWithPagination(ctx context.Context, cluster string, filters storage.ContainerInstanceFilters, limit int, nextToken string) ([]*storage.ContainerInstance, string, error) {
	// Parse the next token to get offset
	offset := 0
	if nextToken != "" {
		if _, err := fmt.Sscanf(nextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	query := `
	SELECT
		id, arn, cluster_arn, ec2_instance_id, status, status_reason,
		agent_connected, agent_update_status, running_tasks_count,
		pending_tasks_count, version, version_info, registered_resources,
		remaining_resources, attributes, attachments, tags,
		capacity_provider_name, health_status, region, account_id,
		registered_at, updated_at, deregistered_at
	FROM container_instances
	WHERE cluster_arn = $1`

	args := []interface{}{cluster}
	argNum := 2

	if filters.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, filters.Status)
		argNum++
	}

	// Handle filter string (could be instance ID, ARN, or attribute)
	if filters.Filter != "" {
		query += fmt.Sprintf(" AND (ec2_instance_id = $%d OR arn LIKE $%d OR attributes LIKE $%d)", argNum, argNum, argNum)
		args = append(args, filters.Filter)
		argNum++
	}

	query += " ORDER BY registered_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list container instances: %w", err)
	}
	defer rows.Close()

	var instances []*storage.ContainerInstance
	for rows.Next() {
		var instance storage.ContainerInstance
		var statusReason, agentUpdateStatus, versionInfo sql.NullString
		var registeredResources, remainingResources, attributes, attachments sql.NullString
		var tags, capacityProviderName, healthStatus, region, accountID sql.NullString
		var deregisteredAt sql.NullTime

		err := rows.Scan(
			&instance.ID, &instance.ARN, &instance.ClusterARN, &instance.EC2InstanceID,
			&instance.Status, &statusReason, &instance.AgentConnected, &agentUpdateStatus,
			&instance.RunningTasksCount, &instance.PendingTasksCount,
			&instance.Version, &versionInfo, &registeredResources, &remainingResources,
			&attributes, &attachments, &tags, &capacityProviderName, &healthStatus,
			&region, &accountID, &instance.RegisteredAt, &instance.UpdatedAt, &deregisteredAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan container instance: %w", err)
		}

		// Convert null values
		instance.StatusReason = fromNullString(statusReason)
		instance.AgentUpdateStatus = fromNullString(agentUpdateStatus)
		instance.VersionInfo = fromNullString(versionInfo)
		instance.RegisteredResources = fromNullString(registeredResources)
		instance.RemainingResources = fromNullString(remainingResources)
		instance.Attributes = fromNullString(attributes)
		instance.Attachments = fromNullString(attachments)
		instance.Tags = fromNullString(tags)
		instance.CapacityProviderName = fromNullString(capacityProviderName)
		instance.HealthStatus = fromNullString(healthStatus)
		instance.Region = fromNullString(region)
		instance.AccountID = fromNullString(accountID)
		instance.DeregisteredAt = fromNullTime(deregisteredAt)

		instances = append(instances, &instance)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if limit > 0 && len(instances) == limit {
		newNextToken = fmt.Sprintf("%d", offset+limit)
	}

	return instances, newNextToken, nil
}

// Update updates a container instance
func (s *containerInstanceStore) Update(ctx context.Context, instance *storage.ContainerInstance) error {
	instance.UpdatedAt = time.Now()

	query := `
	UPDATE container_instances SET
		status = $1, status_reason = $2, agent_connected = $3,
		agent_update_status = $4, running_tasks_count = $5,
		pending_tasks_count = $6, version = $7, version_info = $8,
		registered_resources = $9, remaining_resources = $10,
		attributes = $11, attachments = $12, tags = $13,
		capacity_provider_name = $14, health_status = $15,
		updated_at = $16
	WHERE arn = $17`

	result, err := s.db.ExecContext(ctx, query,
		instance.Status, toNullString(instance.StatusReason),
		instance.AgentConnected, toNullString(instance.AgentUpdateStatus),
		instance.RunningTasksCount, instance.PendingTasksCount,
		instance.Version, toNullString(instance.VersionInfo),
		toNullString(instance.RegisteredResources), toNullString(instance.RemainingResources),
		toNullString(instance.Attributes), toNullString(instance.Attachments),
		toNullString(instance.Tags), toNullString(instance.CapacityProviderName),
		toNullString(instance.HealthStatus), instance.UpdatedAt, instance.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update container instance: %w", err)
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

// Deregister deregisters a container instance
func (s *containerInstanceStore) Deregister(ctx context.Context, arn string) error {
	now := time.Now()

	query := `
	UPDATE container_instances SET
		status = 'INACTIVE',
		deregistered_at = $1,
		updated_at = $2
	WHERE arn = $3`

	result, err := s.db.ExecContext(ctx, query, now, now, arn)
	if err != nil {
		return fmt.Errorf("failed to deregister container instance: %w", err)
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

// GetByARNs retrieves container instances by ARNs
func (s *containerInstanceStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.ContainerInstance, error) {
	if len(arns) == 0 {
		return []*storage.ContainerInstance{}, nil
	}

	query := `
	SELECT
		id, arn, cluster_arn, ec2_instance_id, status, status_reason,
		agent_connected, agent_update_status, running_tasks_count,
		pending_tasks_count, version, version_info, registered_resources,
		remaining_resources, attributes, attachments, tags,
		capacity_provider_name, health_status, region, account_id,
		registered_at, updated_at, deregistered_at
	FROM container_instances
	WHERE arn = ANY($1)
	ORDER BY registered_at DESC`

	rows, err := s.db.QueryContext(ctx, query, pq.Array(arns))
	if err != nil {
		return nil, fmt.Errorf("failed to get container instances by ARNs: %w", err)
	}
	defer rows.Close()

	var instances []*storage.ContainerInstance
	for rows.Next() {
		var instance storage.ContainerInstance
		var statusReason, agentUpdateStatus, versionInfo sql.NullString
		var registeredResources, remainingResources, attributes, attachments sql.NullString
		var tags, capacityProviderName, healthStatus, region, accountID sql.NullString
		var deregisteredAt sql.NullTime

		err := rows.Scan(
			&instance.ID, &instance.ARN, &instance.ClusterARN, &instance.EC2InstanceID,
			&instance.Status, &statusReason, &instance.AgentConnected, &agentUpdateStatus,
			&instance.RunningTasksCount, &instance.PendingTasksCount,
			&instance.Version, &versionInfo, &registeredResources, &remainingResources,
			&attributes, &attachments, &tags, &capacityProviderName, &healthStatus,
			&region, &accountID, &instance.RegisteredAt, &instance.UpdatedAt, &deregisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan container instance: %w", err)
		}

		// Convert null values
		instance.StatusReason = fromNullString(statusReason)
		instance.AgentUpdateStatus = fromNullString(agentUpdateStatus)
		instance.VersionInfo = fromNullString(versionInfo)
		instance.RegisteredResources = fromNullString(registeredResources)
		instance.RemainingResources = fromNullString(remainingResources)
		instance.Attributes = fromNullString(attributes)
		instance.Attachments = fromNullString(attachments)
		instance.Tags = fromNullString(tags)
		instance.CapacityProviderName = fromNullString(capacityProviderName)
		instance.HealthStatus = fromNullString(healthStatus)
		instance.Region = fromNullString(region)
		instance.AccountID = fromNullString(accountID)
		instance.DeregisteredAt = fromNullTime(deregisteredAt)

		instances = append(instances, &instance)
	}

	return instances, nil
}

// DeleteStale deletes container instances that have been inactive before the specified time
func (s *containerInstanceStore) DeleteStale(ctx context.Context, clusterARN string, before time.Time) (int, error) {
	query := `
	DELETE FROM container_instances
	WHERE cluster_arn = $1
	AND status = 'INACTIVE'
	AND deregistered_at < $2`

	result, err := s.db.ExecContext(ctx, query, clusterARN, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete stale container instances: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}
