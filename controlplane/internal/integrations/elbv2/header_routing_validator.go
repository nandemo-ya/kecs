package elbv2

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_elbv2"
)

// HeaderRoutingValidator validates and provides insights for header-based routing rules
type HeaderRoutingValidator struct {
	// Common header patterns
	knownHeaders map[string]HeaderInfo
}

// HeaderInfo contains information about a known header
type HeaderInfo struct {
	Name        string
	Description string
	Examples    []string
	Security    SecurityLevel
}

// SecurityLevel indicates the security sensitivity of a header
type SecurityLevel int

const (
	SecurityLevelPublic SecurityLevel = iota
	SecurityLevelSensitive
	SecurityLevelSecret
)

// NewHeaderRoutingValidator creates a new header routing validator
func NewHeaderRoutingValidator() *HeaderRoutingValidator {
	return &HeaderRoutingValidator{
		knownHeaders: initKnownHeaders(),
	}
}

// ValidateHeaderConditions validates header conditions for security and best practices
func (v *HeaderRoutingValidator) ValidateHeaderConditions(conditions []generated_elbv2.RuleCondition) ([]ValidationIssue, error) {
	var issues []ValidationIssue

	for _, condition := range conditions {
		if condition.HttpHeaderConfig != nil {
			headerIssues := v.validateHttpHeaderConfig(condition.HttpHeaderConfig)
			issues = append(issues, headerIssues...)
		}
	}

	return issues, nil
}

// ValidationIssue represents a validation problem
type ValidationIssue struct {
	Severity    IssueSeverity
	Header      string
	Message     string
	Suggestion  string
}

// IssueSeverity indicates the severity of a validation issue
type IssueSeverity int

const (
	IssueSeverityInfo IssueSeverity = iota
	IssueSeverityWarning
	IssueSeverityError
)

// validateHttpHeaderConfig validates a single HTTP header configuration
func (v *HeaderRoutingValidator) validateHttpHeaderConfig(config *generated_elbv2.HttpHeaderConditionConfig) []ValidationIssue {
	var issues []ValidationIssue

	if config.HttpHeaderName == nil {
		issues = append(issues, ValidationIssue{
			Severity: IssueSeverityError,
			Message:  "HTTP header name is required",
		})
		return issues
	}

	headerName := *config.HttpHeaderName

	// Check for security-sensitive headers
	if issue := v.checkSecurityHeader(headerName); issue != nil {
		issues = append(issues, *issue)
	}

	// Validate header name format
	if issue := v.validateHeaderNameFormat(headerName); issue != nil {
		issues = append(issues, *issue)
	}

	// Validate header values
	for _, value := range config.Values {
		if issue := v.validateHeaderValue(headerName, value); issue != nil {
			issues = append(issues, *issue)
		}
	}

	// Check for known header patterns
	if info, exists := v.knownHeaders[strings.ToLower(headerName)]; exists {
		if info.Security == SecurityLevelSecret {
			issues = append(issues, ValidationIssue{
				Severity:   IssueSeverityError,
				Header:     headerName,
				Message:    fmt.Sprintf("Header '%s' contains sensitive information and should not be used for routing", headerName),
				Suggestion: "Use a different header or implement proper security measures",
			})
		}
	}

	return issues
}

// checkSecurityHeader checks if a header might expose security information
func (v *HeaderRoutingValidator) checkSecurityHeader(headerName string) *ValidationIssue {
	lowerHeader := strings.ToLower(headerName)

	// Check for authentication/authorization headers
	if strings.Contains(lowerHeader, "authorization") ||
		strings.Contains(lowerHeader, "api-key") ||
		strings.Contains(lowerHeader, "secret") ||
		strings.Contains(lowerHeader, "password") {
		return &ValidationIssue{
			Severity:   IssueSeverityWarning,
			Header:     headerName,
			Message:    "Routing based on authentication headers may expose sensitive information",
			Suggestion: "Consider using custom headers like X-User-Role or X-Feature-Flag instead",
		}
	}

	// Check for session/cookie headers
	if strings.Contains(lowerHeader, "cookie") || strings.Contains(lowerHeader, "session") {
		return &ValidationIssue{
			Severity:   IssueSeverityWarning,
			Header:     headerName,
			Message:    "Routing based on cookies or sessions may cause inconsistent behavior",
			Suggestion: "Use dedicated routing headers instead of session data",
		}
	}

	return nil
}

