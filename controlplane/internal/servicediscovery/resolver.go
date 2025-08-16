package servicediscovery

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// DNSResolver provides DNS resolution strategy for service discovery
type DNSResolver interface {
	// ResolveInternal resolves a service name within Kubernetes
	ResolveInternal(ctx context.Context, serviceName, namespace string) ([]string, error)

	// ResolveExternal resolves a service name via Route53
	ResolveExternal(ctx context.Context, hostname string) ([]string, error)

	// Resolve uses a fallback strategy to resolve a service name
	Resolve(ctx context.Context, query string) ([]string, error)
}

// dnsResolver implements the DNSResolver interface
type dnsResolver struct {
	manager *manager
}

// NewDNSResolver creates a new DNS resolver
func NewDNSResolver(mgr Manager) DNSResolver {
	// Type assert to get the concrete manager
	m, ok := mgr.(*manager)
	if !ok {
		return nil
	}
	return &dnsResolver{
		manager: m,
	}
}

// ResolveInternal resolves a service name within Kubernetes
func (r *dnsResolver) ResolveInternal(ctx context.Context, serviceName, namespace string) ([]string, error) {
	// Construct Kubernetes DNS name
	// Format: service-name.namespace.svc.cluster.local
	kubernetesName := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)

	// Use standard DNS resolution
	ips, err := net.LookupHost(kubernetesName)
	if err != nil {
		// Try without the full domain
		kubernetesName = fmt.Sprintf("%s.%s", serviceName, namespace)
		ips, err = net.LookupHost(kubernetesName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", serviceName, err)
		}
	}

	logging.Debug("Resolved service via Kubernetes DNS", "service", serviceName, "namespace", namespace, "ips", ips)
	return ips, nil
}

// ResolveExternal resolves a service name via Route53
func (r *dnsResolver) ResolveExternal(ctx context.Context, hostname string) ([]string, error) {
	if r.manager.route53Manager == nil {
		return nil, fmt.Errorf("Route53 integration not available")
	}

	// Parse hostname to extract service and namespace
	// Expected format: service-name.namespace-name or service-name.namespace-name.domain
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid hostname format: %s", hostname)
	}

	serviceName := parts[0]
	namespace := parts[1]

	// If the namespace includes additional domain parts, join them
	if len(parts) > 2 {
		namespace = strings.Join(parts[1:], ".")
	}

	// Resolve via Route53
	ips, err := r.manager.route53Manager.ResolveService(ctx, namespace, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve via Route53: %w", err)
	}

	logging.Debug("Resolved service via Route53", "hostname", hostname, "ips", ips)
	return ips, nil
}

// Resolve uses a fallback strategy to resolve a service name
func (r *dnsResolver) Resolve(ctx context.Context, query string) ([]string, error) {
	// Parse the query to determine the best resolution strategy
	parts := strings.Split(query, ".")

	// Determine if this looks like a Kubernetes internal name
	isKubernetesName := strings.Contains(query, ".svc.cluster.local") ||
		strings.Contains(query, ".svc") ||
		len(parts) == 2 // Simple service.namespace format

	if isKubernetesName {
		// Try Kubernetes DNS first
		serviceName := parts[0]
		namespace := "default"
		if len(parts) > 1 {
			namespace = parts[1]
		}

		ips, err := r.ResolveInternal(ctx, serviceName, namespace)
		if err == nil && len(ips) > 0 {
			return ips, nil
		}

		logging.Debug("Kubernetes DNS resolution failed, trying Route53", "query", query, "error", err)
	}

	// Try Route53 as fallback or primary for external names
	if r.manager.route53Manager != nil {
		ips, err := r.ResolveExternal(ctx, query)
		if err == nil && len(ips) > 0 {
			return ips, nil
		}
		logging.Debug("Route53 resolution failed", "query", query, "error", err)
	}

	// Last resort: try standard DNS resolution
	ips, err := net.LookupHost(query)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s using any method: %w", query, err)
	}

	logging.Debug("Resolved via standard DNS", "query", query, "ips", ips)
	return ips, nil
}

// DiscoverInstancesWithResolver enhances the DiscoverInstances method with DNS resolution fallback
func (r *dnsResolver) DiscoverInstancesWithResolver(ctx context.Context, req *DiscoverInstancesRequest) (*DiscoverInstancesResponse, error) {
	// First try the standard discovery method
	instances, err := r.manager.DiscoverInstances(ctx, req.NamespaceName, req.ServiceName)
	if err == nil && len(instances) > 0 {
		// Convert instances to InstanceSummary
		var summaries []InstanceSummary
		for _, inst := range instances {
			summaries = append(summaries, InstanceSummary{
				InstanceId:    inst.ID,
				NamespaceName: req.NamespaceName,
				ServiceName:   req.ServiceName,
				HealthStatus:  inst.HealthStatus,
				Attributes:    inst.Attributes,
			})
		}
		return &DiscoverInstancesResponse{
			Instances: summaries,
		}, nil
	}

	// If no instances found, try DNS resolution
	query := fmt.Sprintf("%s.%s", req.ServiceName, req.NamespaceName)
	ips, err := r.Resolve(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("service discovery failed: %w", err)
	}

	// Convert IPs to instance summaries
	dnsInstances := make([]InstanceSummary, 0, len(ips))
	for i, ip := range ips {
		dnsInstances = append(dnsInstances, InstanceSummary{
			InstanceId:    fmt.Sprintf("dns-resolved-%d", i),
			NamespaceName: req.NamespaceName,
			ServiceName:   req.ServiceName,
			HealthStatus:  "HEALTHY",
			Attributes: map[string]string{
				"AWS_INSTANCE_IPV4": ip,
				"DNS_RESOLVED":      "true",
			},
		})
	}

	return &DiscoverInstancesResponse{
		Instances: dnsInstances,
	}, nil
}
