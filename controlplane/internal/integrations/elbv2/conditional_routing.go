package elbv2

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// ConditionalRoutingManager manages complex conditional routing patterns
type ConditionalRoutingManager struct {
	store           storage.ELBv2Store
	priorityManager *PriorityManager
	ruleConverter   *RuleConverter
}

// NewConditionalRoutingManager creates a new conditional routing manager
func NewConditionalRoutingManager(store storage.ELBv2Store) *ConditionalRoutingManager {
	return &ConditionalRoutingManager{
		store:           store,
		priorityManager: NewPriorityManager(store),
		ruleConverter:   NewRuleConverter(),
	}
}

// ConditionalRoute represents a conditional routing configuration
type ConditionalRoute struct {
	Name        string
	Description string
	Conditions  []ConditionalGroup
	Actions     []generated_elbv2.Action
	Priority    *int32 // Optional, will be auto-assigned if nil
}

// ConditionalGroup represents a group of conditions that must all match
type ConditionalGroup struct {
	Operator   string                          // "AND" or "OR"
	Conditions []generated_elbv2.RuleCondition
}

// CreateConditionalRoute creates a rule with complex conditional logic
func (c *ConditionalRoutingManager) CreateConditionalRoute(ctx context.Context, listenerArn string, route ConditionalRoute) (*storage.ELBv2Rule, error) {
	// Flatten conditions for ELBv2 (all conditions in a rule are AND'ed)
	flatConditions := c.flattenConditions(route.Conditions)
	
	// Determine priority if not specified
	if route.Priority == nil {
		priority, err := c.priorityManager.FindPriorityForConditions(ctx, listenerArn, flatConditions)
		if err != nil {
			return nil, fmt.Errorf("failed to determine priority: %w", err)
		}
		route.Priority = &priority
	} else {
		// Validate specified priority
		err := c.priorityManager.ValidatePriority(ctx, listenerArn, *route.Priority, "")
		if err != nil {
			return nil, fmt.Errorf("invalid priority: %w", err)
		}
	}
	
	// Create the rule
	rule := &storage.ELBv2Rule{
		ARN:         generateRuleArn(listenerArn),
		ListenerArn: listenerArn,
		Priority:    *route.Priority,
		Conditions:  c.serializeConditions(flatConditions),
		Actions:     c.serializeActions(route.Actions),
	}
	
	// Add metadata as tags
	if route.Name != "" || route.Description != "" {
		rule.Tags = map[string]string{
			"Name":        route.Name,
			"Description": route.Description,
		}
	}
	
	err := c.store.CreateRule(ctx, rule)
	if err != nil {
		return nil, err
	}
	
	logging.Debug("Created conditional route", "name", route.Name, "priority", *route.Priority)
	return rule, nil
}

// CreateIfThenElseRoutes creates a set of if-then-else routing rules
func (c *ConditionalRoutingManager) CreateIfThenElseRoutes(ctx context.Context, listenerArn string, routes []ConditionalRoute) ([]*storage.ELBv2Rule, error) {
	// Sort routes by specificity to assign appropriate priorities
	sort.Slice(routes, func(i, j int) bool {
		scoreI := c.calculateRouteSpecificity(routes[i])
		scoreJ := c.calculateRouteSpecificity(routes[j])
		return scoreI > scoreJ
	})
	
	var rules []*storage.ELBv2Rule
	
	for i, route := range routes {
		// Auto-assign priorities in order
		if route.Priority == nil {
			// Start from a base priority with gaps
			basePriority := int32(100 + i*10)
			route.Priority = &basePriority
		}
		
		rule, err := c.CreateConditionalRoute(ctx, listenerArn, route)
		if err != nil {
			// Rollback created rules on error
			for _, createdRule := range rules {
				c.store.DeleteRule(ctx, createdRule.ARN)
			}
			return nil, fmt.Errorf("failed to create route %s: %w", route.Name, err)
		}
		
		rules = append(rules, rule)
	}
	
	return rules, nil
}

