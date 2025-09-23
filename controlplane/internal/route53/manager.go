package route53

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// Manager manages Route53 hosted zones and records for service discovery
type Manager struct {
	client *Client
	mu     sync.RWMutex

	// Cache of namespace to hosted zone mapping
	namespaceZones map[string]string // namespace name -> hosted zone ID

	// Cache of service records
	serviceRecords map[string]map[string][]string // zoneID -> service name -> IPs

	// Default VPC configuration
	defaultVPC *VPCConfig
}

// NewManager creates a new Route53 manager
func NewManager(client *Client, defaultVPC *VPCConfig) *Manager {
	return &Manager{
		client:         client,
		namespaceZones: make(map[string]string),
		serviceRecords: make(map[string]map[string][]string),
		defaultVPC:     defaultVPC,
	}
}

// CreateHostedZone creates a hosted zone for a namespace
func (m *Manager) CreateHostedZone(ctx context.Context, namespace string, isPrivate bool) (string, error) {
	// For now, just call CreateNamespaceZone
	// TODO: Handle private vs public zone distinction
	return m.CreateNamespaceZone(ctx, namespace)
}

// CreateNamespaceZone creates a hosted zone for a service discovery namespace
func (m *Manager) CreateNamespaceZone(ctx context.Context, namespace string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if zone already exists in cache
	if zoneID, exists := m.namespaceZones[namespace]; exists {
		logging.Debug("Hosted zone already exists for namespace", "namespace", namespace, "zoneID", zoneID)
		return zoneID, nil
	}

	// Ensure namespace ends with a dot for Route53
	zoneName := namespace
	if !strings.HasSuffix(zoneName, ".") {
		zoneName = zoneName + "."
	}

	// Check if zone exists in Route53
	zones, err := m.client.ListHostedZones(ctx)
	if err != nil {
		logging.Warn("Failed to list hosted zones", "error", err)
		// Continue anyway - we might be in LocalStack mode without Route53
	} else {
		for _, zone := range zones {
			if zone.Name == zoneName {
				m.namespaceZones[namespace] = zone.ID
				logging.Info("Found existing hosted zone for namespace", "namespace", namespace, "zoneID", zone.ID)
				return zone.ID, nil
			}
		}
	}

	// Create new hosted zone
	zone, err := m.client.CreateHostedZone(ctx, namespace, m.defaultVPC)
	if err != nil {
		return "", fmt.Errorf("failed to create hosted zone for namespace %s: %w", namespace, err)
	}

	m.namespaceZones[namespace] = zone.ID
	m.serviceRecords[zone.ID] = make(map[string][]string)

	logging.Info("Created hosted zone for namespace", "namespace", namespace, "zoneID", zone.ID)
	return zone.ID, nil
}

// DeleteNamespaceZone deletes a hosted zone for a namespace
func (m *Manager) DeleteNamespaceZone(ctx context.Context, namespace string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	zoneID, exists := m.namespaceZones[namespace]
	if !exists {
		logging.Debug("No hosted zone found for namespace", "namespace", namespace)
		return nil
	}

	// Delete all records first (except NS and SOA which are managed by Route53)
	records, err := m.client.ListResourceRecordSets(ctx, zoneID, "")
	if err != nil {
		logging.Warn("Failed to list records for cleanup", "error", err)
	} else {
		for _, record := range records {
			// Skip NS and SOA records
			if record.Type == types.RRTypeNs || record.Type == types.RRTypeSoa {
				continue
			}

			err := m.client.DeleteRecord(ctx, zoneID, *record.Name, record.Type)
			if err != nil {
				logging.Warn("Failed to delete record", "name", *record.Name, "type", record.Type, "error", err)
			}
		}
	}

	// Delete the hosted zone
	err = m.client.DeleteHostedZone(ctx, zoneID)
	if err != nil {
		return fmt.Errorf("failed to delete hosted zone: %w", err)
	}

	delete(m.namespaceZones, namespace)
	delete(m.serviceRecords, zoneID)

	logging.Info("Deleted hosted zone for namespace", "namespace", namespace)
	return nil
}

// RegisterService registers a service with its instances
func (m *Manager) RegisterService(ctx context.Context, namespace, serviceName string, ips []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	zoneID, exists := m.namespaceZones[namespace]
	if !exists {
		// Try to create the zone if it doesn't exist
		m.mu.Unlock()
		zoneID, err := m.CreateNamespaceZone(ctx, namespace)
		m.mu.Lock()
		if err != nil {
			return fmt.Errorf("namespace zone not found and failed to create: %w", err)
		}
		m.namespaceZones[namespace] = zoneID
	}

	// Construct the full DNS name
	dnsName := fmt.Sprintf("%s.%s", serviceName, namespace)
	if !strings.HasSuffix(dnsName, ".") {
		dnsName = dnsName + "."
	}

	// Update A record
	err := m.client.UpsertARecord(ctx, zoneID, dnsName, ips)
	if err != nil {
		return fmt.Errorf("failed to update A record: %w", err)
	}

	// Cache the IPs
	if m.serviceRecords[zoneID] == nil {
		m.serviceRecords[zoneID] = make(map[string][]string)
	}
	m.serviceRecords[zoneID][serviceName] = ips

	logging.Info("Registered service in Route53", "namespace", namespace, "service", serviceName, "ips", ips)
	return nil
}

