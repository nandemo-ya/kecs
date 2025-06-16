package localstack

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// localStackManager implements the Manager interface
type localStackManager struct {
	config       *Config
	proxyConfig  *ProxyConfig
	kubeClient   kubernetes.Interface
	kubeManager  KubernetesManager
	healthChecker HealthChecker
	container    *LocalStackContainer
	status       *Status
	mu           sync.RWMutex
	stopCh       chan struct{}
	healthStop   chan struct{}
}

// NewManager creates a new LocalStack manager instance
func NewManager(config *Config, kubeClient kubernetes.Interface) (Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	kubeManager := NewKubernetesManager(kubeClient, config.Namespace)
	healthChecker := NewHealthChecker(fmt.Sprintf("http://localstack.%s.svc.cluster.local:%d", config.Namespace, config.Port))

	return &localStackManager{
		config:        config,
		proxyConfig:   ProxyConfigWithDefaults(fmt.Sprintf("http://localstack.%s.svc.cluster.local:%d", config.Namespace, config.Port)),
		kubeClient:    kubeClient,
		kubeManager:   kubeManager,
		healthChecker: healthChecker,
		status: &Status{
			Running:         false,
			Healthy:         false,
			EnabledServices: config.Services,
			ServiceStatus:   make(map[string]ServiceInfo),
		},
		stopCh:     make(chan struct{}),
		healthStop: make(chan struct{}),
	}, nil
}

// Start starts LocalStack and begins monitoring its health
func (m *localStackManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.status.Running {
		return fmt.Errorf("LocalStack is already running")
	}

	klog.Info("Starting LocalStack...")

	// Create namespace if it doesn't exist
	if err := m.kubeManager.CreateNamespace(ctx); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Deploy LocalStack
	if err := m.kubeManager.DeployLocalStack(ctx, m.config); err != nil {
		return fmt.Errorf("failed to deploy LocalStack: %w", err)
	}

	// Wait for pod to be ready
	if err := m.waitForPodReady(ctx); err != nil {
		return fmt.Errorf("failed to wait for pod ready: %w", err)
	}

	// Get pod information
	podName, err := m.kubeManager.GetLocalStackPod()
	if err != nil {
		return fmt.Errorf("failed to get LocalStack pod: %w", err)
	}

	// Get service endpoint
	endpoint, err := m.kubeManager.GetServiceEndpoint()
	if err != nil {
		return fmt.Errorf("failed to get service endpoint: %w", err)
	}

	m.container = &LocalStackContainer{
		PodName:    podName,
		Namespace:  m.config.Namespace,
		Endpoint:   endpoint,
		StartedAt:  time.Now(),
		KubeClient: m.kubeClient,
	}

	// Update health checker endpoint
	m.healthChecker = NewHealthChecker(endpoint)

	// Wait for LocalStack to output "Ready." in logs
	klog.Info("Waiting for LocalStack to be ready...")
	readyCtx, readyCancel := context.WithTimeout(ctx, DefaultHealthTimeout)
	defer readyCancel()
	
	if kubeManager, ok := m.kubeManager.(*kubernetesManager); ok {
		if err := kubeManager.WaitForLocalStackReady(readyCtx, DefaultHealthTimeout); err != nil {
			klog.Warningf("Failed to detect Ready message: %v, falling back to health check", err)
		}
	}
	
	// Also wait for health check
	if err := m.healthChecker.WaitForHealthy(ctx, DefaultHealthTimeout); err != nil {
		return fmt.Errorf("LocalStack failed to become healthy: %w", err)
	}

	// Update status
	m.status.Running = true
	m.status.Healthy = true
	m.status.Endpoint = endpoint

	// Start health monitoring
	go m.monitorHealth()

	klog.Infof("LocalStack started successfully at %s", endpoint)
	return nil
}

// Stop stops LocalStack and cleans up resources
func (m *localStackManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.status.Running {
		return fmt.Errorf("LocalStack is not running")
	}

	klog.Info("Stopping LocalStack...")

	// Stop health monitoring
	close(m.healthStop)

	// Delete LocalStack deployment
	if err := m.kubeManager.DeleteLocalStack(ctx); err != nil {
		return fmt.Errorf("failed to delete LocalStack: %w", err)
	}

	// Update status
	m.status.Running = false
	m.status.Healthy = false
	m.status.Endpoint = ""
	m.container = nil

	klog.Info("LocalStack stopped successfully")
	return nil
}

