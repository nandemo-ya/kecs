package elbv2

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// k8sIntegration implements the Integration interface using Kubernetes Services
// instead of actual ELBv2 API calls. This avoids the need for LocalStack Pro.
type k8sIntegration struct {
	region    string
	accountID string
	kubeClient kubernetes.Interface
	dynamicClient dynamic.Interface

	// In-memory storage for load balancers and target groups
	// In production, this should be persisted
	mu            sync.RWMutex
	loadBalancers map[string]*LoadBalancer
	targetGroups  map[string]*TargetGroup
	listeners     map[string]*Listener
	targetHealth  map[string]map[string]*TargetHealth // targetGroupArn -> targetId -> health
}

// NewK8sIntegration creates a new Kubernetes-based ELBv2 integration
func NewK8sIntegration(region, accountID string) Integration {
	return &k8sIntegration{
		region:        region,
		accountID:     accountID,
		kubeClient:    nil, // Will be set later when needed
		dynamicClient: nil, // Will be set later when needed
		loadBalancers: make(map[string]*LoadBalancer),
		targetGroups:  make(map[string]*TargetGroup),
		listeners:     make(map[string]*Listener),
		targetHealth:  make(map[string]map[string]*TargetHealth),
	}
}

// CreateLoadBalancer creates a virtual load balancer and deploys Traefik
func (i *k8sIntegration) CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*LoadBalancer, error) {
	klog.V(2).Infof("Creating load balancer with Traefik deployment: %s", name)

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

	// Deploy Traefik for this load balancer
	if err := i.deployTraefikForLoadBalancer(ctx, name, arn); err != nil {
		return nil, fmt.Errorf("failed to deploy Traefik for load balancer %s: %w", name, err)
	}

	// Store in memory with lock
	i.mu.Lock()
	i.loadBalancers[arn] = lb
	i.mu.Unlock()

	klog.V(2).Infof("Created load balancer: %s with DNS: %s and Traefik deployment", arn, lb.DNSName)
	return lb, nil
}

// DeleteLoadBalancer deletes a virtual load balancer
func (i *k8sIntegration) DeleteLoadBalancer(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting virtual load balancer: %s", arn)

	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.loadBalancers[arn]; !exists {
		return fmt.Errorf("load balancer not found: %s", arn)
	}

	delete(i.loadBalancers, arn)
	return nil
}

// CreateTargetGroup creates a virtual target group and Kubernetes resources
func (i *k8sIntegration) CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*TargetGroup, error) {
	klog.V(2).Infof("Creating target group with Kubernetes resources: %s", name)

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

	klog.V(2).Infof("Created target group: %s with Kubernetes resources", arn)
	return tg, nil
}

// DeleteTargetGroup deletes a virtual target group
func (i *k8sIntegration) DeleteTargetGroup(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting virtual target group: %s", arn)

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
func (i *k8sIntegration) RegisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	klog.V(2).Infof("Registering %d targets with virtual target group: %s", len(targets), targetGroupArn)

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
func (i *k8sIntegration) DeregisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	klog.V(2).Infof("Deregistering %d targets from virtual target group: %s", len(targets), targetGroupArn)

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
func (i *k8sIntegration) CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*Listener, error) {
	klog.V(2).Infof("Creating listener on port %d for load balancer: %s", port, loadBalancerArn)

	i.mu.RLock()
	lb, exists := i.loadBalancers[loadBalancerArn]
	if !exists {
		i.mu.RUnlock()
		return nil, fmt.Errorf("load balancer not found: %s", loadBalancerArn)
	}
	lbName := lb.Name

	tg, exists := i.targetGroups[targetGroupArn]
	if !exists && targetGroupArn != "" {
		i.mu.RUnlock()
		return nil, fmt.Errorf("target group not found: %s", targetGroupArn)
	}
	var targetGroupName string
	if tg != nil {
		targetGroupName = tg.Name
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

	// Store in memory with lock
	i.mu.Lock()
	i.listeners[arn] = listener
	i.mu.Unlock()

	klog.V(2).Infof("Created listener: %s with Traefik configuration", arn)
	return listener, nil
}

// DeleteListener deletes a virtual listener
func (i *k8sIntegration) DeleteListener(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting virtual listener: %s", arn)

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
	}
	
	delete(i.listeners, arn)
	i.mu.Unlock()

	// Delete IngressRoute CRD if we have the necessary info
	if lbName != "" && i.dynamicClient != nil {
		if err := i.deleteIngressRoute(ctx, lbName, listener.Port); err != nil {
			klog.V(2).Infof("Failed to delete IngressRoute for listener %s: %v", arn, err)
			// Don't fail the operation if IngressRoute deletion fails
		}
	}

	return nil
}

