package servicediscovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/sirupsen/logrus"
)

// Handler implements the Service Discovery API
type Handler struct {
	logger  *logrus.Logger
	store   storage.Storage
	manager Manager
}

// NewHandler creates a new Service Discovery API handler
func NewHandler(logger *logrus.Logger, store storage.Storage, manager Manager) *Handler {
	return &Handler{
		logger:  logger,
		store:   store,
		manager: manager,
	}
}

// CreatePrivateDnsNamespace creates a private DNS namespace
func (h *Handler) CreatePrivateDnsNamespace(ctx context.Context, input *generated.CreatePrivateDnsNamespaceRequest) (*generated.CreatePrivateDnsNamespaceResponse, error) {
	h.logger.WithField("name", input.Name).Info("Creating private DNS namespace")

	// Generate namespace ID
	namespaceID := fmt.Sprintf("ns-%s", uuid.New().String())
	operationID := fmt.Sprintf("op-%s", uuid.New().String())

	// Store namespace in database
	namespace := &Namespace{
		ID:          namespaceID,
		Name:        input.Name,
		Type:        NamespaceTypeDNSPrivate,
		VPC:         input.Vpc,
		CreatedAt:   time.Now(),
		Description: stringValue(input.Description),
	}

	// Note: PrivateDnsPropertiesMutable doesn't have HostedZoneId field
	// The hosted zone will be created by the manager

	if err := h.manager.CreateNamespace(ctx, namespace); err != nil {
		h.logger.WithError(err).Error("Failed to create namespace")
		return nil, err
	}

	// Store operation
	operation := &Operation{
		ID:        operationID,
		Type:      "CREATE_NAMESPACE",
		Status:    "SUCCESS",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Targets: map[string]string{
			"NAMESPACE": namespaceID,
		},
	}

	if err := h.storeOperation(ctx, operation); err != nil {
		h.logger.WithError(err).Error("Failed to store operation")
	}

	return &generated.CreatePrivateDnsNamespaceResponse{
		OperationId: &operationID,
	}, nil
}

// CreatePublicDnsNamespace creates a public DNS namespace
func (h *Handler) CreatePublicDnsNamespace(ctx context.Context, input *generated.CreatePublicDnsNamespaceRequest) (*generated.CreatePublicDnsNamespaceResponse, error) {
	h.logger.WithField("name", input.Name).Info("Creating public DNS namespace")

	// Generate namespace ID
	namespaceID := fmt.Sprintf("ns-%s", uuid.New().String())
	operationID := fmt.Sprintf("op-%s", uuid.New().String())

	// Store namespace in database
	namespace := &Namespace{
		ID:          namespaceID,
		Name:        input.Name,
		Type:        NamespaceTypeDNSPublic,
		CreatedAt:   time.Now(),
		Description: stringValue(input.Description),
	}

	// Note: PublicDnsPropertiesMutable doesn't have HostedZoneId field
	// The hosted zone will be created by the manager

	if err := h.manager.CreateNamespace(ctx, namespace); err != nil {
		h.logger.WithError(err).Error("Failed to create namespace")
		return nil, err
	}

	// Store operation
	operation := &Operation{
		ID:        operationID,
		Type:      "CREATE_NAMESPACE",
		Status:    "SUCCESS",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Targets: map[string]string{
			"NAMESPACE": namespaceID,
		},
	}

	if err := h.storeOperation(ctx, operation); err != nil {
		h.logger.WithError(err).Error("Failed to store operation")
	}

	return &generated.CreatePublicDnsNamespaceResponse{
		OperationId: &operationID,
	}, nil
}

// CreateHttpNamespace creates an HTTP namespace
func (h *Handler) CreateHttpNamespace(ctx context.Context, input *generated.CreateHttpNamespaceRequest) (*generated.CreateHttpNamespaceResponse, error) {
	h.logger.WithField("name", input.Name).Info("Creating HTTP namespace")

	// Generate namespace ID
	namespaceID := fmt.Sprintf("ns-%s", uuid.New().String())
	operationID := fmt.Sprintf("op-%s", uuid.New().String())

	// Store namespace in database
	namespace := &Namespace{
		ID:          namespaceID,
		Name:        input.Name,
		Type:        NamespaceTypeHTTP,
		CreatedAt:   time.Now(),
		Description: stringValue(input.Description),
	}

	if err := h.manager.CreateNamespace(ctx, namespace); err != nil {
		h.logger.WithError(err).Error("Failed to create namespace")
		return nil, err
	}

	// Store operation
	operation := &Operation{
		ID:        operationID,
		Type:      "CREATE_NAMESPACE",
		Status:    "SUCCESS",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Targets: map[string]string{
			"NAMESPACE": namespaceID,
		},
	}

	if err := h.storeOperation(ctx, operation); err != nil {
		h.logger.WithError(err).Error("Failed to store operation")
	}

	return &generated.CreateHttpNamespaceResponse{
		OperationId: &operationID,
	}, nil
}

