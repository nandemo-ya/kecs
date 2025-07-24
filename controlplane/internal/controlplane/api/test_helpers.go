package api

import (
	"context"
	"time"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
)

// MockClusterManager is a mock implementation of ClusterManager for testing
type MockClusterManager struct {
	DeletedClusters []string
	ClusterMap      map[string]bool
}

func NewMockClusterManager() *MockClusterManager {
	return &MockClusterManager{
		DeletedClusters: []string{},
		ClusterMap:      make(map[string]bool),
	}
}

func (m *MockClusterManager) CreateCluster(ctx context.Context, clusterName string) error {
	m.ClusterMap[clusterName] = true
	return nil
}

func (m *MockClusterManager) DeleteCluster(ctx context.Context, clusterName string) error {
	m.DeletedClusters = append(m.DeletedClusters, clusterName)
	delete(m.ClusterMap, clusterName)
	return nil
}

func (m *MockClusterManager) ClusterExists(ctx context.Context, clusterName string) (bool, error) {
	return m.ClusterMap[clusterName], nil
}

func (m *MockClusterManager) GetKubeClient(clusterName string) (k8s.Interface, error) {
	return nil, nil
}

func (m *MockClusterManager) GetKubeConfig(clusterName string) (*rest.Config, error) {
	return nil, nil
}

func (m *MockClusterManager) WaitForClusterReady(clusterName string, timeout time.Duration) error {
	return nil
}

func (m *MockClusterManager) GetKubeconfigPath(clusterName string) string {
	return ""
}

func (m *MockClusterManager) GetClusterInfo(ctx context.Context, clusterName string) (*kubernetes.ClusterInfo, error) {
	return nil, nil
}

func (m *MockClusterManager) GetTraefikPort(clusterName string) (int, bool) {
	return 0, false
}

func (m *MockClusterManager) StopCluster(ctx context.Context, clusterName string) error {
	return nil
}

func (m *MockClusterManager) StartCluster(ctx context.Context, clusterName string) error {
	return nil
}

func (m *MockClusterManager) ListClusters(ctx context.Context) ([]string, error) {
	return []string{}, nil
}
