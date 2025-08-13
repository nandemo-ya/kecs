package phase1

import (
	"strings"
)

// containsClusterName checks if an ARN contains the given cluster name
func containsClusterName(arn, clusterName string) bool {
	// ARN format: arn:aws:ecs:region:account:cluster/cluster-name
	// We need to check the last part after the last slash
	parts := splitARN(arn)
	if len(parts) > 0 {
		return parts[len(parts)-1] == clusterName
	}
	// Fallback to simple contains check for non-standard ARNs
	return strings.Contains(arn, "cluster/"+clusterName)
}

// splitARN splits ARN by slashes
func splitARN(arn string) []string {
	var parts []string
	current := ""
	for _, char := range arn {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}