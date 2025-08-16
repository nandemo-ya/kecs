package api

import (
	"net/http"

	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/sirupsen/logrus"
)

// ServiceDiscoveryAPI handles Cloud Map API operations
type ServiceDiscoveryAPI struct {
	manager   servicediscovery.Manager
	region    string
	accountID string
	router    *generated.Router
	handler   *servicediscovery.Handler
}

// NewServiceDiscoveryAPI creates a new ServiceDiscoveryAPI
func NewServiceDiscoveryAPI(manager servicediscovery.Manager, store storage.Storage, region, accountID string) *ServiceDiscoveryAPI {
	// Create handler
	logger := logrus.New()
	handler := servicediscovery.NewHandler(logger, store, manager)
	
	// Create router with handler
	router := generated.NewRouter(handler)
	
	return &ServiceDiscoveryAPI{
		manager:   manager,
		region:    region,
		accountID: accountID,
		router:    router,
		handler:   handler,
	}
}

// HandleServiceDiscoveryRequest routes Service Discovery API requests
func (api *ServiceDiscoveryAPI) HandleServiceDiscoveryRequest(w http.ResponseWriter, r *http.Request) {
	// Delegate to the generated router
	api.router.Route(w, r)
}

// Tag represents a resource tag (kept for backward compatibility)
type Tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// HealthCheckCustomConfig represents custom health check configuration (kept for backward compatibility)
type HealthCheckCustomConfig struct {
	FailureThreshold int32 `json:"FailureThreshold,omitempty"`
}