// CreateService creates a service in a namespace
func (h *Handler) CreateService(ctx context.Context, input *generated.CreateServiceRequest) (*generated.CreateServiceResponse, error) {
	h.logger.WithField("name", input.Name).Info("Creating service")

	// Generate service ID
	serviceID := fmt.Sprintf("srv-%s", uuid.New().String())

	// Create service
	service := &Service{
		ID:          serviceID,
		Name:        input.Name,
		NamespaceID: stringValue(input.NamespaceId),
		CreatedAt:   time.Now(),
		Description: stringValue(input.Description),
	}

	// Configure DNS if provided
	if input.DnsConfig != nil {
		service.DNSConfig = &DNSConfig{
			NamespaceID:   stringValue(input.DnsConfig.NamespaceId),
			RoutingPolicy: string(ptrValue(input.DnsConfig.RoutingPolicy, generated.RoutingPolicyMULTIVALUE)),
		}

		for _, record := range input.DnsConfig.DnsRecords {
			service.DNSConfig.DNSRecords = append(service.DNSConfig.DNSRecords, DNSRecord{
				Type: string(record.Type),
				TTL:  record.TTL,
			})
		}
	}

	// Configure health check if provided
	if input.HealthCheckConfig != nil {
		service.HealthCheckConfig = &HealthCheckConfig{
			Type:             string(ptrValue(&input.HealthCheckConfig.Type, generated.HealthCheckTypeTCP)),
			ResourcePath:     stringValue(input.HealthCheckConfig.ResourcePath),
			FailureThreshold: int(ptrValue(input.HealthCheckConfig.FailureThreshold, 3)),
		}
	}

	// Configure custom health check if provided
	if input.HealthCheckCustomConfig != nil {
		service.HealthCheckCustomConfig = &HealthCheckCustomConfig{
			FailureThreshold: int(ptrValue(input.HealthCheckCustomConfig.FailureThreshold, 1)),
		}
	}

	if err := h.manager.CreateService(ctx, service); err != nil {
		h.logger.WithError(err).Error("Failed to create service")
		return nil, err
	}

	// Convert to generated type
	genService := h.convertServiceToGenerated(service)

	return &generated.CreateServiceResponse{
		Service: genService,
	}, nil
}

// RegisterInstance registers an instance with a service
func (h *Handler) RegisterInstance(ctx context.Context, input *generated.RegisterInstanceRequest) (*generated.RegisterInstanceResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"serviceId":  input.ServiceId,
		"instanceId": input.InstanceId,
	}).Info("Registering instance")

	operationID := fmt.Sprintf("op-%s", uuid.New().String())

	// Create instance
	instance := &Instance{
		ID:         input.InstanceId,
		ServiceID:  input.ServiceId,
		Attributes: input.Attributes,
		CreatedAt:  time.Now(),
	}

	if err := h.manager.RegisterInstance(ctx, instance); err != nil {
		h.logger.WithError(err).Error("Failed to register instance")
		return nil, err
	}

	// Store operation
	operation := &Operation{
		ID:        operationID,
		Type:      "REGISTER_INSTANCE",
		Status:    "SUCCESS",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Targets: map[string]string{
			"INSTANCE": input.InstanceId,
			"SERVICE":  input.ServiceId,
		},
	}

	if err := h.storeOperation(ctx, operation); err != nil {
		h.logger.WithError(err).Error("Failed to store operation")
	}

	return &generated.RegisterInstanceResponse{
		OperationId: &operationID,
	}, nil
}

