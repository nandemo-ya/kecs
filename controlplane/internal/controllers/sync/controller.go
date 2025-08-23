package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	k8sclient "k8s.io/client-go/kubernetes"
	appslistersv1 "k8s.io/client-go/listers/apps/v1"
	corelistersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// SyncController manages the synchronization of Kubernetes resources to ECS state
type SyncController struct {
	kubeClient       k8sclient.Interface
	storage          storage.Storage
	taskUpdater      TaskUpdater
	deploymentLister appslistersv1.DeploymentLister
	replicaSetLister appslistersv1.ReplicaSetLister
	podLister        corelistersv1.PodLister
	eventLister      corelistersv1.EventLister

	// Informer synced flags
	deploymentsSynced cache.InformerSynced
	replicaSetsSynced cache.InformerSynced
	podsSynced        cache.InformerSynced
	eventsSynced      cache.InformerSynced

	// Work queues for different resource types
	deploymentQueue workqueue.RateLimitingInterface
	podQueue        workqueue.RateLimitingInterface

	// Batch updater for efficient storage updates
	batchUpdater *BatchUpdater

	// Configuration
	workers      int
	resyncPeriod time.Duration
	accountID    string
	region       string
}

// NewSyncController creates a new synchronization controller
func NewSyncController(
	kubeClient k8sclient.Interface,
	storage storage.Storage,
	deploymentInformer appsinformers.DeploymentInformer,
	replicaSetInformer appsinformers.ReplicaSetInformer,
	podInformer coreinformers.PodInformer,
	eventInformer coreinformers.EventInformer,
	workers int,
	resyncPeriod time.Duration,
) *SyncController {
	// Get configuration
	cfg := config.GetConfig()
	accountID := cfg.AWS.AccountID
	if accountID == "" {
		accountID = "000000000000" // Default
	}
	region := cfg.AWS.DefaultRegion
	if region == "" {
		region = "us-east-1" // Default
	}
	controller := &SyncController{
		kubeClient:       kubeClient,
		storage:          storage,
		deploymentLister: deploymentInformer.Lister(),
		replicaSetLister: replicaSetInformer.Lister(),
		podLister:        podInformer.Lister(),
		eventLister:      eventInformer.Lister(),

		deploymentsSynced: deploymentInformer.Informer().HasSynced,
		replicaSetsSynced: replicaSetInformer.Informer().HasSynced,
		podsSynced:        podInformer.Informer().HasSynced,
		eventsSynced:      eventInformer.Informer().HasSynced,

		deploymentQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"deployments",
		),
		podQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"pods",
		),

		workers:      workers,
		resyncPeriod: resyncPeriod,
		accountID:    accountID,
		region:       region,
	}

	// Create batch updater with reasonable defaults
	controller.batchUpdater = NewBatchUpdater(storage, 100, 2*time.Second)

	// Set up event handlers
	deploymentInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.handleDeploymentAdd,
		UpdateFunc: controller.handleDeploymentUpdate,
		DeleteFunc: controller.handleDeploymentDelete,
	})

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.handlePodAdd,
		UpdateFunc: controller.handlePodUpdate,
		DeleteFunc: controller.handlePodDelete,
	})

	return controller
}

// SetTaskUpdater sets the task updater for the controller
func (c *SyncController) SetTaskUpdater(taskUpdater TaskUpdater, kubeClient k8sclient.Interface) {
	c.taskUpdater = taskUpdater
	if c.batchUpdater != nil && taskUpdater != nil {
		c.batchUpdater.SetTaskUpdater(taskUpdater, kubeClient)
	}
}

// processExistingPods processes existing pods when the controller starts
func (c *SyncController) processExistingPods(ctx context.Context) {
	// Wait a bit for controller to be fully initialized
	time.Sleep(3 * time.Second)

	logging.Info("Processing existing pods")

	// List all pods
	pods, err := c.podLister.List(labels.Everything())
	if err != nil {
		logging.Error("Failed to list existing pods", "error", err)
		return
	}

	count := 0
	for _, pod := range pods {
		if !isECSManagedPod(pod) {
			continue
		}

		// Add pod to queue for processing
		key, err := cache.MetaNamespaceKeyFunc(pod)
		if err != nil {
			logging.Warn("Failed to get key for pod", "pod", pod.Name, "error", err)
			continue
		}

		logging.Debug("Queueing existing pod for processing", "pod", pod.Name, "namespace", pod.Namespace)
		c.podQueue.Add(key)
		count++
	}

	if count > 0 {
		logging.Info("Queued existing pods for processing", "count", count)
	}
}

