package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// AccountSetting represents an ECS account setting
type AccountSetting struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PutAccountSettingRequest represents the request to put an account setting
type PutAccountSettingRequest struct {
	Name      string `json:"name"`
	Value     string `json:"value"`
	PrincipalArn string `json:"principalArn,omitempty"`
}

// PutAccountSettingResponse represents the response from putting an account setting
type PutAccountSettingResponse struct {
	Setting AccountSetting `json:"setting"`
}

// PutAccountSettingDefaultRequest represents the request to put an account setting default
type PutAccountSettingDefaultRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PutAccountSettingDefaultResponse represents the response from putting an account setting default
type PutAccountSettingDefaultResponse struct {
	Setting AccountSetting `json:"setting"`
}

// DeleteAccountSettingRequest represents the request to delete an account setting
type DeleteAccountSettingRequest struct {
	Name         string `json:"name"`
	PrincipalArn string `json:"principalArn,omitempty"`
}

// DeleteAccountSettingResponse represents the response from deleting an account setting
type DeleteAccountSettingResponse struct {
	Setting AccountSetting `json:"setting"`
}

// ListAccountSettingsRequest represents the request to list account settings
type ListAccountSettingsRequest struct {
	Name         string `json:"name,omitempty"`
	Value        string `json:"value,omitempty"`
	PrincipalArn string `json:"principalArn,omitempty"`
	EffectiveSettings bool `json:"effectiveSettings,omitempty"`
	NextToken    string `json:"nextToken,omitempty"`
	MaxResults   int    `json:"maxResults,omitempty"`
}

// ListAccountSettingsResponse represents the response from listing account settings
type ListAccountSettingsResponse struct {
	Settings  []AccountSetting `json:"settings"`
	NextToken string           `json:"nextToken,omitempty"`
}

// registerAccountSettingEndpoints registers all account setting-related API endpoints
func (s *Server) registerAccountSettingEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/putaccountsetting", s.handlePutAccountSetting)
	mux.HandleFunc("/v1/putaccountsettingdefault", s.handlePutAccountSettingDefault)
	mux.HandleFunc("/v1/deleteaccountsetting", s.handleDeleteAccountSetting)
	mux.HandleFunc("/v1/listaccountsettings", s.handleListAccountSettings)
}