// DeregisterInstance deregisters an instance from a service
func (h *Handler) DeregisterInstance(ctx context.Context, input *generated.DeregisterInstanceRequest) (*generated.DeregisterInstanceResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"serviceId":  input.ServiceId,
		"instanceId": input.InstanceId,
	}).Info("Deregistering instance")

	operationID := fmt.Sprintf("op-%s", uuid.New().String())

	if err := h.manager.DeregisterInstance(ctx, input.ServiceId, input.InstanceId); err != nil {
		h.logger.WithError(err).Error("Failed to deregister instance")
		return nil, err
	}

	// Store operation
	operation := &Operation{
		ID:        operationID,
		Type:      "DEREGISTER_INSTANCE",
		Status:    "SUCCESS",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Targets: map[string]string{
			"INSTANCE": input.InstanceId,
			"SERVICE":  input.ServiceId,
		},
	}

	if err := h.storeOperation(ctx, operation); err != nil {
		h.logger.WithError(err).Error("Failed to store operation")
	}

	return &generated.DeregisterInstanceResponse{
		OperationId: &operationID,
	}, nil
}

// DiscoverInstances discovers instances in a namespace
func (h *Handler) DiscoverInstances(ctx context.Context, input *generated.DiscoverInstancesRequest) (*generated.DiscoverInstancesResponse, error) {
	h.logger.WithFields(logrus.Fields{
		"namespace": input.NamespaceName,
		"service":   input.ServiceName,
	}).Info("Discovering instances")

	instances, err := h.manager.DiscoverInstances(ctx, input.NamespaceName, input.ServiceName)
	if err != nil {
		h.logger.WithError(err).Error("Failed to discover instances")
		return nil, err
	}

	// Convert to HTTP instance summaries
	var httpInstances []generated.HttpInstanceSummary
	for _, inst := range instances {
		httpInstances = append(httpInstances, generated.HttpInstanceSummary{
			Attributes:    inst.Attributes,
			HealthStatus:  (*generated.HealthStatus)(stringPtr("HEALTHY")),
			InstanceId:    &inst.ID,
			NamespaceName: &input.NamespaceName,
			ServiceName:   &input.ServiceName,
		})
	}

	revision := time.Now().Unix()
	return &generated.DiscoverInstancesResponse{
		Instances:         httpInstances,
		InstancesRevision: &revision,
	}, nil
}

// ListNamespaces lists namespaces
func (h *Handler) ListNamespaces(ctx context.Context, input *generated.ListNamespacesRequest) (*generated.ListNamespacesResponse, error) {
	h.logger.Info("Listing namespaces")

	namespaces, err := h.manager.ListNamespaces(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list namespaces")
		return nil, err
	}

	// Convert to generated types
	var genNamespaces []generated.NamespaceSummary
	for _, ns := range namespaces {
		nsType := h.convertNamespaceType(ns.Type)
		genNamespaces = append(genNamespaces, generated.NamespaceSummary{
			Arn:         stringPtr(fmt.Sprintf("arn:aws:servicediscovery:us-east-1:123456789012:namespace/%s", ns.ID)),
			CreateDate:  timePtr(ns.CreatedAt),
			Description: stringPtr(ns.Description),
			Id:          &ns.ID,
			Name:        &ns.Name,
			Type:        &nsType,
		})
	}

	return &generated.ListNamespacesResponse{
		Namespaces: genNamespaces,
	}, nil
}

// ListServices lists services in a namespace
func (h *Handler) ListServices(ctx context.Context, input *generated.ListServicesRequest) (*generated.ListServicesResponse, error) {
	h.logger.Info("Listing services")

	var namespaceID string
	if len(input.Filters) > 0 {
		for _, filter := range input.Filters {
			if filter.Name == generated.ServiceFilterNameNAMESPACE_ID {
				if len(filter.Values) > 0 {
					namespaceID = filter.Values[0]
				}
			}
		}
	}

	services, err := h.manager.ListServices(ctx, namespaceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list services")
		return nil, err
	}

	// Convert to generated types
	var genServices []generated.ServiceSummary
	for _, svc := range services {
		genServices = append(genServices, generated.ServiceSummary{
			Arn:         stringPtr(fmt.Sprintf("arn:aws:servicediscovery:us-east-1:123456789012:service/%s", svc.ID)),
			CreateDate:  timePtr(svc.CreatedAt),
			Description: stringPtr(svc.Description),
			Id:          &svc.ID,
			Name:        &svc.Name,
		})
	}

	return &generated.ListServicesResponse{
		Services: genServices,
	}, nil
}

