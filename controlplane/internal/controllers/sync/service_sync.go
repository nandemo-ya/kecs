package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controllers/sync/mappers"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

// syncService syncs a deployment to ECS service state
func (c *SyncController) syncService(ctx context.Context, key string) error {
	klog.Infof("syncService called with key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return fmt.Errorf("invalid resource key: %s", key)
	}

	// Check if this is an ECS-managed deployment
	if !strings.HasPrefix(name, "ecs-service-") {
		klog.Infof("Ignoring non-ECS deployment: %s", name)
		return nil
	}

	// Get the deployment
	klog.Infof("Getting deployment %s/%s", namespace, name)
	deployment, err := c.deploymentLister.Deployments(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			// Deployment was deleted, update service to INACTIVE
			klog.Infof("Deployment %s/%s not found, handling deletion", namespace, name)
			return c.handleDeletedDeployment(ctx, namespace, name)
		}
		return fmt.Errorf("error fetching deployment: %v", err)
	}
	klog.Infof("Got deployment %s/%s successfully", namespace, name)

	// Map deployment to service
	mapper := mappers.NewServiceStateMapper(c.accountID, c.region)
	serviceName := mapper.ExtractServiceNameFromDeployment(name)
	clusterName, region := mapper.ExtractClusterInfoFromNamespace(namespace)

	klog.Infof("Extracted service info - service: %s, cluster: %s, region: %s", serviceName, clusterName, region)

	if clusterName == "" || region == "" {
		klog.Infof("Could not extract cluster info from namespace: %s", namespace)
		return nil
	}

	// Get existing service from storage
	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, c.accountID, clusterName)
	existingService, err := c.storage.ServiceStore().Get(ctx, clusterARN, serviceName)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("error getting service from storage: %v", err)
	}

	// Map deployment to service
	service := mapper.MapDeploymentToService(deployment, existingService)
	if service == nil {
		return fmt.Errorf("failed to map deployment to service")
	}

	// Update service ARN if not set
	if service.ARN == "" {
		service.ARN = fmt.Sprintf("arn:aws:ecs:%s:%s:service/%s/%s",
			region, c.accountID, clusterName, serviceName)
	}

	// Add to batch updater for efficient storage update
	c.batchUpdater.AddServiceUpdate(service)
	klog.Infof("Queued service update %s in cluster %s", serviceName, clusterName)

	// Log the sync result
	klog.Infof("Successfully synced service %s: status=%s, desired=%d, running=%d, pending=%d",
		serviceName, service.Status, service.DesiredCount, service.RunningCount, service.PendingCount)

	return nil
}

// handleDeletedDeployment handles the case when a deployment is deleted
func (c *SyncController) handleDeletedDeployment(ctx context.Context, namespace, deploymentName string) error {
	mapper := mappers.NewServiceStateMapper(c.accountID, c.region)
	serviceName := mapper.ExtractServiceNameFromDeployment(deploymentName)
	clusterName, region := mapper.ExtractClusterInfoFromNamespace(namespace)

	if clusterName == "" || region == "" {
		return nil
	}

	clusterARN := fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/%s", region, c.accountID, clusterName)
	service, err := c.storage.ServiceStore().Get(ctx, clusterARN, serviceName)
	if err != nil {
		if isNotFound(err) {
			// Service doesn't exist, nothing to do
			return nil
		}
		return fmt.Errorf("error getting service: %v", err)
	}

	// Update service to INACTIVE
	service.Status = "INACTIVE"
	service.DesiredCount = 0
	service.RunningCount = 0
	service.PendingCount = 0
	service.UpdatedAt = time.Now()

	// Add to batch updater
	c.batchUpdater.AddServiceUpdate(service)

	klog.V(2).Infof("Marked service %s as INACTIVE due to deployment deletion", serviceName)
	return nil
}

// isNotFound checks if an error indicates a resource was not found
func isNotFound(err error) bool {
	// Check for storage-specific not found error
	// This should be implemented based on your storage interface
	return strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "does not exist")
}
