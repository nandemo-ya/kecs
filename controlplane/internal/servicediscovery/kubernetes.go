package servicediscovery

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// updateKubernetesEndpoints updates Kubernetes endpoints for service discovery
func (m *manager) updateKubernetesEndpoints(ctx context.Context, service *Service, instances map[string]*Instance) error {
	namespace, exists := m.namespaces[service.NamespaceID]
	if !exists {
		return fmt.Errorf("namespace not found: %s", service.NamespaceID)
	}

	// Get Kubernetes namespace
	k8sNamespace := m.dnsToK8sNamespace[namespace.Name]
	if k8sNamespace == "" {
		k8sNamespace = "default"
	}

	// Service name for Kubernetes
	k8sServiceName := fmt.Sprintf("sd-%s", service.Name)

	// Check if Kubernetes Service exists, create if not
	k8sService, err := m.kubeClient.CoreV1().Services(k8sNamespace).Get(ctx, k8sServiceName, metav1.GetOptions{})
	if err != nil {
		// Create headless service for service discovery
		k8sService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sServiceName,
				Namespace: k8sNamespace,
				Labels: map[string]string{
					"kecs.io/service-discovery": "true",
					"kecs.io/namespace":         namespace.Name,
					"kecs.io/service":           service.Name,
				},
				Annotations: map[string]string{
					"kecs.io/service-id":   service.ID,
					"kecs.io/namespace-id": service.NamespaceID,
					"kecs.io/service-arn":  service.ARN,
				},
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: corev1.ClusterIPNone, // Headless service
				Selector:  nil,                  // We'll manage endpoints manually
				Ports:     m.getServicePorts(service),
			},
		}

		if _, err := m.kubeClient.CoreV1().Services(k8sNamespace).Create(ctx, k8sService, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create Kubernetes service: %w", err)
		}
		logging.Info("Created Kubernetes service for service discovery", "namespace", k8sNamespace, "service", k8sServiceName)
	}

	// Update Endpoints
	endpoints := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sServiceName,
			Namespace: k8sNamespace,
			Labels: map[string]string{
				"kecs.io/service-discovery": "true",
				"kecs.io/namespace":         namespace.Name,
				"kecs.io/service":           service.Name,
			},
		},
		Subsets: m.buildEndpointSubsets(instances, service),
	}

	// Try to update first, create if doesn't exist
	if _, err := m.kubeClient.CoreV1().Endpoints(k8sNamespace).Update(ctx, endpoints, metav1.UpdateOptions{}); err != nil {
		if _, err := m.kubeClient.CoreV1().Endpoints(k8sNamespace).Create(ctx, endpoints, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("failed to create/update endpoints: %w", err)
		}
	}

	logging.Debug("Updated endpoints for service", "service", k8sServiceName, "instanceCount", len(instances))

	return nil
}

// getServicePorts extracts ports from service DNS configuration
func (m *manager) getServicePorts(service *Service) []corev1.ServicePort {
	ports := []corev1.ServicePort{}

	// Default port for A/AAAA records
	defaultPort := corev1.ServicePort{
		Name:       "http",
		Port:       80,
		TargetPort: intstr.FromInt(80),
		Protocol:   corev1.ProtocolTCP,
	}

	if service.DnsConfig != nil && len(service.DnsConfig.DnsRecords) > 0 {
		for _, record := range service.DnsConfig.DnsRecords {
			if record.Type == "SRV" {
				// For SRV records, we need to extract port from instance attributes
				// This is handled in buildEndpointSubsets
				continue
			}
		}
	}

	// Always add a default port for DNS resolution
	ports = append(ports, defaultPort)

	return ports
}

// buildEndpointSubsets builds endpoint subsets from instances
func (m *manager) buildEndpointSubsets(instances map[string]*Instance, service *Service) []corev1.EndpointSubset {
	if len(instances) == 0 {
		return []corev1.EndpointSubset{}
	}

	addresses := []corev1.EndpointAddress{}
	notReadyAddresses := []corev1.EndpointAddress{}

	for _, instance := range instances {
		// Get IP address from attributes
		ipAddress := instance.Attributes["AWS_INSTANCE_IPV4"]
		if ipAddress == "" {
			// Try other common attribute names
			ipAddress = instance.Attributes["IP_ADDRESS"]
			if ipAddress == "" {
				ipAddress = instance.Attributes["PRIVATE_IP"]
			}
		}

		if ipAddress == "" {
			logging.Warn("Instance has no IP address in attributes", "instanceID", instance.ID)
			continue
		}

		endpointAddress := corev1.EndpointAddress{
			IP: ipAddress,
			TargetRef: &corev1.ObjectReference{
				Kind:      "Pod",
				Name:      instance.ID,
				Namespace: m.getK8sNamespaceForInstance(instance),
			},
		}

		// Add hostname if available
		if hostname, ok := instance.Attributes["HOSTNAME"]; ok {
			endpointAddress.Hostname = hostname
		}

		// Categorize by health status
		if instance.HealthStatus == "HEALTHY" {
			addresses = append(addresses, endpointAddress)
		} else {
			notReadyAddresses = append(notReadyAddresses, endpointAddress)
		}
	}

	// Build ports
	ports := []corev1.EndpointPort{}

	// Check if we have SRV records that specify ports
	if service.DnsConfig != nil {
		for _, record := range service.DnsConfig.DnsRecords {
			if record.Type == "SRV" {
				// For SRV records, check instance attributes for port
				for _, instance := range instances {
					if portStr, ok := instance.Attributes["PORT"]; ok {
						port := parsePort(portStr)
						if port > 0 {
							ports = append(ports, corev1.EndpointPort{
								Port:     int32(port),
								Protocol: corev1.ProtocolTCP,
							})
							break
						}
					}
				}
			}
		}
	}

	// Default port if none specified
	if len(ports) == 0 {
		ports = append(ports, corev1.EndpointPort{
			Port:     80,
			Protocol: corev1.ProtocolTCP,
		})
	}

	subset := corev1.EndpointSubset{
		Addresses:         addresses,
		NotReadyAddresses: notReadyAddresses,
		Ports:             ports,
	}

	return []corev1.EndpointSubset{subset}
}

// getK8sNamespaceForInstance gets the Kubernetes namespace for an instance
func (m *manager) getK8sNamespaceForInstance(instance *Instance) string {
	// Check if instance has namespace attribute
	if ns, ok := instance.Attributes["K8S_NAMESPACE"]; ok {
		return ns
	}

	// Check if instance has cluster attribute and map to namespace
	if cluster, ok := instance.Attributes["ECS_CLUSTER"]; ok {
		// Map ECS cluster to Kubernetes namespace
		return fmt.Sprintf("%s-%s", cluster, m.region)
	}

	return "default"
}

// parsePort parses a port string to int
func parsePort(portStr string) int {
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}