// GetNamespace gets a namespace by ID
func (h *Handler) GetNamespace(ctx context.Context, input *generated.GetNamespaceRequest) (*generated.GetNamespaceResponse, error) {
	h.logger.WithField("id", input.Id).Info("Getting namespace")

	namespace, err := h.manager.GetNamespace(ctx, input.Id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get namespace")
		return nil, err
	}

	// Convert to generated type
	genNamespace := h.convertNamespaceToGenerated(namespace)

	return &generated.GetNamespaceResponse{
		Namespace: genNamespace,
	}, nil
}

// GetService gets a service by ID
func (h *Handler) GetService(ctx context.Context, input *generated.GetServiceRequest) (*generated.GetServiceResponse, error) {
	h.logger.WithField("id", input.Id).Info("Getting service")

	service, err := h.manager.GetService(ctx, input.Id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get service")
		return nil, err
	}

	// Convert to generated type
	genService := h.convertServiceToGenerated(service)

	return &generated.GetServiceResponse{
		Service: genService,
	}, nil
}

// Implement remaining required methods with basic stubs for now

func (h *Handler) DeleteNamespace(ctx context.Context, input *generated.DeleteNamespaceRequest) (*generated.DeleteNamespaceResponse, error) {
	operationID := fmt.Sprintf("op-%s", uuid.New().String())
	return &generated.DeleteNamespaceResponse{OperationId: &operationID}, nil
}

func (h *Handler) DeleteService(ctx context.Context, input *generated.DeleteServiceRequest) (*generated.DeleteServiceResponse, error) {
	return &generated.DeleteServiceResponse{}, nil
}

func (h *Handler) DeleteServiceAttributes(ctx context.Context, input *generated.DeleteServiceAttributesRequest) (*generated.DeleteServiceAttributesResponse, error) {
	return &generated.DeleteServiceAttributesResponse{}, nil
}

func (h *Handler) DiscoverInstancesRevision(ctx context.Context, input *generated.DiscoverInstancesRevisionRequest) (*generated.DiscoverInstancesRevisionResponse, error) {
	revision := time.Now().Unix()
	return &generated.DiscoverInstancesRevisionResponse{InstancesRevision: &revision}, nil
}

func (h *Handler) GetInstance(ctx context.Context, input *generated.GetInstanceRequest) (*generated.GetInstanceResponse, error) {
	return &generated.GetInstanceResponse{}, nil
}

func (h *Handler) GetInstancesHealthStatus(ctx context.Context, input *generated.GetInstancesHealthStatusRequest) (*generated.GetInstancesHealthStatusResponse, error) {
	return &generated.GetInstancesHealthStatusResponse{}, nil
}

func (h *Handler) GetOperation(ctx context.Context, input *generated.GetOperationRequest) (*generated.GetOperationResponse, error) {
	return &generated.GetOperationResponse{}, nil
}

func (h *Handler) GetServiceAttributes(ctx context.Context, input *generated.GetServiceAttributesRequest) (*generated.GetServiceAttributesResponse, error) {
	return &generated.GetServiceAttributesResponse{}, nil
}

func (h *Handler) ListInstances(ctx context.Context, input *generated.ListInstancesRequest) (*generated.ListInstancesResponse, error) {
	h.logger.WithField("serviceId", input.ServiceId).Info("Listing instances")

	// Get instances from manager
	instances, err := h.manager.ListInstances(ctx, input.ServiceId)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list instances")
		return nil, err
	}

	// Convert to API response format
	summaries := []generated.InstanceSummary{}
	for _, instance := range instances {
		summary := generated.InstanceSummary{
			Id:         &instance.ID,
			Attributes: instance.Attributes,
		}
		summaries = append(summaries, summary)
	}

	return &generated.ListInstancesResponse{
		Instances: summaries,
	}, nil
}

func (h *Handler) ListOperations(ctx context.Context, input *generated.ListOperationsRequest) (*generated.ListOperationsResponse, error) {
	return &generated.ListOperationsResponse{}, nil
}

func (h *Handler) ListTagsForResource(ctx context.Context, input *generated.ListTagsForResourceRequest) (*generated.ListTagsForResourceResponse, error) {
	return &generated.ListTagsForResourceResponse{}, nil
}

func (h *Handler) TagResource(ctx context.Context, input *generated.TagResourceRequest) (*generated.TagResourceResponse, error) {
	return &generated.TagResourceResponse{}, nil
}

func (h *Handler) UntagResource(ctx context.Context, input *generated.UntagResourceRequest) (*generated.UntagResourceResponse, error) {
	return &generated.UntagResourceResponse{}, nil
}

