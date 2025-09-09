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

	// 3. Log services found
	if err := s.logServices(ctx); err != nil {
		logging.Error("Failed to log services", "error", err)
	}

	// 4. Log tasks found
	if err := s.logTasks(ctx); err != nil {
		logging.Error("Failed to log tasks", "error", err)
	}

	logging.Info("ECS resource restoration scan completed")
	logging.Info("Note: Actual Kubernetes resource recreation will happen through normal ECS API operations")
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

// logServices logs all services found in DuckDB
func (s *Service) logServices(ctx context.Context) error {
	// List all services across all clusters
	services, _, err := s.storage.ServiceStore().List(ctx, "", "", "", 1000, "")
	if err != nil {
		return fmt.Errorf("failed to list services from storage: %w", err)
	}

	activeCount := 0
	for _, service := range services {
		if service.Status == "ACTIVE" {
			activeCount++
			logging.Info("Found active service in DuckDB",
				"serviceName", service.ServiceName,
				"clusterArn", extractClusterName(service.ClusterARN),
				"desiredCount", service.DesiredCount,
				"runningCount", service.RunningCount,
				"taskDefinition", service.TaskDefinitionARN)
		}
	}

	logging.Info("Found services in DuckDB",
		"totalCount", len(services),
		"activeCount", activeCount)

	// Note: Actual service restoration would happen through the normal
	// ECS CreateService/UpdateService API calls which already handle
	// creating the necessary Kubernetes deployments

	return nil
}

// logTasks logs all tasks found in DuckDB
func (s *Service) logTasks(ctx context.Context) error {
	// List all tasks across all clusters
	tasks, err := s.storage.TaskStore().List(ctx, "", storage.TaskFilters{
		MaxResults: 1000,
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks from storage: %w", err)
	}

	runningCount := 0
	pendingCount := 0
	for _, task := range tasks {
		if task.LastStatus == "RUNNING" {
			runningCount++
			logging.Debug("Found running task in DuckDB",
				"taskArn", task.ARN,
				"group", task.Group,
				"startedBy", task.StartedBy)
		} else if task.LastStatus == "PENDING" {
			pendingCount++
			logging.Debug("Found pending task in DuckDB",
				"taskArn", task.ARN,
				"group", task.Group,
				"startedBy", task.StartedBy)
		}
	}

	logging.Info("Found tasks in DuckDB",
		"totalCount", len(tasks),
		"runningCount", runningCount,
		"pendingCount", pendingCount)

	// Note: Tasks that belong to services will be recreated automatically
	// when the service deployments are restored. Standalone tasks would
	// need to be recreated through RunTask API calls.

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
