package utils

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// SharedClusterManager manages shared clusters for tests
type SharedClusterManager struct {
	client         ECSClientInterface
	clusters       map[string]*SharedCluster
	mu             sync.RWMutex
	cleanupOnClose bool
}

// SharedCluster represents a shared cluster that can be used by multiple tests
type SharedCluster struct {
	Name      string
	InUse     bool
	CreatedAt time.Time
	mu        sync.Mutex
}

// NewSharedClusterManager creates a new shared cluster manager
func NewSharedClusterManager(client ECSClientInterface, cleanupOnClose bool) *SharedClusterManager {
	return &SharedClusterManager{
		client:         client,
		clusters:       make(map[string]*SharedCluster),
		cleanupOnClose: cleanupOnClose,
	}
}

// GetOrCreateCluster gets an existing cluster or creates a new one
func (m *SharedClusterManager) GetOrCreateCluster(prefix string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find an available cluster with the given prefix
	for name, cluster := range m.clusters {
		if cluster.acquireIfAvailable() {
			log.Printf("[SharedCluster] Reusing existing cluster: %s", name)
			return name, nil
		}
	}

	// No available cluster found, create a new one
	clusterName := GenerateTestName(prefix)
	log.Printf("[SharedCluster] Creating new shared cluster: %s", clusterName)
	
	// Create the cluster
	err := m.client.CreateCluster(clusterName)
	if err != nil {
		return "", fmt.Errorf("failed to create shared cluster: %w", err)
	}

	// Wait for cluster to be ready
	opts := WaitForClusterReadyOptions{
		Timeout:         30 * time.Second,
		PollingInterval: 500 * time.Millisecond,
		RequireNodes:    false,
	}
	if err := WaitForClusterReady(nil, m.client, clusterName, opts); err != nil {
		// Try to clean up the failed cluster
		_ = m.client.DeleteCluster(clusterName)
		return "", fmt.Errorf("cluster not ready: %w", err)
	}

	// Register the cluster
	cluster := &SharedCluster{
		Name:      clusterName,
		InUse:     true,
		CreatedAt: time.Now(),
	}
	m.clusters[clusterName] = cluster

	return clusterName, nil
}

// ReleaseCluster marks a cluster as available for reuse
func (m *SharedClusterManager) ReleaseCluster(clusterName string) {
	m.mu.RLock()
	cluster, exists := m.clusters[clusterName]
	m.mu.RUnlock()

	if exists {
		cluster.release()
		log.Printf("[SharedCluster] Released cluster: %s", clusterName)
	}
}

// CleanupAll deletes all managed clusters
func (m *SharedClusterManager) CleanupAll() {
	if !m.cleanupOnClose {
		log.Printf("[SharedCluster] Skipping cleanup (cleanupOnClose=false)")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[SharedCluster] Cleaning up %d shared clusters", len(m.clusters))
	
	for name := range m.clusters {
		log.Printf("[SharedCluster] Deleting cluster: %s", name)
		if err := m.client.DeleteCluster(name); err != nil {
			log.Printf("[SharedCluster] Warning: Failed to delete cluster %s: %v", name, err)
		}
		
		// Wait for deletion
		if err := WaitForClusterDeleted(nil, m.client, name, 30*time.Second); err != nil {
			log.Printf("[SharedCluster] Warning: Cluster %s may not be fully deleted: %v", name, err)
		}
	}

	// Clear the map
	m.clusters = make(map[string]*SharedCluster)
}

// GetClusterCount returns the number of managed clusters
func (m *SharedClusterManager) GetClusterCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clusters)
}

// acquireIfAvailable attempts to acquire the cluster for use
func (c *SharedCluster) acquireIfAvailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if !c.InUse {
		c.InUse = true
		return true
	}
	return false
}

// release marks the cluster as available
func (c *SharedCluster) release() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.InUse = false
}

// SharedClusterTest provides helper methods for tests using shared clusters
type SharedClusterTest struct {
	Manager     *SharedClusterManager
	ClusterName string
}

// SetupSharedCluster acquires a shared cluster for the test
func (s *SharedClusterTest) SetupSharedCluster(prefix string) error {
	clusterName, err := s.Manager.GetOrCreateCluster(prefix)
	if err != nil {
		return err
	}
	s.ClusterName = clusterName
	return nil
}

// TeardownSharedCluster releases the shared cluster
func (s *SharedClusterTest) TeardownSharedCluster() {
	if s.ClusterName != "" {
		s.Manager.ReleaseCluster(s.ClusterName)
		s.ClusterName = ""
	}
}