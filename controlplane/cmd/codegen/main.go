package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type SmithyModel struct {
	Smithy   string                       `json:"smithy"`
	Metadata map[string]interface{}       `json:"metadata"`
	Shapes   map[string]SmithyShape       `json:"shapes"`
}

type SmithyShape struct {
	Type       string                     `json:"type"`
	Version    string                     `json:"version,omitempty"`
	Operations []SmithyTarget             `json:"operations,omitempty"`
	Input      *SmithyTarget              `json:"input,omitempty"`
	Output     *SmithyTarget              `json:"output,omitempty"`
	Errors     []SmithyTarget             `json:"errors,omitempty"`
	Members    map[string]SmithyMember    `json:"members,omitempty"`
	Member     *SmithyMember              `json:"member,omitempty"` // For list types
	Key        *SmithyMember              `json:"key,omitempty"`    // For map types
	Value      *SmithyMember              `json:"value,omitempty"`  // For map types
	Traits     map[string]interface{}     `json:"traits,omitempty"`
}

type SmithyTarget struct {
	Target string `json:"target"`
}

type SmithyMember struct {
	Target string                 `json:"target"`
	Traits map[string]interface{} `json:"traits,omitempty"`
}

type Operation struct {
	Name         string
	InputType    string
	OutputType   string
	Documentation string
	Errors       []string
}

type TypeDef struct {
	Name         string
	Type         string
	Members      []Member
	Documentation string
}

type Member struct {
	Name         string
	Type         string
	Required     bool
	Documentation string
}

func main() {
	var (
		modelPath  = flag.String("model", "api-models/ecs.json", "Path to Smithy model JSON file")
		outputDir  = flag.String("output", "internal/controlplane/api/generated", "Output directory for generated code")
	)
	flag.Parse()

	// Read and parse Smithy model
	data, err := os.ReadFile(*modelPath)
	if err != nil {
		log.Fatalf("Failed to read model file: %v", err)
	}

	var model SmithyModel
	if err := json.Unmarshal(data, &model); err != nil {
		log.Fatalf("Failed to parse model JSON: %v", err)
	}

	// Extract service operations and types
	operations, types := extractOperationsAndTypes(model)

	// Generate code
	if err := generateCode(*outputDir, operations, types); err != nil {
		log.Fatalf("Failed to generate code: %v", err)
	}

	fmt.Printf("Generated %d operations and %d types to %s\n", len(operations), len(types), *outputDir)
}

func extractOperationsAndTypes(model SmithyModel) ([]Operation, []TypeDef) {
	var operations []Operation
	var types []TypeDef
	typeSet := make(map[string]bool)
	listTypes := make(map[string]string) // Maps list type name to its element type

	// First pass: collect all type names and list mappings
	for shapeID, shape := range model.Shapes {
		typeName := extractTypeName(shapeID)
		if shape.Type == "structure" || shape.Type == "map" || shape.Type == "enum" || 
		   shape.Type == "integer" || shape.Type == "long" || shape.Type == "boolean" || 
		   shape.Type == "string" || shape.Type == "double" || shape.Type == "timestamp" {
			typeSet[typeName] = true
		} else if shape.Type == "list" && shape.Member != nil {
			// Store list type mapping for later use
			elementType := extractTypeName(shape.Member.Target)
			listTypes[typeName] = elementType
		}
	}

	// Second pass: extract operations and types
	for shapeID, shape := range model.Shapes {
		switch shape.Type {
		case "service":
			// Extract operations from service
			for _, opTarget := range shape.Operations {
				if opShape, exists := model.Shapes[opTarget.Target]; exists {
					op := Operation{
						Name:         extractOperationName(opTarget.Target),
						Documentation: extractDocumentation(opShape.Traits),
					}
					
					if opShape.Input != nil {
						op.InputType = extractTypeName(opShape.Input.Target)
					}
					if opShape.Output != nil {
						op.OutputType = extractTypeName(opShape.Output.Target)
					}
					
					for _, errorTarget := range opShape.Errors {
						op.Errors = append(op.Errors, extractTypeName(errorTarget.Target))
					}
					
					operations = append(operations, op)
				}
			}
		case "structure":
			// Extract all structure type definitions
			typeName := extractTypeName(shapeID)
			
			var members []Member
			for memberName, member := range shape.Members {
				memberType := mapTypeWithContext(member.Target, typeSet, listTypes)
				members = append(members, Member{
					Name:         memberName,
					Type:         memberType,
					Required:     isRequired(member.Traits),
					Documentation: extractDocumentation(member.Traits),
				})
			}
			// Sort members for consistent output
			sort.Slice(members, func(i, j int) bool {
				return members[i].Name < members[j].Name
			})
			
			types = append(types, TypeDef{
				Name:         typeName,
				Type:         "structure",
				Members:      members,
				Documentation: extractDocumentation(shape.Traits),
			})
		case "list":
			// Skip list type definitions - they will be inlined as slices
			continue
		case "map":
			// Extract map type definitions
			typeName := extractTypeName(shapeID)
			if shape.Key != nil && shape.Value != nil {
				keyType := mapTypeWithContext(shape.Key.Target, typeSet, listTypes)
				valueType := mapTypeWithContext(shape.Value.Target, typeSet, listTypes)
				types = append(types, TypeDef{
					Name:         typeName,
					Type:         "map",
					Members:      []Member{{Name: "Key", Type: keyType}, {Name: "Value", Type: valueType}},
					Documentation: extractDocumentation(shape.Traits),
				})
			}
		case "enum":
			// Extract enum type definitions
			typeName := extractTypeName(shapeID)
			types = append(types, TypeDef{
				Name:         typeName,
				Type:         "enum",
				Documentation: extractDocumentation(shape.Traits),
			})
		case "integer", "long", "boolean", "string", "double", "timestamp":
			// Extract primitive type aliases
			typeName := extractTypeName(shapeID)
			// Skip built-in Smithy types
			if strings.HasPrefix(shapeID, "smithy.api#") {
				continue
			}
			types = append(types, TypeDef{
				Name:         typeName,
				Type:         shape.Type,
				Documentation: extractDocumentation(shape.Traits),
			})
		}
	}

	// Sort for consistent output
	sort.Slice(operations, func(i, j int) bool {
		return operations[i].Name < operations[j].Name
	})
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	return operations, types
}

