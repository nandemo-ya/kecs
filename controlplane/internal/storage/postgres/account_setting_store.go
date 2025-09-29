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

type accountSettingStore struct {
	db *sql.DB
}

// Upsert creates or updates an account setting
func (s *accountSettingStore) Upsert(ctx context.Context, setting *storage.AccountSetting) error {
	if setting.ID == "" {
		setting.ID = uuid.New().String()
	}

	now := time.Now()
	if setting.CreatedAt.IsZero() {
		setting.CreatedAt = now
	}
	setting.UpdatedAt = now

	query := `
	INSERT INTO account_settings (
		id, name, value, principal_arn, is_default,
		region, account_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9
	) ON CONFLICT (principal_arn, name) DO UPDATE SET
		value = EXCLUDED.value,
		is_default = EXCLUDED.is_default,
		region = EXCLUDED.region,
		account_id = EXCLUDED.account_id,
		updated_at = EXCLUDED.updated_at`

	_, err := s.db.ExecContext(ctx, query,
		setting.ID, setting.Name, setting.Value,
		setting.PrincipalARN, setting.IsDefault,
		toNullString(setting.Region), toNullString(setting.AccountID),
		setting.CreatedAt, setting.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert account setting: %w", err)
	}

	return nil
}

// Get retrieves an account setting by principal ARN and name
func (s *accountSettingStore) Get(ctx context.Context, principalARN, name string) (*storage.AccountSetting, error) {
	query := `
	SELECT id, name, value, principal_arn, is_default,
		region, account_id, created_at, updated_at
	FROM account_settings
	WHERE principal_arn = $1 AND name = $2`

	var setting storage.AccountSetting
	var region, accountID sql.NullString

	err := s.db.QueryRowContext(ctx, query, principalARN, name).Scan(
		&setting.ID, &setting.Name, &setting.Value,
		&setting.PrincipalARN, &setting.IsDefault,
		&region, &accountID,
		&setting.CreatedAt, &setting.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get account setting: %w", err)
	}

	setting.Region = fromNullString(region)
	setting.AccountID = fromNullString(accountID)

	return &setting, nil
}

// GetDefault retrieves the default account setting by name
func (s *accountSettingStore) GetDefault(ctx context.Context, name string) (*storage.AccountSetting, error) {
	query := `
	SELECT id, name, value, principal_arn, is_default,
		region, account_id, created_at, updated_at
	FROM account_settings
	WHERE name = $1 AND is_default = true`

	var setting storage.AccountSetting
	var region, accountID sql.NullString

	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&setting.ID, &setting.Name, &setting.Value,
		&setting.PrincipalARN, &setting.IsDefault,
		&region, &accountID,
		&setting.CreatedAt, &setting.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, storage.ErrResourceNotFound
		}
		return nil, fmt.Errorf("failed to get default account setting: %w", err)
	}

	setting.Region = fromNullString(region)
	setting.AccountID = fromNullString(accountID)

	return &setting, nil
}

// List retrieves account settings with filtering
func (s *accountSettingStore) List(ctx context.Context, filters storage.AccountSettingFilters) ([]*storage.AccountSetting, string, error) {
	// Parse the next token to get offset
	offset := 0
	if filters.NextToken != "" {
		if _, err := fmt.Sscanf(filters.NextToken, "%d", &offset); err != nil {
			return nil, "", fmt.Errorf("invalid next token: %w", err)
		}
	}

	query := `
	SELECT id, name, value, principal_arn, is_default,
		region, account_id, created_at, updated_at
	FROM account_settings
	WHERE 1=1`

	args := []interface{}{}
	argNum := 1

	if filters.Name != "" {
		query += fmt.Sprintf(" AND name = $%d", argNum)
		args = append(args, filters.Name)
		argNum++
	}

	if filters.Value != "" {
		query += fmt.Sprintf(" AND value = $%d", argNum)
		args = append(args, filters.Value)
		argNum++
	}

	if filters.PrincipalARN != "" {
		query += fmt.Sprintf(" AND principal_arn = $%d", argNum)
		args = append(args, filters.PrincipalARN)
		argNum++
	}

	// Handle effective settings
	if filters.EffectiveSettings {
		// When requesting effective settings, we need to get both defaults and overrides
		// This would typically be handled at a higher layer by combining results
		// For now, we'll just return all settings for the principal
	}

	query += " ORDER BY name, principal_arn"

	if filters.MaxResults > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argNum, argNum+1)
		args = append(args, filters.MaxResults, offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list account settings: %w", err)
	}
	defer rows.Close()

	var settings []*storage.AccountSetting
	for rows.Next() {
		var setting storage.AccountSetting
		var region, accountID sql.NullString

		err := rows.Scan(
			&setting.ID, &setting.Name, &setting.Value,
			&setting.PrincipalARN, &setting.IsDefault,
			&region, &accountID,
			&setting.CreatedAt, &setting.UpdatedAt,
		)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan account setting: %w", err)
		}

		setting.Region = fromNullString(region)
		setting.AccountID = fromNullString(accountID)

		settings = append(settings, &setting)
	}

	// Generate next token if there are more results
	newNextToken := ""
	if filters.MaxResults > 0 && len(settings) == filters.MaxResults {
		newNextToken = fmt.Sprintf("%d", offset+filters.MaxResults)
	}

	return settings, newNextToken, nil
}

// Delete removes an account setting
func (s *accountSettingStore) Delete(ctx context.Context, principalARN, name string) error {
	query := `
	DELETE FROM account_settings
	WHERE principal_arn = $1 AND name = $2`

	result, err := s.db.ExecContext(ctx, query, principalARN, name)
	if err != nil {
		return fmt.Errorf("failed to delete account setting: %w", err)
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

// SetDefault sets a default account setting
func (s *accountSettingStore) SetDefault(ctx context.Context, name, value string) error {
	setting := &storage.AccountSetting{
		ID:           uuid.New().String(),
		Name:         name,
		Value:        value,
		PrincipalARN: "default",
		IsDefault:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	query := `
	INSERT INTO account_settings (
		id, name, value, principal_arn, is_default,
		region, account_id, created_at, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9
	) ON CONFLICT (principal_arn, name) DO UPDATE SET
		value = EXCLUDED.value,
		updated_at = EXCLUDED.updated_at`

	_, err := s.db.ExecContext(ctx, query,
		setting.ID, setting.Name, setting.Value,
		setting.PrincipalARN, setting.IsDefault,
		nil, nil, // region and account_id are null for defaults
		setting.CreatedAt, setting.UpdatedAt,
	)

	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return storage.ErrResourceAlreadyExists
		}
		return fmt.Errorf("failed to set default account setting: %w", err)
	}

	return nil
}
