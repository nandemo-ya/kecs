package elbv2

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// RuleConverter handles conversion between ELBv2 rules and Traefik routing rules
type RuleConverter struct{}

// NewRuleConverter creates a new rule converter
func NewRuleConverter() *RuleConverter {
	return &RuleConverter{}
}

// ConvertRuleToTraefikMatch converts an ELBv2 rule condition to a Traefik match expression
func (c *RuleConverter) ConvertRuleToTraefikMatch(conditions []generated_elbv2.RuleCondition) (string, error) {
	if len(conditions) == 0 {
		// Default catch-all route
		return "PathPrefix(`/`)", nil
	}

	var matches []string

	for _, condition := range conditions {
		match, err := c.convertConditionToMatch(&condition)
		if err != nil {
			return "", fmt.Errorf("failed to convert condition: %w", err)
		}
		if match != "" {
			matches = append(matches, match)
		}
	}

	if len(matches) == 0 {
		return "PathPrefix(`/`)", nil
	}

	// Combine multiple conditions with AND operator
	if len(matches) > 1 {
		return fmt.Sprintf("(%s)", strings.Join(matches, " && ")), nil
	}

	return matches[0], nil
}

// convertConditionToMatch converts a single rule condition to a Traefik match expression
func (c *RuleConverter) convertConditionToMatch(condition *generated_elbv2.RuleCondition) (string, error) {
	// Handle new-style conditions with specific config types
	if condition.PathPatternConfig != nil && len(condition.PathPatternConfig.Values) > 0 {
		return c.convertPathPatternToMatch(condition.PathPatternConfig.Values), nil
	}

	if condition.HostHeaderConfig != nil && len(condition.HostHeaderConfig.Values) > 0 {
		return c.convertHostHeaderToMatch(condition.HostHeaderConfig.Values), nil
	}

	if condition.HttpHeaderConfig != nil {
		return c.convertHttpHeaderToMatch(condition.HttpHeaderConfig), nil
	}

	if condition.HttpRequestMethodConfig != nil && len(condition.HttpRequestMethodConfig.Values) > 0 {
		return c.convertHttpMethodToMatch(condition.HttpRequestMethodConfig.Values), nil
	}

	if condition.QueryStringConfig != nil && len(condition.QueryStringConfig.Values) > 0 {
		return c.convertQueryStringToMatch(condition.QueryStringConfig.Values), nil
	}

	if condition.SourceIpConfig != nil && len(condition.SourceIpConfig.Values) > 0 {
		return c.convertSourceIpToMatch(condition.SourceIpConfig.Values), nil
	}

	// Handle legacy style conditions with Field and Values
	if condition.Field != nil && len(condition.Values) > 0 {
		switch *condition.Field {
		case "path-pattern":
			return c.convertPathPatternToMatch(condition.Values), nil
		case "host-header":
			return c.convertHostHeaderToMatch(condition.Values), nil
		case "http-request-method":
			return c.convertHttpMethodToMatch(condition.Values), nil
		case "source-ip":
			return c.convertSourceIpToMatch(condition.Values), nil
		default:
			logging.Debug("Unsupported condition field", "field", *condition.Field)
			return "", nil
		}
	}

	return "", nil
}

// convertPathPatternToMatch converts path patterns to Traefik path matchers
func (c *RuleConverter) convertPathPatternToMatch(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}

	var pathMatches []string
	for _, pattern := range patterns {
		// Convert AWS path patterns to Traefik format
		if pattern == "*" || pattern == "/*" {
			pathMatches = append(pathMatches, "PathPrefix(`/`)")
		} else if strings.HasSuffix(pattern, "*") {
			// Remove trailing * and use PathPrefix
			prefix := strings.TrimSuffix(pattern, "*")
			pathMatches = append(pathMatches, fmt.Sprintf("PathPrefix(`%s`)", prefix))
		} else if strings.Contains(pattern, "*") {
			// Complex pattern with wildcards - use PathRegexp
			// Convert * to .* for regex
			regexPattern := strings.ReplaceAll(pattern, "*", ".*")
			pathMatches = append(pathMatches, fmt.Sprintf("PathRegexp(`^%s$`)", regexPattern))
		} else {
			// Exact path match
			pathMatches = append(pathMatches, fmt.Sprintf("Path(`%s`)", pattern))
		}
	}

	if len(pathMatches) == 1 {
		return pathMatches[0]
	}

	// Multiple paths - combine with OR
	return fmt.Sprintf("(%s)", strings.Join(pathMatches, " || "))
}

// convertHostHeaderToMatch converts host headers to Traefik host matchers
func (c *RuleConverter) convertHostHeaderToMatch(hosts []string) string {
	if len(hosts) == 0 {
		return ""
	}

	var hostMatches []string
	for _, host := range hosts {
		if strings.Contains(host, "*") {
			// Wildcard host - use HostRegexp
			regexPattern := strings.ReplaceAll(host, "*", "[^.]+")
			hostMatches = append(hostMatches, fmt.Sprintf("HostRegexp(`^%s$`)", regexPattern))
		} else {
			// Exact host match
			hostMatches = append(hostMatches, fmt.Sprintf("Host(`%s`)", host))
		}
	}

	if len(hostMatches) == 1 {
		return hostMatches[0]
	}

	// Multiple hosts - combine with OR
	return fmt.Sprintf("(%s)", strings.Join(hostMatches, " || "))
}

