package elbv2

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// K8sIntegration implements the Integration interface using Kubernetes Services
// instead of actual ELBv2 API calls. This avoids the need for LocalStack Pro.
type K8sIntegration struct {
	region        string
	accountID     string
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
	ruleManager   *RuleManager
	store         storage.ELBv2Store // Database storage for persistence

	// In-memory storage for load balancers and target groups
	// This acts as a cache, with database as the source of truth
	mu            sync.RWMutex
	loadBalancers map[string]*LoadBalancer
	targetGroups  map[string]*TargetGroup
	listeners     map[string]*Listener
	targetHealth  map[string]map[string]*TargetHealth // targetGroupArn -> targetId -> health
}

// NewK8sIntegration creates a new Kubernetes-based ELBv2 integration
func NewK8sIntegration(region, accountID string) *K8sIntegration {
	integration := &K8sIntegration{
		region:        region,
		accountID:     accountID,
		kubeClient:    nil, // Will be set later when needed
		dynamicClient: nil, // Will be set later when needed
		store:         nil, // Will be set later when needed
		loadBalancers: make(map[string]*LoadBalancer),
		targetGroups:  make(map[string]*TargetGroup),
		listeners:     make(map[string]*Listener),
		targetHealth:  make(map[string]map[string]*TargetHealth),
	}
	// RuleManager will be initialized when dynamicClient is set
	return integration
}

// SetKubernetesClients sets the Kubernetes clients for the integration
func (i *K8sIntegration) SetKubernetesClients(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) {
	i.kubeClient = kubeClient
	i.dynamicClient = dynamicClient
	// RuleManager requires storage, which we don't have here
	// It will be initialized separately if needed
}

// SetStorage sets the storage backend for persistence
func (i *K8sIntegration) SetStorage(store storage.ELBv2Store) {
	i.store = store
}

// CreateLoadBalancer creates a virtual load balancer and deploys Traefik
func (i *K8sIntegration) CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*LoadBalancer, error) {
	logging.Debug("Creating load balancer with Traefik deployment", "name", name)

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:loadbalancer/app/%s/%s",
		i.region, i.accountID, name, generateID())

	// Create virtual load balancer
	lb := &LoadBalancer{
		Arn:               arn,
		Name:              name,
		DNSName:           fmt.Sprintf("%s-%s.%s.elb.amazonaws.com", name, generateID(), i.region),
		State:             "active",
		Type:              "application",
		Scheme:            "internet-facing",
		VpcId:             "vpc-default",
		SecurityGroups:    securityGroups,
		CreatedTime:       time.Now().Format(time.RFC3339),
		AvailabilityZones: []AvailabilityZone{},
	}

	// Add availability zones based on subnets
	for idx, subnet := range subnets {
		lb.AvailabilityZones = append(lb.AvailabilityZones, AvailabilityZone{
			ZoneName: fmt.Sprintf("%s%c", i.region, 'a'+idx),
			SubnetId: subnet,
		})
	}

	// Store in memory with lock
	i.mu.Lock()
	i.loadBalancers[arn] = lb
	i.mu.Unlock()

	// Save to database if available
	if i.store != nil {
		dbLB := &storage.ELBv2LoadBalancer{
			ARN:            lb.Arn,
			Name:           lb.Name,
			DNSName:        lb.DNSName,
			State:          lb.State,
			Type:           lb.Type,
			Scheme:         lb.Scheme,
			VpcID:          lb.VpcId,
			Subnets:        subnets,
			SecurityGroups: securityGroups,
			Region:         i.region,
			AccountID:      i.accountID,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Add availability zones
		for _, az := range lb.AvailabilityZones {
			dbLB.AvailabilityZones = append(dbLB.AvailabilityZones, az.ZoneName)
		}

		if err := i.store.CreateLoadBalancer(ctx, dbLB); err != nil {
			logging.Warn("Failed to save load balancer to database", "error", err, "arn", arn)
			// Continue anyway - in-memory is still working
		}
	}

	// Note: Traefik is now deployed globally, not per-LoadBalancer
	// The global Traefik instance handles all ALB traffic based on Host headers
	logging.Debug("Created virtual load balancer", "arn", arn, "dnsName", lb.DNSName)
	return lb, nil
}

// DeleteLoadBalancer deletes a virtual load balancer
func (i *K8sIntegration) DeleteLoadBalancer(ctx context.Context, arn string) error {
	logging.Debug("Deleting virtual load balancer", "arn", arn)

	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.loadBalancers[arn]; !exists {
		return fmt.Errorf("load balancer not found: %s", arn)
	}

	delete(i.loadBalancers, arn)
	return nil
}

// CreateTargetGroup creates a virtual target group and Kubernetes resources
func (i *K8sIntegration) CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*TargetGroup, error) {
	logging.Debug("Creating target group with Kubernetes resources", "name", name)

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:targetgroup/%s/%s",
		i.region, i.accountID, name, generateID())

	// Create virtual target group
	tg := &TargetGroup{
		Arn:                     arn,
		Name:                    name,
		Port:                    port,
		Protocol:                protocol,
		VpcId:                   vpcId,
		TargetType:              "ip",
		HealthCheckPath:         "/",
		HealthCheckPort:         fmt.Sprintf("%d", port),
		HealthCheckProtocol:     protocol,
		UnhealthyThresholdCount: 3,
		HealthyThresholdCount:   2,
	}

	// Deploy Kubernetes resources for target group
	if err := i.deployTargetGroupResources(ctx, name, arn, port, protocol); err != nil {
		return nil, fmt.Errorf("failed to deploy target group resources: %w", err)
	}

	// Store in memory with lock
	i.mu.Lock()
	i.targetGroups[arn] = tg
	i.targetHealth[arn] = make(map[string]*TargetHealth)
	i.mu.Unlock()

	logging.Debug("Created target group with Kubernetes resources", "arn", arn)
	return tg, nil
}

