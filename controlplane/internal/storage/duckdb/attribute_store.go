package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// attributeStore implements storage.AttributeStore
type attributeStore struct {
	db *sql.DB
}

// Put creates or updates attributes
func (s *attributeStore) Put(ctx context.Context, attributes []*storage.Attribute) error {
	if len(attributes) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO attributes (
			id, name, value, target_type, target_id, cluster,
			region, account_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (name, target_type, target_id, cluster) 
		DO UPDATE SET
			value = EXCLUDED.value,
			updated_at = EXCLUDED.updated_at
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()

	for _, attr := range attributes {
		// Generate ID if not provided
		if attr.ID == "" {
			attr.ID = uuid.New().String()
		}

		// Set timestamps
		if attr.CreatedAt.IsZero() {
			attr.CreatedAt = now
		}
		attr.UpdatedAt = now

		_, err := stmt.ExecContext(ctx,
			attr.ID, attr.Name, attr.Value, attr.TargetType,
			attr.TargetID, attr.Cluster, attr.Region,
			attr.AccountID, attr.CreatedAt, attr.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to put attribute %s: %w", attr.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete removes attributes
func (s *attributeStore) Delete(ctx context.Context, cluster string, attributes []*storage.Attribute) error {
	if len(attributes) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		DELETE FROM attributes
		WHERE cluster = $1 AND name = $2 AND target_type = $3 AND target_id = $4
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, attr := range attributes {
		_, err := stmt.ExecContext(ctx,
			cluster, attr.Name, attr.TargetType, attr.TargetID,
		)
		if err != nil {
			return fmt.Errorf("failed to delete attribute %s: %w", attr.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListWithPagination retrieves attributes with pagination
func (s *attributeStore) ListWithPagination(ctx context.Context, targetType, cluster string, limit int, nextToken string) ([]*storage.Attribute, string, error) {
	// Build the query
	query := `
		SELECT 
			id, name, value, target_type, target_id, cluster,
			region, account_id, created_at, updated_at
		FROM attributes
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 0

	// Add target type filter if specified
	if targetType != "" {
		argCount++
		query += fmt.Sprintf(" AND target_type = $%d", argCount)
		args = append(args, targetType)
	}

	// Add cluster filter if specified
	if cluster != "" {
		argCount++
		query += fmt.Sprintf(" AND cluster = $%d", argCount)
		args = append(args, cluster)
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
		return nil, "", fmt.Errorf("failed to list attributes: %w", err)
	}
	defer rows.Close()

	attributes := make([]*storage.Attribute, 0, limit)
	var newNextToken string

	for rows.Next() {
		attr := &storage.Attribute{}
		var value sql.NullString

		err := rows.Scan(
			&attr.ID, &attr.Name, &value, &attr.TargetType,
			&attr.TargetID, &attr.Cluster, &attr.Region,
			&attr.AccountID, &attr.CreatedAt, &attr.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan attribute: %w", err)
		}

		// Handle nullable fields
		attr.Value = value.String

		// If we've reached the limit, use this as the next token
		if len(attributes) >= limit {
			newNextToken = attr.ID
			break
		}

		attributes = append(attributes, attr)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to iterate attributes: %w", err)
	}

	return attributes, newNextToken, nil
}
