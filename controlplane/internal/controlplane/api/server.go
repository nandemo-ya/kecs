package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// Server represents the HTTP API server for KECS Control Plane
type Server struct {
	httpServer  *http.Server
	port        int
	kubeconfig  string
	ecsService  generated.ECSServiceInterface
	storage     storage.Storage
	kindManager *kubernetes.KindManager
	region      string
	accountID   string
}

// NewServer creates a new API server instance
func NewServer(port int, kubeconfig string, storage storage.Storage) *Server {
	return &Server{
		port:        port,
		kubeconfig:  kubeconfig,
		region:      "ap-northeast-1", // Default region
		accountID:   "123456789012",   // Default account ID
		ecsService:  generated.NewECSServiceWithStorage(storage),
		storage:     storage,
		kindManager: kubernetes.NewKindManager(),
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

	// AWS ECS API endpoint (AWS CLI format)
	mux.HandleFunc("/", s.handleECSRequest)

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealthCheck)

	return mux
}

// registerGeneratedECSEndpoints registers all generated ECS API endpoints
func (s *Server) registerGeneratedECSEndpoints(mux *http.ServeMux) {
	// Cluster operations
	mux.HandleFunc("/v1/createcluster", generated.HandleCreateCluster(s.ecsService))
	mux.HandleFunc("/v1/deletecluster", generated.HandleDeleteCluster(s.ecsService))
	mux.HandleFunc("/v1/describeclusters", generated.HandleDescribeClusters(s.ecsService))
	mux.HandleFunc("/v1/listclusters", generated.HandleListClusters(s.ecsService))
	mux.HandleFunc("/v1/updatecluster", generated.HandleUpdateCluster(s.ecsService))

	// Task operations
	mux.HandleFunc("/v1/runtask", generated.HandleRunTask(s.ecsService))
	mux.HandleFunc("/v1/starttask", generated.HandleStartTask(s.ecsService))
	mux.HandleFunc("/v1/stoptask", generated.HandleStopTask(s.ecsService))
	mux.HandleFunc("/v1/describetasks", generated.HandleDescribeTasks(s.ecsService))
	mux.HandleFunc("/v1/listtasks", generated.HandleListTasks(s.ecsService))

	// Task Definition operations
	mux.HandleFunc("/v1/registertaskdefinition", generated.HandleRegisterTaskDefinition(s.ecsService))
	mux.HandleFunc("/v1/deregistertaskdefinition", generated.HandleDeregisterTaskDefinition(s.ecsService))
	mux.HandleFunc("/v1/describetaskdefinition", generated.HandleDescribeTaskDefinition(s.ecsService))
	mux.HandleFunc("/v1/listtaskdefinitions", generated.HandleListTaskDefinitions(s.ecsService))
	mux.HandleFunc("/v1/listtaskdefinitionfamilies", generated.HandleListTaskDefinitionFamilies(s.ecsService))

	// Service operations
	mux.HandleFunc("/v1/createservice", generated.HandleCreateService(s.ecsService))
	mux.HandleFunc("/v1/deleteservice", generated.HandleDeleteService(s.ecsService))
	mux.HandleFunc("/v1/describeservices", generated.HandleDescribeServices(s.ecsService))
	mux.HandleFunc("/v1/listservices", generated.HandleListServices(s.ecsService))
	mux.HandleFunc("/v1/updateservice", generated.HandleUpdateService(s.ecsService))

	// Container Instance operations
	mux.HandleFunc("/v1/registercontainerinstance", generated.HandleRegisterContainerInstance(s.ecsService))
	mux.HandleFunc("/v1/deregistercontainerinstance", generated.HandleDeregisterContainerInstance(s.ecsService))
	mux.HandleFunc("/v1/describecontainerinstances", generated.HandleDescribeContainerInstances(s.ecsService))
	mux.HandleFunc("/v1/listcontainerinstances", generated.HandleListContainerInstances(s.ecsService))

	// Capacity Provider operations
	mux.HandleFunc("/v1/createcapacityprovider", generated.HandleCreateCapacityProvider(s.ecsService))
	mux.HandleFunc("/v1/deletecapacityprovider", generated.HandleDeleteCapacityProvider(s.ecsService))
	mux.HandleFunc("/v1/describecapacityproviders", generated.HandleDescribeCapacityProviders(s.ecsService))
	mux.HandleFunc("/v1/updatecapacityprovider", generated.HandleUpdateCapacityProvider(s.ecsService))

	// Account Settings operations
	mux.HandleFunc("/v1/putaccountsetting", generated.HandlePutAccountSetting(s.ecsService))
	mux.HandleFunc("/v1/putaccountsettingdefault", generated.HandlePutAccountSettingDefault(s.ecsService))
	mux.HandleFunc("/v1/deleteaccountsetting", generated.HandleDeleteAccountSetting(s.ecsService))
	mux.HandleFunc("/v1/listaccountsettings", generated.HandleListAccountSettings(s.ecsService))

	// Tag operations
	mux.HandleFunc("/v1/tagresource", generated.HandleTagResource(s.ecsService))
	mux.HandleFunc("/v1/untagresource", generated.HandleUntagResource(s.ecsService))
	mux.HandleFunc("/v1/listtagsforresource", generated.HandleListTagsForResource(s.ecsService))

	// Additional operations
	mux.HandleFunc("/v1/executecommand", generated.HandleExecuteCommand(s.ecsService))
	mux.HandleFunc("/v1/discoverpollendpoint", generated.HandleDiscoverPollEndpoint(s.ecsService))
}

// handleHealthCheck handles the health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
