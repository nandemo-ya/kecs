package proxy

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Manager manages AWS proxy modes for LocalStack integration
type Manager struct {
	mode               localstack.ProxyMode
	config             *localstack.ProxyConfig
	localStackEndpoint string
	kubeClient         kubernetes.Interface
	envProxy           *EnvironmentVariableProxy
	webhookServer      *WebhookServer
}

// NewManager creates a new proxy manager
func NewManager(kubeClient kubernetes.Interface, config *localstack.ProxyConfig) (*Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid proxy config: %w", err)
	}

	return &Manager{
		mode:               config.Mode,
		config:             config,
		localStackEndpoint: config.LocalStackEndpoint,
		kubeClient:         kubeClient,
	}, nil
}

// Start starts the proxy manager with the configured mode
func (m *Manager) Start(ctx context.Context) error {
	klog.Infof("Starting proxy manager with mode: %s", m.mode)

	switch m.mode {
	case localstack.ProxyModeEnvironment:
		return m.startEnvironmentMode(ctx)
	case localstack.ProxyModeSidecar:
		return m.startSidecarMode(ctx)
	case localstack.ProxyModeDisabled:
		klog.Info("Proxy mode is disabled")
		return nil
	default:
		return fmt.Errorf("unsupported proxy mode: %s", m.mode)
	}
}

// Stop stops the proxy manager
func (m *Manager) Stop(ctx context.Context) error {
	klog.Info("Stopping proxy manager")

	if m.webhookServer != nil {
		if err := m.webhookServer.Stop(); err != nil {
			return fmt.Errorf("failed to stop webhook server: %w", err)
		}
	}

	return nil
}

// startEnvironmentMode starts the environment variable injection mode
func (m *Manager) startEnvironmentMode(ctx context.Context) error {
	klog.Info("Starting environment variable proxy mode")

	// Create environment variable proxy
	m.envProxy = NewEnvironmentVariableProxy(m.localStackEndpoint)

	// Create and start webhook server
	webhookConfig := &WebhookConfig{
		Port:        9443,
		CertDir:     "/tmp/k8s-webhook-server/serving-certs",
		ServiceName: "kecs-webhook",
		Namespace:   "kecs-system",
	}

	var err error
	m.webhookServer, err = NewWebhookServer(m.kubeClient, webhookConfig, m.envProxy)
	if err != nil {
		return fmt.Errorf("failed to create webhook server: %w", err)
	}

	return m.webhookServer.Start(ctx)
}

// startSidecarMode starts the sidecar proxy mode
func (m *Manager) startSidecarMode(ctx context.Context) error {
	klog.Info("Starting sidecar proxy mode")
	// TODO: Implement sidecar mode
	return fmt.Errorf("sidecar mode not yet implemented")
}

// GetMode returns the current proxy mode
func (m *Manager) GetMode() localstack.ProxyMode {
	return m.mode
}

// UpdateEndpoint updates the LocalStack endpoint
func (m *Manager) UpdateEndpoint(endpoint string) {
	m.localStackEndpoint = endpoint
	
	if m.envProxy != nil {
		m.envProxy.UpdateEndpoint(endpoint)
	}
}