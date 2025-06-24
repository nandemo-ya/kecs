package elbv2_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kecsELBv2 "github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

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
		Running:         true,
		Healthy:         true,
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

func (m *mockLocalStackManager) IsRunning() bool {
	return true
}

func (m *mockLocalStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return nil
}

func (m *mockLocalStackManager) CheckServiceHealth(service string) error {
	return nil
}

func (m *mockLocalStackManager) GetConfig() *localstack.Config {
	return &localstack.Config{
		Enabled: true,
	}
}

func (m *mockLocalStackManager) GetContainer() *localstack.LocalStackContainer {
	return nil
}

func (m *mockLocalStackManager) EnableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) DisableService(service string) error {
	return nil
}

var _ = Describe("ELBv2 Integration", func() {
	var (
		integration       kecsELBv2.Integration
		localstackManager localstack.Manager
		config            kecsELBv2.Config
		ctx               context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create mock LocalStack manager
		localstackManager = &mockLocalStackManager{}

		// Create config
		config = kecsELBv2.Config{
			Enabled:           true,
			LocalStackManager: localstackManager,
			Region:            "us-east-1",
			AccountID:         "123456789012",
		}

		// Create integration using K8s implementation
		integration = kecsELBv2.NewIntegrationWithClient(
			localstackManager,
			config,
			nil, // No ELB client needed for K8s implementation
		)
	})

	Describe("CreateLoadBalancer", func() {
		It("should create a virtual load balancer successfully", func() {
			// Create load balancer
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb", []string{"subnet-1", "subnet-2"}, []string{"sg-1"})

			Expect(err).NotTo(HaveOccurred())
			Expect(lb).NotTo(BeNil())
			Expect(lb.Name).To(Equal("test-lb"))
			Expect(lb.State).To(Equal("active"))
			Expect(lb.Type).To(Equal("application"))
			Expect(lb.DNSName).To(ContainSubstring("test-lb"))
			Expect(lb.DNSName).To(ContainSubstring(".us-east-1.elb.amazonaws.com"))
		})

		It("should handle multiple load balancers", func() {
			// Create first load balancer
			lb1, err := integration.CreateLoadBalancer(ctx, "test-lb-1", []string{"subnet-1"}, []string{"sg-1"})
			Expect(err).NotTo(HaveOccurred())
			Expect(lb1).NotTo(BeNil())

			// Create second load balancer
			lb2, err := integration.CreateLoadBalancer(ctx, "test-lb-2", []string{"subnet-2"}, []string{"sg-2"})
			Expect(err).NotTo(HaveOccurred())
			Expect(lb2).NotTo(BeNil())

			// Verify they have different ARNs
			Expect(lb1.Arn).NotTo(Equal(lb2.Arn))
		})
	})

	Describe("CreateTargetGroup", func() {
		It("should create a virtual target group successfully", func() {
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
		var targetGroupArn string

		BeforeEach(func() {
			// Create a target group first
			tg, err := integration.CreateTargetGroup(ctx, "test-tg", 80, "HTTP", "vpc-12345")
			Expect(err).NotTo(HaveOccurred())
			targetGroupArn = tg.Arn
		})

		It("should register targets successfully", func() {
			// Register targets
			targets := []kecsELBv2.Target{
				{Id: "10.0.1.10", Port: 80},
				{Id: "10.0.1.11", Port: 80},
			}

			err := integration.RegisterTargets(ctx, targetGroupArn, targets)
			Expect(err).NotTo(HaveOccurred())

			// Get target health
			healthStatuses, err := integration.GetTargetHealth(ctx, targetGroupArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(healthStatuses).To(HaveLen(2))

			// Initially targets should be in "initial" state
			for _, health := range healthStatuses {
				Expect(health.HealthState).To(Equal("initial"))
			}
		})
	})

	Describe("CreateListener", func() {
		var loadBalancerArn, targetGroupArn string

		BeforeEach(func() {
			// Create load balancer and target group
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb", []string{"subnet-1"}, []string{"sg-1"})
			Expect(err).NotTo(HaveOccurred())
			loadBalancerArn = lb.Arn

			tg, err := integration.CreateTargetGroup(ctx, "test-tg", 80, "HTTP", "vpc-12345")
			Expect(err).NotTo(HaveOccurred())
			targetGroupArn = tg.Arn
		})

		It("should create a listener successfully", func() {
			// Create listener
			listener, err := integration.CreateListener(ctx, loadBalancerArn, 80, "HTTP", targetGroupArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(listener).NotTo(BeNil())
			Expect(listener.Port).To(Equal(int32(80)))
			Expect(listener.Protocol).To(Equal("HTTP"))
			Expect(listener.LoadBalancerArn).To(Equal(loadBalancerArn))
			Expect(listener.DefaultActions).To(HaveLen(1))
			Expect(listener.DefaultActions[0].TargetGroupArn).To(Equal(targetGroupArn))
		})
	})

	Describe("GetTargetHealth", func() {
		var targetGroupArn string

		BeforeEach(func() {
			// Create a target group and register targets
			tg, err := integration.CreateTargetGroup(ctx, "test-tg", 80, "HTTP", "vpc-12345")
			Expect(err).NotTo(HaveOccurred())
			targetGroupArn = tg.Arn

			targets := []kecsELBv2.Target{
				{Id: "10.0.1.10", Port: 80},
				{Id: "10.0.1.11", Port: 80},
			}
			err = integration.RegisterTargets(ctx, targetGroupArn, targets)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get target health successfully", func() {
			// Get target health
			healthStatuses, err := integration.GetTargetHealth(ctx, targetGroupArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(healthStatuses).To(HaveLen(2))

			// Check health status
			for _, health := range healthStatuses {
				Expect(health.Target.Port).To(Equal(int32(80)))
				Expect(health.HealthState).To(Equal("initial"))
				Expect(health.Description).To(Equal("Target registration is in progress"))
			}
		})

		It("should transition to healthy state", func() {
			// Wait for health transition (simulated in K8s implementation)
			time.Sleep(6 * time.Second)

			// Get target health again
			healthStatuses, err := integration.GetTargetHealth(ctx, targetGroupArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(healthStatuses).To(HaveLen(2))

			// Targets should be healthy now
			for _, health := range healthStatuses {
				Expect(health.HealthState).To(Equal("healthy"))
				Expect(health.Description).To(Equal("Health checks passed"))
			}
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
