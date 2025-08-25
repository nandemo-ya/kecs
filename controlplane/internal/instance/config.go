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

package instance

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// InstanceConfig represents the configuration for a KECS instance
type InstanceConfig struct {
	// Basic information
	Name      string    `yaml:"name"`
	CreatedAt time.Time `yaml:"createdAt"`

	// Port configuration
	APIPort   int `yaml:"apiPort"`
	AdminPort int `yaml:"adminPort"`

	// Feature toggles
	LocalStack bool `yaml:"localStack"`
	Traefik    bool `yaml:"traefik"`

	// Data directory
	DataDir string `yaml:"dataDir"`
}

// SaveInstanceConfig saves the instance configuration to a YAML file
func SaveInstanceConfig(instanceName string, opts StartOptions) error {
	// Create instance directory if it doesn't exist
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	instanceDir := filepath.Join(home, ".kecs", "instances", instanceName)
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return fmt.Errorf("failed to create instance directory: %w", err)
	}

	// Create config
	config := InstanceConfig{
		Name:       instanceName,
		CreatedAt:  time.Now(),
		APIPort:    opts.ApiPort,
		AdminPort:  opts.AdminPort,
		LocalStack: !opts.NoLocalStack,
		Traefik:    !opts.NoTraefik,
		DataDir:    opts.DataDir,
	}

	// If DataDir is empty, set default
	if config.DataDir == "" {
		config.DataDir = filepath.Join(instanceDir, "data")
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	configPath := filepath.Join(instanceDir, "config.yaml")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadInstanceConfig loads the instance configuration from a YAML file
func LoadInstanceConfig(instanceName string) (*InstanceConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".kecs", "instances", instanceName, "config.yaml")

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("instance config not found: %s", instanceName)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal YAML
	var config InstanceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// ListInstanceConfigs lists all saved instance configurations
func ListInstanceConfigs() ([]InstanceConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	instancesDir := filepath.Join(home, ".kecs", "instances")

	// Check if instances directory exists
	if _, err := os.Stat(instancesDir); os.IsNotExist(err) {
		return []InstanceConfig{}, nil
	}

	// Read directory entries
	entries, err := os.ReadDir(instancesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read instances directory: %w", err)
	}

	var configs []InstanceConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to load config
		config, err := LoadInstanceConfig(entry.Name())
		if err != nil {
			// Skip instances without config
			continue
		}

		configs = append(configs, *config)
	}

	return configs, nil
}
