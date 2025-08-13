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

// Mock ELBv2 store for testing
type mockELBv2Store struct {
	rules map[string]*storage.ELBv2Rule
}

func newMockELBv2Store() *mockELBv2Store {
	return &mockELBv2Store{
		rules: make(map[string]*storage.ELBv2Rule),
	}
}

func (m *mockELBv2Store) ListRules(ctx context.Context, listenerArn string) ([]*storage.ELBv2Rule, error) {
	var rules []*storage.ELBv2Rule
	for _, rule := range m.rules {
		if rule.ListenerArn == listenerArn {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

func (m *mockELBv2Store) GetRule(ctx context.Context, ruleArn string) (*storage.ELBv2Rule, error) {
	if rule, exists := m.rules[ruleArn]; exists {
		return rule, nil
	}
	return nil, fmt.Errorf("rule not found: %s", ruleArn)
}

func (m *mockELBv2Store) UpdateRule(ctx context.Context, rule *storage.ELBv2Rule) error {
	m.rules[rule.ARN] = rule
	return nil
}

// Implement other required methods
func (m *mockELBv2Store) CreateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	return nil
}
func (m *mockELBv2Store) GetListener(ctx context.Context, arn string) (*storage.ELBv2Listener, error) {
	return nil, nil
}
func (m *mockELBv2Store) UpdateListener(ctx context.Context, listener *storage.ELBv2Listener) error {
	return nil
}
func (m *mockELBv2Store) DeleteListener(ctx context.Context, arn string) error { return nil }
func (m *mockELBv2Store) ListListeners(ctx context.Context, lbArn string) ([]*storage.ELBv2Listener, error) {
	return nil, nil
}
func (m *mockELBv2Store) CreateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	return nil
}
func (m *mockELBv2Store) GetLoadBalancer(ctx context.Context, arn string) (*storage.ELBv2LoadBalancer, error) {
	return nil, nil
}
func (m *mockELBv2Store) UpdateLoadBalancer(ctx context.Context, lb *storage.ELBv2LoadBalancer) error {
	return nil
}
func (m *mockELBv2Store) DeleteLoadBalancer(ctx context.Context, arn string) error { return nil }
func (m *mockELBv2Store) ListLoadBalancers(ctx context.Context, prefix string) ([]*storage.ELBv2LoadBalancer, error) {
	return nil, nil
}
func (m *mockELBv2Store) CreateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	return nil
}
func (m *mockELBv2Store) GetTargetGroup(ctx context.Context, arn string) (*storage.ELBv2TargetGroup, error) {
	return nil, nil
}
func (m *mockELBv2Store) UpdateTargetGroup(ctx context.Context, tg *storage.ELBv2TargetGroup) error {
	return nil
}
func (m *mockELBv2Store) DeleteTargetGroup(ctx context.Context, arn string) error { return nil }
func (m *mockELBv2Store) ListTargetGroups(ctx context.Context, prefix string) ([]*storage.ELBv2TargetGroup, error) {
	return nil, nil
}
func (m *mockELBv2Store) CreateRule(ctx context.Context, rule *storage.ELBv2Rule) error { return nil }
func (m *mockELBv2Store) DeleteRule(ctx context.Context, arn string) error              { return nil }
func (m *mockELBv2Store) RegisterTargets(ctx context.Context, tgArn string, targets []*storage.ELBv2Target) error {
	return nil
}
func (m *mockELBv2Store) DeregisterTargets(ctx context.Context, tgArn string, targets []string) error {
	return nil
}
func (m *mockELBv2Store) GetTargetHealth(ctx context.Context, tgArn string) ([]storage.ELBv2TargetHealth, error) {
	return nil, nil
}
func (m *mockELBv2Store) UpdateTargetHealth(ctx context.Context, tgArn string, targetId string, health *storage.ELBv2TargetHealth) error {
	return nil
}
func (m *mockELBv2Store) GetLoadBalancerByName(ctx context.Context, name string) (*storage.ELBv2LoadBalancer, error) {
	return nil, nil
}
func (m *mockELBv2Store) GetTargetGroupByName(ctx context.Context, name string) (*storage.ELBv2TargetGroup, error) {
	return nil, nil
}
func (m *mockELBv2Store) ListTargets(ctx context.Context, tgArn string) ([]*storage.ELBv2Target, error) {
	return nil, nil
}

