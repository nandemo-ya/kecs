package servicediscovery

import (
	"time"
)

// Namespace represents a Cloud Map namespace
type Namespace struct {
	ID           string
	ARN          string
	Name         string
	Type         string // DNS_PRIVATE or DNS_PUBLIC
	Description  string
	ServiceCount int
	CreatedAt    time.Time
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
	ID            string
	ARN           string
	Name          string
	NamespaceID   string
	Description   string
	InstanceCount int
	DnsConfig     *DnsConfig
	HealthCheck   *HealthCheckConfig
	CreatedAt     time.Time
}

// DnsConfig contains DNS configuration for a service
type DnsConfig struct {
	NamespaceId   string      `json:"namespaceId"`
	RoutingPolicy string      `json:"routingPolicy,omitempty"` // MULTIVALUE or WEIGHTED
	DnsRecords    []DnsRecord `json:"dnsRecords"`
}

// DnsRecord represents a DNS record configuration
type DnsRecord struct {
	Type string `json:"type"` // A, AAAA, SRV, CNAME
	TTL  int64  `json:"ttl"`
}

// HealthCheckConfig contains health check configuration
type HealthCheckConfig struct {
	Type             string `json:"type,omitempty"` // HTTP, HTTPS, TCP
	ResourcePath     string `json:"resourcePath,omitempty"`
	FailureThreshold int32  `json:"failureThreshold,omitempty"`
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