// validateHeaderNameFormat validates the format of a header name
func (v *HeaderRoutingValidator) validateHeaderNameFormat(headerName string) *ValidationIssue {
	// Check for valid header name format (RFC 7230)
	validHeaderRegex := regexp.MustCompile(`^[!#$%&'*+\-.0-9A-Z^_` + "`" + `a-z|~]+$`)
	if !validHeaderRegex.MatchString(headerName) {
		return &ValidationIssue{
			Severity:   IssueSeverityError,
			Header:     headerName,
			Message:    "Invalid header name format",
			Suggestion: "Use only alphanumeric characters, hyphens, and underscores",
		}
	}

	// Warn about non-standard headers without X- prefix
	if !strings.HasPrefix(headerName, "X-") && !v.isStandardHeader(headerName) {
		return &ValidationIssue{
			Severity:   IssueSeverityInfo,
			Header:     headerName,
			Message:    "Custom headers should use 'X-' prefix",
			Suggestion: fmt.Sprintf("Consider renaming to 'X-%s'", headerName),
		}
	}

	return nil
}

// validateHeaderValue validates a header value
func (v *HeaderRoutingValidator) validateHeaderValue(headerName, value string) *ValidationIssue {
	// Check for potential injection attempts
	if strings.ContainsAny(value, "\r\n") {
		return &ValidationIssue{
			Severity:   IssueSeverityError,
			Header:     headerName,
			Message:    "Header value contains invalid characters (CR/LF)",
			Suggestion: "Remove newline characters from header values",
		}
	}

	// Warn about overly broad wildcards
	if value == "*" {
		return &ValidationIssue{
			Severity:   IssueSeverityWarning,
			Header:     headerName,
			Message:    "Using wildcard '*' matches all values",
			Suggestion: "Consider using more specific patterns",
		}
	}

	// Check for regex complexity in wildcard patterns
	wildcardCount := strings.Count(value, "*")
	if wildcardCount > 2 {
		return &ValidationIssue{
			Severity:   IssueSeverityWarning,
			Header:     headerName,
			Message:    "Complex wildcard patterns may impact performance",
			Suggestion: "Simplify the pattern or use exact matches",
		}
	}

	return nil
}

// isStandardHeader checks if a header is a standard HTTP header
func (v *HeaderRoutingValidator) isStandardHeader(headerName string) bool {
	standardHeaders := map[string]bool{
		"Accept":              true,
		"Accept-Encoding":     true,
		"Accept-Language":     true,
		"Authorization":       true,
		"Cache-Control":       true,
		"Content-Type":        true,
		"Cookie":              true,
		"Host":                true,
		"Referer":             true,
		"User-Agent":          true,
		"Content-Length":      true,
		"Content-Encoding":    true,
		"If-None-Match":       true,
		"If-Modified-Since":   true,
	}

	return standardHeaders[headerName]
}

// GetHeaderSuggestions provides suggestions for common routing scenarios
func (v *HeaderRoutingValidator) GetHeaderSuggestions(scenario string) []HeaderSuggestion {
	suggestions := map[string][]HeaderSuggestion{
		"api-version": {
			{
				Header:      "X-API-Version",
				Description: "Route based on API version",
				Example:     "X-API-Version: 2.0",
				Pattern:     "2.*",
			},
			{
				Header:      "Accept",
				Description: "Route based on accepted content version",
				Example:     "Accept: application/vnd.api+json;version=2",
				Pattern:     "*version=2*",
			},
		},
		"feature-flag": {
			{
				Header:      "X-Feature-Flag",
				Description: "Route based on feature flags",
				Example:     "X-Feature-Flag: new-ui,dark-mode",
				Pattern:     "*new-ui*",
			},
			{
				Header:      "X-Beta-User",
				Description: "Route beta users to experimental features",
				Example:     "X-Beta-User: true",
				Pattern:     "true",
			},
		},
		"tenant": {
			{
				Header:      "X-Tenant-ID",
				Description: "Route based on tenant identifier",
				Example:     "X-Tenant-ID: enterprise-123",
				Pattern:     "enterprise-*",
			},
			{
				Header:      "X-Customer-Tier",
				Description: "Route based on customer tier",
				Example:     "X-Customer-Tier: premium",
				Pattern:     "premium",
			},
		},
		"mobile": {
			{
				Header:      "User-Agent",
				Description: "Route mobile traffic",
				Example:     "User-Agent: MyApp/1.0 (iPhone; iOS 14.0)",
				Pattern:     "*Mobile*,*Android*,*iOS*",
			},
			{
				Header:      "X-App-Version",
				Description: "Route based on app version",
				Example:     "X-App-Version: 2.5.0",
				Pattern:     "2.5.*",
			},
		},
	}

	if suggestions, exists := suggestions[scenario]; exists {
		return suggestions
	}

	return []HeaderSuggestion{}
}

