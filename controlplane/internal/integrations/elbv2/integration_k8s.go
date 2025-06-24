package elbv2

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// k8sIntegration implements the Integration interface using Kubernetes Services
// instead of actual ELBv2 API calls. This avoids the need for LocalStack Pro.
type k8sIntegration struct {
	region    string
	accountID string

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
		loadBalancers: make(map[string]*LoadBalancer),
		targetGroups:  make(map[string]*TargetGroup),
		listeners:     make(map[string]*Listener),
		targetHealth:  make(map[string]map[string]*TargetHealth),
	}
}

// CreateLoadBalancer creates a virtual load balancer (no actual AWS resources)
func (i *k8sIntegration) CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*LoadBalancer, error) {
	klog.V(2).Infof("Creating virtual load balancer: %s", name)

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

	klog.V(2).Infof("Created virtual load balancer: %s with DNS: %s", arn, lb.DNSName)
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

// CreateTargetGroup creates a virtual target group
func (i *k8sIntegration) CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*TargetGroup, error) {
	klog.V(2).Infof("Creating virtual target group: %s", name)

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

	// Store in memory with lock
	i.mu.Lock()
	i.targetGroups[arn] = tg
	i.targetHealth[arn] = make(map[string]*TargetHealth)
	i.mu.Unlock()

	klog.V(2).Infof("Created virtual target group: %s", arn)
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

// CreateListener creates a virtual listener
func (i *k8sIntegration) CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*Listener, error) {
	klog.V(2).Infof("Creating virtual listener on port %d for load balancer: %s", port, loadBalancerArn)

	i.mu.RLock()
	if _, exists := i.loadBalancers[loadBalancerArn]; !exists {
		i.mu.RUnlock()
		return nil, fmt.Errorf("load balancer not found: %s", loadBalancerArn)
	}

	if _, exists := i.targetGroups[targetGroupArn]; !exists {
		i.mu.RUnlock()
		return nil, fmt.Errorf("target group not found: %s", targetGroupArn)
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

	// Store in memory with lock
	i.mu.Lock()
	i.listeners[arn] = listener
	i.mu.Unlock()

	klog.V(2).Infof("Created virtual listener: %s", arn)
	return listener, nil
}

// DeleteListener deletes a virtual listener
func (i *k8sIntegration) DeleteListener(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting virtual listener: %s", arn)

	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.listeners[arn]; !exists {
		return fmt.Errorf("listener not found: %s", arn)
	}

	delete(i.listeners, arn)
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
