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

type taskDefinitionStore struct {
	db *sql.DB
}

// Register registers a new task definition (creates a new revision)
func (s *taskDefinitionStore) Register(ctx context.Context, td *storage.TaskDefinition) (*storage.TaskDefinition, error) {
	if td.ID == "" {
		td.ID = uuid.New().String()
	}

	if td.RegisteredAt.IsZero() {
		td.RegisteredAt = time.Now()
	}

	// Get the next revision number for this family
	var maxRevision int
	query := `SELECT COALESCE(MAX(revision), 0) FROM task_definitions WHERE family = $1`
	err := s.db.QueryRowContext(ctx, query, td.Family).Scan(&maxRevision)
	if err != nil {
		return nil, fmt.Errorf("failed to get max revision: %w", err)
	}

	// Set revision and ARN
	td.Revision = maxRevision + 1
	if td.Region == "" {
		td.Region = "us-east-1"
	}
	if td.AccountID == "" {
		td.AccountID = "000000000000"
	}
	td.ARN = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:%d",
		td.Region, td.AccountID, td.Family, td.Revision)

	if td.Status == "" {
		td.Status = "ACTIVE"
	}

	insertQuery := `
	INSERT INTO task_definitions (
		id, arn, family, revision, task_role_arn, execution_role_arn,
		network_mode, container_definitions, volumes, placement_constraints,
		requires_compatibilities, cpu, memory, tags, pid_mode, ipc_mode,
		proxy_configuration, inference_accelerators, runtime_platform,
		status, region, account_id, registered_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
		$11, $12, $13, $14, $15, $16, $17, $18, $19,
		$20, $21, $22, $23
	)`

	_, err = s.db.ExecContext(ctx, insertQuery,
		td.ID, td.ARN, td.Family, td.Revision,
		toNullString(td.TaskRoleARN), toNullString(td.ExecutionRoleARN),
		td.NetworkMode, td.ContainerDefinitions,
		toNullString(td.Volumes), toNullString(td.PlacementConstraints),
		toNullString(td.RequiresCompatibilities), toNullString(td.CPU),
		toNullString(td.Memory), toNullString(td.Tags),
		toNullString(td.PidMode), toNullString(td.IpcMode),
		toNullString(td.ProxyConfiguration), toNullString(td.InferenceAccelerators),
		toNullString(td.RuntimePlatform),
		td.Status, td.Region, td.AccountID, td.RegisteredAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return nil, storage.ErrResourceAlreadyExists
		}
		return nil, fmt.Errorf("failed to create task definition: %w", err)
	}

	return td, nil
}

// GetByARN retrieves a task definition by ARN
func (s *taskDefinitionStore) GetByARN(ctx context.Context, taskDefArn string) (*storage.TaskDefinition, error) {
	query := `
	SELECT
		id, arn, family, revision, task_role_arn, execution_role_arn,
		network_mode, container_definitions, volumes, placement_constraints,
		requires_compatibilities, cpu, memory, tags, pid_mode, ipc_mode,
		proxy_configuration, inference_accelerators, runtime_platform,
		status, region, account_id, registered_at, deregistered_at
	FROM task_definitions
	WHERE arn = $1`

	var td storage.TaskDefinition
	var taskRoleARN, executionRoleARN, volumes, placementConstraints sql.NullString
	var requiresCompatibilities, cpu, memory, tags sql.NullString
	var pidMode, ipcMode, proxyConfiguration, inferenceAccelerators sql.NullString
	var runtimePlatform sql.NullString
	var deregisteredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, taskDefArn).Scan(
		&td.ID, &td.ARN, &td.Family, &td.Revision,
		&taskRoleARN, &executionRoleARN,
		&td.NetworkMode, &td.ContainerDefinitions,
		&volumes, &placementConstraints,
		&requiresCompatibilities, &cpu, &memory, &tags,
		&pidMode, &ipcMode, &proxyConfiguration,
		&inferenceAccelerators, &runtimePlatform,
		&td.Status, &td.Region, &td.AccountID,
		&td.RegisteredAt, &deregisteredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get task definition: %w", err)
	}

	// Convert null values
	td.TaskRoleARN = fromNullString(taskRoleARN)
	td.ExecutionRoleARN = fromNullString(executionRoleARN)
	td.Volumes = fromNullString(volumes)
	td.PlacementConstraints = fromNullString(placementConstraints)
	td.RequiresCompatibilities = fromNullString(requiresCompatibilities)
	td.CPU = fromNullString(cpu)
	td.Memory = fromNullString(memory)
	td.Tags = fromNullString(tags)
	td.PidMode = fromNullString(pidMode)
	td.IpcMode = fromNullString(ipcMode)
	td.ProxyConfiguration = fromNullString(proxyConfiguration)
	td.InferenceAccelerators = fromNullString(inferenceAccelerators)
	td.RuntimePlatform = fromNullString(runtimePlatform)
	td.DeregisteredAt = fromNullTime(deregisteredAt)

	return &td, nil
}

