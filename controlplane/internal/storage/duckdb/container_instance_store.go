package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// containerInstanceStore implements storage.ContainerInstanceStore
type containerInstanceStore struct {
	db *sql.DB
}

// Register creates a new container instance
func (s *containerInstanceStore) Register(ctx context.Context, instance *storage.ContainerInstance) error {
	// Generate ID if not provided
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	instance.RegisteredAt = now
	instance.UpdatedAt = now

	query := `
		INSERT INTO container_instances (
			id, arn, cluster_arn, ec2_instance_id, status, status_reason,
			agent_connected, agent_update_status, running_tasks_count, pending_tasks_count,
			version, version_info, registered_resources, remaining_resources,
			attributes, attachments, tags, capacity_provider_name, health_status,
			region, account_id, registered_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19,
			$20, $21, $22, $23
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		instance.ID, instance.ARN, instance.ClusterARN, instance.EC2InstanceID,
		instance.Status, instance.StatusReason, instance.AgentConnected,
		instance.AgentUpdateStatus, instance.RunningTasksCount, instance.PendingTasksCount,
		instance.Version, instance.VersionInfo, instance.RegisteredResources,
		instance.RemainingResources, instance.Attributes, instance.Attachments,
		instance.Tags, instance.CapacityProviderName, instance.HealthStatus,
		instance.Region, instance.AccountID, instance.RegisteredAt, instance.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to register container instance: %w", err)
	}

	return nil
}

// Get retrieves a container instance by ARN
func (s *containerInstanceStore) Get(ctx context.Context, arn string) (*storage.ContainerInstance, error) {
	query := `
		SELECT 
			id, arn, cluster_arn, ec2_instance_id, status, status_reason,
			agent_connected, agent_update_status, running_tasks_count, pending_tasks_count,
			version, version_info, registered_resources, remaining_resources,
			attributes, attachments, tags, capacity_provider_name, health_status,
			region, account_id, registered_at, updated_at, deregistered_at
		FROM container_instances
		WHERE arn = $1
	`

	instance := &storage.ContainerInstance{}
	var statusReason, agentUpdateStatus, versionInfo, registeredResources, remainingResources sql.NullString
	var attributes, attachments, tags, capacityProviderName, healthStatus sql.NullString
	var deregisteredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, arn).Scan(
		&instance.ID, &instance.ARN, &instance.ClusterARN, &instance.EC2InstanceID,
		&instance.Status, &statusReason, &instance.AgentConnected,
		&agentUpdateStatus, &instance.RunningTasksCount, &instance.PendingTasksCount,
		&instance.Version, &versionInfo, &registeredResources, &remainingResources,
		&attributes, &attachments, &tags, &capacityProviderName, &healthStatus,
		&instance.Region, &instance.AccountID, &instance.RegisteredAt,
		&instance.UpdatedAt, &deregisteredAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("container instance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get container instance: %w", err)
	}

	// Handle nullable fields
	instance.StatusReason = statusReason.String
	instance.AgentUpdateStatus = agentUpdateStatus.String
	instance.VersionInfo = versionInfo.String
	instance.RegisteredResources = registeredResources.String
	instance.RemainingResources = remainingResources.String
	instance.Attributes = attributes.String
	instance.Attachments = attachments.String
	instance.Tags = tags.String
	instance.CapacityProviderName = capacityProviderName.String
	instance.HealthStatus = healthStatus.String
	if deregisteredAt.Valid {
		instance.DeregisteredAt = &deregisteredAt.Time
	}

	return instance, nil
}

// ListWithPagination retrieves container instances with pagination
func (s *containerInstanceStore) ListWithPagination(ctx context.Context, cluster string, filters storage.ContainerInstanceFilters, limit int, nextToken string) ([]*storage.ContainerInstance, string, error) {
	// Build the query
	query := `
		SELECT 
			id, arn, cluster_arn, ec2_instance_id, status, status_reason,
			agent_connected, agent_update_status, running_tasks_count, pending_tasks_count,
			version, version_info, registered_resources, remaining_resources,
			attributes, attachments, tags, capacity_provider_name, health_status,
			region, account_id, registered_at, updated_at, deregistered_at
		FROM container_instances
		WHERE cluster_arn = $1
	`

	args := []interface{}{cluster}
	argCount := 1

	// Add status filter if specified
	if filters.Status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filters.Status)
	}

	// Add pagination
	if nextToken != "" {
		argCount++
		query += fmt.Sprintf(" AND id > $%d", argCount)
		args = append(args, nextToken)
	}

	// Order by ID for consistent pagination
	query += " ORDER BY id"

	// Fetch one extra to determine if there are more pages
	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list container instances: %w", err)
	}
	defer rows.Close()

	instances := make([]*storage.ContainerInstance, 0, limit)
	var newNextToken string

	for rows.Next() {
		instance := &storage.ContainerInstance{}
		var statusReason, agentUpdateStatus, versionInfo, registeredResources, remainingResources sql.NullString
		var attributes, attachments, tags, capacityProviderName, healthStatus sql.NullString
		var deregisteredAt sql.NullTime

		err := rows.Scan(
			&instance.ID, &instance.ARN, &instance.ClusterARN, &instance.EC2InstanceID,
			&instance.Status, &statusReason, &instance.AgentConnected,
			&agentUpdateStatus, &instance.RunningTasksCount, &instance.PendingTasksCount,
			&instance.Version, &versionInfo, &registeredResources, &remainingResources,
			&attributes, &attachments, &tags, &capacityProviderName, &healthStatus,
			&instance.Region, &instance.AccountID, &instance.RegisteredAt,
			&instance.UpdatedAt, &deregisteredAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan container instance: %w", err)
		}

		// Handle nullable fields
		instance.StatusReason = statusReason.String
		instance.AgentUpdateStatus = agentUpdateStatus.String
		instance.VersionInfo = versionInfo.String
		instance.RegisteredResources = registeredResources.String
		instance.RemainingResources = remainingResources.String
		instance.Attributes = attributes.String
		instance.Attachments = attachments.String
		instance.Tags = tags.String
		instance.CapacityProviderName = capacityProviderName.String
		instance.HealthStatus = healthStatus.String
		if deregisteredAt.Valid {
			instance.DeregisteredAt = &deregisteredAt.Time
		}

		// If we've reached the limit, use this as the next token
		if len(instances) >= limit {
			newNextToken = instance.ID
			break
		}

		instances = append(instances, instance)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to iterate container instances: %w", err)
	}

	return instances, newNextToken, nil
}

// Update updates a container instance
func (s *containerInstanceStore) Update(ctx context.Context, instance *storage.ContainerInstance) error {
	instance.UpdatedAt = time.Now()

	query := `
		UPDATE container_instances SET
			status = $2, status_reason = $3, agent_connected = $4,
			agent_update_status = $5, running_tasks_count = $6, pending_tasks_count = $7,
			version = $8, version_info = $9, registered_resources = $10,
			remaining_resources = $11, attributes = $12, attachments = $13,
			tags = $14, capacity_provider_name = $15, health_status = $16,
			updated_at = $17, deregistered_at = $18
		WHERE arn = $1
	`

	result, err := s.db.ExecContext(ctx, query,
		instance.ARN, instance.Status, instance.StatusReason,
		instance.AgentConnected, instance.AgentUpdateStatus,
		instance.RunningTasksCount, instance.PendingTasksCount,
		instance.Version, instance.VersionInfo, instance.RegisteredResources,
		instance.RemainingResources, instance.Attributes, instance.Attachments,
		instance.Tags, instance.CapacityProviderName, instance.HealthStatus,
		instance.UpdatedAt, instance.DeregisteredAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update container instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("container instance not found")
	}

	return nil
}

// Deregister marks a container instance as deregistered
func (s *containerInstanceStore) Deregister(ctx context.Context, arn string) error {
	now := time.Now()

	query := `
		UPDATE container_instances SET
			status = 'INACTIVE',
			deregistered_at = $2,
			updated_at = $3
		WHERE arn = $1
	`

	result, err := s.db.ExecContext(ctx, query, arn, now, now)
	if err != nil {
		return fmt.Errorf("failed to deregister container instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("container instance not found")
	}

	return nil
}

// GetByARNs retrieves multiple container instances by their ARNs
func (s *containerInstanceStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.ContainerInstance, error) {
	if len(arns) == 0 {
		return []*storage.ContainerInstance{}, nil
	}

	// Build query with placeholders
	query := `
		SELECT 
			id, arn, cluster_arn, ec2_instance_id, status, status_reason,
			agent_connected, agent_update_status, running_tasks_count, pending_tasks_count,
			version, version_info, registered_resources, remaining_resources,
			attributes, attachments, tags, capacity_provider_name, health_status,
			region, account_id, registered_at, updated_at, deregistered_at
		FROM container_instances
		WHERE arn IN (`

	args := make([]interface{}, len(arns))
	for i, arn := range arns {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", i+1)
		args[i] = arn
	}
	query += ")"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get container instances by ARNs: %w", err)
	}
	defer rows.Close()

	instances := make([]*storage.ContainerInstance, 0, len(arns))

	for rows.Next() {
		instance := &storage.ContainerInstance{}
		var statusReason, agentUpdateStatus, versionInfo, registeredResources, remainingResources sql.NullString
		var attributes, attachments, tags, capacityProviderName, healthStatus sql.NullString
		var deregisteredAt sql.NullTime

		err := rows.Scan(
			&instance.ID, &instance.ARN, &instance.ClusterARN, &instance.EC2InstanceID,
			&instance.Status, &statusReason, &instance.AgentConnected,
			&agentUpdateStatus, &instance.RunningTasksCount, &instance.PendingTasksCount,
			&instance.Version, &versionInfo, &registeredResources, &remainingResources,
			&attributes, &attachments, &tags, &capacityProviderName, &healthStatus,
			&instance.Region, &instance.AccountID, &instance.RegisteredAt,
			&instance.UpdatedAt, &deregisteredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan container instance: %w", err)
		}

		// Handle nullable fields
		instance.StatusReason = statusReason.String
		instance.AgentUpdateStatus = agentUpdateStatus.String
		instance.VersionInfo = versionInfo.String
		instance.RegisteredResources = registeredResources.String
		instance.RemainingResources = remainingResources.String
		instance.Attributes = attributes.String
		instance.Attachments = attachments.String
		instance.Tags = tags.String
		instance.CapacityProviderName = capacityProviderName.String
		instance.HealthStatus = healthStatus.String
		if deregisteredAt.Valid {
			instance.DeregisteredAt = &deregisteredAt.Time
		}

		instances = append(instances, instance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate container instances: %w", err)
	}

	return instances, nil
}