// DeleteTargetGroup deletes a virtual target group
func (i *K8sIntegration) DeleteTargetGroup(ctx context.Context, arn string) error {
	logging.Debug("Deleting virtual target group", "arn", arn)

	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.targetGroups[arn]; !exists {
		return fmt.Errorf("target group not found: %s", arn)
	}

	delete(i.targetGroups, arn)
	delete(i.targetHealth, arn)
	return nil
}

// RegisterTargets registers targets with a virtual target group
func (i *K8sIntegration) RegisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	logging.Debug("Registering targets with virtual target group", "targetCount", len(targets), "targetGroupArn", targetGroupArn)

	i.mu.Lock()
	if _, exists := i.targetGroups[targetGroupArn]; !exists {
		i.mu.Unlock()
		return fmt.Errorf("target group not found: %s", targetGroupArn)
	}

	// Initialize target health map if needed
	if i.targetHealth[targetGroupArn] == nil {
		i.targetHealth[targetGroupArn] = make(map[string]*TargetHealth)
	}

	// Register each target
	for _, target := range targets {
		i.targetHealth[targetGroupArn][target.Id] = &TargetHealth{
			Target:      target,
			HealthState: "initial",
			Reason:      "Elb.RegistrationInProgress",
			Description: "Target registration is in progress",
		}

		// Simulate health check transition
		go func(tgArn, targetId string) {
			time.Sleep(5 * time.Second)
			i.mu.Lock()
			if health, exists := i.targetHealth[tgArn][targetId]; exists {
				health.HealthState = "healthy"
				health.Reason = ""
				health.Description = "Health checks passed"
			}
			i.mu.Unlock()
		}(targetGroupArn, target.Id)
	}
	i.mu.Unlock()

	return nil
}

// DeregisterTargets deregisters targets from a virtual target group
func (i *K8sIntegration) DeregisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	logging.Debug("Deregistering targets from virtual target group", "targetCount", len(targets), "targetGroupArn", targetGroupArn)

	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.targetGroups[targetGroupArn]; !exists {
		return fmt.Errorf("target group not found: %s", targetGroupArn)
	}

	// Remove each target
	for _, target := range targets {
		delete(i.targetHealth[targetGroupArn], target.Id)
	}

	return nil
}

// CreateListener creates a virtual listener and updates Traefik configuration
func (i *K8sIntegration) CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*Listener, error) {
	logging.Debug("Creating listener for load balancer", "port", port, "loadBalancerArn", loadBalancerArn)

	// Extract load balancer name from ARN
	// ARN format: arn:aws:elasticloadbalancing:region:account:loadbalancer/app/name/id
	lbName := ""
	if parts := strings.Split(loadBalancerArn, "/"); len(parts) >= 3 {
		lbName = parts[len(parts)-2]
	} else {
		return nil, fmt.Errorf("invalid load balancer ARN format: %s", loadBalancerArn)
	}

	i.mu.RLock()
	lb, exists := i.loadBalancers[loadBalancerArn]
	if !exists {
		i.mu.RUnlock()

		// Try to get from database if available
		if i.store != nil {
			dbLB, err := i.store.GetLoadBalancer(ctx, loadBalancerArn)
			if err == nil && dbLB != nil {
				// Convert from storage type to our internal type
				lb = &LoadBalancer{
					Arn:     dbLB.ARN,
					Name:    dbLB.Name,
					DNSName: dbLB.DNSName,
					State:   dbLB.State,
					Type:    dbLB.Type,
					Scheme:  dbLB.Scheme,
					VpcId:   dbLB.VpcID,
				}

				// Cache it in memory
				i.mu.Lock()
				i.loadBalancers[loadBalancerArn] = lb
				i.mu.Unlock()
			}
		}

		// If still not found, create a minimal entry for DNS name generation
		if lb == nil {
			i.mu.Lock()
			lb = &LoadBalancer{
				Arn:     loadBalancerArn,
				Name:    lbName,
				DNSName: fmt.Sprintf("%s-%s.%s.elb.amazonaws.com", lbName, generateID(), i.region),
			}
			i.loadBalancers[loadBalancerArn] = lb
			i.mu.Unlock()
		}

		i.mu.RLock()
	}

	// Extract target group name from ARN if provided
	var targetGroupName string
	if targetGroupArn != "" {
		tg, exists := i.targetGroups[targetGroupArn]
		if !exists {
			// Extract target group name from ARN
			// ARN format: arn:aws:elasticloadbalancing:region:account:targetgroup/name/id
			if parts := strings.Split(targetGroupArn, "/"); len(parts) >= 2 {
				targetGroupName = parts[len(parts)-2]
			} else {
				i.mu.RUnlock()
				return nil, fmt.Errorf("invalid target group ARN format: %s", targetGroupArn)
			}
		} else {
			targetGroupName = tg.Name
		}
	}
	i.mu.RUnlock()

	// Generate ARN
	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:listener/app/%s/%s",
		i.region, i.accountID, getResourceName(loadBalancerArn), generateID())

	// Create virtual listener
	listener := &Listener{
		Arn:             arn,
		LoadBalancerArn: loadBalancerArn,
		Port:            port,
		Protocol:        protocol,
		DefaultActions: []Action{
			{
				Type:           "forward",
				TargetGroupArn: targetGroupArn,
				Order:          1,
			},
		},
	}

	// Update Traefik configuration with new listener
	if err := i.updateTraefikConfigForListener(ctx, lbName, arn, port, protocol, targetGroupName); err != nil {
		return nil, fmt.Errorf("failed to update Traefik configuration: %w", err)
	}

	// Check if k3d port mapping exists for this listener
	// Note: Port mappings should be pre-configured when creating k3d cluster
	hostPort := calculateHostPort(port, lbName)
	nodePort := getTraefikNodePort(port)
	clusterName := getClusterNameFromEnvironment()

	logging.Info("Checking k3d port mapping for ALB listener",
		"listenerPort", port,
		"hostPort", hostPort,
		"nodePort", nodePort,
		"albName", lbName)

	// Try to add port mapping (will fail if not pre-configured in k3d)
	if err := i.addK3dPortMapping(ctx, clusterName, hostPort, nodePort); err != nil {
		// This is expected if port is not in pre-configured range
		logging.Debug("k3d port mapping not available",
			"error", err,
			"hostPort", hostPort,
			"nodePort", nodePort,
			"note", "Port may not be in pre-configured range. Use kubectl port-forward if needed.")
	} else {
		logging.Info("ALB listener is accessible directly",
			"url", fmt.Sprintf("http://localhost:%d", hostPort),
			"albDNS", lb.DNSName,
			"usage", fmt.Sprintf("curl -H 'Host: %s' http://localhost:%d/", lb.DNSName, hostPort))
	}

	// Store in memory with lock
	i.mu.Lock()
	i.listeners[arn] = listener
	i.mu.Unlock()

	logging.Debug("Created listener with Traefik configuration", "arn", arn)
	return listener, nil
}

