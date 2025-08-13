package elbv2

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// PriorityManager manages rule priorities for ELBv2 listeners
type PriorityManager struct {
	store storage.ELBv2Store
}

// NewPriorityManager creates a new priority manager
func NewPriorityManager(store storage.ELBv2Store) *PriorityManager {
	return &PriorityManager{
		store: store,
	}
}

// PriorityRange represents a range of priorities
type PriorityRange struct {
	Name        string
	Description string
	Start       int32
	End         int32
}

// Common priority ranges
var (
	PriorityRangeCritical = PriorityRange{
		Name:        "critical",
		Description: "Critical system routes (health checks, admin)",
		Start:       1,
		End:         99,
	}
	PriorityRangeSpecific = PriorityRange{
		Name:        "specific",
		Description: "Specific application routes",
		Start:       100,
		End:         999,
	}
	PriorityRangeGeneral = PriorityRange{
		Name:        "general",
		Description: "General application routes",
		Start:       1000,
		End:         9999,
	}
	PriorityRangeCatchAll = PriorityRange{
		Name:        "catchall",
		Description: "Catch-all and default routes",
		Start:       10000,
		End:         49999,
	}
)

// GetNextAvailablePriority finds the next available priority in a range
func (p *PriorityManager) GetNextAvailablePriority(ctx context.Context, listenerArn string, priorityRange PriorityRange) (int32, error) {
	// Get all rules for the listener
	rules, err := p.store.ListRules(ctx, listenerArn)
	if err != nil {
		return 0, fmt.Errorf("failed to list rules: %w", err)
	}

	// Extract priorities in the range
	usedPriorities := make(map[int32]bool)
	for _, rule := range rules {
		if rule.Priority >= priorityRange.Start && rule.Priority <= priorityRange.End {
			usedPriorities[rule.Priority] = true
		}
	}

	// Find first available priority
	for i := priorityRange.Start; i <= priorityRange.End; i++ {
		if !usedPriorities[i] {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no available priority in range %s (%d-%d)", priorityRange.Name, priorityRange.Start, priorityRange.End)
}

// ValidatePriority checks if a priority is valid and available
func (p *PriorityManager) ValidatePriority(ctx context.Context, listenerArn string, priority int32, excludeRuleArn string) error {
	if priority < 1 || priority >= 50000 {
		return fmt.Errorf("priority must be between 1 and 49999")
	}

	// Check if priority is already in use
	rules, err := p.store.ListRules(ctx, listenerArn)
	if err != nil {
		return fmt.Errorf("failed to list rules: %w", err)
	}

	for _, rule := range rules {
		if rule.Priority == priority && rule.ARN != excludeRuleArn {
			return fmt.Errorf("priority %d is already in use by rule %s", priority, rule.ARN)
		}
	}

	return nil
}

// SetRulePriorities updates priorities for multiple rules atomically
func (p *PriorityManager) SetRulePriorities(ctx context.Context, priorities []RulePriorityUpdate) error {
	// Validate all priorities first
	priorityMap := make(map[int32]string)
	for _, update := range priorities {
		if update.Priority < 1 || update.Priority >= 50000 {
			return fmt.Errorf("invalid priority %d for rule %s", update.Priority, update.RuleArn)
		}
		if existing, exists := priorityMap[update.Priority]; exists {
			return fmt.Errorf("duplicate priority %d for rules %s and %s", update.Priority, existing, update.RuleArn)
		}
		priorityMap[update.Priority] = update.RuleArn
	}

	// Update each rule
	for _, update := range priorities {
		rule, err := p.store.GetRule(ctx, update.RuleArn)
		if err != nil {
			return fmt.Errorf("failed to get rule %s: %w", update.RuleArn, err)
		}

		rule.Priority = update.Priority
		if err := p.store.UpdateRule(ctx, rule); err != nil {
			return fmt.Errorf("failed to update rule %s: %w", update.RuleArn, err)
		}
	}

	return nil
}

// RulePriorityUpdate represents a priority update request
type RulePriorityUpdate struct {
	RuleArn  string
	Priority int32
}

// AnalyzeRulePriorities analyzes rule priority distribution
func (p *PriorityManager) AnalyzeRulePriorities(ctx context.Context, listenerArn string) (*PriorityAnalysis, error) {
	rules, err := p.store.ListRules(ctx, listenerArn)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	analysis := &PriorityAnalysis{
		TotalRules:     len(rules),
		PriorityRanges: make(map[string]int),
		Gaps:           []PriorityGap{},
		Conflicts:      []PriorityConflict{},
		UsedPriorities: make([]int32, 0, len(rules)),
	}

	// Sort rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	// Count rules in each range
	for _, rule := range rules {
		analysis.UsedPriorities = append(analysis.UsedPriorities, rule.Priority)

		switch {
		case rule.Priority >= PriorityRangeCritical.Start && rule.Priority <= PriorityRangeCritical.End:
			analysis.PriorityRanges["critical"]++
		case rule.Priority >= PriorityRangeSpecific.Start && rule.Priority <= PriorityRangeSpecific.End:
			analysis.PriorityRanges["specific"]++
		case rule.Priority >= PriorityRangeGeneral.Start && rule.Priority <= PriorityRangeGeneral.End:
			analysis.PriorityRanges["general"]++
		case rule.Priority >= PriorityRangeCatchAll.Start && rule.Priority <= PriorityRangeCatchAll.End:
			analysis.PriorityRanges["catchall"]++
		}
	}

	// Find gaps
	if len(rules) > 0 {
		for i := 0; i < len(rules)-1; i++ {
			gap := rules[i+1].Priority - rules[i].Priority
			if gap > 10 { // Significant gap
				analysis.Gaps = append(analysis.Gaps, PriorityGap{
					Start: rules[i].Priority + 1,
					End:   rules[i+1].Priority - 1,
					Size:  gap - 1,
				})
			}
		}
	}

	// Detect potential conflicts (rules with very close priorities)
	for i := 0; i < len(rules)-1; i++ {
		if rules[i+1].Priority-rules[i].Priority == 1 {
			analysis.Conflicts = append(analysis.Conflicts, PriorityConflict{
				Rule1:     rules[i].ARN,
				Rule2:     rules[i+1].ARN,
				Priority1: rules[i].Priority,
				Priority2: rules[i+1].Priority,
				Message:   "Adjacent priorities may cause maintenance issues",
			})
		}
	}

	return analysis, nil
}

// PriorityAnalysis contains analysis results
type PriorityAnalysis struct {
	TotalRules     int
	PriorityRanges map[string]int
	Gaps           []PriorityGap
	Conflicts      []PriorityConflict
	UsedPriorities []int32
}

// PriorityGap represents a gap in priority numbers
type PriorityGap struct {
	Start int32
	End   int32
	Size  int32
}

// PriorityConflict represents a potential priority conflict
type PriorityConflict struct {
	Rule1     string
	Rule2     string
	Priority1 int32
	Priority2 int32
	Message   string
}

// OptimizePriorities suggests optimized priority assignments
func (p *PriorityManager) OptimizePriorities(ctx context.Context, listenerArn string) ([]RulePriorityUpdate, error) {
	rules, err := p.store.ListRules(ctx, listenerArn)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	// Sort by current priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	suggestions := make([]RulePriorityUpdate, 0, len(rules))

	// Analyze rule conditions to determine specificity
	for i, rule := range rules {
		conditions, err := p.parseConditions(rule.Conditions)
		if err != nil {
			logging.Debug("Failed to parse conditions for rule", "ruleArn", rule.ARN, "error", err)
			continue
		}

		// Calculate suggested priority based on specificity
		specificity := p.calculateSpecificity(conditions)
		suggestedPriority := p.suggestPriorityFromSpecificity(specificity, i, len(rules))

		if suggestedPriority != rule.Priority {
			suggestions = append(suggestions, RulePriorityUpdate{
				RuleArn:  rule.ARN,
				Priority: suggestedPriority,
			})
		}
	}

	return suggestions, nil
}

// parseConditions parses rule conditions from JSON
func (p *PriorityManager) parseConditions(conditionsJSON string) ([]generated_elbv2.RuleCondition, error) {
	converter := NewRuleConverter()
	return converter.ConvertRuleConditionsFromJSON(conditionsJSON)
}

// calculateSpecificity calculates a specificity score for rule conditions
func (p *PriorityManager) calculateSpecificity(conditions []generated_elbv2.RuleCondition) int {
	score := 0

	for _, condition := range conditions {
		// Exact matches are more specific
		if condition.PathPatternConfig != nil {
			for _, pattern := range condition.PathPatternConfig.Values {
				if !strings.Contains(pattern, "*") {
					score += 15 // Exact path
				} else if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
					score += 5 // Path prefix
				} else {
					score += 3 // Complex pattern
				}
			}
		}

		if condition.HostHeaderConfig != nil {
			for _, host := range condition.HostHeaderConfig.Values {
				if !strings.Contains(host, "*") {
					score += 8 // Exact host
				} else {
					score += 4 // Wildcard host
				}
			}
		}

		if condition.HttpHeaderConfig != nil {
			score += 6 // HTTP header conditions are fairly specific
		}

		if condition.HttpRequestMethodConfig != nil {
			score += 2 // Method conditions are less specific
		}

		if condition.QueryStringConfig != nil {
			score += 7 // Query string conditions are specific
		}

		if condition.SourceIpConfig != nil {
			score += 5 // Source IP conditions
		}
	}

	// Multiple conditions make a rule more specific
	if len(conditions) > 1 {
		score += len(conditions) * 2
	}

	return score
}

// suggestPriorityFromSpecificity suggests a priority based on specificity score
func (p *PriorityManager) suggestPriorityFromSpecificity(specificity int, index int, totalRules int) int32 {
	// Map specificity to priority ranges
	if specificity >= 15 {
		// Very specific - use specific range
		basePriority := int32(100)
		increment := int32(900 / totalRules)
		if increment < 1 {
			increment = 1
		}
		return basePriority + int32(index)*increment
	} else if specificity >= 5 {
		// Moderately specific - use general range
		basePriority := int32(1000)
		increment := int32(8000 / totalRules)
		if increment < 1 {
			increment = 1
		}
		return basePriority + int32(index)*increment
	} else {
		// Not very specific - use catch-all range
		basePriority := int32(10000)
		increment := int32(30000 / totalRules)
		if increment < 1 {
			increment = 1
		}
		return basePriority + int32(index)*increment
	}
}

// ReorderRulesForClarity reorders rules to improve clarity and maintainability
func (p *PriorityManager) ReorderRulesForClarity(ctx context.Context, listenerArn string, gapSize int32) ([]RulePriorityUpdate, error) {
	rules, err := p.store.ListRules(ctx, listenerArn)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	// Sort by current priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})

	updates := make([]RulePriorityUpdate, 0, len(rules))
	currentPriority := int32(10) // Start from 10

	for _, rule := range rules {
		// Skip default rule
		if rule.Priority >= 50000 {
			continue
		}

		updates = append(updates, RulePriorityUpdate{
			RuleArn:  rule.ARN,
			Priority: currentPriority,
		})

		currentPriority += gapSize
	}

	return updates, nil
}

// FindPriorityForConditions suggests an appropriate priority for given conditions
func (p *PriorityManager) FindPriorityForConditions(ctx context.Context, listenerArn string, conditions []generated_elbv2.RuleCondition) (int32, error) {
	// Calculate specificity
	specificity := p.calculateSpecificity(conditions)

	// Determine appropriate range
	var targetRange PriorityRange
	if specificity >= 15 {
		targetRange = PriorityRangeSpecific
	} else if specificity >= 5 {
		targetRange = PriorityRangeGeneral
	} else {
		targetRange = PriorityRangeCatchAll
	}

	// Find available priority in range
	return p.GetNextAvailablePriority(ctx, listenerArn, targetRange)
}