func (h *Handler) UpdateHttpNamespace(ctx context.Context, input *generated.UpdateHttpNamespaceRequest) (*generated.UpdateHttpNamespaceResponse, error) {
	operationID := fmt.Sprintf("op-%s", uuid.New().String())
	return &generated.UpdateHttpNamespaceResponse{OperationId: &operationID}, nil
}

func (h *Handler) UpdateInstanceCustomHealthStatus(ctx context.Context, input *generated.UpdateInstanceCustomHealthStatusRequest) (*generated.Unit, error) {
	return &generated.Unit{}, nil
}

func (h *Handler) UpdatePrivateDnsNamespace(ctx context.Context, input *generated.UpdatePrivateDnsNamespaceRequest) (*generated.UpdatePrivateDnsNamespaceResponse, error) {
	operationID := fmt.Sprintf("op-%s", uuid.New().String())
	return &generated.UpdatePrivateDnsNamespaceResponse{OperationId: &operationID}, nil
}

func (h *Handler) UpdatePublicDnsNamespace(ctx context.Context, input *generated.UpdatePublicDnsNamespaceRequest) (*generated.UpdatePublicDnsNamespaceResponse, error) {
	operationID := fmt.Sprintf("op-%s", uuid.New().String())
	return &generated.UpdatePublicDnsNamespaceResponse{OperationId: &operationID}, nil
}

func (h *Handler) UpdateService(ctx context.Context, input *generated.UpdateServiceRequest) (*generated.UpdateServiceResponse, error) {
	operationID := fmt.Sprintf("op-%s", uuid.New().String())
	return &generated.UpdateServiceResponse{OperationId: &operationID}, nil
}

func (h *Handler) UpdateServiceAttributes(ctx context.Context, input *generated.UpdateServiceAttributesRequest) (*generated.UpdateServiceAttributesResponse, error) {
	return &generated.UpdateServiceAttributesResponse{}, nil
}

// Helper functions

func (h *Handler) storeOperation(ctx context.Context, operation *Operation) error {
	// TODO: Implement operation storage when ServiceDiscoveryStore is available
	// For now, operations are not persisted
	h.logger.WithField("operationID", operation.ID).Debug("Operation storage not yet implemented")
	return nil
}

func (h *Handler) convertServiceToGenerated(service *Service) *generated.Service {
	genService := &generated.Service{
		Arn:         stringPtr(fmt.Sprintf("arn:aws:servicediscovery:us-east-1:123456789012:service/%s", service.ID)),
		CreateDate:  timePtr(service.CreatedAt),
		Description: stringPtr(service.Description),
		Id:          &service.ID,
		Name:        &service.Name,
	}

	if service.DNSConfig != nil {
		genService.DnsConfig = &generated.DnsConfig{
			NamespaceId: stringPtr(service.DNSConfig.NamespaceID),
		}

		if service.DNSConfig.RoutingPolicy != "" {
			policy := generated.RoutingPolicy(service.DNSConfig.RoutingPolicy)
			genService.DnsConfig.RoutingPolicy = &policy
		}

		for _, record := range service.DNSConfig.DNSRecords {
			genService.DnsConfig.DnsRecords = append(genService.DnsConfig.DnsRecords, generated.DnsRecord{
				Type: generated.RecordType(record.Type),
				TTL:  record.TTL,
			})
		}
	}

	return genService
}

func (h *Handler) convertNamespaceToGenerated(namespace *Namespace) *generated.Namespace {
	nsType := h.convertNamespaceType(namespace.Type)
	return &generated.Namespace{
		Arn:         stringPtr(fmt.Sprintf("arn:aws:servicediscovery:us-east-1:123456789012:namespace/%s", namespace.ID)),
		CreateDate:  timePtr(namespace.CreatedAt),
		Description: stringPtr(namespace.Description),
		Id:          &namespace.ID,
		Name:        &namespace.Name,
		Type:        &nsType,
	}
}

func (h *Handler) convertNamespaceType(nsType NamespaceType) generated.NamespaceType {
	switch nsType {
	case NamespaceTypeDNSPrivate:
		return generated.NamespaceTypeDNS_PRIVATE
	case NamespaceTypeDNSPublic:
		return generated.NamespaceTypeDNS_PUBLIC
	case NamespaceTypeHTTP:
		return generated.NamespaceTypeHTTP
	default:
		return generated.NamespaceTypeHTTP
	}
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func ptrValue[T any](ptr *T, defaultValue T) T {
	if ptr == nil {
		return defaultValue
	}
	return *ptr
}
