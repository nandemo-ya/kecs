package servicediscovery

import (
	"fmt"
	"strings"
)

// formatServiceDNSName formats the full DNS name for a service
func formatServiceDNSName(serviceName, namespaceName string) string {
	return fmt.Sprintf("%s.%s", serviceName, namespaceName)
}

// formatKubernetesServiceName formats the Kubernetes service name from ECS service discovery service
func formatKubernetesServiceName(serviceName string) string {
	// Prefix with sd- to distinguish from regular services
	return fmt.Sprintf("sd-%s", serviceName)
}

// formatKubernetesDNSName formats the full Kubernetes DNS name for a service
func formatKubernetesDNSName(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", formatKubernetesServiceName(serviceName), namespace)
}

// isValidDNSName checks if a name is valid for DNS
func isValidDNSName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	// Check each label
	labels := strings.Split(name, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		// Must start with alphanumeric
		if !isAlphanumeric(label[0]) {
			return false
		}
		// Must end with alphanumeric
		if !isAlphanumeric(label[len(label)-1]) {
			return false
		}
		// Check all characters
		for _, ch := range label {
			if !isAlphanumeric(byte(ch)) && ch != '-' {
				return false
			}
		}
	}
	return true
}

// isAlphanumeric checks if a character is alphanumeric
func isAlphanumeric(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// sanitizeDNSLabel sanitizes a string to be a valid DNS label
func sanitizeDNSLabel(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")

	// Remove invalid characters
	var result strings.Builder
	for i, ch := range s {
		if isAlphanumeric(byte(ch)) || ch == '-' {
			// Don't start or end with hyphen
			if ch == '-' && (i == 0 || i == len(s)-1) {
				continue
			}
			result.WriteRune(ch)
		}
	}

	// Truncate if too long
	res := result.String()
	if len(res) > 63 {
		res = res[:63]
	}

	// Ensure doesn't end with hyphen after truncation
	res = strings.TrimSuffix(res, "-")

	if res == "" {
		return "default"
	}
	return res
}