// GetLoadBalancer gets virtual load balancer details
func (i *k8sIntegration) GetLoadBalancer(ctx context.Context, arn string) (*LoadBalancer, error) {
	klog.V(2).Infof("Getting virtual load balancer: %s", arn)

	i.mu.RLock()
	defer i.mu.RUnlock()

	lb, exists := i.loadBalancers[arn]
	if !exists {
		return nil, fmt.Errorf("load balancer not found: %s", arn)
	}

	return lb, nil
}

// GetTargetHealth gets the health status of virtual targets
func (i *k8sIntegration) GetTargetHealth(ctx context.Context, targetGroupArn string) ([]TargetHealth, error) {
	klog.V(2).Infof("Getting target health for virtual target group: %s", targetGroupArn)

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
func (i *k8sIntegration) CheckTargetHealthWithK8s(ctx context.Context, targetIP string, targetPort int32, targetGroupArn string) (string, error) {
	klog.V(2).Infof("Checking target health with Kubernetes for %s:%d", targetIP, targetPort)
	
	if i.kubeClient == nil {
		klog.V(2).Infof("No kubeClient available, falling back to basic connectivity check")
		return i.performBasicConnectivityCheck(targetIP, targetPort)
	}
	
	// Find pod by IP address
	pod, err := i.findPodByIP(ctx, targetIP)
	if err != nil {
		klog.V(2).Infof("Failed to find pod with IP %s: %v", targetIP, err)
		// Fallback to basic connectivity check if pod not found
		return i.performBasicConnectivityCheck(targetIP, targetPort)
	}
	
	if pod == nil {
		klog.V(2).Infof("No pod found with IP %s, performing basic connectivity check", targetIP)
		return i.performBasicConnectivityCheck(targetIP, targetPort)
	}
	
	// Check pod readiness status
	return i.checkPodReadiness(pod, targetPort)
}

// findPodByIP finds a pod by its IP address across all namespaces
func (i *k8sIntegration) findPodByIP(ctx context.Context, targetIP string) (*corev1.Pod, error) {
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
func (i *k8sIntegration) checkPodReadiness(pod *corev1.Pod, targetPort int32) (string, error) {
	// Check pod phase first
	if pod.Status.Phase != corev1.PodRunning {
		klog.V(2).Infof("Pod %s/%s is not running (phase: %s)", pod.Namespace, pod.Name, pod.Status.Phase)
		return "unhealthy", nil
	}
	
	// Check pod readiness conditions
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			if condition.Status == corev1.ConditionTrue {
				klog.V(2).Infof("Pod %s/%s is ready", pod.Namespace, pod.Name)
				
				// Additionally check if the target port is exposed by the pod
				if i.isPodPortExposed(pod, targetPort) {
					return "healthy", nil
				} else {
					klog.V(2).Infof("Pod %s/%s does not expose target port %d", pod.Namespace, pod.Name, targetPort)
					return "unhealthy", nil
				}
			} else {
				klog.V(2).Infof("Pod %s/%s is not ready (reason: %s)", pod.Namespace, pod.Name, condition.Reason)
				return "unhealthy", nil
			}
		}
	}
	
	// If no readiness condition found, consider it unhealthy
	klog.V(2).Infof("Pod %s/%s has no readiness condition", pod.Namespace, pod.Name)
	return "unhealthy", nil
}

