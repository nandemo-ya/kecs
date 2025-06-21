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
	Name       string
	GoType     string
	JSONName   string
	IsPointer  bool
	IsRequired bool
	IsEnum     bool
	EnumValues []string
	Fields     []FieldInfo
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

// generateTypesFile generates the types.go file
func (g *Generator) generateTypesFile(api *parser.SmithyAPI) error {
	tmpl := template.Must(template.New("types").Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	}).Parse(typesTemplate))

	// Collect all types to generate
	types := g.collectTypes(api)
	
	// Sort types for consistent output
	var typeNames []string
	for name := range types {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	// Generate content
	data := struct {
		Package string
		Service string
		Types   map[string]*TypeInfo
		TypeNames []string
	}{
		Package: g.packageName,
		Service: g.service,
		Types:   types,
		TypeNames: typeNames,
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
			// TODO: Handle union types
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
		Name:   name,
		GoType: name,
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
	
	// Determine Go type
	if targetShape != nil {
		field.GoType = g.getGoType(targetName, targetShape, api)
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
	return &TypeInfo{
		Name:       name,
		GoType:     "string",
		IsEnum:     true,
		EnumValues: shape.GetEnumValues(),
	}
}

// generateStringType generates type info for a string
func (g *Generator) generateStringType(name string, shape *parser.SmithyShape) *TypeInfo {
	return &TypeInfo{
		Name:   name,
		GoType: "string",
	}
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
	name := parser.GetShapeName(shapeName)
	
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
		return name // TODO: Handle union types properly
	default:
		// Handle type aliases
		if shape.Type == "" && shape.Target != "" {
			targetShape, targetName := api.ResolveShape(shape.Target)
			if targetShape != nil {
				return g.getGoType(targetName, targetShape, api)
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

import (
	"time"
)

// Unit represents an empty response
type Unit = struct{}

{{range $name := .TypeNames}}
{{$type := index $.Types $name}}
{{if $type.IsEnum}}
// {{$type.Name}} represents the {{$type.Name}} enum type
type {{$type.Name}} string

const (
{{range $value := $type.EnumValues}}
	{{$type.Name}}{{$value}} {{$type.Name}} = "{{$value}}"
{{end}}
)
{{else if eq $type.GoType $type.Name}}
// {{$type.Name}} represents the {{$type.Name}} structure
type {{$type.Name}} struct {
{{range $field := $type.Fields}}
	{{$field.Name}} {{if $field.IsPointer}}*{{end}}{{$field.GoType}} ` + "`" + `json:"{{$field.JSONName}}{{if not $field.IsRequired}},omitempty{{end}}"` + "`" + `
{{end}}
}
{{else}}
// {{$type.Name}} represents the {{$type.Name}} type
type {{$type.Name}} {{$type.GoType}}
{{end}}

{{end}}
`