package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CachedClientManager caches Kubernetes clients per cluster
type CachedClientManager struct {
	mu            sync.RWMutex
	clients       map[string]*cachedClient
	kubeconfig    string
	maxAge        time.Duration
	maxInactive   time.Duration
	cleanupTicker *time.Ticker
}

type cachedClient struct {
	client     kubernetes.Interface
	config     *rest.Config
	lastAccess time.Time
	createdAt  time.Time
}

// NewCachedClientManager creates a new cached client manager
func NewCachedClientManager(kubeconfig string) *CachedClientManager {
	cm := &CachedClientManager{
		clients:       make(map[string]*cachedClient),
		kubeconfig:    kubeconfig,
		maxAge:        30 * time.Minute, // Max client age
		maxInactive:   15 * time.Minute, // Max time without access
		cleanupTicker: time.NewTicker(5 * time.Minute),
	}

	// Start cleanup goroutine
	go cm.cleanupLoop()

	return cm
}

// GetClient returns a cached or new Kubernetes client for the cluster
func (cm *CachedClientManager) GetClient(ctx context.Context, clusterName string) (kubernetes.Interface, error) {
	cm.mu.RLock()
	cached, exists := cm.clients[clusterName]
	cm.mu.RUnlock()

	now := time.Now()

	// Check if cached client is still valid
	if exists {
		if now.Sub(cached.createdAt) < cm.maxAge {
			cm.mu.Lock()
			cached.lastAccess = now
			cm.mu.Unlock()
			return cached.client, nil
		}

		// Client is too old, remove it
		cm.mu.Lock()
		delete(cm.clients, clusterName)
		cm.mu.Unlock()
	}

	// Create new client
	client, config, err := cm.createClient(clusterName)
	if err != nil {
		return nil, err
	}

	// Cache the client
	cm.mu.Lock()
	cm.clients[clusterName] = &cachedClient{
		client:     client,
		config:     config,
		lastAccess: now,
		createdAt:  now,
	}
	cm.mu.Unlock()

	return client, nil
}

// GetConfig returns the REST config for a cluster
func (cm *CachedClientManager) GetConfig(ctx context.Context, clusterName string) (*rest.Config, error) {
	cm.mu.RLock()
	cached, exists := cm.clients[clusterName]
	cm.mu.RUnlock()

	if exists && time.Since(cached.createdAt) < cm.maxAge {
		cm.mu.Lock()
		cached.lastAccess = time.Now()
		cm.mu.Unlock()
		return cached.config, nil
	}

	// Get or create client (which also caches the config)
	_, err := cm.GetClient(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	// Retrieve the cached config
	cm.mu.RLock()
	cached = cm.clients[clusterName]
	cm.mu.RUnlock()

	if cached != nil {
		return cached.config, nil
	}

	return nil, fmt.Errorf("failed to get config for cluster %s", clusterName)
}

// createClient creates a new Kubernetes client
func (cm *CachedClientManager) createClient(clusterName string) (kubernetes.Interface, *rest.Config, error) {
	var config *rest.Config
	var err error

	if cm.kubeconfig != "" {
		// Use provided kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", cm.kubeconfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	} else {
		// Try in-cluster config first, then default kubeconfig
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			configOverrides := &clientcmd.ConfigOverrides{}
			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
			config, err = kubeConfig.ClientConfig()
			if err != nil {
				return nil, nil, fmt.Errorf("failed to build config: %w", err)
			}
		}
	}

	// Adjust config for better performance
	config.QPS = 100
	config.Burst = 200

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return clientset, config, nil
}

// cleanupLoop periodically removes inactive clients
func (cm *CachedClientManager) cleanupLoop() {
	for range cm.cleanupTicker.C {
		cm.cleanup()
	}
}

// cleanup removes expired or inactive clients
func (cm *CachedClientManager) cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for name, cached := range cm.clients {
		if now.Sub(cached.createdAt) > cm.maxAge || now.Sub(cached.lastAccess) > cm.maxInactive {
			toRemove = append(toRemove, name)
		}
	}

	for _, name := range toRemove {
		delete(cm.clients, name)
	}
}

// Stats returns cache statistics
func (cm *CachedClientManager) Stats() ClientCacheStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := ClientCacheStats{
		CachedClients: len(cm.clients),
		Clients:       make(map[string]ClientStats),
	}

	now := time.Now()
	for name, cached := range cm.clients {
		stats.Clients[name] = ClientStats{
			Age:          now.Sub(cached.createdAt),
			LastAccessed: now.Sub(cached.lastAccess),
		}
	}

	return stats
}

// Close stops the cleanup loop and clears all cached clients
func (cm *CachedClientManager) Close() {
	cm.cleanupTicker.Stop()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Clear all cached clients
	cm.clients = make(map[string]*cachedClient)
}

// ClientCacheStats contains statistics about cached clients
type ClientCacheStats struct {
	CachedClients int
	Clients       map[string]ClientStats
}

// ClientStats contains statistics about a single cached client
type ClientStats struct {
	Age          time.Duration
	LastAccessed time.Duration
}
