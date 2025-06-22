package kubernetes

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/k3d-io/k3d/v5/pkg/client"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	k3d "github.com/k3d-io/k3d/v5/pkg/types"
)

// K3dClusterManager implements ClusterManager interface using k3d
type K3dClusterManager struct {
	runtime runtimes.Runtime
	config  *ClusterManagerConfig
}

// NewK3dClusterManager creates a new k3d-based cluster manager
func NewK3dClusterManager(cfg *ClusterManagerConfig) (*K3dClusterManager, error) {
	if cfg == nil {
		cfg = &ClusterManagerConfig{
			Provider:      "k3d",
			ContainerMode: os.Getenv("KECS_CONTAINER_MODE") == "true",
		}
	}

	// Initialize k3d runtime (defaults to Docker)
	runtime, err := runtimes.GetRuntime("docker")
	if err != nil {
		return nil, fmt.Errorf("failed to get k3d runtime: %w", err)
	}

	return &K3dClusterManager{
		runtime: runtime,
		config:  cfg,
	}, nil
}

// CreateCluster creates a new k3d cluster
func (k *K3dClusterManager) CreateCluster(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster already exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if exists {
		log.Printf("k3d cluster %s already exists", normalizedName)
		return nil
	}

	// Create cluster configuration
	cluster := &k3d.Cluster{
		Name: normalizedName,
		Network: k3d.ClusterNetwork{
			Name: fmt.Sprintf("k3d-%s", normalizedName),
		},
		Token: fmt.Sprintf("kecs-%s-token", normalizedName),
	}
	
	// Add server node to cluster
	serverNode := &k3d.Node{
		Name:    fmt.Sprintf("k3d-%s-server-0", normalizedName),
		Role:    k3d.ServerRole,
		Image:   "rancher/k3s:v1.31.4-k3s1",
		Restart: true,
	}
	cluster.Nodes = append(cluster.Nodes, serverNode)

	// Create cluster creation options with minimal configuration
	clusterCreateOpts := &k3d.ClusterCreateOpts{
		WaitForServer: true,
		Timeout:       2 * time.Minute,
	}

	// Create the cluster
	log.Printf("Creating k3d cluster %s...", normalizedName)
	if err := client.ClusterCreate(ctx, k.runtime, cluster, clusterCreateOpts); err != nil {
		return fmt.Errorf("failed to create k3d cluster: %w", err)
	}

	// Write kubeconfig to custom path if in container mode
	if k.config.ContainerMode {
		kubeconfigPath := k.GetKubeconfigPath(clusterName)
		if err := k.writeKubeconfig(ctx, cluster, kubeconfigPath); err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}
	}

	log.Printf("Successfully created k3d cluster %s", normalizedName)
	return nil
}

// DeleteCluster deletes a k3d cluster
func (k *K3dClusterManager) DeleteCluster(ctx context.Context, clusterName string) error {
	normalizedName := k.normalizeClusterName(clusterName)

	// Check if cluster exists
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		log.Printf("k3d cluster %s does not exist", normalizedName)
		return nil
	}

	// Get cluster object
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Delete the cluster
	log.Printf("Deleting k3d cluster %s...", normalizedName)
	deleteOpts := k3d.ClusterDeleteOpts{
		SkipRegistryCheck: true,
	}

	if err := client.ClusterDelete(ctx, k.runtime, cluster, deleteOpts); err != nil {
		return fmt.Errorf("failed to delete k3d cluster: %w", err)
	}

	// Clean up kubeconfig file
	kubeconfigPath := k.GetKubeconfigPath(clusterName)
	if kubeconfigPath != "" {
		os.Remove(kubeconfigPath)
	}

	log.Printf("Successfully deleted k3d cluster %s", normalizedName)
	return nil
}

// ClusterExists checks if a k3d cluster exists
func (k *K3dClusterManager) ClusterExists(ctx context.Context, clusterName string) (bool, error) {
	normalizedName := k.normalizeClusterName(clusterName)

	clusters, err := client.ClusterList(ctx, k.runtime)
	if err != nil {
		return false, fmt.Errorf("failed to list clusters: %w", err)
	}

	for _, cluster := range clusters {
		if cluster.Name == normalizedName {
			return true, nil
		}
	}

	return false, nil
}