// GetLatest retrieves the latest revision of a task definition family
func (s *taskDefinitionStore) GetLatest(ctx context.Context, family string) (*storage.TaskDefinition, error) {
	query := `
	SELECT
		id, arn, family, revision, task_role_arn, execution_role_arn,
		network_mode, container_definitions, volumes, placement_constraints,
		requires_compatibilities, cpu, memory, tags, pid_mode, ipc_mode,
		proxy_configuration, inference_accelerators, runtime_platform,
		status, region, account_id, registered_at, deregistered_at
	FROM task_definitions
	WHERE family = $1 AND status = 'ACTIVE'
	ORDER BY revision DESC
	LIMIT 1`

	var td storage.TaskDefinition
	var taskRoleARN, executionRoleARN, volumes, placementConstraints sql.NullString
	var requiresCompatibilities, cpu, memory, tags sql.NullString
	var pidMode, ipcMode, proxyConfiguration, inferenceAccelerators sql.NullString
	var runtimePlatform sql.NullString
	var deregisteredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, family).Scan(
		&td.ID, &td.ARN, &td.Family, &td.Revision,
		&taskRoleARN, &executionRoleARN,
		&td.NetworkMode, &td.ContainerDefinitions,
		&volumes, &placementConstraints,
		&requiresCompatibilities, &cpu, &memory, &tags,
		&pidMode, &ipcMode, &proxyConfiguration,
		&inferenceAccelerators, &runtimePlatform,
		&td.Status, &td.Region, &td.AccountID,
		&td.RegisteredAt, &deregisteredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get latest task definition: %w", err)
	}

	// Convert null values
	td.TaskRoleARN = fromNullString(taskRoleARN)
	td.ExecutionRoleARN = fromNullString(executionRoleARN)
	td.Volumes = fromNullString(volumes)
	td.PlacementConstraints = fromNullString(placementConstraints)
	td.RequiresCompatibilities = fromNullString(requiresCompatibilities)
	td.CPU = fromNullString(cpu)
	td.Memory = fromNullString(memory)
	td.Tags = fromNullString(tags)
	td.PidMode = fromNullString(pidMode)
	td.IpcMode = fromNullString(ipcMode)
	td.ProxyConfiguration = fromNullString(proxyConfiguration)
	td.InferenceAccelerators = fromNullString(inferenceAccelerators)
	td.RuntimePlatform = fromNullString(runtimePlatform)
	td.DeregisteredAt = fromNullTime(deregisteredAt)

	return &td, nil
}

// Get retrieves a specific task definition revision
func (s *taskDefinitionStore) Get(ctx context.Context, family string, revision int) (*storage.TaskDefinition, error) {
	query := `
	SELECT
		id, arn, family, revision, task_role_arn, execution_role_arn,
		network_mode, container_definitions, volumes, placement_constraints,
		requires_compatibilities, cpu, memory, tags, pid_mode, ipc_mode,
		proxy_configuration, inference_accelerators, runtime_platform,
		status, region, account_id, registered_at, deregistered_at
	FROM task_definitions
	WHERE family = $1 AND revision = $2`

	var td storage.TaskDefinition
	var taskRoleARN, executionRoleARN, volumes, placementConstraints sql.NullString
	var requiresCompatibilities, cpu, memory, tags sql.NullString
	var pidMode, ipcMode, proxyConfiguration, inferenceAccelerators sql.NullString
	var runtimePlatform sql.NullString
	var deregisteredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, family, revision).Scan(
		&td.ID, &td.ARN, &td.Family, &td.Revision,
		&taskRoleARN, &executionRoleARN,
		&td.NetworkMode, &td.ContainerDefinitions,
		&volumes, &placementConstraints,
		&requiresCompatibilities, &cpu, &memory, &tags,
		&pidMode, &ipcMode, &proxyConfiguration,
		&inferenceAccelerators, &runtimePlatform,
		&td.Status, &td.Region, &td.AccountID,
		&td.RegisteredAt, &deregisteredAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get task definition by family and revision: %w", err)
	}

	// Convert null values
	td.TaskRoleARN = fromNullString(taskRoleARN)
	td.ExecutionRoleARN = fromNullString(executionRoleARN)
	td.Volumes = fromNullString(volumes)
	td.PlacementConstraints = fromNullString(placementConstraints)
	td.RequiresCompatibilities = fromNullString(requiresCompatibilities)
	td.CPU = fromNullString(cpu)
	td.Memory = fromNullString(memory)
	td.Tags = fromNullString(tags)
	td.PidMode = fromNullString(pidMode)
	td.IpcMode = fromNullString(ipcMode)
	td.ProxyConfiguration = fromNullString(proxyConfiguration)
	td.InferenceAccelerators = fromNullString(inferenceAccelerators)
	td.RuntimePlatform = fromNullString(runtimePlatform)
	td.DeregisteredAt = fromNullTime(deregisteredAt)

	return &td, nil
}