// Run starts the controller
func (c *SyncController) Run(ctx context.Context) error {
	defer runtime.HandleCrash()
	defer c.deploymentQueue.ShutDown()
	defer c.podQueue.ShutDown()

	logging.Info("Starting sync controller")

	// Start batch updater
	go c.batchUpdater.Start(ctx)
	defer c.batchUpdater.Stop(ctx)

	// Wait for informer caches to sync
	logging.Info("Waiting for informer caches to sync")

	// Check cache sync status periodically
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				logging.Debug("Cache sync status",
					"deployments", c.deploymentsSynced(),
					"replicaSets", c.replicaSetsSynced(),
					"pods", c.podsSynced(),
					"events", c.eventsSynced())
			case <-ctx.Done():
				return
			}
		}
	}()

	if !cache.WaitForCacheSync(ctx.Done(),
		c.deploymentsSynced,
		c.replicaSetsSynced,
		c.podsSynced,
		c.eventsSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logging.Info("Starting workers")
	// Start workers
	for i := 0; i < c.workers; i++ {
		go wait.UntilWithContext(ctx, c.runDeploymentWorker, time.Second)
		go wait.UntilWithContext(ctx, c.runPodWorker, time.Second)
	}

	// Process existing pods after controller starts
	go c.processExistingPods(ctx)

	logging.Info("Sync controller started")
	<-ctx.Done()
	logging.Info("Shutting down sync controller")

	return nil
}

// runDeploymentWorker processes items from the deployment work queue
func (c *SyncController) runDeploymentWorker(ctx context.Context) {
	for c.processNextDeployment(ctx) {
	}
}

// runPodWorker processes items from the pod work queue
func (c *SyncController) runPodWorker(ctx context.Context) {
	for c.processNextPod(ctx) {
	}
}

// processNextDeployment processes the next item from the deployment queue
func (c *SyncController) processNextDeployment(ctx context.Context) bool {
	key, quit := c.deploymentQueue.Get()
	if quit {
		return false
	}
	defer c.deploymentQueue.Done(key)

	logging.Debug("Processing deployment from queue", "key", key)

	err := c.syncDeployment(ctx, key.(string))
	if err == nil {
		c.deploymentQueue.Forget(key)
		return true
	}

	runtime.HandleError(fmt.Errorf("error syncing deployment '%s': %v", key, err))

	// Re-queue with rate limiting
	if c.deploymentQueue.NumRequeues(key) < 5 {
		logging.Debug("Retrying deployment", "key", key)
		c.deploymentQueue.AddRateLimited(key)
		return true
	}

	c.deploymentQueue.Forget(key)
	logging.Warn("Dropping deployment out of the queue after 5 retries", "key", key)
	return true
}

// processNextPod processes the next item from the pod queue
func (c *SyncController) processNextPod(ctx context.Context) bool {
	key, quit := c.podQueue.Get()
	if quit {
		return false
	}
	defer c.podQueue.Done(key)

	logging.Debug("Processing pod from queue", "key", key)

	err := c.syncTask(ctx, key.(string))
	if err == nil {
		c.podQueue.Forget(key)
		return true
	}

	runtime.HandleError(fmt.Errorf("error syncing pod '%s': %v", key, err))

	// Re-queue with rate limiting
	if c.podQueue.NumRequeues(key) < 5 {
		logging.Debug("Retrying pod", "key", key)
		c.podQueue.AddRateLimited(key)
		return true
	}

	c.podQueue.Forget(key)
	logging.Warn("Dropping pod out of the queue after 5 retries", "key", key)
	return true
}

// syncDeployment syncs a deployment to ECS service state
func (c *SyncController) syncDeployment(ctx context.Context, key string) error {
	logging.Debug("Syncing deployment", "key", key)
	return c.syncService(ctx, key)
}

