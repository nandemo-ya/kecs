package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	k8sclient "k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Server represents the HTTP admin server for KECS Control Plane
type Server struct {
	httpServer       *http.Server
	port             int
	healthChecker    *HealthChecker
	metricsCollector *MetricsCollector
	config           *config.Config
	instanceAPI      *InstanceAPI
	ecsProxy         *ECSProxy
	logsAPI          *LogsAPI
	kubeClient       k8sclient.Interface
}

// NewServer creates a new admin server instance
func NewServer(port int, storage storage.Storage) *Server {
	cfg := config.GetConfig()

	s := &Server{
		port:             port,
		metricsCollector: NewMetricsCollector(),
		config:           cfg,
	}

	// Initialize health checker if storage is provided
	if storage != nil {
		s.healthChecker = NewHealthChecker(storage)
	} else {
		// Create health checker without storage
		s.healthChecker = NewHealthChecker(nil)
	}

	// Initialize API handlers
	instanceAPI, err := NewInstanceAPI(cfg, storage)
	if err != nil {
		logging.Error("Failed to create instance API", "error", err)
		// Continue without instance API
	} else {
		s.instanceAPI = instanceAPI
	}

	// Initialize ECS proxy with instance manager if available
	if s.instanceAPI != nil && s.instanceAPI.manager != nil {
		s.ecsProxy = NewECSProxy(cfg, s.instanceAPI.manager)
	}

	// Initialize Logs API with storage (Kubernetes client will be set later)
	s.logsAPI = NewLogsAPI(storage, nil)

	return s
}

// SetKubeClient sets the Kubernetes client for admin APIs
func (s *Server) SetKubeClient(kubeClient k8sclient.Interface) {
	s.kubeClient = kubeClient
	// Initialize Logs API with Kubernetes client
	if s.logsAPI == nil {
		s.logsAPI = NewLogsAPI(nil, kubeClient)
	} else {
		s.logsAPI.SetKubeClient(kubeClient)
	}
}

// SetStorage sets the storage for admin APIs
func (s *Server) SetStorage(storage storage.Storage) {
	// Update Logs API with storage
	if s.logsAPI == nil {
		s.logsAPI = NewLogsAPI(storage, nil)
	} else {
		s.logsAPI = NewLogsAPI(storage, s.logsAPI.kubeClient)
	}
}

