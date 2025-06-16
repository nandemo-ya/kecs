package elbv2

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

const (
	// Kubernetes Service annotations for load balancer configuration
	annotationLoadBalancerName = "kecs.dev/load-balancer-name"
	annotationTargetGroupArn   = "kecs.dev/target-group-arn"
	annotationListenerArn      = "kecs.dev/listener-arn"
)

// integration implements the Integration interface
type integration struct {
	localstackManager localstack.Manager
	elbClient         ELBv2Client
	region            string
	accountID         string
}

// ELBv2Client interface for ELBv2 operations (for testing)
type ELBv2Client interface {
	CreateLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.CreateLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateLoadBalancerOutput, error)
	DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteLoadBalancerOutput, error)
	CreateTargetGroup(ctx context.Context, params *elasticloadbalancingv2.CreateTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateTargetGroupOutput, error)
	DeleteTargetGroup(ctx context.Context, params *elasticloadbalancingv2.DeleteTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteTargetGroupOutput, error)
	RegisterTargets(ctx context.Context, params *elasticloadbalancingv2.RegisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.RegisterTargetsOutput, error)
	DeregisterTargets(ctx context.Context, params *elasticloadbalancingv2.DeregisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeregisterTargetsOutput, error)
	CreateListener(ctx context.Context, params *elasticloadbalancingv2.CreateListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateListenerOutput, error)
	DeleteListener(ctx context.Context, params *elasticloadbalancingv2.DeleteListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteListenerOutput, error)
	DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	DescribeTargetHealth(ctx context.Context, params *elasticloadbalancingv2.DescribeTargetHealthInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetHealthOutput, error)
}

// NewIntegration creates a new ELBv2 integration
func NewIntegration(cfg Config) (Integration, error) {
	if !cfg.Enabled {
		klog.Info("ELBv2 integration is disabled")
		return &noOpIntegration{}, nil
	}

	klog.Info("Initializing ELBv2 integration with LocalStack")

	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Get LocalStack endpoint
	endpoint, err := cfg.LocalStackManager.GetServiceEndpoint("elbv2")
	if err != nil {
		return nil, fmt.Errorf("failed to get LocalStack ELBv2 endpoint: %w", err)
	}

	// Override endpoint for LocalStack
	awsCfg.BaseEndpoint = aws.String(endpoint)

	// Create ELBv2 client
	elbClient := elasticloadbalancingv2.NewFromConfig(awsCfg)

	return &integration{
		localstackManager: cfg.LocalStackManager,
		elbClient:         elbClient,
		region:            cfg.Region,
		accountID:         cfg.AccountID,
	}, nil
}

// NewIntegrationWithClient creates a new ELBv2 integration with a custom client (for testing)
func NewIntegrationWithClient(
	localstackManager localstack.Manager,
	cfg Config,
	elbClient ELBv2Client,
) Integration {
	return &integration{
		localstackManager: localstackManager,
		elbClient:         elbClient,
		region:            cfg.Region,
		accountID:         cfg.AccountID,
	}
}

// CreateLoadBalancer creates a new Application Load Balancer
func (i *integration) CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*LoadBalancer, error) {
	klog.V(2).Infof("Creating load balancer: %s", name)

	input := &elasticloadbalancingv2.CreateLoadBalancerInput{
		Name:           aws.String(name),
		Subnets:        subnets,
		SecurityGroups: securityGroups,
		Type:           elbv2types.LoadBalancerTypeEnumApplication,
		Scheme:         elbv2types.LoadBalancerSchemeEnumInternetFacing,
		Tags: []elbv2types.Tag{
			{
				Key:   aws.String("kecs.dev/managed"),
				Value: aws.String("true"),
			},
		},
	}

	output, err := i.elbClient.CreateLoadBalancer(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create load balancer: %w", err)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, fmt.Errorf("no load balancer created")
	}

	lb := output.LoadBalancers[0]
	return i.convertLoadBalancer(&lb), nil
}

// DeleteLoadBalancer deletes a load balancer
func (i *integration) DeleteLoadBalancer(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting load balancer: %s", arn)

	input := &elasticloadbalancingv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(arn),
	}

	_, err := i.elbClient.DeleteLoadBalancer(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete load balancer: %w", err)
	}

	// Delete associated Kubernetes Service
	// Extract name from ARN (simplified - in production, store mapping)
	// TODO: Implement proper ARN to name mapping
	
	return nil
}

