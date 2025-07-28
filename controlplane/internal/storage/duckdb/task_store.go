package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type taskStore struct {
	db *sql.DB
}

func NewTaskStore(db *sql.DB) storage.TaskStore {
	return &taskStore{db: db}
}

func (s *taskStore) Create(ctx context.Context, task *storage.Task) error {
	query := `
		INSERT INTO tasks (
			id, arn, cluster_arn, task_definition_arn, container_instance_arn,
			overrides, last_status, desired_status, cpu, memory, containers,
			started_by, version, stop_code, stopped_reason, stopping_at,
			stopped_at, connectivity, connectivity_at, pull_started_at,
			pull_stopped_at, execution_stopped_at, created_at, started_at,
			launch_type, platform_version, platform_family, task_group,
			attachments, health_status, tags, attributes, enable_execute_command,
			capacity_provider_name, ephemeral_storage, region, account_id,
			pod_name, namespace
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)`

	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.ARN, task.ClusterARN, task.TaskDefinitionARN,
		nullString(task.ContainerInstanceARN), nullString(task.Overrides),
		task.LastStatus, task.DesiredStatus, nullString(task.CPU),
		nullString(task.Memory), task.Containers, nullString(task.StartedBy),
		task.Version, nullString(task.StopCode), nullString(task.StoppedReason),
		nullTime(task.StoppingAt), nullTime(task.StoppedAt),
		nullString(task.Connectivity), nullTime(task.ConnectivityAt),
		nullTime(task.PullStartedAt), nullTime(task.PullStoppedAt),
		nullTime(task.ExecutionStoppedAt), task.CreatedAt,
		nullTime(task.StartedAt), task.LaunchType,
		nullString(task.PlatformVersion), nullString(task.PlatformFamily),
		nullString(task.Group), nullString(task.Attachments),
		nullString(task.HealthStatus), nullString(task.Tags),
		nullString(task.Attributes), task.EnableExecuteCommand,
		nullString(task.CapacityProviderName), nullString(task.EphemeralStorage),
		task.Region, task.AccountID, nullString(task.PodName),
		nullString(task.Namespace),
	)

	return err
}

func (s *taskStore) Get(ctx context.Context, cluster, taskID string) (*storage.Task, error) {
	// Handle both short task ID and full ARN
	var query string
	var args []interface{}

	if strings.Contains(taskID, "arn:aws:ecs:") {
		// Full ARN provided
		query = `SELECT * FROM tasks WHERE arn = ?`
		args = []interface{}{taskID}
	} else {
		// Short ID provided - need cluster context
		query = `SELECT * FROM tasks WHERE cluster_arn = ? AND (id = ? OR arn LIKE ?)`
		args = []interface{}{cluster, taskID, fmt.Sprintf("%%/%s", taskID)}
	}

	var task storage.Task
	var stoppingAt, stoppedAt, connectivityAt, pullStartedAt, pullStoppedAt sql.NullTime
	var executionStoppedAt, startedAt sql.NullTime
	var containerInstanceARN, overrides, cpu, memory, startedBy sql.NullString
	var stopCode, stoppedReason, connectivity, platformVersion sql.NullString
	var platformFamily, group, attachments, healthStatus, tags sql.NullString
	var attributes, capacityProviderName, ephemeralStorage sql.NullString
	var podName, namespace sql.NullString

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&task.ID, &task.ARN, &task.ClusterARN, &task.TaskDefinitionARN,
		&containerInstanceARN, &overrides, &task.LastStatus,
		&task.DesiredStatus, &cpu, &memory, &task.Containers,
		&startedBy, &task.Version, &stopCode, &stoppedReason,
		&stoppingAt, &stoppedAt, &connectivity, &connectivityAt,
		&pullStartedAt, &pullStoppedAt, &executionStoppedAt,
		&task.CreatedAt, &startedAt, &task.LaunchType,
		&platformVersion, &platformFamily, &group, &attachments,
		&healthStatus, &tags, &attributes, &task.EnableExecuteCommand,
		&capacityProviderName, &ephemeralStorage, &task.Region,
		&task.AccountID, &podName, &namespace,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Set nullable fields
	task.ContainerInstanceARN = containerInstanceARN.String
	task.Overrides = overrides.String
	task.CPU = cpu.String
	task.Memory = memory.String
	task.StartedBy = startedBy.String
	task.StopCode = stopCode.String
	task.StoppedReason = stoppedReason.String
	task.Connectivity = connectivity.String
	task.PlatformVersion = platformVersion.String
	task.PlatformFamily = platformFamily.String
	task.Group = group.String
	task.Attachments = attachments.String
	task.HealthStatus = healthStatus.String
	task.Tags = tags.String
	task.Attributes = attributes.String
	task.CapacityProviderName = capacityProviderName.String
	task.EphemeralStorage = ephemeralStorage.String
	task.PodName = podName.String
	task.Namespace = namespace.String

	if stoppingAt.Valid {
		task.StoppingAt = &stoppingAt.Time
	}
	if stoppedAt.Valid {
		task.StoppedAt = &stoppedAt.Time
	}
	if connectivityAt.Valid {
		task.ConnectivityAt = &connectivityAt.Time
	}
	if pullStartedAt.Valid {
		task.PullStartedAt = &pullStartedAt.Time
	}
	if pullStoppedAt.Valid {
		task.PullStoppedAt = &pullStoppedAt.Time
	}
	if executionStoppedAt.Valid {
		task.ExecutionStoppedAt = &executionStoppedAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}

	return &task, nil
}

