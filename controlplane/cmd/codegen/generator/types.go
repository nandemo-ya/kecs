package generator

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/nandemo-ya/kecs/controlplane/cmd/codegen/parser"
)

// TypeInfo holds information about a generated type
type TypeInfo struct {
	Name        string
	GoType      string
	JSONName    string
	IsPointer   bool
	IsRequired  bool
	IsEnum      bool
	IsError     bool
	ErrorType   string // "client" or "server"
	HTTPStatus  int
	EnumValues  []string
	EnumMembers []EnumMember // New: name-value pairs for enums
	Fields      []FieldInfo
}

// EnumMember represents an enum member with its name and value
type EnumMember struct {
	Name  string // Go constant name (e.g., "ACTIVE")
	Value string // Actual value (e.g., "active")
}

// FieldInfo holds information about a struct field
type FieldInfo struct {
	Name       string
	GoType     string
	JSONName   string
	IsPointer  bool
	IsRequired bool
	Comment    string
}

// generateEnumsFile generates the enums.go file
func (g *Generator) generateEnumsFile(api *parser.SmithyAPI) error {
	tmpl := template.Must(template.New("enums").Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	}).Parse(enumsTemplate))

	// Collect only enum types
	allTypes := g.collectTypes(api)
	enumTypes := make(map[string]*TypeInfo)
	var enumNames []string

	for name, typeInfo := range allTypes {
		if typeInfo.IsEnum {
			enumTypes[name] = typeInfo
			enumNames = append(enumNames, name)
		}
	}

	sort.Strings(enumNames)

	// Generate content
	data := struct {
		Package   string
		Service   string
		Types     map[string]*TypeInfo
		TypeNames []string
	}{
		Package:   g.packageName,
		Service:   g.service,
		Types:     enumTypes,
		TypeNames: enumNames,
	}

	content, err := g.executeTemplate(tmpl, data)
	if err != nil {
		return err
	}

	return g.writeFormattedFile("enums.go", content)
}

// generateTypesFile generates the types.go file
func (g *Generator) generateTypesFile(api *parser.SmithyAPI) error {
	tmpl := template.Must(template.New("types").Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	}).Parse(typesTemplate))

	// Collect all types to generate (excluding enums which go in enums.go)
	allTypes := g.collectTypes(api)
	types := make(map[string]*TypeInfo)

	// Filter out enum types
	for name, typeInfo := range allTypes {
		if !typeInfo.IsEnum {
			types[name] = typeInfo
		}
	}

	// Sort types for consistent output
	var typeNames []string
	for name := range types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	// Check if time package is needed
	needsTime := g.needsTimePackage(types)
	needsCommon := g.needsCommonPackage(types)

	// Generate content
	data := struct {
		Package     string
		Service     string
		Types       map[string]*TypeInfo
		TypeNames   []string
		NeedsTime   bool
		NeedsCommon bool
	}{
		Package:     g.packageName,
		Service:     g.service,
		Types:       types,
		TypeNames:   typeNames,
		NeedsTime:   needsTime,
		NeedsCommon: needsCommon,
	}

	content, err := g.executeTemplate(tmpl, data)
	if err != nil {
		return err
	}

	return g.writeFormattedFile("types.go", content)
}

// collectTypes collects all types from the API definition
func (g *Generator) collectTypes(api *parser.SmithyAPI) map[string]*TypeInfo {
	types := make(map[string]*TypeInfo)

	// Process all shapes
	for shapeName, shape := range api.Shapes {
		// Skip service and operation shapes
		if shape.Type == "service" || shape.Type == "operation" {
			continue
		}

		// Get simple name
		name := parser.GetShapeName(shapeName)

		// Skip if already processed
		if _, exists := types[name]; exists {
			continue
		}

		// Generate type info based on shape type
		switch shape.Type {
		case "structure":
			types[name] = g.generateStructType(name, shape, api)
		case "list", "set":
			types[name] = g.generateListType(name, shape, api)
		case "map":
			types[name] = g.generateMapType(name, shape, api)
		case "string":
			if shape.IsEnum() {
				types[name] = g.generateEnumType(name, shape)
			} else {
				types[name] = g.generateStringType(name, shape)
			}
		case "enum":
			types[name] = g.generateEnumType(name, shape)
		case "union":
			types[name] = g.generateUnionType(name, shape, api)
		default:
			// Handle empty shapes and type aliases
			if shape.Type == "" && shape.Target != "" {
				// This is a type alias
				targetShape, _ := api.ResolveShape(shape.Target)
				if targetShape != nil && targetShape.Type == "structure" {
					// For structure aliases, generate a proper struct
					types[name] = g.generateStructType(name, targetShape, api)
				} else {
					// For other aliases, skip to avoid self-referential types
					continue
				}
			} else if shape.IsPrimitive() {
				types[name] = g.generatePrimitiveType(name, shape)
			}
		}
	}

	return types
}