// RegisterServiceWithPorts registers a service with port information using SRV records
func (m *Manager) RegisterServiceWithPorts(ctx context.Context, namespace, serviceName string, targets []ServiceTarget) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	zoneID, exists := m.namespaceZones[namespace]
	if !exists {
		// Try to create the zone if it doesn't exist
		m.mu.Unlock()
		zoneID, err := m.CreateNamespaceZone(ctx, namespace)
		m.mu.Lock()
		if err != nil {
			return fmt.Errorf("namespace zone not found and failed to create: %w", err)
		}
		m.namespaceZones[namespace] = zoneID
	}

	// Construct the full DNS name for SRV record
	// Format: _service._proto.name
	srvName := fmt.Sprintf("_http._tcp.%s.%s", serviceName, namespace)
	if !strings.HasSuffix(srvName, ".") {
		srvName = srvName + "."
	}

	// Convert service targets to SRV targets
	srvTargets := make([]SRVTarget, 0, len(targets))
	for _, target := range targets {
		srvTargets = append(srvTargets, SRVTarget{
			Priority: 10,
			Weight:   10,
			Port:     target.Port,
			Target:   target.Host,
		})
	}

	// Update SRV record
	err := m.client.UpsertSRVRecord(ctx, zoneID, srvName, srvTargets)
	if err != nil {
		return fmt.Errorf("failed to update SRV record: %w", err)
	}

	// Also update A records for the individual hosts
	ips := make([]string, 0, len(targets))
	for _, target := range targets {
		if target.IP != "" {
			ips = append(ips, target.IP)
		}
	}

	if len(ips) > 0 {
		dnsName := fmt.Sprintf("%s.%s", serviceName, namespace)
		if !strings.HasSuffix(dnsName, ".") {
			dnsName = dnsName + "."
		}
		err = m.client.UpsertARecord(ctx, zoneID, dnsName, ips)
		if err != nil {
			logging.Warn("Failed to update A record for service", "service", serviceName, "error", err)
		}
	}

	logging.Info("Registered service with ports in Route53", "namespace", namespace, "service", serviceName, "targets", len(targets))
	return nil
}

// DeregisterService removes a service from Route53
func (m *Manager) DeregisterService(ctx context.Context, namespace, serviceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	zoneID, exists := m.namespaceZones[namespace]
	if !exists {
		logging.Debug("No hosted zone found for namespace", "namespace", namespace)
		return nil
	}

	// Construct the full DNS names
	dnsName := fmt.Sprintf("%s.%s", serviceName, namespace)
	if !strings.HasSuffix(dnsName, ".") {
		dnsName = dnsName + "."
	}

	srvName := fmt.Sprintf("_http._tcp.%s.%s", serviceName, namespace)
	if !strings.HasSuffix(srvName, ".") {
		srvName = srvName + "."
	}

	// Delete A record
	err := m.client.DeleteRecord(ctx, zoneID, dnsName, types.RRTypeA)
	if err != nil {
		logging.Warn("Failed to delete A record", "name", dnsName, "error", err)
	}

	// Delete SRV record
	err = m.client.DeleteRecord(ctx, zoneID, srvName, types.RRTypeSrv)
	if err != nil {
		logging.Warn("Failed to delete SRV record", "name", srvName, "error", err)
	}

	// Remove from cache
	delete(m.serviceRecords[zoneID], serviceName)

	logging.Info("Deregistered service from Route53", "namespace", namespace, "service", serviceName)
	return nil
}

// ResolveService resolves a service name to IP addresses
func (m *Manager) ResolveService(ctx context.Context, namespace, serviceName string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zoneID, exists := m.namespaceZones[namespace]
	if !exists {
		return nil, fmt.Errorf("namespace not found: %s", namespace)
	}

	// Check cache first
	if records, exists := m.serviceRecords[zoneID][serviceName]; exists {
		return records, nil
	}

	// Query Route53
	dnsName := fmt.Sprintf("%s.%s", serviceName, namespace)
	if !strings.HasSuffix(dnsName, ".") {
		dnsName = dnsName + "."
	}

	records, err := m.client.ListResourceRecordSets(ctx, zoneID, dnsName)
	if err != nil {
		return nil, fmt.Errorf("failed to query Route53: %w", err)
	}

	var ips []string
	for _, record := range records {
		if record.Type == types.RRTypeA && *record.Name == dnsName {
			for _, rr := range record.ResourceRecords {
				ips = append(ips, *rr.Value)
			}
			break
		}
	}

	return ips, nil
}

// GetNamespaceZoneID returns the hosted zone ID for a namespace
func (m *Manager) GetNamespaceZoneID(namespace string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	zoneID, exists := m.namespaceZones[namespace]
	return zoneID, exists
}

// ServiceTarget represents a service instance with host and port information
type ServiceTarget struct {
	Host string
	IP   string
	Port uint16
}
