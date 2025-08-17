package sync

import (
	"context"
	stdsync "sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Type aliases to avoid any potential confusion
type StorageService = storage.Service
type StorageTask = storage.Task

// TaskUpdater interface for updating task status
type TaskUpdater interface {
	UpdateTaskStatus(ctx context.Context, taskARN string, pod *corev1.Pod) error
}

// BatchUpdater efficiently batches updates to the storage layer
type BatchUpdater struct {
	storage      storage.Storage
	taskUpdater  TaskUpdater
	kubeClient   kubernetes.Interface
	serviceCache map[string]*StorageService // key is service ARN
	taskCache    map[string]*StorageTask    // key is task ARN
	mu           stdsync.Mutex
	ticker       *time.Ticker
	stopCh       chan struct{}
	flushCh      chan struct{} // Channel to trigger immediate flush
	batchSize    int
	maxDelay     time.Duration
}

// NewBatchUpdater creates a new batch updater
func NewBatchUpdater(storage storage.Storage, batchSize int, maxDelay time.Duration) *BatchUpdater {
	return &BatchUpdater{
		storage:      storage,
		serviceCache: make(map[string]*StorageService),
		taskCache:    make(map[string]*StorageTask),
		batchSize:    batchSize,
		maxDelay:     maxDelay,
		stopCh:       make(chan struct{}),
		flushCh:      make(chan struct{}, 1),
	}
}

// SetTaskUpdater sets the task updater and kubernetes client for the batch updater
func (b *BatchUpdater) SetTaskUpdater(taskUpdater TaskUpdater, kubeClient kubernetes.Interface) {
	b.taskUpdater = taskUpdater
	b.kubeClient = kubeClient
}

// Start begins the batch update process
func (b *BatchUpdater) Start(ctx context.Context) {
	b.ticker = time.NewTicker(b.maxDelay)
	defer b.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			b.flush(ctx)
			return
		case <-b.stopCh:
			b.flush(ctx)
			return
		case <-b.ticker.C:
			b.flush(ctx)
		case <-b.flushCh:
			b.flush(ctx)
		}
	}
}

// Stop stops the batch updater and flushes pending updates
func (b *BatchUpdater) Stop(ctx context.Context) {
	close(b.stopCh)
	// Give it time to flush
	time.Sleep(100 * time.Millisecond)
}

// AddServiceUpdate adds a service update to the batch
func (b *BatchUpdater) AddServiceUpdate(service *StorageService) {
	if service == nil || service.ARN == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.serviceCache[service.ARN] = service

	// Trigger immediate flush if batch size reached
	if len(b.serviceCache) >= b.batchSize {
		select {
		case b.flushCh <- struct{}{}:
		default:
			// Channel is full, flush will happen soon anyway
		}
	}
}