// generateStructType generates type info for a structure
func (g *Generator) generateStructType(name string, shape *parser.SmithyShape, api *parser.SmithyAPI) *TypeInfo {
	typeInfo := &TypeInfo{
		Name:    name,
		GoType:  name,
		IsError: shape.IsError(),
	}

	// Set error type information
	if typeInfo.IsError {
		typeInfo.ErrorType = shape.GetErrorType()
		typeInfo.HTTPStatus = shape.GetHTTPStatus()
	}

	// Check if this is an empty structure
	if shape.Members == nil || len(shape.Members) == 0 {
		// Empty structures should have at least one field to be valid
		// We'll handle this in the template
		return typeInfo
	}

	// Process members
	var fieldNames []string
	for fieldName := range shape.Members {
		fieldNames = append(fieldNames, fieldName)
	}
	sort.Strings(fieldNames)

	for _, fieldName := range fieldNames {
		member := shape.Members[fieldName]
		field := g.generateField(fieldName, member, api)
		typeInfo.Fields = append(typeInfo.Fields, field)
	}

	return typeInfo
}

// generateField generates field info for a struct member
func (g *Generator) generateField(fieldName string, member *parser.SmithyMember, api *parser.SmithyAPI) FieldInfo {
	// Resolve target shape
	targetShape, targetName := api.ResolveShape(member.Target)

	field := FieldInfo{
		Name:       g.exportFieldName(fieldName),
		JSONName:   member.GetJSONName(fieldName),
		IsRequired: member.IsRequired(),
	}

	// Determine Go type with field name context for timestamp detection
	if targetShape != nil {
		field.GoType = g.getGoTypeWithFieldName(targetName, targetShape, api, fieldName)
		// Make pointers for non-required fields (except slices/maps)
		if !field.IsRequired && !strings.HasPrefix(field.GoType, "[]") && !strings.HasPrefix(field.GoType, "map[") {
			field.IsPointer = true
		}
	} else {
		// Default to interface{} if shape not found
		field.GoType = "interface{}"
	}

	return field
}

// generateListType generates type info for a list
func (g *Generator) generateListType(name string, shape *parser.SmithyShape, api *parser.SmithyAPI) *TypeInfo {
	memberType := "interface{}"
	if shape.Member != nil {
		targetShape, targetName := api.ResolveShape(shape.Member.Target)
		if targetShape != nil {
			memberType = g.getGoType(targetName, targetShape, api)
		}
	}

	return &TypeInfo{
		Name:   name,
		GoType: fmt.Sprintf("[]%s", memberType),
	}
}

// generateMapType generates type info for a map
func (g *Generator) generateMapType(name string, shape *parser.SmithyShape, api *parser.SmithyAPI) *TypeInfo {
	keyType := "string" // Default key type
	valueType := "interface{}"

	if shape.Key != nil {
		targetShape, targetName := api.ResolveShape(shape.Key.Target)
		if targetShape != nil {
			keyType = g.getGoType(targetName, targetShape, api)
		}
	}

	if shape.Value != nil {
		targetShape, targetName := api.ResolveShape(shape.Value.Target)
		if targetShape != nil {
			valueType = g.getGoType(targetName, targetShape, api)
		}
	}

	return &TypeInfo{
		Name:   name,
		GoType: fmt.Sprintf("map[%s]%s", keyType, valueType),
	}
}

// generateEnumType generates type info for an enum
func (g *Generator) generateEnumType(name string, shape *parser.SmithyShape) *TypeInfo {
	enumMembers := shape.GetEnumMembers()

	// Convert parser.EnumMember to generator.EnumMember
	genMembers := make([]EnumMember, len(enumMembers))
	enumValues := make([]string, len(enumMembers))
	for i, member := range enumMembers {
		genMembers[i] = EnumMember{
			Name:  member.Name,
			Value: member.Value,
		}
		enumValues[i] = member.Value
	}

	return &TypeInfo{
		Name:        name,
		GoType:      "string",
		IsEnum:      true,
		EnumValues:  enumValues,
		EnumMembers: genMembers,
	}
}

// generateStringType generates type info for a string
func (g *Generator) generateStringType(name string, shape *parser.SmithyShape) *TypeInfo {
	return &TypeInfo{
		Name:   name,
		GoType: "string",
	}
}

