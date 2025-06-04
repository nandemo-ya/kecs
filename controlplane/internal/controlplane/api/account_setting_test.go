package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// mockAccountSettingStore implements storage.AccountSettingStore for testing
type mockAccountSettingStore struct {
	settings map[string]*storage.AccountSetting
}

func newMockAccountSettingStore() *mockAccountSettingStore {
	return &mockAccountSettingStore{
		settings: make(map[string]*storage.AccountSetting),
	}
}

func (m *mockAccountSettingStore) Upsert(ctx context.Context, setting *storage.AccountSetting) error {
	key := setting.PrincipalARN + ":" + setting.Name
	m.settings[key] = setting
	return nil
}

func (m *mockAccountSettingStore) Get(ctx context.Context, principalARN, name string) (*storage.AccountSetting, error) {
	key := principalARN + ":" + name
	setting, ok := m.settings[key]
	if !ok {
		return nil, nil
	}
	return setting, nil
}

func (m *mockAccountSettingStore) GetDefault(ctx context.Context, name string) (*storage.AccountSetting, error) {
	key := "default:" + name
	setting, ok := m.settings[key]
	if !ok {
		return nil, nil
	}
	return setting, nil
}

func (m *mockAccountSettingStore) List(ctx context.Context, filters storage.AccountSettingFilters) ([]*storage.AccountSetting, string, error) {
	var results []*storage.AccountSetting
	for _, setting := range m.settings {
		// Apply filters
		if filters.Name != "" && setting.Name != filters.Name {
			continue
		}
		if filters.Value != "" && setting.Value != filters.Value {
			continue
		}
		if filters.PrincipalARN != "" && !filters.EffectiveSettings && setting.PrincipalARN != filters.PrincipalARN {
			continue
		}
		results = append(results, setting)
	}
	return results, "", nil
}

func (m *mockAccountSettingStore) Delete(ctx context.Context, principalARN, name string) error {
	key := principalARN + ":" + name
	delete(m.settings, key)
	return nil
}

func (m *mockAccountSettingStore) SetDefault(ctx context.Context, name, value string) error {
	setting := &storage.AccountSetting{
		Name:         name,
		Value:        value,
		PrincipalARN: "default",
		IsDefault:    true,
		Region:       "us-east-1",
		AccountID:    "000000000000",
	}
	key := "default:" + name
	m.settings[key] = setting
	return nil
}

// mockStorage implements storage.Storage for testing
type mockStorage struct {
	accountSettingStore storage.AccountSettingStore
}

func (m *mockStorage) Initialize(ctx context.Context) error { return nil }
func (m *mockStorage) Close() error { return nil }
func (m *mockStorage) ClusterStore() storage.ClusterStore { return nil }
func (m *mockStorage) TaskDefinitionStore() storage.TaskDefinitionStore { return nil }
func (m *mockStorage) ServiceStore() storage.ServiceStore { return nil }
func (m *mockStorage) TaskStore() storage.TaskStore { return nil }
func (m *mockStorage) AccountSettingStore() storage.AccountSettingStore { return m.accountSettingStore }
func (m *mockStorage) BeginTx(ctx context.Context) (storage.Transaction, error) { return nil, nil }

func TestPutAccountSetting(t *testing.T) {
	// Create mock storage
	mockStore := newMockAccountSettingStore()
	mockStrg := &mockStorage{accountSettingStore: mockStore}
	
	// Create server
	server := &Server{
		storage:   mockStrg,
		region:    "us-east-1",
		accountID: "123456789012",
	}
	
	// Test cases
	tests := []struct {
		name           string
		request        PutAccountSettingRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid setting",
			request: PutAccountSettingRequest{
				Name:  "serviceLongArnFormat",
				Value: "enabled",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid setting name",
			request: PutAccountSettingRequest{
				Name:  "invalidSetting",
				Value: "enabled",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid setting name: invalidSetting",
		},
		{
			name: "Invalid value",
			request: PutAccountSettingRequest{
				Name:  "serviceLongArnFormat",
				Value: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid value: invalid. Must be 'enabled' or 'disabled'",
		},
		{
			name: "With principal ARN",
			request: PutAccountSettingRequest{
				Name:         "taskLongArnFormat",
				Value:        "disabled",
				PrincipalArn: "arn:aws:iam::123456789012:user/test",
			},
			expectedStatus: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/v1/putaccountsetting", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			
			// Create response recorder
			rr := httptest.NewRecorder()
			
			// Handle request
			server.handlePutAccountSetting(rr, req)
			
			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
			
			// Check error message if expected
			if tt.expectedError != "" {
				if body := rr.Body.String(); !contains(body, tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, body)
				}
			}
			
			// Check successful response
			if tt.expectedStatus == http.StatusOK {
				var resp PutAccountSettingResponse
				if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp.Setting.Name != tt.request.Name {
					t.Errorf("Expected name %s, got %s", tt.request.Name, resp.Setting.Name)
				}
				if resp.Setting.Value != tt.request.Value {
					t.Errorf("Expected value %s, got %s", tt.request.Value, resp.Setting.Value)
				}
			}
		})
	}
}

