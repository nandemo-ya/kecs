package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Server represents the HTTP admin server for KECS Control Plane
type Server struct {
	httpServer      *http.Server
	port            int
	healthChecker   *HealthChecker
	metricsCollector *MetricsCollector
}

// NewServer creates a new admin server instance
func NewServer(port int, storage storage.Storage) *Server {
	s := &Server{
		port:             port,
		metricsCollector: NewMetricsCollector(),
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

	log.Printf("Starting Admin server on port %d", s.port)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully stops the HTTP admin server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down Admin server...")
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
