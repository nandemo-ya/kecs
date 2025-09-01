package webhook

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

// GenerateSelfSignedCert generates a self-signed certificate for the webhook
func GenerateSelfSignedCert(certPath, keyPath, hostname string) error {
	// Generate RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"KECS"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Add DNS names and IP addresses
	template.DNSNames = []string{
		hostname,
		"kecs-webhook",
		"kecs-webhook.kecs-system",
		"kecs-webhook.kecs-system.svc",
		"kecs-webhook.kecs-system.svc.cluster.local",
		"kecs-controlplane",
		"kecs-controlplane.kecs-system",
		"kecs-controlplane.kecs-system.svc",
		"kecs-controlplane.kecs-system.svc.cluster.local",
		"localhost",
	}
	template.IPAddresses = []net.IP{
		net.IPv4(127, 0, 0, 1),
		net.IPv6loopback,
	}

	// Generate certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	// Write certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return err
	}

	// Write private key
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privKeyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privKeyDER}); err != nil {
		return err
	}

	return nil
}