func TestPutAccountSettingDefault(t *testing.T) {
	// Create mock storage
	mockStore := newMockAccountSettingStore()
	mockStrg := &mockStorage{accountSettingStore: mockStore}
	
	// Create server
	server := &Server{
		storage:   mockStrg,
		region:    "us-east-1",
		accountID: "123456789012",
	}
	
	// Test valid request
	request := PutAccountSettingDefaultRequest{
		Name:  "containerInsights",
		Value: "enabled",
	}
	
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/v1/putaccountsettingdefault", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	server.handlePutAccountSettingDefault(rr, req)
	
	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	
	// Check response
	var resp PutAccountSettingDefaultResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Setting.Name != request.Name {
		t.Errorf("Expected name %s, got %s", request.Name, resp.Setting.Name)
	}
	if resp.Setting.Value != request.Value {
		t.Errorf("Expected value %s, got %s", request.Value, resp.Setting.Value)
	}
	
	// Verify it was stored as default
	setting, _ := mockStore.GetDefault(context.Background(), request.Name)
	if setting == nil {
		t.Error("Default setting was not stored")
	} else if !setting.IsDefault {
		t.Error("Setting was not marked as default")
	}
}

func TestDeleteAccountSetting(t *testing.T) {
	// Create mock storage with a setting
	mockStore := newMockAccountSettingStore()
	mockStrg := &mockStorage{accountSettingStore: mockStore}
	
	// Add a setting to delete
	testSetting := &storage.AccountSetting{
		Name:         "serviceLongArnFormat",
		Value:        "enabled",
		PrincipalARN: "arn:aws:iam::123456789012:root",
		Region:       "us-east-1",
		AccountID:    "123456789012",
	}
	mockStore.Upsert(context.Background(), testSetting)
	
	// Create server
	server := &Server{
		storage:   mockStrg,
		region:    "us-east-1",
		accountID: "123456789012",
	}
	
	// Test delete request
	request := DeleteAccountSettingRequest{
		Name: "serviceLongArnFormat",
	}
	
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/v1/deleteaccountsetting", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	server.handleDeleteAccountSetting(rr, req)
	
	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	
	// Check response
	var resp DeleteAccountSettingResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp.Setting.Name != request.Name {
		t.Errorf("Expected name %s, got %s", request.Name, resp.Setting.Name)
	}
	
	// Verify it was deleted
	setting, _ := mockStore.Get(context.Background(), "arn:aws:iam::123456789012:root", request.Name)
	if setting != nil {
		t.Error("Setting was not deleted")
	}
}

func TestListAccountSettings(t *testing.T) {
	// Create mock storage with some settings
	mockStore := newMockAccountSettingStore()
	mockStrg := &mockStorage{accountSettingStore: mockStore}
	
	// Add some settings
	settings := []*storage.AccountSetting{
		{
			Name:         "serviceLongArnFormat",
			Value:        "enabled",
			PrincipalARN: "arn:aws:iam::123456789012:root",
			Region:       "us-east-1",
			AccountID:    "123456789012",
		},
		{
			Name:         "taskLongArnFormat",
			Value:        "disabled",
			PrincipalARN: "arn:aws:iam::123456789012:root",
			Region:       "us-east-1",
			AccountID:    "123456789012",
		},
	}
	for _, s := range settings {
		mockStore.Upsert(context.Background(), s)
	}
	
	// Create server
	server := &Server{
		storage:   mockStrg,
		region:    "us-east-1",
		accountID: "123456789012",
	}
	
	// Test list request
	request := ListAccountSettingsRequest{}
	
	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/v1/listaccountsettings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	server.handleListAccountSettings(rr, req)
	
	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
	
	// Check response
	var resp ListAccountSettingsResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if len(resp.Settings) != 2 {
		t.Errorf("Expected 2 settings, got %d", len(resp.Settings))
	}
}

// Helper function
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}