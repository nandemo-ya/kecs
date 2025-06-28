package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// K3dClusterExists checks if a k3d cluster exists
func K3dClusterExists(clusterName string) (bool, error) {
	cmd := exec.Command("k3d", "cluster", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	// Simple check for cluster name in output
	return strings.Contains(string(output), fmt.Sprintf(`"name":"%s"`, clusterName)), nil
}

// DeleteK3dCluster deletes a k3d cluster if it exists
func DeleteK3dCluster(clusterName string) error {
	exists, err := K3dClusterExists(clusterName)
	if err != nil {
		return err
	}

	if exists {
		cmd := exec.Command("k3d", "cluster", "delete", clusterName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to delete k3d cluster %s: %w", clusterName, err)
		}
	}

	return nil
}

// GetK3dClusters returns a list of k3d cluster names
func GetK3dClusters() ([]string, error) {
	cmd := exec.Command("k3d", "cluster", "list", "-o", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list k3d clusters: %w", err)
	}

	// Parse JSON output to extract cluster names
	// For simplicity, use a basic approach
	clusters := []string{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, `"name":"`) {
			// Extract cluster name
			start := strings.Index(line, `"name":"`) + 8
			end := strings.Index(line[start:], `"`)
			if end > 0 {
				clusterName := line[start : start+end]
				clusters = append(clusters, clusterName)
			}
		}
	}

	return clusters, nil
}