package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type taskDefinitionStore struct {
	db *sql.DB
}

// NewTaskDefinitionStore creates a new TaskDefinitionStore instance
func NewTaskDefinitionStore(db *sql.DB) storage.TaskDefinitionStore {
	return &taskDefinitionStore{db: db}
}

// Register creates a new task definition revision
func (s *taskDefinitionStore) Register(ctx context.Context, taskDef *storage.TaskDefinition) (*storage.TaskDefinition, error) {
	// Get the next revision number for this family
	var nextRevision int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(revision), 0) + 1 
		FROM task_definitions 
		WHERE family = ?
	`, taskDef.Family).Scan(&nextRevision)
	if err != nil {
		return nil, fmt.Errorf("failed to get next revision: %w", err)
	}

	taskDef.Revision = nextRevision
	taskDef.ARN = fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/%s:%d",
		taskDef.Region, taskDef.AccountID, taskDef.Family, taskDef.Revision)
	taskDef.Status = "ACTIVE"
	taskDef.RegisteredAt = time.Now()

	// Insert the new task definition
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO task_definitions (
			id, arn, family, revision, task_role_arn, execution_role_arn,
			network_mode, container_definitions, volumes, placement_constraints,
			requires_compatibilities, cpu, memory, tags, pid_mode, ipc_mode,
			proxy_configuration, inference_accelerators, runtime_platform,
			status, region, account_id, registered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		taskDef.ID, taskDef.ARN, taskDef.Family, taskDef.Revision,
		taskDef.TaskRoleARN, taskDef.ExecutionRoleARN, taskDef.NetworkMode,
		taskDef.ContainerDefinitions, taskDef.Volumes, taskDef.PlacementConstraints,
		taskDef.RequiresCompatibilities, taskDef.CPU, taskDef.Memory, taskDef.Tags,
		taskDef.PidMode, taskDef.IpcMode, taskDef.ProxyConfiguration,
		taskDef.InferenceAccelerators, taskDef.RuntimePlatform,
		taskDef.Status, taskDef.Region, taskDef.AccountID, taskDef.RegisteredAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert task definition: %w", err)
	}

	return taskDef, nil
}

// Get retrieves a specific task definition revision
func (s *taskDefinitionStore) Get(ctx context.Context, family string, revision int) (*storage.TaskDefinition, error) {
	var taskDef storage.TaskDefinition
	var deregisteredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT 
			id, arn, family, revision, task_role_arn, execution_role_arn,
			network_mode, container_definitions, volumes, placement_constraints,
			requires_compatibilities, cpu, memory, tags, pid_mode, ipc_mode,
			proxy_configuration, inference_accelerators, runtime_platform,
			status, region, account_id, registered_at, deregistered_at
		FROM task_definitions
		WHERE family = ? AND revision = ?
	`, family, revision).Scan(
		&taskDef.ID, &taskDef.ARN, &taskDef.Family, &taskDef.Revision,
		&taskDef.TaskRoleARN, &taskDef.ExecutionRoleARN, &taskDef.NetworkMode,
		&taskDef.ContainerDefinitions, &taskDef.Volumes, &taskDef.PlacementConstraints,
		&taskDef.RequiresCompatibilities, &taskDef.CPU, &taskDef.Memory, &taskDef.Tags,
		&taskDef.PidMode, &taskDef.IpcMode, &taskDef.ProxyConfiguration,
		&taskDef.InferenceAccelerators, &taskDef.RuntimePlatform,
		&taskDef.Status, &taskDef.Region, &taskDef.AccountID, &taskDef.RegisteredAt, &deregisteredAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task definition not found: %s:%d", family, revision)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task definition: %w", err)
	}

	if deregisteredAt.Valid {
		taskDef.DeregisteredAt = &deregisteredAt.Time
	}

	return &taskDef, nil
}

