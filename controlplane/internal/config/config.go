package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// Config represents the KECS configuration
type Config struct {
	Server     ServerConfig      `yaml:"server"`
	LocalStack localstack.Config `yaml:"localstack"`
}

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	Port      int    `yaml:"port"`
	AdminPort int    `yaml:"adminPort"`
	DataDir   string `yaml:"dataDir"`
	LogLevel  string `yaml:"logLevel"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultDataDir := filepath.Join(homeDir, ".kecs", "data")

	return &Config{
		Server: ServerConfig{
			Port:      8080,
			AdminPort: 8081,
			DataDir:   defaultDataDir,
			LogLevel:  "info",
		},
		LocalStack: *localstack.DefaultConfig(),
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	if configPath == "" {
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// If config file doesn't exist, return default config
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.AdminPort <= 0 || c.Server.AdminPort > 65535 {
		return fmt.Errorf("invalid admin port: %d", c.Server.AdminPort)
	}

	// Validate LocalStack config if enabled
	if c.LocalStack.Enabled {
		if err := c.LocalStack.Validate(); err != nil {
			return fmt.Errorf("invalid LocalStack config: %w", err)
		}
	}

	return nil
}
