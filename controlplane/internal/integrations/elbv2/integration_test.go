package elbv2_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kecsELBv2 "github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// Mock ELBv2 client for testing
type mockELBv2Client struct {
	createLoadBalancerFunc    func(ctx context.Context, params *elasticloadbalancingv2.CreateLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateLoadBalancerOutput, error)
	deleteLoadBalancerFunc    func(ctx context.Context, params *elasticloadbalancingv2.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteLoadBalancerOutput, error)
	createTargetGroupFunc     func(ctx context.Context, params *elasticloadbalancingv2.CreateTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateTargetGroupOutput, error)
	deleteTargetGroupFunc     func(ctx context.Context, params *elasticloadbalancingv2.DeleteTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteTargetGroupOutput, error)
	registerTargetsFunc       func(ctx context.Context, params *elasticloadbalancingv2.RegisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.RegisterTargetsOutput, error)
	deregisterTargetsFunc     func(ctx context.Context, params *elasticloadbalancingv2.DeregisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeregisterTargetsOutput, error)
	createListenerFunc        func(ctx context.Context, params *elasticloadbalancingv2.CreateListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateListenerOutput, error)
	deleteListenerFunc        func(ctx context.Context, params *elasticloadbalancingv2.DeleteListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteListenerOutput, error)
	describeLoadBalancersFunc func(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error)
	describeTargetHealthFunc  func(ctx context.Context, params *elasticloadbalancingv2.DescribeTargetHealthInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetHealthOutput, error)
}

