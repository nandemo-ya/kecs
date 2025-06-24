package servicediscovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// Manager manages service discovery operations
type Manager interface {
	// Namespace operations
	CreatePrivateDnsNamespace(ctx context.Context, name, vpc string, properties *NamespaceProperties) (*Namespace, error)
	GetNamespace(ctx context.Context, namespaceID string) (*Namespace, error)
	ListNamespaces(ctx context.Context) ([]*Namespace, error)
	DeleteNamespace(ctx context.Context, namespaceID string) error

	// Service operations
	CreateService(ctx context.Context, name, namespaceID string, dnsConfig *DnsConfig, healthCheck *HealthCheckConfig) (*Service, error)
	GetService(ctx context.Context, serviceID string) (*Service, error)
	DeleteService(ctx context.Context, serviceID string) error

	// Instance operations
	RegisterInstance(ctx context.Context, serviceID string, instanceID string, attributes map[string]string) (*Instance, error)
	DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error
	DiscoverInstances(ctx context.Context, req *DiscoverInstancesRequest) (*DiscoverInstancesResponse, error)
	UpdateInstanceHealthStatus(ctx context.Context, serviceID, instanceID string, status string) error
}

// manager implements the Manager interface
type manager struct {
	kubeClient kubernetes.Interface
	region     string
	accountID  string

	// In-memory storage for namespaces, services, and instances
	mu         sync.RWMutex
	namespaces map[string]*Namespace
	services   map[string]*Service
	instances  map[string]map[string]*Instance // serviceID -> instanceID -> Instance

	// DNS namespace to Kubernetes namespace mapping
	dnsToK8sNamespace map[string]string
}

// NewManager creates a new service discovery manager
func NewManager(kubeClient kubernetes.Interface, region, accountID string) Manager {
	return &manager{
		kubeClient:        kubeClient,
		region:            region,
		accountID:         accountID,
		namespaces:        make(map[string]*Namespace),
		services:          make(map[string]*Service),
		instances:         make(map[string]map[string]*Instance),
		dnsToK8sNamespace: make(map[string]string),
	}
}

// CreatePrivateDnsNamespace creates a private DNS namespace
func (m *manager) CreatePrivateDnsNamespace(ctx context.Context, name, vpc string, properties *NamespaceProperties) (*Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if namespace already exists
	for _, ns := range m.namespaces {
		if ns.Name == name {
			return nil, fmt.Errorf("namespace with name %s already exists", name)
		}
	}

	// Generate IDs
	namespaceID := fmt.Sprintf("ns-%s", generateID())
	arn := fmt.Sprintf("arn:aws:servicediscovery:%s:%s:namespace/%s", m.region, m.accountID, namespaceID)

	// Create namespace
	namespace := &Namespace{
		ID:           namespaceID,
		ARN:          arn,
		Name:         name,
		Type:         "DNS_PRIVATE",
		ServiceCount: 0,
		CreatedAt:    time.Now(),
		Properties:   properties,
	}

	// If no properties provided, create default
	if namespace.Properties == nil {
		namespace.Properties = &NamespaceProperties{
			DnsProperties: &DnsProperties{
				HostedZoneId: fmt.Sprintf("Z%s", generateID()),
			},
		}
	}

	m.namespaces[namespaceID] = namespace

	// Map DNS namespace to Kubernetes namespace
	// Use the cluster name as Kubernetes namespace (e.g., "local.ecs" -> "default")
	k8sNamespace := "default"
	if name != "local.ecs" {
		// Extract first part as namespace name
		k8sNamespace = extractK8sNamespace(name)
	}
	m.dnsToK8sNamespace[name] = k8sNamespace

	klog.Infof("Created private DNS namespace: %s (ID: %s)", name, namespaceID)

	return namespace, nil
}

// GetNamespace retrieves a namespace by ID
func (m *manager) GetNamespace(ctx context.Context, namespaceID string) (*Namespace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	namespace, exists := m.namespaces[namespaceID]
	if !exists {
		return nil, fmt.Errorf("namespace not found: %s", namespaceID)
	}

	return namespace, nil
}