// Deployment event handlers
func (c *SyncController) handleDeploymentAdd(obj interface{}) {
	deployment := obj.(*appsv1.Deployment)
	// Add debug logging
	logging.Debug("Deployment add event", "namespace", deployment.Namespace, "name", deployment.Name, "managed", isECSManagedDeployment(deployment))
	if !isECSManagedDeployment(deployment) {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(deployment)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	logging.Debug("ECS deployment added", "name", deployment.Name)
	c.deploymentQueue.Add(key)
}

func (c *SyncController) handleDeploymentUpdate(oldObj, newObj interface{}) {
	oldDep := oldObj.(*appsv1.Deployment)
	newDep := newObj.(*appsv1.Deployment)

	// Add debug logging
	logging.Debug("Deployment update event", "namespace", newDep.Namespace, "name", newDep.Name, "managed", isECSManagedDeployment(newDep))

	if !isECSManagedDeployment(newDep) {
		return
	}

	// Only sync if status changed or scaling occurred
	if oldDep.Status.Replicas != newDep.Status.Replicas ||
		oldDep.Status.ReadyReplicas != newDep.Status.ReadyReplicas ||
		oldDep.Status.UpdatedReplicas != newDep.Status.UpdatedReplicas ||
		oldDep.Status.AvailableReplicas != newDep.Status.AvailableReplicas ||
		hasDeploymentConditionChanged(oldDep, newDep) {
		key, err := cache.MetaNamespaceKeyFunc(newDep)
		if err != nil {
			runtime.HandleError(err)
			return
		}
		logging.Debug("ECS deployment updated",
			"name", newDep.Name, "readyReplicas", newDep.Status.ReadyReplicas, "replicas", newDep.Status.Replicas)
		c.deploymentQueue.Add(key)
	}
}

func (c *SyncController) handleDeploymentDelete(obj interface{}) {
	deployment := obj.(*appsv1.Deployment)
	if !isECSManagedDeployment(deployment) {
		return
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	logging.Debug("ECS deployment deleted", "name", deployment.Name)
	c.deploymentQueue.Add(key)
}

// Pod event handlers
func (c *SyncController) handlePodAdd(obj interface{}) {
	pod := obj.(*corev1.Pod)
	logging.Debug("Pod add event", "namespace", pod.Namespace, "name", pod.Name, "managed", isECSManagedPod(pod))
	if !isECSManagedPod(pod) {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(pod)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	logging.Debug("ECS pod added", "name", pod.Name)
	c.podQueue.Add(key)
}

func (c *SyncController) handlePodUpdate(oldObj, newObj interface{}) {
	oldPod := oldObj.(*corev1.Pod)
	newPod := newObj.(*corev1.Pod)

	logging.Debug("Pod update event", "namespace", newPod.Namespace, "name", newPod.Name, "managed", isECSManagedPod(newPod))

	if !isECSManagedPod(newPod) {
		return
	}

	// Only sync if status changed
	if oldPod.Status.Phase != newPod.Status.Phase ||
		len(oldPod.Status.ContainerStatuses) != len(newPod.Status.ContainerStatuses) {
		logging.Debug("Pod status changed", "name", newPod.Name, "oldPhase", oldPod.Status.Phase, "newPhase", newPod.Status.Phase)
		key, err := cache.MetaNamespaceKeyFunc(newPod)
		if err != nil {
			runtime.HandleError(err)
			return
		}
		c.podQueue.Add(key)
	}
}

func (c *SyncController) handlePodDelete(obj interface{}) {
	pod := obj.(*corev1.Pod)
	if !isECSManagedPod(pod) {
		return
	}
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}

	// Log pod deletion with task ID for debugging
	taskID := pod.Labels["kecs.dev/task-id"]
	logging.Info("Pod deletion event detected",
		"pod", pod.Name,
		"namespace", pod.Namespace,
		"taskID", taskID,
		"labels", pod.Labels)

	c.podQueue.Add(key)
}

// isECSManagedPod checks if a pod is managed by KECS
func isECSManagedPod(pod *corev1.Pod) bool {
	// Check if pod has KECS management label
	if val, exists := pod.Labels["kecs.dev/managed-by"]; exists && val == "kecs" {
		return true
	}
	// Also check for ECS-specific labels
	if _, exists := pod.Labels["ecs.amazonaws.com/task-arn"]; exists {
		return true
	}
	return false
}

// isECSManagedDeployment checks if a deployment is managed by KECS
func isECSManagedDeployment(deployment *appsv1.Deployment) bool {
	// Check if deployment has ECS service prefix
	if strings.HasPrefix(deployment.Name, "ecs-service-") {
		return true
	}

	// Check labels
	if val, exists := deployment.Labels["kecs.dev/managed-by"]; exists && val == "kecs" {
		return true
	}

	// Check for ECS-specific annotations
	if _, exists := deployment.Annotations["ecs.amazonaws.com/task-definition"]; exists {
		return true
	}

	return false
}

// hasDeploymentConditionChanged checks if deployment conditions have changed
func hasDeploymentConditionChanged(oldDep, newDep *appsv1.Deployment) bool {
	// Check if number of conditions changed
	if len(oldDep.Status.Conditions) != len(newDep.Status.Conditions) {
		return true
	}

	// Create a map of old conditions for comparison
	oldConditions := make(map[appsv1.DeploymentConditionType]appsv1.DeploymentCondition)
	for _, c := range oldDep.Status.Conditions {
		oldConditions[c.Type] = c
	}

	// Compare each new condition with old
	for _, newCond := range newDep.Status.Conditions {
		if oldCond, exists := oldConditions[newCond.Type]; exists {
			if oldCond.Status != newCond.Status || oldCond.Reason != newCond.Reason {
				return true
			}
		} else {
			// New condition added
			return true
		}
	}

	return false
}