func extractOperationName(target string) string {
	parts := strings.Split(target, "#")
	if len(parts) == 2 {
		return parts[1]
	}
	return target
}

func extractTypeName(target string) string {
	parts := strings.Split(target, "#")
	if len(parts) == 2 {
		return parts[1]
	}
	return target
}

func extractDocumentation(traits map[string]interface{}) string {
	if traits == nil {
		return ""
	}
	if doc, exists := traits["smithy.api#documentation"]; exists {
		if str, ok := doc.(string); ok {
			return strings.TrimSpace(str)
		}
	}
	return ""
}

func isRequired(traits map[string]interface{}) bool {
	if traits == nil {
		return false
	}
	_, exists := traits["smithy.api#required"]
	return exists
}

func generateCode(outputDir string, operations []Operation, types []TypeDef) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Generate operations file
	if err := generateOperations(filepath.Join(outputDir, "operations.go"), operations); err != nil {
		return fmt.Errorf("failed to generate operations: %v", err)
	}

	// Generate types file
	if err := generateTypes(filepath.Join(outputDir, "types.go"), types); err != nil {
		return fmt.Errorf("failed to generate types: %v", err)
	}

	return nil
}

const operationsTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package generated

import (
	"context"
	"net/http"
)

// ECSServiceInterface defines the interface for all ECS operations
type ECSServiceInterface interface {
{{- range .Operations }}
	{{ .Name }}(ctx context.Context, req *{{ .InputType }}) (*{{ .OutputType }}, error)
{{- end }}
}

// ECSService implements the ECS service operations
type ECSService struct{}

// NewECSService creates a new ECS service instance
func NewECSService() *ECSService {
	return &ECSService{}
}

{{- range .Operations }}

// {{ .Name }} implements the {{ .Name }} operation
func (s *ECSService) {{ .Name }}(ctx context.Context, req *{{ .InputType }}) (*{{ .OutputType }}, error) {
	// TODO: Implement {{ .Name }} operation
	return &{{ .OutputType }}{}, nil
}
{{- end }}

// HTTP handlers for each operation
{{- range .Operations }}

