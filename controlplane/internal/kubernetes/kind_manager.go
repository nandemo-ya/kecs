package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

	err = k.provider.Create(
		kecsClusterName,
		cluster.CreateWithNodeImage("kindest/node:v1.31.4"),
		cluster.CreateWithWaitForReady(0),
		cluster.CreateWithKubeconfigPath(k.getKubeconfigPath(kecsClusterName)),
		cluster.CreateWithDisplayUsage(false),
		cluster.CreateWithDisplaySalutation(false),
	)

	if err != nil {
		return fmt.Errorf("failed to create kind cluster: %w", err)
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