// DeleteListener deletes a virtual listener
func (i *K8sIntegration) DeleteListener(ctx context.Context, arn string) error {
	logging.Debug("Deleting virtual listener", "arn", arn)

	i.mu.Lock()
	listener, exists := i.listeners[arn]
	if !exists {
		i.mu.Unlock()
		return fmt.Errorf("listener not found: %s", arn)
	}

	// Get load balancer info for IngressRoute deletion
	lb, lbExists := i.loadBalancers[listener.LoadBalancerArn]
	var lbName string
	if lbExists {
		lbName = lb.Name
	} else {
		// Extract load balancer name from ARN if not in memory
		// ARN format: arn:aws:elasticloadbalancing:region:account:loadbalancer/app/name/id
		if parts := strings.Split(listener.LoadBalancerArn, "/"); len(parts) >= 3 {
			lbName = parts[len(parts)-2]
		}
	}

	delete(i.listeners, arn)
	i.mu.Unlock()

	// Delete Ingress if we have the necessary info
	if lbName != "" && i.kubeClient != nil {
		if err := i.deleteGlobalIngress(ctx, lbName, listener.Port); err != nil {
			logging.Debug("Failed to delete Ingress for listener", "arn", arn, "error", err)
			// Don't fail the operation if Ingress deletion fails
		}
	}

	return nil
}

// GetLoadBalancer gets virtual load balancer details
func (i *K8sIntegration) GetLoadBalancer(ctx context.Context, arn string) (*LoadBalancer, error) {
	logging.Debug("Getting virtual load balancer", "arn", arn)

	i.mu.RLock()
	lb, exists := i.loadBalancers[arn]
	i.mu.RUnlock()

	if !exists && i.store != nil {
		// Try to get from database
		dbLB, err := i.store.GetLoadBalancer(ctx, arn)
		if err == nil && dbLB != nil {
			// Convert from storage type to our internal type
			lb = &LoadBalancer{
				Arn:            dbLB.ARN,
				Name:           dbLB.Name,
				DNSName:        dbLB.DNSName,
				State:          dbLB.State,
				Type:           dbLB.Type,
				Scheme:         dbLB.Scheme,
				VpcId:          dbLB.VpcID,
				SecurityGroups: dbLB.SecurityGroups,
				CreatedTime:    dbLB.CreatedAt.Format(time.RFC3339),
			}

			// Convert availability zones
			for idx, azName := range dbLB.AvailabilityZones {
				var subnet string
				if idx < len(dbLB.Subnets) {
					subnet = dbLB.Subnets[idx]
				}
				lb.AvailabilityZones = append(lb.AvailabilityZones, AvailabilityZone{
					ZoneName: azName,
					SubnetId: subnet,
				})
			}

			// Cache it in memory
			i.mu.Lock()
			i.loadBalancers[arn] = lb
			i.mu.Unlock()

			return lb, nil
		}
	}

	if lb == nil {
		return nil, fmt.Errorf("load balancer not found: %s", arn)
	}

	return lb, nil
}

// GetTargetHealth gets the health status of virtual targets
func (i *K8sIntegration) GetTargetHealth(ctx context.Context, targetGroupArn string) ([]TargetHealth, error) {
	logging.Debug("Getting target health for virtual target group", "targetGroupArn", targetGroupArn)

	i.mu.RLock()
	defer i.mu.RUnlock()

	if _, exists := i.targetGroups[targetGroupArn]; !exists {
		return nil, fmt.Errorf("target group not found: %s", targetGroupArn)
	}

	healthMap, exists := i.targetHealth[targetGroupArn]
	if !exists {
		return []TargetHealth{}, nil
	}

	results := make([]TargetHealth, 0, len(healthMap))
	for _, health := range healthMap {
		results = append(results, *health)
	}

	return results, nil
}

// CheckTargetHealthWithK8s performs health check using Kubernetes pod status
func (i *K8sIntegration) CheckTargetHealthWithK8s(ctx context.Context, targetIP string, targetPort int32, targetGroupArn string) (string, error) {
	logging.Debug("Checking target health with Kubernetes", "targetIP", targetIP, "targetPort", targetPort)

	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, falling back to basic connectivity check")
		return i.performBasicConnectivityCheck(targetIP, targetPort)
	}

	// Find pod by IP address
	pod, err := i.findPodByIP(ctx, targetIP)
	if err != nil {
		logging.Debug("Failed to find pod with IP", "targetIP", targetIP, "error", err)
		// Fallback to basic connectivity check if pod not found
		return i.performBasicConnectivityCheck(targetIP, targetPort)
	}

	if pod == nil {
		logging.Debug("No pod found with IP, performing basic connectivity check", "targetIP", targetIP)
		return i.performBasicConnectivityCheck(targetIP, targetPort)
	}

	// Check pod readiness status
	return i.checkPodReadiness(pod, targetPort)
}

// findPodByIP finds a pod by its IP address across all namespaces
func (i *K8sIntegration) findPodByIP(ctx context.Context, targetIP string) (*corev1.Pod, error) {
	// List pods across all namespaces
	pods, err := i.kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	for _, pod := range pods.Items {
		if pod.Status.PodIP == targetIP {
			return &pod, nil
		}
	}

	return nil, nil // Pod not found
}