// GetLatest retrieves the latest revision of a task definition family
func (s *taskDefinitionStore) GetLatest(ctx context.Context, family string) (*storage.TaskDefinition, error) {
	log.Printf("DEBUG: GetLatest called for family: %s", family)
	
	var revision sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT MAX(revision) 
		FROM task_definitions 
		WHERE family = ? AND status = 'ACTIVE'
	`, family).Scan(&revision)
	if err != nil {
		log.Printf("DEBUG: GetLatest query error: %v", err)
		return nil, fmt.Errorf("failed to get latest revision: %w", err)
	}
	
	// Check if we got a NULL (no active revisions found)
	if !revision.Valid {
		log.Printf("DEBUG: No active revisions found for family: %s", family)
		return nil, fmt.Errorf("task definition family not found: %s", family)
	}

	log.Printf("DEBUG: Found latest revision %d for family %s", revision.Int64, family)
	return s.Get(ctx, family, int(revision.Int64))
}

// ListFamilies lists task definition families with pagination
func (s *taskDefinitionStore) ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionFamily, string, error) {
	query := `
		SELECT 
			family,
			MAX(revision) as latest_revision,
			COUNT(CASE WHEN status = 'ACTIVE' THEN 1 END) as active_revisions
		FROM task_definitions
		WHERE 1=1
	`
	args := []interface{}{}

	if familyPrefix != "" {
		query += " AND family LIKE ?"
		args = append(args, familyPrefix+"%")
	}

	if status != "" {
		query += " AND family IN (SELECT DISTINCT family FROM task_definitions WHERE status = ?)"
		args = append(args, status)
	}

	// Handle pagination token
	if nextToken != "" {
		query += " AND family > ?"
		args = append(args, nextToken)
	}

	query += " GROUP BY family ORDER BY family"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit+1) // Get one extra to determine if there are more results
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list task definition families: %w", err)
	}
	defer rows.Close()

	var families []*storage.TaskDefinitionFamily
	for rows.Next() {
		var family storage.TaskDefinitionFamily
		if err := rows.Scan(&family.Family, &family.LatestRevision, &family.ActiveRevisions); err != nil {
			return nil, "", fmt.Errorf("failed to scan task definition family: %w", err)
		}
		families = append(families, &family)
	}

	// Handle pagination
	var newNextToken string
	if limit > 0 && len(families) > limit {
		families = families[:limit]
		newNextToken = families[len(families)-1].Family
	}

	return families, newNextToken, nil
}

// ListRevisions lists revisions of a specific task definition family
func (s *taskDefinitionStore) ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*storage.TaskDefinitionRevision, string, error) {
	query := `
		SELECT arn, family, revision, status, registered_at
		FROM task_definitions
		WHERE family = ?
	`
	args := []interface{}{family}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	// Handle pagination token (assuming it's the revision number)
	if nextToken != "" {
		revision, err := strconv.Atoi(nextToken)
		if err != nil {
			return nil, "", fmt.Errorf("invalid pagination token: %w", err)
		}
		query += " AND revision < ?"
		args = append(args, revision)
	}

	query += " ORDER BY revision DESC"

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit+1)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list task definition revisions: %w", err)
	}
	defer rows.Close()

	var revisions []*storage.TaskDefinitionRevision
	for rows.Next() {
		var rev storage.TaskDefinitionRevision
		if err := rows.Scan(&rev.ARN, &rev.Family, &rev.Revision, &rev.Status, &rev.RegisteredAt); err != nil {
			return nil, "", fmt.Errorf("failed to scan task definition revision: %w", err)
		}
		revisions = append(revisions, &rev)
	}

	// Handle pagination
	var newNextToken string
	if limit > 0 && len(revisions) > limit {
		revisions = revisions[:limit]
		newNextToken = strconv.Itoa(revisions[len(revisions)-1].Revision)
	}

	return revisions, newNextToken, nil
}

// Deregister marks a task definition revision as INACTIVE
func (s *taskDefinitionStore) Deregister(ctx context.Context, family string, revision int) error {
	// DuckDB has a known issue with UPDATE on tables with primary key constraints
	// See: https://github.com/duckdb/duckdb/issues/8764
	// The error happens even when not updating the primary key column
	
	// First check if it exists and is active
	var currentStatus string
	err := s.db.QueryRowContext(ctx, `
		SELECT status FROM task_definitions 
		WHERE family = ? AND revision = ?
	`, family, revision).Scan(&currentStatus)
	
	if err == sql.ErrNoRows {
		return fmt.Errorf("task definition not found: %s:%d", family, revision)
	}
	if err != nil {
		return fmt.Errorf("failed to check task definition status: %w", err)
	}
	
	if currentStatus == "INACTIVE" {
		// Already inactive, return success (idempotent)
		return nil
	}
	
	// Try the UPDATE
	result, err := s.db.ExecContext(ctx, `
		UPDATE task_definitions 
		SET status = 'INACTIVE', deregistered_at = ?
		WHERE family = ? AND revision = ? AND status = 'ACTIVE'
	`, time.Now(), family, revision)
	
	if err != nil {
		// Check if this is the known DuckDB constraint error
		if strings.Contains(err.Error(), "Duplicate key") && strings.Contains(err.Error(), "violates primary key constraint") {
			// This is the known DuckDB bug - the UPDATE actually succeeded
			// but DuckDB incorrectly reports a constraint violation
			// We'll verify the update worked
			var updatedStatus string
			err2 := s.db.QueryRowContext(ctx, `
				SELECT status FROM task_definitions 
				WHERE family = ? AND revision = ?
			`, family, revision).Scan(&updatedStatus)
			
			if err2 == nil && updatedStatus == "INACTIVE" {
				// The update actually worked despite the error
				return nil
			}
			// If we can't verify, return the original error
			return fmt.Errorf("failed to deregister task definition due to DuckDB bug: %w", err)
		}
		return fmt.Errorf("failed to deregister task definition: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("task definition not found or already inactive: %s:%d", family, revision)
	}

	return nil
}

// GetByARN retrieves a task definition by its ARN
func (s *taskDefinitionStore) GetByARN(ctx context.Context, arn string) (*storage.TaskDefinition, error) {
	// Parse ARN to extract family and revision
	// Format: arn:aws:ecs:region:account:task-definition/family:revision
	parts := strings.Split(arn, ":")
	if len(parts) < 7 {
		return nil, fmt.Errorf("invalid task definition ARN: %s", arn)
	}

	// For an ARN like "arn:aws:ecs:us-east-1:123456789012:task-definition/nginx-fargate:2"
	// parts[5] = "task-definition/nginx-fargate"
	// parts[6] = "2" (revision)
	taskDefPart := parts[5]
	revisionStr := parts[6]
	if !strings.HasPrefix(taskDefPart, "task-definition/") {
		return nil, fmt.Errorf("invalid task definition ARN: %s", arn)
	}

	// Extract "nginx-fargate" from "task-definition/nginx-fargate"
	family := strings.TrimPrefix(taskDefPart, "task-definition/")
	revision, err := strconv.Atoi(revisionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid task definition revision: %s", revisionStr)
	}

	return s.Get(ctx, family, revision)
}

// createTaskDefinitionTable creates the task_definitions table if it doesn't exist
func createTaskDefinitionTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS task_definitions (
		id VARCHAR PRIMARY KEY,
		arn VARCHAR NOT NULL UNIQUE,
		family VARCHAR NOT NULL,
		revision INTEGER NOT NULL,
		task_role_arn VARCHAR,
		execution_role_arn VARCHAR,
		network_mode VARCHAR NOT NULL,
		container_definitions TEXT NOT NULL,
		volumes TEXT,
		placement_constraints TEXT,
		requires_compatibilities TEXT,
		cpu VARCHAR,
		memory VARCHAR,
		tags TEXT,
		pid_mode VARCHAR,
		ipc_mode VARCHAR,
		proxy_configuration TEXT,
		inference_accelerators TEXT,
		runtime_platform TEXT,
		status VARCHAR NOT NULL,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		registered_at TIMESTAMP NOT NULL,
		deregistered_at TIMESTAMP,
		UNIQUE(family, revision)
	)
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create task_definitions table: %w", err)
	}

	// Create indexes for common queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_family ON task_definitions(family)",
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_status ON task_definitions(status)",
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_family_status ON task_definitions(family, status)",
		"CREATE INDEX IF NOT EXISTS idx_task_definitions_family_revision ON task_definitions(family, revision DESC)",
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}