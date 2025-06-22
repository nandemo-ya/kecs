package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// Validation constants
const (
	// Cluster name constraints
	minClusterNameLength = 1
	maxClusterNameLength = 255
	
	// Valid cluster name pattern: alphanumeric characters and hyphens only
	clusterNamePattern = `^[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`
)

var (
	clusterNameRegex = regexp.MustCompile(clusterNamePattern)
	
	// Valid capacity providers
	validCapacityProviders = map[string]bool{
		"FARGATE":      true,
		"FARGATE_SPOT": true,
		"EC2":          true,
	}
	
	// Valid setting names
	validSettingNames = map[generated.ClusterSettingName]bool{
		generated.ClusterSettingNameContainerInsights: true,
	}
	
	// Valid setting values
	validSettingValues = map[generated.ClusterSettingName]map[string]bool{
		generated.ClusterSettingNameContainerInsights: {
			"enabled":  true,
			"disabled": true,
		},
	}
)

// ValidateClusterName validates cluster name according to AWS ECS rules
func ValidateClusterName(name string) error {
	// Check length
	if len(name) < minClusterNameLength || len(name) > maxClusterNameLength {
		return fmt.Errorf("Invalid parameter: Cluster name length must be between %d and %d characters", 
			minClusterNameLength, maxClusterNameLength)
	}
	
	// Check for invalid characters
	if !clusterNameRegex.MatchString(name) {
		return fmt.Errorf("Invalid parameter: Cluster name can only contain alphanumeric characters and hyphens")
	}
	
	// Check for consecutive hyphens (AWS doesn't allow this)
	if strings.Contains(name, "--") {
		return fmt.Errorf("Invalid parameter: Cluster name cannot contain consecutive hyphens")
	}
	
	return nil
}

// ValidateClusterARN validates cluster ARN format
func ValidateClusterARN(arn string) error {
	// Basic ARN format validation
	// Format: arn:aws:ecs:region:account-id:cluster/cluster-name
	if !strings.HasPrefix(arn, "arn:aws:ecs:") {
		return fmt.Errorf("Invalid parameter: ARN must start with 'arn:aws:ecs:'")
	}
	
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return fmt.Errorf("Invalid parameter: Malformed ARN")
	}
	
	// Check resource type
	resourceParts := strings.Split(parts[5], "/")
	if len(resourceParts) != 2 || resourceParts[0] != "cluster" {
		return fmt.Errorf("Invalid parameter: ARN must be a cluster ARN")
	}
	
	// Validate the cluster name part
	clusterName := resourceParts[1]
	if clusterName == "" {
		return fmt.Errorf("Invalid parameter: ARN missing cluster name")
	}
	
	return nil
}

// ValidateClusterSettings validates cluster settings
func ValidateClusterSettings(settings []generated.ClusterSetting) error {
	for _, setting := range settings {
		if setting.Name == nil || setting.Value == nil {
			return fmt.Errorf("Invalid parameter: Setting name and value are required")
		}
		
		// Check if setting name is valid
		if !validSettingNames[*setting.Name] {
			return fmt.Errorf("Invalid parameter: Unknown setting name '%s'", *setting.Name)
		}
		
		// Check if setting value is valid for the given name
		if validValues, ok := validSettingValues[*setting.Name]; ok {
			if !validValues[*setting.Value] {
				return fmt.Errorf("Invalid parameter: Invalid value '%s' for setting '%s'", 
					*setting.Value, *setting.Name)
			}
		}
	}
	
	return nil
}

// ValidateCapacityProviders validates capacity providers and strategy
func ValidateCapacityProviders(providers []string, strategy []generated.CapacityProviderStrategyItem) error {
	// Validate providers
	for _, provider := range providers {
		if !validCapacityProviders[provider] {
			return fmt.Errorf("Invalid parameter: Unknown capacity provider '%s'", provider)
		}
	}
	
	// Validate strategy
	for _, item := range strategy {
		if item.CapacityProvider == nil {
			return fmt.Errorf("Invalid parameter: Capacity provider is required in strategy")
		}
		
		// Check if the provider in strategy is in the providers list
		found := false
		for _, provider := range providers {
			if provider == *item.CapacityProvider {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("Invalid parameter: Capacity provider '%s' in strategy is not in the providers list", 
				*item.CapacityProvider)
		}
		
		// Validate weight
		if item.Weight != nil && *item.Weight < 0 {
			return fmt.Errorf("Invalid parameter: Weight must be non-negative")
		}
		
		// Validate base
		if item.Base != nil && *item.Base < 0 {
			return fmt.Errorf("Invalid parameter: Base must be non-negative")
		}
	}
	
	return nil
}

// ValidateExecuteCommandConfiguration validates execute command configuration
func ValidateExecuteCommandConfiguration(config *generated.ExecuteCommandConfiguration) error {
	if config == nil {
		return nil
	}
	
	// Validate KMS key ID format if provided
	if config.KmsKeyId != nil && *config.KmsKeyId != "" {
		// Basic validation - in real AWS, this would validate against actual KMS keys
		if !strings.HasPrefix(*config.KmsKeyId, "arn:aws:kms:") && 
		   !regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`).MatchString(*config.KmsKeyId) {
			return fmt.Errorf("Invalid parameter: Invalid KMS key ID format")
		}
	}
	
	// Validate logging configuration
	if config.Logging != nil {
		validLogging := map[generated.ExecuteCommandLogging]bool{
			generated.ExecuteCommandLoggingNone:     true,
			generated.ExecuteCommandLoggingDefault:  true,
			generated.ExecuteCommandLoggingOverride: true,
		}
		
		if !validLogging[*config.Logging] {
			return fmt.Errorf("Invalid parameter: Invalid logging value '%s'", *config.Logging)
		}
		
		// If logging is OVERRIDE, logConfiguration must be provided
		if *config.Logging == generated.ExecuteCommandLoggingOverride && config.LogConfiguration == nil {
			return fmt.Errorf("Invalid parameter: logConfiguration is required when logging is set to OVERRIDE")
		}
	}
	
	return nil
}

// ValidateClusterIdentifier validates cluster identifier (name or ARN)
func ValidateClusterIdentifier(identifier string) error {
	if identifier == "" {
		return fmt.Errorf("Invalid parameter: Cluster identifier is required")
	}
	
	// Check if it's an ARN
	if strings.HasPrefix(identifier, "arn:") {
		return ValidateClusterARN(identifier)
	}
	
	// Otherwise, validate as cluster name
	return ValidateClusterName(identifier)
}

// ValidateResourceARN validates a generic resource ARN for tagging operations
func ValidateResourceARN(arn string) error {
	if arn == "" {
		return fmt.Errorf("Invalid parameter: Resource ARN is required")
	}
	
	if !strings.HasPrefix(arn, "arn:aws:ecs:") {
		return fmt.Errorf("Invalid parameter: Invalid resource ARN format")
	}
	
	return nil
}