// checkPodReadiness checks if a pod is ready and healthy
func (i *K8sIntegration) checkPodReadiness(pod *corev1.Pod, targetPort int32) (string, error) {
	// Check pod phase first
	if pod.Status.Phase != corev1.PodRunning {
		logging.Debug("Pod is not running", "namespace", pod.Namespace, "name", pod.Name, "phase", pod.Status.Phase)
		return "unhealthy", nil
	}

	// Check pod readiness conditions
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			if condition.Status == corev1.ConditionTrue {
				logging.Debug("Pod is ready", "namespace", pod.Namespace, "name", pod.Name)

				// Additionally check if the target port is exposed by the pod
				if i.isPodPortExposed(pod, targetPort) {
					return "healthy", nil
				} else {
					logging.Debug("Pod does not expose target port", "namespace", pod.Namespace, "name", pod.Name, "targetPort", targetPort)
					return "unhealthy", nil
				}
			} else {
				logging.Debug("Pod is not ready", "namespace", pod.Namespace, "name", pod.Name, "reason", condition.Reason)
				return "unhealthy", nil
			}
		}
	}

	// If no readiness condition found, consider it unhealthy
	logging.Debug("Pod has no readiness condition", "namespace", pod.Namespace, "name", pod.Name)
	return "unhealthy", nil
}

// isPodPortExposed checks if a pod exposes the given port
func (i *K8sIntegration) isPodPortExposed(pod *corev1.Pod, targetPort int32) bool {
	for _, container := range pod.Spec.Containers {
		for _, port := range container.Ports {
			if port.ContainerPort == targetPort {
				return true
			}
		}
	}
	return false
}

// performBasicConnectivityCheck performs a basic TCP connectivity check
func (i *K8sIntegration) performBasicConnectivityCheck(targetIP string, targetPort int32) (string, error) {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, targetPort), timeout)
	if err != nil {
		logging.Debug("Basic connectivity check failed", "targetIP", targetIP, "targetPort", targetPort, "error", err)
		return "unhealthy", nil
	}
	conn.Close()
	logging.Debug("Basic connectivity check passed", "targetIP", targetIP, "targetPort", targetPort)
	return "healthy", nil
}

// Helper functions

func generateID() string {
	// Simple ID generation for demo purposes
	// In production, use a proper UUID generator
	return fmt.Sprintf("%d", time.Now().UnixNano())[:16]
}

func getResourceName(arn string) string {
	// Extract resource name from ARN
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return "unknown"
}

// deployTraefikForLoadBalancer deploys Traefik resources for a load balancer
func (i *K8sIntegration) deployTraefikForLoadBalancer(ctx context.Context, lbName, lbArn string) error {
	if i.kubeClient == nil {
		// If no kubeClient is available, just log and continue
		logging.Debug("No kubeClient available, skipping Traefik deployment for load balancer", "lbName", lbName)
		return nil
	}

	namespace := "kecs-system"
	traefikName := fmt.Sprintf("traefik-elbv2-%s", lbName)

	// Create namespace if it doesn't exist
	if err := i.createNamespaceIfNotExists(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// Create ServiceAccount
	if err := i.createServiceAccount(ctx, namespace, traefikName, lbName, lbArn); err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}

	// Create ConfigMap
	if err := i.createConfigMap(ctx, namespace, traefikName, lbName, lbArn); err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}

	// Create Deployment
	if err := i.createDeployment(ctx, namespace, traefikName, lbName, lbArn); err != nil {
		return fmt.Errorf("failed to create Deployment: %w", err)
	}

	// Create Service
	if err := i.createService(ctx, namespace, traefikName, lbName, lbArn); err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}

	logging.Debug("Successfully deployed Traefik resources for load balancer", "lbName", lbName)
	return nil
}

// createNamespaceIfNotExists creates the namespace if it doesn't exist
func (i *K8sIntegration) createNamespaceIfNotExists(ctx context.Context, namespace string) error {
	_, err := i.kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		// Namespace doesn't exist, create it
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		_, err = i.kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create namespace %s: %w", namespace, err)
		}
	}
	return nil
}

// createServiceAccount creates a ServiceAccount for Traefik with load balancer annotations
func (i *K8sIntegration) createServiceAccount(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traefikName,
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/elbv2-load-balancer-arn":  lbArn,
				"kecs.io/elbv2-proxy-type":         "load-balancer",
			},
			Labels: map[string]string{
				"app":                              traefikName,
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/component":                "elbv2-proxy",
			},
		},
	}

	_, err := i.kubeClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ServiceAccount: %w", err)
	}
	return nil
}

// createConfigMap creates a ConfigMap for Traefik configuration
func (i *K8sIntegration) createConfigMap(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-config", traefikName),
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/elbv2-load-balancer-arn":  lbArn,
				"kecs.io/elbv2-proxy-type":         "load-balancer",
			},
			Labels: map[string]string{
				"app":                              traefikName,
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/component":                "elbv2-proxy",
			},
		},
		Data: map[string]string{
			"traefik.yml": `
api:
  dashboard: true
  debug: true
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
providers:
  kubernetesIngress: {}
log:
  level: INFO
`,
		},
	}

	_, err := i.kubeClient.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create ConfigMap: %w", err)
	}
	return nil
}