func (s *taskStore) List(ctx context.Context, cluster string, filters storage.TaskFilters) ([]*storage.Task, error) {
	query := `SELECT * FROM tasks WHERE cluster_arn = ?`
	args := []interface{}{cluster}

	// Build dynamic WHERE clauses based on filters
	var conditions []string

	if filters.ServiceName != "" {
		conditions = append(conditions, "started_by = ?")
		args = append(args, fmt.Sprintf("ecs-svc/%s", filters.ServiceName))
	}

	if filters.Family != "" {
		conditions = append(conditions, "task_definition_arn LIKE ?")
		args = append(args, fmt.Sprintf("%%:%s:%%", filters.Family))
	}

	if filters.ContainerInstance != "" {
		conditions = append(conditions, "container_instance_arn = ?")
		args = append(args, filters.ContainerInstance)
	}

	if filters.LaunchType != "" {
		conditions = append(conditions, "launch_type = ?")
		args = append(args, filters.LaunchType)
	}

	if filters.DesiredStatus != "" {
		conditions = append(conditions, "desired_status = ?")
		args = append(args, filters.DesiredStatus)
	}

	if filters.StartedBy != "" {
		conditions = append(conditions, "started_by = ?")
		args = append(args, filters.StartedBy)
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if filters.MaxResults > 0 {
		query += fmt.Sprintf(" LIMIT %d", filters.MaxResults)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*storage.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

func (s *taskStore) Update(ctx context.Context, task *storage.Task) error {
	query := `
		UPDATE tasks SET
			last_status = ?, desired_status = ?, containers = ?,
			version = ?, stop_code = ?, stopped_reason = ?,
			stopping_at = ?, stopped_at = ?, connectivity = ?,
			connectivity_at = ?, pull_started_at = ?, pull_stopped_at = ?,
			execution_stopped_at = ?, started_at = ?, health_status = ?,
			pod_name = ?, namespace = ?
		WHERE arn = ?`

	_, err := s.db.ExecContext(ctx, query,
		task.LastStatus, task.DesiredStatus, task.Containers,
		task.Version, nullString(task.StopCode), nullString(task.StoppedReason),
		nullTime(task.StoppingAt), nullTime(task.StoppedAt),
		nullString(task.Connectivity), nullTime(task.ConnectivityAt),
		nullTime(task.PullStartedAt), nullTime(task.PullStoppedAt),
		nullTime(task.ExecutionStoppedAt), nullTime(task.StartedAt),
		nullString(task.HealthStatus), nullString(task.PodName),
		nullString(task.Namespace), task.ARN,
	)

	return err
}

func (s *taskStore) Delete(ctx context.Context, cluster, taskID string) error {
	// Handle both short task ID and full ARN
	var query string
	var args []interface{}

	if strings.Contains(taskID, "arn:aws:ecs:") {
		query = `DELETE FROM tasks WHERE arn = ?`
		args = []interface{}{taskID}
	} else {
		query = `DELETE FROM tasks WHERE cluster_arn = ? AND (id = ? OR arn LIKE ?)`
		args = []interface{}{cluster, taskID, fmt.Sprintf("%%/%s", taskID)}
	}

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *taskStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.Task, error) {
	if len(arns) == 0 {
		return []*storage.Task{}, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(arns))
	args := make([]interface{}, len(arns))
	for i, arn := range arns {
		placeholders[i] = "?"
		args[i] = arn
	}

	query := fmt.Sprintf("SELECT * FROM tasks WHERE arn IN (%s)",
		strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*storage.Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// Helper function to scan a task row
func scanTask(rows *sql.Rows) (*storage.Task, error) {
	var task storage.Task
	var stoppingAt, stoppedAt, connectivityAt, pullStartedAt, pullStoppedAt sql.NullTime
	var executionStoppedAt, startedAt sql.NullTime
	var containerInstanceARN, overrides, cpu, memory, startedBy sql.NullString
	var stopCode, stoppedReason, connectivity, platformVersion sql.NullString
	var platformFamily, group, attachments, healthStatus, tags sql.NullString
	var attributes, capacityProviderName, ephemeralStorage sql.NullString
	var podName, namespace sql.NullString

	err := rows.Scan(
		&task.ID, &task.ARN, &task.ClusterARN, &task.TaskDefinitionARN,
		&containerInstanceARN, &overrides, &task.LastStatus,
		&task.DesiredStatus, &cpu, &memory, &task.Containers,
		&startedBy, &task.Version, &stopCode, &stoppedReason,
		&stoppingAt, &stoppedAt, &connectivity, &connectivityAt,
		&pullStartedAt, &pullStoppedAt, &executionStoppedAt,
		&task.CreatedAt, &startedAt, &task.LaunchType,
		&platformVersion, &platformFamily, &group, &attachments,
		&healthStatus, &tags, &attributes, &task.EnableExecuteCommand,
		&capacityProviderName, &ephemeralStorage, &task.Region,
		&task.AccountID, &podName, &namespace,
	)

	if err != nil {
		return nil, err
	}

	// Set nullable fields
	task.ContainerInstanceARN = containerInstanceARN.String
	task.Overrides = overrides.String
	task.CPU = cpu.String
	task.Memory = memory.String
	task.StartedBy = startedBy.String
	task.StopCode = stopCode.String
	task.StoppedReason = stoppedReason.String
	task.Connectivity = connectivity.String
	task.PlatformVersion = platformVersion.String
	task.PlatformFamily = platformFamily.String
	task.Group = group.String
	task.Attachments = attachments.String
	task.HealthStatus = healthStatus.String
	task.Tags = tags.String
	task.Attributes = attributes.String
	task.CapacityProviderName = capacityProviderName.String
	task.EphemeralStorage = ephemeralStorage.String
	task.PodName = podName.String
	task.Namespace = namespace.String

	if stoppingAt.Valid {
		task.StoppingAt = &stoppingAt.Time
	}
	if stoppedAt.Valid {
		task.StoppedAt = &stoppedAt.Time
	}
	if connectivityAt.Valid {
		task.ConnectivityAt = &connectivityAt.Time
	}
	if pullStartedAt.Valid {
		task.PullStartedAt = &pullStartedAt.Time
	}
	if pullStoppedAt.Valid {
		task.PullStoppedAt = &pullStoppedAt.Time
	}
	if executionStoppedAt.Valid {
		task.ExecutionStoppedAt = &executionStoppedAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}

	return &task, nil
}

// Helper functions for nullable values
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// CreateOrUpdate creates a new task or updates if it already exists
func (s *taskStore) CreateOrUpdate(ctx context.Context, task *storage.Task) error {
	query := `
		INSERT INTO tasks (
			id, arn, cluster_arn, task_definition_arn, container_instance_arn,
			overrides, last_status, desired_status, cpu, memory, containers,
			started_by, version, stop_code, stopped_reason, stopping_at,
			stopped_at, connectivity, connectivity_at, pull_started_at,
			pull_stopped_at, execution_stopped_at, created_at, started_at,
			launch_type, platform_version, platform_family, task_group,
			attachments, health_status, tags, attributes, enable_execute_command,
			capacity_provider_name, ephemeral_storage, region, account_id,
			pod_name, namespace
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		) ON CONFLICT (arn) DO UPDATE SET
			last_status = EXCLUDED.last_status,
			desired_status = EXCLUDED.desired_status,
			containers = EXCLUDED.containers,
			version = EXCLUDED.version,
			stop_code = EXCLUDED.stop_code,
			stopped_reason = EXCLUDED.stopped_reason,
			stopping_at = EXCLUDED.stopping_at,
			stopped_at = EXCLUDED.stopped_at,
			connectivity = EXCLUDED.connectivity,
			connectivity_at = EXCLUDED.connectivity_at,
			pull_started_at = EXCLUDED.pull_started_at,
			pull_stopped_at = EXCLUDED.pull_stopped_at,
			execution_stopped_at = EXCLUDED.execution_stopped_at,
			started_at = EXCLUDED.started_at,
			health_status = EXCLUDED.health_status,
			pod_name = EXCLUDED.pod_name,
			namespace = EXCLUDED.namespace`

	_, err := s.db.ExecContext(ctx, query,
		task.ID, task.ARN, task.ClusterARN, task.TaskDefinitionARN,
		nullString(task.ContainerInstanceARN), nullString(task.Overrides),
		task.LastStatus, task.DesiredStatus, nullString(task.CPU),
		nullString(task.Memory), task.Containers, nullString(task.StartedBy),
		task.Version, nullString(task.StopCode), nullString(task.StoppedReason),
		nullTime(task.StoppingAt), nullTime(task.StoppedAt),
		nullString(task.Connectivity), nullTime(task.ConnectivityAt),
		nullTime(task.PullStartedAt), nullTime(task.PullStoppedAt),
		nullTime(task.ExecutionStoppedAt), task.CreatedAt,
		nullTime(task.StartedAt), task.LaunchType,
		nullString(task.PlatformVersion), nullString(task.PlatformFamily),
		nullString(task.Group), nullString(task.Attachments),
		nullString(task.HealthStatus), nullString(task.Tags),
		nullString(task.Attributes), task.EnableExecuteCommand,
		nullString(task.CapacityProviderName), nullString(task.EphemeralStorage),
		task.Region, task.AccountID, nullString(task.PodName),
		nullString(task.Namespace),
	)

	return err
}