// generateUnionType generates type info for a union
func (g *Generator) generateUnionType(name string, shape *parser.SmithyShape, api *parser.SmithyAPI) *TypeInfo {
	typeInfo := &TypeInfo{
		Name:   name,
		GoType: name,
	}

	// Process union members as optional fields
	if shape.Members != nil {
		var fieldNames []string
		for fieldName := range shape.Members {
			fieldNames = append(fieldNames, fieldName)
		}
		sort.Strings(fieldNames)

		for _, fieldName := range fieldNames {
			member := shape.Members[fieldName]
			field := g.generateField(fieldName, member, api)
			// Union members are always optional
			field.IsRequired = false
			field.IsPointer = true

			// Add documentation comment from traits
			if member.Traits != nil {
				if doc, ok := member.Traits["smithy.api#documentation"]; ok {
					if docStr, ok := doc.(string); ok {
						field.Comment = strings.ReplaceAll(docStr, "\n", " ")
						// Clean up HTML tags and excessive whitespace
						field.Comment = strings.TrimSpace(field.Comment)
					}
				}
			}

			typeInfo.Fields = append(typeInfo.Fields, field)
		}
	}

	return typeInfo
}

// generatePrimitiveType generates type info for primitive types
func (g *Generator) generatePrimitiveType(name string, shape *parser.SmithyShape) *TypeInfo {
	goType := g.getPrimitiveGoType(shape.Type)
	return &TypeInfo{
		Name:   name,
		GoType: goType,
	}
}

// getGoType returns the Go type for a shape
func (g *Generator) getGoType(shapeName string, shape *parser.SmithyShape, api *parser.SmithyAPI) string {
	return g.getGoTypeWithFieldName(shapeName, shape, api, "")
}

// getGoTypeWithFieldName gets the Go type for a shape, with special handling for timestamp fields
func (g *Generator) getGoTypeWithFieldName(shapeName string, shape *parser.SmithyShape, api *parser.SmithyAPI, fieldName string) string {
	name := parser.GetShapeName(shapeName)

	// Check if this is a timestamp field that needs UnixTime handling
	// This applies to ECS, SSM and Secrets Manager services for fields containing "Date", "Time", or "At"
	isTimestampField := false
	if fieldName != "" && (g.service == "ecs" || g.service == "ssm" || g.service == "secretsmanager") {
		// Check for common timestamp field name patterns
		lowerFieldName := strings.ToLower(fieldName)
		if strings.Contains(lowerFieldName, "date") || strings.Contains(lowerFieldName, "time") || strings.Contains(lowerFieldName, "at") {
			isTimestampField = true
		}
	}

	// Handle smithy built-in types
	if strings.HasPrefix(shapeName, "smithy.api#") {
		switch shapeName {
		case "smithy.api#Unit":
			return "struct{}"
		case "smithy.api#String":
			return "string"
		case "smithy.api#Integer":
			return "int32"
		case "smithy.api#Long":
			return "int64"
		case "smithy.api#Boolean":
			return "bool"
		case "smithy.api#Timestamp":
			if isTimestampField {
				return "common.UnixTime"
			}
			return "time.Time"
		case "smithy.api#Blob":
			return "[]byte"
		case "smithy.api#Float":
			return "float32"
		case "smithy.api#Double":
			return "float64"
		}
	}

	switch shape.Type {
	case "string":
		if shape.IsEnum() {
			return name
		}
		return "string"
	case "boolean":
		return "bool"
	case "byte":
		return "int8"
	case "short":
		return "int16"
	case "integer":
		return "int32"
	case "long":
		return "int64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "timestamp":
		if isTimestampField {
			return "common.UnixTime"
		}
		return "time.Time"
	case "blob":
		return "[]byte"
	case "list", "set":
		if shape.Member != nil {
			memberShape, memberName := api.ResolveShape(shape.Member.Target)
			if memberShape != nil {
				memberType := g.getGoType(memberName, memberShape, api)
				return fmt.Sprintf("[]%s", memberType)
			}
		}
		return "[]interface{}"
	case "map":
		keyType := "string"
		valueType := "interface{}"
		if shape.Key != nil {
			keyShape, keyName := api.ResolveShape(shape.Key.Target)
			if keyShape != nil {
				keyType = g.getGoType(keyName, keyShape, api)
			}
		}
		if shape.Value != nil {
			valueShape, valueName := api.ResolveShape(shape.Value.Target)
			if valueShape != nil {
				valueType = g.getGoType(valueName, valueShape, api)
			}
		}
		return fmt.Sprintf("map[%s]%s", keyType, valueType)
	case "structure":
		return name
	case "union":
		return name
	case "enum":
		// Handle enum type directly
		return name
	default:
		// Handle type aliases
		if shape.Type == "" && shape.Target != "" {
			targetShape, targetName := api.ResolveShape(shape.Target)
			if targetShape != nil {
				return g.getGoTypeWithFieldName(targetName, targetShape, api, fieldName)
			}
		}
		return "interface{}"
	}
}

