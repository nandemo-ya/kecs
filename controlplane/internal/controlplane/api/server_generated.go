package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	k8s "k8s.io/client-go/kubernetes"
)

// ServerGenerated represents the HTTP API server for KECS Control Plane using generated types
type ServerGenerated struct {
	httpServer        *http.Server
	port              int
	kubeconfig        string
	ecsAPI            ECSAPIGenerated
	storage           storage.Storage
	kindManager       *kubernetes.KindManager
	taskManager       *kubernetes.TaskManager
	region            string
	accountID         string
	webSocketHub      *WebSocketHub
	webUIHandler      *WebUIHandler
	testModeWorker    *TestModeTaskWorker
	localStackManager localstack.Manager
	awsProxyRouter    *AWSProxyRouter
}

// ServerConfigGenerated represents configuration for the API server
type ServerConfigGenerated struct {
	Port              int
	Kubeconfig        string
	Storage           storage.Storage
	K8sClient         k8s.Interface
	WebUIEnabled      bool
	LocalStackEnabled bool
}

// NewServerGenerated creates a new API server using generated types
func NewServerGenerated(cfg ServerConfigGenerated) (*ServerGenerated, error) {
	// Create Kind manager
	kindManager := kubernetes.NewKindManager()

	// Create task manager (simplified for demo)
	taskManager, err := kubernetes.NewTaskManager(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create task manager: %w", err)
	}

	// Create ECS API implementation
	ecsAPI := NewDefaultECSAPIGenerated(cfg.Storage, kindManager)

	// Create WebSocket hub
	webSocketHub := NewWebSocketHub()

	// Create server
	s := &ServerGenerated{
		port:         cfg.Port,
		kubeconfig:   cfg.Kubeconfig,
		ecsAPI:       ecsAPI,
		storage:      cfg.Storage,
		kindManager:  kindManager,
		taskManager:  taskManager,
		region:       getEnvOrDefault("AWS_REGION", "ap-northeast-1"),
		accountID:    getEnvOrDefault("AWS_ACCOUNT_ID", "123456789012"),
		webSocketHub: webSocketHub,
	}

	// Create WebUI handler if enabled (simplified for demo)
	if cfg.WebUIEnabled {
		log.Println("Web UI is enabled")
		// TODO: Implement WebUI handler with proper filesystem
	}

	// Create test mode worker if in test mode (simplified for demo)
	if os.Getenv("KECS_TEST_MODE") == "true" {
		log.Println("Test mode is enabled - starting task status worker")
		// TODO: Implement test mode worker
	}

	return s, nil
}

// Start starts the HTTP server
func (s *ServerGenerated) Start(ctx context.Context) error {
	// Create HTTP mux
	mux := http.NewServeMux()

	// Register ECS routes using generated types
	RegisterECSRoutesGenerated(mux, s.ecsAPI)

	// Register AWS proxy handler for non-ECS services (simplified for demo)
	mux.HandleFunc("/aws/", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement AWS proxy router
		http.Error(w, "AWS proxy not implemented", http.StatusNotImplemented)
	})

	// Register WebSocket endpoint (simplified for demo)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// TODO: Implement WebSocket handler
		http.Error(w, "WebSocket not implemented", http.StatusNotImplemented)
	})

	// Register Web UI if enabled
	if s.webUIHandler != nil {
		// TODO: Implement Web UI handler
	}

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: withLogging(withCORS(mux)),
	}

	// Start WebSocket hub (simplified for demo)
	// TODO: Implement WebSocket hub properly

	// Start server
	log.Printf("Starting API server on port %d", s.port)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	return nil
}

// Stop stops the server
func (s *ServerGenerated) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// withLogging adds request logging middleware
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Log request
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		
		// Call next handler
		next.ServeHTTP(w, r)
		
		// Log duration
		log.Printf("%s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// withCORS adds CORS headers
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Amz-Target, X-Amz-User-Agent, X-Amz-Date")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}