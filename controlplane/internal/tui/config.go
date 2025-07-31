// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tui

import (
	"os"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

// Config holds TUI configuration
type Config struct {
	// APIEndpoint is the base URL for the KECS API
	APIEndpoint string
	
	// UseMockData determines whether to use mock data or real API
	UseMockData bool
}

// LoadConfig loads TUI configuration from environment variables
func LoadConfig() Config {
	cfg := Config{
		APIEndpoint: os.Getenv("KECS_API_ENDPOINT"),
		UseMockData: true, // Default to mock data
	}
	
	// Use real API if endpoint is set
	if cfg.APIEndpoint != "" {
		cfg.UseMockData = false
	}
	
	// Allow explicit mock mode
	if os.Getenv("KECS_TUI_MOCK") == "true" {
		cfg.UseMockData = true
	}
	
	// Default endpoint if not set
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = "http://localhost:8080"
	}
	
	return cfg
}

// CreateAPIClient creates an API client based on configuration
func CreateAPIClient(cfg Config) api.Client {
	if cfg.UseMockData {
		return api.NewMockClient()
	}
	return api.NewHTTPClient(cfg.APIEndpoint)
}