// createDeployment creates a Deployment for Traefik
func (i *K8sIntegration) createDeployment(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traefikName,
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/elbv2-load-balancer-arn":  lbArn,
				"kecs.io/elbv2-proxy-type":         "load-balancer",
			},
			Labels: map[string]string{
				"app":                              traefikName,
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/component":                "elbv2-proxy",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": traefikName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                              traefikName,
						"kecs.io/elbv2-load-balancer-name": lbName,
						"kecs.io/component":                "elbv2-proxy",
					},
					Annotations: map[string]string{
						"kecs.io/elbv2-load-balancer-name": lbName,
						"kecs.io/elbv2-load-balancer-arn":  lbArn,
						"kecs.io/elbv2-proxy-type":         "load-balancer",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: traefikName,
					Containers: []corev1.Container{
						{
							Name:  "traefik",
							Image: "traefik:v3.5.0",
							Args: []string{
								"--configfile=/config/traefik.yml",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "web",
									ContainerPort: 80,
								},
								{
									Name:          "websecure",
									ContainerPort: 443,
								},
								{
									Name:          "dashboard",
									ContainerPort: 8080,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/config",
									ReadOnly:  true,
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: mustParseResource("128Mi"),
									corev1.ResourceCPU:    mustParseResource("500m"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: mustParseResource("64Mi"),
									corev1.ResourceCPU:    mustParseResource("100m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-config", traefikName),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := i.kubeClient.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Deployment: %w", err)
	}
	return nil
}

// createService creates a Service for Traefik
func (i *K8sIntegration) createService(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      traefikName,
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/elbv2-load-balancer-arn":  lbArn,
				"kecs.io/elbv2-proxy-type":         "load-balancer",
			},
			Labels: map[string]string{
				"app":                              traefikName,
				"kecs.io/elbv2-load-balancer-name": lbName,
				"kecs.io/component":                "elbv2-proxy",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": traefikName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       80,
					TargetPort: intstr.FromInt(80),
					NodePort:   30080,
				},
				{
					Name:       "websecure",
					Port:       443,
					TargetPort: intstr.FromInt(443),
					NodePort:   30443,
				},
				{
					Name:       "dashboard",
					Port:       8080,
					TargetPort: intstr.FromInt(8080),
					NodePort:   30808,
				},
			},
		},
	}

	_, err := i.kubeClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}
	return nil
}

// Helper function to parse resource requirements
func mustParseResource(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(err)
	}
	return q
}

// deployTargetGroupResources deploys Kubernetes resources for a target group
func (i *K8sIntegration) deployTargetGroupResources(ctx context.Context, tgName, tgArn string, port int32, protocol string) error {
	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, skipping target group resources deployment", "tgName", tgName)
		return nil
	}

	// Don't create the Service here anymore
	// It will be created later when we know which namespace (ECS cluster) it belongs to
	// For now, just log that we're deferring Service creation

	logging.Debug("Deferring Service creation for target group until ECS service is created",
		"tgName", tgName,
		"tgArn", tgArn,
		"port", port,
		"protocol", protocol)

	// Store target group metadata for later use
	// This metadata will be used when creating the Service in the correct namespace
	if i.store != nil {
		// Store in database for persistence
		tgRecord := &storage.ELBv2TargetGroup{
			ARN:       tgArn,
			Name:      tgName,
			Port:      port,
			Protocol:  protocol,
			Region:    i.region,
			AccountID: i.accountID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := i.store.CreateTargetGroup(ctx, tgRecord); err != nil {
			logging.Warn("Failed to store target group metadata", "error", err, "tgName", tgName)
			// Continue anyway - we can still work without persistence
		}
	}

	return nil
}

// CreateTargetGroupServiceInNamespace creates a Kubernetes Service for a target group in a specific namespace
// This is called when an ECS service is created with a load balancer configuration
func (i *K8sIntegration) CreateTargetGroupServiceInNamespace(ctx context.Context, targetGroupArn, namespace string) error {
	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, skipping target group Service creation", "targetGroupArn", targetGroupArn)
		return nil
	}

	// Get target group from memory first
	i.mu.RLock()
	tg, exists := i.targetGroups[targetGroupArn]
	i.mu.RUnlock()

	// If not found in memory, try to get from database storage
	if !exists && i.store != nil {
		dbTG, err := i.store.GetTargetGroup(ctx, targetGroupArn)
		if err == nil && dbTG != nil {
			// Convert from storage type to our internal type
			tg = &TargetGroup{
				Arn:      dbTG.ARN,
				Name:     dbTG.Name,
				Port:     dbTG.Port,
				Protocol: dbTG.Protocol,
				VpcId:    "vpc-default", // Default VPC for consistency
			}

			// Cache it in memory for future use
			i.mu.Lock()
			i.targetGroups[targetGroupArn] = tg
			if i.targetHealth[targetGroupArn] == nil {
				i.targetHealth[targetGroupArn] = make(map[string]*TargetHealth)
			}
			i.mu.Unlock()

			exists = true
		}
	}

	if !exists {
		return fmt.Errorf("target group not found: %s", targetGroupArn)
	}

	serviceName := fmt.Sprintf("tg-%s", tg.Name)

	// Check if Service already exists in the namespace
	existingService, err := i.kubeClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err == nil && existingService != nil {
		logging.Debug("Service for target group already exists in namespace",
			"serviceName", serviceName,
			"namespace", namespace,
			"targetGroup", tg.Name)
		return nil
	}

	// Create a Service for the target group in the ECS cluster's namespace
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-target-group-name":     tg.Name,
				"kecs.io/elbv2-target-group-arn":      targetGroupArn,
				"kecs.io/elbv2-target-group-protocol": tg.Protocol,
			},
			Labels: map[string]string{
				"kecs.io/elbv2-target-group-name": tg.Name,
				"kecs.io/component":               "target-group",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"kecs.io/elbv2-target-group-name": tg.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "main",
					Port:       tg.Port,
					TargetPort: intstr.FromInt(int(tg.Port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// Create the service
	_, err = i.kubeClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Service for target group in namespace %s: %w", namespace, err)
	}

	logging.Info("Created Service for target group in ECS cluster namespace",
		"serviceName", serviceName,
		"namespace", namespace,
		"targetGroup", tg.Name)

	// Update the IngressRoute to point to the actual service in the correct namespace
	// This implements the dynamic service discovery pattern
	if err := i.UpdateIngressRouteForService(ctx, targetGroupArn, serviceName, namespace); err != nil {
		logging.Warn("Failed to update IngressRoute for service",
			"error", err,
			"serviceName", serviceName,
			"namespace", namespace,
			"targetGroupArn", targetGroupArn)
		// Don't fail the entire operation if IngressRoute update fails
	}

	return nil
}

// updateTraefikConfigForListener creates Ingress for global Traefik with Host header routing
func (i *K8sIntegration) updateTraefikConfigForListener(ctx context.Context, lbName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, skipping Ingress creation for listener", "listenerArn", listenerArn)
		return nil
	}

	// Generate a DNS name for the load balancer
	// Since we're using Host header routing, the actual DNS name format is important
	// Format: <lb-name>-<hash>.region.elb.amazonaws.com
	lbDNSName := fmt.Sprintf("%s-%s.%s.elb.amazonaws.com", lbName, generateID()[:8], i.region)

	// Create Ingress for the global Traefik instance with Host header routing
	if targetGroupName != "" {
		if err := i.createGlobalIngress(ctx, lbName, lbDNSName, listenerArn, port, protocol, targetGroupName); err != nil {
			return fmt.Errorf("failed to create Ingress for global Traefik: %w", err)
		}
	}

	logging.Debug("Created Ingress for global Traefik", "lbName", lbName, "host", lbDNSName, "port", port)
	return nil
}

