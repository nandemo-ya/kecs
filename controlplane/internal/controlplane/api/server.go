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
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Server represents the HTTP API server for KECS Control Plane
type Server struct {
	httpServer   *http.Server
	port         int
	kubeconfig   string
	ecsAPI       generated.ECSAPIInterface
	storage      storage.Storage
	kindManager  *kubernetes.KindManager
	taskManager  *kubernetes.TaskManager
	region       string
	accountID    string
	webSocketHub *WebSocketHub
	webUIHandler *WebUIHandler
}

// NewServer creates a new API server instance
func NewServer(port int, kubeconfig string, storage storage.Storage) (*Server, error) {
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
	if os.Getenv("KECS_TEST_MODE") != "true" {
		kindManager = kubernetes.NewKindManager()
	} else {
		log.Println("Running in test mode - Kubernetes operations will be simulated")
	}

	s := &Server{
		port:         port,
		kubeconfig:   kubeconfig,
		region:       "ap-northeast-1", // Default region
		accountID:    "123456789012",   // Default account ID
		ecsAPI:       NewDefaultECSAPI(storage, kindManager),
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

	return s, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Start WebSocket hub
	ctx := context.Background()
	go s.webSocketHub.Run(ctx)

	router := s.setupRoutes()

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
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// AWS ECS API endpoint (AWS CLI format)
	mux.HandleFunc("/", generated.HandleECSRequest(s.ecsAPI))

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealthCheck)

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
