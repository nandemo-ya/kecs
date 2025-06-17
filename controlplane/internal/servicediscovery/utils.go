package servicediscovery

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// generateID generates a random ID for resources
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:12]
}

// extractK8sNamespace extracts a Kubernetes namespace from DNS namespace name
func extractK8sNamespace(dnsNamespace string) string {
	// Remove common suffixes
	name := strings.TrimSuffix(dnsNamespace, ".local")
	name = strings.TrimSuffix(name, ".ecs")
	name = strings.TrimSuffix(name, ".internal")
	
	// Replace dots with hyphens
	name = strings.ReplaceAll(name, ".", "-")
	
	// Ensure it's a valid Kubernetes namespace name
	name = strings.ToLower(name)
	
	// Limit length
	if len(name) > 63 {
		name = name[:63]
	}
	
	// Remove trailing hyphens
	name = strings.TrimRight(name, "-")
	
	if name == "" {
		return "default"
	}
	
	return name
}