// getPrimitiveGoType returns the Go type for primitive Smithy types
func (g *Generator) getPrimitiveGoType(smithyType string) string {
	switch smithyType {
	case "string":
		return "string"
	case "boolean":
		return "bool"
	case "byte":
		return "int8"
	case "short":
		return "int16"
	case "integer":
		return "int32"
	case "long":
		return "int64"
	case "float":
		return "float32"
	case "double":
		return "float64"
	case "bigInteger":
		return "*big.Int"
	case "bigDecimal":
		return "*big.Float"
	case "timestamp":
		return "time.Time"
	case "blob":
		return "[]byte"
	case "document":
		return "interface{}"
	default:
		return "interface{}"
	}
}

// needsTimePackage checks if any type uses time.Time
func (g *Generator) needsTimePackage(types map[string]*TypeInfo) bool {
	for _, typeInfo := range types {
		if strings.Contains(typeInfo.GoType, "time.Time") {
			return true
		}
		for _, field := range typeInfo.Fields {
			if strings.Contains(field.GoType, "time.Time") {
				return true
			}
		}
	}
	return false
}

// needsCommonPackage checks if any type uses common.UnixTime
func (g *Generator) needsCommonPackage(types map[string]*TypeInfo) bool {
	for _, typeInfo := range types {
		if strings.Contains(typeInfo.GoType, "common.UnixTime") {
			return true
		}
		for _, field := range typeInfo.Fields {
			if strings.Contains(field.GoType, "common.UnixTime") {
				return true
			}
		}
	}
	return false
}

// exportFieldName converts a field name to an exported Go field name
func (g *Generator) exportFieldName(name string) string {
	if name == "" {
		return name
	}
	// Convert first character to uppercase
	return strings.ToUpper(name[:1]) + name[1:]
}

const typesTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package {{.Package}}

{{if or .NeedsTime .NeedsCommon}}
import (
{{if .NeedsCommon}}	"github.com/nandemo-ya/kecs/controlplane/internal/common"
{{end}}{{if .NeedsTime}}	"time"
{{end}})
{{end}}

// Unit represents an empty response
type Unit = struct{}

{{range $name := .TypeNames}}
{{$type := index $.Types $name}}
{{if eq $type.GoType $type.Name}}
// {{$type.Name}} represents the {{$type.Name}} structure
type {{$type.Name}} struct {
{{range $field := $type.Fields}}
{{if $field.Comment}}	// {{$field.Comment}}
{{end}}	{{$field.Name}} {{if $field.IsPointer}}*{{end}}{{$field.GoType}} ` + "`" + `json:"{{$field.JSONName}}{{if not $field.IsRequired}},omitempty{{end}}"` + "`" + `
{{end}}}
{{if $type.IsError}}

// Error implements the error interface for {{$type.Name}}
func (e {{$type.Name}}) Error() string {
	return "{{$type.Name}}: AWS {{$type.ErrorType}} error{{if $type.HTTPStatus}} (HTTP {{$type.HTTPStatus}}){{end}}"
}

// ErrorCode returns the AWS error code
func (e {{$type.Name}}) ErrorCode() string {
	return "{{$type.Name}}"
}

// ErrorFault indicates whether this is a client or server error
func (e {{$type.Name}}) ErrorFault() string {
	return "{{$type.ErrorType}}"
}
{{end}}
{{else}}
// {{$type.Name}} represents the {{$type.Name}} type
type {{$type.Name}} {{$type.GoType}}
{{end}}

{{end}}
`

const enumsTemplate = `// Code generated by cmd/codegen. DO NOT EDIT.

package {{.Package}}

{{range $name := .TypeNames}}
{{$type := index $.Types $name}}
// {{$type.Name}} represents the {{$type.Name}} enum type
type {{$type.Name}} string

const (
{{range $member := $type.EnumMembers}}
	{{$type.Name}}{{$member.Name}} {{$type.Name}} = "{{$member.Value}}"
{{end}}
)

{{end}}
`
