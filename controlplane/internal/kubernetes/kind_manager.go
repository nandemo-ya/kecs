package kubernetes

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cmd"
)

type KindManager struct {
	provider *cluster.Provider
}

func NewKindManager() *KindManager {
	return &KindManager{
		provider: cluster.NewProvider(
			cluster.ProviderWithLogger(cmd.NewLogger()),
		),
	}
}

func (k *KindManager) CreateCluster(ctx context.Context, clusterName string) error {
	// If the cluster name already has the kecs- prefix, use it as is
	kecsClusterName := clusterName
	if !strings.HasPrefix(clusterName, "kecs-") {
		kecsClusterName = fmt.Sprintf("kecs-%s", clusterName)
	}

	exists, err := k.ClusterExists(kecsClusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if exists {
		return nil
	}

	kubeconfigPath := k.getKubeconfigPath(kecsClusterName)
	
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
	if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		if err := k.adjustKubeconfigForContainer(kubeconfigPath); err != nil {
			return fmt.Errorf("failed to adjust kubeconfig for container: %w", err)
		}
	}

	return nil
}

func (k *KindManager) DeleteCluster(ctx context.Context, clusterName string) error {
	// If the cluster name already has the kecs- prefix, use it as is
	kecsClusterName := clusterName
	if !strings.HasPrefix(clusterName, "kecs-") {
		kecsClusterName = fmt.Sprintf("kecs-%s", clusterName)
	}

	exists, err := k.ClusterExists(kecsClusterName)
	if err != nil {
		return fmt.Errorf("failed to check if cluster exists: %w", err)
	}

	if !exists {
		return nil
	}

	return k.provider.Delete(kecsClusterName, k.getKubeconfigPath(kecsClusterName))
}

func (k *KindManager) ClusterExists(kecsClusterName string) (bool, error) {
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

func (k *KindManager) GetKubeClient(clusterName string) (*kubernetes.Clientset, error) {
	// If the cluster name already has the kecs- prefix, use it as is
	kecsClusterName := clusterName
	if !strings.HasPrefix(clusterName, "kecs-") {
		kecsClusterName = fmt.Sprintf("kecs-%s", clusterName)
	}
	kubeconfigPath := k.getKubeconfigPath(kecsClusterName)

	// In container mode, ensure kubeconfig is adjusted
	if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		if _, err := os.Stat(kubeconfigPath); err == nil {
			// Kubeconfig exists, ensure it's adjusted for container access
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

func (k *KindManager) GetClusterNodes(clusterName string) ([]nodes.Node, error) {
	// If the cluster name already has the kecs- prefix, use it as is
	kecsClusterName := clusterName
	if !strings.HasPrefix(clusterName, "kecs-") {
		kecsClusterName = fmt.Sprintf("kecs-%s", clusterName)
	}
	return k.provider.ListNodes(kecsClusterName)
}

func (k *KindManager) getKubeconfigPath(kecsClusterName string) string {
	// In container mode, use a specific path that can be mounted
	if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		kubeconfigPath := os.Getenv("KECS_KUBECONFIG_PATH")
		if kubeconfigPath == "" {
			kubeconfigPath = "/kecs/kubeconfig"
		}
		// Ensure directory exists
		os.MkdirAll(kubeconfigPath, 0755)
		return filepath.Join(kubeconfigPath, fmt.Sprintf("%s.config", kecsClusterName))
	}
	
	// Default behavior for non-container mode
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", fmt.Sprintf("%s.config", kecsClusterName))
}

// GetKubeConfig returns a kubernetes config for the current context
func GetKubeConfig() (*rest.Config, error) {
	// Try to use in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// adjustKubeconfigForContainer modifies the kubeconfig to work from within a container
func (k *KindManager) adjustKubeconfigForContainer(kubeconfigPath string) error {
	// Read the kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Determine the host address based on the platform
	hostAddr := k.getHostAddress()

	// Update all cluster server addresses
	for _, cluster := range config.Clusters {
		// Replace localhost or 127.0.0.1 with the host address
		if strings.Contains(cluster.Server, "localhost") || strings.Contains(cluster.Server, "127.0.0.1") {
			if strings.Contains(cluster.Server, "https://localhost:") {
				cluster.Server = strings.Replace(cluster.Server, "localhost", hostAddr, 1)
			} else if strings.Contains(cluster.Server, "https://127.0.0.1:") {
				cluster.Server = strings.Replace(cluster.Server, "127.0.0.1", hostAddr, 1)
			}
			log.Printf("Adjusted cluster server address to: %s", cluster.Server)
		}
	}

	// Write the adjusted kubeconfig back
	if err := clientcmd.WriteToFile(*config, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to write adjusted kubeconfig: %w", err)
	}

	return nil
}

// getHostAddress returns the appropriate host address for accessing from within a container
func (k *KindManager) getHostAddress() string {
	// Check if a custom host address is specified
	if hostAddr := os.Getenv("KECS_HOST_DOCKER_INTERNAL"); hostAddr != "" {
		return hostAddr
	}

	// Default based on platform
	switch runtime.GOOS {
	case "darwin", "windows":
		// macOS and Windows use host.docker.internal
		return "host.docker.internal"
	default:
		// Linux typically uses the docker bridge gateway
		// This is a common default, but might need adjustment
		return "172.17.0.1"
	}
}
