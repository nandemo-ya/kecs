package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ContainerConfig represents the configuration for a KECS container instance
type ContainerConfig struct {
	Name      string                 `yaml:"name"`
	Image     string                 `yaml:"image,omitempty"`
	Ports     ContainerPortConfig    `yaml:"ports,omitempty"`
	DataDir   string                 `yaml:"dataDir,omitempty"`
	Env       map[string]string      `yaml:"env,omitempty"`
	Labels    map[string]string      `yaml:"labels,omitempty"`
	AutoStart bool                   `yaml:"autoStart,omitempty"`
}

// ContainerPortConfig represents port configuration
type ContainerPortConfig struct {
	API   int `yaml:"api,omitempty"`
	Admin int `yaml:"admin,omitempty"`
}

// InstancesConfig represents multiple KECS instances configuration
type InstancesConfig struct {
	DefaultInstance string            `yaml:"defaultInstance,omitempty"`
	Instances       []ContainerConfig `yaml:"instances"`
}

// LoadContainerConfig loads container configuration from a file
func LoadContainerConfig(configPath string) (*InstancesConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config InstancesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and set defaults
	for i := range config.Instances {
		instance := &config.Instances[i]
		
		// Set default name if not specified
		if instance.Name == "" {
			instance.Name = fmt.Sprintf("kecs-instance-%d", i+1)
		}
		
		// Set default image if not specified
		if instance.Image == "" {
			instance.Image = defaultImage
		}
		
		// Set default ports if not specified
		if instance.Ports.API == 0 {
			instance.Ports.API = 8080 + i*10
		}
		if instance.Ports.Admin == 0 {
			instance.Ports.Admin = 8081 + i*10
		}
		
		// Set default data directory if not specified
		if instance.DataDir == "" {
			homeDir, _ := os.UserHomeDir()
			instance.DataDir = filepath.Join(homeDir, ".kecs", "instances", instance.Name, "data")
		}
	}

	// Set default instance if not specified
	if config.DefaultInstance == "" && len(config.Instances) > 0 {
		config.DefaultInstance = config.Instances[0].Name
	}

	return &config, nil
}

// SaveContainerConfig saves container configuration to a file
func SaveContainerConfig(configPath string, config *InstancesConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfigPath returns the default configuration file path
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "kecs-instances.yaml"
	}
	return filepath.Join(homeDir, ".kecs", "instances.yaml")
}