var _ = Describe("PriorityManager", func() {
	var (
		manager     *elbv2.PriorityManager
		store       *mockELBv2Store
		ctx         context.Context
		listenerArn string
	)

	BeforeEach(func() {
		store = newMockELBv2Store()
		manager = elbv2.NewPriorityManager(store)
		ctx = context.Background()
		listenerArn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:listener/app/my-lb/50dc6c495c0c9188/f2f7dc8efc522ab2"
	})

	Describe("GetNextAvailablePriority", func() {
		It("should find the first available priority in range", func() {
			// Add some existing rules
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    100,
			}
			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    102,
			}

			priority, err := manager.GetNextAvailablePriority(ctx, listenerArn, elbv2.PriorityRangeSpecific)
			Expect(err).NotTo(HaveOccurred())
			Expect(priority).To(Equal(int32(101)))
		})

		It("should return error when range is full", func() {
			// Fill the critical range
			for i := int32(1); i <= 99; i++ {
				store.rules[fmt.Sprintf("rule%d", i)] = &storage.ELBv2Rule{
					ARN:         fmt.Sprintf("rule%d", i),
					ListenerArn: listenerArn,
					Priority:    i,
				}
			}

			_, err := manager.GetNextAvailablePriority(ctx, listenerArn, elbv2.PriorityRangeCritical)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no available priority"))
		})
	})

	Describe("ValidatePriority", func() {
		It("should accept valid priority", func() {
			err := manager.ValidatePriority(ctx, listenerArn, 100, "")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject priority outside valid range", func() {
			err := manager.ValidatePriority(ctx, listenerArn, 50001, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be between 1 and 49999"))
		})

		It("should reject priority already in use", func() {
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    100,
			}

			err := manager.ValidatePriority(ctx, listenerArn, 100, "rule2")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already in use"))
		})

		It("should accept priority when updating same rule", func() {
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    100,
			}

			err := manager.ValidatePriority(ctx, listenerArn, 100, "rule1")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SetRulePriorities", func() {
		BeforeEach(func() {
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    100,
			}
			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    200,
			}
		})

		It("should update multiple rule priorities", func() {
			updates := []elbv2.RulePriorityUpdate{
				{RuleArn: "rule1", Priority: 150},
				{RuleArn: "rule2", Priority: 250},
			}

			err := manager.SetRulePriorities(ctx, updates)
			Expect(err).NotTo(HaveOccurred())

			Expect(store.rules["rule1"].Priority).To(Equal(int32(150)))
			Expect(store.rules["rule2"].Priority).To(Equal(int32(250)))
		})

		It("should reject duplicate priorities", func() {
			updates := []elbv2.RulePriorityUpdate{
				{RuleArn: "rule1", Priority: 150},
				{RuleArn: "rule2", Priority: 150},
			}

			err := manager.SetRulePriorities(ctx, updates)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate priority"))
		})
	})

	Describe("AnalyzeRulePriorities", func() {
		It("should analyze priority distribution", func() {
			// Add rules in different ranges
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    10,
			}
			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    150,
			}
			store.rules["rule3"] = &storage.ELBv2Rule{
				ARN:         "rule3",
				ListenerArn: listenerArn,
				Priority:    1500,
			}
			store.rules["rule4"] = &storage.ELBv2Rule{
				ARN:         "rule4",
				ListenerArn: listenerArn,
				Priority:    15000,
			}

			analysis, err := manager.AnalyzeRulePriorities(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			Expect(analysis.TotalRules).To(Equal(4))
			Expect(analysis.PriorityRanges["critical"]).To(Equal(1))
			Expect(analysis.PriorityRanges["specific"]).To(Equal(1))
			Expect(analysis.PriorityRanges["general"]).To(Equal(1))
			Expect(analysis.PriorityRanges["catchall"]).To(Equal(1))
		})

		It("should detect priority gaps", func() {
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    10,
			}
			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    100,
			}

			analysis, err := manager.AnalyzeRulePriorities(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			Expect(analysis.Gaps).To(HaveLen(1))
			Expect(analysis.Gaps[0].Start).To(Equal(int32(11)))
			Expect(analysis.Gaps[0].End).To(Equal(int32(99)))
			Expect(analysis.Gaps[0].Size).To(Equal(int32(89)))
		})

		It("should detect adjacent priorities", func() {
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    10,
			}
			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    11,
			}

			analysis, err := manager.AnalyzeRulePriorities(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())

			Expect(analysis.Conflicts).To(HaveLen(1))
			Expect(analysis.Conflicts[0].Message).To(ContainSubstring("Adjacent priorities"))
		})
	})

	Describe("ReorderRulesForClarity", func() {
		It("should reorder rules with consistent gaps", func() {
			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    5,
			}
			store.rules["rule2"] = &storage.ELBv2Rule{
				ARN:         "rule2",
				ListenerArn: listenerArn,
				Priority:    6,
			}
			store.rules["rule3"] = &storage.ELBv2Rule{
				ARN:         "rule3",
				ListenerArn: listenerArn,
				Priority:    100,
			}

			updates, err := manager.ReorderRulesForClarity(ctx, listenerArn, 10)
			Expect(err).NotTo(HaveOccurred())

			Expect(updates).To(HaveLen(3))
			Expect(updates[0].Priority).To(Equal(int32(10)))
			Expect(updates[1].Priority).To(Equal(int32(20)))
			Expect(updates[2].Priority).To(Equal(int32(30)))
		})
	})

	Describe("FindPriorityForConditions", func() {
		It("should suggest priority based on specificity", func() {
			// Very specific conditions
			conditions := []generated_elbv2.RuleCondition{
				{
					PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
						Values: []string{"/api/v2/users/123"},
					},
				},
				{
					HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
						HttpHeaderName: utils.Ptr("X-API-Key"),
						Values:         []string{"specific-key"},
					},
				},
			}

			priority, err := manager.FindPriorityForConditions(ctx, listenerArn, conditions)
			Expect(err).NotTo(HaveOccurred())
			Expect(priority).To(BeNumerically(">=", 100))
			Expect(priority).To(BeNumerically("<=", 999))
		})

		It("should use general range for less specific conditions", func() {
			// Less specific conditions
			conditions := []generated_elbv2.RuleCondition{
				{
					PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
						Values: []string{"/api/*"},
					},
				},
			}

			priority, err := manager.FindPriorityForConditions(ctx, listenerArn, conditions)
			Expect(err).NotTo(HaveOccurred())
			Expect(priority).To(BeNumerically(">=", 1000))
			Expect(priority).To(BeNumerically("<=", 9999))
		})
	})

	Describe("OptimizePriorities", func() {
		It("should suggest optimized priorities based on specificity", func() {
			// Create rules with conditions
			pathCondition := generated_elbv2.RuleCondition{
				PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
					Values: []string{"/api/v2/users/123"},
				},
			}
			conditionsJSON, _ := json.Marshal([]generated_elbv2.RuleCondition{pathCondition})

			store.rules["rule1"] = &storage.ELBv2Rule{
				ARN:         "rule1",
				ListenerArn: listenerArn,
				Priority:    5000, // Currently in wrong range
				Conditions:  string(conditionsJSON),
			}

			suggestions, err := manager.OptimizePriorities(ctx, listenerArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(suggestions).NotTo(BeEmpty())

			// Should suggest moving to specific range
			Expect(suggestions[0].Priority).To(BeNumerically("<", 1000))
		})
	})
})
