package admission

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// WebhookIntegration manages the admission webhook lifecycle
type WebhookIntegration struct {
	client           kubernetes.Interface
	namespace        string
	port             int
	certManager      *CertificateManager
	webhookServer    *WebhookServer
	localStackMgr    localstack.Manager
}

// NewWebhookIntegration creates a new webhook integration
func NewWebhookIntegration(client kubernetes.Interface, namespace string, port int, proxyImage string, localStackMgr localstack.Manager) *WebhookIntegration {
	return &WebhookIntegration{
		client:        client,
		namespace:     namespace,
		port:          port,
		certManager:   NewCertificateManager(client, namespace),
		localStackMgr: localStackMgr,
	}
}

// Start starts the admission webhook
func (wi *WebhookIntegration) Start(ctx context.Context) error {
	klog.Info("Starting admission webhook integration")

	// Get or create TLS certificates
	tlsConfig, caBundle, err := wi.certManager.GetOrCreateCertificate(ctx)
	if err != nil {
		return fmt.Errorf("failed to get certificate: %w", err)
	}

	// Create sidecar injector
	sidecarInjector := NewSidecarInjector("kecs/aws-sdk-proxy:latest", wi.localStackMgr)

	// Create and start webhook server
	wi.webhookServer = NewWebhookServer(wi.port, sidecarInjector, tlsConfig)
	
	// Start webhook server in background
	go func() {
		if err := wi.webhookServer.Start(ctx); err != nil {
			klog.Errorf("Webhook server error: %v", err)
		}
	}()

	// Create webhook configuration
	if err := wi.certManager.CreateWebhookConfiguration(ctx, caBundle); err != nil {
		return fmt.Errorf("failed to create webhook configuration: %w", err)
	}

	klog.Info("Admission webhook integration started successfully")
	return nil
}

// Stop stops the admission webhook
func (wi *WebhookIntegration) Stop(ctx context.Context) error {
	klog.Info("Stopping admission webhook integration")

	if wi.webhookServer != nil {
		if err := wi.webhookServer.Stop(ctx); err != nil {
			klog.Errorf("Error stopping webhook server: %v", err)
		}
	}

	// Note: We don't delete the webhook configuration or certificates
	// as they might be needed for graceful shutdown

	return nil
}