package runtime

import (
	"context"
	"io"
	"time"
)

// Runtime represents a container runtime interface
type Runtime interface {
	// Container lifecycle operations
	CreateContainer(ctx context.Context, config *ContainerConfig) (*Container, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeout *int) error
	RemoveContainer(ctx context.Context, id string, force bool) error

	// Container information
	GetContainer(ctx context.Context, id string) (*Container, error)
	ListContainers(ctx context.Context, opts ListContainersOptions) ([]*Container, error)
	ContainerLogs(ctx context.Context, id string, opts LogsOptions) (io.ReadCloser, error)

	// Image operations
	PullImage(ctx context.Context, image string, opts PullImageOptions) (io.ReadCloser, error)

	// Runtime information
	Name() string
	IsAvailable() bool
}

// ContainerConfig represents container configuration
type ContainerConfig struct {
	Name   string
	Image  string
	Env    []string
	Cmd    []string
	Labels map[string]string

	// Port bindings
	Ports []PortBinding

	// Volume mounts
	Mounts []Mount

	// Resource limits
	Resources *Resources

	// Restart policy
	RestartPolicy RestartPolicy

	// Network configuration
	NetworkMode string
	Networks    []string

	// Health check
	HealthCheck *HealthCheck

	// User and group settings
	User     string   // User or UID
	GroupAdd []string // Additional groups
}

// Container represents a container
type Container struct {
	ID       string
	Name     string
	Image    string
	State    string
	Status   string
	Created  time.Time
	Labels   map[string]string
	Ports    []PortBinding
	Networks []string
}

// PortBinding represents a port binding
type PortBinding struct {
	ContainerPort uint16
	HostPort      uint16
	Protocol      string // tcp or udp
	HostIP        string
}

// Mount represents a volume mount
type Mount struct {
	Type     string // bind, volume, tmpfs
	Source   string
	Target   string
	ReadOnly bool
}

// Resources represents resource constraints
type Resources struct {
	CPUShares  int64
	Memory     int64
	MemorySwap int64
	CPUQuota   int64
	CPUPeriod  int64
}

// RestartPolicy represents container restart policy
type RestartPolicy struct {
	Name              string // no, on-failure, always, unless-stopped
	MaximumRetryCount int
}

// HealthCheck represents container health check configuration
type HealthCheck struct {
	Test        []string
	Interval    time.Duration
	Timeout     time.Duration
	StartPeriod time.Duration
	Retries     int
}

// ListContainersOptions represents options for listing containers
type ListContainersOptions struct {
	All    bool
	Labels map[string]string
	Names  []string
}

// LogsOptions represents options for getting container logs
type LogsOptions struct {
	Follow     bool
	Stdout     bool
	Stderr     bool
	Since      string
	Until      string
	Timestamps bool
	Tail       string
}

// PullImageOptions represents options for pulling images
type PullImageOptions struct {
	Platform string
	Auth     *AuthConfig
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Username      string
	Password      string
	Auth          string
	ServerAddress string
}
