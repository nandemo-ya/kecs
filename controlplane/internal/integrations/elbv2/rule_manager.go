package elbv2

import (
	"context"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"
)

// RuleManager manages the synchronization between ELBv2 rules and Traefik IngressRoute rules
type RuleManager struct {
	dynamicClient dynamic.Interface
	converter     *RuleConverter
}

// NewRuleManager creates a new rule manager
func NewRuleManager(dynamicClient dynamic.Interface) *RuleManager {
	return &RuleManager{
		dynamicClient: dynamicClient,
		converter:     NewRuleConverter(),
	}
}

// SyncRulesForListener synchronizes all rules for a listener to Traefik IngressRoute
func (r *RuleManager) SyncRulesForListener(ctx context.Context, storage storage.Storage, listenerArn string, lbName string, port int32) error {
	if r.dynamicClient == nil {
		klog.V(2).Infof("No dynamicClient available, skipping rule sync")
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

	klog.V(2).Infof("Successfully synced %d rules for listener %s", len(rules), listenerArn)
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
			klog.Errorf("Failed to convert rule %s: %v", rule.ARN, err)
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

	// Extract target group
	targetGroupArn, err := r.converter.ExtractTargetGroupFromActions(actions)
	if err != nil {
		// Rule doesn't have a forward action, skip it
		klog.V(2).Infof("Rule %s has no forward action, skipping", rule.ARN)
		return nil, nil
	}

	// Get target group details
	targetGroup, err := storageInstance.ELBv2Store().GetTargetGroup(ctx, targetGroupArn)
	if err != nil {
		return nil, fmt.Errorf("failed to get target group %s: %w", targetGroupArn, err)
	}

	// Extract target group name from ARN
	targetGroupName := extractNameFromArn(targetGroupArn, "targetgroup")

	// Build Traefik route
	route := map[string]interface{}{
		"match":    match,
		"kind":     "Rule",
		"priority": int(rule.Priority),
		"services": []interface{}{
			map[string]interface{}{
				"name": fmt.Sprintf("tg-%s", targetGroupName),
				"port": targetGroup.Port,
			},
		},
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