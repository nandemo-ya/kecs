package runtime

import (
	"fmt"
	"os"
	"sync"
)

var (
	defaultManager *Manager
	once           sync.Once
)

// Manager manages container runtime instances
type Manager struct {
	runtime     Runtime
	runtimeType string
	mu          sync.RWMutex
}

// NewManager creates a new runtime manager
func NewManager() *Manager {
	return &Manager{}
}

// GetDefaultManager returns the default runtime manager
func GetDefaultManager() *Manager {
	once.Do(func() {
		defaultManager = NewManager()
		// Auto-detect runtime on first access
		defaultManager.AutoDetect()
	})
	return defaultManager
}

// GetRuntime returns the current runtime
func (m *Manager) GetRuntime() (Runtime, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.runtime == nil {
		return nil, fmt.Errorf("no container runtime available")
	}

	return m.runtime, nil
}

// GetRuntimeType returns the current runtime type
func (m *Manager) GetRuntimeType() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.runtimeType
}

// SetRuntime sets a specific runtime
func (m *Manager) SetRuntime(runtimeType string, runtime Runtime) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Close previous runtime if exists
	if m.runtime != nil {
		if closer, ok := m.runtime.(interface{ Close() error }); ok {
			closer.Close()
		}
	}

	m.runtime = runtime
	m.runtimeType = runtimeType

	return nil
}

// AutoDetect automatically detects and sets the available runtime
func (m *Manager) AutoDetect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Try Docker first
	if dockerRuntime, err := NewDockerRuntime(); err == nil && dockerRuntime.IsAvailable() {
		m.runtime = dockerRuntime
		m.runtimeType = "docker"
		return nil
	}

	// Try containerd
	if containerdRuntime, err := NewContainerdRuntime(""); err == nil && containerdRuntime.IsAvailable() {
		m.runtime = containerdRuntime
		m.runtimeType = "containerd"
		return nil
	}

	return fmt.Errorf("no container runtime found")
}

// UseDocker forces the use of Docker runtime
func (m *Manager) UseDocker() error {
	runtime, err := NewDockerRuntime()
	if err != nil {
		return fmt.Errorf("failed to create Docker runtime: %w", err)
	}

	if !runtime.IsAvailable() {
		return fmt.Errorf("Docker is not available")
	}

	return m.SetRuntime("docker", runtime)
}

// UseContainerd forces the use of containerd runtime
func (m *Manager) UseContainerd(socketPath string) error {
	runtime, err := NewContainerdRuntime(socketPath)
	if err != nil {
		return fmt.Errorf("failed to create containerd runtime: %w", err)
	}

	if !runtime.IsAvailable() {
		return fmt.Errorf("containerd is not available")
	}

	return m.SetRuntime("containerd", runtime)
}

// GetAvailableRuntimes returns a list of available runtimes
func GetAvailableRuntimes() []string {
	var runtimes []string

	// Check Docker
	if dockerRuntime, err := NewDockerRuntime(); err == nil && dockerRuntime.IsAvailable() {
		runtimes = append(runtimes, "docker")
		dockerRuntime.Close()
	}

	// Check containerd
	if containerdRuntime, err := NewContainerdRuntime(""); err == nil && containerdRuntime.IsAvailable() {
		runtimes = append(runtimes, "containerd")
		containerdRuntime.Close()
	}

	return runtimes
}

// GetPreferredRuntime returns the preferred runtime based on environment
func GetPreferredRuntime() string {
	// Check environment variable
	if runtime := os.Getenv("KECS_CONTAINER_RUNTIME"); runtime != "" {
		return runtime
	}

	// Check if running in k3s/k3d (prefer containerd)
	if _, err := os.Stat("/run/k3s/containerd/containerd.sock"); err == nil {
		return "containerd"
	}

	// Check if running in Kind (prefer containerd)
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return "containerd"
	}

	// Default to Docker for backward compatibility
	return "docker"
}