// GetKubeClient returns a Kubernetes client for the specified cluster
func (k *K3dClusterManager) GetKubeClient(clusterName string) (kubernetes.Interface, error) {
	kubeconfigPath := k.GetKubeconfigPath(clusterName)

	// Check if kubeconfig file exists
	if _, err := os.Stat(kubeconfigPath); err != nil {
		return nil, fmt.Errorf("kubeconfig not found at %s: %w", kubeconfigPath, err)
	}

	// Build config from kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// WaitForClusterReady waits for a k3d cluster to be ready
func (k *K3dClusterManager) WaitForClusterReady(clusterName string, timeout time.Duration) error {
	kubeconfigPath := k.GetKubeconfigPath(clusterName)
	startTime := time.Now()

	log.Printf("Waiting for k3d cluster %s to be ready (kubeconfig: %s)", clusterName, kubeconfigPath)

	for {
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout waiting for cluster %s to be ready after %v", clusterName, timeout)
		}

		// Check if kubeconfig file exists
		if _, err := os.Stat(kubeconfigPath); err != nil {
			if os.IsNotExist(err) {
				log.Printf("Kubeconfig not found yet for cluster %s, retrying...", clusterName)
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("error checking kubeconfig: %w", err)
		}

		// Try to create a client and check connectivity
		client, err := k.GetKubeClient(clusterName)
		if err != nil {
			log.Printf("Failed to create client for cluster %s: %v, retrying...", clusterName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Try to list nodes to verify connectivity
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		cancel()

		if err != nil {
			log.Printf("Failed to connect to cluster %s API: %v, retrying...", clusterName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		log.Printf("k3d cluster %s is ready", clusterName)
		return nil
	}
}

// GetKubeconfigPath returns the path to the kubeconfig file for the cluster
func (k *K3dClusterManager) GetKubeconfigPath(clusterName string) string {
	if k.config.ContainerMode {
		kubeconfigPath := k.config.KubeconfigPath
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("KECS_KUBECONFIG_PATH")
			if kubeconfigPath == "" {
				kubeconfigPath = "/kecs/kubeconfig"
			}
		}
		os.MkdirAll(kubeconfigPath, 0755)
		normalizedName := k.normalizeClusterName(clusterName)
		return filepath.Join(kubeconfigPath, fmt.Sprintf("%s.config", normalizedName))
	}

	// For non-container mode, use k3d's default kubeconfig location
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", "config")
}

// GetClusterInfo returns information about the cluster
func (k *K3dClusterManager) GetClusterInfo(ctx context.Context, clusterName string) (*ClusterInfo, error) {
	normalizedName := k.normalizeClusterName(clusterName)

	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("cluster %s does not exist", clusterName)
	}

	// Get cluster details
	cluster, err := client.ClusterGet(ctx, k.runtime, &k3d.Cluster{Name: normalizedName})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster details: %w", err)
	}

	// Count nodes
	nodeCount := len(cluster.Nodes)

	// Try to get Kubernetes version
	version := "unknown"
	if kubeClient, err := k.GetKubeClient(clusterName); err == nil {
		if serverVersion, err := kubeClient.Discovery().ServerVersion(); err == nil {
			version = serverVersion.GitVersion
		}
	}

	return &ClusterInfo{
		Name:      clusterName,
		Status:    "Running", // k3d clusters are either running or don't exist
		Provider:  "k3d",
		NodeCount: nodeCount,
		Version:   version,
		Metadata: map[string]string{
			"internal_name": normalizedName,
			"image":         "rancher/k3s:v1.31.4-k3s1",
		},
	}, nil
}

// normalizeClusterName ensures cluster name has the kecs- prefix for k3d
func (k *K3dClusterManager) normalizeClusterName(clusterName string) string {
	if !strings.HasPrefix(clusterName, "kecs-") {
		return fmt.Sprintf("kecs-%s", clusterName)
	}
	return clusterName
}

// writeKubeconfig writes the kubeconfig for the cluster to the specified path
func (k *K3dClusterManager) writeKubeconfig(ctx context.Context, cluster *k3d.Cluster, kubeconfigPath string) error {
	// Get kubeconfig from k3d
	kubecfg, err := client.KubeconfigGet(ctx, k.runtime, cluster)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Write kubeconfig to file
	if err := client.KubeconfigWrite(ctx, kubecfg, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	log.Printf("Wrote kubeconfig for cluster %s to %s", cluster.Name, kubeconfigPath)
	return nil
}