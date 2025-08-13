package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// Config represents the KECS configuration
type Config struct {
	Server     ServerConfig      `yaml:"server"`
	LocalStack localstack.Config `yaml:"localstack"`
	Kubernetes KubernetesConfig  `yaml:"kubernetes"`
	Features   FeaturesConfig    `yaml:"features"`
	AWS        AWSConfig         `yaml:"aws"`
}

// ServerConfig represents server-specific configuration
type ServerConfig struct {
	Port              int      `yaml:"port" mapstructure:"port"`
	AdminPort         int      `yaml:"adminPort" mapstructure:"adminPort"`
	DataDir           string   `yaml:"dataDir" mapstructure:"dataDir"`
	LogLevel          string   `yaml:"logLevel" mapstructure:"logLevel"`
	AllowedOrigins    []string `yaml:"allowedOrigins" mapstructure:"allowedOrigins"`
	Endpoint          string   `yaml:"endpoint" mapstructure:"endpoint"`
	ControlPlaneImage string   `yaml:"controlPlaneImage" mapstructure:"controlPlaneImage"`
}

// KubernetesConfig represents Kubernetes-related configuration
type KubernetesConfig struct {
	KubeconfigPath         string `yaml:"kubeconfigPath" mapstructure:"kubeconfigPath"`
	K3DOptimized           bool   `yaml:"k3dOptimized" mapstructure:"k3dOptimized"`
	K3DAsync               bool   `yaml:"k3dAsync" mapstructure:"k3dAsync"`
	DisableCoreDNS         bool   `yaml:"disableCoreDNS" mapstructure:"disableCoreDNS"`
	KeepClustersOnShutdown bool   `yaml:"keepClustersOnShutdown" mapstructure:"keepClustersOnShutdown"`
}

// FeaturesConfig represents feature toggles
type FeaturesConfig struct {
	TestMode         bool `yaml:"testMode" mapstructure:"testMode"`
	ContainerMode    bool `yaml:"containerMode" mapstructure:"containerMode"`
	AutoRecoverState bool `yaml:"autoRecoverState" mapstructure:"autoRecoverState"`
	DevMode          bool `yaml:"devMode" mapstructure:"devMode"`
	Traefik          bool `yaml:"traefik" mapstructure:"traefik"`
	IAMIntegration   bool `yaml:"iamIntegration" mapstructure:"iamIntegration"`
	TUIMock          bool `yaml:"tuiMock" mapstructure:"tuiMock"`
}

// AWSConfig represents AWS-related configuration
type AWSConfig struct {
	DefaultRegion string `yaml:"defaultRegion" mapstructure:"defaultRegion"`
	AccountID     string `yaml:"accountID" mapstructure:"accountID"`
	ProxyImage    string `yaml:"proxyImage" mapstructure:"proxyImage"`
}

var (
	v        *viper.Viper
	instance *Config
	initOnce sync.Once
	mu       sync.RWMutex
)

// ResetConfig resets the configuration instance (for testing)
func ResetConfig() {
	mu.Lock()
	defer mu.Unlock()
	v = nil
	instance = nil
	initOnce = sync.Once{}
}

// InitConfig initializes the configuration with Viper
func InitConfig() error {
	var initErr error
	initOnce.Do(func() {
		mu.Lock()
		defer mu.Unlock()

		v = viper.New()

		// Set default values
		homeDir, _ := os.UserHomeDir()
		defaultDataDir := filepath.Join(homeDir, ".kecs", "data")

		// Server defaults
		v.SetDefault("server.port", 8080)
		v.SetDefault("server.adminPort", 8081)
		v.SetDefault("server.dataDir", defaultDataDir)
		v.SetDefault("server.logLevel", "info")
		v.SetDefault("server.allowedOrigins", []string{})
		v.SetDefault("server.endpoint", "")
		v.SetDefault("server.controlPlaneImage", "ghcr.io/nandemo-ya/kecs:latest")

		// Kubernetes defaults
		v.SetDefault("kubernetes.kubeconfigPath", "")
		v.SetDefault("kubernetes.k3dOptimized", false)
		v.SetDefault("kubernetes.k3dAsync", false)
		v.SetDefault("kubernetes.disableCoreDNS", false)
		v.SetDefault("kubernetes.keepClustersOnShutdown", false)

		// Features defaults
		v.SetDefault("features.testMode", false)
		v.SetDefault("features.containerMode", false)
		v.SetDefault("features.autoRecoverState", true)
		v.SetDefault("features.traefik", true)         // Enable Traefik by default
		v.SetDefault("features.iamIntegration", false) // Disable IAM integration by default
		v.SetDefault("features.tuiMock", true)         // Enable TUI mock mode by default

		// AWS defaults
		v.SetDefault("aws.defaultRegion", "us-east-1")
		v.SetDefault("aws.accountID", "000000000000")
		v.SetDefault("aws.proxyImage", "")

		// LocalStack defaults
		v.SetDefault("localstack.enabled", true)    // Enable LocalStack by default
		v.SetDefault("localstack.useTraefik", true) // Enable Traefik for LocalStack by default
		v.SetDefault("localstack.image", "localstack/localstack")
		v.SetDefault("localstack.version", "latest")

		// Enable environment variable support
		v.SetEnvPrefix("KECS")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		// Map legacy environment variables to new structure
		bindLegacyEnvVars()
	})

	return initErr
}

