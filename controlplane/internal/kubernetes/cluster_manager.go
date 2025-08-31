package kubernetes

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClusterManager defines the interface for managing local Kubernetes clusters
type ClusterManager interface {
	// CreateCluster creates a new Kubernetes cluster
	CreateCluster(ctx context.Context, clusterName string) error

	// DeleteCluster deletes an existing Kubernetes cluster
	DeleteCluster(ctx context.Context, clusterName string) error

	// StopCluster stops a running Kubernetes cluster
	StopCluster(ctx context.Context, clusterName string) error

	// StartCluster starts a stopped Kubernetes cluster
	StartCluster(ctx context.Context, clusterName string) error

	// ClusterExists checks if a cluster exists
	ClusterExists(ctx context.Context, clusterName string) (bool, error)

	// GetKubeClient returns a Kubernetes client for the specified cluster
	GetKubeClient(ctx context.Context, clusterName string) (kubernetes.Interface, error)

	// GetKubeConfig returns the REST config for the specified cluster
	GetKubeConfig(ctx context.Context, clusterName string) (*rest.Config, error)

	// WaitForClusterReady waits for a cluster to be ready with the specified timeout
	WaitForClusterReady(ctx context.Context, clusterName string) error

	// GetKubeconfigPath returns the path to the kubeconfig file for the cluster
	GetKubeconfigPath(clusterName string) string

	// GetClusterInfo returns information about the cluster
	GetClusterInfo(ctx context.Context, clusterName string) (*ClusterInfo, error)

	// ListClusters returns a list of all existing clusters
	ListClusters(ctx context.Context) ([]ClusterInfo, error)

	// IsClusterRunning checks if a cluster is currently running
	IsClusterRunning(ctx context.Context, clusterName string) (bool, error)
}

// ClusterInfo contains information about a cluster
type ClusterInfo struct {
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	Provider       string            `json:"provider"` // "kind" or "k3d"
	NodeCount      int               `json:"nodeCount"`
	Version        string            `json:"version"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	KubeconfigPath string            `json:"kubeconfigPath,omitempty"`
	APIPort        int               `json:"apiPort,omitempty"`
	TraefikPort    int               `json:"traefikPort,omitempty"`
	Running        bool              `json:"running"`
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

	// VolumeMounts specifies volume mounts for the cluster
	VolumeMounts []VolumeMount `json:"volumeMounts,omitempty"`

	// APIPort specifies the port to expose for the k3d API server
	APIPort int `json:"apiPort,omitempty"`

	// K3dImage specifies the k3s image to use
	K3dImage string `json:"k3dImage,omitempty"`

	// EnableRegistry enables k3d registry for dev mode
	EnableRegistry bool `json:"enableRegistry,omitempty"`

	// RegistryPort specifies the port for the k3d registry (default: 5000)
	RegistryPort int `json:"registryPort,omitempty"`

	// TestMode enables test mode which uses mock implementations (for CI/testing)
	TestMode bool `json:"testMode,omitempty"`
}

// VolumeMount represents a volume mount configuration
type VolumeMount struct {
	// HostPath is the path on the host
	HostPath string `json:"hostPath"`
	// ContainerPath is the path inside the container
	ContainerPath string `json:"containerPath"`
}

// NewClusterManager is deprecated and should not be used.
// K3d cluster manager has been moved to host/k3d package.
// For host-side operations (TUI, CLI), use host/k3d.NewK3dClusterManager.
// For cluster-internal operations (control plane), use in-cluster config directly.
//
// Deprecated: This function will panic if called. Use host/k3d package instead.
func NewClusterManager(config *ClusterManagerConfig) (ClusterManager, error) {
	panic("NewClusterManager is deprecated. Use host/k3d.NewK3dClusterManager for host-side operations.")
}
