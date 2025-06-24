package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
)

// ServiceDiscoveryAPI handles Cloud Map API operations
type ServiceDiscoveryAPI struct {
	manager   servicediscovery.Manager
	region    string
	accountID string
}

// NewServiceDiscoveryAPI creates a new ServiceDiscoveryAPI
func NewServiceDiscoveryAPI(manager servicediscovery.Manager, region, accountID string) *ServiceDiscoveryAPI {
	return &ServiceDiscoveryAPI{
		manager:   manager,
		region:    region,
		accountID: accountID,
	}
}

// CreatePrivateDnsNamespaceRequest represents the request for creating a private DNS namespace
type CreatePrivateDnsNamespaceRequest struct {
	Name        string                                `json:"Name"`
	Vpc         string                                `json:"Vpc"`
	Description string                                `json:"Description,omitempty"`
	Tags        []Tag                                 `json:"Tags,omitempty"`
	Properties  *servicediscovery.NamespaceProperties `json:"Properties,omitempty"`
}

// CreatePrivateDnsNamespaceResponse represents the response for creating a private DNS namespace
type CreatePrivateDnsNamespaceResponse struct {
	OperationId string `json:"OperationId"`
}

// CreateServiceDiscoveryServiceRequest represents the request for creating a service
type CreateServiceDiscoveryServiceRequest struct {
	Name                    string                              `json:"Name"`
	NamespaceId             string                              `json:"NamespaceId"`
	Description             string                              `json:"Description,omitempty"`
	DnsConfig               *servicediscovery.DnsConfig         `json:"DnsConfig"`
	HealthCheckConfig       *servicediscovery.HealthCheckConfig `json:"HealthCheckConfig,omitempty"`
	HealthCheckCustomConfig *HealthCheckCustomConfig            `json:"HealthCheckCustomConfig,omitempty"`
	Tags                    []Tag                               `json:"Tags,omitempty"`
}

// HealthCheckCustomConfig represents custom health check configuration
type HealthCheckCustomConfig struct {
	FailureThreshold int32 `json:"FailureThreshold,omitempty"`
}

// CreateServiceDiscoveryServiceResponse represents the response for creating a service
type CreateServiceDiscoveryServiceResponse struct {
	Service *servicediscovery.Service `json:"Service"`
}

// RegisterInstanceRequest represents the request for registering an instance
type RegisterInstanceRequest struct {
	ServiceId  string            `json:"ServiceId"`
	InstanceId string            `json:"InstanceId"`
	Attributes map[string]string `json:"Attributes"`
}

// RegisterInstanceResponse represents the response for registering an instance
type RegisterInstanceResponse struct {
	OperationId string `json:"OperationId"`
}

// DeregisterInstanceRequest represents the request for deregistering an instance
type DeregisterInstanceRequest struct {
	ServiceId  string `json:"ServiceId"`
	InstanceId string `json:"InstanceId"`
}

// DeregisterInstanceResponse represents the response for deregistering an instance
type DeregisterInstanceResponse struct {
	OperationId string `json:"OperationId"`
}

// Tag represents a resource tag
type Tag struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// HandleServiceDiscoveryRequest routes Service Discovery API requests
func (api *ServiceDiscoveryAPI) HandleServiceDiscoveryRequest(w http.ResponseWriter, r *http.Request) {
	// Extract action from X-Amz-Target header
	target := r.Header.Get("X-Amz-Target")
	if target == "" {
		http.Error(w, "Missing X-Amz-Target header", http.StatusBadRequest)
		return
	}

	// Route to appropriate handler
	switch {
	case strings.HasSuffix(target, "CreatePrivateDnsNamespace"):
		api.handleCreatePrivateDnsNamespace(w, r)
	case strings.HasSuffix(target, "CreateService"):
		api.handleCreateService(w, r)
	case strings.HasSuffix(target, "RegisterInstance"):
		api.handleRegisterInstance(w, r)
	case strings.HasSuffix(target, "DeregisterInstance"):
		api.handleDeregisterInstance(w, r)
	case strings.HasSuffix(target, "DiscoverInstances"):
		api.handleDiscoverInstances(w, r)
	default:
		http.Error(w, fmt.Sprintf("Unknown action: %s", target), http.StatusBadRequest)
	}
}

// handleCreatePrivateDnsNamespace handles CreatePrivateDnsNamespace requests
func (api *ServiceDiscoveryAPI) handleCreatePrivateDnsNamespace(w http.ResponseWriter, r *http.Request) {
	var req CreatePrivateDnsNamespaceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create namespace
	namespace, err := api.manager.CreatePrivateDnsNamespace(r.Context(), req.Name, req.Vpc, req.Properties)
	if err != nil {
		klog.Errorf("Failed to create namespace: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return operation ID (namespace creation is synchronous in our implementation)
	resp := CreatePrivateDnsNamespaceResponse{
		OperationId: fmt.Sprintf("op-%s", namespace.ID),
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(resp)
}

// handleCreateService handles CreateService requests for Service Discovery
func (api *ServiceDiscoveryAPI) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var req CreateServiceDiscoveryServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create service
	service, err := api.manager.CreateService(r.Context(), req.Name, req.NamespaceId, req.DnsConfig, req.HealthCheckConfig)
	if err != nil {
		klog.Errorf("Failed to create service: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := CreateServiceDiscoveryServiceResponse{
		Service: service,
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(resp)
}

// handleRegisterInstance handles RegisterInstance requests
func (api *ServiceDiscoveryAPI) handleRegisterInstance(w http.ResponseWriter, r *http.Request) {
	var req RegisterInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Register instance
	instance, err := api.manager.RegisterInstance(r.Context(), req.ServiceId, req.InstanceId, req.Attributes)
	if err != nil {
		klog.Errorf("Failed to register instance: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return operation ID
	resp := RegisterInstanceResponse{
		OperationId: fmt.Sprintf("op-reg-%s", instance.ID),
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(resp)
}

// handleDeregisterInstance handles DeregisterInstance requests
func (api *ServiceDiscoveryAPI) handleDeregisterInstance(w http.ResponseWriter, r *http.Request) {
	var req DeregisterInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Deregister instance
	if err := api.manager.DeregisterInstance(r.Context(), req.ServiceId, req.InstanceId); err != nil {
		klog.Errorf("Failed to deregister instance: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return operation ID
	resp := DeregisterInstanceResponse{
		OperationId: fmt.Sprintf("op-dereg-%s", req.InstanceId),
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(resp)
}

// handleDiscoverInstances handles DiscoverInstances requests
func (api *ServiceDiscoveryAPI) handleDiscoverInstances(w http.ResponseWriter, r *http.Request) {
	var req servicediscovery.DiscoverInstancesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Discover instances
	resp, err := api.manager.DiscoverInstances(r.Context(), &req)
	if err != nil {
		klog.Errorf("Failed to discover instances: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	json.NewEncoder(w).Encode(resp)
}
