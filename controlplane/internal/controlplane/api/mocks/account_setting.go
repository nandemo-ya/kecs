package mocks

import (
	"context"
	"errors"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockAccountSettingStore implements storage.AccountSettingStore for testing
type MockAccountSettingStore struct {
	settings map[string]*storage.AccountSetting
}

func NewMockAccountSettingStore() *MockAccountSettingStore {
	return &MockAccountSettingStore{
		settings: make(map[string]*storage.AccountSetting),
	}
}

func (m *MockAccountSettingStore) Upsert(ctx context.Context, setting *storage.AccountSetting) error {
	if m.settings == nil {
		m.settings = make(map[string]*storage.AccountSetting)
	}
	key := fmt.Sprintf("%s:%s", setting.PrincipalARN, setting.Name)
	m.settings[key] = setting
	return nil
}

func (m *MockAccountSettingStore) Get(ctx context.Context, principalARN, name string) (*storage.AccountSetting, error) {
	key := fmt.Sprintf("%s:%s", principalARN, name)
	setting, exists := m.settings[key]
	if !exists {
		return nil, errors.New("account setting not found")
	}
	return setting, nil
}

func (m *MockAccountSettingStore) GetDefault(ctx context.Context, name string) (*storage.AccountSetting, error) {
	key := fmt.Sprintf("default:%s", name)
	setting, exists := m.settings[key]
	if !exists {
		return nil, errors.New("default account setting not found")
	}
	return setting, nil
}

func (m *MockAccountSettingStore) List(ctx context.Context, filters storage.AccountSettingFilters) ([]*storage.AccountSetting, string, error) {
	var results []*storage.AccountSetting
	for _, setting := range m.settings {
		// Apply filters
		if filters.Name != "" && setting.Name != filters.Name {
			continue
		}
		if filters.Value != "" && setting.Value != filters.Value {
			continue
		}
		if filters.PrincipalARN != "" && setting.PrincipalARN != filters.PrincipalARN {
			continue
		}
		results = append(results, setting)
	}
	return results, "", nil
}

func (m *MockAccountSettingStore) Delete(ctx context.Context, principalARN, name string) error {
	key := fmt.Sprintf("%s:%s", principalARN, name)
	if _, exists := m.settings[key]; !exists {
		return errors.New("account setting not found")
	}
	delete(m.settings, key)
	return nil
}

func (m *MockAccountSettingStore) SetDefault(ctx context.Context, name, value string) error {
	setting := &storage.AccountSetting{
		Name:         name,
		Value:        value,
		PrincipalARN: "default",
		IsDefault:    true,
	}
	key := fmt.Sprintf("default:%s", name)
	m.settings[key] = setting
	return nil
}