// createIngressRoute creates a Traefik IngressRoute CRD for routing to target groups
func (i *K8sIntegration) createIngressRoute(ctx context.Context, lbName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.dynamicClient == nil {
		logging.Debug("No dynamicClient available, skipping IngressRoute creation")
		return nil
	}

	namespace := "kecs-system"
	// Generate a safe name for the IngressRoute
	ingressRouteName := fmt.Sprintf("listener-%s-%d", sanitizeName(lbName), port)

	// Create the IngressRoute unstructured object
	ingressRoute := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "traefik.io/v1alpha1",
			"kind":       "IngressRoute",
			"metadata": map[string]interface{}{
				"name":      ingressRouteName,
				"namespace": namespace,
				"annotations": map[string]interface{}{
					"kecs.io/elbv2-listener-arn":  listenerArn,
					"kecs.io/elbv2-load-balancer": lbName,
					"kecs.io/elbv2-target-group":  targetGroupName,
					"kecs.io/pending-namespace":   "true", // Mark as waiting for namespace resolution
				},
				"labels": map[string]interface{}{
					"kecs.io/elbv2-load-balancer": lbName,
					"kecs.io/component":           "elbv2-listener",
				},
			},
			"spec": map[string]interface{}{
				"entryPoints": []string{fmt.Sprintf("listener%d", port)},
				"routes": []interface{}{
					map[string]interface{}{
						"match":    "PathPrefix(`/`)", // Default catch-all route
						"kind":     "Rule",
						"priority": 50000, // Very low priority for default rule
						"services": []interface{}{
							map[string]interface{}{
								"name": "placeholder-service", // Placeholder until actual service is created
								"port": port,
								// Namespace will be added when service is created
							},
						},
					},
				},
			},
		},
	}

	// Define the GVR for IngressRoute
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	// Create the IngressRoute
	_, err := i.dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, ingressRoute, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create IngressRoute: %w", err)
	}

	logging.Debug("Created IngressRoute for listener routing to target group", "ingressRouteName", ingressRouteName, "port", port, "targetGroupName", targetGroupName)
	return nil
}

// sanitizeName converts a name to be suitable for Kubernetes resource names
func sanitizeName(name string) string {
	// Replace non-alphanumeric characters with hyphens
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, "_", "-")
	result = strings.ReplaceAll(result, " ", "-")
	// Remove any non-alphanumeric characters except hyphens
	return result
}

// extractTargetGroupName extracts the target group name from an ARN
// ARN format: arn:aws:elasticloadbalancing:region:account:targetgroup/name/id
func extractTargetGroupName(arn string) string {
	parts := strings.Split(arn, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// deleteIngressRoute deletes a Traefik IngressRoute CRD
func (i *K8sIntegration) deleteIngressRoute(ctx context.Context, lbName string, port int32) error {
	if i.dynamicClient == nil {
		logging.Debug("No dynamicClient available, skipping IngressRoute deletion")
		return nil
	}

	namespace := "kecs-system"
	ingressRouteName := fmt.Sprintf("listener-%s-%d", sanitizeName(lbName), port)

	// Define the GVR for IngressRoute
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	// Delete the IngressRoute
	err := i.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, ingressRouteName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete IngressRoute: %w", err)
	}

	logging.Debug("Deleted IngressRoute", "ingressRouteName", ingressRouteName)
	return nil
}

// createGlobalIngress creates a standard Kubernetes Ingress for the global Traefik instance
// It uses Host header-based routing to distinguish between different ALBs
func (i *K8sIntegration) createGlobalIngress(ctx context.Context, lbName, lbDNSName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, skipping global Ingress creation")
		return nil
	}

	namespace := "kecs-system"
	// Generate a unique name for this ALB's Ingress
	ingressName := fmt.Sprintf("alb-%s-port-%d", sanitizeName(lbName), port)

	// Create path type for Ingress
	pathType := networkingv1.PathTypePrefix

	// Create the backend service port
	backendPort := networkingv1.ServiceBackendPort{
		Number: port,
	}

	// Create the Ingress object
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ingressName,
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-listener-arn":  listenerArn,
				"kecs.io/elbv2-load-balancer": lbName,
				"kecs.io/elbv2-target-group":  targetGroupName,
				"kecs.io/elbv2-dns-name":      lbDNSName,
			},
			Labels: map[string]string{
				"kecs.io/elbv2-load-balancer": lbName,
				"kecs.io/component":           "elbv2-listener",
				"kecs.io/traefik-scope":       "global",
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					// Use Host header to route to this specific ALB
					Host: lbDNSName,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: fmt.Sprintf("tg-%s", targetGroupName),
											Port: backendPort,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Check if Ingress already exists
	existing, err := i.kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, ingressName, metav1.GetOptions{})
	if err == nil {
		// Update existing Ingress
		existing.Spec = ingress.Spec
		_, err = i.kubeClient.NetworkingV1().Ingresses(namespace).Update(ctx, existing, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update global Ingress: %w", err)
		}
		logging.Debug("Updated global Ingress", "name", ingressName, "host", lbDNSName)
	} else {
		// Create new Ingress
		_, err = i.kubeClient.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create global Ingress: %w", err)
		}
		logging.Debug("Created global Ingress", "name", ingressName, "host", lbDNSName)
	}

	return nil
}

