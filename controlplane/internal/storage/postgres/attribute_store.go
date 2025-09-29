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

type attributeStore struct {
	db *sql.DB
}

// Put creates or updates attributes
func (s *attributeStore) Put(ctx context.Context, attributes []*storage.Attribute) error {
	if len(attributes) == 0 {
		return nil
	}

	// Use a transaction for batch insert/update
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	INSERT INTO attributes (
		id, name, value, target_type, target_id, cluster,
		region, account_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
	) ON CONFLICT (name, target_type, target_id, cluster) DO UPDATE SET
		value = EXCLUDED.value,
		updated_at = EXCLUDED.updated_at`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()
	for _, attr := range attributes {
		if attr.ID == "" {
			attr.ID = uuid.New().String()
		}
		if attr.CreatedAt.IsZero() {
			attr.CreatedAt = now
		}
		attr.UpdatedAt = now

		_, err = stmt.ExecContext(ctx,
			attr.ID, attr.Name, toNullString(attr.Value),
			attr.TargetType, attr.TargetID, attr.Cluster,
			toNullString(attr.Region), toNullString(attr.AccountID),
			attr.CreatedAt, attr.UpdatedAt,
		)
		if err != nil {
			if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
				// This shouldn't happen due to ON CONFLICT, but handle it anyway
				continue
			}
			return fmt.Errorf("failed to put attribute: %w", err)
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

	// Use a transaction for batch delete
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
	DELETE FROM attributes
	WHERE cluster = $1 AND name = $2 AND target_type = $3 AND target_id = $4`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, attr := range attributes {
		_, err = stmt.ExecContext(ctx, cluster, attr.Name, attr.TargetType, attr.TargetID)
		if err != nil {
			return fmt.Errorf("failed to delete attribute: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ListWithPagination retrieves attributes with pagination
func (s *attributeStore) ListWithPagination(ctx context.Context, targetType, cluster string, limit int, nextToken string) ([]*storage.Attribute, string, error) {
	// Parse the next token to get offset
	offset := 0
	if nextToken != "" {
		if _, err := fmt.Sscanf(nextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	query := `
	SELECT id, name, value, target_type, target_id, cluster,
		region, account_id, created_at, updated_at
	FROM attributes
	WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	if targetType != "" {
		query += fmt.Sprintf(" AND target_type = $%d", argNum)
		args = append(args, targetType)
		argNum++
	}

	if cluster != "" {
		query += fmt.Sprintf(" AND cluster = $%d", argNum)
		args = append(args, cluster)
		argNum++
	}

	query += " ORDER BY name, target_id"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list attributes: %w", err)
	}
	defer rows.Close()

	var attributes []*storage.Attribute
	for rows.Next() {
		var attr storage.Attribute
		var value, region, accountID sql.NullString

		err := rows.Scan(
			&attr.ID, &attr.Name, &value,
			&attr.TargetType, &attr.TargetID, &attr.Cluster,
			&region, &accountID,
			&attr.CreatedAt, &attr.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan attribute: %w", err)
		}

		attr.Value = fromNullString(value)
		attr.Region = fromNullString(region)
		attr.AccountID = fromNullString(accountID)

		attributes = append(attributes, &attr)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if limit > 0 && len(attributes) == limit {
		newNextToken = fmt.Sprintf("%d", offset+limit)
	}

	return attributes, newNextToken, nil
}
