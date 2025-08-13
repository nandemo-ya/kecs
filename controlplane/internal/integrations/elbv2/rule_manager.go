package elbv2

import (
	"context"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// RuleManager manages the synchronization between ELBv2 rules and Traefik IngressRoute rules
type RuleManager struct {
	dynamicClient    dynamic.Interface
	converter        *RuleConverter
	weightedManager  *WeightedRoutingManager
	priorityManager  *PriorityManager
	conditionalManager *ConditionalRoutingManager
}

// NewRuleManager creates a new rule manager
func NewRuleManager(dynamicClient dynamic.Interface, store storage.ELBv2Store) *RuleManager {
	return &RuleManager{
		dynamicClient:    dynamicClient,
		converter:        NewRuleConverter(),
		weightedManager:  NewWeightedRoutingManager(),
		priorityManager:  NewPriorityManager(store),
		conditionalManager: NewConditionalRoutingManager(store),
	}
}

// SyncRulesForListener synchronizes all rules for a listener to Traefik IngressRoute
func (r *RuleManager) SyncRulesForListener(ctx context.Context, storage storage.Storage, listenerArn string, lbName string, port int32) error {
	if r.dynamicClient == nil {
		logging.Debug("No dynamicClient available, skipping rule sync")
		return nil
	}

	// Get all rules for this listener
	rules, err := storage.ELBv2Store().ListRules(ctx, listenerArn)
	if err != nil {
		return fmt.Errorf("failed to list rules for listener %s: %w", listenerArn, err)
	}

	// Get the IngressRoute name
	namespace := "kecs-system"
	ingressRouteName := fmt.Sprintf("listener-%s-%d", sanitizeName(lbName), port)

	// Define the GVR for IngressRoute
	gvr := schema.GroupVersionResource{
		Group:    "traefik.io",
		Version:  "v1alpha1",
		Resource: "ingressroutes",
	}

	// Get existing IngressRoute
	existingRoute, err := r.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, ingressRouteName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get IngressRoute %s: %w", ingressRouteName, err)
	}

	// Convert rules to Traefik routes
	routes, err := r.convertRulesToRoutes(rules, storage, ctx)
	if err != nil {
		return fmt.Errorf("failed to convert rules to routes: %w", err)
	}

	// Update the IngressRoute with all routes
	spec, ok := existingRoute.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		existingRoute.Object["spec"] = spec
	}
	spec["routes"] = routes

	// Update the IngressRoute
	_, err = r.dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, existingRoute, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update IngressRoute: %w", err)
	}

	logging.Debug("Successfully synced rules for listener", "ruleCount", len(rules), "listenerArn", listenerArn)
	return nil
}

// AddRuleToListener adds a new rule to the listener's IngressRoute
func (r *RuleManager) AddRuleToListener(ctx context.Context, storage storage.Storage, ruleArn string, listenerArn string, lbName string, port int32) error {
	// Simply trigger a full sync - this ensures proper priority ordering
	return r.SyncRulesForListener(ctx, storage, listenerArn, lbName, port)
}

// RemoveRuleFromListener removes a rule from the listener's IngressRoute
func (r *RuleManager) RemoveRuleFromListener(ctx context.Context, storage storage.Storage, ruleArn string, listenerArn string, lbName string, port int32) error {
	// Simply trigger a full sync - this ensures proper priority ordering
	return r.SyncRulesForListener(ctx, storage, listenerArn, lbName, port)
}

// convertRulesToRoutes converts ELBv2 rules to Traefik routes
func (r *RuleManager) convertRulesToRoutes(rules []*storage.ELBv2Rule, storageInstance storage.Storage, ctx context.Context) ([]interface{}, error) {
	var routes []interface{}

	// Sort rules by priority (lower number = higher priority)
	// In Traefik, we'll map this to route order (first match wins)
	sortedRules := make([]*storage.ELBv2Rule, len(rules))
	copy(sortedRules, rules)
	
	// Simple bubble sort for now
	for i := 0; i < len(sortedRules)-1; i++ {
		for j := 0; j < len(sortedRules)-i-1; j++ {
			if sortedRules[j].Priority > sortedRules[j+1].Priority {
				sortedRules[j], sortedRules[j+1] = sortedRules[j+1], sortedRules[j]
			}
		}
	}

	// Convert each rule to a route
	for _, rule := range sortedRules {
		route, err := r.convertRuleToRoute(rule, storageInstance, ctx)
		if err != nil {
			logging.Error("Failed to convert rule", "ruleArn", rule.ARN, "error", err)
			continue
		}
		if route != nil {
			routes = append(routes, route)
		}
	}

	// Always add a default catch-all route at the end
	defaultRoute := map[string]interface{}{
		"match":    "PathPrefix(`/`)",
		"kind":     "Rule",
		"priority": 99999, // Very low priority
		"services": []interface{}{
			map[string]interface{}{
				"name": "default-backend",
				"port": 80,
			},
		},
	}
	routes = append(routes, defaultRoute)

	return routes, nil
}

