package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server represents the HTTP admin server for KECS Control Plane
type Server struct {
	httpServer *http.Server
	port       int
}

// NewServer creates a new admin server instance
func NewServer(port int) *Server {
	return &Server{
		port: port,
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

	// Health check endpoint
	mux.HandleFunc("/healthz", s.handleHealthCheck)

	return mux
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

// handleHealthCheck handles the health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "OK",
		Timestamp: time.Now(),
		Version:   "1.0.0", // TODO: Use actual version from build info
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
