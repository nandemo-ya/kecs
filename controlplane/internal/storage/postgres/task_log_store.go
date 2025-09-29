package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

type taskLogStore struct {
	db *sql.DB
}

// SaveLogs saves multiple log entries for a task container
func (s *taskLogStore) SaveLogs(ctx context.Context, logs []storage.TaskLog) error {
	if len(logs) == 0 {
		return nil
	}

	// Use a transaction for batch insert
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO task_logs (
		id, task_arn, container_name, timestamp, log_line,
		log_level, created_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7
	)`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, log := range logs {
		if log.ID == "" {
			log.ID = uuid.New().String()
		}
		if log.CreatedAt.IsZero() {
			log.CreatedAt = now
		}

		_, err = stmt.ExecContext(ctx,
			log.ID, log.TaskArn, log.ContainerName,
			log.Timestamp, log.LogLine,
			toNullString(log.LogLevel), log.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to save log: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetLogs retrieves logs based on filter criteria
func (s *taskLogStore) GetLogs(ctx context.Context, filter storage.TaskLogFilter) ([]storage.TaskLog, error) {
	query := `
	SELECT id, task_arn, container_name, timestamp, log_line,
		log_level, created_at
	FROM task_logs
	WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	// Build dynamic query based on filter
	if filter.TaskArn != "" {
		query += fmt.Sprintf(" AND task_arn = $%d", argNum)
		args = append(args, filter.TaskArn)
		argNum++
	}

	if filter.ContainerName != "" {
		query += fmt.Sprintf(" AND container_name = $%d", argNum)
		args = append(args, filter.ContainerName)
		argNum++
	}

	if filter.From != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, *filter.From)
		argNum++
	}

	if filter.To != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argNum)
		args = append(args, *filter.To)
		argNum++
	}

	if filter.LogLevel != "" {
		query += fmt.Sprintf(" AND log_level = $%d", argNum)
		args = append(args, filter.LogLevel)
		argNum++
	}

	if filter.SearchText != "" {
		query += fmt.Sprintf(" AND log_line ILIKE $%d", argNum)
		args = append(args, "%"+filter.SearchText+"%")
		argNum++
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, filter.Limit)
		argNum++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, filter.Offset)
		argNum++
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer rows.Close()

	var logs []storage.TaskLog
	for rows.Next() {
		var log storage.TaskLog
		var logLevel sql.NullString

		err := rows.Scan(
			&log.ID, &log.TaskArn, &log.ContainerName,
			&log.Timestamp, &log.LogLine,
			&logLevel, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		log.LogLevel = fromNullString(logLevel)
		logs = append(logs, log)
	}

	return logs, nil
}

// GetLogCount returns the count of logs matching the filter
func (s *taskLogStore) GetLogCount(ctx context.Context, filter storage.TaskLogFilter) (int64, error) {
	query := `
	SELECT COUNT(*)
	FROM task_logs
	WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	// Build dynamic query based on filter (same as GetLogs but without ORDER BY and LIMIT)
	if filter.TaskArn != "" {
		query += fmt.Sprintf(" AND task_arn = $%d", argNum)
		args = append(args, filter.TaskArn)
		argNum++
	}

	if filter.ContainerName != "" {
		query += fmt.Sprintf(" AND container_name = $%d", argNum)
		args = append(args, filter.ContainerName)
		argNum++
	}

	if filter.From != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argNum)
		args = append(args, *filter.From)
		argNum++
	}

	if filter.To != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argNum)
		args = append(args, *filter.To)
		argNum++
	}

	if filter.LogLevel != "" {
		query += fmt.Sprintf(" AND log_level = $%d", argNum)
		args = append(args, filter.LogLevel)
		argNum++
	}

	if filter.SearchText != "" {
		query += fmt.Sprintf(" AND log_line ILIKE $%d", argNum)
		args = append(args, "%"+filter.SearchText+"%")
		argNum++
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get log count: %w", err)
	}

	return count, nil
}

// DeleteOldLogs removes logs older than the specified retention period
func (s *taskLogStore) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `
	DELETE FROM task_logs
	WHERE created_at < $1`

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// DeleteTaskLogs removes all logs for a specific task
func (s *taskLogStore) DeleteTaskLogs(ctx context.Context, taskArn string) error {
	query := `
	DELETE FROM task_logs
	WHERE task_arn = $1`

	_, err := s.db.ExecContext(ctx, query, taskArn)
	if err != nil {
		return fmt.Errorf("failed to delete task logs: %w", err)
	}

	return nil
}

// Helper function to build WHERE clause from filter
func buildWhereClause(filter storage.TaskLogFilter) (string, []interface{}) {
	conditions := []string{}
	args := []interface{}{}

	if filter.TaskArn != "" {
		conditions = append(conditions, fmt.Sprintf("task_arn = $%d", len(args)+1))
		args = append(args, filter.TaskArn)
	}

	if filter.ContainerName != "" {
		conditions = append(conditions, fmt.Sprintf("container_name = $%d", len(args)+1))
		args = append(args, filter.ContainerName)
	}

	if filter.From != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", len(args)+1))
		args = append(args, *filter.From)
	}

	if filter.To != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", len(args)+1))
		args = append(args, *filter.To)
	}

	if filter.LogLevel != "" {
		conditions = append(conditions, fmt.Sprintf("log_level = $%d", len(args)+1))
		args = append(args, filter.LogLevel)
	}

	if filter.SearchText != "" {
		conditions = append(conditions, fmt.Sprintf("log_line ILIKE $%d", len(args)+1))
		args = append(args, "%"+filter.SearchText+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}