// CreateTargetGroup creates a new target group
func (i *integration) CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*TargetGroup, error) {
	klog.V(2).Infof("Creating target group: %s", name)

	input := &elasticloadbalancingv2.CreateTargetGroupInput{
		Name:     aws.String(name),
		Port:     aws.Int32(port),
		Protocol: elbv2types.ProtocolEnum(protocol),
		VpcId:    aws.String(vpcId),
		TargetType: elbv2types.TargetTypeEnumIp, // Use IP targets for container IPs
		HealthCheckEnabled: aws.Bool(true),
		HealthCheckPath: aws.String("/"),
		HealthCheckProtocol: elbv2types.ProtocolEnum(protocol),
		HealthCheckIntervalSeconds: aws.Int32(30),
		HealthCheckTimeoutSeconds: aws.Int32(5),
		HealthyThresholdCount: aws.Int32(2),
		UnhealthyThresholdCount: aws.Int32(3),
		Tags: []elbv2types.Tag{
			{
				Key:   aws.String("kecs.dev/managed"),
				Value: aws.String("true"),
			},
		},
	}

	output, err := i.elbClient.CreateTargetGroup(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create target group: %w", err)
	}

	if len(output.TargetGroups) == 0 {
		return nil, fmt.Errorf("no target group created")
	}

	tg := output.TargetGroups[0]
	return i.convertTargetGroup(&tg), nil
}

// DeleteTargetGroup deletes a target group
func (i *integration) DeleteTargetGroup(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting target group: %s", arn)

	input := &elasticloadbalancingv2.DeleteTargetGroupInput{
		TargetGroupArn: aws.String(arn),
	}

	_, err := i.elbClient.DeleteTargetGroup(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete target group: %w", err)
	}

	return nil
}

// RegisterTargets registers targets with a target group
func (i *integration) RegisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	klog.V(2).Infof("Registering %d targets with target group: %s", len(targets), targetGroupArn)

	elbTargets := make([]elbv2types.TargetDescription, len(targets))
	for idx, target := range targets {
		elbTargets[idx] = elbv2types.TargetDescription{
			Id:   aws.String(target.Id),
			Port: aws.Int32(target.Port),
		}
		if target.AvailabilityZone != "" {
			elbTargets[idx].AvailabilityZone = aws.String(target.AvailabilityZone)
		}
	}

	input := &elasticloadbalancingv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(targetGroupArn),
		Targets:        elbTargets,
	}

	_, err := i.elbClient.RegisterTargets(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to register targets: %w", err)
	}

	return nil
}

// DeregisterTargets deregisters targets from a target group
func (i *integration) DeregisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error {
	klog.V(2).Infof("Deregistering %d targets from target group: %s", len(targets), targetGroupArn)

	elbTargets := make([]elbv2types.TargetDescription, len(targets))
	for idx, target := range targets {
		elbTargets[idx] = elbv2types.TargetDescription{
			Id:   aws.String(target.Id),
			Port: aws.Int32(target.Port),
		}
	}

	input := &elasticloadbalancingv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(targetGroupArn),
		Targets:        elbTargets,
	}

	_, err := i.elbClient.DeregisterTargets(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to deregister targets: %w", err)
	}

	return nil
}

// CreateListener creates a listener for a load balancer
func (i *integration) CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*Listener, error) {
	klog.V(2).Infof("Creating listener on port %d for load balancer: %s", port, loadBalancerArn)

	input := &elasticloadbalancingv2.CreateListenerInput{
		LoadBalancerArn: aws.String(loadBalancerArn),
		Port:            aws.Int32(port),
		Protocol:        elbv2types.ProtocolEnum(protocol),
		DefaultActions: []elbv2types.Action{
			{
				Type:           elbv2types.ActionTypeEnumForward,
				TargetGroupArn: aws.String(targetGroupArn),
				Order:          aws.Int32(1),
			},
		},
		Tags: []elbv2types.Tag{
			{
				Key:   aws.String("kecs.dev/managed"),
				Value: aws.String("true"),
			},
		},
	}

	output, err := i.elbClient.CreateListener(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	if len(output.Listeners) == 0 {
		return nil, fmt.Errorf("no listener created")
	}

	listener := output.Listeners[0]
	return i.convertListener(&listener), nil
}

// DeleteListener deletes a listener
func (i *integration) DeleteListener(ctx context.Context, arn string) error {
	klog.V(2).Infof("Deleting listener: %s", arn)

	input := &elasticloadbalancingv2.DeleteListenerInput{
		ListenerArn: aws.String(arn),
	}

	_, err := i.elbClient.DeleteListener(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete listener: %w", err)
	}

	return nil
}

// GetLoadBalancer gets load balancer details
func (i *integration) GetLoadBalancer(ctx context.Context, arn string) (*LoadBalancer, error) {
	klog.V(2).Infof("Getting load balancer: %s", arn)

	input := &elasticloadbalancingv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{arn},
	}

	output, err := i.elbClient.DescribeLoadBalancers(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe load balancer: %w", err)
	}

	if len(output.LoadBalancers) == 0 {
		return nil, fmt.Errorf("load balancer not found")
	}

	return i.convertLoadBalancer(&output.LoadBalancers[0]), nil
}

