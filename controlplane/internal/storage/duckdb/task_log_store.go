package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// TaskLogStore implements storage.TaskLogStore for DuckDB
type TaskLogStore struct {
	db *sql.DB
}

// NewTaskLogStore creates a new TaskLogStore
func NewTaskLogStore(db *sql.DB) *TaskLogStore {
	return &TaskLogStore{
		db: db,
	}
}

// SaveLogs saves multiple log entries for a task container
func (s *TaskLogStore) SaveLogs(ctx context.Context, logs []storage.TaskLog) error {
	if len(logs) == 0 {
		return nil
	}

	// Begin transaction for batch insert
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO task_logs (task_arn, container_name, timestamp, log_line, log_level)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each log entry
	for _, log := range logs {
		_, err := stmt.ExecContext(ctx,
			log.TaskArn,
			log.ContainerName,
			log.Timestamp,
			log.LogLine,
			log.LogLevel,
		)
		if err != nil {
			return fmt.Errorf("failed to insert log: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logging.Info("Saved task logs",
		"count", len(logs),
		"task_arn", logs[0].TaskArn,
		"container", logs[0].ContainerName,
	)

	return nil
}

// GetLogs retrieves logs based on filter criteria
func (s *TaskLogStore) GetLogs(ctx context.Context, filter storage.TaskLogFilter) ([]storage.TaskLog, error) {
	query := `
		SELECT id, task_arn, container_name, timestamp, log_line, log_level, created_at
		FROM task_logs
		WHERE 1=1
	`
	args := []interface{}{}

	// Build dynamic query based on filter
	if filter.TaskArn != "" {
		query += " AND task_arn = ?"
		args = append(args, filter.TaskArn)
	}

	if filter.ContainerName != "" {
		query += " AND container_name = ?"
		args = append(args, filter.ContainerName)
	}

	if filter.From != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.From)
	}

	if filter.To != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.To)
	}

	if filter.LogLevel != "" {
		query += " AND log_level = ?"
		args = append(args, filter.LogLevel)
	}

	if filter.SearchText != "" {
		query += " AND log_line LIKE ?"
		args = append(args, "%"+filter.SearchText+"%")
	}

	// Add ordering
	query += " ORDER BY timestamp ASC"

	// Add pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []storage.TaskLog
	for rows.Next() {
		var log storage.TaskLog
		var logLevel sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.TaskArn,
			&log.ContainerName,
			&log.Timestamp,
			&log.LogLine,
			&logLevel,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		if logLevel.Valid {
			log.LogLevel = logLevel.String
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return logs, nil
}

// GetLogCount returns the count of logs matching the filter
func (s *TaskLogStore) GetLogCount(ctx context.Context, filter storage.TaskLogFilter) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM task_logs
		WHERE 1=1
	`
	args := []interface{}{}

	// Build dynamic query based on filter (same as GetLogs but without ORDER BY, LIMIT, OFFSET)
	if filter.TaskArn != "" {
		query += " AND task_arn = ?"
		args = append(args, filter.TaskArn)
	}

	if filter.ContainerName != "" {
		query += " AND container_name = ?"
		args = append(args, filter.ContainerName)
	}

	if filter.From != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.From)
	}

	if filter.To != nil {
		query += " AND timestamp <= ?"
		args = append(args, *filter.To)
	}

	if filter.LogLevel != "" {
		query += " AND log_level = ?"
		args = append(args, filter.LogLevel)
	}

	if filter.SearchText != "" {
		query += " AND log_line LIKE ?"
		args = append(args, "%"+filter.SearchText+"%")
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}

	return count, nil
}

// DeleteOldLogs removes logs older than the specified retention period
func (s *TaskLogStore) DeleteOldLogs(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := s.db.ExecContext(ctx,
		"DELETE FROM task_logs WHERE created_at < ?",
		olderThan,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		logging.Info("Deleted old task logs",
			"count", rowsAffected,
			"older_than", olderThan,
		)
	}

	return rowsAffected, nil
}

// DeleteTaskLogs removes all logs for a specific task
func (s *TaskLogStore) DeleteTaskLogs(ctx context.Context, taskArn string) error {
	_, err := s.db.ExecContext(ctx,
		"DELETE FROM task_logs WHERE task_arn = ?",
		taskArn,
	)
	if err != nil {
		return fmt.Errorf("failed to delete task logs: %w", err)
	}

	logging.Debug("Deleted logs for task", "task_arn", taskArn)
	return nil
}

// StreamLogs retrieves logs and sends them through a channel for streaming
func (s *TaskLogStore) StreamLogs(ctx context.Context, filter storage.TaskLogFilter, logChan chan<- storage.TaskLog) error {
	defer close(logChan)

	// Build query similar to GetLogs but optimized for streaming
	query := `
		SELECT id, task_arn, container_name, timestamp, log_line, log_level, created_at
		FROM task_logs
		WHERE task_arn = ? AND container_name = ?
	`
	args := []interface{}{filter.TaskArn, filter.ContainerName}

	if filter.From != nil {
		query += " AND timestamp >= ?"
		args = append(args, *filter.From)
	}

	query += " ORDER BY timestamp ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var log storage.TaskLog
		var logLevel sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.TaskArn,
			&log.ContainerName,
			&log.Timestamp,
			&log.LogLine,
			&logLevel,
			&log.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan log: %w", err)
		}

		if logLevel.Valid {
			log.LogLevel = logLevel.String
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case logChan <- log:
		}
	}

	return rows.Err()
}

// parseLogLevel attempts to extract log level from log line
func parseLogLevel(logLine string) string {
	logLine = strings.ToUpper(logLine)

	// Common log level patterns
	if strings.Contains(logLine, "[ERROR]") || strings.Contains(logLine, "ERROR:") {
		return "ERROR"
	}
	if strings.Contains(logLine, "[WARN]") || strings.Contains(logLine, "WARN:") ||
		strings.Contains(logLine, "[WARNING]") || strings.Contains(logLine, "WARNING:") {
		return "WARN"
	}
	if strings.Contains(logLine, "[INFO]") || strings.Contains(logLine, "INFO:") {
		return "INFO"
	}
	if strings.Contains(logLine, "[DEBUG]") || strings.Contains(logLine, "DEBUG:") {
		return "DEBUG"
	}
	if strings.Contains(logLine, "[TRACE]") || strings.Contains(logLine, "TRACE:") {
		return "TRACE"
	}

	return ""
}
