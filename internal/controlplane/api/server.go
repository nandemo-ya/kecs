package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server represents the HTTP API server for KECS Control Plane
type Server struct {
	httpServer *http.Server
	port       int
	kubeconfig string
}

// NewServer creates a new API server instance
func NewServer(port int, kubeconfig string) *Server {
	return &Server{
		port:       port,
		kubeconfig: kubeconfig,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := s.setupRoutes()

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

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("Shutting down API server...")
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Register ECS API endpoints
	s.registerClusterEndpoints(mux)
	s.registerTaskDefinitionEndpoints(mux)
	s.registerTaskEndpoints(mux)
	s.registerServiceEndpoints(mux)
	s.registerContainerInstanceEndpoints(mux)
	s.registerCapacityProviderEndpoints(mux)
	s.registerAccountSettingEndpoints(mux)
	s.registerTagEndpoints(mux)
	s.registerAttributeEndpoints(mux)
	s.registerTaskSetEndpoints(mux)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealthCheck)

	return mux
}

// handleHealthCheck handles the health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