// deleteGlobalIngress deletes a standard Kubernetes Ingress
func (i *K8sIntegration) deleteGlobalIngress(ctx context.Context, lbName string, port int32) error {
	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, skipping global Ingress deletion")
		return nil
	}

	namespace := "kecs-system"
	ingressName := fmt.Sprintf("alb-%s-port-%d", sanitizeName(lbName), port)

	// Delete the Ingress
	err := i.kubeClient.NetworkingV1().Ingresses(namespace).Delete(ctx, ingressName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete global Ingress: %w", err)
	}

	logging.Debug("Deleted global Ingress", "name", ingressName)
	return nil
}

// updateIngressRoute updates an existing Traefik IngressRoute CRD
func (i *K8sIntegration) updateIngressRoute(ctx context.Context, lbName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.dynamicClient == nil {
		logging.Debug("No dynamicClient available, skipping IngressRoute update")
		return nil
	}

	namespace := "kecs-system"
	ingressRouteName := fmt.Sprintf("listener-%s-%d", sanitizeName(lbName), port)

	// Define the GVR for IngressRoute
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	// Try to get existing IngressRoute
	existingRoute, err := i.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, ingressRouteName, metav1.GetOptions{})
	if err != nil {
		// If not found, create a new one
		return i.createIngressRoute(ctx, lbName, listenerArn, port, protocol, targetGroupName)
	}

	// Update the existing IngressRoute
	existingRoute.Object["spec"] = map[string]interface{}{
		"entryPoints": []string{fmt.Sprintf("listener%d", port)},
		"routes": []interface{}{
			map[string]interface{}{
				"match": "PathPrefix(`/`)", // Default catch-all route
				"kind":  "Rule",
				"services": []interface{}{
					map[string]interface{}{
						"name": fmt.Sprintf("tg-%s", targetGroupName),
						"port": port,
					},
				},
			},
		},
	}

	// Update annotations
	metadata, ok := existingRoute.Object["metadata"].(map[string]interface{})
	if ok {
		annotations, ok := metadata["annotations"].(map[string]interface{})
		if !ok {
			annotations = make(map[string]interface{})
			metadata["annotations"] = annotations
		}
		annotations["kecs.io/elbv2-target-group"] = targetGroupName
		annotations["kecs.io/elbv2-listener-arn"] = listenerArn
	}

	_, err = i.dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, existingRoute, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update IngressRoute: %w", err)
	}

	logging.Debug("Updated IngressRoute for listener routing to target group", "ingressRouteName", ingressRouteName, "port", port, "targetGroupName", targetGroupName)
	return nil
}

// UpdateIngressRouteForService updates the Ingress when a service is created with a target group
// This implements the dynamic service discovery pattern using ExternalName services
func (i *K8sIntegration) UpdateIngressRouteForService(ctx context.Context, targetGroupArn, serviceName, namespace string) error {
	if i.kubeClient == nil {
		logging.Debug("No kubeClient available, skipping Ingress update for service")
		return nil
	}

	// Get the listener associated with this target group
	i.mu.RLock()
	var listenerArn string
	var listenerPort int32
	var lbName string

	// Find the listener that references this target group
	for lArn, listener := range i.listeners {
		for _, action := range listener.DefaultActions {
			if action.TargetGroupArn == targetGroupArn {
				listenerArn = lArn
				listenerPort = listener.Port

				// Get load balancer name
				if lb, exists := i.loadBalancers[listener.LoadBalancerArn]; exists {
					lbName = lb.Name
				}
				break
			}
		}
		if listenerArn != "" {
			break
		}
	}
	i.mu.RUnlock()

	if listenerArn == "" {
		logging.Debug("No listener found for target group, skipping Ingress update",
			"targetGroupArn", targetGroupArn)
		return nil
	}

	// Extract target group name from ARN
	targetGroupName := extractTargetGroupName(targetGroupArn)
	if targetGroupName == "" {
		return fmt.Errorf("failed to extract target group name from ARN: %s", targetGroupArn)
	}

	logging.Info("Updating Ingress for service creation",
		"targetGroupArn", targetGroupArn,
		"targetGroupName", targetGroupName,
		"serviceName", serviceName,
		"namespace", namespace,
		"listenerPort", listenerPort,
		"lbName", lbName)

	// Create an ExternalName service in kecs-system that points to the service in the target namespace
	// This is the cross-namespace service discovery solution
	externalServiceName := fmt.Sprintf("tg-%s", targetGroupName)
	externalService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      externalServiceName,
			Namespace: "kecs-system",
			Labels: map[string]string{
				"kecs.io/component":        "elbv2-target",
				"kecs.io/target-group":     targetGroupName,
				"kecs.io/target-namespace": namespace,
				"kecs.io/target-service":   serviceName,
			},
			Annotations: map[string]string{
				"kecs.io/target-namespace": namespace,
				"kecs.io/target-service":   serviceName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: fmt.Sprintf("tg-%s.%s.svc.cluster.local", targetGroupName, namespace),
		},
	}

	// Create or update the ExternalName service
	existingService, err := i.kubeClient.CoreV1().Services("kecs-system").Get(ctx, externalServiceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the service
			_, err = i.kubeClient.CoreV1().Services("kecs-system").Create(ctx, externalService, metav1.CreateOptions{})
			if err != nil {
				logging.Error("Failed to create ExternalName service",
					"error", err,
					"serviceName", externalServiceName)
				return fmt.Errorf("failed to create ExternalName service: %w", err)
			}
			logging.Info("Created ExternalName service for cross-namespace routing",
				"externalServiceName", externalServiceName,
				"targetService", serviceName,
				"targetNamespace", namespace)
		} else {
			return fmt.Errorf("failed to get ExternalName service: %w", err)
		}
	} else {
		// Update the existing service
		existingService.Spec.ExternalName = fmt.Sprintf("tg-%s.%s.svc.cluster.local", targetGroupName, namespace)
		if existingService.Annotations == nil {
			existingService.Annotations = make(map[string]string)
		}
		existingService.Annotations["kecs.io/target-namespace"] = namespace
		existingService.Annotations["kecs.io/target-service"] = serviceName
		existingService.Labels["kecs.io/target-namespace"] = namespace
		existingService.Labels["kecs.io/target-service"] = serviceName

		_, err = i.kubeClient.CoreV1().Services("kecs-system").Update(ctx, existingService, metav1.UpdateOptions{})
		if err != nil {
			logging.Error("Failed to update ExternalName service",
				"error", err,
				"serviceName", externalServiceName)
			return fmt.Errorf("failed to update ExternalName service: %w", err)
		}
		logging.Info("Updated ExternalName service for cross-namespace routing",
			"externalServiceName", externalServiceName,
			"targetService", serviceName,
			"targetNamespace", namespace)
	}

	// Update the Ingress annotations
	ingressName := fmt.Sprintf("alb-%s-port-%d", lbName, listenerPort)
	ingress, err := i.kubeClient.NetworkingV1().Ingresses("kecs-system").Get(ctx, ingressName, metav1.GetOptions{})
	if err != nil {
		logging.Warn("Failed to get Ingress for update",
			"error", err,
			"ingressName", ingressName)
		// Ingress doesn't exist yet, that's OK
		return nil
	}

	// Update annotations on the Ingress to reflect the resolved namespace
	if ingress.Annotations == nil {
		ingress.Annotations = make(map[string]string)
	}
	ingress.Annotations["kecs.io/target-namespace"] = namespace
	ingress.Annotations["kecs.io/target-service"] = serviceName

	// Update the Ingress
	_, err = i.kubeClient.NetworkingV1().Ingresses("kecs-system").Update(ctx, ingress, metav1.UpdateOptions{})
	if err != nil {
		logging.Error("Failed to update Ingress annotations",
			"error", err,
			"ingressName", ingressName)
		return fmt.Errorf("failed to update Ingress: %w", err)
	}

	logging.Info("Successfully updated Ingress and created ExternalName service for dynamic service discovery",
		"ingressName", ingressName,
		"externalServiceName", externalServiceName,
		"targetService", serviceName,
		"targetNamespace", namespace)

	return nil
}