// convertHttpHeaderToMatch converts HTTP header conditions to Traefik header matchers
func (c *RuleConverter) convertHttpHeaderToMatch(config *generated_elbv2.HttpHeaderConditionConfig) string {
	if config.HttpHeaderName == nil || len(config.Values) == 0 {
		return ""
	}

	var headerMatches []string
	headerName := *config.HttpHeaderName
	
	for _, value := range config.Values {
		if strings.Contains(value, "*") {
			// Wildcard value - use HeaderRegexp
			regexPattern := strings.ReplaceAll(value, "*", ".*")
			headerMatches = append(headerMatches, fmt.Sprintf("HeaderRegexp(`%s`, `^%s$`)", headerName, regexPattern))
		} else {
			// Exact header match
			headerMatches = append(headerMatches, fmt.Sprintf("Header(`%s`, `%s`)", headerName, value))
		}
	}

	if len(headerMatches) == 1 {
		return headerMatches[0]
	}

	// Multiple values - combine with OR
	return fmt.Sprintf("(%s)", strings.Join(headerMatches, " || "))
}

// convertHttpMethodToMatch converts HTTP methods to Traefik method matchers
func (c *RuleConverter) convertHttpMethodToMatch(methods []string) string {
	if len(methods) == 0 {
		return ""
	}

	var methodMatches []string
	for _, method := range methods {
		methodMatches = append(methodMatches, fmt.Sprintf("Method(`%s`)", strings.ToUpper(method)))
	}

	if len(methodMatches) == 1 {
		return methodMatches[0]
	}

	// Multiple methods - combine with OR
	return fmt.Sprintf("(%s)", strings.Join(methodMatches, " || "))
}

// convertQueryStringToMatch converts query string conditions to Traefik query matchers
func (c *RuleConverter) convertQueryStringToMatch(values []generated_elbv2.QueryStringKeyValuePair) string {
	if len(values) == 0 {
		return ""
	}

	var queryMatches []string
	for _, kv := range values {
		if kv.Key != nil {
			if kv.Value != nil {
				// Key-value pair
				queryMatches = append(queryMatches, fmt.Sprintf("Query(`%s=%s`)", *kv.Key, *kv.Value))
			} else {
				// Key exists (any value)
				queryMatches = append(queryMatches, fmt.Sprintf("Query(`%s`)", *kv.Key))
			}
		}
	}

	if len(queryMatches) == 1 {
		return queryMatches[0]
	}

	// Multiple query conditions - combine with AND (all must match)
	return fmt.Sprintf("(%s)", strings.Join(queryMatches, " && "))
}

// convertSourceIpToMatch converts source IP conditions to Traefik client IP matchers
func (c *RuleConverter) convertSourceIpToMatch(ips []string) string {
	if len(ips) == 0 {
		return ""
	}

	var ipMatches []string
	for _, ip := range ips {
		// Traefik uses ClientIP matcher for source IP
		ipMatches = append(ipMatches, fmt.Sprintf("ClientIP(`%s`)", ip))
	}

	if len(ipMatches) == 1 {
		return ipMatches[0]
	}

	// Multiple IPs - combine with OR
	return fmt.Sprintf("(%s)", strings.Join(ipMatches, " || "))
}

// ConvertRuleConditionsFromJSON deserializes rule conditions from JSON string
func (c *RuleConverter) ConvertRuleConditionsFromJSON(conditionsJSON string) ([]generated_elbv2.RuleCondition, error) {
	var conditions []generated_elbv2.RuleCondition
	if err := json.Unmarshal([]byte(conditionsJSON), &conditions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
	}
	return conditions, nil
}

// ConvertRuleActionsFromJSON deserializes rule actions from JSON string
func (c *RuleConverter) ConvertRuleActionsFromJSON(actionsJSON string) ([]generated_elbv2.Action, error) {
	var actions []generated_elbv2.Action
	if err := json.Unmarshal([]byte(actionsJSON), &actions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal actions: %w", err)
	}
	return actions, nil
}

// ExtractTargetGroupFromActions extracts the target group ARN from rule actions
func (c *RuleConverter) ExtractTargetGroupFromActions(actions []generated_elbv2.Action) (string, error) {
	for _, action := range actions {
		if action.Type == generated_elbv2.ActionTypeEnumFORWARD {
			if action.TargetGroupArn != nil {
				return *action.TargetGroupArn, nil
			}
			// Check forward config for weighted targets
			if action.ForwardConfig != nil && len(action.ForwardConfig.TargetGroups) > 0 {
				// For now, just use the first target group
				// TODO: Implement weighted routing support
				if action.ForwardConfig.TargetGroups[0].TargetGroupArn != nil {
					return *action.ForwardConfig.TargetGroups[0].TargetGroupArn, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no forward action with target group found")
}