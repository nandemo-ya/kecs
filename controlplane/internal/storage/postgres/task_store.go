package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type taskStore struct {
	db *sql.DB
}

// Create creates a new task
func (s *taskStore) Create(ctx context.Context, task *storage.Task) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	query := `
	INSERT INTO tasks (
		id, arn, cluster_arn, task_definition_arn, container_instance_arn,
		overrides, last_status, desired_status, cpu, memory,
		containers, started_by, version, stop_code, stopped_reason,
		stopping_at, stopped_at, connectivity, connectivity_at,
		pull_started_at, pull_stopped_at, execution_stopped_at,
		created_at, started_at, launch_type, platform_version,
		platform_family, task_group, attachments, health_status,
		tags, attributes, enable_execute_command, capacity_provider_name,
		ephemeral_storage, region, account_id, pod_name, namespace,
		service_registries
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19,
		$20, $21, $22, $23, $24, $25, $26, $27, $28,
		$29, $30, $31, $32, $33, $34, $35, $36, $37,
		$38, $39, $40
	)`

	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.ARN, task.ClusterARN, task.TaskDefinitionARN,
		toNullString(task.ContainerInstanceARN),
		toNullString(task.Overrides), task.LastStatus, task.DesiredStatus,
		toNullString(task.CPU), toNullString(task.Memory),
		task.Containers, toNullString(task.StartedBy), task.Version,
		toNullString(task.StopCode), toNullString(task.StoppedReason),
		toNullTime(task.StoppingAt), toNullTime(task.StoppedAt),
		toNullString(task.Connectivity), toNullTime(task.ConnectivityAt),
		toNullTime(task.PullStartedAt), toNullTime(task.PullStoppedAt),
		toNullTime(task.ExecutionStoppedAt),
		task.CreatedAt, toNullTime(task.StartedAt),
		task.LaunchType, toNullString(task.PlatformVersion),
		toNullString(task.PlatformFamily), toNullString(task.Group),
		toNullString(task.Attachments), toNullString(task.HealthStatus),
		toNullString(task.Tags), toNullString(task.Attributes),
		task.EnableExecuteCommand, toNullString(task.CapacityProviderName),
		toNullString(task.EphemeralStorage), task.Region, task.AccountID,
		toNullString(task.PodName), toNullString(task.Namespace),
		toNullString(task.ServiceRegistries),
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// Get retrieves a task by cluster and task ID/ARN
func (s *taskStore) Get(ctx context.Context, clusterARN, taskID string) (*storage.Task, error) {
	query := `
	SELECT
		id, arn, cluster_arn, task_definition_arn, container_instance_arn,
		overrides, last_status, desired_status, cpu, memory,
		containers, started_by, version, stop_code, stopped_reason,
		stopping_at, stopped_at, connectivity, connectivity_at,
		pull_started_at, pull_stopped_at, execution_stopped_at,
		created_at, started_at, launch_type, platform_version,
		platform_family, task_group, attachments, health_status,
		tags, attributes, enable_execute_command, capacity_provider_name,
		ephemeral_storage, region, account_id, pod_name, namespace,
		service_registries
	FROM tasks
	WHERE cluster_arn = $1 AND (arn = $2 OR id = $2)`

	return scanTask(s.db.QueryRowContext(ctx, query, clusterARN, taskID))
}