// isPodPortExposed checks if a pod exposes the given port
func (i *k8sIntegration) isPodPortExposed(pod *corev1.Pod, targetPort int32) bool {
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
func (i *k8sIntegration) performBasicConnectivityCheck(targetIP string, targetPort int32) (string, error) {
	timeout := 5 * time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, targetPort), timeout)
	if err != nil {
		klog.V(2).Infof("Basic connectivity check failed for %s:%d: %v", targetIP, targetPort, err)
		return "unhealthy", nil
	}
	conn.Close()
	klog.V(2).Infof("Basic connectivity check passed for %s:%d", targetIP, targetPort)
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
func (i *k8sIntegration) deployTraefikForLoadBalancer(ctx context.Context, lbName, lbArn string) error {
	if i.kubeClient == nil {
		// If no kubeClient is available, just log and continue
		klog.V(2).Infof("No kubeClient available, skipping Traefik deployment for load balancer: %s", lbName)
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

	klog.V(2).Infof("Successfully deployed Traefik resources for load balancer: %s", lbName)
	return nil
}

// createNamespaceIfNotExists creates the namespace if it doesn't exist
func (i *k8sIntegration) createNamespaceIfNotExists(ctx context.Context, namespace string) error {
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
func (i *k8sIntegration) createServiceAccount(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
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
func (i *k8sIntegration) createConfigMap(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
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
func (i *k8sIntegration) createDeployment(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
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
							Image: "traefik:v3.0",
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
func (i *k8sIntegration) createService(ctx context.Context, namespace, traefikName, lbName, lbArn string) error {
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
func (i *k8sIntegration) deployTargetGroupResources(ctx context.Context, tgName, tgArn string, port int32, protocol string) error {
	if i.kubeClient == nil {
		klog.V(2).Infof("No kubeClient available, skipping target group resources deployment for: %s", tgName)
		return nil
	}

	namespace := "kecs-system"
	serviceName := fmt.Sprintf("tg-%s", tgName)

	// Create a Service for the target group
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Annotations: map[string]string{
				"kecs.io/elbv2-target-group-name": tgName,
				"kecs.io/elbv2-target-group-arn":  tgArn,
				"kecs.io/elbv2-target-group-protocol": protocol,
			},
			Labels: map[string]string{
				"kecs.io/elbv2-target-group-name": tgName,
				"kecs.io/component":               "target-group",
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"kecs.io/elbv2-target-group-name": tgName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "main",
					Port:       port,
					TargetPort: intstr.FromInt(int(port)),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	// Create the service
	_, err := i.kubeClient.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Service for target group: %w", err)
	}

	klog.V(2).Infof("Created Service %s for target group %s", serviceName, tgName)
	return nil
}

// updateTraefikConfigForListener updates Traefik configuration for a new listener
func (i *k8sIntegration) updateTraefikConfigForListener(ctx context.Context, lbName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.kubeClient == nil {
		klog.V(2).Infof("No kubeClient available, skipping Traefik config update for listener: %s", listenerArn)
		return nil
	}

	namespace := "kecs-system"
	traefikName := fmt.Sprintf("traefik-elbv2-%s", lbName)
	
	// Update the ConfigMap with new listener configuration
	cm, err := i.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, fmt.Sprintf("%s-config", traefikName), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get ConfigMap: %w", err)
	}

	// Update the traefik.yml to include the new listener port
	traefikYaml := cm.Data["traefik.yml"]
	if !strings.Contains(traefikYaml, fmt.Sprintf(":%d", port)) {
		// Add new entrypoint for the listener
		newEntry := fmt.Sprintf(`
  listener%d:
    address: ":%d"`, port, port)
		
		// Insert after the existing entryPoints
		traefikYaml = strings.Replace(traefikYaml, "entryPoints:", "entryPoints:"+newEntry, 1)
		cm.Data["traefik.yml"] = traefikYaml
		
		// Update ConfigMap
		_, err = i.kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update ConfigMap: %w", err)
		}
	}

	// Create or update IngressRoute CRD for routing rules
	if targetGroupName != "" {
		if err := i.updateIngressRoute(ctx, lbName, listenerArn, port, protocol, targetGroupName); err != nil {
			return fmt.Errorf("failed to create/update IngressRoute: %w", err)
		}
	}

	klog.V(2).Infof("Updated Traefik configuration for listener on port %d", port)
	return nil
}

// createIngressRoute creates a Traefik IngressRoute CRD for routing to target groups
func (i *k8sIntegration) createIngressRoute(ctx context.Context, lbName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.dynamicClient == nil {
		klog.V(2).Infof("No dynamicClient available, skipping IngressRoute creation")
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
					"kecs.io/elbv2-listener-arn":      listenerArn,
					"kecs.io/elbv2-load-balancer":     lbName,
					"kecs.io/elbv2-target-group":      targetGroupName,
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

	klog.V(2).Infof("Created IngressRoute %s for listener on port %d routing to target group %s", ingressRouteName, port, targetGroupName)
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

// deleteIngressRoute deletes a Traefik IngressRoute CRD
func (i *k8sIntegration) deleteIngressRoute(ctx context.Context, lbName string, port int32) error {
	if i.dynamicClient == nil {
		klog.V(2).Infof("No dynamicClient available, skipping IngressRoute deletion")
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

	klog.V(2).Infof("Deleted IngressRoute %s", ingressRouteName)
	return nil
}

// updateIngressRoute updates an existing Traefik IngressRoute CRD
func (i *k8sIntegration) updateIngressRoute(ctx context.Context, lbName, listenerArn string, port int32, protocol, targetGroupName string) error {
	if i.dynamicClient == nil {
		klog.V(2).Infof("No dynamicClient available, skipping IngressRoute update")
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

	klog.V(2).Infof("Updated IngressRoute %s for listener on port %d routing to target group %s", ingressRouteName, port, targetGroupName)
	return nil
}