// Handle{{ .Name }} handles HTTP requests for {{ .Name }}
func Handle{{ .Name }}(service ECSServiceInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Parse request body into {{ .InputType }}
		// TODO: Call service.{{ .Name }}(ctx, req)
		// TODO: Write response as JSON
		w.Header().Set("Content-Type", "application/x-amz-json-1.1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}
}
{{- end }}
`

const typesTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package generated

import "time"

{{- range .Types }}
{{- if eq .Type "structure" }}

// {{ .Name }} represents the {{ .Name }} structure
type {{ .Name }} struct {
{{- range .Members }}
	{{ toCamelCase .Name }} {{ .Type }} ` + "`json:\"{{ toLowerCamel .Name }}{{ if not .Required }},omitempty{{ end }}\"`" + `
{{- end }}
}
{{- else if eq .Type "map" }}

// {{ .Name }} represents a map type  
type {{ .Name }} map[{{ (index .Members 0).Type }}]{{ (index .Members 1).Type }}
{{- else if eq .Type "enum" }}

// {{ .Name }} represents an enum type
type {{ .Name }} string
{{- else if eq .Type "integer" }}

// {{ .Name }} represents an integer type alias
type {{ .Name }} int32
{{- else if eq .Type "long" }}

// {{ .Name }} represents a long type alias
type {{ .Name }} int64
{{- else if eq .Type "boolean" }}

// {{ .Name }} represents a boolean type alias
type {{ .Name }} bool
{{- else if eq .Type "string" }}

// {{ .Name }} represents a string type alias
type {{ .Name }} string
{{- else if eq .Type "double" }}

// {{ .Name }} represents a double type alias
type {{ .Name }} float64
{{- else if eq .Type "timestamp" }}

// {{ .Name }} represents a timestamp type alias
type {{ .Name }} time.Time
{{- end }}
{{- end }}
`

func generateOperations(filename string, operations []Operation) error {
	tmpl, err := template.New("operations").Parse(operationsTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, map[string]interface{}{
		"Operations": operations,
	})
}

func generateTypes(filename string, types []TypeDef) error {
	funcMap := template.FuncMap{
		"mapType": mapType,
		"toLowerCamel": toLowerCamel,
		"toCamelCase": toCamelCase,
	}

	tmpl, err := template.New("types").Funcs(funcMap).Parse(typesTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, map[string]interface{}{
		"Types": types,
	})
}

func mapType(smithyType string) string {
	return mapTypeWithContext(smithyType, nil, nil)
}

func mapTypeWithContext(smithyType string, typeSet map[string]bool, listTypes map[string]string) string {
	// Extract the type name without namespace
	typeName := extractTypeName(smithyType)
	
	// Check basic types first (with or without namespace)
	switch typeName {
	case "String":
		return "*string"
	case "Integer":
		return "*int32"
	case "Long":
		return "*int64"
	case "Boolean":
		return "*bool"
	case "Double":
		return "*float64"
	case "Timestamp":
		return "*time.Time"
	case "BoxedInteger":
		return "*int32"
	case "BoxedBoolean":
		return "*bool"
	}
	
	// Also check with full namespace
	switch smithyType {
	case "smithy.api#String", "com.amazonaws.ecs#String":
		return "*string"
	case "smithy.api#Integer", "com.amazonaws.ecs#Integer":
		return "*int32"
	case "smithy.api#Long", "com.amazonaws.ecs#Long":
		return "*int64"
	case "smithy.api#Boolean", "com.amazonaws.ecs#Boolean":
		return "*bool"
	case "smithy.api#Double", "com.amazonaws.ecs#Double":
		return "*float64"
	case "smithy.api#Timestamp", "com.amazonaws.ecs#Timestamp":
		return "*time.Time"
	default:
		if strings.HasPrefix(smithyType, "smithy.api#") {
			return "*string" // Default to string for unknown smithy types
		}
		
		// Check if this is a list type
		if listTypes != nil {
			if elementType, ok := listTypes[typeName]; ok {
				// Special case for StringList
				if typeName == "StringList" {
					return "[]string"
				}
				// Convert list type to slice
				elemType := mapTypeWithContext("com.amazonaws.ecs#" + elementType, typeSet, listTypes)
				// Remove pointer from element type if present
				if strings.HasPrefix(elemType, "*") {
					elemType = elemType[1:]
				}
				return "[]" + elemType
			}
		}
		
		// Check if this is a known custom type
		if typeSet != nil && typeSet[typeName] {
			// Map types remain as type aliases
			if strings.HasSuffix(typeName, "Map") {
				return typeName // Return as-is for type alias
			}
			return "*" + typeName
		}
		
		// For unknown types, use interface{}
		return "interface{}"
	}
}

func toLowerCamel(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func toCamelCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}