// AddTaskUpdate adds a task update to the batch
func (b *BatchUpdater) AddTaskUpdate(task *StorageTask) {
	if task == nil || task.ARN == "" {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.taskCache[task.ARN] = task

	// Trigger immediate flush if batch size reached
	if len(b.taskCache) >= b.batchSize {
		select {
		case b.flushCh <- struct{}{}:
		default:
			// Channel is full, flush will happen soon anyway
		}
	}
}

// flush performs the actual batch update
func (b *BatchUpdater) flush(ctx context.Context) {
	b.mu.Lock()

	// Copy and clear the caches
	services := make([]*StorageService, 0, len(b.serviceCache))
	for _, svc := range b.serviceCache {
		services = append(services, svc)
	}
	b.serviceCache = make(map[string]*StorageService)

	tasks := make([]*StorageTask, 0, len(b.taskCache))
	for _, task := range b.taskCache {
		tasks = append(tasks, task)
	}
	b.taskCache = make(map[string]*StorageTask)

	b.mu.Unlock()

	// Update services
	if len(services) > 0 {
		logging.Info("Flushing service updates", "count", len(services))
		for _, service := range services {
			if err := b.updateService(ctx, service); err != nil {
				logging.Error("Failed to update service", "serviceName", service.ServiceName, "error", err)
			}
		}
	}

	// Update tasks
	if len(tasks) > 0 {
		logging.Info("Flushing task updates", "count", len(tasks))
		for _, task := range tasks {
			if err := b.updateTask(ctx, task); err != nil {
				logging.Error("Failed to update task", "taskARN", task.ARN, "error", err)
			}
		}
	}
}

// updateService updates a single service in storage
func (b *BatchUpdater) updateService(ctx context.Context, service *StorageService) error {
	logging.Info("Updating service", "serviceName", service.ServiceName, "clusterARN", service.ClusterARN)

	// Check if service exists
	existingService, err := b.storage.ServiceStore().Get(ctx, service.ClusterARN, service.ServiceName)
	if err != nil {
		// Service doesn't exist, create it
		logging.Info("Service not found, creating new", "serviceName", service.ServiceName)
		return b.storage.ServiceStore().Create(ctx, service)
	}

	// Merge with existing service to preserve fields we don't sync
	mergedService := b.mergeServices(existingService, service)
	logging.Info("Updating existing service with new state",
		"serviceName", service.ServiceName, "runningCount", mergedService.RunningCount, "pendingCount", mergedService.PendingCount)
	return b.storage.ServiceStore().Update(ctx, mergedService)
}

// mergeServices merges the synced service data with existing service data
func (b *BatchUpdater) mergeServices(existing, updated *StorageService) *StorageService {
	// Start with existing service
	merged := *existing

	// Update fields that are synced from Kubernetes
	merged.Status = updated.Status
	merged.DesiredCount = updated.DesiredCount
	merged.RunningCount = updated.RunningCount
	merged.PendingCount = updated.PendingCount
	merged.UpdatedAt = updated.UpdatedAt

	// Update deployment configuration if provided
	if updated.DeploymentConfiguration != "" {
		merged.DeploymentConfiguration = updated.DeploymentConfiguration
	}

	return &merged
}

// updateTask updates a single task in storage
func (b *BatchUpdater) updateTask(ctx context.Context, task *StorageTask) error {
	logging.Info("Updating task", "taskARN", task.ARN, "clusterARN", task.ClusterARN)

	// Use CreateOrUpdate to avoid duplicate key errors
	if err := b.storage.TaskStore().CreateOrUpdate(ctx, task); err != nil {
		logging.Error("Failed to create or update task", "taskARN", task.ARN, "error", err)
		return err
	}

	logging.Debug("Successfully created or updated task", "taskARN", task.ARN, "lastStatus", task.LastStatus)

	// If TaskUpdater is available and task is RUNNING, get the pod and call UpdateTaskStatus
	// This will trigger Service Discovery registration
	if b.taskUpdater != nil && b.kubeClient != nil {
		if task.LastStatus == "RUNNING" && task.PodName != "" && task.Namespace != "" {
			// Get the pod from Kubernetes
			pod, err := b.kubeClient.CoreV1().Pods(task.Namespace).Get(ctx, task.PodName, metav1.GetOptions{})
			if err == nil && pod != nil {
				// Call UpdateTaskStatus which will trigger Service Discovery registration
				logging.Info("Triggering UpdateTaskStatus for Service Discovery registration",
					"task", task.ARN,
					"pod", task.PodName,
					"status", task.LastStatus)
				if err := b.taskUpdater.UpdateTaskStatus(ctx, task.ARN, pod); err != nil {
					logging.Warn("Failed to update task status via TaskUpdater",
						"task", task.ARN,
						"error", err)
				}
			}
		}
	}

	return nil
}

// Flush forces an immediate flush of pending updates
func (b *BatchUpdater) Flush() {
	select {
	case b.flushCh <- struct{}{}:
	default:
		// Channel is full, flush will happen soon anyway
	}
}