// Start starts the HTTP admin server
func (s *Server) Start() error {
	router := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logging.Info("Starting Admin server", "port", s.port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP admin server
func (s *Server) Stop(ctx context.Context) error {
	logging.Info("Shutting down Admin server")
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all the admin routes
func (s *Server) setupRoutes() http.Handler {
	router := mux.NewRouter()

	// Health check endpoints
	router.HandleFunc("/health", s.handleHealth).Methods("GET")
	router.HandleFunc("/live", s.handleLiveness).Methods("GET")
	router.HandleFunc("/ready", s.handleReadiness(s.healthChecker)).Methods("GET")

	// Metrics endpoints
	router.HandleFunc("/metrics", s.handleMetrics(s.metricsCollector)).Methods("GET")
	router.HandleFunc("/metrics/prometheus", s.handlePrometheusMetrics(s.metricsCollector)).Methods("GET")

	// Configuration endpoint
	router.HandleFunc("/config", s.handleConfig).Methods("GET")

	// Register TUI API endpoints
	// IMPORTANT: ECS Proxy must be registered before instance API
	// to ensure specific routes are matched before generic ones
	if s.ecsProxy != nil {
		s.ecsProxy.RegisterRoutes(router)
	}
	if s.instanceAPI != nil {
		s.instanceAPI.RegisterRoutes(router)
	}

	// Register Logs API endpoints
	if s.logsAPI != nil {
		logging.Info("Registering Logs API routes")
		s.logsAPI.RegisterRoutes(router)
	} else {
		logging.Warn("Logs API is nil, not registering routes")
	}

	// Add middleware
	handler := http.Handler(router)

	// Add CORS middleware if allowed origins are configured
	if len(s.config.Server.AllowedOrigins) > 0 {
		handler = CORSMiddleware(s.config.Server.AllowedOrigins)(handler)
	}

	// Add API key middleware if configured
	// TODO: Add APIKey field to config
	// if s.config.Server.APIKey != "" {
	// 	handler = APIKeyMiddleware(s.config.Server.APIKey)(handler)
	// }

	return handler
}

// ConfigResponse represents the configuration response
type ConfigResponse struct {
	Server     ServerConfigResponse     `json:"server"`
	LocalStack LocalStackConfigResponse `json:"localstack"`
	Kubernetes KubernetesConfigResponse `json:"kubernetes"`
	Features   FeaturesConfigResponse   `json:"features"`
	AWS        AWSConfigResponse        `json:"aws"`
}

// ServerConfigResponse represents the server configuration in the response
type ServerConfigResponse struct {
	Port           int      `json:"port"`
	AdminPort      int      `json:"adminPort"`
	DataDir        string   `json:"dataDir"`
	LogLevel       string   `json:"logLevel"`
	Endpoint       string   `json:"endpoint,omitempty"`
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
}

// LocalStackConfigResponse represents the LocalStack configuration in the response
type LocalStackConfigResponse struct {
	Enabled  bool `json:"enabled"`
	Port     int  `json:"port,omitempty"`
	EdgePort int  `json:"edgePort,omitempty"`
}

// KubernetesConfigResponse represents the Kubernetes configuration in the response
type KubernetesConfigResponse struct {
	KubeconfigPath         string `json:"kubeconfigPath,omitempty"`
	K3DOptimized           bool   `json:"k3dOptimized"`
	K3DAsync               bool   `json:"k3dAsync"`
	DisableCoreDNS         bool   `json:"disableCoreDNS"`
	KeepClustersOnShutdown bool   `json:"keepClustersOnShutdown"`
}

// FeaturesConfigResponse represents the features configuration in the response
type FeaturesConfigResponse struct {
	TestMode         bool `json:"testMode"`
	ContainerMode    bool `json:"containerMode"`
	AutoRecoverState bool `json:"autoRecoverState"`
}

// AWSConfigResponse represents the AWS configuration in the response
type AWSConfigResponse struct {
	DefaultRegion string `json:"defaultRegion"`
	AccountID     string `json:"accountID"`
}

// handleConfig handles the configuration endpoint
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := s.config
	if cfg == nil {
		http.Error(w, "Configuration not available", http.StatusInternalServerError)
		return
	}

	// Build response with only non-sensitive configuration
	response := ConfigResponse{
		Server: ServerConfigResponse{
			Port:           cfg.Server.Port,
			AdminPort:      cfg.Server.AdminPort,
			DataDir:        cfg.Server.DataDir,
			LogLevel:       cfg.Server.LogLevel,
			Endpoint:       cfg.Server.Endpoint,
			AllowedOrigins: cfg.Server.AllowedOrigins,
		},
		LocalStack: LocalStackConfigResponse{
			Enabled:  cfg.LocalStack.Enabled,
			Port:     cfg.LocalStack.Port,
			EdgePort: cfg.LocalStack.EdgePort,
		},
		Kubernetes: KubernetesConfigResponse{
			KubeconfigPath:         cfg.Kubernetes.KubeconfigPath,
			K3DOptimized:           cfg.Kubernetes.K3DOptimized,
			K3DAsync:               cfg.Kubernetes.K3DAsync,
			DisableCoreDNS:         cfg.Kubernetes.DisableCoreDNS,
			KeepClustersOnShutdown: cfg.Kubernetes.KeepClustersOnShutdown,
		},
		Features: FeaturesConfigResponse{
			TestMode:         cfg.Features.TestMode,
			ContainerMode:    cfg.Features.ContainerMode,
			AutoRecoverState: cfg.Features.AutoRecoverState,
		},
		AWS: AWSConfigResponse{
			DefaultRegion: cfg.AWS.DefaultRegion,
			AccountID:     cfg.AWS.AccountID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
