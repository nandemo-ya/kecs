package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/informers"

	apiconfig "github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/controllers/sync"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Server represents the HTTP API server for KECS Control Plane
type Server struct {
	httpServer                *http.Server
	port                      int
	kubeconfig                string
	ecsAPI                    generated.ECSAPIInterface
	storage                   storage.Storage
	clusterManager            kubernetes.ClusterManager
	taskManager               *kubernetes.TaskManager
	region                    string
	accountID                 string
	testModeWorker            *TestModeTaskWorker
	localStackManager         localstack.Manager
	awsProxyRouter            *AWSProxyRouter
	iamIntegration            iam.Integration
	cloudWatchIntegration     cloudwatch.Integration
	ssmIntegration            ssm.Integration
	secretsManagerIntegration secretsmanager.Integration
	s3Integration             s3.Integration
	elbv2Integration          elbv2.Integration
	elbv2Router               *generated_elbv2.Router
	serviceDiscoveryAPI       *ServiceDiscoveryAPI
	syncController            *sync.SyncController
	syncCancelFunc            context.CancelFunc
	informerFactory           informers.SharedInformerFactory
}

// NewServer creates a new API server instance
func NewServer(port int, kubeconfig string, storage storage.Storage, localStackConfig *localstack.Config) (*Server, error) {

	// Initialize cluster manager first
	var clusterManager kubernetes.ClusterManager
	if apiconfig.GetBool("features.testMode") {
		logging.Info("Running in test mode - Kubernetes operations will be simulated")
	} else {
		// Create cluster manager (k3d only)
		clusterConfig := &kubernetes.ClusterManagerConfig{
			ContainerMode:  apiconfig.GetBool("features.containerMode"),
			KubeconfigPath: kubeconfig,
			EnableTraefik:  apiconfig.GetBool("features.traefik"),
			TraefikPort:    0, // 0 means dynamic port allocation
		}

		cm, err := kubernetes.NewClusterManager(clusterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster manager: %w", err)
		}
		clusterManager = cm

		logging.Info("Initialized k3d cluster manager",
			"containerMode", clusterConfig.ContainerMode)
	}

	// Get region and accountID from configuration
	region := apiconfig.GetString("aws.defaultRegion")
	accountID := apiconfig.GetString("aws.accountID")

	s := &Server{
		port:           port,
		kubeconfig:     kubeconfig,
		region:         region,
		accountID:      accountID,
		ecsAPI:         nil,              // Will be set after IAM integration
		storage:        storage,
		clusterManager: clusterManager,
	}

	// Initialize task manager
	taskManager, err := kubernetes.NewTaskManager(storage)
	if err != nil {
		if apiconfig.GetBool("features.testMode") || apiconfig.GetBool("features.containerMode") {
			logging.Warn("Failed to initialize task manager (continuing without it)",
				"error", err)
			// Continue without task manager in test/container mode - some features may not work
		} else {
			// Check if we're in recovery mode and allow startup without task manager
			if apiconfig.GetBool("features.autoRecoverState") {
				logging.Warn("Failed to initialize task manager during recovery (continuing without it)",
					"error", err)
				// Continue without task manager initially - it will be initialized when clusters are created
			} else {
				logging.Error("Failed to initialize task manager",
					"error", err)
				// TaskManager is critical for normal operation, return error
				return nil, fmt.Errorf("failed to initialize task manager: %w", err)
			}
		}
	} else {
		s.taskManager = taskManager
	}

	// Initialize sync controller
	if !apiconfig.GetBool("features.testMode") && s.taskManager != nil {
		// Try to get kubernetes client
		var kubeClient k8s.Interface
		kubeConfig, err := kubernetes.GetKubeConfig()
		if err != nil {
			logging.Warn("Failed to get kubernetes config for sync controller",
				"error", err)
		} else {
			kubeClient, err = k8s.NewForConfig(kubeConfig)
			if err != nil {
				logging.Warn("Failed to create kubernetes client for sync controller",
					"error", err)
			}
		}

		if kubeClient != nil {
			// Create informer factory
			resyncPeriod := 5 * time.Minute
			informerFactory := informers.NewSharedInformerFactory(kubeClient, resyncPeriod)
			
			// Get informers
			deploymentInformer := informerFactory.Apps().V1().Deployments()
			replicaSetInformer := informerFactory.Apps().V1().ReplicaSets()
			podInformer := informerFactory.Core().V1().Pods()
			eventInformer := informerFactory.Core().V1().Events()
			
			// Initialize sync controller
			syncController := sync.NewSyncController(
				kubeClient,
				storage,
				deploymentInformer,
				replicaSetInformer,
				podInformer,
				eventInformer,
				2, // workers
				resyncPeriod,
			)
			
			// Store informer factory to start it later with proper context
			s.informerFactory = informerFactory
			s.syncController = syncController
			logging.Info("Sync controller initialized successfully")
		} else {
			logging.Info("Kubernetes client not available, sync controller will not be initialized")
		}
	}

	// Initialize test mode worker if in test mode
	if apiconfig.GetBool("features.testMode") {
		s.testModeWorker = NewTestModeTaskWorker(storage)
	}

	// Initialize LocalStack manager if configured
	if localStackConfig != nil && localStackConfig.Enabled {
		logging.Info("LocalStack config is enabled, initializing...")
		// Get Kubernetes client for LocalStack
		var kubeClient k8s.Interface
		if s.taskManager != nil {
			logging.Info("TaskManager is available, getting kubernetes client...")
			// Get the kubernetes client from task manager
			kubeConfig, err := kubernetes.GetKubeConfig()
			if err != nil {
				logging.Warn("Failed to get kubernetes config for LocalStack",
					"error", err)
			} else {
				kubeClient, err = k8s.NewForConfig(kubeConfig)
				if err != nil {
					logging.Warn("Failed to create kubernetes client for LocalStack",
						"error", err)
				}
			}
		} else {
			logging.Info("TaskManager is nil, skipping kubernetes client creation")
		}

		if kubeClient != nil {
			logging.Info("KubeClient created successfully, proceeding with LocalStack initialization...")
			// Check if Traefik is enabled
			if apiconfig.GetBool("features.traefik") {
				localStackConfig.UseTraefik = true
				// Don't set ProxyEndpoint here - it will be set dynamically when LocalStack is deployed
				logging.Info("Traefik proxy enabled for LocalStack (port will be assigned dynamically)")
			}
			
			// Set container mode
			localStackConfig.ContainerMode = apiconfig.GetBool("features.containerMode")
			
			// Get kubeconfig if available
			var kubeConfig *rest.Config
			// We'll create LocalStack managers per-cluster when they're created
			// At startup, we don't have a cluster yet
			kubeConfig = nil
			
			localStackManager, err := localstack.NewManager(localStackConfig, kubeClient, kubeConfig)
			if err != nil {
				logging.Warn("Failed to initialize LocalStack manager",
					"error", err)
			} else {
				s.localStackManager = localStackManager
				// Create AWS proxy router
				awsProxyRouter, err := NewAWSProxyRouter(localStackManager)
				if err != nil {
					logging.Warn("Failed to initialize AWS proxy router",
						"error", err)
				} else {
					s.awsProxyRouter = awsProxyRouter
					logging.Info("AWS proxy router initialized successfully")
				}


				// Initialize IAM integration if LocalStack is available and IAM integration is enabled
				if kubeClient != nil && apiconfig.GetBool("features.iamIntegration") {
					iamConfig := &iam.Config{
						LocalStackEndpoint: fmt.Sprintf("http://localhost:%d", localStackConfig.Port),
						KubeNamespace:      "default",
						RolePrefix:         "kecs-",
					}
					iamIntegration, err := iam.NewIntegration(kubeClient, localStackManager, iamConfig)
					if err != nil {
						logging.Warn("Failed to initialize IAM integration",
							"error", err)
					} else {
						s.iamIntegration = iamIntegration
						logging.Info("IAM integration initialized successfully")
					}
				} else if kubeClient != nil && !apiconfig.GetBool("features.iamIntegration") {
					logging.Info("IAM integration is disabled by configuration")
				}

				// Initialize CloudWatch integration if LocalStack is available
				if kubeClient != nil {
					cwConfig := &cloudwatch.Config{
						LocalStackEndpoint: fmt.Sprintf("http://localhost:%d", localStackConfig.Port),
						LogGroupPrefix:     "/ecs/",
						RetentionDays:      7,
						KubeNamespace:      "default",
					}
					cwIntegration, err := cloudwatch.NewIntegration(kubeClient, localStackManager, cwConfig)
					if err != nil {
						logging.Warn("Failed to initialize CloudWatch integration",
							"error", err)
					} else {
						s.cloudWatchIntegration = cwIntegration
						logging.Info("CloudWatch integration initialized successfully")
					}
				}

				// Initialize SSM integration if LocalStack is available
				if kubeClient != nil {
					ssmConfig := &ssm.Config{
						LocalStackEndpoint: fmt.Sprintf("http://localhost:%d", localStackConfig.Port),
						SecretPrefix:       "ssm-",
						KubeNamespace:      "default",
						SyncRetries:        3,
						CacheTTL:           5 * time.Minute,
					}
					ssmIntegration, err := ssm.NewIntegration(kubeClient, localStackManager, ssmConfig)
					if err != nil {
						logging.Warn("Failed to initialize SSM integration",
							"error", err)
					} else {
						s.ssmIntegration = ssmIntegration
						logging.Info("SSM Parameter Store integration initialized successfully")
					}
				}

				// Initialize Secrets Manager integration if LocalStack is available
				if kubeClient != nil {
					smConfig := &secretsmanager.Config{
						LocalStackEndpoint: fmt.Sprintf("http://localhost:%d", localStackConfig.Port),
						SecretPrefix:       "sm-",
						KubeNamespace:      "default",
						SyncRetries:        3,
						CacheTTL:           5 * time.Minute,
					}
					smIntegration, err := secretsmanager.NewIntegration(kubeClient, localStackManager, smConfig)
					if err != nil {
						logging.Warn("Failed to initialize Secrets Manager integration",
							"error", err)
					} else {
						s.secretsManagerIntegration = smIntegration
						logging.Info("Secrets Manager integration initialized successfully")
					}
				}

				// Initialize S3 integration if LocalStack is available
				if kubeClient != nil {
					s3Config := &s3.Config{
						LocalStackEndpoint: fmt.Sprintf("http://localhost:%d", localStackConfig.Port),
						Region:             "us-east-1",
						ForcePathStyle:     true, // Required for LocalStack
					}
					s3Integration, err := s3.NewIntegration(kubeClient, localStackManager, s3Config)
					if err != nil {
						logging.Warn("Failed to initialize S3 integration",
							"error", err)
					} else {
						s.s3Integration = s3Integration
						logging.Info("S3 integration initialized successfully")
					}
				}

			}
		} else {
			logging.Info("KubeClient is nil, cannot initialize LocalStack manager and AWS proxy router")
		}
	}

	// Initialize ELBv2 integration (independent of LocalStack)
	if clusterManager != nil {
		elbv2Integration := elbv2.NewK8sIntegration(s.region, s.accountID)
		s.elbv2Integration = elbv2Integration
		
		// Initialize ELBv2 API and router
		elbv2API := NewELBv2API(storage, elbv2Integration, s.region, s.accountID)
		s.elbv2Router = generated_elbv2.NewRouter(elbv2API)
		
		logging.Info("ELBv2 integration and API initialized successfully")
	} else {
		logging.Info("ClusterManager is nil, cannot initialize ELBv2 integration")
	}

	// Create ECS API with integrations
	var ecsAPI generated.ECSAPIInterface
	if clusterManager != nil {
		ecsAPI = NewDefaultECSAPIWithClusterManager(storage, clusterManager, s.region, s.accountID)
	} else {
		// Fallback for test mode or when no cluster manager is available
		ecsAPI = NewDefaultECSAPI(storage)
	}
	if defaultAPI, ok := ecsAPI.(*DefaultECSAPI); ok {
		if s.iamIntegration != nil {
			defaultAPI.SetIAMIntegration(s.iamIntegration)
		}
		if s.cloudWatchIntegration != nil {
			defaultAPI.SetCloudWatchIntegration(s.cloudWatchIntegration)
		}
		if s.ssmIntegration != nil {
			defaultAPI.SetSSMIntegration(s.ssmIntegration)
		}
		if s.secretsManagerIntegration != nil {
			defaultAPI.SetSecretsManagerIntegration(s.secretsManagerIntegration)
		}
		if s.s3Integration != nil {
			defaultAPI.SetS3Integration(s.s3Integration)
		}
		if s.elbv2Integration != nil {
			defaultAPI.SetELBv2Integration(s.elbv2Integration)
		}
		if s.localStackManager != nil {
			defaultAPI.SetLocalStackManager(s.localStackManager)
		}
		// Also set the LocalStack config so it can be used when creating clusters
		if localStackConfig != nil {
			defaultAPI.SetLocalStackConfig(localStackConfig)
		}
		
		// Set callback to re-initialize AWS proxy router when LocalStack manager is updated
		defaultAPI.SetLocalStackUpdateCallback(func(newManager localstack.Manager) {
			logging.Info("LocalStack manager updated, re-initializing AWS proxy router...")
			s.localStackManager = newManager
			
			// Re-initialize AWS proxy router with the new LocalStack manager
			if s.localStackManager != nil {
				awsProxyRouter, err := NewAWSProxyRouter(s.localStackManager)
				if err != nil {
					logging.Warn("Failed to re-initialize AWS proxy router",
						"error", err)
				} else {
					s.awsProxyRouter = awsProxyRouter
					logging.Info("AWS proxy router re-initialized successfully")
					logging.Info("LocalStackProxyMiddleware will now use the updated awsProxyRouter")
				}
			}
		})

		// Initialize Service Discovery if we have kubernetes client
		if localStackConfig != nil && localStackConfig.Enabled {
			var kubeClient k8s.Interface
			if s.taskManager != nil {
				kubeConfig, err := kubernetes.GetKubeConfig()
				if err == nil {
					kubeClient, _ = k8s.NewForConfig(kubeConfig)
				}
			}

			if kubeClient != nil {
				serviceDiscoveryManager := servicediscovery.NewManager(kubeClient, s.region, s.accountID)
				defaultAPI.SetServiceDiscoveryManager(serviceDiscoveryManager)

				// Create Service Discovery API handler
				s.serviceDiscoveryAPI = NewServiceDiscoveryAPI(serviceDiscoveryManager, s.region, s.accountID)

				logging.Info("Service Discovery integration initialized successfully")
			}
		}
	}
	s.ecsAPI = ecsAPI

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	ctx := context.Background()

	// Recover state if enabled and not in test mode
	if !apiconfig.GetBool("features.testMode") && apiconfig.GetBool("features.autoRecoverState") {
		logging.Info("Starting state recovery...")
		if err := s.RecoverState(ctx); err != nil {
			logging.Error("State recovery failed",
				"error", err)
			// Continue startup even if recovery fails
		} else {
			logging.Info("State recovery completed")
		}
	}

	// Start test mode worker if available
	if s.testModeWorker != nil {
		s.testModeWorker.Start(ctx)
	}

	// Start sync controller if available
	if s.syncController != nil && s.informerFactory != nil {
		logging.Info("Starting sync controller...")
		syncCtx, cancel := context.WithCancel(ctx)
		s.syncCancelFunc = cancel
		
		// Start informers with the sync context
		logging.Info("Starting informers...")
		s.informerFactory.Start(syncCtx.Done())
		
		go func() {
			if err := s.syncController.Run(syncCtx); err != nil {
				logging.Error("Sync controller stopped with error",
					"error", err)
			}
		}()
	} else {
		logging.Warn("Sync controller or informer factory not available",
			"syncController", s.syncController != nil,
			"informerFactory", s.informerFactory != nil)
	}

	// Start LocalStack manager if available
	if s.localStackManager != nil {
		if err := s.localStackManager.Start(ctx); err != nil {
			logging.Error("Failed to start LocalStack manager",
				"error", err)
		} else {
		}
	}

	router := s.SetupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logging.Info("Starting API server",
		"port", s.port)
	return s.httpServer.ListenAndServe()
}

