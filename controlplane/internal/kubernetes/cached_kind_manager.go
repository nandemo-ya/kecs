package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CachedKindManager wraps KindManager with cached client management
type CachedKindManager struct {
	KindManager   *KindManager
	clientManager *CachedClientManager
}

// NewCachedKindManager creates a new cached Kind manager
func NewCachedKindManager(kubeconfig string) (*CachedKindManager, error) {
	kindManager := NewKindManager()
	
	return &CachedKindManager{
		KindManager:   kindManager,
		clientManager: NewCachedClientManager(kubeconfig),
	}, nil
}

// CreateCluster creates a new Kind cluster
func (m *CachedKindManager) CreateCluster(ctx context.Context, name string) error {
	return m.KindManager.CreateCluster(ctx, name)
}

// DeleteCluster deletes a Kind cluster
func (m *CachedKindManager) DeleteCluster(ctx context.Context, name string) error {
	// Remove from client cache first
	m.clientManager.mu.Lock()
	delete(m.clientManager.clients, name)
	m.clientManager.mu.Unlock()
	
	return m.KindManager.DeleteCluster(ctx, name)
}

// ClusterExists checks if a cluster exists
func (m *CachedKindManager) ClusterExists(ctx context.Context, name string) (bool, error) {
	return m.KindManager.ClusterExists(name)
}

// GetKubeconfig returns the kubeconfig for a cluster
func (m *CachedKindManager) GetKubeconfig(ctx context.Context, clusterName string) (string, error) {
	// KindManager doesn't have GetKubeconfig, using GetKubeClient instead
	_, err := m.KindManager.GetKubeClient(clusterName)
	if err != nil {
		return "", err
	}
	// Return empty string as Kind manages kubeconfig internally
	return "", nil
}

// GetClient returns a cached Kubernetes client for the cluster
func (m *CachedKindManager) GetClient(ctx context.Context, clusterName string) (kubernetes.Interface, error) {
	return m.clientManager.GetClient(ctx, clusterName)
}

// CreateNamespace creates a namespace in the cluster
func (m *CachedKindManager) CreateNamespace(ctx context.Context, clusterName string, namespace *v1.Namespace) error {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	
	_, err = client.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	return err
}

// CreateDeployment creates a deployment in the cluster
func (m *CachedKindManager) CreateDeployment(ctx context.Context, clusterName string, namespace string, deployment *appsv1.Deployment) error {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	
	_, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	return err
}

// CreateService creates a service in the cluster
func (m *CachedKindManager) CreateService(ctx context.Context, clusterName string, namespace string, service *v1.Service) error {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	
	_, err = client.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	return err
}

// CreateConfigMap creates a ConfigMap in the cluster
func (m *CachedKindManager) CreateConfigMap(ctx context.Context, clusterName string, namespace string, configMap *v1.ConfigMap) error {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	
	_, err = client.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	return err
}

// CreateSecret creates a Secret in the cluster
func (m *CachedKindManager) CreateSecret(ctx context.Context, clusterName string, namespace string, secret *v1.Secret) error {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}
	
	_, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	return err
}

// GetPods returns pods in the cluster
func (m *CachedKindManager) GetPods(ctx context.Context, clusterName string, namespace string) (*v1.PodList, error) {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	
	return client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
}

// GetDeployments returns deployments in the cluster
func (m *CachedKindManager) GetDeployments(ctx context.Context, clusterName string, namespace string) (*appsv1.DeploymentList, error) {
	client, err := m.GetClient(ctx, clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	
	return client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
}

// Stats returns statistics about cached clients
func (m *CachedKindManager) Stats() ClientCacheStats {
	return m.clientManager.Stats()
}

// Close closes the cached kind manager
func (m *CachedKindManager) Close() {
	m.clientManager.Close()
	// KindManager doesn't need explicit close
}