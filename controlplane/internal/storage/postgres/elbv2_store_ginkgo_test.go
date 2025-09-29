package postgres_test

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("ELBv2Store", func() {
	var (
		store storage.Storage
		ctx   context.Context
	)

	BeforeEach(func() {
		store = setupTestDB()
		ctx = context.Background()
	})

	AfterEach(func() {
		// Don't close the shared connection, just clean data
		cleanupDatabase()
	})

	Describe("Load Balancer Operations", func() {
		Describe("CreateLoadBalancer", func() {
			Context("when creating a new load balancer", func() {
				It("should create the load balancer successfully", func() {
					lb := &storage.ELBv2LoadBalancer{
						ARN:                   "arn:aws:elasticloadbalancing:us-east-1:000000000000:loadbalancer/app/test-lb/1234567890abcdef",
						Name:                  "test-lb",
						DNSName:               "test-lb-123456789.us-east-1.elb.amazonaws.com",
						CanonicalHostedZoneID: "Z35SXDOTRQ7X7K",
						State:                 "provisioning",
						Type:                  "application",
						Scheme:                "internet-facing",
						VpcID:                 "vpc-12345678",
						Subnets:               []string{"subnet-12345678", "subnet-87654321"},
						AvailabilityZones:     []string{"us-east-1a", "us-east-1b"},
						SecurityGroups:        []string{"sg-12345678"},
						IpAddressType:         "ipv4",
						Tags:                  map[string]string{"Environment": "test"},
						Region:                "us-east-1",
						AccountID:             "000000000000",
						CreatedAt:             time.Now(),
						UpdatedAt:             time.Now(),
					}

					err := store.ELBv2Store().CreateLoadBalancer(ctx, lb)
					Expect(err).NotTo(HaveOccurred())

					// Verify load balancer was created
					retrieved, err := store.ELBv2Store().GetLoadBalancer(ctx, lb.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Name).To(Equal(lb.Name))
					Expect(retrieved.Type).To(Equal(lb.Type))
				})
			})

			Context("when creating a duplicate load balancer", func() {
				It("should return ErrResourceAlreadyExists", func() {
					lb := createTestLoadBalancer(store, "duplicate-lb")

					// Try to create duplicate
					lb2 := &storage.ELBv2LoadBalancer{
						ARN:       lb.ARN, // Same ARN
						Name:      "duplicate-lb-2",
						Region:    "us-east-1",
						AccountID: "000000000000",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}

					err := store.ELBv2Store().CreateLoadBalancer(ctx, lb2)
					Expect(err).To(MatchError(storage.ErrResourceAlreadyExists))
				})
			})
		})

		Describe("GetLoadBalancer", func() {
			Context("when getting an existing load balancer", func() {
				It("should return the load balancer", func() {
					lb := createTestLoadBalancer(store, "test-get-lb")

					retrieved, err := store.ELBv2Store().GetLoadBalancer(ctx, lb.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.ARN).To(Equal(lb.ARN))
					Expect(retrieved.Name).To(Equal(lb.Name))
				})
			})

			Context("when getting a non-existent load balancer", func() {
				It("should return ErrResourceNotFound", func() {
					_, err := store.ELBv2Store().GetLoadBalancer(ctx, "arn:aws:elasticloadbalancing:us-east-1:000000000000:loadbalancer/app/non-existent/1234567890abcdef")
					Expect(err).To(MatchError(storage.ErrResourceNotFound))
				})
			})
		})

		Describe("GetLoadBalancerByName", func() {
			Context("when getting by name", func() {
				It("should return the load balancer", func() {
					lb := createTestLoadBalancer(store, "test-by-name")

					retrieved, err := store.ELBv2Store().GetLoadBalancerByName(ctx, "test-by-name")
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.ARN).To(Equal(lb.ARN))
				})
			})
		})

		Describe("ListLoadBalancers", func() {
			Context("when listing load balancers", func() {
				It("should return all load balancers in region", func() {
					// Create load balancers
					for i := 0; i < 3; i++ {
						createTestLoadBalancer(store, fmt.Sprintf("test-list-lb-%d", i))
					}

					lbs, err := store.ELBv2Store().ListLoadBalancers(ctx, "us-east-1")
					Expect(err).NotTo(HaveOccurred())
					Expect(lbs).To(HaveLen(3))
				})
			})
		})

		Describe("UpdateLoadBalancer", func() {
			Context("when updating a load balancer", func() {
				It("should update successfully", func() {
					lb := createTestLoadBalancer(store, "test-update-lb")

					// Update the load balancer
					lb.State = "active"
					lb.DNSName = "updated-dns-name.elb.amazonaws.com"
					lb.UpdatedAt = time.Now()

					err := store.ELBv2Store().UpdateLoadBalancer(ctx, lb)
					Expect(err).NotTo(HaveOccurred())

					// Verify update
					retrieved, err := store.ELBv2Store().GetLoadBalancer(ctx, lb.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.State).To(Equal("active"))
					Expect(retrieved.DNSName).To(Equal("updated-dns-name.elb.amazonaws.com"))
				})
			})
		})

		Describe("DeleteLoadBalancer", func() {
			Context("when deleting a load balancer", func() {
				It("should delete successfully", func() {
					lb := createTestLoadBalancer(store, "test-delete-lb")

					err := store.ELBv2Store().DeleteLoadBalancer(ctx, lb.ARN)
					Expect(err).NotTo(HaveOccurred())

					// Verify deletion
					_, err = store.ELBv2Store().GetLoadBalancer(ctx, lb.ARN)
					Expect(err).To(MatchError(storage.ErrResourceNotFound))
				})
			})
		})
	})

	Describe("Target Group Operations", func() {
		Describe("CreateTargetGroup", func() {
			Context("when creating a new target group", func() {
				It("should create the target group successfully", func() {
					tg := &storage.ELBv2TargetGroup{
						ARN:                        "arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/test-tg/1234567890abcdef",
						Name:                       "test-tg",
						Protocol:                   "HTTP",
						Port:                       80,
						VpcID:                      "vpc-12345678",
						TargetType:                 "instance",
						HealthCheckEnabled:         true,
						HealthCheckProtocol:        "HTTP",
						HealthCheckPort:            "traffic-port",
						HealthCheckPath:            "/health",
						HealthCheckIntervalSeconds: 30,
						HealthCheckTimeoutSeconds:  5,
						HealthyThresholdCount:      2,
						UnhealthyThresholdCount:    2,
						Matcher:                    "200",
						LoadBalancerArns:           []string{},
						Tags:                       map[string]string{"Environment": "test"},
						Region:                     "us-east-1",
						AccountID:                  "000000000000",
						CreatedAt:                  time.Now(),
						UpdatedAt:                  time.Now(),
					}

					err := store.ELBv2Store().CreateTargetGroup(ctx, tg)
					Expect(err).NotTo(HaveOccurred())

					// Verify target group was created
					retrieved, err := store.ELBv2Store().GetTargetGroup(ctx, tg.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Name).To(Equal(tg.Name))
					Expect(retrieved.Protocol).To(Equal(tg.Protocol))
				})
			})
		})

		Describe("GetTargetGroup", func() {
			Context("when getting an existing target group", func() {
				It("should return the target group", func() {
					tg := createTestTargetGroup(store, "test-get-tg")

					retrieved, err := store.ELBv2Store().GetTargetGroup(ctx, tg.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.ARN).To(Equal(tg.ARN))
					Expect(retrieved.Name).To(Equal(tg.Name))
				})
			})
		})

		Describe("ListTargetGroups", func() {
			Context("when listing target groups", func() {
				It("should return all target groups in region", func() {
					// Create target groups
					for i := 0; i < 3; i++ {
						createTestTargetGroup(store, fmt.Sprintf("test-list-tg-%d", i))
					}

					tgs, err := store.ELBv2Store().ListTargetGroups(ctx, "us-east-1")
					Expect(err).NotTo(HaveOccurred())
					Expect(tgs).To(HaveLen(3))
				})
			})
		})
	})

	Describe("Listener Operations", func() {
		var lb *storage.ELBv2LoadBalancer
		var tg *storage.ELBv2TargetGroup

		BeforeEach(func() {
			// Create a load balancer and target group for listener tests
			lb = createTestLoadBalancer(store, "listener-test-lb")
			tg = createTestTargetGroup(store, "listener-test-tg")
		})

		Describe("CreateListener", func() {
			Context("when creating a new listener", func() {
				It("should create the listener successfully", func() {
					listener := &storage.ELBv2Listener{
						ARN:             fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:listener/app/listener-test-lb/%s/1234567890abcdef", uuid.New().String()),
						LoadBalancerArn: lb.ARN,
						Port:            80,
						Protocol:        "HTTP",
						DefaultActions:  fmt.Sprintf(`[{"Type":"forward","TargetGroupArn":"%s"}]`, tg.ARN),
						Region:          "us-east-1",
						AccountID:       "000000000000",
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					}

					err := store.ELBv2Store().CreateListener(ctx, listener)
					Expect(err).NotTo(HaveOccurred())

					// Verify listener was created
					retrieved, err := store.ELBv2Store().GetListener(ctx, listener.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Port).To(Equal(listener.Port))
					Expect(retrieved.Protocol).To(Equal(listener.Protocol))
				})
			})
		})

		Describe("ListListeners", func() {
			Context("when listing listeners for a load balancer", func() {
				It("should return all listeners", func() {
					// Create listeners
					for i := 0; i < 3; i++ {
						listener := &storage.ELBv2Listener{
							ARN:             fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:listener/app/listener-test-lb/%s/%d", uuid.New().String(), i),
							LoadBalancerArn: lb.ARN,
							Port:            int32(8080 + i),
							Protocol:        "HTTP",
							DefaultActions:  fmt.Sprintf(`[{"Type":"forward","TargetGroupArn":"%s"}]`, tg.ARN),
							Region:          "us-east-1",
							AccountID:       "000000000000",
							CreatedAt:       time.Now(),
							UpdatedAt:       time.Now(),
						}
						err := store.ELBv2Store().CreateListener(ctx, listener)
						Expect(err).NotTo(HaveOccurred())
					}

					listeners, err := store.ELBv2Store().ListListeners(ctx, lb.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(listeners).To(HaveLen(3))
				})
			})
		})
	})

	Describe("Target Operations", func() {
		var tg *storage.ELBv2TargetGroup

		BeforeEach(func() {
			// Create a target group for target tests
			tg = createTestTargetGroup(store, "target-test-tg")
		})

		Describe("RegisterTargets", func() {
			Context("when registering targets", func() {
				It("should register targets successfully", func() {
					targets := []*storage.ELBv2Target{
						{
							TargetGroupArn:   tg.ARN,
							ID:               "i-1234567890abcdef0",
							Port:             80,
							AvailabilityZone: "us-east-1a",
							HealthState:      "initial",
							RegisteredAt:     time.Now(),
							UpdatedAt:        time.Now(),
						},
						{
							TargetGroupArn:   tg.ARN,
							ID:               "i-0987654321fedcba0",
							Port:             80,
							AvailabilityZone: "us-east-1b",
							HealthState:      "initial",
							RegisteredAt:     time.Now(),
							UpdatedAt:        time.Now(),
						},
					}

					err := store.ELBv2Store().RegisterTargets(ctx, tg.ARN, targets)
					Expect(err).NotTo(HaveOccurred())

					// Verify targets were registered
					registeredTargets, err := store.ELBv2Store().ListTargets(ctx, tg.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(registeredTargets).To(HaveLen(2))
				})
			})
		})

		Describe("DeregisterTargets", func() {
			Context("when deregistering targets", func() {
				It("should deregister targets successfully", func() {
					// Register targets first
					target := &storage.ELBv2Target{
						TargetGroupArn: tg.ARN,
						ID:             "i-deregister-test",
						Port:           80,
						HealthState:    "healthy",
						RegisteredAt:   time.Now(),
						UpdatedAt:      time.Now(),
					}
					err := store.ELBv2Store().RegisterTargets(ctx, tg.ARN, []*storage.ELBv2Target{target})
					Expect(err).NotTo(HaveOccurred())

					// Deregister the target
					err = store.ELBv2Store().DeregisterTargets(ctx, tg.ARN, []string{"i-deregister-test"})
					Expect(err).NotTo(HaveOccurred())

					// Verify target was deregistered
					targets, err := store.ELBv2Store().ListTargets(ctx, tg.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(targets).To(BeEmpty())
				})
			})
		})

		Describe("UpdateTargetHealth", func() {
			Context("when updating target health", func() {
				It("should update health successfully", func() {
					// Register a target
					target := &storage.ELBv2Target{
						TargetGroupArn: tg.ARN,
						ID:             "i-health-test",
						Port:           80,
						HealthState:    "initial",
						RegisteredAt:   time.Now(),
						UpdatedAt:      time.Now(),
					}
					err := store.ELBv2Store().RegisterTargets(ctx, tg.ARN, []*storage.ELBv2Target{target})
					Expect(err).NotTo(HaveOccurred())

					// Update target health
					health := &storage.ELBv2TargetHealth{
						State:       "healthy",
						Reason:      "Target.ResponseCodeMismatch",
						Description: "Health checks passed",
					}
					err = store.ELBv2Store().UpdateTargetHealth(ctx, tg.ARN, "i-health-test", health)
					Expect(err).NotTo(HaveOccurred())

					// Verify health was updated
					targets, err := store.ELBv2Store().ListTargets(ctx, tg.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(targets).To(HaveLen(1))
					Expect(targets[0].HealthState).To(Equal("healthy"))
				})
			})
		})
	})

	Describe("Rule Operations", func() {
		var listener *storage.ELBv2Listener

		BeforeEach(func() {
			// Create a load balancer, target group, and listener for rule tests
			lb := createTestLoadBalancer(store, "rule-test-lb")
			tg := createTestTargetGroup(store, "rule-test-tg")
			listener = &storage.ELBv2Listener{
				ARN:             fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:listener/app/rule-test-lb/%s/1234567890abcdef", uuid.New().String()),
				LoadBalancerArn: lb.ARN,
				Port:            80,
				Protocol:        "HTTP",
				DefaultActions:  fmt.Sprintf(`[{"Type":"forward","TargetGroupArn":"%s"}]`, tg.ARN),
				Region:          "us-east-1",
				AccountID:       "000000000000",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			err := store.ELBv2Store().CreateListener(ctx, listener)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("CreateRule", func() {
			Context("when creating a new rule", func() {
				It("should create the rule successfully", func() {
					rule := &storage.ELBv2Rule{
						ARN:         fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:listener-rule/app/rule-test-lb/%s", uuid.New().String()),
						ListenerArn: listener.ARN,
						Priority:    100,
						Conditions:  `[{"Field":"path-pattern","Values":["/*"]}]`,
						Actions:     `[{"Type":"forward","TargetGroupArn":"arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/test-tg/1234567890abcdef"}]`,
						IsDefault:   false,
						Region:      "us-east-1",
						AccountID:   "000000000000",
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}

					err := store.ELBv2Store().CreateRule(ctx, rule)
					Expect(err).NotTo(HaveOccurred())

					// Verify rule was created
					retrieved, err := store.ELBv2Store().GetRule(ctx, rule.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(retrieved.Priority).To(Equal(rule.Priority))
				})
			})
		})

		Describe("ListRules", func() {
			Context("when listing rules for a listener", func() {
				It("should return all rules", func() {
					// Create rules
					for i := 0; i < 3; i++ {
						rule := &storage.ELBv2Rule{
							ARN:         fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:listener-rule/app/rule-test-lb/%s", uuid.New().String()),
							ListenerArn: listener.ARN,
							Priority:    int32(100 + i),
							Conditions:  fmt.Sprintf(`[{"Field":"path-pattern","Values":["/path%d"]}]`, i),
							Actions:     `[{"Type":"forward","TargetGroupArn":"arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/test-tg/1234567890abcdef"}]`,
							IsDefault:   false,
							Region:      "us-east-1",
							AccountID:   "000000000000",
							CreatedAt:   time.Now(),
							UpdatedAt:   time.Now(),
						}
						err := store.ELBv2Store().CreateRule(ctx, rule)
						Expect(err).NotTo(HaveOccurred())
					}

					rules, err := store.ELBv2Store().ListRules(ctx, listener.ARN)
					Expect(err).NotTo(HaveOccurred())
					Expect(rules).To(HaveLen(3))
				})
			})
		})
	})
})

// Helper functions
func createTestLoadBalancer(store storage.Storage, name string) *storage.ELBv2LoadBalancer {
	lb := &storage.ELBv2LoadBalancer{
		ARN:                   fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:loadbalancer/app/%s/%s", name, uuid.New().String()),
		Name:                  name,
		DNSName:               fmt.Sprintf("%s-123456789.us-east-1.elb.amazonaws.com", name),
		CanonicalHostedZoneID: "Z35SXDOTRQ7X7K",
		State:                 "active",
		Type:                  "application",
		Scheme:                "internet-facing",
		VpcID:                 "vpc-12345678",
		Subnets:               []string{"subnet-12345678", "subnet-87654321"},
		AvailabilityZones:     []string{"us-east-1a", "us-east-1b"},
		SecurityGroups:        []string{"sg-12345678"},
		IpAddressType:         "ipv4",
		Region:                "us-east-1",
		AccountID:             "000000000000",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
	err := store.ELBv2Store().CreateLoadBalancer(context.Background(), lb)
	Expect(err).NotTo(HaveOccurred())
	return lb
}

func createTestTargetGroup(store storage.Storage, name string) *storage.ELBv2TargetGroup {
	tg := &storage.ELBv2TargetGroup{
		ARN:                        fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:000000000000:targetgroup/%s/%s", name, uuid.New().String()),
		Name:                       name,
		Protocol:                   "HTTP",
		Port:                       80,
		VpcID:                      "vpc-12345678",
		TargetType:                 "instance",
		HealthCheckEnabled:         true,
		HealthCheckProtocol:        "HTTP",
		HealthCheckPort:            "traffic-port",
		HealthCheckPath:            "/health",
		HealthCheckIntervalSeconds: 30,
		HealthCheckTimeoutSeconds:  5,
		HealthyThresholdCount:      2,
		UnhealthyThresholdCount:    2,
		Matcher:                    "200",
		Region:                     "us-east-1",
		AccountID:                  "000000000000",
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}
	err := store.ELBv2Store().CreateTargetGroup(context.Background(), tg)
	Expect(err).NotTo(HaveOccurred())
	return tg
}