// convertRuleToRoute converts a single ELBv2 rule to a Traefik route
func (r *RuleManager) convertRuleToRoute(rule *storage.ELBv2Rule, storageInstance storage.Storage, ctx context.Context) (map[string]interface{}, error) {
	// Parse conditions
	conditions, err := r.converter.ConvertRuleConditionsFromJSON(rule.Conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conditions: %w", err)
	}

	// Convert to Traefik match expression
	match, err := r.converter.ConvertRuleToTraefikMatch(conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to convert conditions to match: %w", err)
	}

	// Parse actions
	actions, err := r.converter.ConvertRuleActionsFromJSON(rule.Actions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}

	// Create target group resolver for weighted routing
	resolver := &storageTargetGroupResolver{
		store: storageInstance.ELBv2Store(),
		ctx:   ctx,
	}

	// Convert actions to weighted services
	services, err := r.weightedManager.ConvertActionsToWeightedServices(actions, resolver)
	if err != nil {
		logging.Debug("Failed to convert actions for rule", "ruleArn", rule.ARN, "error", err)
		return nil, nil
	}

	if len(services) == 0 {
		logging.Debug("Rule has no forward action, skipping", "ruleArn", rule.ARN)
		return nil, nil
	}

	// Convert weighted services to Traefik service format
	traefikServices := make([]interface{}, 0, len(services))
	for _, service := range services {
		svc := map[string]interface{}{
			"name": service.Name,
			"port": service.Port,
		}
		
		// Add weight if multiple services or weight is not 100
		if len(services) > 1 || service.Weight != 100 {
			svc["weight"] = service.Weight
		}
		
		// Add sticky configuration if present
		if service.Sticky != nil && service.Sticky.Cookie != nil {
			svc["sticky"] = map[string]interface{}{
				"cookie": map[string]interface{}{
					"name":     service.Sticky.Cookie.Name,
					"secure":   service.Sticky.Cookie.Secure,
					"httpOnly": service.Sticky.Cookie.HTTPOnly,
					"sameSite": service.Sticky.Cookie.SameSite,
				},
			}
		}
		
		traefikServices = append(traefikServices, svc)
	}

	// Build Traefik route
	route := map[string]interface{}{
		"match":    match,
		"kind":     "Rule",
		"priority": int(rule.Priority),
		"services": traefikServices,
	}

	// Add middleware for advanced features (future enhancement)
	// For now, we'll just add a comment
	if rule.Priority < 50000 { // Non-default rules
		if metadata, ok := route["metadata"].(map[string]interface{}); ok {
			metadata["comment"] = fmt.Sprintf("ELBv2 Rule %s (Priority: %d)", rule.ARN, rule.Priority)
		} else {
			route["metadata"] = map[string]interface{}{
				"comment": fmt.Sprintf("ELBv2 Rule %s (Priority: %d)", rule.ARN, rule.Priority),
			}
		}
	}

	return route, nil
}

// extractNameFromArn extracts the resource name from an ARN
func extractNameFromArn(arn string, resourceType string) string {
	// ARN format: arn:aws:elasticloadbalancing:region:account:resourcetype/resourcename/id
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return "unknown"
	}
	
	resourcePart := parts[5]
	resourceParts := strings.Split(resourcePart, "/")
	if len(resourceParts) >= 2 {
		return resourceParts[1]
	}
	
	return "unknown"
}

// GetListenerInfoFromArn extracts load balancer name and port from listener ARN
func GetListenerInfoFromArn(listenerArn string) (lbName string, port int32, err error) {
	// Listener ARN format: arn:aws:elasticloadbalancing:region:account:listener/app/lb-name/lb-id/listener-id
	parts := strings.Split(listenerArn, "/")
	if len(parts) < 4 {
		return "", 0, fmt.Errorf("invalid listener ARN format: %s", listenerArn)
	}

	lbName = parts[2]
	
	// For now, we'll need to look up the port from storage
	// In a real implementation, we'd query the listener details
	// TODO: Implement proper listener lookup
	port = 80 // Default port
	
	return lbName, port, nil
}

