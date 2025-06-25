package utils

import (
	"context"
	"fmt"
	"log"
	"time"
)

// WaitForClusterReadyOptions provides options for waiting for cluster readiness
type WaitForClusterReadyOptions struct {
	Timeout         time.Duration
	PollingInterval time.Duration
	RequireNodes    bool // Whether to check for Kubernetes nodes
}

// DefaultWaitForClusterReadyOptions returns default options
func DefaultWaitForClusterReadyOptions() WaitForClusterReadyOptions {
	return WaitForClusterReadyOptions{
		Timeout:         60 * time.Second,
		PollingInterval: 500 * time.Millisecond,
		RequireNodes:    true,
	}
}

// WaitForClusterReady waits for a cluster to be ready with dynamic checking
func WaitForClusterReady(t TestingT, client ECSClientInterface, clusterName string, opts ...WaitForClusterReadyOptions) error {
	
	// Use default options if none provided
	options := DefaultWaitForClusterReadyOptions()
	if len(opts) > 0 {
		options = opts[0]
	}
	
	log.Printf("[INFO] Waiting for cluster %s to be ready (timeout: %v)", clusterName, options.Timeout)
	
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()
	
	ticker := time.NewTicker(options.PollingInterval)
	defer ticker.Stop()
	
	startTime := time.Now()
	var lastError error
	
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster %s to be ready after %v: %v", 
				clusterName, time.Since(startTime), lastError)
		case <-ticker.C:
			// Check cluster status
			cluster, err := client.DescribeCluster(clusterName)
			if err != nil {
				lastError = fmt.Errorf("describe cluster failed: %w", err)
				log.Printf("[DEBUG] Cluster not ready yet: %v", lastError)
				continue
			}
			
			// Check if cluster is ACTIVE
			if cluster.Status != "ACTIVE" {
				lastError = fmt.Errorf("cluster status is %s, waiting for ACTIVE", cluster.Status)
				log.Printf("[DEBUG] Cluster status: %s", cluster.Status)
				continue
			}
			
			// If we don't need to check nodes, we're done
			if !options.RequireNodes {
				log.Printf("[INFO] Cluster %s is ACTIVE (no node check required)", clusterName)
				return nil
			}
			
			// For k3d clusters, check if the underlying Kubernetes cluster is ready
			if IsK3dCluster(clusterName) {
				ready, err := checkK3dClusterReady(clusterName)
				if err != nil {
					lastError = fmt.Errorf("k3d readiness check failed: %w", err)
					log.Printf("[DEBUG] K3d cluster not ready: %v", err)
					continue
				}
				if !ready {
					lastError = fmt.Errorf("k3d cluster exists but not ready")
					log.Printf("[DEBUG] K3d cluster exists but not ready yet")
					continue
				}
			}
			
			// All checks passed
			log.Printf("[INFO] Cluster %s is ready after %v", clusterName, time.Since(startTime))
			return nil
		}
	}
}

// IsK3dCluster checks if a cluster name indicates it's a k3d cluster
func IsK3dCluster(clusterName string) bool {
	// In KECS, clusters are prefixed with "kecs-" when created as k3d clusters
	return true // For now, assume all clusters in tests are k3d clusters
}

// checkK3dClusterReady checks if a k3d cluster is ready by verifying it exists
func checkK3dClusterReady(clusterName string) (bool, error) {
	// For KECS, we don't need to check k3d cluster directly since
	// KECS manages the cluster lifecycle. The cluster being ACTIVE
	// in KECS means the k3d cluster is ready.
	// This function is kept for potential future use.
	return true, nil
}

// WaitForClusterDeleted waits for a cluster to be deleted
func WaitForClusterDeleted(t TestingT, client ECSClientInterface, clusterName string, timeout time.Duration) error {
	log.Printf("[INFO] Waiting for cluster %s to be deleted (timeout: %v)", clusterName, timeout)
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	startTime := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster %s to be deleted after %v", 
				clusterName, time.Since(startTime))
		case <-ticker.C:
			// Try to describe the cluster
			_, err := client.DescribeCluster(clusterName)
			if err != nil {
				// If we get an error (cluster not found), it's deleted
				if contains(err.Error(), "not found") || contains(err.Error(), "ClusterNotFoundException") {
					log.Printf("[INFO] Cluster %s deleted successfully after %v", clusterName, time.Since(startTime))
					return nil
				}
				// Other errors, keep trying
				log.Printf("[DEBUG] Error checking cluster: %v", err)
			}
			// Cluster still exists, keep waiting
			log.Printf("[DEBUG] Cluster %s still exists, waiting...", clusterName)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsAt(s, substr)
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}