// CreateCanaryRoute creates a canary deployment route with conditions
func (c *ConditionalRoutingManager) CreateCanaryRoute(ctx context.Context, listenerArn string, config CanaryConfig) (*storage.ELBv2Rule, error) {
	// Build conditions for canary
	var conditions []generated_elbv2.RuleCondition
	
	// Add path condition if specified
	if len(config.Paths) > 0 {
		conditions = append(conditions, generated_elbv2.RuleCondition{
			Field: strPtr("path-pattern"),
			PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
				Values: config.Paths,
			},
		})
	}
	
	// Add header conditions for canary selection
	if config.HeaderName != "" && len(config.HeaderValues) > 0 {
		conditions = append(conditions, generated_elbv2.RuleCondition{
			Field: strPtr("http-header"),
			HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
				HttpHeaderName: &config.HeaderName,
				Values:         config.HeaderValues,
			},
		})
	}
	
	// Create weighted forward action
	actionType := generated_elbv2.ActionTypeEnumFORWARD
	action := generated_elbv2.Action{
		Type: actionType,
		ForwardConfig: &generated_elbv2.ForwardActionConfig{
			TargetGroups: []generated_elbv2.TargetGroupTuple{
				{
					TargetGroupArn: &config.CanaryTargetGroup,
					Weight:         &config.CanaryWeight,
				},
				{
					TargetGroupArn: &config.StableTargetGroup,
					Weight:         &config.StableWeight,
				},
			},
		},
	}
	
	route := ConditionalRoute{
		Name:        fmt.Sprintf("canary-%s", config.Name),
		Description: fmt.Sprintf("Canary deployment: %d%% to new version", config.CanaryWeight),
		Conditions: []ConditionalGroup{
			{
				Operator:   "AND",
				Conditions: conditions,
			},
		},
		Actions:  []generated_elbv2.Action{action},
		Priority: config.Priority,
	}
	
	return c.CreateConditionalRoute(ctx, listenerArn, route)
}

// CanaryConfig represents canary deployment configuration
type CanaryConfig struct {
	Name              string
	Paths             []string
	HeaderName        string
	HeaderValues      []string
	CanaryTargetGroup string
	StableTargetGroup string
	CanaryWeight      int32
	StableWeight      int32
	Priority          *int32
}

// CreateMultiStageRoute creates a multi-stage feature rollout
func (c *ConditionalRoutingManager) CreateMultiStageRoute(ctx context.Context, listenerArn string, stages []StageConfig) ([]*storage.ELBv2Rule, error) {
	var routes []ConditionalRoute
	
	for _, stage := range stages {
		conditions := []generated_elbv2.RuleCondition{}
		
		// Add path conditions
		if len(stage.Paths) > 0 {
			conditions = append(conditions, generated_elbv2.RuleCondition{
				Field: strPtr("path-pattern"),
				PathPatternConfig: &generated_elbv2.PathPatternConditionConfig{
					Values: stage.Paths,
				},
			})
		}
		
		// Add source IP conditions for internal users
		if len(stage.SourceIPs) > 0 {
			conditions = append(conditions, generated_elbv2.RuleCondition{
				Field: strPtr("source-ip"),
				SourceIpConfig: &generated_elbv2.SourceIpConditionConfig{
					Values: stage.SourceIPs,
				},
			})
		}
		
		// Add header conditions
		for headerName, headerValues := range stage.Headers {
			conditions = append(conditions, generated_elbv2.RuleCondition{
				Field: strPtr("http-header"),
				HttpHeaderConfig: &generated_elbv2.HttpHeaderConditionConfig{
					HttpHeaderName: strPtr(headerName),
					Values:         headerValues,
				},
			})
		}
		
		actionType := generated_elbv2.ActionTypeEnumFORWARD
		action := generated_elbv2.Action{
			Type:           actionType,
			TargetGroupArn: &stage.TargetGroup,
		}
		
		routes = append(routes, ConditionalRoute{
			Name:        stage.Name,
			Description: stage.Description,
			Conditions: []ConditionalGroup{
				{
					Operator:   "AND",
					Conditions: conditions,
				},
			},
			Actions:  []generated_elbv2.Action{action},
			Priority: stage.Priority,
		})
	}
	
	return c.CreateIfThenElseRoutes(ctx, listenerArn, routes)
}

