package duckdb

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// accountSettingStore implements storage.AccountSettingStore using DuckDB
type accountSettingStore struct {
	db      *sql.DB
	storage *DuckDBStorage
}

// NewAccountSettingStore creates a new DuckDB-based account setting store
func NewAccountSettingStore(db *sql.DB) storage.AccountSettingStore {
	return &accountSettingStore{db: db}
}

// CreateSchema creates the account_settings table if it doesn't exist
func (s *accountSettingStore) CreateSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS account_settings (
		id VARCHAR PRIMARY KEY,
		name VARCHAR NOT NULL,
		value VARCHAR NOT NULL,
		principal_arn VARCHAR NOT NULL,
		is_default BOOLEAN DEFAULT FALSE,
		region VARCHAR NOT NULL,
		account_id VARCHAR NOT NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		UNIQUE(name, principal_arn, account_id, region)
	)`

	_, err := s.storage.ExecContextWithRecovery(ctx, query)
	return err
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
		id, name, value, principal_arn, is_default, region, account_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT (name, principal_arn, account_id, region) DO UPDATE SET
		value = EXCLUDED.value,
		is_default = EXCLUDED.is_default,
		updated_at = EXCLUDED.updated_at`

	_, err := s.storage.ExecContextWithRecovery(ctx, query,
		setting.ID,
		setting.Name,
		setting.Value,
		setting.PrincipalARN,
		setting.IsDefault,
		setting.Region,
		setting.AccountID,
		setting.CreatedAt,
		setting.UpdatedAt,
	)

	return err
}

// Get retrieves an account setting by principal ARN and name
func (s *accountSettingStore) Get(ctx context.Context, principalARN, name string) (*storage.AccountSetting, error) {
	query := `
	SELECT id, name, value, principal_arn, is_default, region, account_id, created_at, updated_at
	FROM account_settings
	WHERE principal_arn = ? AND name = ?
	LIMIT 1`

	var setting storage.AccountSetting
	err := s.storage.QueryRowContextWithRecovery(ctx, query, principalARN, name).Scan(
		&setting.ID,
		&setting.Name,
		&setting.Value,
		&setting.PrincipalARN,
		&setting.IsDefault,
		&setting.Region,
		&setting.AccountID,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &setting, nil
}

// GetDefault retrieves the default account setting by name
func (s *accountSettingStore) GetDefault(ctx context.Context, name string) (*storage.AccountSetting, error) {
	query := `
	SELECT id, name, value, principal_arn, is_default, region, account_id, created_at, updated_at
	FROM account_settings
	WHERE name = ? AND is_default = true
	LIMIT 1`

	var setting storage.AccountSetting
	err := s.storage.QueryRowContextWithRecovery(ctx, query, name).Scan(
		&setting.ID,
		&setting.Name,
		&setting.Value,
		&setting.PrincipalARN,
		&setting.IsDefault,
		&setting.Region,
		&setting.AccountID,
		&setting.CreatedAt,
		&setting.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &setting, nil
}

// List retrieves account settings with filtering
func (s *accountSettingStore) List(ctx context.Context, filters storage.AccountSettingFilters) ([]*storage.AccountSetting, string, error) {
	// Build the WHERE clause
	var conditions []string
	var args []interface{}
	argCount := 0

	if filters.Name != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("name = $%d", argCount))
		args = append(args, filters.Name)
	}

	if filters.Value != "" {
		argCount++
		conditions = append(conditions, fmt.Sprintf("value = $%d", argCount))
		args = append(args, filters.Value)
	}

	if filters.PrincipalARN != "" && !filters.EffectiveSettings {
		argCount++
		conditions = append(conditions, fmt.Sprintf("principal_arn = $%d", argCount))
		args = append(args, filters.PrincipalARN)
	}

	// Build the query
	query := "SELECT id, name, value, principal_arn, is_default, region, account_id, created_at, updated_at FROM account_settings"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ordering and pagination
	query += " ORDER BY name, principal_arn"

	limit := filters.MaxResults
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	offset := 0
	if filters.NextToken != "" {
		// Decode the next token (base64 encoded offset)
		data, err := base64.StdEncoding.DecodeString(filters.NextToken)
		if err == nil {
			fmt.Sscanf(string(data), "%d", &offset)
		}
	}

	argCount++
	query += fmt.Sprintf(" LIMIT $%d", argCount)
	args = append(args, limit+1) // Fetch one extra to determine if there are more results

	if offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
	}

	// Execute the query
	rows, err := s.storage.QueryContextWithRecovery(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var settings []*storage.AccountSetting
	for rows.Next() {
		var setting storage.AccountSetting
		err := rows.Scan(
			&setting.ID,
			&setting.Name,
			&setting.Value,
			&setting.PrincipalARN,
			&setting.IsDefault,
			&setting.Region,
			&setting.AccountID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, "", err
		}
		settings = append(settings, &setting)
	}

	// Check if there are more results
	var nextToken string
	if len(settings) > limit {
		// Remove the extra result
		settings = settings[:limit]
		// Create next token
		nextOffset := offset + limit
		nextToken = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nextOffset)))
	}

	// If effective settings are requested, merge defaults with principal-specific settings
	if filters.EffectiveSettings && filters.PrincipalARN != "" {
		effectiveSettings, err := s.getEffectiveSettings(ctx, filters.PrincipalARN, filters.Name)
		if err != nil {
			return nil, "", err
		}
		settings = effectiveSettings
	}

	return settings, nextToken, nil
}

