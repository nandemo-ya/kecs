package servicediscovery

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/route53"
)

// Manager manages service discovery operations
type Manager interface {
	// Namespace operations
	CreateNamespace(ctx context.Context, namespace *Namespace) error
	CreatePrivateDnsNamespace(ctx context.Context, name, vpc string, properties *NamespaceProperties) (*Namespace, error)
	GetNamespace(ctx context.Context, namespaceID string) (*Namespace, error)
	ListNamespaces(ctx context.Context) ([]*Namespace, error)
	DeleteNamespace(ctx context.Context, namespaceID string) error

	// Service operations
	CreateService(ctx context.Context, service *Service) error
	GetService(ctx context.Context, serviceID string) (*Service, error)
	ListServices(ctx context.Context, namespaceID string) ([]*Service, error)
	DeleteService(ctx context.Context, serviceID string) error

	// Instance operations
	RegisterInstance(ctx context.Context, instance *Instance) error
	DeregisterInstance(ctx context.Context, serviceID string, instanceID string) error
	ListInstances(ctx context.Context, serviceID string) ([]*Instance, error)
	DiscoverInstances(ctx context.Context, namespaceName, serviceName string) ([]*Instance, error)
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

	// Route53 integration (optional)
	route53Manager *route53.Manager
}

// NewManager creates a new service discovery manager
func NewManager(kubeClient kubernetes.Interface, region, accountID string) Manager {
	mgr := &manager{
		kubeClient:        kubeClient,
		region:            region,
		accountID:         accountID,
		namespaces:        make(map[string]*Namespace),
		services:          make(map[string]*Service),
		instances:         make(map[string]map[string]*Instance),
		dnsToK8sNamespace: make(map[string]string),
	}

	// Initialize Route53 integration if LocalStack endpoint is configured
	localstackEndpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	if localstackEndpoint == "" {
		localstackEndpoint = os.Getenv("AWS_ENDPOINT_URL")
	}

	if localstackEndpoint != "" {
		ctx := context.Background()
		r53Client, err := route53.NewClient(ctx, localstackEndpoint)
		if err != nil {
			logging.Warn("Failed to initialize Route53 client", "error", err)
		} else {
			// Default VPC configuration for LocalStack
			defaultVPC := &route53.VPCConfig{
				VPCID:  "vpc-default",
				Region: region,
			}
			mgr.route53Manager = route53.NewManager(r53Client, defaultVPC)
			logging.Info("Route53 integration enabled", "endpoint", localstackEndpoint)
		}
	}

	return mgr
}

// CreateNamespace creates a namespace (generic method for any type)
func (m *manager) CreateNamespace(ctx context.Context, namespace *Namespace) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if namespace already exists
	for _, ns := range m.namespaces {
		if ns.Name == namespace.Name {
			logging.Info("Namespace already exists, updating existing namespace",
				"name", namespace.Name,
				"existingID", ns.ID,
				"newID", namespace.ID)
			// Update the existing namespace's ID in our records if needed
			// This handles the case where controlplane restarts and tries to recreate
			namespace.ID = ns.ID
			namespace.ARN = ns.ARN
			namespace.CreatedAt = ns.CreatedAt
			m.namespaces[ns.ID] = namespace
			return nil
		}
	}

	// Generate ARN if not set
	if namespace.ARN == "" {
		namespace.ARN = fmt.Sprintf("arn:aws:servicediscovery:%s:%s:namespace/%s", m.region, m.accountID, namespace.ID)
	}

	// Set creation time if not set
	if namespace.CreatedAt.IsZero() {
		namespace.CreatedAt = time.Now()
	}
	namespace.UpdatedAt = time.Now()

	// For DNS namespaces, create Route53 hosted zone if manager is available
	if m.route53Manager != nil && (namespace.Type == NamespaceTypeDNSPrivate || namespace.Type == NamespaceTypeDNSPublic) {
		// Determine if it's private or public
		isPrivate := namespace.Type == NamespaceTypeDNSPrivate

		// Create hosted zone
		hostedZoneID, err := m.route53Manager.CreateHostedZone(ctx, namespace.Name, isPrivate)
		if err != nil {
			return fmt.Errorf("failed to create Route53 hosted zone: %w", err)
		}
		namespace.HostedZoneID = hostedZoneID

		logging.Info("Created Route53 hosted zone for namespace",
			"namespace", namespace.Name,
			"hostedZoneID", hostedZoneID,
			"type", namespace.Type)
	}

	// Store namespace
	m.namespaces[namespace.ID] = namespace

	// Update CoreDNS configuration for DNS namespaces
	if namespace.Type == NamespaceTypeDNSPrivate || namespace.Type == NamespaceTypeDNSPublic {
		if err := m.updateCoreDNSConfig(ctx, namespace); err != nil {
			logging.Warn("Failed to update CoreDNS configuration", "namespace", namespace.Name, "error", err)
			// Don't fail namespace creation for CoreDNS update issues
		}
	}

	logging.Info("Created namespace", "namespaceID", namespace.ID, "name", namespace.Name, "type", namespace.Type)

	return nil
}