func (m *mockELBv2Client) CreateLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.CreateLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateLoadBalancerOutput, error) {
	if m.createLoadBalancerFunc != nil {
		return m.createLoadBalancerFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.CreateLoadBalancerOutput{}, nil
}

func (m *mockELBv2Client) DeleteLoadBalancer(ctx context.Context, params *elasticloadbalancingv2.DeleteLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteLoadBalancerOutput, error) {
	if m.deleteLoadBalancerFunc != nil {
		return m.deleteLoadBalancerFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.DeleteLoadBalancerOutput{}, nil
}

func (m *mockELBv2Client) CreateTargetGroup(ctx context.Context, params *elasticloadbalancingv2.CreateTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateTargetGroupOutput, error) {
	if m.createTargetGroupFunc != nil {
		return m.createTargetGroupFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.CreateTargetGroupOutput{}, nil
}

func (m *mockELBv2Client) DeleteTargetGroup(ctx context.Context, params *elasticloadbalancingv2.DeleteTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteTargetGroupOutput, error) {
	if m.deleteTargetGroupFunc != nil {
		return m.deleteTargetGroupFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.DeleteTargetGroupOutput{}, nil
}

func (m *mockELBv2Client) RegisterTargets(ctx context.Context, params *elasticloadbalancingv2.RegisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.RegisterTargetsOutput, error) {
	if m.registerTargetsFunc != nil {
		return m.registerTargetsFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.RegisterTargetsOutput{}, nil
}

func (m *mockELBv2Client) DeregisterTargets(ctx context.Context, params *elasticloadbalancingv2.DeregisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeregisterTargetsOutput, error) {
	if m.deregisterTargetsFunc != nil {
		return m.deregisterTargetsFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.DeregisterTargetsOutput{}, nil
}

func (m *mockELBv2Client) CreateListener(ctx context.Context, params *elasticloadbalancingv2.CreateListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateListenerOutput, error) {
	if m.createListenerFunc != nil {
		return m.createListenerFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.CreateListenerOutput{}, nil
}

func (m *mockELBv2Client) DeleteListener(ctx context.Context, params *elasticloadbalancingv2.DeleteListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DeleteListenerOutput, error) {
	if m.deleteListenerFunc != nil {
		return m.deleteListenerFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.DeleteListenerOutput{}, nil
}

func (m *mockELBv2Client) DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	if m.describeLoadBalancersFunc != nil {
		return m.describeLoadBalancersFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.DescribeLoadBalancersOutput{}, nil
}

func (m *mockELBv2Client) DescribeTargetHealth(ctx context.Context, params *elasticloadbalancingv2.DescribeTargetHealthInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetHealthOutput, error) {
	if m.describeTargetHealthFunc != nil {
		return m.describeTargetHealthFunc(ctx, params, optFns...)
	}
	return &elasticloadbalancingv2.DescribeTargetHealthOutput{}, nil
}

// Mock LocalStack manager
type mockLocalStackManager struct{}

func (m *mockLocalStackManager) Start(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) Stop(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) Restart(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) GetStatus() (*localstack.Status, error) {
	return &localstack.Status{
		Running: true,
		Healthy: true,
		EnabledServices: []string{"elbv2"},
	}, nil
}

func (m *mockLocalStackManager) UpdateServices(services []string) error {
	return nil
}

func (m *mockLocalStackManager) GetEnabledServices() ([]string, error) {
	return []string{"elbv2"}, nil
}

func (m *mockLocalStackManager) GetEndpoint() (string, error) {
	return "http://localhost:4566", nil
}

func (m *mockLocalStackManager) GetServiceEndpoint(service string) (string, error) {
	return "http://localhost:4566", nil
}

func (m *mockLocalStackManager) IsHealthy() bool {
	return true
}

func (m *mockLocalStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return nil
}

var _ = Describe("ELBv2 Integration", func() {
	var (
		integration       kecsELBv2.Integration
		localstackManager localstack.Manager
		elbClient         *mockELBv2Client
		config            kecsELBv2.Config
		ctx              context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Create mock LocalStack manager
		localstackManager = &mockLocalStackManager{}
		
		// Create mock ELBv2 client
		elbClient = &mockELBv2Client{}
		
		// Create config
		config = kecsELBv2.Config{
			Enabled:           true,
			LocalStackManager: localstackManager,
			Region:            "us-east-1",
			AccountID:         "123456789012",
		}
		
		// Create integration with mocked clients
		integration = kecsELBv2.NewIntegrationWithClient(
			localstackManager,
			config,
			elbClient,
		)
	})

	Describe("CreateLoadBalancer", func() {
		It("should create a load balancer successfully", func() {
			// Setup mock response
			elbClient.createLoadBalancerFunc = func(ctx context.Context, params *elasticloadbalancingv2.CreateLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateLoadBalancerOutput, error) {
				Expect(params.Name).To(Equal(aws.String("test-lb")))
				Expect(params.Subnets).To(ConsistOf([]string{"subnet-1", "subnet-2"}))
				Expect(params.SecurityGroups).To(ConsistOf([]string{"sg-1"}))
				
				return &elasticloadbalancingv2.CreateLoadBalancerOutput{
					LoadBalancers: []elbv2types.LoadBalancer{
						{
							LoadBalancerArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/50dc6c495c0c9188"),
							LoadBalancerName: aws.String("test-lb"),
							DNSName:          aws.String("test-lb-1234567890.us-east-1.elb.amazonaws.com"),
							State: &elbv2types.LoadBalancerState{
								Code: elbv2types.LoadBalancerStateEnumActive,
							},
							Type:   elbv2types.LoadBalancerTypeEnumApplication,
							Scheme: elbv2types.LoadBalancerSchemeEnumInternetFacing,
							VpcId:  aws.String("vpc-12345"),
							CreatedTime: aws.Time(time.Now()),
						},
					},
				}, nil
			}
			
			// Create load balancer
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb", []string{"subnet-1", "subnet-2"}, []string{"sg-1"})
			
			Expect(err).NotTo(HaveOccurred())
			Expect(lb).NotTo(BeNil())
			Expect(lb.Name).To(Equal("test-lb"))
			Expect(lb.DNSName).To(Equal("test-lb-1234567890.us-east-1.elb.amazonaws.com"))
			Expect(lb.State).To(Equal("active"))
			
			// Load balancer created successfully
		})
		
		It("should handle creation errors", func() {
			// Setup mock to return error
			elbClient.createLoadBalancerFunc = func(ctx context.Context, params *elasticloadbalancingv2.CreateLoadBalancerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateLoadBalancerOutput, error) {
				return nil, fmt.Errorf("load balancer limit exceeded")
			}
			
			// Create load balancer
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb", []string{"subnet-1"}, []string{"sg-1"})
			
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("load balancer limit exceeded"))
			Expect(lb).To(BeNil())
		})
	})
	
	Describe("CreateTargetGroup", func() {
		It("should create a target group successfully", func() {
			// Setup mock response
			elbClient.createTargetGroupFunc = func(ctx context.Context, params *elasticloadbalancingv2.CreateTargetGroupInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateTargetGroupOutput, error) {
				Expect(params.Name).To(Equal(aws.String("test-tg")))
				Expect(params.Port).To(Equal(aws.Int32(80)))
				Expect(params.Protocol).To(Equal(elbv2types.ProtocolEnumHttp))
				Expect(params.VpcId).To(Equal(aws.String("vpc-12345")))
				
				return &elasticloadbalancingv2.CreateTargetGroupOutput{
					TargetGroups: []elbv2types.TargetGroup{
						{
							TargetGroupArn:  aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/50dc6c495c0c9188"),
							TargetGroupName: aws.String("test-tg"),
							Port:            aws.Int32(80),
							Protocol:        elbv2types.ProtocolEnumHttp,
							VpcId:           aws.String("vpc-12345"),
							TargetType:      elbv2types.TargetTypeEnumIp,
							HealthCheckPath: aws.String("/"),
							HealthCheckPort: aws.String("traffic-port"),
							HealthCheckProtocol: elbv2types.ProtocolEnumHttp,
							UnhealthyThresholdCount: aws.Int32(3),
							HealthyThresholdCount:   aws.Int32(2),
						},
					},
				}, nil
			}
			
			// Create target group
			tg, err := integration.CreateTargetGroup(ctx, "test-tg", 80, "HTTP", "vpc-12345")
			
			Expect(err).NotTo(HaveOccurred())
			Expect(tg).NotTo(BeNil())
			Expect(tg.Name).To(Equal("test-tg"))
			Expect(tg.Port).To(Equal(int32(80)))
			Expect(tg.Protocol).To(Equal("HTTP"))
			Expect(tg.VpcId).To(Equal("vpc-12345"))
			Expect(tg.TargetType).To(Equal("ip"))
		})
	})
	
	Describe("RegisterTargets", func() {
		It("should register targets successfully", func() {
			// Setup mock response
			targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/50dc6c495c0c9188"
			
			elbClient.registerTargetsFunc = func(ctx context.Context, params *elasticloadbalancingv2.RegisterTargetsInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.RegisterTargetsOutput, error) {
				Expect(params.TargetGroupArn).To(Equal(aws.String(targetGroupArn)))
				Expect(params.Targets).To(HaveLen(2))
				
				return &elasticloadbalancingv2.RegisterTargetsOutput{}, nil
			}
			
			// Register targets
			targets := []kecsELBv2.Target{
				{Id: "10.0.1.10", Port: 80},
				{Id: "10.0.1.11", Port: 80},
			}
			
			err := integration.RegisterTargets(ctx, targetGroupArn, targets)
			Expect(err).NotTo(HaveOccurred())
		})
	})
	
	Describe("CreateListener", func() {
		It("should create a listener successfully", func() {
			// Setup mock response
			loadBalancerArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/50dc6c495c0c9188"
			targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/50dc6c495c0c9188"
			
			elbClient.createListenerFunc = func(ctx context.Context, params *elasticloadbalancingv2.CreateListenerInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.CreateListenerOutput, error) {
				Expect(params.LoadBalancerArn).To(Equal(aws.String(loadBalancerArn)))
				Expect(params.Port).To(Equal(aws.Int32(80)))
				Expect(params.Protocol).To(Equal(elbv2types.ProtocolEnumHttp))
				
				return &elasticloadbalancingv2.CreateListenerOutput{
					Listeners: []elbv2types.Listener{
						{
							ListenerArn:     aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/test-lb/50dc6c495c0c9188/50dc6c495c0c9188"),
							LoadBalancerArn: aws.String(loadBalancerArn),
							Port:            aws.Int32(80),
							Protocol:        elbv2types.ProtocolEnumHttp,
							DefaultActions: []elbv2types.Action{
								{
									Type:           elbv2types.ActionTypeEnumForward,
									TargetGroupArn: aws.String(targetGroupArn),
									Order:          aws.Int32(1),
								},
							},
						},
					},
				}, nil
			}
			
			// Create listener
			listener, err := integration.CreateListener(ctx, loadBalancerArn, 80, "HTTP", targetGroupArn)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(listener).NotTo(BeNil())
			Expect(listener.Port).To(Equal(int32(80)))
			Expect(listener.Protocol).To(Equal("HTTP"))
			Expect(listener.DefaultActions).To(HaveLen(1))
			Expect(listener.DefaultActions[0].TargetGroupArn).To(Equal(targetGroupArn))
		})
	})
	
	Describe("GetTargetHealth", func() {
		It("should get target health successfully", func() {
			// Setup mock response
			targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/50dc6c495c0c9188"
			
			elbClient.describeTargetHealthFunc = func(ctx context.Context, params *elasticloadbalancingv2.DescribeTargetHealthInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeTargetHealthOutput, error) {
				Expect(params.TargetGroupArn).To(Equal(aws.String(targetGroupArn)))
				
				return &elasticloadbalancingv2.DescribeTargetHealthOutput{
					TargetHealthDescriptions: []elbv2types.TargetHealthDescription{
						{
							Target: &elbv2types.TargetDescription{
								Id:   aws.String("10.0.1.10"),
								Port: aws.Int32(80),
							},
							TargetHealth: &elbv2types.TargetHealth{
								State:       elbv2types.TargetHealthStateEnumHealthy,
									Description: aws.String("Target registration is in progress"),
							},
						},
						{
							Target: &elbv2types.TargetDescription{
								Id:   aws.String("10.0.1.11"),
								Port: aws.Int32(80),
							},
							TargetHealth: &elbv2types.TargetHealth{
								State:       elbv2types.TargetHealthStateEnumUnhealthy,
									Description: aws.String("Request timed out"),
							},
						},
					},
				}, nil
			}
			
			// Get target health
			healthStatuses, err := integration.GetTargetHealth(ctx, targetGroupArn)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(healthStatuses).To(HaveLen(2))
			
			Expect(healthStatuses[0].Target.Id).To(Equal("10.0.1.10"))
			Expect(healthStatuses[0].HealthState).To(Equal("healthy"))
			
			Expect(healthStatuses[1].Target.Id).To(Equal("10.0.1.11"))
			Expect(healthStatuses[1].HealthState).To(Equal("unhealthy"))
			Expect(healthStatuses[1].Description).To(Equal("Request timed out"))
		})
	})
	
	Describe("Disabled Integration", func() {
		BeforeEach(func() {
			// Create integration with disabled config
			disabledConfig := kecsELBv2.Config{
				Enabled: false,
			}
			
			var err error
			integration, err = kecsELBv2.NewIntegration(disabledConfig)
			Expect(err).NotTo(HaveOccurred())
		})
		
		It("should return error when creating load balancer", func() {
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb", []string{"subnet-1"}, []string{"sg-1"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ELBv2 integration is disabled"))
			Expect(lb).To(BeNil())
		})
		
		It("should return error when creating target group", func() {
			tg, err := integration.CreateTargetGroup(ctx, "test-tg", 80, "HTTP", "vpc-12345")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ELBv2 integration is disabled"))
			Expect(tg).To(BeNil())
		})
	})
})