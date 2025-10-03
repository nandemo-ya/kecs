package servicediscovery

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

const (
	coreDNSNamespace       = "kube-system"
	coreDNSConfigMap       = "coredns"
	customCoreDNSConfigMap = "coredns-custom"
)

// updateCoreDNSConfig updates CoreDNS configuration to include service discovery domains
func (m *manager) updateCoreDNSConfig(ctx context.Context, namespace *Namespace) error {
	if m.kubeClient == nil {
		logging.Debug("Kubernetes client not available, skipping CoreDNS update")
		return nil
	}

	// For HTTP namespaces, no DNS configuration needed
	if namespace.Type == NamespaceTypeHTTP {
		return nil
	}

	// Get or create custom CoreDNS ConfigMap
	customCM, err := m.kubeClient.CoreV1().ConfigMaps(coreDNSNamespace).Get(ctx, customCoreDNSConfigMap, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create custom ConfigMap
			customCM = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      customCoreDNSConfigMap,
					Namespace: coreDNSNamespace,
					Labels: map[string]string{
						"kecs.io/managed": "true",
					},
				},
				Data: make(map[string]string),
			}
			customCM, err = m.kubeClient.CoreV1().ConfigMaps(coreDNSNamespace).Create(ctx, customCM, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create custom CoreDNS ConfigMap: %w", err)
			}
			logging.Info("Created custom CoreDNS ConfigMap", "name", customCoreDNSConfigMap)
		} else {
			return fmt.Errorf("failed to get custom CoreDNS ConfigMap: %w", err)
		}
	}

	// Add configuration for this namespace domain
	corefile := m.buildCoreDNSConfig(namespace)

	// Store configuration with namespace ID as key
	if customCM.Data == nil {
		customCM.Data = make(map[string]string)
	}

	// Check for duplicate domain configurations
	// Remove any existing entries for the same domain to prevent CoreDNS conflicts
	for key, existingConfig := range customCM.Data {
		if strings.Contains(existingConfig, fmt.Sprintf("%s:53", namespace.Name)) {
			logging.Warn("Removing duplicate CoreDNS configuration for domain",
				"domain", namespace.Name,
				"existingKey", key,
				"newKey", fmt.Sprintf("kecs-%s.server", namespace.ID))
			delete(customCM.Data, key)
		}
	}

	customCM.Data[fmt.Sprintf("kecs-%s.server", namespace.ID)] = corefile

	// Update ConfigMap
	_, err = m.kubeClient.CoreV1().ConfigMaps(coreDNSNamespace).Update(ctx, customCM, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update custom CoreDNS ConfigMap: %w", err)
	}

	// Restart CoreDNS to pick up new configuration
	if err := m.restartCoreDNS(ctx); err != nil {
		logging.Warn("Failed to restart CoreDNS, configuration will be picked up eventually", "error", err)
	}

	logging.Info("Updated CoreDNS configuration for namespace", "namespace", namespace.Name)
	return nil
}

// removeCoreDNSConfig removes CoreDNS configuration for a namespace
func (m *manager) removeCoreDNSConfig(ctx context.Context, namespaceID string) error {
	if m.kubeClient == nil {
		return nil
	}

	customCM, err := m.kubeClient.CoreV1().ConfigMaps(coreDNSNamespace).Get(ctx, customCoreDNSConfigMap, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Nothing to remove
		}
		return fmt.Errorf("failed to get custom CoreDNS ConfigMap: %w", err)
	}

	// Remove configuration for this namespace
	delete(customCM.Data, fmt.Sprintf("kecs-%s.server", namespaceID))

	// Update ConfigMap
	_, err = m.kubeClient.CoreV1().ConfigMaps(coreDNSNamespace).Update(ctx, customCM, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update custom CoreDNS ConfigMap: %w", err)
	}

	// Restart CoreDNS
	if err := m.restartCoreDNS(ctx); err != nil {
		logging.Warn("Failed to restart CoreDNS", "error", err)
	}

	logging.Info("Removed CoreDNS configuration for namespace", "namespaceID", namespaceID)
	return nil
}

// buildCoreDNSConfig builds CoreDNS configuration for a namespace
func (m *manager) buildCoreDNSConfig(namespace *Namespace) string {
	var sb strings.Builder

	// Get Kubernetes namespace for this DNS namespace
	k8sNamespace := m.dnsToK8sNamespace[namespace.Name]
	if k8sNamespace == "" {
		k8sNamespace = "default"
	}

	// Build CoreDNS server block for this domain
	sb.WriteString(fmt.Sprintf("%s:53 {\n", namespace.Name))
	sb.WriteString("    errors\n")
	sb.WriteString("    health {\n")
	sb.WriteString("        lameduck 5s\n")
	sb.WriteString("    }\n")
	sb.WriteString("    ready\n")

	// Use Kubernetes plugin to resolve services
	// Format: kubernetes <k8s-namespace> <zone> in-addr.arpa ip6.arpa
	// This tells CoreDNS to resolve <zone> queries using services in <k8s-namespace>
	sb.WriteString(fmt.Sprintf("    kubernetes %s %s in-addr.arpa ip6.arpa {\n", k8sNamespace, namespace.Name))
	sb.WriteString("        pods insecure\n")
	sb.WriteString("        fallthrough in-addr.arpa ip6.arpa\n")
	sb.WriteString("    }\n")

	// If Route53 integration is available, forward to Route53
	if m.route53Manager != nil && namespace.HostedZoneID != "" {
		// Get LocalStack endpoint from environment
		localstackEndpoint := m.getLocalStackDNSEndpoint()
		if localstackEndpoint != "" {
			sb.WriteString(fmt.Sprintf("    forward . %s {\n", localstackEndpoint))
			sb.WriteString("        except kubernetes.default.svc.cluster.local\n")
			sb.WriteString("    }\n")
		}
	}

	sb.WriteString("    cache 30\n")
	sb.WriteString("    loop\n")
	sb.WriteString("    reload\n")
	sb.WriteString("    loadbalance\n")
	sb.WriteString("}\n")

	return sb.String()
}

