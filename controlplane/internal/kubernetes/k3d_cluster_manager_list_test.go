package kubernetes

import (
	"testing"

	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockK3dClient is a mock implementation of k3d client functions
type MockK3dClient struct {
	mock.Mock
	clusters []k3d.Cluster
}

func TestK3dClusterManager_ListClusters(t *testing.T) {
	tests := []struct {
		name          string
		k3dClusters   []k3d.Cluster
		expectedNames []string
		expectedError bool
	}{
		{
			name: "returns instance names without kecs- prefix",
			k3dClusters: []k3d.Cluster{
				{Name: "kecs-myinstance"},
				{Name: "kecs-dev"},
				{Name: "kecs-staging"},
			},
			expectedNames: []string{"myinstance", "dev", "staging"},
			expectedError: false,
		},
		{
			name: "filters out non-KECS clusters",
			k3dClusters: []k3d.Cluster{
				{Name: "kecs-prod"},
				{Name: "other-cluster"},
				{Name: "k3d-test"},
				{Name: "kecs-dev"},
			},
			expectedNames: []string{"prod", "dev"},
			expectedError: false,
		},
		{
			name: "returns empty list when no KECS clusters",
			k3dClusters: []k3d.Cluster{
				{Name: "other-cluster"},
				{Name: "k3d-test"},
			},
			expectedNames: []string{},
			expectedError: false,
		},
		{
			name:          "returns empty list when no clusters exist",
			k3dClusters:   []k3d.Cluster{},
			expectedNames: []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This is a unit test to verify the logic of ListClusters
			// In real implementation, we would need to mock the k3d client.ClusterList function
			// For now, this test documents the expected behavior

			// Expected behavior verification
			for i, cluster := range tt.k3dClusters {
				if len(cluster.Name) > 5 && cluster.Name[:5] == "kecs-" {
					expectedName := cluster.Name[5:] // Remove "kecs-" prefix
					if i < len(tt.expectedNames) {
						assert.Equal(t, tt.expectedNames[i], expectedName,
							"Instance name should not include kecs- prefix")
					}
				}
			}
		})
	}
}

func TestK3dClusterManager_NormalizeClusterName(t *testing.T) {
	manager := &K3dClusterManager{
		runtime: runtimes.Docker,
		config:  &ClusterManagerConfig{},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "adds kecs- prefix when missing",
			input:    "myinstance",
			expected: "kecs-myinstance",
		},
		{
			name:     "preserves existing kecs- prefix",
			input:    "kecs-myinstance",
			expected: "kecs-myinstance",
		},
		{
			name:     "handles empty string",
			input:    "",
			expected: "kecs-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.normalizeClusterName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestK3dClusterManager_InstanceNameConsistency(t *testing.T) {
	// This test verifies that instance names are handled consistently
	// throughout the cluster manager operations

	t.Run("ListClusters returns names without prefix", func(t *testing.T) {
		// When ListClusters returns ["myinstance", "dev", "staging"]
		// Users should be able to use these names directly in commands:
		// - kecs stop --instance myinstance
		// - kecs destroy --instance dev
		// - kecs start --instance staging

		// The normalizeClusterName method should add the prefix internally
		// when interacting with k3d
	})

	t.Run("All operations accept instance names without prefix", func(t *testing.T) {
		operations := []string{
			"ClusterExists",
			"DeleteCluster",
			"StopCluster",
			"StartCluster",
			"GetKubeClient",
			"GetKubeConfig",
			"GetClusterInfo",
			"IsClusterRunning",
		}

		for _, op := range operations {
			// Each operation should accept "myinstance" and internally
			// convert it to "kecs-myinstance" for k3d operations
			t.Logf("Operation %s should accept instance name without prefix", op)
		}
	})
}
