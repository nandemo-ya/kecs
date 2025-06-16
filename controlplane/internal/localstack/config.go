package localstack

import (
	"fmt"
	"regexp"
	"strings"
)

// DefaultConfig returns the default LocalStack configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     false,
		Services:    DefaultServices,
		Persistence: true,
		Image:       DefaultImage,
		Namespace:   DefaultNamespace,
		Port:        DefaultPort,
		EdgePort:    DefaultEdgePort,
		Resources: ResourceLimits{
			Memory:      "2Gi",
			CPU:         "1000m",
			StorageSize: "10Gi",
		},
		Environment: map[string]string{
			"DEFAULT_REGION":             "us-east-1",
			"DOCKER_HOST":                "unix:///var/run/docker.sock",
			"LAMBDA_EXECUTOR":            "local",
			"DISABLE_CORS_CHECKS":        "1",
			"DISABLE_CUSTOM_CORS_S3":     "1",
			"EXTRA_CORS_ALLOWED_ORIGINS": "*",
			"EXTRA_CORS_ALLOWED_HEADERS": "x-amz-*",
		},
		Debug:           false,
		DataDir:         "/var/lib/localstack",
		CustomEndpoints: make(map[string]string),
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate image
	if c.Image == "" {
		return fmt.Errorf("image cannot be empty")
	}

	// Validate namespace
	if c.Namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	if !isValidKubernetesName(c.Namespace) {
		return fmt.Errorf("invalid namespace name: %s", c.Namespace)
	}

	// Validate ports
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}
	if c.EdgePort <= 0 || c.EdgePort > 65535 {
		return fmt.Errorf("invalid edge port: %d", c.EdgePort)
	}

	// Validate services
	for _, service := range c.Services {
		if !IsValidService(service) {
			return fmt.Errorf("invalid service: %s", service)
		}
	}

	// Validate resources
	if err := c.Resources.Validate(); err != nil {
		return fmt.Errorf("invalid resource limits: %w", err)
	}

	return nil
}

// Validate checks if the resource limits are valid
func (r *ResourceLimits) Validate() error {
	// Validate memory
	if r.Memory != "" {
		if !isValidResourceQuantity(r.Memory) {
			return fmt.Errorf("invalid memory limit: %s", r.Memory)
		}
	}

	// Validate CPU
	if r.CPU != "" {
		if !isValidResourceQuantity(r.CPU) {
			return fmt.Errorf("invalid CPU limit: %s", r.CPU)
		}
	}

	// Validate storage size
	if r.StorageSize != "" {
		if !isValidResourceQuantity(r.StorageSize) {
			return fmt.Errorf("invalid storage size: %s", r.StorageSize)
		}
	}

	return nil
}

// GetServicesString returns a comma-separated string of services
func (c *Config) GetServicesString() string {
	return strings.Join(c.Services, ",")
}

// SetServicesFromString sets services from a comma-separated string
func (c *Config) SetServicesFromString(servicesStr string) error {
	if servicesStr == "" {
		c.Services = []string{}
		return nil
	}

	services := strings.Split(servicesStr, ",")
	for i, service := range services {
		service = strings.TrimSpace(service)
		if !IsValidService(service) {
			return fmt.Errorf("invalid service: %s", service)
		}
		services[i] = service
	}

	c.Services = services
	return nil
}

// MergeEnvironment merges additional environment variables
func (c *Config) MergeEnvironment(env map[string]string) {
	if c.Environment == nil {
		c.Environment = make(map[string]string)
	}

	for k, v := range env {
		c.Environment[k] = v
	}
}

// GetEnvironmentVars returns all environment variables for LocalStack
func (c *Config) GetEnvironmentVars() map[string]string {
	env := make(map[string]string)

	// Copy custom environment variables
	for k, v := range c.Environment {
		env[k] = v
	}

	// Set standard LocalStack environment variables
	env[EnvServices] = c.GetServicesString()
	env[EnvDebug] = boolToString(c.Debug)
	env[EnvPersistence] = boolToString(c.Persistence)
	env[EnvDataDir] = c.DataDir
	env[EnvEdgePort] = fmt.Sprintf("%d", c.EdgePort)

	if c.DockerHost != "" {
		env[EnvDockerHost] = c.DockerHost
	}

	return env
}

// ProxyConfigWithDefaults returns proxy configuration with defaults
func ProxyConfigWithDefaults(endpoint string) *ProxyConfig {
	return &ProxyConfig{
		Mode:               ProxyModeEnvironment,
		LocalStackEndpoint: endpoint,
		FallbackEnabled:    true,
		FallbackOrder:      []ProxyMode{ProxyModeSidecar, ProxyModeEnvironment},
		CustomEndpoints:    make(map[string]string),
	}
}

// Validate checks if the proxy configuration is valid
func (p *ProxyConfig) Validate() error {
	if p.LocalStackEndpoint == "" {
		return fmt.Errorf("localstack endpoint cannot be empty")
	}

	// Validate proxy mode
	switch p.Mode {
	case ProxyModeEnvironment, ProxyModeSidecar, ProxyModeEBPF, ProxyModeDisabled:
		// Valid modes
	default:
		return fmt.Errorf("invalid proxy mode: %s", p.Mode)
	}

	// Validate fallback order
	for _, mode := range p.FallbackOrder {
		switch mode {
		case ProxyModeEnvironment, ProxyModeSidecar, ProxyModeEBPF:
			// Valid modes
		default:
			return fmt.Errorf("invalid fallback mode: %s", mode)
		}
	}

	return nil
}

// Helper functions

func isValidKubernetesName(name string) bool {
	// Kubernetes names must be lowercase alphanumeric or '-', and must start and end with alphanumeric
	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	matched, _ := regexp.MatchString(pattern, name)
	return matched
}

func isValidResourceQuantity(quantity string) bool {
	// Simple validation for Kubernetes resource quantities
	// Examples: 100m, 1000Mi, 2Gi, 0.5
	pattern := `^([0-9]+\.?[0-9]*)(m|Mi|Gi|Ki|M|G|K)?$`
	matched, _ := regexp.MatchString(pattern, quantity)
	return matched
}

func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}