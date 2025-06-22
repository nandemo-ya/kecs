package kubernetes

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"
)

// KindClusterManager implements ClusterManager interface using Kind
type KindClusterManager struct {
	provider *cluster.Provider
	config   *ClusterManagerConfig
}

// NewKindClusterManager creates a new Kind-based cluster manager
func NewKindClusterManager(config *ClusterManagerConfig) (*KindClusterManager, error) {
	if config == nil {
		config = &ClusterManagerConfig{
			Provider:      "kind",
			ContainerMode: os.Getenv("KECS_CONTAINER_MODE") == "true",
		}
	}

	return &KindClusterManager{
		provider: cluster.NewProvider(
			cluster.ProviderWithLogger(cmd.NewLogger()),
		),
		config: config,
	}, nil
}

// CreateCluster creates a new Kind cluster
func (k *KindClusterManager) CreateCluster(ctx context.Context, clusterName string) error {
	kecsClusterName := k.normalizeClusterName(clusterName)

	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if exists {
		return nil
	}

	kubeconfigPath := k.GetKubeconfigPath(clusterName)

	err = k.provider.Create(
		kecsClusterName,
		cluster.CreateWithNodeImage("kindest/node:v1.31.4"),
		cluster.CreateWithWaitForReady(0),
		cluster.CreateWithKubeconfigPath(kubeconfigPath),
		cluster.CreateWithDisplayUsage(false),
		cluster.CreateWithDisplaySalutation(false),
	)

	if err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
	}

	// In container mode, adjust the kubeconfig for container access
	if k.config.ContainerMode {
		if err := k.adjustKubeconfigForContainer(kubeconfigPath); err != nil {
			return fmt.Errorf("failed to adjust kubeconfig for container: %w", err)
		}
	}

	return nil
}

// DeleteCluster deletes a Kind cluster
func (k *KindClusterManager) DeleteCluster(ctx context.Context, clusterName string) error {
	kecsClusterName := k.normalizeClusterName(clusterName)

	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		return nil
	}

	return k.provider.Delete(kecsClusterName, k.GetKubeconfigPath(clusterName))
}

// ClusterExists checks if a Kind cluster exists
func (k *KindClusterManager) ClusterExists(ctx context.Context, clusterName string) (bool, error) {
	kecsClusterName := k.normalizeClusterName(clusterName)
	clusters, err := k.provider.List()
	if err != nil {
		return false, err
	}

	for _, c := range clusters {
		if c == kecsClusterName {
			return true, nil
		}
	}

	return false, nil
}

