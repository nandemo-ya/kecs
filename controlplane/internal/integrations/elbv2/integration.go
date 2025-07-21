package elbv2

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// NewIntegration creates a new ELBv2 integration
func NewIntegration(cfg Config) (Integration, error) {
	if !cfg.Enabled {
		logging.Info("ELBv2 integration is disabled")
		return &noOpIntegration{}, nil
	}

	logging.Info("Initializing ELBv2 integration with Kubernetes-based implementation")

	// Use Kubernetes-based implementation instead of LocalStack
	// This avoids the need for LocalStack Pro
	return NewK8sIntegration(cfg.Region, cfg.AccountID), nil
}

// NewIntegrationWithClient creates a new ELBv2 integration (for testing)
func NewIntegrationWithClient(
	localstackManager localstack.Manager,
	cfg Config,
	elbClient interface{}, // Keeping for API compatibility
) Integration {
	// Ignore the ELB client and use K8s implementation
	return NewK8sIntegration(cfg.Region, cfg.AccountID)
}

// noOpIntegration is a no-op implementation when ELBv2 integration is disabled
type noOpIntegration struct{}

func (n *noOpIntegration) CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*LoadBalancer, error) {
	return nil, fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) DeleteLoadBalancer(ctx context.Context, arn string) error {
	return fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*TargetGroup, error) {
	return nil, fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) DeleteTargetGroup(ctx context.Context, arn string) error {
	return fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) RegisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	return fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) DeregisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	return fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*Listener, error) {
	return nil, fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) DeleteListener(ctx context.Context, arn string) error {
	return fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) GetLoadBalancer(ctx context.Context, arn string) (*LoadBalancer, error) {
	return nil, fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) GetTargetHealth(ctx context.Context, targetGroupArn string) ([]TargetHealth, error) {
	return nil, fmt.Errorf("ELBv2 integration is disabled")
}

func (n *noOpIntegration) CheckTargetHealthWithK8s(ctx context.Context, targetIP string, targetPort int32, targetGroupArn string) (string, error) {
	return "", fmt.Errorf("ELBv2 integration is disabled")
}
