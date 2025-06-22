package kubernetes

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"
)

// ClusterManager defines the interface for managing local Kubernetes clusters
type ClusterManager interface {
	// CreateCluster creates a new Kubernetes cluster
	CreateCluster(ctx context.Context, clusterName string) error

	// DeleteCluster deletes an existing Kubernetes cluster
	DeleteCluster(ctx context.Context, clusterName string) error

	// ClusterExists checks if a cluster exists
	ClusterExists(ctx context.Context, clusterName string) (bool, error)

	// GetKubeClient returns a Kubernetes client for the specified cluster
	GetKubeClient(clusterName string) (kubernetes.Interface, error)

	// WaitForClusterReady waits for a cluster to be ready with the specified timeout
	WaitForClusterReady(clusterName string, timeout time.Duration) error

	// GetKubeconfigPath returns the path to the kubeconfig file for the cluster
	GetKubeconfigPath(clusterName string) string

	// GetClusterInfo returns information about the cluster
	GetClusterInfo(ctx context.Context, clusterName string) (*ClusterInfo, error)
}

// ClusterInfo contains information about a cluster
type ClusterInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Provider  string            `json:"provider"` // "kind" or "k3d"
	NodeCount int               `json:"nodeCount"`
	Version   string            `json:"version"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ClusterManagerConfig contains configuration for cluster managers
type ClusterManagerConfig struct {
	// Provider specifies which cluster manager to use ("kind" or "k3d")
	Provider string `json:"provider"`
	
	// ContainerMode indicates if running in container mode
	ContainerMode bool `json:"containerMode"`
	
	// KubeconfigPath specifies custom kubeconfig directory
	KubeconfigPath string `json:"kubeconfigPath,omitempty"`
	
	// HostAddress for container mode networking
	HostAddress string `json:"hostAddress,omitempty"`
	
	// AdditionalOptions for provider-specific configuration
	AdditionalOptions map[string]interface{} `json:"additionalOptions,omitempty"`
}

// NewClusterManager creates a new cluster manager based on the configuration
func NewClusterManager(config *ClusterManagerConfig) (ClusterManager, error) {
	switch config.Provider {
	case "kind":
		return NewKindClusterManager(config)
	case "k3d":
		return NewK3dClusterManager(config)
	default:
		// Default to k3d for new installations
		return NewK3dClusterManager(config)
	}
}