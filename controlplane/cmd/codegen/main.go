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
	Smithy   string                 `json:"smithy"`
	Metadata map[string]interface{} `json:"metadata"`
	Shapes   map[string]SmithyShape `json:"shapes"`
}

type SmithyShape struct {
	Type       string                  `json:"type"`
	Version    string                  `json:"version,omitempty"`
	Operations []SmithyTarget          `json:"operations,omitempty"`
	Input      *SmithyTarget           `json:"input,omitempty"`
	Output     *SmithyTarget           `json:"output,omitempty"`
	Errors     []SmithyTarget          `json:"errors,omitempty"`
	Members    map[string]SmithyMember `json:"members,omitempty"`
	Member     *SmithyMember           `json:"member,omitempty"` // For list types
	Key        *SmithyMember           `json:"key,omitempty"`    // For map types
	Value      *SmithyMember           `json:"value,omitempty"`  // For map types
	Traits     map[string]interface{}  `json:"traits,omitempty"`
}

type SmithyTarget struct {
	Target string `json:"target"`
}

type SmithyMember struct {
	Target string                 `json:"target"`
	Traits map[string]interface{} `json:"traits,omitempty"`
}

type Operation struct {
	Name          string
	InputType     string
	OutputType    string
	Documentation string
	Errors        []string
}

type TypeDef struct {
	Name          string
	Type          string
	Members       []Member
	EnumValues    []EnumValue // For enum types
	Documentation string
}

type EnumValue struct {
	Name  string
	Value string
}

type Member struct {
	Name          string
	Type          string
	Required      bool
	Documentation string
}

func main() {
	var (
		modelPath = flag.String("model", "api-models/ecs.json", "Path to Smithy model JSON file")
		outputDir = flag.String("output", "internal/controlplane/api/generated", "Output directory for generated code")
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
						Name:          extractOperationName(opTarget.Target),
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
					Name:          memberName,
					Type:          memberType,
					Required:      isRequired(member.Traits),
					Documentation: extractDocumentation(member.Traits),
				})
			}
			// Sort members for consistent output
			sort.Slice(members, func(i, j int) bool {
				return members[i].Name < members[j].Name
			})

			types = append(types, TypeDef{
				Name:          typeName,
				Type:          "structure",
				Members:       members,
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
					Name:          typeName,
					Type:          "map",
					Members:       []Member{{Name: "Key", Type: keyType}, {Name: "Value", Type: valueType}},
					Documentation: extractDocumentation(shape.Traits),
				})
			}
		case "enum":
			// Extract enum type definitions
			typeName := extractTypeName(shapeID)
			var enumValues []EnumValue
			for memberName, member := range shape.Members {
				value := memberName // Default to member name
				if member.Traits != nil {
					if enumValue, ok := member.Traits["smithy.api#enumValue"].(string); ok {
						value = enumValue
					}
				}
				enumValues = append(enumValues, EnumValue{
					Name:  memberName,
					Value: value,
				})
			}
			// Sort enum values for consistent output
			sort.Slice(enumValues, func(i, j int) bool {
				return enumValues[i].Name < enumValues[j].Name
			})
			types = append(types, TypeDef{
				Name:          typeName,
				Type:          "enum",
				EnumValues:    enumValues,
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
				Name:          typeName,
				Type:          shape.Type,
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

	// Separate enums from other types
	var enums []TypeDef
	var otherTypes []TypeDef
	for _, typ := range types {
		if typ.Type == "enum" {
			enums = append(enums, typ)
		} else {
			otherTypes = append(otherTypes, typ)
		}
	}

	// Generate types file
	if err := generateTypes(filepath.Join(outputDir, "types.go"), otherTypes); err != nil {
		return fmt.Errorf("failed to generate types: %v", err)
	}

	// Generate enums file
	if err := generateEnums(filepath.Join(outputDir, "enums.go"), enums); err != nil {
		return fmt.Errorf("failed to generate enums: %v", err)
	}

	// Generate routing file
	if err := generateRouting(filepath.Join(outputDir, "routing.go"), operations); err != nil {
		return fmt.Errorf("failed to generate routing: %v", err)
	}

	return nil
}

const operationsTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package generated

import (
	"context"
)

// ECSAPIInterface defines the interface for all ECS API operations
type ECSAPIInterface interface {
{{- range .Operations }}
	{{ .Name }}(ctx context.Context, req *{{ .InputType }}) (*{{ .OutputType }}, error)
{{- end }}
}
`

const typesTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package generated

import "time"

{{- range .Types }}
{{- if eq .Type "structure" }}

// {{ .Name }} represents the {{ .Name }} structure
type {{ .Name }} struct {
{{- range .Members }}
	{{ toCamelCase .Name }} {{ .Type }} ` + "`json:\"{{ toLowerCamel .Name }}{{ if and (not .Required) (not (hasPrefix .Type \"[]\")) }},omitempty{{ end }}\"`" + `
{{- end }}
}
{{- else if eq .Type "map" }}

// {{ .Name }} represents a map type  
type {{ .Name }} map[{{ (index .Members 0).Type }}]{{ (index .Members 1).Type }}
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
		"mapType":      mapType,
		"toLowerCamel": toLowerCamel,
		"toCamelCase":  toCamelCase,
		"hasPrefix":    strings.HasPrefix,
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
				elemType := mapTypeWithContext("com.amazonaws.ecs#"+elementType, typeSet, listTypes)
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

func toConstantName(s string) string {
	// Convert GRPC -> Grpc, HTTP -> Http, HTTP2 -> Http2, etc.
	s = strings.ReplaceAll(s, "GRPC", "Grpc")
	s = strings.ReplaceAll(s, "HTTP2", "Http2")
	s = strings.ReplaceAll(s, "HTTP", "Http")
	s = strings.ReplaceAll(s, "ARM64", "Arm64")
	s = strings.ReplaceAll(s, "X86_64", "X8664")

	// Convert snake_case to CamelCase
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

const enumsTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package generated

{{- range .Enums }}
{{- $enumName := .Name }}

// {{ $enumName }} represents an enum type
{{- if .Documentation }}
// {{ .Documentation }}
{{- end }}
type {{ $enumName }} string

// Enum values for {{ $enumName }}
const (
{{- range .EnumValues }}
	{{ $enumName }}{{ toConstantName .Name }} {{ $enumName }} = "{{ .Value }}"
{{- end }}
)

// Values returns all known values for {{ $enumName }}. Note that this can be
// expanded in the future, and so it is only as up to date as the client.
func ({{ $enumName }}) Values() []{{ $enumName }} {
	return []{{ $enumName }}{
{{- range .EnumValues }}
		"{{ .Value }}",
{{- end }}
	}
}
{{- end }}
`

func generateEnums(filename string, enums []TypeDef) error {
	funcMap := template.FuncMap{
		"toCamelCase":    toCamelCase,
		"toConstantName": toConstantName,
	}

	tmpl, err := template.New("enums").Funcs(funcMap).Parse(enumsTemplate)
	if err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, map[string]interface{}{
		"Enums": enums,
	})
}

const routingTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package generated

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// handleRequest is a generic handler for all ECS operations
func handleRequest[TReq any, TResp any](
	serviceFunc func(context.Context, *TReq) (*TResp, error),
	w http.ResponseWriter,
	r *http.Request,
) {
	var req TReq
	if r.Body != nil {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		if len(body) > 0 {
			if err := json.Unmarshal(body, &req); err != nil {
				http.Error(w, "Invalid request format", http.StatusBadRequest)
				return
			}
		}
	}

	resp, err := serviceFunc(r.Context(), &req)
	if err != nil {
		// TODO: Handle specific error types
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HandleECSRequest routes ECS API requests based on X-Amz-Target header
func HandleECSRequest(api ECSAPIInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept POST requests
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract the target operation from X-Amz-Target header
		target := r.Header.Get("X-Amz-Target")
		if target == "" {
			http.Error(w, "Missing X-Amz-Target header", http.StatusBadRequest)
			return
		}

		// Parse the operation name from the target header
		// Format: "AmazonEC2ContainerServiceV20141113.OperationName"
		parts := strings.Split(target, ".")
		if len(parts) != 2 {
			http.Error(w, "Invalid X-Amz-Target format", http.StatusBadRequest)
			return
		}
		operation := parts[1]

		// Route to appropriate handler based on operation
		switch operation {
{{- range .Operations }}
		case "{{ .Name }}":
			handleRequest(api.{{ .Name }}, w, r)
{{- end }}
		default:
			// Return a basic empty response for unsupported operations
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{}"))
		}
	}
}
`

func generateRouting(filename string, operations []Operation) error {
	tmpl, err := template.New("routing").Parse(routingTemplate)
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