// ListRevisions lists revisions of a specific task definition family
func (s *taskDefinitionStore) ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionRevision, string, error) {
	// Parse the next token to get offset
	offset := 0
	if nextToken != "" {
		if _, err := fmt.Sscanf(nextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	query := `
	SELECT family, revision, arn, status, registered_at, deregistered_at
	FROM task_definitions
	WHERE family = $1`

	args := []interface{}{family}
	argNum := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}

	query += " ORDER BY revision DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list task definition revisions: %w", err)
	}
	defer rows.Close()

	var revisions []*storage.TaskDefinitionRevision
	for rows.Next() {
		var rev storage.TaskDefinitionRevision
		var deregisteredAt sql.NullTime

		err := rows.Scan(
			&rev.Family, &rev.Revision, &rev.ARN, &rev.Status,
			&rev.RegisteredAt, &deregisteredAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan revision row: %w", err)
		}

		// Note: deregisteredAt is not part of TaskDefinitionRevision struct
		// but we query it for completeness
		revisions = append(revisions, &rev)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if limit > 0 && len(revisions) == limit {
		newNextToken = fmt.Sprintf("%d", offset+limit)
	}

	return revisions, newNextToken, nil
}

// ListFamilies lists task definition families with pagination
func (s *taskDefinitionStore) ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionFamily, string, error) {
	// Parse the next token to get offset
	offset := 0
	if nextToken != "" {
		if _, err := fmt.Sscanf(nextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	query := `
	SELECT DISTINCT family
	FROM task_definitions
	WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	if familyPrefix != "" {
		query += fmt.Sprintf(" AND family LIKE $%d", argNum)
		args = append(args, familyPrefix+"%")
		argNum++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, status)
		argNum++
	}

	query += " ORDER BY family"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list task definition families: %w", err)
	}
	defer rows.Close()

	var families []*storage.TaskDefinitionFamily
	for rows.Next() {
		var family storage.TaskDefinitionFamily
		err := rows.Scan(&family.Family)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan family row: %w", err)
		}
		families = append(families, &family)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if limit > 0 && len(families) == limit {
		newNextToken = fmt.Sprintf("%d", offset+limit)
	}

	return families, newNextToken, nil
}

// Update updates a task definition
func (s *taskDefinitionStore) Update(ctx context.Context, td *storage.TaskDefinition) error {
	query := `
	UPDATE task_definitions SET
		status = $1, tags = $2
	WHERE arn = $3`

	result, err := s.db.ExecContext(ctx, query,
		td.Status, toNullString(td.Tags), td.ARN,
	)

	if err != nil {
		return fmt.Errorf("failed to update task definition: %w", err)
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

// Delete deletes a task definition (logical delete)
func (s *taskDefinitionStore) Delete(ctx context.Context, taskDefArn string) error {
	query := `
	UPDATE task_definitions SET
		status = 'INACTIVE',
		deregistered_at = $1
	WHERE arn = $2`

	result, err := s.db.ExecContext(ctx, query, time.Now(), taskDefArn)
	if err != nil {
		return fmt.Errorf("failed to delete task definition: %w", err)
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

// Deregister marks a task definition as INACTIVE
func (s *taskDefinitionStore) Deregister(ctx context.Context, family string, revision int) error {
	query := `
	UPDATE task_definitions SET
		status = 'INACTIVE',
		deregistered_at = $1
	WHERE family = $2 AND revision = $3`

	result, err := s.db.ExecContext(ctx, query, time.Now(), family, revision)
	if err != nil {
		return fmt.Errorf("failed to deregister task definition: %w", err)
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
