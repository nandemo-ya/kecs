package api

import (
	"encoding/json"
	"net/http"
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

	// TODO: Implement actual account setting update logic

	// For now, return a mock response
	resp := PutAccountSettingResponse{
		Setting: AccountSetting{
			Name:  req.Name,
			Value: req.Value,
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

	// TODO: Implement actual account setting default update logic

	// For now, return a mock response
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

	// TODO: Implement actual account setting deletion logic

	// For now, return a mock response
	resp := DeleteAccountSettingResponse{
		Setting: AccountSetting{
			Name:  req.Name,
			Value: "",
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

	// TODO: Implement actual account setting listing logic

	// For now, return a mock response with some default settings
	resp := ListAccountSettingsResponse{
		Settings: []AccountSetting{
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
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