// List lists tasks with filtering
func (s *taskStore) List(ctx context.Context, clusterARN string, filters storage.TaskFilters) ([]*storage.Task, error) {
	query := `
	SELECT
		id, arn, cluster_arn, task_definition_arn, container_instance_arn,
		overrides, last_status, desired_status, cpu, memory,
		containers, started_by, version, stop_code, stopped_reason,
		stopping_at, stopped_at, connectivity, connectivity_at,
		pull_started_at, pull_stopped_at, execution_stopped_at,
		created_at, started_at, launch_type, platform_version,
		platform_family, task_group, attachments, health_status,
		tags, attributes, enable_execute_command, capacity_provider_name,
		ephemeral_storage, region, account_id, pod_name, namespace,
		service_registries
	FROM tasks
	WHERE cluster_arn = $1`

	args := []interface{}{clusterARN}
	argNum := 2

	if filters.ServiceName != "" {
		query += fmt.Sprintf(" AND started_by = $%d", argNum)
		args = append(args, filters.ServiceName)
		argNum++
	}
	if filters.DesiredStatus != "" {
		query += fmt.Sprintf(" AND desired_status = $%d", argNum)
		args = append(args, filters.DesiredStatus)
		argNum++
	}
	if filters.LaunchType != "" {
		query += fmt.Sprintf(" AND launch_type = $%d", argNum)
		args = append(args, filters.LaunchType)
		argNum++
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*storage.Task
	for rows.Next() {
		task, err := scanTaskFromRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Update updates a task
func (s *taskStore) Update(ctx context.Context, task *storage.Task) error {
	query := `
	UPDATE tasks SET
		last_status = $1, desired_status = $2, containers = $3,
		version = $4, stop_code = $5, stopped_reason = $6,
		stopping_at = $7, stopped_at = $8, connectivity = $9,
		connectivity_at = $10, pull_started_at = $11, pull_stopped_at = $12,
		execution_stopped_at = $13, started_at = $14, health_status = $15,
		attachments = $16, pod_name = $17, namespace = $18
	WHERE arn = $19`

	result, err := s.db.ExecContext(ctx, query,
		task.LastStatus, task.DesiredStatus, task.Containers,
		task.Version, toNullString(task.StopCode), toNullString(task.StoppedReason),
		toNullTime(task.StoppingAt), toNullTime(task.StoppedAt),
		toNullString(task.Connectivity), toNullTime(task.ConnectivityAt),
		toNullTime(task.PullStartedAt), toNullTime(task.PullStoppedAt),
		toNullTime(task.ExecutionStoppedAt), toNullTime(task.StartedAt),
		toNullString(task.HealthStatus), toNullString(task.Attachments),
		toNullString(task.PodName), toNullString(task.Namespace),
		task.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
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

// Delete deletes a task
func (s *taskStore) Delete(ctx context.Context, clusterARN, taskID string) error {
	query := `DELETE FROM tasks WHERE cluster_arn = $1 AND (arn = $2 OR id = $2)`

	result, err := s.db.ExecContext(ctx, query, clusterARN, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
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

// GetByARNs retrieves tasks by ARNs
func (s *taskStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.Task, error) {
	if len(arns) == 0 {
		return []*storage.Task{}, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(arns))
	args := make([]interface{}, len(arns))
	for i, arn := range arns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = arn
	}

	query := fmt.Sprintf(`
	SELECT
		id, arn, cluster_arn, task_definition_arn, container_instance_arn,
		overrides, last_status, desired_status, cpu, memory,
		containers, started_by, version, stop_code, stopped_reason,
		stopping_at, stopped_at, connectivity, connectivity_at,
		pull_started_at, pull_stopped_at, execution_stopped_at,
		created_at, started_at, launch_type, platform_version,
		platform_family, task_group, attachments, health_status,
		tags, attributes, enable_execute_command, capacity_provider_name,
		ephemeral_storage, region, account_id, pod_name, namespace,
		service_registries
	FROM tasks
	WHERE arn IN (%s)`, strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by ARNs: %w", err)
	}
	defer rows.Close()

	var tasks []*storage.Task
	for rows.Next() {
		task, err := scanTaskFromRows(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// CreateOrUpdate creates a new task or updates if it already exists
func (s *taskStore) CreateOrUpdate(ctx context.Context, task *storage.Task) error {
	// Try to create first
	err := s.Create(ctx, task)
	if err == nil {
		return nil
	}

	// If already exists, update
	if err == storage.ErrResourceAlreadyExists {
		return s.Update(ctx, task)
	}

	return err
}

// DeleteOlderThan deletes tasks older than the specified time with the given status
func (s *taskStore) DeleteOlderThan(ctx context.Context, clusterARN string, before time.Time, status string) (int, error) {
	query := `DELETE FROM tasks WHERE cluster_arn = $1 AND created_at < $2`
	args := []interface{}{clusterARN, before}

	if status != "" {
		query += " AND last_status = $3"
		args = append(args, status)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old tasks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// scanTask scans a task from a row
func scanTask(row *sql.Row) (*storage.Task, error) {
	var task storage.Task
	var containerInstanceARN, overrides, cpu, memory, startedBy sql.NullString
	var stopCode, stoppedReason, connectivity, platformVersion sql.NullString
	var platformFamily, group, attachments, healthStatus sql.NullString
	var tags, attributes, capacityProviderName, ephemeralStorage sql.NullString
	var podName, namespace, serviceRegistries sql.NullString
	var stoppingAt, stoppedAt, connectivityAt, pullStartedAt sql.NullTime
	var pullStoppedAt, executionStoppedAt, startedAt sql.NullTime

	err := row.Scan(
		&task.ID, &task.ARN, &task.ClusterARN, &task.TaskDefinitionARN,
		&containerInstanceARN, &overrides, &task.LastStatus, &task.DesiredStatus,
		&cpu, &memory, &task.Containers, &startedBy, &task.Version,
		&stopCode, &stoppedReason, &stoppingAt, &stoppedAt,
		&connectivity, &connectivityAt, &pullStartedAt, &pullStoppedAt,
		&executionStoppedAt, &task.CreatedAt, &startedAt,
		&task.LaunchType, &platformVersion, &platformFamily,
		&group, &attachments, &healthStatus, &tags, &attributes,
		&task.EnableExecuteCommand, &capacityProviderName,
		&ephemeralStorage, &task.Region, &task.AccountID,
		&podName, &namespace, &serviceRegistries,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}

	// Convert null values
	task.ContainerInstanceARN = fromNullString(containerInstanceARN)
	task.Overrides = fromNullString(overrides)
	task.CPU = fromNullString(cpu)
	task.Memory = fromNullString(memory)
	task.StartedBy = fromNullString(startedBy)
	task.StopCode = fromNullString(stopCode)
	task.StoppedReason = fromNullString(stoppedReason)
	task.Connectivity = fromNullString(connectivity)
	task.PlatformVersion = fromNullString(platformVersion)
	task.PlatformFamily = fromNullString(platformFamily)
	task.Group = fromNullString(group)
	task.Attachments = fromNullString(attachments)
	task.HealthStatus = fromNullString(healthStatus)
	task.Tags = fromNullString(tags)
	task.Attributes = fromNullString(attributes)
	task.CapacityProviderName = fromNullString(capacityProviderName)
	task.EphemeralStorage = fromNullString(ephemeralStorage)
	task.PodName = fromNullString(podName)
	task.Namespace = fromNullString(namespace)
	task.ServiceRegistries = fromNullString(serviceRegistries)

	task.StoppingAt = fromNullTime(stoppingAt)
	task.StoppedAt = fromNullTime(stoppedAt)
	task.ConnectivityAt = fromNullTime(connectivityAt)
	task.PullStartedAt = fromNullTime(pullStartedAt)
	task.PullStoppedAt = fromNullTime(pullStoppedAt)
	task.ExecutionStoppedAt = fromNullTime(executionStoppedAt)
	task.StartedAt = fromNullTime(startedAt)

	return &task, nil
}

// scanTaskFromRows scans a task from rows
func scanTaskFromRows(rows *sql.Rows) (*storage.Task, error) {
	var task storage.Task
	var containerInstanceARN, overrides, cpu, memory, startedBy sql.NullString
	var stopCode, stoppedReason, connectivity, platformVersion sql.NullString
	var platformFamily, group, attachments, healthStatus sql.NullString
	var tags, attributes, capacityProviderName, ephemeralStorage sql.NullString
	var podName, namespace, serviceRegistries sql.NullString
	var stoppingAt, stoppedAt, connectivityAt, pullStartedAt sql.NullTime
	var pullStoppedAt, executionStoppedAt, startedAt sql.NullTime

	err := rows.Scan(
		&task.ID, &task.ARN, &task.ClusterARN, &task.TaskDefinitionARN,
		&containerInstanceARN, &overrides, &task.LastStatus, &task.DesiredStatus,
		&cpu, &memory, &task.Containers, &startedBy, &task.Version,
		&stopCode, &stoppedReason, &stoppingAt, &stoppedAt,
		&connectivity, &connectivityAt, &pullStartedAt, &pullStoppedAt,
		&executionStoppedAt, &task.CreatedAt, &startedAt,
		&task.LaunchType, &platformVersion, &platformFamily,
		&group, &attachments, &healthStatus, &tags, &attributes,
		&task.EnableExecuteCommand, &capacityProviderName,
		&ephemeralStorage, &task.Region, &task.AccountID,
		&podName, &namespace, &serviceRegistries,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan task row: %w", err)
	}

	// Convert null values
	task.ContainerInstanceARN = fromNullString(containerInstanceARN)
	task.Overrides = fromNullString(overrides)
	task.CPU = fromNullString(cpu)
	task.Memory = fromNullString(memory)
	task.StartedBy = fromNullString(startedBy)
	task.StopCode = fromNullString(stopCode)
	task.StoppedReason = fromNullString(stoppedReason)
	task.Connectivity = fromNullString(connectivity)
	task.PlatformVersion = fromNullString(platformVersion)
	task.PlatformFamily = fromNullString(platformFamily)
	task.Group = fromNullString(group)
	task.Attachments = fromNullString(attachments)
	task.HealthStatus = fromNullString(healthStatus)
	task.Tags = fromNullString(tags)
	task.Attributes = fromNullString(attributes)
	task.CapacityProviderName = fromNullString(capacityProviderName)
	task.EphemeralStorage = fromNullString(ephemeralStorage)
	task.PodName = fromNullString(podName)
	task.Namespace = fromNullString(namespace)
	task.ServiceRegistries = fromNullString(serviceRegistries)

	task.StoppingAt = fromNullTime(stoppingAt)
	task.StoppedAt = fromNullTime(stoppedAt)
	task.ConnectivityAt = fromNullTime(connectivityAt)
	task.PullStartedAt = fromNullTime(pullStartedAt)
	task.PullStoppedAt = fromNullTime(pullStoppedAt)
	task.ExecutionStoppedAt = fromNullTime(executionStoppedAt)
	task.StartedAt = fromNullTime(startedAt)

	return &task, nil
}
