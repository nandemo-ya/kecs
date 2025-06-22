package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/nandemo-ya/kecs/controlplane/cmd/codegen/generator"
	"github.com/nandemo-ya/kecs/controlplane/cmd/codegen/parser"
)

var (
	service    = flag.String("service", "ecs", "AWS service name")
	input      = flag.String("input", "", "Path to AWS API definition JSON file")
	output     = flag.String("output", "", "Output directory for generated code")
	pkgName    = flag.String("package", "generated", "Go package name for generated code")
	genTypes   = flag.Bool("types", true, "Generate type definitions")
	genOps     = flag.Bool("operations", true, "Generate operation interfaces")
	genRouting = flag.Bool("routing", true, "Generate HTTP routing")
	genClient  = flag.Bool("client", false, "Generate HTTP client")
)

func main() {
	flag.Parse()

	if *input == "" {
		// Default to api-models/<service>.json
		*input = filepath.Join("api-models", fmt.Sprintf("%s.json", *service))
	}

	if *output == "" {
		// Default to internal/controlplane/api/generated
		*output = "internal/controlplane/api/generated"
	}

	// Check if input file exists
	if _, err := os.Stat(*input); os.IsNotExist(err) {
		log.Fatalf("Input file does not exist: %s", *input)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*output, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Parse API definition
	log.Printf("Parsing API definition from %s", *input)
	apiDef, err := parser.ParseSmithyJSON(*input)
	if err != nil {
		log.Fatalf("Failed to parse API definition: %v", err)
	}

	// Create generator
	gen := generator.New(*service, *pkgName, *output)

	// Generate requested files
	if *genTypes {
		log.Println("Generating type definitions...")
		if err := gen.GenerateTypes(apiDef); err != nil {
			log.Fatalf("Failed to generate types: %v", err)
		}
	}

	if *genOps {
		log.Println("Generating operation interfaces...")
		if err := gen.GenerateOperations(apiDef); err != nil {
			log.Fatalf("Failed to generate operations: %v", err)
		}
	}

	if *genRouting {
		log.Println("Generating HTTP routing...")
		if err := gen.GenerateRouting(apiDef); err != nil {
			log.Fatalf("Failed to generate routing: %v", err)
		}
	}

	if *genClient {
		log.Println("Generating HTTP client...")
		if err := gen.GenerateClient(apiDef); err != nil {
			log.Fatalf("Failed to generate client: %v", err)
		}
	}

	log.Println("Code generation completed successfully")
}