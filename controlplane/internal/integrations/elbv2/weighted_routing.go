package elbv2

import (
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"k8s.io/klog/v2"
)

// WeightedRoutingManager handles weighted routing configuration for ELBv2
type WeightedRoutingManager struct {
	converter *RuleConverter
}

// NewWeightedRoutingManager creates a new weighted routing manager
func NewWeightedRoutingManager() *WeightedRoutingManager {
	return &WeightedRoutingManager{
		converter: NewRuleConverter(),
	}
}

// TraefikWeightedService represents a weighted service in Traefik
type TraefikWeightedService struct {
	Name   string
	Port   int32
	Weight int32
	Sticky *TraefikSticky `json:"sticky,omitempty"`
}

// TraefikSticky represents sticky session configuration
type TraefikSticky struct {
	Cookie *TraefikCookie `json:"cookie,omitempty"`
}

// TraefikCookie represents cookie configuration for sticky sessions
type TraefikCookie struct {
	Name     string `json:"name"`
	Secure   bool   `json:"secure"`
	HTTPOnly bool   `json:"httpOnly"`
	SameSite string `json:"sameSite"`
}

// ConvertActionsToWeightedServices converts ELBv2 forward actions to Traefik weighted services
func (w *WeightedRoutingManager) ConvertActionsToWeightedServices(actions []generated_elbv2.Action, targetGroupResolver TargetGroupResolver) ([]TraefikWeightedService, error) {
	var services []TraefikWeightedService

	for _, action := range actions {
		if action.Type == generated_elbv2.ActionTypeEnumFORWARD {
			// Simple forward action
			if action.TargetGroupArn != nil {
				service, err := w.createServiceFromTargetGroup(*action.TargetGroupArn, 100, targetGroupResolver)
				if err != nil {
					return nil, err
				}
				services = append(services, service)
			} else if action.ForwardConfig != nil {
				// Weighted forward config
				weightedServices, err := w.convertForwardConfig(action.ForwardConfig, targetGroupResolver)
				if err != nil {
					return nil, err
				}
				services = append(services, weightedServices...)
			}
		}
	}

	return services, nil
}

// convertForwardConfig converts ELBv2 ForwardConfig to Traefik weighted services
func (w *WeightedRoutingManager) convertForwardConfig(config *generated_elbv2.ForwardActionConfig, resolver TargetGroupResolver) ([]TraefikWeightedService, error) {
	var services []TraefikWeightedService
	
	// Validate total weight
	totalWeight := int32(0)
	for _, tg := range config.TargetGroups {
		if tg.Weight != nil {
			totalWeight += *tg.Weight
		}
	}
	
	if totalWeight == 0 {
		klog.V(2).Info("Total weight is 0, normalizing weights")
		// If all weights are 0, distribute equally
		equalWeight := int32(100 / len(config.TargetGroups))
		for i := range config.TargetGroups {
			config.TargetGroups[i].Weight = &equalWeight
		}
	}

	// Create weighted services
	for _, tg := range config.TargetGroups {
		if tg.TargetGroupArn == nil {
			continue
		}

		weight := int32(1) // Default weight
		if tg.Weight != nil {
			weight = *tg.Weight
		}

		service, err := w.createServiceFromTargetGroup(*tg.TargetGroupArn, weight, resolver)
		if err != nil {
			klog.V(2).Infof("Failed to create service for target group %s: %v", *tg.TargetGroupArn, err)
			continue
		}

		// Add sticky session if configured
		if config.TargetGroupStickinessConfig != nil && config.TargetGroupStickinessConfig.Enabled != nil && *config.TargetGroupStickinessConfig.Enabled {
			service.Sticky = w.createStickyConfig(config.TargetGroupStickinessConfig)
		}

		services = append(services, service)
	}

	return services, nil
}

// createServiceFromTargetGroup creates a Traefik service from a target group ARN
func (w *WeightedRoutingManager) createServiceFromTargetGroup(targetGroupArn string, weight int32, resolver TargetGroupResolver) (TraefikWeightedService, error) {
	// Extract target group name from ARN
	tgName := extractResourceNameFromArn(targetGroupArn, "targetgroup")
	if tgName == "" {
		return TraefikWeightedService{}, fmt.Errorf("invalid target group ARN: %s", targetGroupArn)
	}

	// Resolve target group details if resolver is provided
	port := int32(80) // Default port
	if resolver != nil {
		tgInfo, err := resolver.GetTargetGroupInfo(targetGroupArn)
		if err != nil {
			klog.V(2).Infof("Failed to resolve target group %s: %v", targetGroupArn, err)
		} else if tgInfo.Port != nil {
			port = *tgInfo.Port
		}
	}

	return TraefikWeightedService{
		Name:   fmt.Sprintf("tg-%s", tgName),
		Port:   port,
		Weight: weight,
	}, nil
}

// createStickyConfig creates sticky session configuration
func (w *WeightedRoutingManager) createStickyConfig(config *generated_elbv2.TargetGroupStickinessConfig) *TraefikSticky {
	cookieName := "kecs-sticky"
	
	// Use duration to generate a unique cookie name if needed
	if config.DurationSeconds != nil {
		cookieName = fmt.Sprintf("kecs-sticky-%d", *config.DurationSeconds)
	}

	return &TraefikSticky{
		Cookie: &TraefikCookie{
			Name:     cookieName,
			Secure:   true,
			HTTPOnly: true,
			SameSite: "lax",
		},
	}
}

