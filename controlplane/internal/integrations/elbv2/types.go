package elbv2

import (
	"context"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// Integration defines the interface for ELBv2 integration
type Integration interface {
	// CreateLoadBalancer creates a new Application Load Balancer
	CreateLoadBalancer(ctx context.Context, name string, subnets []string, securityGroups []string) (*LoadBalancer, error)
	
	// DeleteLoadBalancer deletes a load balancer
	DeleteLoadBalancer(ctx context.Context, arn string) error
	
	// CreateTargetGroup creates a new target group
	CreateTargetGroup(ctx context.Context, name string, port int32, protocol string, vpcId string) (*TargetGroup, error)
	
	// DeleteTargetGroup deletes a target group
	DeleteTargetGroup(ctx context.Context, arn string) error
	
	// RegisterTargets registers targets with a target group
	RegisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error
	
	// DeregisterTargets deregisters targets from a target group
	DeregisterTargets(ctx context.Context, targetGroupArn string, targets []Target) error
	
	// CreateListener creates a listener for a load balancer
	CreateListener(ctx context.Context, loadBalancerArn string, port int32, protocol string, targetGroupArn string) (*Listener, error)
	
	// DeleteListener deletes a listener
	DeleteListener(ctx context.Context, arn string) error
	
	// GetLoadBalancer gets load balancer details
	GetLoadBalancer(ctx context.Context, arn string) (*LoadBalancer, error)
	
	// GetTargetHealth gets the health status of targets
	GetTargetHealth(ctx context.Context, targetGroupArn string) ([]TargetHealth, error)
}

// LoadBalancer represents an Application Load Balancer
type LoadBalancer struct {
	Arn            string
	Name           string
	DNSName        string
	State          string
	Type           string
	Scheme         string
	VpcId          string
	AvailabilityZones []AvailabilityZone
	SecurityGroups []string
	CreatedTime    string
}

// AvailabilityZone represents an availability zone for a load balancer
type AvailabilityZone struct {
	ZoneName     string
	SubnetId     string
	LoadBalancerAddresses []LoadBalancerAddress
}

// LoadBalancerAddress represents an IP address for a load balancer
type LoadBalancerAddress struct {
	IpAddress    string
	AllocationId string
}

// TargetGroup represents a target group
type TargetGroup struct {
	Arn              string
	Name             string
	Port             int32
	Protocol         string
	VpcId            string
	TargetType       string
	HealthCheckPath  string
	HealthCheckPort  string
	HealthCheckProtocol string
	UnhealthyThresholdCount int32
	HealthyThresholdCount   int32
}

// Target represents a target in a target group
type Target struct {
	Id               string
	Port             int32
	AvailabilityZone string
}

// TargetHealth represents the health status of a target
type TargetHealth struct {
	Target      Target
	HealthState string
	Reason      string
	Description string
}

// Listener represents a load balancer listener
type Listener struct {
	Arn             string
	LoadBalancerArn string
	Port            int32
	Protocol        string
	DefaultActions  []Action
}

// Action represents a listener action
type Action struct {
	Type           string
	TargetGroupArn string
	Order          int32
}

// Config holds the configuration for ELBv2 integration
type Config struct {
	Enabled          bool
	LocalStackManager localstack.Manager
	Region           string
	AccountID        string
}