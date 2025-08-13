package proxy

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"k8s.io/client-go/kubernetes"
)

// Manager manages AWS proxy modes for LocalStack integration
type Manager struct {
	mode               localstack.ProxyMode
	config             *localstack.ProxyConfig
	localStackEndpoint string
	kubeClient         kubernetes.Interface
	envProxy           *EnvironmentVariableProxy
	sidecarProxy       *SidecarProxy
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
	logging.Info("Starting proxy manager", "mode", m.mode)

	switch m.mode {
	case localstack.ProxyModeEnvironment:
		return m.startEnvironmentMode(ctx)
	case localstack.ProxyModeSidecar:
		return m.startSidecarMode(ctx)
	case localstack.ProxyModeDisabled:
		logging.Info("Proxy mode is disabled")
		return nil
	default:
		return fmt.Errorf("unsupported proxy mode: %s", m.mode)
	}
}

// Stop stops the proxy manager
func (m *Manager) Stop(ctx context.Context) error {
	logging.Info("Stopping proxy manager")

	if m.webhookServer != nil {
		if err := m.webhookServer.Stop(); err != nil {
			return fmt.Errorf("failed to stop webhook server: %w", err)
		}
	}

	return nil
}

// startEnvironmentMode starts the environment variable injection mode
func (m *Manager) startEnvironmentMode(ctx context.Context) error {
	logging.Info("Starting environment variable proxy mode")

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
	logging.Info("Starting sidecar proxy mode")

	// Create sidecar proxy
	sidecarProxy := NewSidecarProxy(m.localStackEndpoint)

	// Set custom proxy image if configured
	if proxyImage := config.GetString("aws.proxyImage"); proxyImage != "" {
		sidecarProxy.SetProxyImage(proxyImage)
	}

	// Store reference for later use
	m.sidecarProxy = sidecarProxy

	logging.Info("Sidecar proxy mode initialized successfully")
	return nil
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

	if m.sidecarProxy != nil {
		m.sidecarProxy.UpdateEndpoint(endpoint)
	}
}

// GetSidecarProxy returns the sidecar proxy if available
func (m *Manager) GetSidecarProxy() *SidecarProxy {
	return m.sidecarProxy
}