// handlePutAccountSetting handles the PutAccountSetting API endpoint
func (s *Server) handlePutAccountSetting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PutAccountSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the setting name
	validSettings := []string{
		"serviceLongArnFormat",
		"taskLongArnFormat",
		"containerInstanceLongArnFormat",
		"awsvpcTrunking",
		"containerInsights",
		"fargateFIPSMode",
		"guardDutyActivate",
		"tagResourceAuthorization",
	}
	
	isValid := false
	for _, valid := range validSettings {
		if req.Name == valid {
			isValid = true
			break
		}
	}
	
	if !isValid {
		http.Error(w, fmt.Sprintf("Invalid setting name: %s", req.Name), http.StatusBadRequest)
		return
	}

	// Validate the value
	if req.Value != "enabled" && req.Value != "disabled" {
		http.Error(w, fmt.Sprintf("Invalid value: %s. Must be 'enabled' or 'disabled'", req.Value), http.StatusBadRequest)
		return
	}

	// Use principal ARN if provided, otherwise use default
	principalArn := req.PrincipalArn
	if principalArn == "" {
		principalArn = fmt.Sprintf("arn:aws:iam::%s:root", s.accountID)
	}

	// Create the account setting
	setting := &storage.AccountSetting{
		Name:         req.Name,
		Value:        req.Value,
		PrincipalARN: principalArn,
		IsDefault:    false,
		Region:       s.region,
		AccountID:    s.accountID,
	}

	// Store in database
	if err := s.storage.AccountSettingStore().Upsert(r.Context(), setting); err != nil {
		log.Printf("Failed to store account setting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	resp := PutAccountSettingResponse{
		Setting: AccountSetting{
			Name:  setting.Name,
			Value: setting.Value,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handlePutAccountSettingDefault handles the PutAccountSettingDefault API endpoint
func (s *Server) handlePutAccountSettingDefault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PutAccountSettingDefaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the setting name
	validSettings := []string{
		"serviceLongArnFormat",
		"taskLongArnFormat",
		"containerInstanceLongArnFormat",
		"awsvpcTrunking",
		"containerInsights",
		"fargateFIPSMode",
		"guardDutyActivate",
		"tagResourceAuthorization",
	}
	
	isValid := false
	for _, valid := range validSettings {
		if req.Name == valid {
			isValid = true
			break
		}
	}
	
	if !isValid {
		http.Error(w, fmt.Sprintf("Invalid setting name: %s", req.Name), http.StatusBadRequest)
		return
	}

	// Validate the value
	if req.Value != "enabled" && req.Value != "disabled" {
		http.Error(w, fmt.Sprintf("Invalid value: %s. Must be 'enabled' or 'disabled'", req.Value), http.StatusBadRequest)
		return
	}

	// Set default account setting
	if err := s.storage.AccountSettingStore().SetDefault(r.Context(), req.Name, req.Value); err != nil {
		log.Printf("Failed to set default account setting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	resp := PutAccountSettingDefaultResponse{
		Setting: AccountSetting{
			Name:  req.Name,
			Value: req.Value,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteAccountSetting handles the DeleteAccountSetting API endpoint
func (s *Server) handleDeleteAccountSetting(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteAccountSettingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use principal ARN if provided, otherwise use default
	principalArn := req.PrincipalArn
	if principalArn == "" {
		principalArn = fmt.Sprintf("arn:aws:iam::%s:root", s.accountID)
	}

	// Get the setting before deletion for the response
	setting, err := s.storage.AccountSettingStore().Get(r.Context(), principalArn, req.Name)
	if err != nil {
		log.Printf("Failed to get account setting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if setting == nil {
		http.Error(w, fmt.Sprintf("Setting not found: %s", req.Name), http.StatusNotFound)
		return
	}

	// Delete the setting
	if err := s.storage.AccountSettingStore().Delete(r.Context(), principalArn, req.Name); err != nil {
		log.Printf("Failed to delete account setting: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Return response
	resp := DeleteAccountSettingResponse{
		Setting: AccountSetting{
			Name:  setting.Name,
			Value: setting.Value,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListAccountSettings handles the ListAccountSettings API endpoint
func (s *Server) handleListAccountSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListAccountSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use principal ARN if provided, otherwise use default
	principalArn := req.PrincipalArn
	if principalArn == "" && !req.EffectiveSettings {
		principalArn = fmt.Sprintf("arn:aws:iam::%s:root", s.accountID)
	}

	// Build filters
	filters := storage.AccountSettingFilters{
		Name:              req.Name,
		Value:             req.Value,
		PrincipalARN:      principalArn,
		EffectiveSettings: req.EffectiveSettings,
		MaxResults:        req.MaxResults,
		NextToken:         req.NextToken,
	}

	// List settings from storage
	settings, nextToken, err := s.storage.AccountSettingStore().List(r.Context(), filters)
	if err != nil {
		log.Printf("Failed to list account settings: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert to API response format
	apiSettings := make([]AccountSetting, 0, len(settings))
	for _, setting := range settings {
		apiSettings = append(apiSettings, AccountSetting{
			Name:  setting.Name,
			Value: setting.Value,
		})
	}

	// If no settings found and we haven't set defaults yet, return common defaults
	if len(apiSettings) == 0 && req.EffectiveSettings {
		apiSettings = []AccountSetting{
			{
				Name:  "serviceLongArnFormat",
				Value: "enabled",
			},
			{
				Name:  "taskLongArnFormat",
				Value: "enabled",
			},
			{
				Name:  "containerInstanceLongArnFormat",
				Value: "enabled",
			},
		}
	}

	// Return response
	resp := ListAccountSettingsResponse{
		Settings:  apiSettings,
		NextToken: nextToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