// handleELBv2Request handles ELBv2 API requests using the generated router
func (s *Server) handleELBv2Request(w http.ResponseWriter, r *http.Request) {
	if s.elbv2Router == nil {
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"__type":"ServiceUnavailable","message":"ELBv2 API not available"}`)
		return
	}
	
	// Use the generated router to handle the request
	s.elbv2Router.Route(w, r)
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	logging.Info("Shutting down API server...")

	// Stop test mode worker if running
	if s.testModeWorker != nil {
		s.testModeWorker.Stop()
	}

	// Stop sync controller if running
	if s.syncController != nil && s.syncCancelFunc != nil {
		logging.Info("Stopping sync controller and informers...")
		s.syncCancelFunc()
		// Give it a moment to shut down gracefully
		select {
		case <-time.After(2 * time.Second):
			logging.Info("Sync controller stopped")
		case <-ctx.Done():
			logging.Warn("Context cancelled while waiting for sync controller to stop")
		}
	}

	// Stop LocalStack manager if running
	if s.localStackManager != nil {
		if err := s.localStackManager.Stop(ctx); err != nil {
			logging.Error("Error stopping LocalStack manager",
				"error", err)
		}
	}

	// In the new architecture, the KECS instance (k3d cluster) is managed by the CLI,
	// not by the API server. We don't clean up k3d clusters here anymore.
	// Namespaces will be cleaned up when the KECS instance is stopped.
	if apiconfig.GetBool("kubernetes.keepClustersOnShutdown") {
		logging.Info("KECS_KEEP_CLUSTERS_ON_SHUTDOWN is set (legacy setting, no longer needed)")
	}

	return s.httpServer.Shutdown(ctx)
}

// RecoverState recovers k3d clusters and Kubernetes resources from storage
func (s *Server) RecoverState(ctx context.Context) error {
	if s.storage == nil || s.clusterManager == nil {
		logging.Info("Skipping state recovery: storage or cluster manager not available")
		return nil
	}

	// Get all clusters from storage
	clusters, err := s.storage.ClusterStore().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clusters from storage: %w", err)
	}

	if len(clusters) == 0 {
		logging.Info("No clusters found in storage, nothing to recover")
		return nil
	}

	logging.Info("Found clusters in storage, checking which need recovery...",
		"count", len(clusters))

	// Track recovery results
	var recoveredCount, skippedCount, failedCount int

	for _, cluster := range clusters {
		if cluster.K8sClusterName == "" {
			logging.Info("Cluster has no k8s cluster name, skipping",
				"cluster", cluster.Name)
			skippedCount++
			continue
		}

		// In the new architecture, we don't recreate k3d clusters
		// The KECS instance (k3d cluster) should already exist
		// We only need to ensure namespaces exist
		logging.Info("Ensuring namespace exists for ECS cluster",
			"cluster", cluster.Name)

		// Recover LocalStack if it was deployed
		if err := s.recoverLocalStackForCluster(ctx, cluster); err != nil {
			logging.Error("Failed to recover LocalStack for cluster",
				"cluster", cluster.Name,
				"error", err)
			// Don't count as failed since cluster was recovered
		}
		
		// Recover services for this cluster
		if err := s.recoverServicesForCluster(ctx, cluster); err != nil {
			logging.Error("Failed to recover services for cluster",
				"cluster", cluster.Name,
				"error", err)
			// Don't count as failed since cluster was recovered
		}

		recoveredCount++
		logging.Info("Successfully recovered k3d cluster",
			"k8sCluster", cluster.K8sClusterName)
	}

	logging.Info("State recovery summary",
		"recovered", recoveredCount,
		"skipped", skippedCount,
		"failed", failedCount)

	if failedCount > 0 {
		return fmt.Errorf("failed to recover %d clusters", failedCount)
	}

	return nil
}

// recoverServicesForCluster recovers services and their deployments for a cluster
func (s *Server) recoverServicesForCluster(ctx context.Context, cluster *storage.Cluster) error {
	// Skip if storage is not available
	if s.storage == nil || s.storage.ServiceStore() == nil {
		logging.Info("Storage not available, skipping service recovery for cluster",
			"cluster", cluster.Name)
		return nil
	}

	// Get all services for this cluster
	services, _, err := s.storage.ServiceStore().List(ctx, cluster.Name, "", "", 100, "")
	if err != nil {
		return fmt.Errorf("failed to list services for cluster %s: %w", cluster.Name, err)
	}

	if len(services) == 0 {
		logging.Info("No services found for cluster",
			"cluster", cluster.Name)
		return nil
	}

	logging.Info("Found services to recover for cluster",
		"count", len(services),
		"cluster", cluster.Name)

	// Get Kubernetes client for the cluster
	kubeClient, err := s.clusterManager.GetKubeClient(cluster.K8sClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kubernetes client for cluster %s: %w", cluster.K8sClusterName, err)
	}

	// Create namespace if it doesn't exist
	namespace := fmt.Sprintf("kecs-%s", cluster.Name)
	if err := kubernetes.EnsureNamespace(ctx, kubeClient, namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace %s: %w", namespace, err)
	}

	// Recover each service
	for _, service := range services {
		logging.Info("Recovering service in cluster...",
			"service", service.ServiceName,
			"cluster", cluster.Name)

		// Get task definition
		taskDefArn := service.TaskDefinitionARN
		if taskDefArn == "" {
			logging.Warn("Service has no task definition, skipping",
				"service", service.ServiceName)
			continue
		}

		// Parse task definition family and revision
		taskDef, err := s.storage.TaskDefinitionStore().GetByARN(ctx, taskDefArn)
		if err != nil {
			logging.Error("Failed to get task definition",
				"taskDefinitionArn", taskDefArn,
				"error", err)
			continue
		}

		// Create service manager if not exists
		if s.taskManager == nil {
			logging.Warn("Task manager not available, skipping service deployment",
				"service", service.ServiceName)
			continue
		}

		// Create deployment for the service
		err = s.taskManager.CreateServiceDeployment(ctx, cluster, service, taskDef)
		if err != nil {
			logging.Error("Failed to create deployment for service",
				"service", service.ServiceName,
				"error", err)
			continue
		}

		// Update service status after successful deployment
		service.RunningCount = 0 // Will be updated by deployment controller
		service.Status = "ACTIVE"
		service.UpdatedAt = time.Now()

		if err := s.storage.ServiceStore().Update(ctx, service); err != nil {
			logging.Error("Failed to update service after recovery",
				"service", service.ServiceName,
				"error", err)
		}

		// Schedule tasks to match desired count
		if service.DesiredCount > 0 {
			logging.Info("Scheduling tasks for service",
				"count", service.DesiredCount,
				"service", service.ServiceName)
			if err := s.scheduleServiceTasks(ctx, cluster, service, taskDef, service.DesiredCount); err != nil {
				logging.Error("Failed to schedule tasks for service",
					"service", service.ServiceName,
					"error", err)
				// Don't fail the whole recovery process
			}
		}

		logging.Info("Successfully recovered service",
			"service", service.ServiceName)
	}

	return nil
}

// scheduleServiceTasks creates tasks for a service to match its desired count
func (s *Server) scheduleServiceTasks(ctx context.Context, cluster *storage.Cluster, service *storage.Service, taskDef *storage.TaskDefinition, count int) error {
	logging.Info("Creating tasks for service",
		"count", count,
		"service", service.ServiceName)

	// Check if we have the necessary components
	if s.taskManager == nil {
		return fmt.Errorf("task manager not available")
	}

	// Define namespace for the cluster
	namespace := fmt.Sprintf("kecs-%s", cluster.Name)

	// Create tasks for the service
	for i := 0; i < count; i++ {
		// Generate task ID
		taskID := uuid.New().String()
		taskARN := fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", s.region, s.accountID, cluster.Name, taskID)

		// Create task in storage
		task := &storage.Task{
			ID:                taskID,
			ARN:               taskARN,
			ClusterARN:        cluster.ARN,
			TaskDefinitionARN: taskDef.ARN,
			DesiredStatus:     "RUNNING",
			LastStatus:        "PENDING",
			LaunchType:        service.LaunchType,
			StartedBy:         fmt.Sprintf("ecs-svc/%s", service.ServiceName),
			CreatedAt:         time.Now(),
			Region:            s.region,
			AccountID:         s.accountID,
			CPU:               taskDef.CPU,
			Memory:            taskDef.Memory,
		}

		// Store task
		if err := s.storage.TaskStore().Create(ctx, task); err != nil {
			logging.Error("Failed to create task in storage",
				"taskId", taskID,
				"error", err)
			continue
		}

		// Create Kubernetes pod
		pod, err := s.createPodForTask(ctx, cluster, service, taskDef, task)
		if err != nil {
			logging.Error("Failed to create pod for task",
				"taskId", taskID,
				"error", err)
			// Update task status to failed
			task.LastStatus = "FAILED"
			task.StoppedReason = err.Error()
			now := time.Now()
			task.StoppedAt = &now
			s.storage.TaskStore().Update(ctx, task)
			continue
		}

		// Update task with pod name
		if pod != nil {
			task.PodName = pod.Name
			task.Namespace = namespace
			task.LastStatus = "PROVISIONING"
			if err := s.storage.TaskStore().Update(ctx, task); err != nil {
				logging.Error("Failed to update task with pod info",
					"taskId", taskID,
					"error", err)
			}
		}

		logging.Info("Created task for service",
			"taskId", taskID,
			"service", service.ServiceName)
	}

	return nil
}

// createPodForTask creates a Kubernetes pod for an ECS task
func (s *Server) createPodForTask(ctx context.Context, cluster *storage.Cluster, service *storage.Service, taskDef *storage.TaskDefinition, task *storage.Task) (*corev1.Pod, error) {
	// Get Kubernetes client
	kubeClient, err := s.clusterManager.GetKubeClient(cluster.K8sClusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes client: %w", err)
	}

	// Create task converter
	taskConverter := converters.NewTaskConverter(s.region, s.accountID)

	// Convert task definition to pod
	namespace := fmt.Sprintf("kecs-%s", cluster.Name)
	
	// Create a minimal RunTask request for the converter
	runTaskReq := map[string]interface{}{
		"cluster":        cluster.Name,
		"taskDefinition": taskDef.ARN,
		"launchType":     service.LaunchType,
	}
	runTaskReqJSON, err := json.Marshal(runTaskReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run task request: %w", err)
	}
	
	pod, err := taskConverter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, task.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert task definition to pod: %w", err)
	}

	// Set pod name and labels
	podName := fmt.Sprintf("%s-%s", service.ServiceName, strings.Split(task.ARN, "/")[2])
	pod.Name = podName
	pod.Labels = map[string]string{
		"app":         service.ServiceName,
		"ecs-service": service.ServiceName,
		"ecs-task":    strings.Split(task.ARN, "/")[2],
		"ecs-cluster": cluster.Name,
	}

	// Add service account if needed
	if taskDef.TaskRoleARN != "" {
		pod.Spec.ServiceAccountName = fmt.Sprintf("%s-task-role", taskDef.Family)
	}

	// Create pod
	createdPod, err := kubeClient.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create pod: %w", err)
	}

	return createdPod, nil
}

// recoverLocalStackForCluster recovers LocalStack deployment if it was previously deployed
func (s *Server) recoverLocalStackForCluster(ctx context.Context, cluster *storage.Cluster) error {
	// Check if LocalStack was deployed
	if cluster.LocalStackState == "" {
		logging.Info("No LocalStack state found for cluster",
			"cluster", cluster.Name)
		return nil
	}
	
	// Deserialize LocalStack state
	state, err := storage.DeserializeLocalStackState(cluster.LocalStackState)
	if err != nil {
		return fmt.Errorf("failed to deserialize LocalStack state: %w", err)
	}
	
	if state == nil || !state.Deployed {
		logging.Info("LocalStack was not deployed in cluster",
			"cluster", cluster.Name)
		return nil
	}
	
	logging.Info("LocalStack was deployed in cluster, attempting recovery...",
		"cluster", cluster.Name,
		"status", state.Status)
	
	// Check if LocalStack is enabled
	var config *localstack.Config
	if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok && defaultAPI.localStackConfig != nil {
		// Create a copy of the config from ECS API
		configCopy := *defaultAPI.localStackConfig
		config = &configCopy
	} else if s.localStackManager != nil {
		config = s.localStackManager.GetConfig()
	} else {
		// Use default config and check if enabled via environment
		config = localstack.DefaultConfig()
		// Use Viper config which handles environment variables
		appConfig := apiconfig.GetConfig()
		if appConfig.LocalStack.Enabled {
			config.Enabled = true
		}
		// Check features.traefik configuration
		if appConfig.Features.Traefik {
			config.UseTraefik = true
			logging.Info("Traefik is enabled for LocalStack recovery via features.traefik")
		}
		// Set container mode
		if appConfig.Features.ContainerMode {
			config.ContainerMode = true
			logging.Info("Container mode is enabled for LocalStack recovery")
		}
	}
	
	if config == nil || !config.Enabled {
		logging.Info("LocalStack is not enabled in configuration, skipping recovery")
		return nil
	}
	
	// Get Kubernetes client for the specific k3d cluster
	kubeClient, err := s.clusterManager.GetKubeClient(cluster.K8sClusterName)
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}
	
	// If Traefik is enabled, get the dynamic port from cluster manager
	if config.UseTraefik && s.clusterManager != nil {
		if port, exists := s.clusterManager.GetTraefikPort(cluster.K8sClusterName); exists {
			config.ProxyEndpoint = fmt.Sprintf("http://localhost:%d", port)
			logging.Info("Using dynamic Traefik port for LocalStack proxy endpoint",
				"port", port,
				"proxyEndpoint", config.ProxyEndpoint)
		} else {
			logging.Warn("Traefik is enabled but no port found for cluster",
				"k8sCluster", cluster.K8sClusterName)
		}
	}

	// Get kube config
	kubeConfig, err := s.clusterManager.GetKubeConfig(cluster.K8sClusterName)
	if err != nil {
		return fmt.Errorf("failed to get kube config: %w", err)
	}

	// Create a new LocalStack manager with the cluster-specific client
	clusterLocalStackManager, err := localstack.NewManager(config, kubeClient.(*k8s.Clientset), kubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create LocalStack manager: %w", err)
	}
	
	// Update the server's LocalStack manager
	s.localStackManager = clusterLocalStackManager
	
	// Re-initialize AWS proxy router with the new LocalStack manager
	if s.localStackManager != nil {
		logging.Info("Re-initializing AWS proxy router after LocalStack recovery...")
		awsProxyRouter, err := NewAWSProxyRouter(s.localStackManager)
		if err != nil {
			logging.Warn("Failed to re-initialize AWS proxy router",
				"error", err)
		} else {
			s.awsProxyRouter = awsProxyRouter
			logging.Info("AWS proxy router re-initialized successfully")
		}
	}
	
	// Check if LocalStack is already running in this cluster
	if clusterLocalStackManager.IsRunning() {
		logging.Info("LocalStack is already running in cluster",
			"cluster", cluster.Name)
		// Update state to running
		if s.ecsAPI != nil {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.updateLocalStackState(cluster, "running", "")
			}
		}
		return nil
	}
	
	// Start LocalStack in the cluster
	logging.Info("Starting LocalStack in cluster...",
		"cluster", cluster.Name)
	if err := clusterLocalStackManager.Start(ctx); err != nil {
		logging.Error("Failed to start LocalStack in cluster",
			"cluster", cluster.Name,
			"error", err)
		// Update state to failed
		if s.ecsAPI != nil {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.updateLocalStackState(cluster, "failed", err.Error())
			}
		}
		return err
	}
	
	// Wait for LocalStack to be ready
	logging.Info("Waiting for LocalStack to be ready in cluster...",
		"cluster", cluster.Name)
	if err := clusterLocalStackManager.WaitForReady(ctx, 2*time.Minute); err != nil {
		logging.Error("LocalStack failed to become ready in cluster",
			"cluster", cluster.Name,
			"error", err)
		// Update state to failed
		if s.ecsAPI != nil {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.updateLocalStackState(cluster, "failed", err.Error())
			}
		}
		return err
	}
	
	logging.Info("LocalStack successfully recovered in cluster",
		"cluster", cluster.Name)
	// Update state to running
	if s.ecsAPI != nil {
		if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
			defaultAPI.updateLocalStackState(cluster, "running", "")
		}
	}
	
	return nil
}

// SetupRoutes configures all the API routes
func (s *Server) SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// AWS ECS API endpoint (AWS CLI format)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if it's a Service Discovery request
		target := r.Header.Get("X-Amz-Target")
		if target != "" && strings.Contains(target, "ServiceDiscovery") && s.serviceDiscoveryAPI != nil {
			s.serviceDiscoveryAPI.HandleServiceDiscoveryRequest(w, r)
			return
		}
		// Check if it's an ELBv2 request
		if target != "" && strings.Contains(target, "ElasticLoadBalancing") {
			s.handleELBv2Request(w, r)
			return
		}
		
		// Handle custom KECS endpoints
		// Check both URL path and X-Amz-Target header
		if r.URL.Path == "/v1/GetTaskLogs" || 
		   (r.URL.Path == "/" && r.Header.Get("X-Amz-Target") == "AWSie.GetTaskLogs") {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.HandleGetTaskLogs(w, r)
				return
			}
		}
		
		// Otherwise handle as ECS request
		// Create router and handle request
		router := generated.NewRouter(s.ecsAPI)
		router.Route(w, r)
	})

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealthCheck)

	// LocalStack endpoints
	mux.HandleFunc("/api/localstack/status", s.GetLocalStackStatus)
	mux.HandleFunc("/localstack/dashboard", s.GetLocalStackDashboard)



	// Apply middleware
	handler := http.Handler(mux)
	handler = APIProxyMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = CORSMiddleware(handler)
	handler = LoggingMiddleware(handler)
	
	// Add LocalStack proxy middleware LAST so it runs FIRST
	// This ensures AWS API calls are intercepted before reaching ECS handlers
	// Pass the server instance so the middleware can dynamically check awsProxyRouter
	handler = LocalStackProxyMiddleware(handler, s)

	return handler
}

// handleHealthCheck handles the health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
