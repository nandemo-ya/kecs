package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// InformerConfig holds configuration for setting up informers
type InformerConfig struct {
	// ResyncPeriod is how often the informer will do a full resync
	ResyncPeriod time.Duration
	
	// Namespace to watch, empty string means all namespaces
	Namespace string
	
	// LabelSelector to filter resources
	LabelSelector labels.Selector
}

// SetupInformers creates and configures the shared informer factory
func SetupInformers(kubeClient kubernetes.Interface, config InformerConfig) informers.SharedInformerFactory {
	// Create options for the informer factory
	options := []informers.SharedInformerOption{}
	
	// Add namespace filter if specified
	if config.Namespace != "" {
		options = append(options, informers.WithNamespace(config.Namespace))
	}
	
	// Add label selector if specified
	if config.LabelSelector != nil {
		options = append(options, informers.WithTweakListOptions(func(listOptions *metav1.ListOptions) {
			listOptions.LabelSelector = config.LabelSelector.String()
		}))
	}
	
	// Create the shared informer factory with resync period
	factory := informers.NewSharedInformerFactoryWithOptions(kubeClient, config.ResyncPeriod, options...)
	
	klog.V(2).Infof("Created shared informer factory with resync period: %v", config.ResyncPeriod)
	
	return factory
}

// StartInformers starts all registered informers and waits for their caches to sync
func StartInformers(ctx context.Context, factory informers.SharedInformerFactory) error {
	// Start all informers
	factory.Start(ctx.Done())
	
	// Wait for caches to sync
	klog.Info("Waiting for informer caches to sync...")
	
	// Get all the informers that were registered
	synced := factory.WaitForCacheSync(ctx.Done())
	for informerType, hasSynced := range synced {
		if !hasSynced {
			return fmt.Errorf("failed to sync informer cache for type: %v", informerType)
		}
		klog.V(2).Infof("Informer cache synced for type: %v", informerType)
	}
	
	klog.Info("All informer caches synced successfully")
	return nil
}

// NewControllerWithInformers creates a new sync controller with properly configured informers
func NewControllerWithInformers(
	ctx context.Context,
	kubeClient kubernetes.Interface,
	storage storage.Storage,
	config InformerConfig,
	workers int,
) (*SyncController, error) {
	// Create the informer factory
	factory := SetupInformers(kubeClient, config)
	
	// Get specific informers we need
	deploymentInformer := factory.Apps().V1().Deployments()
	replicaSetInformer := factory.Apps().V1().ReplicaSets()
	podInformer := factory.Core().V1().Pods()
	eventInformer := factory.Core().V1().Events()
	
	// Create the controller
	controller := NewSyncController(
		kubeClient,
		storage,
		deploymentInformer,
		replicaSetInformer,
		podInformer,
		eventInformer,
		workers,
		config.ResyncPeriod,
	)
	
	// Start the informers
	if err := StartInformers(ctx, factory); err != nil {
		return nil, fmt.Errorf("failed to start informers: %w", err)
	}
	
	return controller, nil
}

// WatchOptions provides options for watching specific resource types
type WatchOptions struct {
	// WatchDeployments enables watching deployment resources
	WatchDeployments bool
	
	// WatchPods enables watching pod resources
	WatchPods bool
	
	// WatchReplicaSets enables watching replicaset resources
	WatchReplicaSets bool
	
	// WatchEvents enables watching event resources
	WatchEvents bool
	
	// DeploymentSelector is an optional label selector for deployments
	DeploymentSelector labels.Selector
	
	// PodSelector is an optional label selector for pods
	PodSelector labels.Selector
}

// SetupSelectiveInformers creates informers only for selected resource types
func SetupSelectiveInformers(
	kubeClient kubernetes.Interface,
	options WatchOptions,
	resyncPeriod time.Duration,
) informers.SharedInformerFactory {
	// Create base factory
	factory := informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resyncPeriod,
	)
	
	// Only create informers for resources we want to watch
	if options.WatchDeployments {
		deploymentInformer := factory.Apps().V1().Deployments()
		klog.V(2).Info("Created deployment informer")
		_ = deploymentInformer.Lister() // Force creation
	}
	
	if options.WatchPods {
		podInformer := factory.Core().V1().Pods()
		klog.V(2).Info("Created pod informer")
		_ = podInformer.Lister() // Force creation
	}
	
	if options.WatchReplicaSets {
		rsInformer := factory.Apps().V1().ReplicaSets()
		klog.V(2).Info("Created replicaset informer")
		_ = rsInformer.Lister() // Force creation
	}
	
	if options.WatchEvents {
		eventInformer := factory.Core().V1().Events()
		klog.V(2).Info("Created event informer")
		_ = eventInformer.Lister() // Force creation
	}
	
	return factory
}