// StageConfig represents a stage in multi-stage rollout
type StageConfig struct {
	Name        string
	Description string
	Paths       []string
	SourceIPs   []string
	Headers     map[string][]string
	TargetGroup string
	Priority    *int32
}

// AnalyzeConditionalRoutes analyzes the effectiveness of conditional routes
func (c *ConditionalRoutingManager) AnalyzeConditionalRoutes(ctx context.Context, listenerArn string) (*ConditionalRoutingAnalysis, error) {
	rules, err := c.store.ListRules(ctx, listenerArn)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	
	analysis := &ConditionalRoutingAnalysis{
		TotalRules:          len(rules),
		ConditionalRules:    0,
		ComplexityScore:     0,
		ConditionTypes:      make(map[string]int),
		AverageConditions:   0,
		PotentialConflicts:  []ConflictInfo{},
		OptimizationTips:    []string{},
	}
	
	totalConditions := 0
	
	for _, rule := range rules {
		conditions, err := c.ruleConverter.ConvertRuleConditionsFromJSON(rule.Conditions)
		if err != nil {
			logging.Debug("Failed to parse conditions for rule", "ruleArn", rule.ARN, "error", err)
			continue
		}
		
		if len(conditions) > 1 {
			analysis.ConditionalRules++
		}
		
		totalConditions += len(conditions)
		
		// Count condition types
		for _, condition := range conditions {
			if condition.Field != nil {
				analysis.ConditionTypes[*condition.Field]++
			}
		}
		
		// Calculate complexity score
		analysis.ComplexityScore += len(conditions) * len(conditions)
	}
	
	if len(rules) > 0 {
		analysis.AverageConditions = float32(totalConditions) / float32(len(rules))
	}
	
	// Detect potential conflicts
	analysis.PotentialConflicts = c.detectConflicts(rules)
	
	// Generate optimization tips
	analysis.OptimizationTips = c.generateOptimizationTips(analysis)
	
	return analysis, nil
}

// ConditionalRoutingAnalysis contains analysis results
type ConditionalRoutingAnalysis struct {
	TotalRules         int
	ConditionalRules   int
	ComplexityScore    int
	ConditionTypes     map[string]int
	AverageConditions  float32
	PotentialConflicts []ConflictInfo
	OptimizationTips   []string
}

// ConflictInfo represents a potential routing conflict
type ConflictInfo struct {
	Rule1       string
	Rule2       string
	Description string
}

// Helper functions

func (c *ConditionalRoutingManager) flattenConditions(groups []ConditionalGroup) []generated_elbv2.RuleCondition {
	// For now, we only support AND operations (ELBv2 limitation)
	// OR operations would require creating multiple rules
	var conditions []generated_elbv2.RuleCondition
	
	for _, group := range groups {
		if group.Operator == "AND" || group.Operator == "" {
			conditions = append(conditions, group.Conditions...)
		} else if group.Operator == "OR" {
			logging.Debug("OR operations require multiple rules; only using first condition")
			if len(group.Conditions) > 0 {
				conditions = append(conditions, group.Conditions[0])
			}
		}
	}
	
	return conditions
}

func (c *ConditionalRoutingManager) calculateRouteSpecificity(route ConditionalRoute) int {
	conditions := c.flattenConditions(route.Conditions)
	return c.priorityManager.calculateSpecificity(conditions)
}