// GetTargetHealth gets the health status of targets
func (i *integration) GetTargetHealth(ctx context.Context, targetGroupArn string) ([]TargetHealth, error) {
	klog.V(2).Infof("Getting target health for target group: %s", targetGroupArn)

	input := &elasticloadbalancingv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(targetGroupArn),
	}

	output, err := i.elbClient.DescribeTargetHealth(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe target health: %w", err)
	}

	results := make([]TargetHealth, len(output.TargetHealthDescriptions))
	for idx, thd := range output.TargetHealthDescriptions {
		results[idx] = TargetHealth{
			Target: Target{
				Id:   aws.ToString(thd.Target.Id),
				Port: aws.ToInt32(thd.Target.Port),
			},
			HealthState: string(thd.TargetHealth.State),
			Reason:      string(thd.TargetHealth.Reason),
			Description: aws.ToString(thd.TargetHealth.Description),
		}
		if thd.Target.AvailabilityZone != nil {
			results[idx].Target.AvailabilityZone = aws.ToString(thd.Target.AvailabilityZone)
		}
	}

	return results, nil
}

// Helper functions

func (i *integration) convertLoadBalancer(lb *elbv2types.LoadBalancer) *LoadBalancer {
	result := &LoadBalancer{
		Arn:     aws.ToString(lb.LoadBalancerArn),
		Name:    aws.ToString(lb.LoadBalancerName),
		DNSName: aws.ToString(lb.DNSName),
		State:   string(lb.State.Code),
		Type:    string(lb.Type),
		Scheme:  string(lb.Scheme),
		VpcId:   aws.ToString(lb.VpcId),
	}

	if lb.CreatedTime != nil {
		result.CreatedTime = lb.CreatedTime.Format(time.RFC3339)
	}

	result.AvailabilityZones = make([]AvailabilityZone, len(lb.AvailabilityZones))
	for idx, az := range lb.AvailabilityZones {
		result.AvailabilityZones[idx] = AvailabilityZone{
			ZoneName: aws.ToString(az.ZoneName),
			SubnetId: aws.ToString(az.SubnetId),
		}
		
		result.AvailabilityZones[idx].LoadBalancerAddresses = make([]LoadBalancerAddress, len(az.LoadBalancerAddresses))
		for addrIdx, addr := range az.LoadBalancerAddresses {
			result.AvailabilityZones[idx].LoadBalancerAddresses[addrIdx] = LoadBalancerAddress{
				IpAddress:    aws.ToString(addr.IpAddress),
				AllocationId: aws.ToString(addr.AllocationId),
			}
		}
	}

	result.SecurityGroups = lb.SecurityGroups

	return result
}

func (i *integration) convertTargetGroup(tg *elbv2types.TargetGroup) *TargetGroup {
	return &TargetGroup{
		Arn:              aws.ToString(tg.TargetGroupArn),
		Name:             aws.ToString(tg.TargetGroupName),
		Port:             aws.ToInt32(tg.Port),
		Protocol:         string(tg.Protocol),
		VpcId:            aws.ToString(tg.VpcId),
		TargetType:       string(tg.TargetType),
		HealthCheckPath:  aws.ToString(tg.HealthCheckPath),
		HealthCheckPort:  aws.ToString(tg.HealthCheckPort),
		HealthCheckProtocol: string(tg.HealthCheckProtocol),
		UnhealthyThresholdCount: aws.ToInt32(tg.UnhealthyThresholdCount),
		HealthyThresholdCount:   aws.ToInt32(tg.HealthyThresholdCount),
	}
}

func (i *integration) convertListener(listener *elbv2types.Listener) *Listener {
	result := &Listener{
		Arn:             aws.ToString(listener.ListenerArn),
		LoadBalancerArn: aws.ToString(listener.LoadBalancerArn),
		Port:            aws.ToInt32(listener.Port),
		Protocol:        string(listener.Protocol),
	}

	result.DefaultActions = make([]Action, len(listener.DefaultActions))
	for idx, action := range listener.DefaultActions {
		result.DefaultActions[idx] = Action{
			Type:           string(action.Type),
			TargetGroupArn: aws.ToString(action.TargetGroupArn),
			Order:          aws.ToInt32(action.Order),
		}
	}

	return result
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