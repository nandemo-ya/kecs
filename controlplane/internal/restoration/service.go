package restoration

import (
	"context"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Service handles restoration of ECS resources from DuckDB to Kubernetes
type Service struct {
	storage           storage.Storage
	taskManager       *kubernetes.TaskManager
	serviceManager    *kubernetes.ServiceManager
	localStackManager localstack.Manager
}

// NewService creates a new restoration service
func NewService(
	storage storage.Storage,
	taskManager *kubernetes.TaskManager,
	serviceManager *kubernetes.ServiceManager,
	localStackManager localstack.Manager,
) *Service {
	return &Service{
		storage:           storage,
		taskManager:       taskManager,
		serviceManager:    serviceManager,
		localStackManager: localStackManager,
	}
}

// RestoreAll restores all ECS resources from DuckDB
func (s *Service) RestoreAll(ctx context.Context) error {
	logging.Info("Starting ECS resource restoration from DuckDB")

	// Check if there's any data to restore
	clusters, err := s.storage.ClusterStore().List(ctx)
	if err != nil {
		logging.Warn("Failed to check for existing clusters", "error", err)
		return nil // Don't fail startup if no data exists
	}

	if len(clusters) == 0 {
		logging.Info("No clusters found in DuckDB, skipping restoration")
		return nil
	}

	// 1. Log clusters found
	if err := s.logClusters(ctx, clusters); err != nil {
		logging.Error("Failed to log clusters", "error", err)
	}

	// 2. Task definitions are already persisted in DuckDB
	logging.Info("Task definitions are already persisted in DuckDB, no restoration needed")

	// 3. Restore services (which will also recreate their tasks)
	if err := s.restoreServices(ctx); err != nil {
		logging.Error("Failed to restore services", "error", err)
	}

	// 4. Restore standalone tasks (not managed by services)
	if err := s.restoreStandaloneTasks(ctx); err != nil {
		logging.Error("Failed to restore standalone tasks", "error", err)
	}

	logging.Info("ECS resource restoration completed")
	return nil
}

// logClusters logs all clusters found in DuckDB
func (s *Service) logClusters(ctx context.Context, clusters []*storage.Cluster) error {
	logging.Info("Found clusters in DuckDB", "count", len(clusters))

	for _, cluster := range clusters {
		logging.Info("Found cluster in DuckDB",
			"clusterName", cluster.Name,
			"clusterArn", cluster.ARN,
			"status", cluster.Status,
			"activeServices", cluster.ActiveServicesCount,
			"runningTasks", cluster.RunningTasksCount)
	}

	return nil
}

// restoreServices restores all services found in DuckDB
func (s *Service) restoreServices(ctx context.Context) error {
	// List all services across all clusters
	services, _, err := s.storage.ServiceStore().List(ctx, "", "", "", 1000, "")
	if err != nil {
		return fmt.Errorf("failed to list services from storage: %w", err)
	}

	activeCount := 0
	restoredCount := 0
	for _, service := range services {
		if service.Status == "ACTIVE" {
			activeCount++
			logging.Info("Restoring service from DuckDB",
				"serviceName", service.ServiceName,
				"clusterArn", extractClusterName(service.ClusterARN),
				"desiredCount", service.DesiredCount,
				"taskDefinition", service.TaskDefinitionARN)

			// Check if the service deployment already exists in Kubernetes
			if s.serviceManager != nil {
				if err := s.serviceManager.RestoreService(ctx, service); err != nil {
					logging.Error("Failed to restore service",
						"serviceName", service.ServiceName,
						"error", err)
				} else {
					restoredCount++
				}
			}
		}
	}

	logging.Info("Service restoration completed",
		"totalCount", len(services),
		"activeCount", activeCount,
		"restoredCount", restoredCount)

	return nil
}

// restoreStandaloneTasks restores standalone tasks (not managed by services)
func (s *Service) restoreStandaloneTasks(ctx context.Context) error {
	// List all tasks across all clusters
	tasks, err := s.storage.TaskStore().List(ctx, "", storage.TaskFilters{
		MaxResults: 1000,
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks from storage: %w", err)
	}

	standaloneCount := 0
	restoredCount := 0
	for _, task := range tasks {
		// Skip tasks that are managed by services (have a service group)
		if task.Group != "" && strings.HasPrefix(task.Group, "service:") {
			continue
		}

		// Only restore tasks that were running or pending
		if task.LastStatus == "RUNNING" || task.LastStatus == "PENDING" {
			standaloneCount++
			logging.Info("Restoring standalone task from DuckDB",
				"taskArn", task.ARN,
				"taskDefinition", task.TaskDefinitionARN,
				"lastStatus", task.LastStatus)

			// Restore the task using TaskManager
			if s.taskManager != nil {
				if err := s.taskManager.RestoreTask(ctx, task); err != nil {
					logging.Error("Failed to restore standalone task",
						"taskArn", task.ARN,
						"error", err)
				} else {
					restoredCount++
				}
			}
		}
	}

	logging.Info("Standalone task restoration completed",
		"standaloneCount", standaloneCount,
		"restoredCount", restoredCount)

	return nil
}

// extractClusterName extracts the cluster name from a cluster ARN
func extractClusterName(clusterARN string) string {
	if clusterARN == "" {
		return "default"
	}

	// ARN format: arn:aws:ecs:region:account:cluster/cluster-name
	parts := strings.Split(clusterARN, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}

	return "default"
}
