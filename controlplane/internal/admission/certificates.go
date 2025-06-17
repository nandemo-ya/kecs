package admission

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	WebhookSecretName = "kecs-webhook-certs"
	WebhookServiceName = "kecs-webhook"
	WebhookNamespace = "kecs-system"
)

// CertificateManager manages webhook TLS certificates
type CertificateManager struct {
	client    kubernetes.Interface
	namespace string
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(client kubernetes.Interface, namespace string) *CertificateManager {
	return &CertificateManager{
		client:    client,
		namespace: namespace,
	}
}

// GetOrCreateCertificate gets existing or creates new TLS certificate
func (cm *CertificateManager) GetOrCreateCertificate(ctx context.Context) (*tls.Config, []byte, error) {
	// Try to get existing certificate
	secret, err := cm.client.CoreV1().Secrets(cm.namespace).Get(ctx, WebhookSecretName, metav1.GetOptions{})
	if err == nil {
		klog.Info("Using existing webhook certificate")
		return cm.tlsConfigFromSecret(secret)
	}
	
	klog.Info("Creating new webhook certificate")
	
	// Generate new certificate
	cert, key, caCert, err := cm.generateCertificate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate certificate: %w", err)
	}
	
	// Create secret
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WebhookSecretName,
			Namespace: cm.namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
			"ca.crt":  caCert,
		},
	}
	
	if _, err := cm.client.CoreV1().Secrets(cm.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return nil, nil, fmt.Errorf("failed to create secret: %w", err)
	}
	
	return cm.tlsConfigFromSecret(secret)
}

// generateCertificate generates a self-signed certificate for the webhook
func (cm *CertificateManager) generateCertificate() ([]byte, []byte, []byte, error) {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}
	
	// Create CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"KECS"},
			CommonName:   "KECS Webhook CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	
	// Generate server private key
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}
	
	// Create server certificate
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"KECS"},
			CommonName:   fmt.Sprintf("%s.%s.svc", WebhookServiceName, cm.namespace),
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames: []string{
			WebhookServiceName,
			fmt.Sprintf("%s.%s", WebhookServiceName, cm.namespace),
			fmt.Sprintf("%s.%s.svc", WebhookServiceName, cm.namespace),
			fmt.Sprintf("%s.%s.svc.cluster.local", WebhookServiceName, cm.namespace),
		},
	}
	
	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	
	// Encode certificates and keys to PEM
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	serverCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	serverKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	
	return serverCertPEM, serverKeyPEM, caCertPEM, nil
}

// tlsConfigFromSecret creates TLS config from secret
func (cm *CertificateManager) tlsConfigFromSecret(secret *corev1.Secret) (*tls.Config, []byte, error) {
	cert, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, nil, fmt.Errorf("tls.crt not found in secret")
	}
	
	key, ok := secret.Data["tls.key"]
	if !ok {
		return nil, nil, fmt.Errorf("tls.key not found in secret")
	}
	
	caCert, ok := secret.Data["ca.crt"]
	if !ok {
		return nil, nil, fmt.Errorf("ca.crt not found in secret")
	}
	
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load key pair: %w", err)
	}
	
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{certificate},
	}
	
	return tlsConfig, caCert, nil
}

// CreateWebhookConfiguration creates the mutating webhook configuration
func (cm *CertificateManager) CreateWebhookConfiguration(ctx context.Context, caBundle []byte) error {
	path := "/inject"
	sideEffects := admissionregistrationv1.SideEffectClassNone
	failurePolicy := admissionregistrationv1.Fail
	reinvocationPolicy := admissionregistrationv1.NeverReinvocationPolicy
	
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kecs-sidecar-injector",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				Name: "sidecar-injector.kecs.io",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Name:      WebhookServiceName,
						Namespace: cm.namespace,
						Path:      &path,
					},
					CABundle: caBundle,
				},
				Rules: []admissionregistrationv1.RuleWithOperations{
					{
						Operations: []admissionregistrationv1.OperationType{
							admissionregistrationv1.Create,
						},
						Rule: admissionregistrationv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "kecs.io/localstack-enabled",
							Operator: metav1.LabelSelectorOpIn,
							Values:   []string{"true"},
						},
					},
				},
				SideEffects:             &sideEffects,
				FailurePolicy:           &failurePolicy,
				ReinvocationPolicy:      &reinvocationPolicy,
				AdmissionReviewVersions: []string{"v1", "v1beta1"},
			},
		},
	}
	
	_, err := cm.client.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(ctx, webhookConfig, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create webhook configuration: %w", err)
	}
	
	return nil
}