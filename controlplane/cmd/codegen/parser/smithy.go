package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// SmithyAPI represents the parsed Smithy API definition
type SmithyAPI struct {
	Smithy   string                         `json:"smithy"`
	Metadata map[string]interface{}         `json:"metadata"`
	Shapes   map[string]*SmithyShape        `json:"shapes"`
}

// SmithyShape represents a shape in the Smithy model
type SmithyShape struct {
	Type       string                 `json:"type"`
	Members    map[string]*SmithyMember `json:"members,omitempty"`
	Member     *SmithyMember          `json:"member,omitempty"` // For list/map types
	Key        *SmithyMember          `json:"key,omitempty"`    // For map types
	Value      *SmithyMember          `json:"value,omitempty"`  // For map types
	Traits     map[string]interface{} `json:"traits,omitempty"`
	Target     string                 `json:"target,omitempty"`
	Input      *SmithyRef             `json:"input,omitempty"`
	Output     *SmithyRef             `json:"output,omitempty"`
	Errors     []SmithyRef            `json:"errors,omitempty"`
	Operations []SmithyRef            `json:"operations,omitempty"`
	Resources  []SmithyRef            `json:"resources,omitempty"`
	Min        *int                   `json:"min,omitempty"`
	Max        *int                   `json:"max,omitempty"`
}

// SmithyMember represents a member of a structure
type SmithyMember struct {
	Target string                 `json:"target"`
	Traits map[string]interface{} `json:"traits,omitempty"`
}

// SmithyRef represents a reference to another shape
type SmithyRef struct {
	Target string `json:"target"`
}

// ParseSmithyJSON parses a Smithy JSON file and returns the API definition
func ParseSmithyJSON(filename string) (*SmithyAPI, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var api SmithyAPI
	if err := json.Unmarshal(data, &api); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &api, nil
}

// GetServiceShape returns the service shape from the API definition
func (api *SmithyAPI) GetServiceShape() (*SmithyShape, string, error) {
	for name, shape := range api.Shapes {
		if shape.Type == "service" {
			return shape, name, nil
		}
	}
	return nil, "", fmt.Errorf("no service shape found")
}

// GetOperations returns all operation shapes for the service
func (api *SmithyAPI) GetOperations() map[string]*SmithyShape {
	operations := make(map[string]*SmithyShape)
	
	serviceShape, _, err := api.GetServiceShape()
	if err != nil {
		return operations
	}

	for _, opRef := range serviceShape.Operations {
		opName := opRef.Target
		if shape, exists := api.Shapes[opName]; exists && shape.Type == "operation" {
			operations[opName] = shape
		}
	}

	return operations
}

// ResolveShape follows the target chain to get the actual shape
func (api *SmithyAPI) ResolveShape(target string) (*SmithyShape, string) {
	shape, exists := api.Shapes[target]
	if !exists {
		return nil, ""
	}

	// Follow target references for simple types
	if shape.Type == "" && shape.Target != "" {
		return api.ResolveShape(shape.Target)
	}

	return shape, target
}

// GetShapeName extracts the simple name from a fully qualified shape name
func GetShapeName(fqn string) string {
	parts := strings.Split(fqn, "#")
	if len(parts) > 1 {
		return parts[1]
	}
	return fqn
}

// IsRequired checks if a member is required based on its traits
func (m *SmithyMember) IsRequired() bool {
	if m.Traits == nil {
		return false
	}
	_, required := m.Traits["smithy.api#required"]
	return required
}

// GetJSONName returns the JSON field name for a member
func (m *SmithyMember) GetJSONName(fieldName string) string {
	if m.Traits != nil {
		if jsonName, ok := m.Traits["smithy.api#jsonName"]; ok {
			if name, ok := jsonName.(string); ok {
				return name
			}
		}
	}
	// Convert to camelCase by default
	return toCamelCase(fieldName)
}

// toCamelCase converts a string to camelCase
func toCamelCase(s string) string {
	if s == "" {
		return s
	}
	// Handle acronyms
	if s == "ECS" {
		return "ecs"
	}
	if len(s) <= 3 && strings.ToUpper(s) == s {
		return strings.ToLower(s)
	}
	// Standard camelCase conversion
	return strings.ToLower(s[:1]) + s[1:]
}

// IsEnum checks if a shape is an enum
func (s *SmithyShape) IsEnum() bool {
	if s.Traits == nil {
		return false
	}
	_, hasEnum := s.Traits["smithy.api#enum"]
	return hasEnum || s.Type == "enum"
}

// GetEnumValues returns the enum values for an enum shape
func (s *SmithyShape) GetEnumValues() []string {
	var values []string
	
	if s.Traits != nil {
		if enumTrait, ok := s.Traits["smithy.api#enum"]; ok {
			if enumList, ok := enumTrait.([]interface{}); ok {
				for _, item := range enumList {
					if enumItem, ok := item.(map[string]interface{}); ok {
						if value, ok := enumItem["value"].(string); ok {
							values = append(values, value)
						}
					}
				}
			}
		}
	}
	
	// For member-based enums
	if len(values) == 0 && s.Type == "enum" && s.Members != nil {
		for name := range s.Members {
			values = append(values, name)
		}
	}
	
	return values
}

// IsPrimitive checks if a shape is a primitive type
func (s *SmithyShape) IsPrimitive() bool {
	switch s.Type {
	case "string", "boolean", "byte", "short", "integer", "long", 
	     "float", "double", "bigInteger", "bigDecimal", "timestamp",
	     "blob", "document":
		return true
	}
	return false
}

// IsCollection checks if a shape is a collection type
func (s *SmithyShape) IsCollection() bool {
	return s.Type == "list" || s.Type == "set"
}

// IsMap checks if a shape is a map type
func (s *SmithyShape) IsMap() bool {
	return s.Type == "map"
}