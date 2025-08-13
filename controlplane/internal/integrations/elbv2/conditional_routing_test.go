package elbv2_test

import (
	"context"
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/utils"
)

// Helper function to create forward action type
func forwardActionType() generated_elbv2.ActionTypeEnum {
	return generated_elbv2.ActionTypeEnumFORWARD
}

var _ = Describe("ConditionalRoutingManager", func() {
	var (
		manager     *elbv2.ConditionalRoutingManager
		store       *mockELBv2Store
		ctx         context.Context
		listenerArn string
	)

	BeforeEach(func() {
		store = newMockELBv2Store()
		manager = elbv2.NewConditionalRoutingManager(store)
		ctx = context.Background()
		listenerArn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2"
	})

	Describe("CreateConditionalRoute", func() {
		It("should create a route with multiple conditions", func() {
			route := elbv2.ConditionalRoute{
				Name:        "beta-api-v2",
				Description: "Route beta users to API v2",
				Conditions: []elbv2.ConditionalGroup{
					{
						Operator: "AND",
						Conditions: []generated_elbv2.RuleCondition{
							{
								Field: utils.Ptr("path-pattern"),
								PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
									Values: []string{"/api/*"},
								},
							},
							{
								Field: utils.Ptr("http-header"),
								HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
									HttpHeaderName: utils.Ptr("X-Beta-User"),
									Values:         []string{"true"},
								},
							},
						},
					},
				},
				Actions: []generated_elbv2.Action{
					{
						Type:           forwardActionType(),
						TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/api-v2/73e2d6bc24d8a067"),
					},
				},
			}

			rule, err := manager.CreateConditionalRoute(ctx, listenerArn, route)
			Expect(err).NotTo(HaveOccurred())
			Expect(rule).NotTo(BeNil())
			Expect(rule.Priority).To(BeNumerically(">=", 100))

			// Verify conditions were properly stored
			var conditions []generated_elbv2.RuleCondition
			err = json.Unmarshal([]byte(rule.Conditions), &conditions)
			Expect(err).NotTo(HaveOccurred())
			Expect(conditions).To(HaveLen(2))
		})

		It("should respect specified priority", func() {
			priority := int32(250)
			route := elbv2.ConditionalRoute{
				Name:        "custom-priority",
				Description: "Route with custom priority",
				Priority:    &priority,
				Conditions: []elbv2.ConditionalGroup{
					{
						Operator: "AND",
						Conditions: []generated_elbv2.RuleCondition{
							{
								Field: utils.Ptr("path-pattern"),
								PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
									Values: []string{"/custom/*"},
								},
							},
						},
					},
				},
				Actions: []generated_elbv2.Action{
					{
						Type:           forwardActionType(),
						TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/custom/83e2d6bc24d8a067"),
					},
				},
			}

			rule, err := manager.CreateConditionalRoute(ctx, listenerArn, route)
			Expect(err).NotTo(HaveOccurred())
			Expect(rule.Priority).To(Equal(priority))
		})
	})

	Describe("CreateIfThenElseRoutes", func() {
		It("should create multiple routes with proper priority ordering", func() {
			routes := []elbv2.ConditionalRoute{
				{
					Name:        "route-1",
					Description: "Less specific route",
					Conditions: []elbv2.ConditionalGroup{
						{
							Operator: "AND",
							Conditions: []generated_elbv2.RuleCondition{
								{
									Field: utils.Ptr("path-pattern"),
									PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
										Values: []string{"/api/*"},
									},
								},
							},
						},
					},
					Actions: []generated_elbv2.Action{
						{
							Type:           forwardActionType(),
							TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/general/93e2d6bc24d8a067"),
						},
					},
				},
				{
					Name:        "route-2",
					Description: "More specific route",
					Conditions: []elbv2.ConditionalGroup{
						{
							Operator: "AND",
							Conditions: []generated_elbv2.RuleCondition{
								{
									Field: utils.Ptr("path-pattern"),
									PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
										Values: []string{"/api/v2/users"},
									},
								},
								{
									Field: utils.Ptr("http-header"),
									HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
										HttpHeaderName: utils.Ptr("X-Premium"),
										Values:         []string{"true"},
									},
								},
							},
						},
					},
					Actions: []generated_elbv2.Action{
						{
							Type:           forwardActionType(),
							TargetGroupArn: utils.Ptr("arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/premium/a3e2d6bc24d8a067"),
						},
					},
				},
			}

			rules, err := manager.CreateIfThenElseRoutes(ctx, listenerArn, routes)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(2))

			// More specific route should have lower priority
			Expect(rules[0].Priority).To(BeNumerically("<", rules[1].Priority))
		})
	})

	Describe("CreateCanaryRoute", func() {
		It("should create a canary route with weighted routing", func() {
			config := elbv2.CanaryConfig{
				Name:              "feature-x",
				Paths:             []string{"/feature/*"},
				HeaderName:        "X-Canary",
				HeaderValues:      []string{"true"},
				CanaryTargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/canary/b3e2d6bc24d8a067",
				StableTargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/stable/c3e2d6bc24d8a067",
				CanaryWeight:      20,
				StableWeight:      80,
			}

			rule, err := manager.CreateCanaryRoute(ctx, listenerArn, config)
			Expect(err).NotTo(HaveOccurred())
			Expect(rule).NotTo(BeNil())

			// Verify actions contain weighted routing
			var actions []generated_elbv2.Action
			err = json.Unmarshal([]byte(rule.Actions), &actions)
			Expect(err).NotTo(HaveOccurred())
			Expect(actions).To(HaveLen(1))
			Expect(actions[0].ForwardConfig).NotTo(BeNil())
			Expect(actions[0].ForwardConfig.TargetGroups).To(HaveLen(2))
			Expect(*actions[0].ForwardConfig.TargetGroups[0].Weight).To(Equal(int32(20)))
			Expect(*actions[0].ForwardConfig.TargetGroups[1].Weight).To(Equal(int32(80)))
		})
	})

	Describe("CreateMultiStageRoute", func() {
		It("should create multi-stage rollout rules", func() {
			stages := []elbv2.StageConfig{
				{
					Name:        "internal-users",
					Description: "Internal users get new feature first",
					Paths:       []string{"/feature/*"},
					SourceIPs:   []string{"10.0.0.0/8"},
					TargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/internal/d3e2d6bc24d8a067",
				},
				{
					Name:        "beta-users",
					Description: "Beta users get feature second",
					Paths:       []string{"/feature/*"},
					Headers: map[string][]string{
						"X-User-Group": {"beta", "early-adopter"},
					},
					TargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/beta/e3e2d6bc24d8a067",
				},
				{
					Name:        "general-users",
					Description: "General availability",
					Paths:       []string{"/feature/*"},
					TargetGroup: "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/ga/f3e2d6bc24d8a067",
				},
			}

			rules, err := manager.CreateMultiStageRoute(ctx, listenerArn, stages)
			Expect(err).NotTo(HaveOccurred())
			Expect(rules).To(HaveLen(3))

			// Verify priority ordering
			Expect(rules[0].Priority).To(BeNumerically("<", rules[1].Priority))
			Expect(rules[1].Priority).To(BeNumerically("<", rules[2].Priority))
		})
	})

	Describe("AnalyzeConditionalRoutes", func() {
		BeforeEach(func() {
			// Add some test rules
			conditions1 := []generated_elbv2.RuleCondition{
				{
					Field: utils.Ptr("path-pattern"),
					PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
						Values: []string{"/api/*"},
					},
				},
			}
			conditions1JSON, _ := json.Marshal(conditions1)

			conditions2 := []generated_elbv2.RuleCondition{
				{
					Field: utils.Ptr("path-pattern"),
					PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
						Values: []string{"/api/*"},
					},
				},
				{
					Field: utils.Ptr("http-header"),
					HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
						HttpHeaderName: utils.Ptr("X-Version"),
						Values:         []string{"2.0"},
					},
				},
			}
			conditions2JSON, _ := json.Marshal(conditions2)

			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    100,
				Conditions:  string(conditions1JSON),
			}

			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    200,
				Conditions:  string(conditions2JSON),
			}
		})

		It("should analyze conditional routing complexity", func() {
			analysis, err := manager.AnalyzeConditionalRoutes(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			Expect(analysis.TotalRules).To(Equal(2))
			Expect(analysis.ConditionalRules).To(Equal(1)) // Only rule2 has multiple conditions
			Expect(analysis.AverageConditions).To(BeNumerically("==", 1.5))
			Expect(analysis.ConditionTypes["path-pattern"]).To(Equal(2))
			Expect(analysis.ConditionTypes["http-header"]).To(Equal(1))
		})

		It("should detect potential conflicts", func() {
			analysis, err := manager.AnalyzeConditionalRoutes(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			// Should detect that rule1 might catch traffic intended for rule2
			Expect(analysis.PotentialConflicts).NotTo(BeEmpty())
		})

		It("should provide optimization tips", func() {
			// Add more complex rules
			for i := 0; i < 5; i++ {
				conditions := []generated_elbv2.RuleCondition{}
				for j := 0; j < 4; j++ {
					conditions = append(conditions, generated_elbv2.RuleCondition{
						Field: utils.Ptr("http-header"),
						HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
							HttpHeaderName: utils.Ptr(fmt.Sprintf("X-Header-%d", j)),
							Values:         []string{"value"},
						},
					})
				}
				conditionsJSON, _ := json.Marshal(conditions)

				store.rules[fmt.Sprintf("complex-rule%d", i)] = &storage.ELBv2Rule{
					ARN:         fmt.Sprintf("complex-rule%d", i),
					ListenerArn: listenerArn,
					Priority:    int32(300 + i),
					Conditions:  string(conditionsJSON),
				}
			}

			analysis, err := manager.AnalyzeConditionalRoutes(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			Expect(analysis.OptimizationTips).NotTo(BeEmpty())
			Expect(analysis.OptimizationTips[0]).To(ContainSubstring("simplifying rules"))
		})
	})
})