// Restart restarts LocalStack
func (m *localStackManager) Restart(ctx context.Context) error {
	klog.Info("Restarting LocalStack...")

	if err := m.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop LocalStack: %w", err)
	}

	// Wait a bit for resources to be cleaned up
	time.Sleep(5 * time.Second)

	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("failed to start LocalStack: %w", err)
	}

	return nil
}

// GetStatus returns the current status of LocalStack
func (m *localStackManager) GetStatus() (*Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy of the status
	status := &Status{
		Running:         m.status.Running,
		Healthy:         m.status.Healthy,
		Endpoint:        m.status.Endpoint,
		EnabledServices: make([]string, len(m.status.EnabledServices)),
		ServiceStatus:   make(map[string]ServiceInfo),
		LastHealthCheck: m.status.LastHealthCheck,
		Version:         m.status.Version,
	}

	copy(status.EnabledServices, m.status.EnabledServices)

	for k, v := range m.status.ServiceStatus {
		status.ServiceStatus[k] = v
	}

	if m.container != nil && m.status.Running {
		status.Uptime = time.Since(m.container.StartedAt)
	}

	return status, nil
}

// UpdateServices updates the list of enabled services
func (m *localStackManager) UpdateServices(services []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate services
	for _, service := range services {
		if !IsValidService(service) {
			return fmt.Errorf("invalid service: %s", service)
		}
	}

	// Update configuration
	m.config.Services = services

	// If LocalStack is running, update the deployment
	if m.status.Running {
		if err := m.kubeManager.UpdateDeployment(context.Background(), m.config); err != nil {
			return fmt.Errorf("failed to update deployment: %w", err)
		}
	}

	// Update status
	m.status.EnabledServices = services

	klog.Infof("Updated enabled services: %v", services)
	return nil
}

// GetEnabledServices returns the list of enabled services
func (m *localStackManager) GetEnabledServices() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services := make([]string, len(m.status.EnabledServices))
	copy(services, m.status.EnabledServices)

	return services, nil
}

// GetEndpoint returns the LocalStack endpoint
func (m *localStackManager) GetEndpoint() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.status.Running {
		return "", fmt.Errorf("LocalStack is not running")
	}

	return m.status.Endpoint, nil
}

// GetServiceEndpoint returns the endpoint for a specific service
func (m *localStackManager) GetServiceEndpoint(service string) (string, error) {
	endpoint, err := m.GetEndpoint()
	if err != nil {
		return "", err
	}

	return GetServiceURL(endpoint, service), nil
}

// IsHealthy returns true if LocalStack is healthy
func (m *localStackManager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.status.Healthy
}

// WaitForReady waits for LocalStack to become ready
func (m *localStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	if !m.status.Running {
		return fmt.Errorf("LocalStack is not running")
	}

	return m.healthChecker.WaitForHealthy(ctx, timeout)
}

// waitForPodReady waits for the LocalStack pod to be ready
func (m *localStackManager) waitForPodReady(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for pod to be ready")
		case <-ticker.C:
			podName, err := m.kubeManager.GetLocalStackPod()
			if err == nil && podName != "" {
				klog.Infof("LocalStack pod %s is ready", podName)
				return nil
			}
		}
	}
}

// monitorHealth continuously monitors LocalStack health
func (m *localStackManager) monitorHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.healthStop:
			return
		case <-ticker.C:
			m.checkHealth()
		}
	}
}

// checkHealth performs a health check and updates status
func (m *localStackManager) checkHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	healthStatus, err := m.healthChecker.CheckHealth(ctx)

	m.mu.Lock()
	defer m.mu.Unlock()

	m.status.LastHealthCheck = time.Now()

	if err != nil {
		klog.Errorf("Health check failed: %v", err)
		m.status.Healthy = false
		return
	}

	m.status.Healthy = healthStatus.Healthy

	// Update service status
	for _, sh := range healthStatus.ServiceHealth {
		m.status.ServiceStatus[sh.Service] = ServiceInfo{
			Name:    sh.Service,
			Enabled: true,
			Healthy: sh.Healthy,
			Endpoint: GetServiceURL(m.status.Endpoint, sh.Service),
		}
	}

	if !healthStatus.Healthy {
		klog.Warningf("LocalStack is unhealthy: %s", healthStatus.Message)
	}
}