package webhook

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Server represents the webhook server
type Server struct {
	server     *http.Server
	podMutator *PodMutator
	tlsConfig  *tls.Config
}

// Config holds webhook server configuration
type Config struct {
	Port      int
	CertFile  string
	KeyFile   string
	Storage   storage.Storage
	Region    string
	AccountID string
}

// NewServer creates a new webhook server
func NewServer(cfg Config) (*Server, error) {
	// Create pod mutator
	podMutator := NewPodMutator(cfg.Storage, cfg.Region, cfg.AccountID)

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate/pods", podMutator.Handle)
	mux.HandleFunc("/health", healthHandler)

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Configure TLS if certificates are provided
	var tlsConfig *tls.Config
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		server.TLSConfig = tlsConfig
	}

	return &Server{
		server:     server,
		podMutator: podMutator,
		tlsConfig:  tlsConfig,
	}, nil
}

// Start starts the webhook server
func (s *Server) Start(ctx context.Context) error {
	logging.Info("Starting webhook server", "addr", s.server.Addr)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		var err error
		if s.tlsConfig != nil {
			err = s.server.ListenAndServeTLS("", "")
		} else {
			logging.Warn("Webhook server running without TLS - only for development")
			err = s.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return s.Shutdown()
	case err := <-errCh:
		return err
	}
}

// Shutdown gracefully shuts down the webhook server
func (s *Server) Shutdown() error {
	logging.Info("Shutting down webhook server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// healthHandler handles health check requests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
