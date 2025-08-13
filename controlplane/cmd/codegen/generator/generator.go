package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"github.com/nandemo-ya/kecs/controlplane/cmd/codegen/parser"
)

// Generator handles code generation for AWS services
type Generator struct {
	service     string
	packageName string
	outputDir   string
	templates   map[string]*template.Template
}

// New creates a new code generator
func New(service, packageName, outputDir string) *Generator {
	return &Generator{
		service:     service,
		packageName: packageName,
		outputDir:   outputDir,
		templates:   make(map[string]*template.Template),
	}
}

// GenerateTypes generates type definitions from the API definition
func (g *Generator) GenerateTypes(api *parser.SmithyAPI) error {
	// Generate both types.go and enums.go
	if err := g.generateTypesFile(api); err != nil {
		return err
	}
	return g.generateEnumsFile(api)
}

// GenerateOperations generates operation interfaces from the API definition
func (g *Generator) GenerateOperations(api *parser.SmithyAPI) error {
	return g.generateOperationsFile(api)
}

// GenerateRouting generates HTTP routing code from the API definition
func (g *Generator) GenerateRouting(api *parser.SmithyAPI) error {
	return g.generateRoutingFile(api)
}

// GenerateClient generates HTTP client code from the API definition
func (g *Generator) GenerateClient(api *parser.SmithyAPI) error {
	return g.generateClientFile(api)
}

// writeFormattedFile writes formatted Go code to a file
func (g *Generator) writeFormattedFile(filename string, content []byte) error {
	formatted, err := format.Source(content)
	if err != nil {
		// If formatting fails, write unformatted for debugging
		if writeErr := os.WriteFile(filepath.Join(g.outputDir, filename+".unformatted"), content, 0644); writeErr != nil {
			return fmt.Errorf("failed to write unformatted file: %w", writeErr)
		}
		return fmt.Errorf("failed to format Go code: %w (unformatted version saved)", err)
	}

	fullPath := filepath.Join(g.outputDir, filename)
	if err := os.WriteFile(fullPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// executeTemplate executes a template with the given data
func (g *Generator) executeTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.Bytes(), nil
}