// ListNamespaces lists all namespaces
func (m *manager) ListNamespaces(ctx context.Context) ([]*Namespace, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	namespaces := make([]*Namespace, 0, len(m.namespaces))
	for _, ns := range m.namespaces {
		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}

// DeleteNamespace deletes a namespace
func (m *manager) DeleteNamespace(ctx context.Context, namespaceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	namespace, exists := m.namespaces[namespaceID]
	if !exists {
		return fmt.Errorf("namespace not found: %s", namespaceID)
	}

	// Check if namespace has services
	if namespace.ServiceCount > 0 {
		return fmt.Errorf("namespace has %d services, cannot delete", namespace.ServiceCount)
	}

	delete(m.namespaces, namespaceID)
	delete(m.dnsToK8sNamespace, namespace.Name)

	klog.Infof("Deleted namespace: %s", namespaceID)

	return nil
}

// CreateService creates a service in a namespace
func (m *manager) CreateService(ctx context.Context, name, namespaceID string, dnsConfig *DnsConfig, healthCheck *HealthCheckConfig) (*Service, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify namespace exists
	namespace, exists := m.namespaces[namespaceID]
	if !exists {
		return nil, fmt.Errorf("namespace not found: %s", namespaceID)
	}

	// Check if service already exists in namespace
	for _, svc := range m.services {
		if svc.Name == name && svc.NamespaceID == namespaceID {
			return nil, fmt.Errorf("service %s already exists in namespace %s", name, namespaceID)
		}
	}

	// Generate IDs
	serviceID := fmt.Sprintf("srv-%s", generateID())
	arn := fmt.Sprintf("arn:aws:servicediscovery:%s:%s:service/%s", m.region, m.accountID, serviceID)

	// Create service
	service := &Service{
		ID:            serviceID,
		ARN:           arn,
		Name:          name,
		NamespaceID:   namespaceID,
		InstanceCount: 0,
		DnsConfig:     dnsConfig,
		HealthCheck:   healthCheck,
		CreatedAt:     time.Now(),
	}

	m.services[serviceID] = service
	m.instances[serviceID] = make(map[string]*Instance)

	// Update namespace service count
	namespace.ServiceCount++

	klog.Infof("Created service: %s in namespace %s (ID: %s)", name, namespaceID, serviceID)

	return service, nil
}

// GetService retrieves a service by ID
func (m *manager) GetService(ctx context.Context, serviceID string) (*Service, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	service, exists := m.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	return service, nil
}

// DeleteService deletes a service
func (m *manager) DeleteService(ctx context.Context, serviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	// Check if service has instances
	if service.InstanceCount > 0 {
		return fmt.Errorf("service has %d instances, cannot delete", service.InstanceCount)
	}

	// Update namespace service count
	if namespace, exists := m.namespaces[service.NamespaceID]; exists {
		namespace.ServiceCount--
	}

	delete(m.services, serviceID)
	delete(m.instances, serviceID)

	klog.Infof("Deleted service: %s", serviceID)

	return nil
}

// RegisterInstance registers an instance with a service
func (m *manager) RegisterInstance(ctx context.Context, serviceID string, instanceID string, attributes map[string]string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	// Check if instance already exists
	if _, exists := m.instances[serviceID][instanceID]; exists {
		return nil, fmt.Errorf("instance %s already registered", instanceID)
	}

	// Create instance
	instance := &Instance{
		ID:           instanceID,
		ServiceID:    serviceID,
		Attributes:   attributes,
		HealthStatus: "UNKNOWN",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Add required attributes
	if instance.Attributes == nil {
		instance.Attributes = make(map[string]string)
	}

	// AWS SDK compatibility attributes
	instance.Attributes["AWS_INSTANCE_ID"] = instanceID

	m.instances[serviceID][instanceID] = instance
	service.InstanceCount++

	klog.Infof("Registered instance %s with service %s", instanceID, serviceID)

	// Create/update Kubernetes Endpoints
	if err := m.updateKubernetesEndpoints(ctx, service, m.instances[serviceID]); err != nil {
		klog.Errorf("Failed to update Kubernetes endpoints: %v", err)
		// Don't fail the registration, just log the error
	}

	return instance, nil
}

// DeregisterInstance deregisters an instance from a service
func (m *manager) DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	if _, exists := m.instances[serviceID][instanceID]; !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	delete(m.instances[serviceID], instanceID)
	service.InstanceCount--

	klog.Infof("Deregistered instance %s from service %s", instanceID, serviceID)

	// Update Kubernetes Endpoints
	if err := m.updateKubernetesEndpoints(ctx, service, m.instances[serviceID]); err != nil {
		klog.Errorf("Failed to update Kubernetes endpoints: %v", err)
	}

	return nil
}

// DiscoverInstances discovers instances for a service
func (m *manager) DiscoverInstances(ctx context.Context, req *DiscoverInstancesRequest) (*DiscoverInstancesResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find namespace by name
	var namespaceID string
	for id, ns := range m.namespaces {
		if ns.Name == req.NamespaceName {
			namespaceID = id
			break
		}
	}

	if namespaceID == "" {
		return nil, fmt.Errorf("namespace not found: %s", req.NamespaceName)
	}

	// Find service by name in namespace
	var serviceID string
	for id, svc := range m.services {
		if svc.Name == req.ServiceName && svc.NamespaceID == namespaceID {
			serviceID = id
			break
		}
	}

	if serviceID == "" {
		return nil, fmt.Errorf("service not found: %s in namespace %s", req.ServiceName, req.NamespaceName)
	}

	// Get instances
	instances := make([]InstanceSummary, 0)
	for _, instance := range m.instances[serviceID] {
		// Filter by health status if specified
		if req.HealthStatus != "" && req.HealthStatus != "ALL" {
			if instance.HealthStatus != req.HealthStatus {
				continue
			}
		}

		summary := InstanceSummary{
			InstanceId:    instance.ID,
			NamespaceName: req.NamespaceName,
			ServiceName:   req.ServiceName,
			HealthStatus:  instance.HealthStatus,
			Attributes:    instance.Attributes,
		}

		instances = append(instances, summary)

		// Apply max results limit
		if req.MaxResults > 0 && int32(len(instances)) >= req.MaxResults {
			break
		}
	}

	return &DiscoverInstancesResponse{
		Instances: instances,
	}, nil
}

// UpdateInstanceHealthStatus updates the health status of an instance
func (m *manager) UpdateInstanceHealthStatus(ctx context.Context, serviceID, instanceID string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.services[serviceID]; !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	instance, exists := m.instances[serviceID][instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	instance.HealthStatus = status
	instance.UpdatedAt = time.Now()

	klog.V(2).Infof("Updated health status for instance %s to %s", instanceID, status)

	return nil
}
