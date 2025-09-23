package webhook

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// CertificateManager manages webhook certificates
type CertificateManager struct {
	clientset kubernetes.Interface
	namespace string
	service   string
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager(clientset kubernetes.Interface, namespace, service string) *CertificateManager {
	return &CertificateManager{
		clientset: clientset,
		namespace: namespace,
		service:   service,
	}
}

// EnsureCertificates ensures webhook certificates exist
func (cm *CertificateManager) EnsureCertificates(ctx context.Context) ([]byte, error) {
	secretName := "kecs-webhook-certs"

	// Check if secret already exists
	secret, err := cm.clientset.CoreV1().Secrets(cm.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil && secret != nil {
		// Secret exists, return CA certificate
		if caCert, ok := secret.Data["ca.crt"]; ok {
			logging.Info("Using existing webhook certificates")
			return caCert, nil
		}
	}

	// Generate new certificates
	logging.Info("Generating new webhook certificates")
	caCert, serverCert, serverKey, err := cm.generateCertificates()
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Create or update secret
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: cm.namespace,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"ca.crt":  caCert,
			"tls.crt": serverCert,
			"tls.key": serverKey,
		},
	}

	if _, err := cm.clientset.CoreV1().Secrets(cm.namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		// Try to update if creation fails
		if _, err := cm.clientset.CoreV1().Secrets(cm.namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
			return nil, fmt.Errorf("failed to create/update certificate secret: %w", err)
		}
	}

	return caCert, nil
}

// generateCertificates generates self-signed certificates for the webhook
func (cm *CertificateManager) generateCertificates() (caCert, serverCert, serverKey []byte, err error) {
	// Generate CA private key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate CA private key: %w", err)
	}

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"KECS"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Encode CA certificate
	caCert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertDER,
	})

	// Generate server private key
	serverPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate server private key: %w", err)
	}

	// Create server certificate template
	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"KECS"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		DNSNames: []string{
			cm.service,
			fmt.Sprintf("%s.%s", cm.service, cm.namespace),
			fmt.Sprintf("%s.%s.svc", cm.service, cm.namespace),
			fmt.Sprintf("%s.%s.svc.cluster.local", cm.service, cm.namespace),
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Create server certificate signed by CA
	serverCertDER, err := x509.CreateCertificate(rand.Reader, &serverTemplate, &caTemplate, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Encode server certificate
	serverCert = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertDER,
	})

	// Encode server private key
	serverKey = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})

	return caCert, serverCert, serverKey, nil
}
