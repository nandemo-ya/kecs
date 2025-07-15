package duckdb

import (
	"context"
	"database/sql"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ELBv2Store Relationship Management", func() {
	var (
		ctx        context.Context
		db         *sql.DB
		elbv2Store storage.ELBv2Store
	)

	BeforeEach(func() {
		ctx = context.Background()
		
		// Create in-memory DuckDB for testing
		var err error
		db, err = sql.Open("duckdb", ":memory:")
		Expect(err).NotTo(HaveOccurred())
		
		// Run migrations
		s := &DuckDBStorage{db: db}
		err = s.migrateSchema(ctx)
		Expect(err).NotTo(HaveOccurred())
		
		elbv2Store = NewELBv2Store(db)
	})

	AfterEach(func() {
		if db != nil {
			db.Close()
		}
	})

	Describe("LoadBalancer-TargetGroup Relationship", func() {
		var (
			loadBalancer *storage.ELBv2LoadBalancer
			targetGroup  *storage.ELBv2TargetGroup
			listener     *storage.ELBv2Listener
		)

		BeforeEach(func() {
			// Create test load balancer
			loadBalancer = &storage.ELBv2LoadBalancer{
				ARN:       "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb/1234567890",
				Name:      "test-lb",
				DNSName:   "test-lb.us-east-1.elb.amazonaws.com",
				State:     "active",
				Type:      "application",
				Scheme:    "internet-facing",
				VpcID:     "vpc-12345",
				Region:    "us-east-1",
				AccountID: "123456789012",
			}
			err := elbv2Store.CreateLoadBalancer(ctx, loadBalancer)
			Expect(err).NotTo(HaveOccurred())

			// Create test target group
			targetGroup = &storage.ELBv2TargetGroup{
				ARN:                "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/test-tg/1234567890",
				Name:               "test-tg",
				Protocol:           "HTTP",
				Port:               80,
				VpcID:              "vpc-12345",
				TargetType:         "instance",
				LoadBalancerArns:   []string{},
				Region:             "us-east-1",
				AccountID:          "123456789012",
			}
			err = elbv2Store.CreateTargetGroup(ctx, targetGroup)
			Expect(err).NotTo(HaveOccurred())

			// Create test listener
			listener = &storage.ELBv2Listener{
				ARN:             "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/test-lb/1234567890/1234567890",
				LoadBalancerArn: loadBalancer.ARN,
				Port:            80,
				Protocol:        "HTTP",
				DefaultActions:  `[{"Type":"forward","TargetGroupArn":"` + targetGroup.ARN + `"}]`,
				Region:          "us-east-1",
				AccountID:       "123456789012",
			}
			err = elbv2Store.CreateListener(ctx, listener)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("AssociateTargetGroupWithLoadBalancer", func() {
			It("should associate target group with load balancer", func() {
				err := elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Verify association
				tg, err := elbv2Store.GetTargetGroup(ctx, targetGroup.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(tg.LoadBalancerArns).To(ContainElement(loadBalancer.ARN))
			})

			It("should not duplicate associations", func() {
				// Associate once
				err := elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Associate again
				err = elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Verify only one association exists
				tg, err := elbv2Store.GetTargetGroup(ctx, targetGroup.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(tg.LoadBalancerArns).To(HaveLen(1))
			})

			It("should handle non-existent target group", func() {
				err := elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, "non-existent-arn", loadBalancer.ARN)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})

		Context("DisassociateTargetGroupFromLoadBalancer", func() {
			BeforeEach(func() {
				// Associate first
				err := elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should disassociate target group from load balancer", func() {
				err := elbv2Store.DisassociateTargetGroupFromLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Verify disassociation
				tg, err := elbv2Store.GetTargetGroup(ctx, targetGroup.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(tg.LoadBalancerArns).NotTo(ContainElement(loadBalancer.ARN))
			})

			It("should handle non-existent associations gracefully", func() {
				// Disassociate non-existent
				err := elbv2Store.DisassociateTargetGroupFromLoadBalancer(ctx, targetGroup.ARN, "non-existent-lb-arn")
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("GetTargetGroupsByLoadBalancer", func() {
			It("should return target groups associated with load balancer", func() {
				// Associate target group
				err := elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Get target groups
				targetGroups, err := elbv2Store.GetTargetGroupsByLoadBalancer(ctx, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(targetGroups).To(HaveLen(1))
				Expect(targetGroups[0].ARN).To(Equal(targetGroup.ARN))
			})

			It("should return empty list for load balancer with no listeners", func() {
				// Create another load balancer without listeners
				lb2 := &storage.ELBv2LoadBalancer{
					ARN:       "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb2/1234567890",
					Name:      "test-lb2",
					DNSName:   "test-lb2.us-east-1.elb.amazonaws.com",
					State:     "active",
					Type:      "application",
					Scheme:    "internet-facing",
					VpcID:     "vpc-12345",
					Region:    "us-east-1",
					AccountID: "123456789012",
				}
				err := elbv2Store.CreateLoadBalancer(ctx, lb2)
				Expect(err).NotTo(HaveOccurred())

				targetGroups, err := elbv2Store.GetTargetGroupsByLoadBalancer(ctx, lb2.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(targetGroups).To(BeEmpty())
			})
		})

		Context("GetLoadBalancersByTargetGroup", func() {
			It("should return load balancers associated with target group", func() {
				// Associate target group
				err := elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Get load balancers
				loadBalancers, err := elbv2Store.GetLoadBalancersByTargetGroup(ctx, targetGroup.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(loadBalancers).To(HaveLen(1))
				Expect(loadBalancers[0].ARN).To(Equal(loadBalancer.ARN))
			})

			It("should handle multiple associations", func() {
				// Create another load balancer
				lb2 := &storage.ELBv2LoadBalancer{
					ARN:       "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/test-lb2/1234567890",
					Name:      "test-lb2",
					DNSName:   "test-lb2.us-east-1.elb.amazonaws.com",
					State:     "active",
					Type:      "application",
					Scheme:    "internet-facing",
					VpcID:     "vpc-12345",
					Region:    "us-east-1",
					AccountID: "123456789012",
				}
				err := elbv2Store.CreateLoadBalancer(ctx, lb2)
				Expect(err).NotTo(HaveOccurred())

				// Associate with both load balancers
				err = elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, loadBalancer.ARN)
				Expect(err).NotTo(HaveOccurred())
				err = elbv2Store.AssociateTargetGroupWithLoadBalancer(ctx, targetGroup.ARN, lb2.ARN)
				Expect(err).NotTo(HaveOccurred())

				// Get load balancers
				loadBalancers, err := elbv2Store.GetLoadBalancersByTargetGroup(ctx, targetGroup.ARN)
				Expect(err).NotTo(HaveOccurred())
				Expect(loadBalancers).To(HaveLen(2))
			})
		})
	})
})