// SyncRulesToListener synchronizes ELBv2 rules to Traefik IngressRoute
func (i *K8sIntegration) SyncRulesToListener(ctx context.Context, storageInstance interface{}, listenerArn string, lbName string, port int32) error {
	// Cast storage to the correct type
	storageImpl, ok := storageInstance.(storage.Storage)
	if !ok {
		return fmt.Errorf("invalid storage type")
	}

	// Initialize rule manager if not already done
	if i.ruleManager == nil && i.dynamicClient != nil {
		i.ruleManager = NewRuleManager(i.dynamicClient, storageImpl.ELBv2Store())
	}

	if i.ruleManager == nil {
		logging.Debug("No rule manager available, skipping rule sync")
		return nil
	}

	// Sync rules using the rule manager
	return i.ruleManager.SyncRulesForListener(ctx, storageImpl, listenerArn, lbName, port)
}

// addK3dPortMapping adds a port mapping to the k3d cluster's load balancer
func (i *K8sIntegration) addK3dPortMapping(ctx context.Context, clusterName string, hostPort, nodePort int32) error {
	// Skip if running in container mode (k3d command not available)
	if os.Getenv("KECS_CONTAINER_MODE") == "true" {
		logging.Info("Running in container mode, k3d port mapping should be pre-configured",
			"hostPort", hostPort,
			"nodePort", nodePort,
			"note", "Ensure k3d cluster was started with appropriate port mappings")
		return nil
	}

	// Format: k3d node edit k3d-<cluster>-serverlb --port-add <hostPort>:<nodePort>
	nodeName := fmt.Sprintf("k3d-%s-serverlb", clusterName)
	portMapping := fmt.Sprintf("%d:%d", hostPort, nodePort)

	logging.Info("Adding k3d port mapping for ALB listener",
		"node", nodeName,
		"mapping", portMapping)

	cmd := exec.CommandContext(ctx, "k3d", "node", "edit", nodeName, "--port-add", portMapping)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// Check if the error is because the port is already mapped
		if strings.Contains(string(output), "already") || strings.Contains(string(output), "exists") {
			logging.Info("Port mapping already exists",
				"hostPort", hostPort,
				"nodePort", nodePort)
			return nil
		}
		return fmt.Errorf("failed to add k3d port mapping: %w, output: %s", err, string(output))
	}

	logging.Info("Successfully added k3d port mapping",
		"hostPort", hostPort,
		"nodePort", nodePort)
	return nil
}

// calculateHostPort calculates the host port based on listener port and ALB name
func calculateHostPort(listenerPort int32, lbName string) int32 {
	// Default mapping for common ports
	switch listenerPort {
	case 80:
		return 8080
	case 443:
		return 8443
	case 8080:
		// For alternative ALBs, use 8088
		if !strings.Contains(lbName, "default") && !strings.Contains(lbName, "main") {
			return 8088
		}
		return 8080
	default:
		// For custom ports, try to use 8000 + last two digits
		// This provides a predictable mapping
		base := int32(8000)
		offset := listenerPort % 100
		return base + offset
	}
}

// getTraefikNodePort returns the Traefik NodePort for a given listener port
func getTraefikNodePort(listenerPort int32) int32 {
	// Standard port mappings
	switch listenerPort {
	case 80:
		return 30880 // Standard HTTP
	case 443:
		return 30443 // Standard HTTPS
	case 8080:
		return 30880 // Alternative HTTP (same NodePort)
	default:
		// For custom ports, map to 30800 + offset
		// This assumes ports 8000-8099 are mapped to NodePorts 30800-30899
		if listenerPort >= 8000 && listenerPort <= 8099 {
			return 30800 + (listenerPort - 8000)
		}
		// For other custom ports, use modulo to stay in range
		return 30800 + (listenerPort % 100)
	}
}

// getClusterNameFromEnvironment gets the k3d cluster name
func getClusterNameFromEnvironment() string {
	// Try to get from environment variable
	if clusterName := os.Getenv("KECS_CLUSTER_NAME"); clusterName != "" {
		return clusterName
	}
	// Default to "kecs-cluster"
	return "kecs-cluster"
}