// TargetGroupResolver interface for resolving target group information
type TargetGroupResolver interface {
	GetTargetGroupInfo(arn string) (*generated_elbv2.TargetGroup, error)
}

// ValidateWeightedRouting validates weighted routing configuration
func (w *WeightedRoutingManager) ValidateWeightedRouting(actions []generated_elbv2.Action) error {
	for _, action := range actions {
		if action.Type == generated_elbv2.ActionTypeEnumFORWARD && action.ForwardConfig != nil {
			if err := w.validateForwardConfig(action.ForwardConfig); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateForwardConfig validates ForwardConfig
func (w *WeightedRoutingManager) validateForwardConfig(config *generated_elbv2.ForwardActionConfig) error {
	if len(config.TargetGroups) == 0 {
		return fmt.Errorf("ForwardConfig must have at least one target group")
	}

	if len(config.TargetGroups) > 5 {
		return fmt.Errorf("ForwardConfig cannot have more than 5 target groups")
	}

	// Validate weights
	totalWeight := int32(0)
	for _, tg := range config.TargetGroups {
		if tg.TargetGroupArn == nil {
			return fmt.Errorf("target group ARN is required")
		}
		
		if tg.Weight != nil {
			if *tg.Weight < 0 || *tg.Weight > 999 {
				return fmt.Errorf("weight must be between 0 and 999")
			}
			totalWeight += *tg.Weight
		}
	}

	// AWS allows total weight to be 0 (will be normalized)
	// But warn if weights don't add up to 100 for clarity
	if totalWeight > 0 && totalWeight != 100 {
		klog.V(2).Infof("Total weight is %d, not 100. Traefik will normalize weights", totalWeight)
	}

	return nil
}

// NormalizeWeights normalizes weights to ensure they sum to 100
func (w *WeightedRoutingManager) NormalizeWeights(services []TraefikWeightedService) []TraefikWeightedService {
	if len(services) == 0 {
		return services
	}

	// Calculate total weight
	totalWeight := int32(0)
	for _, service := range services {
		totalWeight += service.Weight
	}

	if totalWeight == 0 {
		// Distribute equally
		equalWeight := int32(100 / len(services))
		for i := range services {
			services[i].Weight = equalWeight
		}
		// Add remainder to first service
		if remainder := int32(100 % len(services)); remainder > 0 {
			services[0].Weight += remainder
		}
	} else if totalWeight != 100 {
		// Normalize to 100
		for i := range services {
			services[i].Weight = (services[i].Weight * 100) / totalWeight
		}
		
		// Adjust for rounding errors
		currentTotal := int32(0)
		for _, service := range services {
			currentTotal += service.Weight
		}
		if diff := 100 - currentTotal; diff != 0 {
			services[0].Weight += diff
		}
	}

	return services
}

// CalculateWeightDistribution calculates expected request distribution
func (w *WeightedRoutingManager) CalculateWeightDistribution(services []TraefikWeightedService, totalRequests int) map[string]int {
	distribution := make(map[string]int)
	
	normalizedServices := w.NormalizeWeights(services)
	
	for _, service := range normalizedServices {
		expectedRequests := (totalRequests * int(service.Weight)) / 100
		distribution[service.Name] = expectedRequests
	}
	
	// Adjust for rounding
	totalDistributed := 0
	for _, count := range distribution {
		totalDistributed += count
	}
	
	if diff := totalRequests - totalDistributed; diff > 0 && len(services) > 0 {
		distribution[services[0].Name] += diff
	}
	
	return distribution
}

// extractResourceNameFromArn extracts resource name from ARN
func extractResourceNameFromArn(arn string, resourceType string) string {
	// ARN format: arn:aws:elasticloadbalancing:region:account:targetgroup/name/id
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return ""
	}
	
	resourcePart := parts[5]
	resourceParts := strings.Split(resourcePart, "/")
	if len(resourceParts) < 2 {
		return ""
	}
	
	if resourceParts[0] != resourceType {
		return ""
	}
	
	return resourceParts[1]
}

// GenerateTraefikServiceYAML generates YAML representation of weighted services
func (w *WeightedRoutingManager) GenerateTraefikServiceYAML(services []TraefikWeightedService) string {
	if len(services) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, "services:")
	
	for _, service := range services {
		lines = append(lines, fmt.Sprintf("  - name: %s", service.Name))
		lines = append(lines, fmt.Sprintf("    port: %d", service.Port))
		if service.Weight != 100 || len(services) > 1 {
			lines = append(lines, fmt.Sprintf("    weight: %d", service.Weight))
		}
		
		if service.Sticky != nil && service.Sticky.Cookie != nil {
			lines = append(lines, "    sticky:")
			lines = append(lines, "      cookie:")
			lines = append(lines, fmt.Sprintf("        name: %s", service.Sticky.Cookie.Name))
			lines = append(lines, fmt.Sprintf("        secure: %t", service.Sticky.Cookie.Secure))
			lines = append(lines, fmt.Sprintf("        httpOnly: %t", service.Sticky.Cookie.HTTPOnly))
			lines = append(lines, fmt.Sprintf("        sameSite: %s", service.Sticky.Cookie.SameSite))
		}
	}
	
	return strings.Join(lines, "\n")
}