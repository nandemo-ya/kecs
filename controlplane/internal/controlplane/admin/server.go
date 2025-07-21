package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
}

// NewServer creates a new admin server instance
func NewServer(port int, storage storage.Storage) *Server {
	s := &Server{
		port:             port,
		metricsCollector: NewMetricsCollector(),
		config:           config.GetConfig(),
	}

	// Initialize health checker if storage is provided
	if storage != nil {
		s.healthChecker = NewHealthChecker(storage)
	} else {
		// Create health checker without storage
		s.healthChecker = NewHealthChecker(nil)
	}

	return s
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
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealthCheck) // Legacy endpoint
	mux.HandleFunc("/live", s.handleLiveness)
	mux.HandleFunc("/ready", s.handleReadiness(s.healthChecker))
	mux.HandleFunc("/health/detailed", s.handleHealthDetailed(s.healthChecker))

	// Metrics endpoints
	mux.HandleFunc("/metrics", s.handleMetrics(s.metricsCollector))
	mux.HandleFunc("/metrics/prometheus", s.handlePrometheusMetrics(s.metricsCollector))

	// Configuration endpoint
	mux.HandleFunc("/config", s.handleConfig)

	return mux
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// handleHealthCheck handles the legacy health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "OK",
		Timestamp: time.Now(),
		Version:   getVersion(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
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
	Port           int    `json:"port"`
	AdminPort      int    `json:"adminPort"`
	DataDir        string `json:"dataDir"`
	LogLevel       string `json:"logLevel"`
	Endpoint       string `json:"endpoint,omitempty"`
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
}

// LocalStackConfigResponse represents the LocalStack configuration in the response
type LocalStackConfigResponse struct {
	Enabled    bool   `json:"enabled"`
	UseTraefik bool   `json:"useTraefik"`
	Port       int    `json:"port,omitempty"`
	EdgePort   int    `json:"edgePort,omitempty"`
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
	Traefik          bool `json:"traefik"`
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
			Enabled:    cfg.LocalStack.Enabled,
			UseTraefik: cfg.LocalStack.UseTraefik,
			Port:       cfg.LocalStack.Port,
			EdgePort:   cfg.LocalStack.EdgePort,
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
			Traefik:          cfg.Features.Traefik,
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
