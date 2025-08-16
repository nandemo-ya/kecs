package servicediscovery

import (
	"time"
)

// NamespaceType represents the type of namespace
type NamespaceType string

const (
	// NamespaceTypeDNSPrivate represents a private DNS namespace
	NamespaceTypeDNSPrivate NamespaceType = "DNS_PRIVATE"
	// NamespaceTypeDNSPublic represents a public DNS namespace
	NamespaceTypeDNSPublic NamespaceType = "DNS_PUBLIC"
	// NamespaceTypeHTTP represents an HTTP namespace
	NamespaceTypeHTTP NamespaceType = "HTTP"
)

// Namespace represents a Cloud Map namespace
type Namespace struct {
	ID           string
	ARN          string
	Name         string
	Type         NamespaceType // DNS_PRIVATE, DNS_PUBLIC, or HTTP
	Description  string
	ServiceCount int
	VPC          string // VPC ID for private DNS namespaces
	HostedZoneID string // Route53 hosted zone ID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Properties   *NamespaceProperties
}

// NamespaceProperties contains DNS properties for a namespace
type NamespaceProperties struct {
	DnsProperties *DnsProperties `json:"dnsProperties,omitempty"`
}

// DnsProperties contains DNS configuration
type DnsProperties struct {
	HostedZoneId string `json:"hostedZoneId,omitempty"`
}

// Service represents a Cloud Map service
type Service struct {
	ID                      string
	ARN                     string
	Name                    string
	NamespaceID             string
	Description             string
	InstanceCount           int
	DNSConfig               *DNSConfig
	HealthCheckConfig       *HealthCheckConfig
	HealthCheckCustomConfig *HealthCheckCustomConfig
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// DNSConfig contains DNS configuration for a service
type DNSConfig struct {
	NamespaceID   string      `json:"namespaceId"`
	RoutingPolicy string      `json:"routingPolicy,omitempty"` // MULTIVALUE or WEIGHTED
	DNSRecords    []DNSRecord `json:"dnsRecords"`
}

// DNSRecord represents a DNS record configuration
type DNSRecord struct {
	Type string `json:"type"` // A, AAAA, SRV, CNAME
	TTL  int64  `json:"ttl"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Type             string `json:"type,omitempty"` // HTTP, HTTPS, TCP
	ResourcePath     string `json:"resourcePath,omitempty"`
	FailureThreshold int    `json:"failureThreshold,omitempty"`
}

// HealthCheckCustomConfig represents custom health check configuration
type HealthCheckCustomConfig struct {
	FailureThreshold int `json:"failureThreshold"`
}

// Instance represents a service instance
type Instance struct {
	ID           string
	ServiceID    string
	Attributes   map[string]string
	HealthStatus string // HEALTHY, UNHEALTHY, UNKNOWN
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// DiscoverInstancesRequest represents a request to discover instances
type DiscoverInstancesRequest struct {
	NamespaceName      string            `json:"namespaceName"`
	ServiceName        string            `json:"serviceName"`
	MaxResults         int32             `json:"maxResults,omitempty"`
	OptionalParameters map[string]string `json:"optionalParameters,omitempty"`
	HealthStatus       string            `json:"healthStatus,omitempty"` // ALL, HEALTHY, UNHEALTHY
}

// DiscoverInstancesResponse represents the response from discover instances
type DiscoverInstancesResponse struct {
	Instances []InstanceSummary `json:"instances"`
}

// InstanceSummary contains summary information about an instance
type InstanceSummary struct {
	InstanceId    string            `json:"instanceId"`
	NamespaceName string            `json:"namespaceName"`
	ServiceName   string            `json:"serviceName"`
	HealthStatus  string            `json:"healthStatus"`
	Attributes    map[string]string `json:"attributes"`
}

// Operation represents an async operation
type Operation struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Status    string            `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Targets   map[string]string `json:"targets"`
}