// storageTargetGroupResolver implements TargetGroupResolver using storage
type storageTargetGroupResolver struct {
	store storage.ELBv2Store
	ctx   context.Context
}

func (s *storageTargetGroupResolver) GetTargetGroupInfo(arn string) (*generated_elbv2.TargetGroup, error) {
	tg, err := s.store.GetTargetGroup(s.ctx, arn)
	if err != nil {
		return nil, err
	}
	
	// Convert storage TargetGroup to generated TargetGroup
	protocol := generated_elbv2.ProtocolEnum(tg.Protocol)
	return &generated_elbv2.TargetGroup{
		TargetGroupArn:  &tg.ARN,
		TargetGroupName: &tg.Name,
		Port:            &tg.Port,
		Protocol:        &protocol,
		VpcId:           &tg.VpcID,
	}, nil
}

// GetNextAvailablePriority finds the next available priority in a range for a listener
func (r *RuleManager) GetNextAvailablePriority(ctx context.Context, listenerArn string, priorityRange PriorityRange) (int32, error) {
	return r.priorityManager.GetNextAvailablePriority(ctx, listenerArn, priorityRange)
}

// ValidatePriority checks if a priority is valid and available for a listener
func (r *RuleManager) ValidatePriority(ctx context.Context, listenerArn string, priority int32, excludeRuleArn string) error {
	return r.priorityManager.ValidatePriority(ctx, listenerArn, priority, excludeRuleArn)
}

// SetRulePriorities updates priorities for multiple rules atomically
func (r *RuleManager) SetRulePriorities(ctx context.Context, priorities []RulePriorityUpdate) error {
	return r.priorityManager.SetRulePriorities(ctx, priorities)
}

// AnalyzeRulePriorities analyzes rule priority distribution for a listener
func (r *RuleManager) AnalyzeRulePriorities(ctx context.Context, listenerArn string) (*PriorityAnalysis, error) {
	return r.priorityManager.AnalyzeRulePriorities(ctx, listenerArn)
}

// OptimizePriorities suggests optimized priority assignments for rules
func (r *RuleManager) OptimizePriorities(ctx context.Context, listenerArn string) ([]RulePriorityUpdate, error) {
	return r.priorityManager.OptimizePriorities(ctx, listenerArn)
}

// ReorderRulesForClarity reorders rules to improve clarity and maintainability
func (r *RuleManager) ReorderRulesForClarity(ctx context.Context, listenerArn string, gapSize int32) ([]RulePriorityUpdate, error) {
	return r.priorityManager.ReorderRulesForClarity(ctx, listenerArn, gapSize)
}

// FindPriorityForConditions suggests an appropriate priority for given conditions
func (r *RuleManager) FindPriorityForConditions(ctx context.Context, listenerArn string, conditions []generated_elbv2.RuleCondition) (int32, error) {
	return r.priorityManager.FindPriorityForConditions(ctx, listenerArn, conditions)
}

// CreateConditionalRoute creates a rule with complex conditional logic
func (r *RuleManager) CreateConditionalRoute(ctx context.Context, listenerArn string, route ConditionalRoute) (*storage.ELBv2Rule, error) {
	return r.conditionalManager.CreateConditionalRoute(ctx, listenerArn, route)
}

// CreateIfThenElseRoutes creates a set of if-then-else routing rules
func (r *RuleManager) CreateIfThenElseRoutes(ctx context.Context, listenerArn string, routes []ConditionalRoute) ([]*storage.ELBv2Rule, error) {
	return r.conditionalManager.CreateIfThenElseRoutes(ctx, listenerArn, routes)
}

// CreateCanaryRoute creates a canary deployment route with conditions
func (r *RuleManager) CreateCanaryRoute(ctx context.Context, listenerArn string, config CanaryConfig) (*storage.ELBv2Rule, error) {
	return r.conditionalManager.CreateCanaryRoute(ctx, listenerArn, config)
}

// CreateMultiStageRoute creates a multi-stage feature rollout
func (r *RuleManager) CreateMultiStageRoute(ctx context.Context, listenerArn string, stages []StageConfig) ([]*storage.ELBv2Rule, error) {
	return r.conditionalManager.CreateMultiStageRoute(ctx, listenerArn, stages)
}

// AnalyzeConditionalRoutes analyzes the effectiveness of conditional routes
func (r *RuleManager) AnalyzeConditionalRoutes(ctx context.Context, listenerArn string) (*ConditionalRoutingAnalysis, error) {
	return r.conditionalManager.AnalyzeConditionalRoutes(ctx, listenerArn)
}