// GetKubeClient returns a Kubernetes client for the specified cluster
func (k *KindClusterManager) GetKubeClient(clusterName string) (kubernetes.Interface, error) {
	kubeconfigPath := k.GetKubeconfigPath(clusterName)

	// In container mode, ensure kubeconfig is adjusted
	if k.config.ContainerMode {
		if _, err := os.Stat(kubeconfigPath); err == nil {
			if err := k.adjustKubeconfigForContainer(kubeconfigPath); err != nil {
				log.Printf("Warning: failed to adjust kubeconfig for container: %v", err)
			}
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return clientset, nil
}

// WaitForClusterReady waits for a Kind cluster to be ready
func (k *KindClusterManager) WaitForClusterReady(clusterName string, timeout time.Duration) error {
	kubeconfigPath := k.GetKubeconfigPath(clusterName)
	startTime := time.Now()

	log.Printf("Waiting for cluster %s to be ready (kubeconfig: %s)", clusterName, kubeconfigPath)

	for {
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout waiting for cluster %s to be ready after %v", clusterName, timeout)
		}

		if _, err := os.Stat(kubeconfigPath); err != nil {
			if os.IsNotExist(err) {
				log.Printf("Kubeconfig not found yet for cluster %s, retrying...", clusterName)
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("error checking kubeconfig: %w", err)
		}

		client, err := k.GetKubeClient(clusterName)
		if err != nil {
			log.Printf("Failed to create client for cluster %s: %v, retrying...", clusterName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
		cancel()

		if err != nil {
			log.Printf("Failed to connect to cluster %s API: %v, retrying...", clusterName, err)
			time.Sleep(2 * time.Second)
			continue
		}

		log.Printf("Cluster %s is ready", clusterName)
		return nil
	}
}

// GetKubeconfigPath returns the path to the kubeconfig file for the cluster
func (k *KindClusterManager) GetKubeconfigPath(clusterName string) string {
	kecsClusterName := k.normalizeClusterName(clusterName)

	if k.config.ContainerMode {
		kubeconfigPath := k.config.KubeconfigPath
		if kubeconfigPath == "" {
			kubeconfigPath = os.Getenv("KECS_KUBECONFIG_PATH")
			if kubeconfigPath == "" {
				kubeconfigPath = "/kecs/kubeconfig"
			}
		}
		os.MkdirAll(kubeconfigPath, 0755)
		return filepath.Join(kubeconfigPath, fmt.Sprintf("%s.config", kecsClusterName))
	}

	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", fmt.Sprintf("%s.config", kecsClusterName))
}

// GetClusterInfo returns information about the cluster
func (k *KindClusterManager) GetClusterInfo(ctx context.Context, clusterName string) (*ClusterInfo, error) {
	exists, err := k.ClusterExists(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("cluster %s does not exist", clusterName)
	}

	// Get cluster nodes
	kecsClusterName := k.normalizeClusterName(clusterName)
	nodes, err := k.provider.ListNodes(kecsClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	status := "Running"
	if len(nodes) == 0 {
		status = "Unknown"
	}

	// Try to get Kubernetes version
	version := "unknown"
	if client, err := k.GetKubeClient(clusterName); err == nil {
		if serverVersion, err := client.Discovery().ServerVersion(); err == nil {
			version = serverVersion.GitVersion
		}
	}

	return &ClusterInfo{
		Name:      clusterName,
		Status:    status,
		Provider:  "kind",
		NodeCount: len(nodes),
		Version:   version,
		Metadata: map[string]string{
			"internal_name": kecsClusterName,
		},
	}, nil
}

// normalizeClusterName ensures cluster name has the kecs- prefix for Kind
func (k *KindClusterManager) normalizeClusterName(clusterName string) string {
	if !strings.HasPrefix(clusterName, "kecs-") {
		return fmt.Sprintf("kecs-%s", clusterName)
	}
	return clusterName
}

// adjustKubeconfigForContainer modifies the kubeconfig to work from within a container
func (k *KindClusterManager) adjustKubeconfigForContainer(kubeconfigPath string) error {
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	hostAddr := k.getHostAddress()

	for _, cluster := range config.Clusters {
		if strings.Contains(cluster.Server, "localhost") || strings.Contains(cluster.Server, "127.0.0.1") {
			if strings.Contains(cluster.Server, "https://localhost:") {
				cluster.Server = strings.Replace(cluster.Server, "localhost", hostAddr, 1)
			} else if strings.Contains(cluster.Server, "https://127.0.0.1:") {
				cluster.Server = strings.Replace(cluster.Server, "127.0.0.1", hostAddr, 1)
			}
			log.Printf("Adjusted cluster server address to: %s", cluster.Server)
		}
	}

	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to write adjusted kubeconfig: %w", err)
	}

	return nil
}

// getHostAddress returns the appropriate host address for accessing from within a container
func (k *KindClusterManager) getHostAddress() string {
	if k.config.HostAddress != "" {
		return k.config.HostAddress
	}

	if hostAddr := os.Getenv("KECS_HOST_DOCKER_INTERNAL"); hostAddr != "" {
		return hostAddr
	}

	switch runtime.GOOS {
	case "darwin", "windows":
		return "host.docker.internal"
	default:
		return "172.17.0.1"
	}
}