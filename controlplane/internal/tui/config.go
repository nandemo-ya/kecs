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
	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/tui/api"
)

// Config holds TUI configuration
type Config struct {
	// APIEndpoint is the base URL for the KECS API
	APIEndpoint string
}

// LoadConfig loads TUI configuration from environment variables
func LoadConfig() Config {
	cfg := Config{
		APIEndpoint: config.GetString("server.endpoint"),
	}

	// Default endpoint if not set
	if cfg.APIEndpoint == "" {
		cfg.APIEndpoint = "http://localhost:5373"
	}

	return cfg
}

// CreateAPIClient creates an API client based on configuration
func CreateAPIClient(cfg Config) api.Client {
	return api.NewHTTPClient(cfg.APIEndpoint)
}
