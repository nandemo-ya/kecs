package kubernetes

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestClusterManagerFactory(t *testing.T) {
	tests := []struct {
		name     string
		config   *ClusterManagerConfig
		expected string
	}{
		{
			name: "k3d provider",
			config: &ClusterManagerConfig{
				Provider: "k3d",
			},
			expected: "*kubernetes.K3dClusterManager",
		},
		{
			name: "kind provider",
			config: &ClusterManagerConfig{
				Provider: "kind",
			},
			expected: "*kubernetes.KindClusterManager",
		},
		{
			name:     "default provider (should be k3d)",
			config:   &ClusterManagerConfig{},
			expected: "*kubernetes.K3dClusterManager",
		},
		{
			name: "unknown provider (should default to k3d)",
			config: &ClusterManagerConfig{
				Provider: "unknown",
			},
			expected: "*kubernetes.K3dClusterManager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewClusterManager(tt.config)
			if err != nil {
				t.Fatalf("NewClusterManager() error = %v", err)
			}

			managerType := fmt.Sprintf("%T", manager)
			if managerType != tt.expected {
				t.Errorf("Expected manager type %s, got %s", tt.expected, managerType)
			}
		})
	}
}

func TestK3dClusterManagerBasicOperations(t *testing.T) {
	// Skip if Docker is not available
	if !isDockerAvailable() {
		t.Skip("Docker not available, skipping k3d tests")
	}

	config := &ClusterManagerConfig{
		Provider:      "k3d",
		ContainerMode: false, // Use normal mode for testing
	}

	manager, err := NewK3dClusterManager(config)
	if err != nil {
		t.Fatalf("Failed to create K3dClusterManager: %v", err)
	}

	ctx := context.Background()
	testClusterName := "test-cluster-k3d"

	t.Run("ClusterExists_NonExistent", func(t *testing.T) {
		exists, err := manager.ClusterExists(ctx, testClusterName)
		if err != nil {
			t.Fatalf("ClusterExists() error = %v", err)
		}
		if exists {
			t.Error("Expected cluster to not exist")
		}
	})

	t.Run("GetKubeconfigPath", func(t *testing.T) {
		path := manager.GetKubeconfigPath(testClusterName)
		if path == "" {
			t.Error("Expected non-empty kubeconfig path")
		}
		t.Logf("Kubeconfig path: %s", path)
	})

	t.Run("ClusterConfiguration", func(t *testing.T) {
		// Test that manager is properly configured
		if manager.runtime == nil {
			t.Error("Expected runtime to be initialized")
		}
		if manager.config == nil {
			t.Error("Expected config to be initialized")
		}
		if manager.config.Provider != "k3d" {
			t.Errorf("Expected provider k3d, got %s", manager.config.Provider)
		}
	})

	// Skip actual cluster operations that require Docker daemon
	t.Run("CreateCluster_SkipActualCreation", func(t *testing.T) {
		t.Skip("Skipping actual cluster creation to avoid Docker daemon dependency in CI")
	})
}

func TestKindClusterManagerCompatibility(t *testing.T) {
	// This test ensures that KindClusterManager implements the interface correctly
	config := &ClusterManagerConfig{
		Provider: "kind",
	}

	manager, err := NewKindClusterManager(config)
	if err != nil {
		t.Fatalf("Failed to create KindClusterManager: %v", err)
	}

	// Test that it implements the interface
	var _ ClusterManager = manager

	// Test basic interface methods (without actually creating clusters)
	ctx := context.Background()
	testClusterName := "test-cluster-kind"

	t.Run("ClusterExists", func(t *testing.T) {
		exists, err := manager.ClusterExists(ctx, testClusterName)
		if err != nil {
			// This might fail if Docker CLI is not available, which is expected
			t.Logf("ClusterExists() error (expected in container mode): %v", err)
		} else {
			// Should return false for non-existent cluster
			if exists {
				t.Error("Expected cluster to not exist")
			}
		}
	})

	t.Run("GetKubeconfigPath", func(t *testing.T) {
		path := manager.GetKubeconfigPath(testClusterName)
		if path == "" {
			t.Error("Expected non-empty kubeconfig path")
		}
		t.Logf("Kubeconfig path: %s", path)
	})
}

// isDockerAvailable checks if Docker is available for testing
func isDockerAvailable() bool {
	// Check if Docker socket exists
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		return true
	}

	// For non-Unix systems or different Docker setups
	// We could also try to create a Docker client and ping
	return false
}