// HeaderSuggestion provides a suggested header configuration
type HeaderSuggestion struct {
	Header      string
	Description string
	Example     string
	Pattern     string
}

// initKnownHeaders initializes the known headers database
func initKnownHeaders() map[string]HeaderInfo {
	return map[string]HeaderInfo{
		"authorization": {
			Name:        "Authorization",
			Description: "Contains authentication credentials",
			Examples:    []string{"Bearer token", "Basic credentials"},
			Security:    SecurityLevelSecret,
		},
		"x-api-key": {
			Name:        "X-API-Key",
			Description: "API key for authentication",
			Examples:    []string{"abc123def456"},
			Security:    SecurityLevelSecret,
		},
		"user-agent": {
			Name:        "User-Agent",
			Description: "Client application identifier",
			Examples:    []string{"Mozilla/5.0", "MyApp/1.0"},
			Security:    SecurityLevelPublic,
		},
		"x-api-version": {
			Name:        "X-API-Version",
			Description: "API version indicator",
			Examples:    []string{"1.0", "2.1", "v3"},
			Security:    SecurityLevelPublic,
		},
		"x-feature-flag": {
			Name:        "X-Feature-Flag",
			Description: "Feature flags for A/B testing",
			Examples:    []string{"new-ui", "experimental", "beta"},
			Security:    SecurityLevelPublic,
		},
		"content-type": {
			Name:        "Content-Type",
			Description: "Media type of the request body",
			Examples:    []string{"application/json", "text/html"},
			Security:    SecurityLevelPublic,
		},
		"accept": {
			Name:        "Accept",
			Description: "Acceptable response media types",
			Examples:    []string{"application/json", "text/html"},
			Security:    SecurityLevelPublic,
		},
		"cookie": {
			Name:        "Cookie",
			Description: "HTTP cookies",
			Examples:    []string{"sessionid=abc123"},
			Security:    SecurityLevelSensitive,
		},
	}
}

// AnalyzeHeaderRoutingComplexity analyzes the complexity of header-based routing rules
func (v *HeaderRoutingValidator) AnalyzeHeaderRoutingComplexity(rules []generated_elbv2.Rule) ComplexityReport {
	report := ComplexityReport{
		TotalRules:      len(rules),
		HeaderBasedRules: 0,
		UniqueHeaders:   make(map[string]int),
		ComplexityScore: 0,
	}

	for _, rule := range rules {
		if rule.Conditions != nil {
			for _, condition := range rule.Conditions {
				if condition.HttpHeaderConfig != nil && condition.HttpHeaderConfig.HttpHeaderName != nil {
					report.HeaderBasedRules++
					headerName := *condition.HttpHeaderConfig.HttpHeaderName
					report.UniqueHeaders[headerName]++
					
					// Add complexity for wildcards
					for _, value := range condition.HttpHeaderConfig.Values {
						if strings.Contains(value, "*") {
							report.ComplexityScore += 2
						} else {
							report.ComplexityScore += 1
						}
					}
				}
			}
		}
	}

	// Calculate overall complexity
	report.ComplexityScore += len(report.UniqueHeaders) * 5
	
	return report
}

// ComplexityReport contains analysis of routing complexity
type ComplexityReport struct {
	TotalRules       int
	HeaderBasedRules int
	UniqueHeaders    map[string]int
	ComplexityScore  int
}

// GetComplexityLevel returns a human-readable complexity level
func (r *ComplexityReport) GetComplexityLevel() string {
	if r.ComplexityScore < 10 {
		return "Simple"
	} else if r.ComplexityScore < 50 {
		return "Moderate"
	} else if r.ComplexityScore < 100 {
		return "Complex"
	}
	return "Very Complex"
}