// CreatePrivateDnsNamespace creates a private DNS namespace
func (m *manager) CreatePrivateDnsNamespace(ctx context.Context, name, vpc string, properties *NamespaceProperties) (*Namespace, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if namespace already exists
	for _, ns := range m.namespaces {
		if ns.Name == name {
			logging.Info("Namespace already exists, returning existing namespace",
				"name", name,
				"id", ns.ID)
			return ns, nil
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

	// Create Route53 hosted zone if integration is enabled
	if m.route53Manager != nil {
		_, err := m.route53Manager.CreateNamespaceZone(ctx, name)
		if err != nil {
			logging.Warn("Failed to create Route53 hosted zone", "namespace", name, "error", err)
			// Continue anyway - Kubernetes DNS will still work
		}
	}

	logging.Info("Created private DNS namespace", "name", name, "id", namespaceID)

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

	// Remove CoreDNS configuration
	if namespace.Type == NamespaceTypeDNSPrivate || namespace.Type == NamespaceTypeDNSPublic {
		if err := m.removeCoreDNSConfig(ctx, namespaceID); err != nil {
			logging.Warn("Failed to remove CoreDNS configuration", "namespace", namespace.Name, "error", err)
		}
	}

	// Delete Route53 hosted zone if integration is enabled
	if m.route53Manager != nil {
		err := m.route53Manager.DeleteNamespaceZone(ctx, namespace.Name)
		if err != nil {
			logging.Warn("Failed to delete Route53 hosted zone", "namespace", namespace.Name, "error", err)
		}
	}

	logging.Info("Deleted namespace", "namespaceID", namespaceID)

	return nil
}

// CreateService creates a service in a namespace
func (m *manager) CreateService(ctx context.Context, service *Service) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Verify namespace exists
	namespace, exists := m.namespaces[service.NamespaceID]
	if !exists {
		return fmt.Errorf("namespace not found: %s", service.NamespaceID)
	}

	// Check if service already exists in namespace
	for _, svc := range m.services {
		if svc.Name == service.Name && svc.NamespaceID == service.NamespaceID {
			return fmt.Errorf("service %s already exists in namespace %s", service.Name, service.NamespaceID)
		}
	}

	// Generate ARN if not set
	if service.ARN == "" {
		service.ARN = fmt.Sprintf("arn:aws:servicediscovery:%s:%s:service/%s", m.region, m.accountID, service.ID)
	}

	// Set creation time if not set
	if service.CreatedAt.IsZero() {
		service.CreatedAt = time.Now()
	}
	service.UpdatedAt = time.Now()

	m.services[service.ID] = service
	m.instances[service.ID] = make(map[string]*Instance)

	// Update namespace service count
	namespace.ServiceCount++

	// Create DNS alias for the service
	if err := m.createServiceDNSAlias(ctx, namespace, service); err != nil {
		logging.Warn("Failed to create DNS alias for service", "service", service.Name, "error", err)
		// Don't fail service creation for DNS alias issues
	}

	logging.Info("Created service in namespace", "name", service.Name, "namespaceID", service.NamespaceID, "serviceID", service.ID)

	return nil
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

// ListServices lists services in a namespace
func (m *manager) ListServices(ctx context.Context, namespaceID string) ([]*Service, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var services []*Service
	for _, svc := range m.services {
		if namespaceID == "" || svc.NamespaceID == namespaceID {
			services = append(services, svc)
		}
	}

	return services, nil
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
	namespace, exists := m.namespaces[service.NamespaceID]
	if exists {
		namespace.ServiceCount--

		// Remove DNS alias for the service
		if err := m.removeServiceDNSAlias(ctx, namespace, service); err != nil {
			logging.Warn("Failed to remove DNS alias for service", "service", service.Name, "error", err)
		}
	}

	delete(m.services, serviceID)
	delete(m.instances, serviceID)

	logging.Info("Deleted service", "serviceID", serviceID)

	return nil
}

// RegisterInstance registers an instance with a service
func (m *manager) RegisterInstance(ctx context.Context, instance *Instance) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[instance.ServiceID]
	if !exists {
		logging.Debug("Service not found during instance registration",
			"serviceID", instance.ServiceID,
			"instanceID", instance.ID,
			"availableServices", len(m.services))
		return fmt.Errorf("service not found: %s", instance.ServiceID)
	}

	// Initialize instance map for service if needed
	if m.instances[instance.ServiceID] == nil {
		m.instances[instance.ServiceID] = make(map[string]*Instance)
	}

	// Check if instance already exists
	if _, exists := m.instances[instance.ServiceID][instance.ID]; exists {
		logging.Debug("Instance already registered, updating",
			"instanceID", instance.ID,
			"serviceID", instance.ServiceID)
		// Update existing instance instead of failing
		existingInstance := m.instances[instance.ServiceID][instance.ID]
		existingInstance.UpdatedAt = time.Now()
		existingInstance.HealthStatus = instance.HealthStatus
		if instance.Attributes != nil {
			for k, v := range instance.Attributes {
				existingInstance.Attributes[k] = v
			}
		}
		return nil
	}

	// Set creation time if not set
	if instance.CreatedAt.IsZero() {
		instance.CreatedAt = time.Now()
	}
	instance.UpdatedAt = time.Now()

	// Add required attributes
	if instance.Attributes == nil {
		instance.Attributes = make(map[string]string)
	}

	// AWS SDK compatibility attributes
	instance.Attributes["AWS_INSTANCE_ID"] = instance.ID

	// Set default health status if not set
	if instance.HealthStatus == "" {
		instance.HealthStatus = "UNKNOWN"
	}

	m.instances[instance.ServiceID][instance.ID] = instance
	service.InstanceCount++

	logging.Info("Registered instance with service",
		"instanceID", instance.ID,
		"serviceID", instance.ServiceID,
		"ip", instance.Attributes["AWS_INSTANCE_IPV4"],
		"port", instance.Attributes["AWS_INSTANCE_PORT"])

	// Create/update Kubernetes Endpoints
	if err := m.updateKubernetesEndpoints(ctx, service, m.instances[instance.ServiceID]); err != nil {
		logging.Error("Failed to update Kubernetes endpoints", "error", err)
		// Don't fail the registration, just log the error
	}

	// Update Route53 records if integration is enabled
	if m.route53Manager != nil {
		namespace := m.namespaces[service.NamespaceID]
		if namespace != nil {
			// Collect all IPs from instances
			ips := m.collectInstanceIPs(m.instances[instance.ServiceID])
			if len(ips) > 0 {
				err := m.route53Manager.RegisterService(ctx, namespace.Name, service.Name, ips)
				if err != nil {
					logging.Warn("Failed to update Route53 records", "service", service.Name, "error", err)
				}
			}
		}
	}

	return nil
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

	logging.Info("Deregistered instance from service", "instanceID", instanceID, "serviceID", serviceID)

	// Update Kubernetes Endpoints
	if err := m.updateKubernetesEndpoints(ctx, service, m.instances[serviceID]); err != nil {
		logging.Error("Failed to update Kubernetes endpoints", "error", err)
	}

	// Update Route53 records if integration is enabled
	if m.route53Manager != nil {
		namespace := m.namespaces[service.NamespaceID]
		if namespace != nil {
			// Collect remaining IPs from instances
			ips := m.collectInstanceIPs(m.instances[serviceID])
			var err error
			if len(ips) > 0 {
				err = m.route53Manager.RegisterService(ctx, namespace.Name, service.Name, ips)
			} else {
				// No more instances, remove the service from Route53
				err = m.route53Manager.DeregisterService(ctx, namespace.Name, service.Name)
			}
			if err != nil {
				logging.Warn("Failed to update Route53 records", "service", service.Name, "error", err)
			}
		}
	}

	return nil
}

// ListInstances lists all instances for a service
func (m *manager) ListInstances(ctx context.Context, serviceID string) ([]*Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if service exists
	if _, exists := m.services[serviceID]; !exists {
		return nil, fmt.Errorf("service not found: %s", serviceID)
	}

	// Get all instances for the service
	instances := []*Instance{}
	if serviceInstances, exists := m.instances[serviceID]; exists {
		for _, instance := range serviceInstances {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// DiscoverInstances discovers instances for a service
func (m *manager) DiscoverInstances(ctx context.Context, namespaceName, serviceName string) ([]*Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Find namespace by name
	var namespaceID string
	for id, ns := range m.namespaces {
		if ns.Name == namespaceName {
			namespaceID = id
			break
		}
	}

	if namespaceID == "" {
		return nil, fmt.Errorf("namespace not found: %s", namespaceName)
	}

	// Find service by name in namespace
	var serviceID string
	for id, svc := range m.services {
		if svc.Name == serviceName && svc.NamespaceID == namespaceID {
			serviceID = id
			break
		}
	}

	if serviceID == "" {
		return nil, fmt.Errorf("service not found: %s in namespace %s", serviceName, namespaceName)
	}

	// Get only healthy instances for DNS resolution
	var instances []*Instance
	for _, instance := range m.instances[serviceID] {
		// Only include healthy instances in DNS responses
		// This implements ECS behavior where unhealthy containers are excluded from Service Discovery
		if instance.HealthStatus == "HEALTHY" || instance.HealthStatus == "" {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

// UpdateInstanceHealthStatus updates the health status of an instance
func (m *manager) UpdateInstanceHealthStatus(ctx context.Context, serviceID, instanceID string, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	service, exists := m.services[serviceID]
	if !exists {
		return fmt.Errorf("service not found: %s", serviceID)
	}

	instance, exists := m.instances[serviceID][instanceID]
	if !exists {
		return fmt.Errorf("instance %s not found", instanceID)
	}

	// Only update if status actually changed
	if instance.HealthStatus == status {
		return nil
	}

	previousStatus := instance.HealthStatus
	instance.HealthStatus = status
	instance.UpdatedAt = time.Now()

	logging.Info("Updated health status for instance",
		"instanceID", instanceID,
		"previousStatus", previousStatus,
		"newStatus", status)

	// Update Kubernetes Endpoints to reflect health status change
	// This ensures DNS responses are updated immediately
	if err := m.updateKubernetesEndpoints(ctx, service, m.instances[serviceID]); err != nil {
		logging.Error("Failed to update Kubernetes endpoints after health status change",
			"serviceID", serviceID,
			"instanceID", instanceID,
			"error", err)
		// Don't fail the health status update, just log the error
	}

	// Update Route53 records if integration is enabled
	if m.route53Manager != nil {
		namespace := m.namespaces[service.NamespaceID]
		if namespace != nil {
			// Collect only healthy instance IPs
			var ips []string
			for _, inst := range m.instances[serviceID] {
				if inst.HealthStatus == "HEALTHY" || inst.HealthStatus == "" {
					if ip := inst.Attributes["AWS_INSTANCE_IPV4"]; ip != "" {
						ips = append(ips, ip)
					}
				}
			}

			if err := m.route53Manager.RegisterService(ctx, namespace.Name, service.Name, ips); err != nil {
				logging.Warn("Failed to update Route53 records after health status change",
					"service", service.Name,
					"error", err)
			}
		}
	}

	return nil
}
