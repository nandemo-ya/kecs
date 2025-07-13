package localstack

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// localStackManager implements the Manager interface
type localStackManager struct {
	config        *Config
	proxyConfig   *ProxyConfig
	kubeClient    kubernetes.Interface
	kubeManager   KubernetesManager
	healthChecker HealthChecker
	container     *LocalStackContainer
	status        *Status
	mu            sync.RWMutex
	stopCh        chan struct{}
	healthStop    chan struct{}
}

// NewManager creates a new LocalStack manager instance
func NewManager(config *Config, kubeClient kubernetes.Interface, kubeConfig *rest.Config) (Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	kubeManager := NewKubernetesManager(kubeClient, kubeConfig, config.Namespace)
	
	// Determine health check endpoint based on runtime configuration
	var healthEndpoint string
	if config.ContainerMode {
		// In container mode, we can use cluster-internal DNS since KECS will be in the same network
		healthEndpoint = fmt.Sprintf("http://localstack.%s.svc.cluster.local:%d", config.Namespace, config.Port)
		klog.Infof("Container mode: using cluster-internal endpoint for health checker: %s", healthEndpoint)
	} else if config.UseTraefik && config.ProxyEndpoint != "" {
		// In host mode with Traefik, use the proxy endpoint
		healthEndpoint = config.ProxyEndpoint
		klog.Infof("Host mode with Traefik: using proxy endpoint for health checker: %s", healthEndpoint)
	} else {
		// Fallback to NodePort or other external access method
		healthEndpoint = fmt.Sprintf("http://localhost:%d", config.Port)
		klog.Infof("Host mode: using localhost endpoint for health checker: %s", healthEndpoint)
	}
	healthChecker := NewHealthChecker(healthEndpoint)

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

	// Update health checker endpoint based on runtime configuration
	var healthEndpoint string
	if m.config.ContainerMode {
		// In container mode, use cluster-internal endpoint
		healthEndpoint = endpoint
		klog.Infof("Container mode: using cluster-internal endpoint for runtime health checks: %s", healthEndpoint)
	} else if m.config.UseTraefik && m.config.ProxyEndpoint != "" {
		// In host mode with Traefik, use the proxy endpoint
		healthEndpoint = m.config.ProxyEndpoint
		klog.Infof("Host mode with Traefik: using proxy endpoint for runtime health checks: %s", healthEndpoint)
	} else {
		// Fallback - this would need NodePort or other external access
		healthEndpoint = fmt.Sprintf("http://localhost:%d", m.config.Port)
		klog.Warningf("Host mode without Traefik: health checks may fail. Using: %s", healthEndpoint)
	}
	
	// Update the health checker with the correct endpoint
	m.healthChecker.UpdateEndpoint(healthEndpoint)

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
		// Log the error but don't fail - DNS resolution issues might be temporary
		klog.Warningf("LocalStack health check failed (DNS resolution issue?): %v", err)
		klog.Info("Continuing despite health check failure - LocalStack pod is running")
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

	// Return appropriate endpoint based on runtime configuration
	if m.config.ContainerMode {
		// In container mode, return the cluster-internal endpoint
		return m.status.Endpoint, nil
	} else if m.config.UseTraefik && m.config.ProxyEndpoint != "" {
		// In host mode with Traefik, return the proxy endpoint
		return m.config.ProxyEndpoint, nil
	}

	// Fallback to status endpoint (might be NodePort or other external access)
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

// IsRunning returns whether LocalStack is currently running
func (m *localStackManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status.Running
}

// GetConfig returns the current LocalStack configuration
func (m *localStackManager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// CheckServiceHealth checks if a specific service is healthy
func (m *localStackManager) CheckServiceHealth(service string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.status.Running {
		return fmt.Errorf("LocalStack is not running")
	}

	// Check if service is enabled
	serviceEnabled := false
	for _, s := range m.config.Services {
		if s == service {
			serviceEnabled = true
			break
		}
	}

	if !serviceEnabled {
		return fmt.Errorf("service %s is not enabled", service)
	}

	// Perform health check
	if m.healthChecker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		health, err := m.healthChecker.CheckHealth(ctx)
		if err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}

		if !health.Healthy {
			return fmt.Errorf("LocalStack is not healthy: %s", health.Message)
		}

		// Check specific service health if available
		if serviceHealth, ok := health.ServiceHealth[service]; ok {
			if !serviceHealth.Healthy {
				return fmt.Errorf("service %s is not healthy: %s", service, serviceHealth.Error)
			}
		}
	}

	return nil
}

// WaitForReady waits for LocalStack to become ready
func (m *localStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	if !m.status.Running {
		return fmt.Errorf("LocalStack is not running")
	}

	// Try health check but don't fail on DNS issues
	if err := m.healthChecker.WaitForHealthy(ctx, timeout); err != nil {
		// Check if it's a DNS resolution error
		if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "dial tcp: lookup") {
			klog.Warningf("LocalStack health check failed due to DNS resolution: %v", err)
			klog.Info("LocalStack pod is running, continuing despite DNS issue")
			return nil
		}
		return err
	}
	
	return nil
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
			Name:     sh.Service,
			Enabled:  true,
			Healthy:  sh.Healthy,
			Endpoint: GetServiceURL(m.status.Endpoint, sh.Service),
		}
	}

	if !healthStatus.Healthy {
		klog.Warningf("LocalStack is unhealthy: %s", healthStatus.Message)
	}
}