func (c *ConditionalRoutingManager) detectConflicts(rules []*storage.ELBv2Rule) []ConflictInfo {
	var conflicts []ConflictInfo
	
	// Sort rules by priority
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})
	
	// Check for overlapping conditions
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if c.rulesOverlap(rules[i], rules[j]) {
				conflicts = append(conflicts, ConflictInfo{
					Rule1:       rules[i].ARN,
					Rule2:       rules[j].ARN,
					Description: fmt.Sprintf("Rules have overlapping conditions; rule with priority %d will always match first", rules[i].Priority),
				})
			}
		}
	}
	
	return conflicts
}

func (c *ConditionalRoutingManager) rulesOverlap(rule1, rule2 *storage.ELBv2Rule) bool {
	// Simple overlap detection - could be enhanced
	conditions1, _ := c.ruleConverter.ConvertRuleConditionsFromJSON(rule1.Conditions)
	conditions2, _ := c.ruleConverter.ConvertRuleConditionsFromJSON(rule2.Conditions)
	
	// Check if rule1's conditions are a subset of rule2's conditions
	pathMatch := false
	for _, c1 := range conditions1 {
		if c1.PathPatternConfig != nil {
			for _, c2 := range conditions2 {
				if c2.PathPatternConfig != nil {
					// Check for path overlap
					for _, p1 := range c1.PathPatternConfig.Values {
						for _, p2 := range c2.PathPatternConfig.Values {
							if pathsOverlap(p1, p2) {
								pathMatch = true
							}
						}
					}
				}
			}
		}
	}
	
	return pathMatch
}

func pathsOverlap(path1, path2 string) bool {
	// Simple overlap check
	if path1 == path2 {
		return true
	}
	
	// Check if one is a prefix of the other
	if strings.HasSuffix(path1, "*") {
		prefix := strings.TrimSuffix(path1, "*")
		return strings.HasPrefix(path2, prefix)
	}
	
	if strings.HasSuffix(path2, "*") {
		prefix := strings.TrimSuffix(path2, "*")
		return strings.HasPrefix(path1, prefix)
	}
	
	return false
}

func (c *ConditionalRoutingManager) generateOptimizationTips(analysis *ConditionalRoutingAnalysis) []string {
	var tips []string
	
	if analysis.AverageConditions > 3 {
		tips = append(tips, "Consider simplifying rules with many conditions for better performance")
	}
	
	if analysis.ComplexityScore > 100 {
		tips = append(tips, "High complexity score indicates potential for rule consolidation")
	}
	
	if len(analysis.PotentialConflicts) > 0 {
		tips = append(tips, fmt.Sprintf("Found %d potential conflicts that may cause unexpected routing", len(analysis.PotentialConflicts)))
	}
	
	if headerCount, exists := analysis.ConditionTypes["http-header"]; exists && headerCount > 10 {
		tips = append(tips, "Consider using fewer header conditions for better performance")
	}
	
	return tips
}

func generateRuleArn(listenerArn string) string {
	// Generate a unique rule ARN
	parts := strings.Split(listenerArn, "/")
	if len(parts) >= 4 {
		ruleID := generateConditionalID()
		return fmt.Sprintf("%s/rule/%s", strings.Join(parts[:len(parts)-1], "/"), ruleID)
	}
	return fmt.Sprintf("arn:aws:elasticloadbalancing:us-east-1:123456789012:rule/app/lb/id/listener/rule/%s", generateConditionalID())
}

func generateConditionalID() string {
	// Simple ID generation - in production, use a proper UUID
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func strPtr(s string) *string {
	return &s
}

// serializeConditions converts conditions to JSON string
func (c *ConditionalRoutingManager) serializeConditions(conditions []generated_elbv2.RuleCondition) string {
	data, err := json.Marshal(conditions)
	if err != nil {
		logging.Debug("Failed to serialize conditions", "error", err)
		return "[]"
	}
	return string(data)
}

// serializeActions converts actions to JSON string
func (c *ConditionalRoutingManager) serializeActions(actions []generated_elbv2.Action) string {
	data, err := json.Marshal(actions)
	if err != nil {
		logging.Debug("Failed to serialize actions", "error", err)
		return "[]"
	}
	return string(data)
}