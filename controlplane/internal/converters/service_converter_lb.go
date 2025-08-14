package converters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"

	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// ServiceConverterWithLB extends ServiceConverter with load balancer support
type ServiceConverterWithLB struct {
	*ServiceConverter
	elbv2Integration elbv2.Integration
}

// NewServiceConverterWithLB creates a new ServiceConverter with ELBv2 integration
func NewServiceConverterWithLB(region, accountID string, elbv2Integration elbv2.Integration) *ServiceConverterWithLB {
	return &ServiceConverterWithLB{
		ServiceConverter: NewServiceConverter(region, accountID),
		elbv2Integration: elbv2Integration,
	}
}

// ProcessLoadBalancers processes load balancer configuration for a service
func (c *ServiceConverterWithLB) ProcessLoadBalancers(
	ctx context.Context,
	service *storage.Service,
	kubeService *corev1.Service,
	cluster *storage.Cluster,
) error {
	// Parse load balancers
	if service.LoadBalancers == "" || service.LoadBalancers == "null" || service.LoadBalancers == "[]" {
		return nil
	}

	var loadBalancers []map[string]interface{}
	if err := json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers); err != nil {
		return fmt.Errorf("failed to parse load balancers: %w", err)
	}

	if len(loadBalancers) == 0 {
		return nil
	}

	// Process each load balancer configuration
	for _, lbConfig := range loadBalancers {
		if err := c.processLoadBalancer(ctx, lbConfig, service, kubeService, cluster); err != nil {
			return fmt.Errorf("failed to process load balancer: %w", err)
		}
	}

	return nil
}

func (c *ServiceConverterWithLB) processLoadBalancer(
	ctx context.Context,
	lbConfig map[string]interface{},
	service *storage.Service,
	kubeService *corev1.Service,
	cluster *storage.Cluster,
) error {
	// Extract load balancer configuration
	var targetGroupArn, loadBalancerName, containerName string
	var containerPort int32

	if tg, ok := lbConfig["targetGroupArn"].(string); ok {
		targetGroupArn = tg
	}
	if lb, ok := lbConfig["loadBalancerName"].(string); ok {
		loadBalancerName = lb
	}
	if cn, ok := lbConfig["containerName"].(string); ok {
		containerName = cn
	}
	if cp, ok := lbConfig["containerPort"].(float64); ok {
		containerPort = int32(cp)
	}

	logging.Debug("Processing load balancer configuration",
		"targetGroup", targetGroupArn, "loadBalancer", loadBalancerName, "container", containerName, "port", containerPort)

	// If target group ARN is provided, register the service endpoints
	if targetGroupArn != "" {
		// Get pod IPs from the service endpoints
		targets, err := c.getServiceTargets(ctx, kubeService, containerPort)
		if err != nil {
			logging.Warn("Failed to get service targets", "error", err)
			// Don't fail the operation, targets can be registered later
			return nil
		}

		if len(targets) > 0 {
			// Register targets with the target group
			if err := c.elbv2Integration.RegisterTargets(ctx, targetGroupArn, targets); err != nil {
				logging.Error("Failed to register targets with target group", "targetGroupArn", targetGroupArn, "error", err)
				// Don't fail the operation, targets can be registered later
			} else {
				logging.Debug("Registered targets with target group", "targetCount", len(targets), "targetGroupArn", targetGroupArn)
			}
		}

		// Add annotations to the Kubernetes service for tracking
		if kubeService != nil {
			if kubeService.Annotations == nil {
				kubeService.Annotations = make(map[string]string)
			}
			kubeService.Annotations["kecs.dev/target-group-arn"] = targetGroupArn
			if loadBalancerName != "" {
				kubeService.Annotations["kecs.dev/load-balancer-name"] = loadBalancerName
			}
		}
	}

	// TODO: If only load balancer name is provided, look up or create the load balancer

	return nil
}

// getServiceTargets gets the targets (pod IPs) for a service
func (c *ServiceConverterWithLB) getServiceTargets(
	ctx context.Context,
	kubeService *corev1.Service,
	port int32,
) ([]elbv2.Target, error) {
	if kubeService == nil {
		return nil, nil
	}

	// In a real implementation, we would query the Kubernetes API for endpoints
	// For now, return empty list - targets will be registered when pods are created
	return []elbv2.Target{}, nil
}

// CreateTargetGroupForService creates a target group for the service
func (c *ServiceConverterWithLB) CreateTargetGroupForService(
	ctx context.Context,
	service *storage.Service,
	cluster *storage.Cluster,
	port int32,
	protocol string,
) (*elbv2.TargetGroup, error) {
	// Generate target group name
	tgName := fmt.Sprintf("%s-%s", service.ServiceName, strings.ToLower(protocol))
	if len(tgName) > 32 {
		tgName = tgName[:32]
	}

	// Get VPC ID from cluster (simplified - in production, this would come from network configuration)
	vpcId := "vpc-default" // TODO: Get actual VPC ID from network configuration

	// Create target group
	tg, err := c.elbv2Integration.CreateTargetGroup(ctx, tgName, port, protocol, vpcId)
	if err != nil {
		return nil, fmt.Errorf("failed to create target group: %w", err)
	}

	logging.Debug("Created target group for service", "targetGroupArn", tg.Arn, "serviceName", service.ServiceName)
	return tg, nil
}

// CreateLoadBalancerForService creates a load balancer for the service
func (c *ServiceConverterWithLB) CreateLoadBalancerForService(
	ctx context.Context,
	service *storage.Service,
	cluster *storage.Cluster,
	subnets []string,
	securityGroups []string,
) (*elbv2.LoadBalancer, error) {
	// Generate load balancer name
	lbName := fmt.Sprintf("%s-lb", service.ServiceName)
	if len(lbName) > 32 {
		lbName = lbName[:32]
	}

	// Create load balancer
	lb, err := c.elbv2Integration.CreateLoadBalancer(ctx, lbName, subnets, securityGroups)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	logging.Debug("Created load balancer for service", "loadBalancerArn", lb.Arn, "serviceName", service.ServiceName)
	return lb, nil
}

// UpdateServiceWithLoadBalancer updates the service storage with load balancer information
func (c *ServiceConverterWithLB) UpdateServiceWithLoadBalancer(
	ctx context.Context,
	service *storage.Service,
	targetGroupArn string,
	loadBalancerName string,
) error {
	// Parse existing load balancers
	var loadBalancers []map[string]interface{}
	if service.LoadBalancers != "" && service.LoadBalancers != "null" {
		if err := json.Unmarshal([]byte(service.LoadBalancers), &loadBalancers); err != nil {
			logging.Warn("Failed to parse existing load balancers", "error", err)
			loadBalancers = []map[string]interface{}{}
		}
	}

	// Check if this target group is already configured
	found := false
	for _, lb := range loadBalancers {
		if tg, ok := lb["targetGroupArn"].(string); ok && tg == targetGroupArn {
			found = true
			break
		}
	}

	if !found {
		// Add new load balancer configuration
		loadBalancers = append(loadBalancers, map[string]interface{}{
			"targetGroupArn":   targetGroupArn,
			"loadBalancerName": loadBalancerName,
		})

		// Update service
		lbJSON, err := json.Marshal(loadBalancers)
		if err != nil {
			return fmt.Errorf("failed to marshal load balancers: %w", err)
		}
		service.LoadBalancers = string(lbJSON)
	}

	return nil
}