// restartCoreDNS restarts CoreDNS pods to pick up configuration changes
func (m *manager) restartCoreDNS(ctx context.Context) error {
	// Delete CoreDNS pods to trigger restart
	pods, err := m.kubeClient.CoreV1().Pods(coreDNSNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: "k8s-app=kube-dns",
	})
	if err != nil {
		return fmt.Errorf("failed to list CoreDNS pods: %w", err)
	}

	for _, pod := range pods.Items {
		err := m.kubeClient.CoreV1().Pods(coreDNSNamespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			logging.Warn("Failed to delete CoreDNS pod", "pod", pod.Name, "error", err)
		}
	}

	logging.Debug("Restarted CoreDNS pods", "count", len(pods.Items))
	return nil
}

// getLocalStackDNSEndpoint gets the LocalStack DNS endpoint
func (m *manager) getLocalStackDNSEndpoint() string {
	// Check if LocalStack is running and get its DNS endpoint
	// For now, return a default value
	// TODO: Get actual LocalStack service endpoint
	return "" // Disabled for now until LocalStack DNS is properly configured
}

// createServiceDNSAlias creates a DNS alias for the service in Kubernetes
func (m *manager) createServiceDNSAlias(ctx context.Context, namespace *Namespace, service *Service) error {
	if m.kubeClient == nil {
		return nil
	}

	// Get Kubernetes namespace
	k8sNamespace := m.dnsToK8sNamespace[namespace.Name]
	if k8sNamespace == "" {
		k8sNamespace = "default"
	}

	// Create an ExternalName service that points to the headless service
	// This allows resolution of service.namespace.domain format
	aliasServiceName := service.Name
	headlessServiceName := fmt.Sprintf("sd-%s", service.Name)

	aliasService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      aliasServiceName,
			Namespace: k8sNamespace,
			Labels: map[string]string{
				"kecs.io/service-discovery": "true",
				"kecs.io/type":              "alias",
				"kecs.io/namespace":         namespace.Name,
				"kecs.io/service":           service.Name,
			},
			Annotations: map[string]string{
				"kecs.io/service-id":   service.ID,
				"kecs.io/namespace-id": service.NamespaceID,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", headlessServiceName, k8sNamespace),
		},
	}

	// Create or update the alias service
	existingService, err := m.kubeClient.CoreV1().Services(k8sNamespace).Get(ctx, aliasServiceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			if _, err := m.kubeClient.CoreV1().Services(k8sNamespace).Create(ctx, aliasService, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create alias service: %w", err)
			}
			logging.Info("Created DNS alias service", "name", aliasServiceName, "namespace", k8sNamespace)
		} else {
			return fmt.Errorf("failed to get alias service: %w", err)
		}
	} else {
		// Update existing service
		existingService.Spec.ExternalName = aliasService.Spec.ExternalName
		if _, err := m.kubeClient.CoreV1().Services(k8sNamespace).Update(ctx, existingService, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to update alias service: %w", err)
		}
		logging.Debug("Updated DNS alias service", "name", aliasServiceName, "namespace", k8sNamespace)
	}

	return nil
}

// updateServiceDNSAlias updates the ExternalName Service to point to the actual ECS service
func (m *manager) updateServiceDNSAlias(ctx context.Context, namespace *Namespace, service *Service, ecsServiceFQDN string) error {
	// Get the Kubernetes namespace for this Service Discovery namespace
	k8sNamespace := "default" // Service Discovery services are in the default namespace

	// Get the existing ExternalName service
	aliasServiceName := service.Name
	aliasService, err := m.kubeClient.CoreV1().Services(k8sNamespace).Get(ctx, aliasServiceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ExternalName service: %w", err)
	}

	// Update the ExternalName to point to the actual ECS service
	aliasService.Spec.ExternalName = ecsServiceFQDN

	// Update the service
	_, err = m.kubeClient.CoreV1().Services(k8sNamespace).Update(ctx, aliasService, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ExternalName service: %w", err)
	}

	logging.Info("Updated ExternalName Service",
		"service", aliasServiceName,
		"namespace", k8sNamespace,
		"externalName", ecsServiceFQDN)

	return nil
}

// removeServiceDNSAlias removes the DNS alias for a service
func (m *manager) removeServiceDNSAlias(ctx context.Context, namespace *Namespace, service *Service) error {
	if m.kubeClient == nil {
		return nil
	}

	// Get Kubernetes namespace
	k8sNamespace := m.dnsToK8sNamespace[namespace.Name]
	if k8sNamespace == "" {
		k8sNamespace = "default"
	}

	// Delete the alias service
	err := m.kubeClient.CoreV1().Services(k8sNamespace).Delete(ctx, service.Name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete alias service: %w", err)
	}

	logging.Info("Removed DNS alias service", "name", service.Name, "namespace", k8sNamespace)
	return nil
}
