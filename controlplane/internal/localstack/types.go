package localstack

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"
)

// Manager represents the LocalStack lifecycle manager
type Manager interface {
	// Lifecycle management
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	GetStatus() (*Status, error)

	// Service management
	UpdateServices(services []string) error
	GetEnabledServices() ([]string, error)

	// Endpoint management
	GetEndpoint() (string, error)
	GetServiceEndpoint(service string) (string, error)

	// Health checks
	IsHealthy() bool
	IsRunning() bool
	WaitForReady(ctx context.Context, timeout time.Duration) error
	CheckServiceHealth(service string) error

	// Configuration
	GetConfig() *Config
}

// Status represents the current status of LocalStack
type Status struct {
	Running         bool                   `json:"running"`
	Healthy         bool                   `json:"healthy"`
	Endpoint        string                 `json:"endpoint"`
	EnabledServices []string               `json:"enabled_services"`
	ServiceStatus   map[string]ServiceInfo `json:"service_status"`
	LastHealthCheck time.Time              `json:"last_health_check"`
	Uptime          time.Duration          `json:"uptime"`
	Version         string                 `json:"version"`
}

// ServiceInfo contains information about a specific AWS service
type ServiceInfo struct {
	Name     string    `json:"name"`
	Enabled  bool      `json:"enabled"`
	Healthy  bool      `json:"healthy"`
	Endpoint string    `json:"endpoint"`
	LastUsed time.Time `json:"last_used,omitempty"`
}

// Config represents LocalStack configuration
type Config struct {
	// Basic configuration
	Enabled     bool     `yaml:"enabled" json:"enabled"`
	Services    []string `yaml:"services" json:"services"`
	Persistence bool     `yaml:"persistence" json:"persistence"`

	// Deployment configuration
	Image     string `yaml:"image" json:"image"`
	Version   string `yaml:"version" json:"version"`
	Namespace string `yaml:"namespace" json:"namespace"`
	Port      int    `yaml:"port" json:"port"`
	EdgePort  int    `yaml:"edge_port" json:"edge_port"`

	// Resource limits
	Resources ResourceLimits `yaml:"resources" json:"resources"`

	// Custom environment variables
	Environment map[string]string `yaml:"environment" json:"environment"`

	// Advanced configuration
	Debug           bool              `yaml:"debug" json:"debug"`
	DataDir         string            `yaml:"data_dir" json:"data_dir"`
	DockerHost      string            `yaml:"docker_host" json:"docker_host"`
	CustomEndpoints map[string]string `yaml:"custom_endpoints" json:"custom_endpoints"`
	
	// Runtime configuration
	UseExternalAccess bool   `yaml:"use_external_access" json:"use_external_access"`
	ProxyEndpoint     string `yaml:"proxy_endpoint" json:"proxy_endpoint"`
	UseTraefik        bool   `yaml:"use_traefik" json:"use_traefik"`
}

// ResourceLimits defines resource constraints for LocalStack
type ResourceLimits struct {
	Memory      string `yaml:"memory" json:"memory"`
	CPU         string `yaml:"cpu" json:"cpu"`
	StorageSize string `yaml:"storage_size" json:"storage_size"`
}

// HealthChecker provides health checking functionality
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*HealthStatus, error)
	WaitForHealthy(ctx context.Context, timeout time.Duration) error
}

// HealthStatus represents the health check result
type HealthStatus struct {
	Healthy          bool                     `json:"healthy"`
	Message          string                   `json:"message"`
	ServiceHealth    map[string]ServiceHealth `json:"service_health"`
	LastCheck        time.Time                `json:"last_check"`
	ConsecutiveFails int                      `json:"consecutive_fails"`
}

// ServiceHealth represents health status of a specific service
type ServiceHealth struct {
	Service string        `json:"service"`
	Healthy bool          `json:"healthy"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

// KubernetesManager handles Kubernetes resource management
type KubernetesManager interface {
	CreateNamespace(ctx context.Context) error
	DeployLocalStack(ctx context.Context, config *Config) error
	DeleteLocalStack(ctx context.Context) error
	GetLocalStackPod() (string, error)
	GetServiceEndpoint() (string, error)
	UpdateDeployment(ctx context.Context, config *Config) error
	GetExternalEndpoint(ctx context.Context) (string, error)
}

// LocalStackContainer represents the LocalStack container state
type LocalStackContainer struct {
	PodName    string
	Namespace  string
	Endpoint   string
	InternalIP string
	StartedAt  time.Time
	KubeClient kubernetes.Interface
}

// ProxyMode represents the AWS proxy mode
type ProxyMode string

const (
	ProxyModeEnvironment ProxyMode = "environment"
	ProxyModeSidecar     ProxyMode = "sidecar"
	ProxyModeEBPF        ProxyMode = "ebpf"
	ProxyModeDisabled    ProxyMode = "disabled"
)

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	Mode               ProxyMode         `yaml:"mode" json:"mode"`
	LocalStackEndpoint string            `yaml:"localstack_endpoint" json:"localstack_endpoint"`
	FallbackEnabled    bool              `yaml:"fallback_enabled" json:"fallback_enabled"`
	FallbackOrder      []ProxyMode       `yaml:"fallback_order" json:"fallback_order"`
	CustomEndpoints    map[string]string `yaml:"custom_endpoints" json:"custom_endpoints"`
}

// Constants for LocalStack
const (
	DefaultNamespace     = "aws-services"
	DefaultImage         = "localstack/localstack"
	DefaultVersion       = "latest"
	DefaultPort          = 4566
	DefaultEdgePort      = 4566
	DefaultHealthTimeout = 2 * time.Minute

	// Labels and annotations
	LabelApp       = "app"
	LabelComponent = "component"
	LabelManagedBy = "app.kubernetes.io/managed-by"

	// LocalStack environment variables
	EnvServices    = "SERVICES"
	EnvDebug       = "DEBUG"
	EnvPersistence = "PERSISTENCE"
	EnvDataDir     = "DATA_DIR"
	EnvDockerHost  = "DOCKER_HOST"
	EnvEdgePort    = "EDGE_PORT"

	// Health check paths
	HealthCheckPath   = "/_localstack/health"
	ServiceHealthPath = "/_localstack/health/services"
)

// Default services to enable
var DefaultServices = []string{
	"iam",
	"logs",
	"ssm",
	"secretsmanager",
	"elbv2",
	"s3",
}

// ServicePortMap maps service names to their default ports
var ServicePortMap = map[string]int{
	"s3":             4566,
	"iam":            4566,
	"ecs":            4566,
	"logs":           4566,
	"ssm":            4566,
	"secretsmanager": 4566,
	"elbv2":          4566,
	"rds":            4566,
	"dynamodb":       4566,
}

// IsValidService checks if a service name is valid
func IsValidService(service string) bool {
	_, exists := ServicePortMap[service]
	return exists
}

// GetServiceURL returns the URL for a specific service
func GetServiceURL(endpoint string, service string) string {
	// LocalStack uses edge service, so all services use the same endpoint
	return endpoint
}