// Delete removes an account setting
func (s *accountSettingStore) Delete(ctx context.Context, principalARN, name string) error {
	query := `DELETE FROM account_settings WHERE principal_arn = ? AND name = ?`
	_, err := s.storage.ExecContextWithRecovery(ctx, query, principalARN, name)
	return err
}

// SetDefault sets a default account setting
func (s *accountSettingStore) SetDefault(ctx context.Context, name, value string) error {
	setting := &storage.AccountSetting{
		Name:         name,
		Value:        value,
		PrincipalARN: "default",
		IsDefault:    true,
		Region:       "us-east-1",    // Default region
		AccountID:    "000000000000", // Default account ID
	}
	return s.Upsert(ctx, setting)
}

// getEffectiveSettings returns the effective settings for a principal (defaults + overrides)
func (s *accountSettingStore) getEffectiveSettings(ctx context.Context, principalARN, nameFilter string) ([]*storage.AccountSetting, error) {
	// Build the query to get both defaults and principal-specific settings
	query := `
	WITH defaults AS (
		SELECT * FROM account_settings WHERE is_default = true
	),
	principal_settings AS (
		SELECT * FROM account_settings WHERE principal_arn = ?
	)
	SELECT 
		COALESCE(p.id, d.id) as id,
		COALESCE(p.name, d.name) as name,
		COALESCE(p.value, d.value) as value,
		COALESCE(p.principal_arn, ?) as principal_arn,
		false as is_default,
		COALESCE(p.region, d.region) as region,
		COALESCE(p.account_id, d.account_id) as account_id,
		COALESCE(p.created_at, d.created_at) as created_at,
		COALESCE(p.updated_at, d.updated_at) as updated_at
	FROM defaults d
	LEFT JOIN principal_settings p ON d.name = p.name`

	args := []interface{}{principalARN, principalARN}

	if nameFilter != "" {
		query += " WHERE COALESCE(p.name, d.name) = ?"
		args = append(args, nameFilter)
	}

	query += " ORDER BY name"

	rows, err := s.storage.QueryContextWithRecovery(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*storage.AccountSetting
	for rows.Next() {
		var setting storage.AccountSetting
		err := rows.Scan(
			&setting.ID,
			&setting.Name,
			&setting.Value,
			&setting.PrincipalARN,
			&setting.IsDefault,
			&setting.Region,
			&setting.AccountID,
			&setting.CreatedAt,
			&setting.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		settings = append(settings, &setting)
	}

	return settings, nil
}
