package elbv2

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// SNIManager manages SNI configuration for HTTPS listeners
type SNIManager struct {
	dynamicClient dynamic.Interface
}

// NewSNIManager creates a new SNI manager
func NewSNIManager(dynamicClient dynamic.Interface) *SNIManager {
	return &SNIManager{
		dynamicClient: dynamicClient,
	}
}

// UpdateSNIConfiguration updates the TLS configuration for an IngressRoute based on host rules
func (s *SNIManager) UpdateSNIConfiguration(ctx context.Context, ingressRouteName, namespace string, hostRules []string) error {
	if s.dynamicClient == nil {
		klog.V(2).Infof("No dynamicClient available, skipping SNI configuration")
		return nil
	}

	// Extract unique hosts from rules
	hosts := s.extractHostsFromRules(hostRules)
	if len(hosts) == 0 {
		klog.V(2).Infof("No host rules found, skipping SNI configuration")
		return nil
	}

	// Define the GVR for IngressRoute
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	// Get existing IngressRoute
	existingRoute, err := s.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, ingressRouteName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get IngressRoute %s: %w", ingressRouteName, err)
	}

	// Update TLS configuration
	spec, ok := existingRoute.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		existingRoute.Object["spec"] = spec
	}

	// Build TLS configuration
	tlsConfig := s.buildTLSConfig(hosts)
	if tlsConfig != nil {
		spec["tls"] = tlsConfig
	}

	// Update the IngressRoute
	_, err = s.dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, existingRoute, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update IngressRoute with SNI configuration: %w", err)
	}

	klog.V(2).Infof("Successfully updated SNI configuration for %d hosts", len(hosts))
	return nil
}

// extractHostsFromRules extracts unique hostnames from Traefik match rules
func (s *SNIManager) extractHostsFromRules(rules []string) []string {
	hostMap := make(map[string]bool)
	
	for _, rule := range rules {
		// Extract Host(`hostname`) patterns
		if strings.Contains(rule, "Host(`") {
			start := strings.Index(rule, "Host(`") + 6
			end := strings.Index(rule[start:], "`")
			if end > 0 {
				host := rule[start : start+end]
				hostMap[host] = true
			}
		}
		
		// Extract HostRegexp patterns and convert to wildcard
		if strings.Contains(rule, "HostRegexp(`") {
			start := strings.Index(rule, "HostRegexp(`") + 12
			end := strings.Index(rule[start:], "`")
			if end > 0 {
				pattern := rule[start : start+end]
				// Convert regex to wildcard domain
				if strings.Contains(pattern, "[^.]+") {
					// Convert ^[^.]+.example.com$ to *.example.com
					wildcard := strings.ReplaceAll(pattern, "^[^.]+", "*")
					wildcard = strings.TrimSuffix(wildcard, "$")
					hostMap[wildcard] = true
				}
			}
		}
	}
	
	// Convert map to slice
	var hosts []string
	for host := range hostMap {
		hosts = append(hosts, host)
	}
	
	return hosts
}

// buildTLSConfig builds the TLS configuration for the given hosts
func (s *SNIManager) buildTLSConfig(hosts []string) map[string]interface{} {
	if len(hosts) == 0 {
		return nil
	}

	// Group hosts by certificate
	certGroups := s.groupHostsByCertificate(hosts)
	
	// Build TLS domains configuration
	var domains []interface{}
	for _, group := range certGroups {
		domain := map[string]interface{}{
			"main": group.main,
		}
		
		// Add SANs if multiple hosts share the same certificate
		if len(group.sans) > 0 {
			domain["sans"] = group.sans
		}
		
		// Set secret name based on the main domain
		secretName := s.getSecretNameForHost(group.main)
		if secretName != "" {
			domain["secretName"] = secretName
		}
		
		domains = append(domains, domain)
	}
	
	tlsConfig := map[string]interface{}{
		"domains": domains,
		// Enable SNI strict mode
		"options": map[string]interface{}{
			"name": "default",
			"sniStrict": true,
		},
	}
	
	return tlsConfig
}

// CertificateGroup represents a group of hosts that share the same certificate
type CertificateGroup struct {
	main string
	sans []string
}

// groupHostsByCertificate groups hosts that can share the same certificate
func (s *SNIManager) groupHostsByCertificate(hosts []string) []CertificateGroup {
	var groups []CertificateGroup
	wildcardMap := make(map[string]*CertificateGroup)
	
	for _, host := range hosts {
		if strings.HasPrefix(host, "*.") {
			// Wildcard certificate
			domain := strings.TrimPrefix(host, "*.")
			if group, exists := wildcardMap[domain]; exists {
				// Add to existing wildcard group
				group.sans = append(group.sans, host)
			} else {
				// Create new wildcard group
				wildcardMap[domain] = &CertificateGroup{
					main: host,
					sans: []string{},
				}
			}
		} else {
			// Check if this host is covered by a wildcard
			parts := strings.Split(host, ".")
			if len(parts) > 2 {
				// Check for wildcard certificate
				domain := strings.Join(parts[1:], ".")
				if group, exists := wildcardMap[domain]; exists {
					group.sans = append(group.sans, host)
					continue
				}
			}
			
			// Individual certificate
			groups = append(groups, CertificateGroup{
				main: host,
				sans: []string{},
			})
		}
	}
	
	// Add wildcard groups to the result
	for _, group := range wildcardMap {
		groups = append(groups, *group)
	}
	
	return groups
}

// getSecretNameForHost returns the expected secret name for a given host
func (s *SNIManager) getSecretNameForHost(host string) string {
	// Convert hostname to a valid secret name
	// Replace dots with hyphens and remove wildcards
	secretName := strings.ReplaceAll(host, ".", "-")
	secretName = strings.ReplaceAll(secretName, "*", "wildcard")
	return fmt.Sprintf("%s-tls", secretName)
}

// CreateTLSSecret creates a Kubernetes secret from certificate data
func (s *SNIManager) CreateTLSSecret(ctx context.Context, namespace, secretName string, certData, keyData []byte) error {
	if s.dynamicClient == nil {
		return fmt.Errorf("no dynamicClient available")
	}

	// Create secret object
	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"kecs.io/managed": "true",
					"kecs.io/type":    "tls-certificate",
				},
			},
			"type": "kubernetes.io/tls",
			"data": map[string]interface{}{
				"tls.crt": certData,
				"tls.key": keyData,
			},
		},
	}

	// Define the GVR for Secret
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}

	// Create the secret
	_, err := s.dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create TLS secret: %w", err)
	}

	klog.V(2).Infof("Created TLS secret %s in namespace %s", secretName, namespace)
	return nil
}

// ListTLSSecrets lists all TLS secrets managed by KECS
func (s *SNIManager) ListTLSSecrets(ctx context.Context, namespace string) ([]string, error) {
	if s.dynamicClient == nil {
		return nil, fmt.Errorf("no dynamicClient available")
	}

	// Define the GVR for Secret
	gvr := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}

	// List secrets with KECS labels
	secrets, err := s.dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "kecs.io/managed=true,kecs.io/type=tls-certificate",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list TLS secrets: %w", err)
	}

	var secretNames []string
	for _, item := range secrets.Items {
		metadata, ok := item.Object["metadata"].(map[string]interface{})
		if ok {
			if name, ok := metadata["name"].(string); ok {
				secretNames = append(secretNames, name)
			}
		}
	}

	return secretNames, nil
}