package servicediscovery

import (
	"fmt"
	"strings"
)

// CoreDNSConfig generates CoreDNS configuration for service discovery
type CoreDNSConfig struct {
	namespaces map[string]*Namespace
	services   map[string]*Service
}

// GenerateCoreDNSConfig generates CoreDNS configuration for Cloud Map compatibility
func GenerateCoreDNSConfig(namespaces map[string]*Namespace, services map[string]*Service) string {
	var config strings.Builder

	// Group services by namespace
	servicesByNamespace := make(map[string][]*Service)
	for _, service := range services {
		servicesByNamespace[service.NamespaceID] = append(servicesByNamespace[service.NamespaceID], service)
	}

	// Generate configuration for each namespace
	for namespaceID, namespace := range namespaces {
		if namespace.Type != "DNS_PRIVATE" {
			continue
		}

		config.WriteString(fmt.Sprintf("# Namespace: %s\n", namespace.Name))
		config.WriteString(fmt.Sprintf("%s:53 {\n", namespace.Name))
		config.WriteString("    errors\n")
		config.WriteString("    health\n")
		config.WriteString("    ready\n")

		// Use Kubernetes plugin for service discovery
		config.WriteString("    kubernetes cluster.local in-addr.arpa ip6.arpa {\n")
		config.WriteString("        pods insecure\n")
		config.WriteString("        fallthrough in-addr.arpa ip6.arpa\n")
		config.WriteString("        ttl 30\n")
		config.WriteString("    }\n")

		// Rewrite rules for Cloud Map compatibility
		if services, ok := servicesByNamespace[namespaceID]; ok {
			for _, service := range services {
				// Rewrite service.namespace to sd-service.k8s-namespace
				k8sServiceName := fmt.Sprintf("sd-%s", service.Name)
				config.WriteString(fmt.Sprintf("    rewrite name %s.%s %s.default.svc.cluster.local\n",
					service.Name, namespace.Name, k8sServiceName))
			}
		}

		config.WriteString("    forward . /etc/resolv.conf\n")
		config.WriteString("    cache 30\n")
		config.WriteString("    loop\n")
		config.WriteString("    reload\n")
		config.WriteString("    loadbalance\n")
		config.WriteString("}\n\n")
	}

	return config.String()
}

// GenerateCoreDNSConfigMap generates a ConfigMap for CoreDNS
func GenerateCoreDNSConfigMap(namespaces map[string]*Namespace, services map[string]*Service) map[string]string {
	corefile := GenerateCoreDNSConfig(namespaces, services)

	// Add default configuration
	defaultConfig := `.:53 {
    errors
    health
    ready
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
        fallthrough in-addr.arpa ip6.arpa
        ttl 30
    }
    forward . /etc/resolv.conf
    cache 30
    loop
    reload
    loadbalance
}
`

	return map[string]string{
		"Corefile": defaultConfig + "\n" + corefile,
	}
}
