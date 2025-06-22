package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	k8s "k8s.io/client-go/kubernetes"
)

// Server represents the HTTP API server for KECS Control Plane
type Server struct {
	httpServer        *http.Server
	port              int
	kubeconfig        string
	ecsAPI            generated.ECSAPIInterface
	storage           storage.Storage
	kindManager       *kubernetes.KindManager
	taskManager       *kubernetes.TaskManager
	region            string
	accountID         string
	webSocketHub      *WebSocketHub
	webUIHandler      *WebUIHandler
	testModeWorker    *TestModeTaskWorker
	localStackManager       localstack.Manager
	awsProxyRouter          *AWSProxyRouter
	localStackEvents        *LocalStackEventIntegration
	iamIntegration          iam.Integration
	cloudWatchIntegration   cloudwatch.Integration
	ssmIntegration          ssm.Integration
	secretsManagerIntegration secretsmanager.Integration
	s3Integration           s3.Integration
	serviceDiscoveryAPI     *ServiceDiscoveryAPI
}

// NewServer creates a new API server instance
func NewServer(port int, kubeconfig string, storage storage.Storage, localStackConfig *localstack.Config) (*Server, error) {
	// Create WebSocket configuration
	wsConfig := &WebSocketConfig{
		AllowedOrigins: []string{
			"http://localhost:3000",                  // React development server
			"http://localhost:8080",                  // API server
			fmt.Sprintf("http://localhost:%d", port), // Dynamic port
		},
		AllowCredentials: true,
	}

	// Add environment-specific origins
	if envOrigins := os.Getenv("KECS_ALLOWED_ORIGINS"); envOrigins != "" {
		additionalOrigins := strings.Split(envOrigins, ",")
		for _, origin := range additionalOrigins {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				wsConfig.AllowedOrigins = append(wsConfig.AllowedOrigins, origin)
			}
		}
	}

	// Initialize kindManager first
	var kindManager *kubernetes.KindManager
	if os.Getenv("KECS_TEST_MODE") == "true" {
		log.Println("Running in test mode - Kubernetes operations will be simulated")
	} else if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		log.Println("Running in container mode - Kind cluster will be created on host")
		// Create KindManager directly without requiring kubeconfig
		kindManager = kubernetes.NewKindManager()
		log.Println("Created KindManager for container mode")
	} else {
		// Normal mode
		cachedKindManager, err := kubernetes.NewCachedKindManager(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cached kind manager: %w", err)
		}
		kindManager = cachedKindManager.KindManager
		log.Println("Initialized Kubernetes client cache for improved performance")
	}

	s := &Server{
		port:         port,
		kubeconfig:   kubeconfig,
		region:       "ap-northeast-1", // Default region
		accountID:    "123456789012",   // Default account ID
		ecsAPI:       nil, // Will be set after IAM integration
		storage:      storage,
		kindManager:  kindManager,
		webSocketHub: NewWebSocketHubWithConfig(wsConfig),
	}

	// Initialize task manager
	taskManager, err := kubernetes.NewTaskManager(storage)
	if err != nil {
		if os.Getenv("KECS_TEST_MODE") == "true" {
			log.Printf("Warning: Failed to initialize task manager in test mode: %v", err)
			// Continue without task manager in test mode - some features may not work
		} else {
			log.Printf("Error: Failed to initialize task manager: %v", err)
			// TaskManager is critical for normal operation, return error
			return nil, fmt.Errorf("failed to initialize task manager: %w", err)
		}
	} else {
		s.taskManager = taskManager
	}

	// Initialize Web UI handler if enabled
	if EnableWebUI() && GetWebUIFS != nil {
		if fs := GetWebUIFS(); fs != nil {
			s.webUIHandler = NewWebUIHandler(fs)
		}
	}

	// Initialize test mode worker if in test mode
	if os.Getenv("KECS_TEST_MODE") == "true" {
		s.testModeWorker = NewTestModeTaskWorker(storage)
	}

	// Initialize LocalStack manager if configured
	if localStackConfig != nil && localStackConfig.Enabled {
		// Get Kubernetes client for LocalStack
		var kubeClient k8s.Interface
		if s.taskManager != nil {
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
		}
		
		if kubeClient != nil {
			localStackManager, err := localstack.NewManager(localStackConfig, kubeClient)
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
				}
				
				// Create LocalStack event integration
				s.localStackEvents = NewLocalStackEventIntegration(
					localStackManager,
					s.webSocketHub,
					DefaultLocalStackEventConfig(),
				)
				
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
		}
	}

	// Create ECS API with integrations
	ecsAPI := NewDefaultECSAPI(storage, kindManager)
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
				serviceDiscoveryManager := servicediscovery.NewManager(kubeClient, "us-east-1", "123456789012")
				defaultAPI.SetServiceDiscoveryManager(serviceDiscoveryManager)
				
				// Create Service Discovery API handler
				s.serviceDiscoveryAPI = NewServiceDiscoveryAPI(serviceDiscoveryManager, "us-east-1", "123456789012")
				
				log.Println("Service Discovery integration initialized successfully")
			}
		}
	}
	s.ecsAPI = ecsAPI
	

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	ctx := context.Background()
	go s.webSocketHub.Run(ctx)

	// Start test mode worker if available
	if s.testModeWorker != nil {
		s.testModeWorker.Start(ctx)
	}

	// Start LocalStack manager if available
	if s.localStackManager != nil {
		if err := s.localStackManager.Start(ctx); err != nil {
			log.Printf("Failed to start LocalStack manager: %v", err)
		} else {
			// Start LocalStack event integration after LocalStack is running
			if s.localStackEvents != nil {
				if err := s.localStackEvents.Start(ctx); err != nil {
					log.Printf("Failed to start LocalStack event integration: %v", err)
				}
			}
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
	if s.webUIHandler != nil {
		uiBasePath := os.Getenv("KECS_UI_BASE_PATH")
		if uiBasePath == "" {
			uiBasePath = "/ui"
		}
		log.Printf("Web UI available at http://localhost:%d%s/", s.port, uiBasePath)
	}
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down API server...")
	
	// Stop test mode worker if running
	if s.testModeWorker != nil {
		s.testModeWorker.Stop()
	}
	
	// Stop LocalStack event integration if running
	if s.localStackEvents != nil {
		if err := s.localStackEvents.Stop(ctx); err != nil {
			log.Printf("Error stopping LocalStack event integration: %v", err)
		}
	}
	
	// Stop LocalStack manager if running
	if s.localStackManager != nil {
		if err := s.localStackManager.Stop(ctx); err != nil {
			log.Printf("Error stopping LocalStack manager: %v", err)
		}
	}
	
	return s.httpServer.Shutdown(ctx)
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

	// WebSocket endpoints
	mux.HandleFunc("/ws", s.HandleWebSocket(s.webSocketHub))
	mux.HandleFunc("/ws/logs", s.HandleWebSocket(s.webSocketHub))
	mux.HandleFunc("/ws/metrics", s.HandleWebSocket(s.webSocketHub))
	mux.HandleFunc("/ws/notifications", s.HandleWebSocket(s.webSocketHub))
	mux.HandleFunc("/ws/tasks", s.HandleWebSocket(s.webSocketHub))

	// Web UI endpoint (must be last to catch all)
	if s.webUIHandler != nil {
		// Support configurable UI base path
		uiBasePath := os.Getenv("KECS_UI_BASE_PATH")
		if uiBasePath == "" {
			uiBasePath = "/ui"
		}
		// Ensure base path starts with /
		if !strings.HasPrefix(uiBasePath, "/") {
			uiBasePath = "/" + uiBasePath
		}
		// Remove trailing slash
		uiBasePath = strings.TrimSuffix(uiBasePath, "/")

		// Handle UI routes - this will match /ui/* paths
		mux.Handle(uiBasePath+"/", http.StripPrefix(uiBasePath, s.webUIHandler))

		// Redirect /ui to /ui/
		mux.HandleFunc(uiBasePath, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, uiBasePath+"/", http.StatusMovedPermanently)
		})
	}

	// Apply middleware
	handler := http.Handler(mux)
	// Add LocalStack proxy middleware if available
	if s.awsProxyRouter != nil {
		handler = LocalStackProxyMiddleware(handler, s.awsProxyRouter)
	}
	handler = APIProxyMiddleware(handler)
	handler = SecurityHeadersMiddleware(handler)
	handler = CORSMiddleware(handler)
	handler = LoggingMiddleware(handler)

	return handler
}

// handleHealthCheck handles the health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
