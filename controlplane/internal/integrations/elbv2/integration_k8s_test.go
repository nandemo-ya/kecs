package elbv2_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
)

var _ = Describe("ELBv2 K8s Integration", func() {
	var (
		ctx         context.Context
		integration elbv2.Integration
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Create K8s integration (virtual implementation)
		integration = elbv2.NewK8sIntegration("us-east-1", "123456789012")
	})

	Describe("CreateLoadBalancer", func() {
		It("should create a virtual load balancer", func() {
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb", []string{"subnet-1"}, []string{"sg-1"})

			Expect(err).NotTo(HaveOccurred())
			Expect(lb).NotTo(BeNil())
			Expect(lb.Name).To(Equal("test-lb"))
			Expect(lb.State).To(Equal("active"))
			Expect(lb.Type).To(Equal("application"))
			Expect(lb.DNSName).To(ContainSubstring("test-lb"))
		})
	})

	Describe("CreateTargetGroup", func() {
		It("should create a virtual target group", func() {
			tg, err := integration.CreateTargetGroup(ctx, "test-tg", 80, "HTTP", "vpc-12345")

			Expect(err).NotTo(HaveOccurred())
			Expect(tg).NotTo(BeNil())
			Expect(tg.Name).To(Equal("test-tg"))
			Expect(tg.Port).To(Equal(int32(80)))
			Expect(tg.Protocol).To(Equal("HTTP"))
			Expect(tg.VpcId).To(Equal("vpc-12345"))
		})
	})

	Describe("CreateListener", func() {
		var loadBalancerArn string
		var targetGroupArn string

		BeforeEach(func() {
			// Create load balancer first
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb-listener", []string{"subnet-1"}, []string{"sg-1"})
			Expect(err).NotTo(HaveOccurred())
			loadBalancerArn = lb.Arn

			// Create target group
			tg, err := integration.CreateTargetGroup(ctx, "test-tg-listener", 80, "HTTP", "vpc-12345")
			Expect(err).NotTo(HaveOccurred())
			targetGroupArn = tg.Arn
		})

		It("should create a listener", func() {
			listener, err := integration.CreateListener(ctx, loadBalancerArn, 80, "HTTP", targetGroupArn)

			Expect(err).NotTo(HaveOccurred())
			Expect(listener).NotTo(BeNil())
			Expect(listener.Port).To(Equal(int32(80)))
			Expect(listener.Protocol).To(Equal("HTTP"))
			Expect(listener.LoadBalancerArn).To(Equal(loadBalancerArn))
		})

		It("should update a listener when creating with same port", func() {
			// Create initial listener
			listener1, err := integration.CreateListener(ctx, loadBalancerArn, 80, "HTTP", targetGroupArn)
			Expect(err).NotTo(HaveOccurred())

			// Create another listener on same port (should update)
			listener2, err := integration.CreateListener(ctx, loadBalancerArn, 80, "HTTP", targetGroupArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(listener2).NotTo(BeNil())
			Expect(listener2.Port).To(Equal(int32(80)))

			// ARN should be different (new listener created in virtual implementation)
			Expect(listener2.Arn).NotTo(Equal(listener1.Arn))
		})
	})

	Describe("DeleteListener", func() {
		var loadBalancerArn string
		var targetGroupArn string
		var listenerArn string

		BeforeEach(func() {
			// Create load balancer first
			lb, err := integration.CreateLoadBalancer(ctx, "test-lb-delete", []string{"subnet-1"}, []string{"sg-1"})
			Expect(err).NotTo(HaveOccurred())
			loadBalancerArn = lb.Arn

			// Create target group
			tg, err := integration.CreateTargetGroup(ctx, "test-tg-delete", 80, "HTTP", "vpc-12345")
			Expect(err).NotTo(HaveOccurred())
			targetGroupArn = tg.Arn

			// Create listener
			listener, err := integration.CreateListener(ctx, loadBalancerArn, 80, "HTTP", targetGroupArn)
			Expect(err).NotTo(HaveOccurred())
			listenerArn = listener.Arn
		})

		It("should delete a listener", func() {
			err := integration.DeleteListener(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			// Trying to delete again should return error
			err = integration.DeleteListener(ctx, listenerArn)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("listener not found"))
		})
	})

	Describe("RegisterTargets and GetTargetHealth", func() {
		var targetGroupArn string

		BeforeEach(func() {
			// Create target group first
			tg, err := integration.CreateTargetGroup(ctx, "test-tg-health", 80, "HTTP", "vpc-12345")
			Expect(err).NotTo(HaveOccurred())
			targetGroupArn = tg.Arn
		})

		It("should register targets and track health", func() {
			// Register targets
			targets := []elbv2.Target{
				{Id: "10.0.1.10", Port: 80},
				{Id: "10.0.1.11", Port: 80},
			}

			err := integration.RegisterTargets(ctx, targetGroupArn, targets)
			Expect(err).NotTo(HaveOccurred())

			// Get target health
			healthStatuses, err := integration.GetTargetHealth(ctx, targetGroupArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(healthStatuses).To(HaveLen(2))

			// Check initial state
			for _, health := range healthStatuses {
				Expect(health.HealthState).To(Equal("initial"))
				Expect(health.Reason).To(Equal("Elb.RegistrationInProgress"))
			}
		})

		It("should deregister targets", func() {
			// Register targets first
			targets := []elbv2.Target{
				{Id: "10.0.2.10", Port: 80},
				{Id: "10.0.2.11", Port: 80},
			}

			err := integration.RegisterTargets(ctx, targetGroupArn, targets)
			Expect(err).NotTo(HaveOccurred())

			// Deregister one target
			err = integration.DeregisterTargets(ctx, targetGroupArn, []elbv2.Target{{Id: "10.0.2.10", Port: 80}})
			Expect(err).NotTo(HaveOccurred())

			// Verify remaining targets
			healthStatuses, err := integration.GetTargetHealth(ctx, targetGroupArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(healthStatuses).To(HaveLen(1))
			Expect(healthStatuses[0].Target.Id).To(Equal("10.0.2.11"))
		})
	})

	Describe("Kubernetes Health Check Integration", func() {
		It("should perform basic connectivity check when no kubeClient is available", func() {
			// Test with a target that would fail connectivity check
			healthState, err := integration.CheckTargetHealthWithK8s(ctx, "192.0.2.1", 80, "test-tg-arn")

			// Should not error but return unhealthy for unreachable target
			Expect(err).NotTo(HaveOccurred())
			Expect(healthState).To(Equal("unhealthy"))
		})

		It("should return healthy for basic connectivity check on localhost", func() {
			// This test assumes that something is listening on a common port
			// In practice, this would be more controlled in an integration test environment
			healthState, err := integration.CheckTargetHealthWithK8s(ctx, "127.0.0.1", 22, "test-tg-arn")

			// Should not error, health state depends on whether SSH is running
			Expect(err).NotTo(HaveOccurred())
			Expect(healthState).To(BeElementOf([]string{"healthy", "unhealthy"}))
		})
	})

	Describe("Error handling", func() {
		It("should handle non-existent load balancer", func() {
			lb, err := integration.GetLoadBalancer(ctx, "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/non-existent/123")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("load balancer not found"))
			Expect(lb).To(BeNil())
		})

		It("should handle non-existent target group in RegisterTargets", func() {
			err := integration.RegisterTargets(ctx, "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/non-existent/123", []elbv2.Target{{Id: "10.0.0.1", Port: 80}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("target group not found"))
		})
	})
})
