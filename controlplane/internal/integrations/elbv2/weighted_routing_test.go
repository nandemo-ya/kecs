package elbv2_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// Mock target group resolver
type mockTargetGroupResolver struct {
	targetGroups map[string]*generated_elbv2.TargetGroup
}

func (m *mockTargetGroupResolver) GetTargetGroupInfo(arn string) (*generated_elbv2.TargetGroup, error) {
	if tg, exists := m.targetGroups[arn]; exists {
		return tg, nil
	}
	return nil, fmt.Errorf("target group not found: %s", arn)
}

var _ = Describe("WeightedRoutingManager", func() {
	var manager *elbv2.WeightedRoutingManager
	var resolver *mockTargetGroupResolver

	BeforeEach(func() {
		manager = elbv2.NewWeightedRoutingManager()
		resolver = &mockTargetGroupResolver{
			targetGroups: map[string]*generated_elbv2.TargetGroup{
				"arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067": {
					TargetGroupArn:  utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
					TargetGroupName: utils.Ptr("api-v1"),
					Port:            utils.Ptr(int32(8080)),
				},
				"arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/83e2d6bc24d8a067": {
					TargetGroupArn:  utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/83e2d6bc24d8a067"),
					TargetGroupName: utils.Ptr("api-v2"),
					Port:            utils.Ptr(int32(8080)),
				},
			},
		}
	})

	Describe("ConvertActionsToWeightedServices", func() {
		Context("with simple forward action", func() {
			It("should convert to single service with 100% weight", func() {
				actions := []generated_elbv2.Action{
					{
						Type:           generated_elbv2.ActionTypeEnumFORWARD,
						TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
					},
				}

				services, err := manager.ConvertActionsToWeightedServices(actions, resolver)
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(HaveLen(1))
				Expect(services[0].Name).To(Equal("tg-api-v1"))
				Expect(services[0].Port).To(Equal(int32(8080)))
				Expect(services[0].Weight).To(Equal(int32(100)))
			})
		})

		Context("with weighted forward config", func() {
			It("should convert to multiple weighted services", func() {
				actions := []generated_elbv2.Action{
					{
						Type: generated_elbv2.ActionTypeEnumFORWARD,
						ForwardConfig: &generated_elbv2.ForwardActionConfig{
							TargetGroups: []generated_elbv2.TargetGroupTuple{
								{
									TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
									Weight:         utils.Ptr(int32(70)),
								},
								{
									TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/83e2d6bc24d8a067"),
									Weight:         utils.Ptr(int32(30)),
								},
							},
						},
					},
				}

				services, err := manager.ConvertActionsToWeightedServices(actions, resolver)
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(HaveLen(2))

				Expect(services[0].Name).To(Equal("tg-api-v1"))
				Expect(services[0].Weight).To(Equal(int32(70)))

				Expect(services[1].Name).To(Equal("tg-api-v2"))
				Expect(services[1].Weight).To(Equal(int32(30)))
			})

			It("should handle zero weights by distributing equally", func() {
				actions := []generated_elbv2.Action{
					{
						Type: generated_elbv2.ActionTypeEnumFORWARD,
						ForwardConfig: &generated_elbv2.ForwardActionConfig{
							TargetGroups: []generated_elbv2.TargetGroupTuple{
								{
									TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
									Weight:         utils.Ptr(int32(0)),
								},
								{
									TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/83e2d6bc24d8a067"),
									Weight:         utils.Ptr(int32(0)),
								},
							},
						},
					},
				}

				services, err := manager.ConvertActionsToWeightedServices(actions, resolver)
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(HaveLen(2))

				// Should distribute equally (50/50)
				Expect(services[0].Weight).To(Equal(int32(50)))
				Expect(services[1].Weight).To(Equal(int32(50)))
			})

			It("should add sticky session configuration when enabled", func() {
				actions := []generated_elbv2.Action{
					{
						Type: generated_elbv2.ActionTypeEnumFORWARD,
						ForwardConfig: &generated_elbv2.ForwardActionConfig{
							TargetGroups: []generated_elbv2.TargetGroupTuple{
								{
									TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
									Weight:         utils.Ptr(int32(100)),
								},
							},
							TargetGroupStickinessConfig: &generated_elbv2.TargetGroupStickinessConfig{
								Enabled:         utils.Ptr(true),
								DurationSeconds: utils.Ptr(int32(3600)),
							},
						},
					},
				}

				services, err := manager.ConvertActionsToWeightedServices(actions, resolver)
				Expect(err).NotTo(HaveOccurred())
				Expect(services).To(HaveLen(1))

				Expect(services[0].Sticky).NotTo(BeNil())
				Expect(services[0].Sticky.Cookie).NotTo(BeNil())
				Expect(services[0].Sticky.Cookie.Name).To(Equal("kecs-sticky-3600"))
				Expect(services[0].Sticky.Cookie.Secure).To(BeTrue())
				Expect(services[0].Sticky.Cookie.HTTPOnly).To(BeTrue())
				Expect(services[0].Sticky.Cookie.SameSite).To(Equal("lax"))
			})
		})
	})

	Describe("ValidateWeightedRouting", func() {
		It("should accept valid weighted routing configuration", func() {
			actions := []generated_elbv2.Action{
				{
					Type: generated_elbv2.ActionTypeEnumFORWARD,
					ForwardConfig: &generated_elbv2.ForwardActionConfig{
						TargetGroups: []generated_elbv2.TargetGroupTuple{
							{
								TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
								Weight:         utils.Ptr(int32(70)),
							},
							{
								TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/83e2d6bc24d8a067"),
								Weight:         utils.Ptr(int32(30)),
							},
						},
					},
				},
			}

			err := manager.ValidateWeightedRouting(actions)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject empty target groups", func() {
			actions := []generated_elbv2.Action{
				{
					Type: generated_elbv2.ActionTypeEnumFORWARD,
					ForwardConfig: &generated_elbv2.ForwardActionConfig{
						TargetGroups: []generated_elbv2.TargetGroupTuple{},
					},
				},
			}

			err := manager.ValidateWeightedRouting(actions)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("at least one target group"))
		})

		It("should reject more than 5 target groups", func() {
			targetGroups := make([]generated_elbv2.TargetGroupTuple, 6)
			for i := 0; i < 6; i++ {
				targetGroups[i] = generated_elbv2.TargetGroupTuple{
					TargetGroupArn: utils.Ptr(fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v%d/abc", i)),
					Weight:         utils.Ptr(int32(10)),
				}
			}

			actions := []generated_elbv2.Action{
				{
					Type: generated_elbv2.ActionTypeEnumFORWARD,
					ForwardConfig: &generated_elbv2.ForwardActionConfig{
						TargetGroups: targetGroups,
					},
				},
			}

			err := manager.ValidateWeightedRouting(actions)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("more than 5 target groups"))
		})

		It("should reject invalid weights", func() {
			actions := []generated_elbv2.Action{
				{
					Type: generated_elbv2.ActionTypeEnumFORWARD,
					ForwardConfig: &generated_elbv2.ForwardActionConfig{
						TargetGroups: []generated_elbv2.TargetGroupTuple{
							{
								TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v1/73e2d6bc24d8a067"),
								Weight:         utils.Ptr(int32(1000)), // Too high
							},
						},
					},
				},
			}

			err := manager.ValidateWeightedRouting(actions)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("weight must be between 0 and 999"))
		})
	})

	Describe("NormalizeWeights", func() {
		It("should normalize weights to sum to 100", func() {
			services := []elbv2.TraefikWeightedService{
				{Name: "service1", Weight: 30},
				{Name: "service2", Weight: 20},
				{Name: "service3", Weight: 10},
			}

			normalized := manager.NormalizeWeights(services)

			totalWeight := int32(0)
			for _, service := range normalized {
				totalWeight += service.Weight
			}
			Expect(totalWeight).To(Equal(int32(100)))
		})

		It("should distribute equally when all weights are 0", func() {
			services := []elbv2.TraefikWeightedService{
				{Name: "service1", Weight: 0},
				{Name: "service2", Weight: 0},
				{Name: "service3", Weight: 0},
			}

			normalized := manager.NormalizeWeights(services)

			// Should be approximately 33, 33, 34
			Expect(normalized[0].Weight).To(BeNumerically(">=", 33))
			Expect(normalized[1].Weight).To(Equal(int32(33)))
			Expect(normalized[2].Weight).To(Equal(int32(33)))

			totalWeight := int32(0)
			for _, service := range normalized {
				totalWeight += service.Weight
			}
			Expect(totalWeight).To(Equal(int32(100)))
		})
	})

	Describe("CalculateWeightDistribution", func() {
		It("should calculate expected request distribution", func() {
			services := []elbv2.TraefikWeightedService{
				{Name: "service1", Weight: 70},
				{Name: "service2", Weight: 30},
			}

			distribution := manager.CalculateWeightDistribution(services, 1000)

			Expect(distribution["service1"]).To(Equal(700))
			Expect(distribution["service2"]).To(Equal(300))
		})

		It("should handle rounding correctly", func() {
			services := []elbv2.TraefikWeightedService{
				{Name: "service1", Weight: 33},
				{Name: "service2", Weight: 33},
				{Name: "service3", Weight: 34},
			}

			distribution := manager.CalculateWeightDistribution(services, 100)

			total := 0
			for _, count := range distribution {
				total += count
			}
			Expect(total).To(Equal(100))
		})
	})

	Describe("GenerateTraefikServiceYAML", func() {
		It("should generate YAML for weighted services", func() {
			services := []elbv2.TraefikWeightedService{
				{
					Name:   "tg-api-v1",
					Port:   8080,
					Weight: 70,
				},
				{
					Name:   "tg-api-v2",
					Port:   8080,
					Weight: 30,
				},
			}

			yaml := manager.GenerateTraefikServiceYAML(services)

			Expect(yaml).To(ContainSubstring("services:"))
			Expect(yaml).To(ContainSubstring("- name: tg-api-v1"))
			Expect(yaml).To(ContainSubstring("  port: 8080"))
			Expect(yaml).To(ContainSubstring("  weight: 70"))
			Expect(yaml).To(ContainSubstring("- name: tg-api-v2"))
			Expect(yaml).To(ContainSubstring("  weight: 30"))
		})

		It("should include sticky session configuration", func() {
			services := []elbv2.TraefikWeightedService{
				{
					Name:   "tg-api",
					Port:   8080,
					Weight: 100,
					Sticky: &elbv2.TraefikSticky{
						Cookie: &elbv2.TraefikCookie{
							Name:     "kecs-sticky",
							Secure:   true,
							HTTPOnly: true,
							SameSite: "lax",
						},
					},
				},
			}

			yaml := manager.GenerateTraefikServiceYAML(services)

			Expect(yaml).To(ContainSubstring("sticky:"))
			Expect(yaml).To(ContainSubstring("cookie:"))
			Expect(yaml).To(ContainSubstring("name: kecs-sticky"))
			Expect(yaml).To(ContainSubstring("secure: true"))
			Expect(yaml).To(ContainSubstring("httpOnly: true"))
			Expect(yaml).To(ContainSubstring("sameSite: lax"))
		})
	})
})
