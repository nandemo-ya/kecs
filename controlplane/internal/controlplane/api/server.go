package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apiconfig "github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
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
}

// NewServer creates a new API server instance
func NewServer(port int, kubeconfig string, storage storage.Storage, localStackConfig *localstack.Config) (*Server, error) {

	// Initialize cluster manager first
	var clusterManager kubernetes.ClusterManager
	if apiconfig.GetBool("features.testMode") {
		log.Println("Running in test mode - Kubernetes operations will be simulated")
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

		log.Printf("Initialized k3d cluster manager (container mode: %v)",
			clusterConfig.ContainerMode)
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
			log.Printf("Warning: Failed to initialize task manager: %v (continuing without it)", err)
			// Continue without task manager in test/container mode - some features may not work
		} else {
			// Check if we're in recovery mode and allow startup without task manager
			if apiconfig.GetBool("features.autoRecoverState") {
				log.Printf("Warning: Failed to initialize task manager during recovery: %v (continuing without it)", err)
				// Continue without task manager initially - it will be initialized when clusters are created
			} else {
				log.Printf("Error: Failed to initialize task manager: %v", err)
				// TaskManager is critical for normal operation, return error
				return nil, fmt.Errorf("failed to initialize task manager: %w", err)
			}
		}
	} else {
		s.taskManager = taskManager
	}


	// Initialize test mode worker if in test mode
	if apiconfig.GetBool("features.testMode") {
		s.testModeWorker = NewTestModeTaskWorker(storage)
	}

	// Initialize LocalStack manager if configured
	if localStackConfig != nil && localStackConfig.Enabled {
		log.Printf("LocalStack config is enabled, initializing...")
		// Get Kubernetes client for LocalStack
		var kubeClient k8s.Interface
		if s.taskManager != nil {
			log.Printf("TaskManager is available, getting kubernetes client...")
			// Get the kubernetes client from task manager
			kubeConfig, err := kubernetes.GetKubeConfig()
			if err != nil {
				log.Printf("Warning: Failed to get kubernetes config for LocalStack: %v", err)
			} else {
				kubeClient, err = k8s.NewForConfig(kubeConfig)
				if err != nil {
					log.Printf("Warning: Failed to create kubernetes client for LocalStack: %v", err)
				}
			}
		} else {
			log.Printf("TaskManager is nil, skipping kubernetes client creation")
		}

		if kubeClient != nil {
			log.Printf("KubeClient created successfully, proceeding with LocalStack initialization...")
			// Check if Traefik is enabled
			if apiconfig.GetBool("features.traefik") {
				localStackConfig.UseTraefik = true
				// Don't set ProxyEndpoint here - it will be set dynamically when LocalStack is deployed
				log.Println("Traefik proxy enabled for LocalStack (port will be assigned dynamically)")
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
				log.Printf("Warning: Failed to initialize LocalStack manager: %v", err)
			} else {
				s.localStackManager = localStackManager
				// Create AWS proxy router
				awsProxyRouter, err := NewAWSProxyRouter(localStackManager)
				if err != nil {
					log.Printf("Warning: Failed to initialize AWS proxy router: %v", err)
				} else {
					s.awsProxyRouter = awsProxyRouter
					log.Printf("AWS proxy router initialized successfully")
				}


				// Initialize IAM integration if LocalStack is available
				if kubeClient != nil {
					iamConfig := &iam.Config{
						LocalStackEndpoint: fmt.Sprintf("http://localhost:%d", localStackConfig.Port),
						KubeNamespace:      "default",
						RolePrefix:         "kecs-",
					}
					iamIntegration, err := iam.NewIntegration(kubeClient, localStackManager, iamConfig)
					if err != nil {
						log.Printf("Warning: Failed to initialize IAM integration: %v", err)
					} else {
						s.iamIntegration = iamIntegration
						log.Println("IAM integration initialized successfully")
					}
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
						log.Printf("Warning: Failed to initialize CloudWatch integration: %v", err)
					} else {
						s.cloudWatchIntegration = cwIntegration
						log.Println("CloudWatch integration initialized successfully")
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
						log.Printf("Warning: Failed to initialize SSM integration: %v", err)
					} else {
						s.ssmIntegration = ssmIntegration
						log.Println("SSM Parameter Store integration initialized successfully")
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
						log.Printf("Warning: Failed to initialize Secrets Manager integration: %v", err)
					} else {
						s.secretsManagerIntegration = smIntegration
						log.Println("Secrets Manager integration initialized successfully")
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
						log.Printf("Warning: Failed to initialize S3 integration: %v", err)
					} else {
						s.s3Integration = s3Integration
						log.Println("S3 integration initialized successfully")
					}
				}

			}
		} else {
			log.Printf("KubeClient is nil, cannot initialize LocalStack manager and AWS proxy router")
		}
	}

	// Initialize ELBv2 integration (independent of LocalStack)
	if clusterManager != nil {
		elbv2Integration := elbv2.NewK8sIntegration(s.region, s.accountID)
		s.elbv2Integration = elbv2Integration
		
		// Initialize ELBv2 API and router
		elbv2API := NewELBv2API(storage, elbv2Integration, s.region, s.accountID)
		s.elbv2Router = generated_elbv2.NewRouter(elbv2API)
		
		log.Println("ELBv2 integration and API initialized successfully")
	} else {
		log.Printf("ClusterManager is nil, cannot initialize ELBv2 integration")
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
			log.Printf("LocalStack manager updated, re-initializing AWS proxy router...")
			s.localStackManager = newManager
			
			// Re-initialize AWS proxy router with the new LocalStack manager
			if s.localStackManager != nil {
				awsProxyRouter, err := NewAWSProxyRouter(s.localStackManager)
				if err != nil {
					log.Printf("Warning: Failed to re-initialize AWS proxy router: %v", err)
				} else {
					s.awsProxyRouter = awsProxyRouter
					log.Printf("AWS proxy router re-initialized successfully")
					log.Printf("LocalStackProxyMiddleware will now use the updated awsProxyRouter")
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

				log.Println("Service Discovery integration initialized successfully")
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
		log.Println("Starting state recovery...")
		if err := s.RecoverState(ctx); err != nil {
			log.Printf("State recovery failed: %v", err)
			// Continue startup even if recovery fails
		} else {
			log.Println("State recovery completed")
		}
	}

	// Start test mode worker if available
	if s.testModeWorker != nil {
		s.testModeWorker.Start(ctx)
	}

	// Start LocalStack manager if available
	if s.localStackManager != nil {
		if err := s.localStackManager.Start(ctx); err != nil {
			log.Printf("Failed to start LocalStack manager: %v", err)
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

	log.Printf("Starting API server on port %d", s.port)
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
	log.Println("Shutting down API server...")

	// Stop test mode worker if running
	if s.testModeWorker != nil {
		s.testModeWorker.Stop()
	}


	// Stop LocalStack manager if running
	if s.localStackManager != nil {
		if err := s.localStackManager.Stop(ctx); err != nil {
			log.Printf("Error stopping LocalStack manager: %v", err)
		}
	}

	// In the new architecture, the KECS instance (k3d cluster) is managed by the CLI,
	// not by the API server. We don't clean up k3d clusters here anymore.
	// Namespaces will be cleaned up when the KECS instance is stopped.
	if apiconfig.GetBool("kubernetes.keepClustersOnShutdown") {
		log.Println("KECS_KEEP_CLUSTERS_ON_SHUTDOWN is set (legacy setting, no longer needed)")
	}

	return s.httpServer.Shutdown(ctx)
}

// RecoverState recovers k3d clusters and Kubernetes resources from storage
func (s *Server) RecoverState(ctx context.Context) error {
	if s.storage == nil || s.clusterManager == nil {
		log.Println("Skipping state recovery: storage or cluster manager not available")
		return nil
	}

	// Get all clusters from storage
	clusters, err := s.storage.ClusterStore().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list clusters from storage: %w", err)
	}

	if len(clusters) == 0 {
		log.Println("No clusters found in storage, nothing to recover")
		return nil
	}

	log.Printf("Found %d clusters in storage, checking which need recovery...", len(clusters))

	// Track recovery results
	var recoveredCount, skippedCount, failedCount int

	for _, cluster := range clusters {
		if cluster.K8sClusterName == "" {
			log.Printf("Cluster %s has no k8s cluster name, skipping", cluster.Name)
			skippedCount++
			continue
		}

		// In the new architecture, we don't recreate k3d clusters
		// The KECS instance (k3d cluster) should already exist
		// We only need to ensure namespaces exist
		log.Printf("Ensuring namespace exists for ECS cluster %s", cluster.Name)

		// Recover LocalStack if it was deployed
		if err := s.recoverLocalStackForCluster(ctx, cluster); err != nil {
			log.Printf("Failed to recover LocalStack for cluster %s: %v", cluster.Name, err)
			// Don't count as failed since cluster was recovered
		}
		
		// Recover services for this cluster
		if err := s.recoverServicesForCluster(ctx, cluster); err != nil {
			log.Printf("Failed to recover services for cluster %s: %v", cluster.Name, err)
			// Don't count as failed since cluster was recovered
		}

		recoveredCount++
		log.Printf("Successfully recovered k3d cluster %s", cluster.K8sClusterName)
	}

	log.Printf("State recovery summary: %d recovered, %d skipped, %d failed",
		recoveredCount, skippedCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("failed to recover %d clusters", failedCount)
	}

	return nil
}

// recoverServicesForCluster recovers services and their deployments for a cluster
func (s *Server) recoverServicesForCluster(ctx context.Context, cluster *storage.Cluster) error {
	// Skip if storage is not available
	if s.storage == nil || s.storage.ServiceStore() == nil {
		log.Printf("Storage not available, skipping service recovery for cluster %s", cluster.Name)
		return nil
	}

	// Get all services for this cluster
	services, _, err := s.storage.ServiceStore().List(ctx, cluster.Name, "", "", 100, "")
	if err != nil {
		return fmt.Errorf("failed to list services for cluster %s: %w", cluster.Name, err)
	}

	if len(services) == 0 {
		log.Printf("No services found for cluster %s", cluster.Name)
		return nil
	}

	log.Printf("Found %d services to recover for cluster %s", len(services), cluster.Name)

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
		log.Printf("Recovering service %s in cluster %s...", service.ServiceName, cluster.Name)

		// Get task definition
		taskDefArn := service.TaskDefinitionARN
		if taskDefArn == "" {
			log.Printf("Service %s has no task definition, skipping", service.ServiceName)
			continue
		}

		// Parse task definition family and revision
		taskDef, err := s.storage.TaskDefinitionStore().GetByARN(ctx, taskDefArn)
		if err != nil {
			log.Printf("Failed to get task definition %s: %v", taskDefArn, err)
			continue
		}

		// Create service manager if not exists
		if s.taskManager == nil {
			log.Printf("Task manager not available, skipping service deployment for %s", service.ServiceName)
			continue
		}

		// Create deployment for the service
		err = s.taskManager.CreateServiceDeployment(ctx, cluster, service, taskDef)
		if err != nil {
			log.Printf("Failed to create deployment for service %s: %v", service.ServiceName, err)
			continue
		}

		// Update service status after successful deployment
		service.RunningCount = 0 // Will be updated by deployment controller
		service.Status = "ACTIVE"
		service.UpdatedAt = time.Now()

		if err := s.storage.ServiceStore().Update(ctx, service); err != nil {
			log.Printf("Failed to update service %s after recovery: %v", service.ServiceName, err)
		}

		// Schedule tasks to match desired count
		if service.DesiredCount > 0 {
			log.Printf("Scheduling %d tasks for service %s", service.DesiredCount, service.ServiceName)
			if err := s.scheduleServiceTasks(ctx, cluster, service, taskDef, service.DesiredCount); err != nil {
				log.Printf("Failed to schedule tasks for service %s: %v", service.ServiceName, err)
				// Don't fail the whole recovery process
			}
		}

		log.Printf("Successfully recovered service %s", service.ServiceName)
	}

	return nil
}

// scheduleServiceTasks creates tasks for a service to match its desired count
func (s *Server) scheduleServiceTasks(ctx context.Context, cluster *storage.Cluster, service *storage.Service, taskDef *storage.TaskDefinition, count int) error {
	log.Printf("Creating %d tasks for service %s", count, service.ServiceName)

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
			log.Printf("Failed to create task %s in storage: %v", taskID, err)
			continue
		}

		// Create Kubernetes pod
		pod, err := s.createPodForTask(ctx, cluster, service, taskDef, task)
		if err != nil {
			log.Printf("Failed to create pod for task %s: %v", taskID, err)
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
				log.Printf("Failed to update task %s with pod info: %v", taskID, err)
			}
		}

		log.Printf("Created task %s for service %s", taskID, service.ServiceName)
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
		log.Printf("No LocalStack state found for cluster %s", cluster.Name)
		return nil
	}
	
	// Deserialize LocalStack state
	state, err := storage.DeserializeLocalStackState(cluster.LocalStackState)
	if err != nil {
		return fmt.Errorf("failed to deserialize LocalStack state: %w", err)
	}
	
	if state == nil || !state.Deployed {
		log.Printf("LocalStack was not deployed in cluster %s", cluster.Name)
		return nil
	}
	
	log.Printf("LocalStack was deployed in cluster %s with status %s, attempting recovery...", cluster.Name, state.Status)
	
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
			log.Printf("Traefik is enabled for LocalStack recovery via features.traefik")
		}
		// Set container mode
		if appConfig.Features.ContainerMode {
			config.ContainerMode = true
			log.Printf("Container mode is enabled for LocalStack recovery")
		}
	}
	
	if config == nil || !config.Enabled {
		log.Printf("LocalStack is not enabled in configuration, skipping recovery")
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
			log.Printf("Using dynamic Traefik port %d for LocalStack proxy endpoint: %s", port, config.ProxyEndpoint)
		} else {
			log.Printf("Warning: Traefik is enabled but no port found for cluster %s", cluster.K8sClusterName)
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
		log.Printf("Re-initializing AWS proxy router after LocalStack recovery...")
		awsProxyRouter, err := NewAWSProxyRouter(s.localStackManager)
		if err != nil {
			log.Printf("Warning: Failed to re-initialize AWS proxy router: %v", err)
		} else {
			s.awsProxyRouter = awsProxyRouter
			log.Printf("AWS proxy router re-initialized successfully")
		}
	}
	
	// Check if LocalStack is already running in this cluster
	if clusterLocalStackManager.IsRunning() {
		log.Printf("LocalStack is already running in cluster %s", cluster.Name)
		// Update state to running
		if s.ecsAPI != nil {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.updateLocalStackState(cluster, "running", "")
			}
		}
		return nil
	}
	
	// Start LocalStack in the cluster
	log.Printf("Starting LocalStack in cluster %s...", cluster.Name)
	if err := clusterLocalStackManager.Start(ctx); err != nil {
		log.Printf("Failed to start LocalStack in cluster %s: %v", cluster.Name, err)
		// Update state to failed
		if s.ecsAPI != nil {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.updateLocalStackState(cluster, "failed", err.Error())
			}
		}
		return err
	}
	
	// Wait for LocalStack to be ready
	log.Printf("Waiting for LocalStack to be ready in cluster %s...", cluster.Name)
	if err := clusterLocalStackManager.WaitForReady(ctx, 2*time.Minute); err != nil {
		log.Printf("LocalStack failed to become ready in cluster %s: %v", cluster.Name, err)
		// Update state to failed
		if s.ecsAPI != nil {
			if defaultAPI, ok := s.ecsAPI.(*DefaultECSAPI); ok {
				defaultAPI.updateLocalStackState(cluster, "failed", err.Error())
			}
		}
		return err
	}
	
	log.Printf("LocalStack successfully recovered in cluster %s", cluster.Name)
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
