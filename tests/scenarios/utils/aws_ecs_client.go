package utils

import (
	"encoding/json"
	"fmt"
)

// ECSClient wraps CurlClient to provide backward compatibility
type ECSClient struct {
	*CurlClient
}

// NewECSClient creates a new ECS client (backward compatibility)
func NewECSClient(endpoint string) *ECSClient {
	return &ECSClient{
		CurlClient: NewCurlClient(endpoint),
	}
}

// Legacy API methods for backward compatibility with existing tests

// RegisterTaskDefinition (legacy) registers a task definition using map format
func (c *ECSClient) RegisterTaskDefinition(taskDef map[string]interface{}) (map[string]interface{}, error) {
	// Convert map to JSON
	jsonData, err := json.Marshal(taskDef)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task definition: %w", err)
	}

	// Extract family name
	family, ok := taskDef["family"].(string)
	if !ok {
		return nil, fmt.Errorf("family field is required")
	}

	// Call new API
	result, err := c.CurlClient.RegisterTaskDefinition(family, string(jsonData))
	if err != nil {
		return nil, err
	}

	// Convert result to map for backward compatibility
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var tdMap map[string]interface{}
	if err := json.Unmarshal(resultJSON, &tdMap); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"taskDefinition": tdMap,
	}, nil
}

// DescribeTaskDefinition (legacy) describes a task definition
func (c *ECSClient) DescribeTaskDefinition(taskDefID string) (map[string]interface{}, error) {
	result, err := c.CurlClient.DescribeTaskDefinition(taskDefID)
	if err != nil {
		return nil, err
	}

	// Convert result to map
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	var tdMap map[string]interface{}
	if err := json.Unmarshal(resultJSON, &tdMap); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"taskDefinition": tdMap,
	}, nil
}

// ListTaskDefinitionFamilies lists task definition families
func (c *ECSClient) ListTaskDefinitionFamilies() (map[string]interface{}, error) {
	// For now, just return empty list as this is not implemented in the interface
	return map[string]interface{}{
		"families": []string{},
	}, nil
}

// ListTaskDefinitionsWithOptions lists task definitions with options
func (c *ECSClient) ListTaskDefinitionsWithOptions(params map[string]interface{}) (map[string]interface{}, error) {
	arns, err := c.CurlClient.ListTaskDefinitions()
	if err != nil {
		return nil, err
	}

	// Convert []string to []interface{} for test compatibility
	interfaceArns := make([]interface{}, len(arns))
	for i, arn := range arns {
		interfaceArns[i] = arn
	}

	return map[string]interface{}{
		"taskDefinitionArns": interfaceArns,
	}, nil
}

// DeregisterTaskDefinition (legacy) deregisters a task definition
func (c *ECSClient) DeregisterTaskDefinition(taskDefArn string) (map[string]interface{}, error) {
	err := c.CurlClient.DeregisterTaskDefinition(taskDefArn)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"taskDefinition": map[string]interface{}{
			"taskDefinitionArn": taskDefArn,
			"status":            "INACTIVE",
		},
	}, nil
}