// bindLegacyEnvVars maps legacy environment variables to the new structure
func bindLegacyEnvVars() {
	// Direct mappings
	v.BindEnv("features.testMode", "KECS_TEST_MODE")
	v.BindEnv("features.containerMode", "KECS_CONTAINER_MODE")
	v.BindEnv("server.dataDir", "KECS_DATA_DIR")
	v.BindEnv("aws.defaultRegion", "KECS_DEFAULT_REGION")
	v.BindEnv("aws.accountID", "KECS_ACCOUNT_ID")
	v.BindEnv("server.allowedOrigins", "KECS_ALLOWED_ORIGINS")
	v.BindEnv("server.endpoint", "KECS_ENDPOINT")
	v.BindEnv("kubernetes.kubeconfigPath", "KECS_KUBECONFIG_PATH")
	v.BindEnv("kubernetes.k3dOptimized", "KECS_K3D_OPTIMIZED")
	v.BindEnv("kubernetes.k3dAsync", "KECS_K3D_ASYNC")
	v.BindEnv("kubernetes.disableCoreDNS", "KECS_DISABLE_COREDNS")
	v.BindEnv("kubernetes.keepClustersOnShutdown", "KECS_KEEP_CLUSTERS_ON_SHUTDOWN")
	v.BindEnv("features.autoRecoverState", "KECS_AUTO_RECOVER_STATE")
	v.BindEnv("aws.proxyImage", "KECS_AWS_PROXY_IMAGE")
	v.BindEnv("features.traefik", "KECS_ENABLE_TRAEFIK", "KECS_FEATURES_TRAEFIK")
	v.BindEnv("features.devMode", "KECS_DEV_MODE")
	v.BindEnv("features.iamIntegration", "KECS_IAM_INTEGRATION")
	v.BindEnv("localstack.enabled", "KECS_LOCALSTACK_ENABLED")
	v.BindEnv("localstack.useTraefik", "KECS_LOCALSTACK_USE_TRAEFIK")
	v.BindEnv("server.controlPlaneImage", "KECS_CONTROLPLANE_IMAGE")
	v.BindEnv("features.tuiMock", "KECS_TUI_MOCK")
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	if instance == nil {
		if err := InitConfig(); err != nil {
			// Fallback to basic defaults if initialization fails
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

		cfg := &Config{
			LocalStack: *localstack.DefaultConfig(),
		}
		if err := v.Unmarshal(cfg); err != nil {
			// Return minimal config on error
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

		instance = cfg
	}

	return instance
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	if err := InitConfig(); err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	if configPath != "" {
		// Check if the config file exists before trying to read it
		if _, err := os.Stat(configPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("config file does not exist: %s", configPath)
			}
			return nil, fmt.Errorf("failed to access config file: %w", err)
		}

		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		// Look for config file in standard locations
		v.SetConfigName("kecs")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.kecs")
		v.AddConfigPath("/etc/kecs")

		// Read config file if it exists
		if err := v.ReadInConfig(); err != nil {
			// It's ok if config file doesn't exist
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	config := &Config{
		LocalStack: *localstack.DefaultConfig(),
	}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	instance = config
	return config, nil
}

// GetConfig returns the current configuration instance
func GetConfig() *Config {
	if instance == nil {
		return DefaultConfig()
	}
	return instance
}

// Get returns a configuration value by key
func Get(key string) interface{} {
	if v == nil {
		InitConfig()
	}
	return v.Get(key)
}

// GetString returns a string configuration value
func GetString(key string) string {
	ensureInitialized()
	mu.RLock()
	defer mu.RUnlock()
	return v.GetString(key)
}

// GetInt returns an int configuration value
func GetInt(key string) int {
	ensureInitialized()
	mu.RLock()
	defer mu.RUnlock()
	return v.GetInt(key)
}

// GetBool returns a bool configuration value
func GetBool(key string) bool {
	ensureInitialized()
	mu.RLock()
	defer mu.RUnlock()
	return v.GetBool(key)
}

// GetStringSlice returns a string slice configuration value
func GetStringSlice(key string) []string {
	ensureInitialized()
	mu.RLock()
	defer mu.RUnlock()
	return v.GetStringSlice(key)
}

// Set sets a configuration value
func Set(key string, value interface{}) {
	ensureInitialized()
	mu.Lock()
	defer mu.Unlock()
	v.Set(key, value)
}

// ensureInitialized ensures the configuration is initialized
func ensureInitialized() {
	mu.RLock()
	initialized := v != nil
	mu.RUnlock()

	if !initialized {
		InitConfig()
	}
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
	if c.Server.DataDir == "" {
		return fmt.Errorf("data directory cannot be empty")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Server.LogLevel] {
		return fmt.Errorf("invalid log level: %s", c.Server.LogLevel)
	}

	// Validate AWS region if specified
	if c.AWS.DefaultRegion != "" {
		// Basic region format validation
		if len(c.AWS.DefaultRegion) < 3 {
			return fmt.Errorf("invalid AWS region format: %s", c.AWS.DefaultRegion)
		}
	}

	// Validate AWS account ID if specified
	if c.AWS.AccountID != "" && len(c.AWS.AccountID) != 12 {
		return fmt.Errorf("AWS account ID must be 12 digits: %s", c.AWS.AccountID)
	}

	// Validate LocalStack config if enabled
	if c.LocalStack.Enabled {
		if err := c.LocalStack.Validate(); err != nil {
			return fmt.Errorf("invalid LocalStack config: %w", err)
		}
	}

	return nil
}
