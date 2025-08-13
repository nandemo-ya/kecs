# Code Generation Architecture

This document describes the architecture and design decisions behind KECS's AWS API code generation system.

## Overview

KECS implements a comprehensive code generation system that automatically creates AWS API types and handlers from official AWS Smithy JSON specifications. This approach provides AWS CLI compatibility while eliminating dependencies on AWS SDK v2.

## Architecture Components

### 1. Smithy Parser (`cmd/codegen/parser/smithy.go`)

The parser component reads and interprets AWS Smithy JSON API definitions:

```
Input: AWS API Definition (e.g., s3.json, ecs.json)
  ↓
Smithy Parser
  ↓
Output: Structured API Model (SmithyAPI)
```

**Key Features:**
- Parses Smithy JSON format used by AWS SDK Go v2
- Extracts service definitions, operations, and type information
- Handles traits for documentation, validation, and metadata
- Supports all Smithy built-in types and custom shapes

**Core Types:**
```go
type SmithyAPI struct {
    Smithy   string                 // Smithy version
    Metadata map[string]interface{} // Service metadata
    Shapes   map[string]*SmithyShape // All type definitions
}

type SmithyShape struct {
    Type       string                 // structure, union, list, enum, etc.
    Members    map[string]*SmithyMember // Struct/union members
    Traits     map[string]interface{} // Smithy traits
    // ... additional fields
}
```

### 2. Code Generator (`cmd/codegen/generator/`)

The generator transforms parsed Smithy models into Go code:

```
SmithyAPI Model
  ↓
Type Generator → types.go (Data structures)
  ↓
Operations Generator → operations.go (Interface definitions)
  ↓
Routing Generator → routing.go (HTTP handlers)
  ↓
Client Generator → client.go (HTTP client - optional)
```

#### Type Generation (`generator/types.go`)

Converts Smithy shapes into Go type definitions:

- **Structures** → Go structs with JSON tags
- **Unions** → Go structs with optional pointer fields
- **Enums** → Go string types with constants
- **Lists/Sets** → Go slices
- **Maps** → Go maps
- **Primitives** → Go basic types

**Special Handling:**
- Union types: All members as optional pointers
- Error types: Implement Go `error` interface
- Optional fields: Use pointers for omitempty support
- JSON naming: camelCase for AWS CLI compatibility

#### Operations Generation (`generator/operations.go`)

Creates Go interfaces for each AWS service operation:

```go
type ECSOperations interface {
    CreateCluster(ctx context.Context, req *CreateClusterRequest) (*CreateClusterOutput, error)
    ListClusters(ctx context.Context, req *ListClustersRequest) (*ListClustersOutput, error)
    // ... all operations
}
```

#### Routing Generation (`generator/routing.go`)

Generates HTTP routing logic for AWS API endpoints:

- Maps AWS service operations to HTTP endpoints
- Handles X-Amz-Target headers for JSON APIs
- Provides request/response serialization helpers
- Supports both REST and JSON protocol APIs

### 3. Template System

Uses Go's `text/template` for code generation with shared templates:

- **Consistent Formatting**: All generated code follows same patterns
- **Maintainable**: Templates easier to modify than string concatenation
- **Extensible**: Easy to add new output formats

## Smithy Type Mapping

### Basic Types

| Smithy Type | Go Type | Notes |
|-------------|---------|-------|
| `string` | `string` | Basic string type |
| `boolean` | `bool` | Boolean values |
| `integer` | `int32` | 32-bit integers |
| `long` | `int64` | 64-bit integers |
| `float` | `float32` | 32-bit floats |
| `double` | `float64` | 64-bit floats |
| `timestamp` | `time.Time` | RFC3339 timestamps |
| `blob` | `[]byte` | Binary data |

### Complex Types

| Smithy Type | Go Type | Implementation |
|-------------|---------|----------------|
| `structure` | `struct` | Go struct with JSON tags |
| `union` | `struct` | Struct with optional pointer fields |
| `list` | `[]T` | Go slice of element type |
| `set` | `[]T` | Go slice (sets not natively supported) |
| `map` | `map[K]V` | Go map with key/value types |
| `enum` | `string` + constants | String type with const definitions |

### Special Cases

**Union Types:**
```smithy
union MyUnion {
    fieldA: String,
    fieldB: Integer,
    fieldC: ComplexType
}
```

Generates:
```go
type MyUnion struct {
    FieldA *string      `json:"fieldA,omitempty"`
    FieldB *int32       `json:"fieldB,omitempty"` 
    FieldC *ComplexType `json:"fieldC,omitempty"`
}
```

**Error Types:**
```smithy
structure MyError {
    message: String
}
@error("client")
@httpError(404)
```

Generates:
```go
type MyError struct {
    Message *string `json:"message,omitempty"`
}

func (e MyError) Error() string {
    return "MyError: AWS client error (HTTP 404)"
}

func (e MyError) ErrorCode() string {
    return "MyError"
}

func (e MyError) ErrorFault() string {
    return "client"
}
```

## JSON Field Naming Strategy

KECS uses camelCase for JSON field names to maintain AWS CLI compatibility:

```go
type ContainerDefinition struct {
    ContainerName *string `json:"containerName,omitempty"` // camelCase
    ImageUri      *string `json:"imageUri,omitempty"`     // camelCase
}
```

This ensures generated requests/responses work seamlessly with AWS CLI and other AWS tools.

## Code Generation Pipeline

### 1. API Definition Download

```bash
./scripts/download-aws-api-definitions.sh
```

Downloads latest AWS API definitions from AWS SDK Go v2 source:
- Extracts Smithy JSON files from SDK packages
- Supports multiple AWS services (ECS, S3, IAM, etc.)
- Maintains consistent versioning

### 2. Generation Process

```bash
go run cmd/codegen/main.go -service s3 -input s3.json -output internal/s3/generated
```

Generation steps:
1. Parse Smithy JSON → `SmithyAPI` model
2. Collect and organize type definitions
3. Generate Go source files from templates
4. Format code with `gofmt`
5. Write output files

### 3. Integration

Generated code integrates into KECS via:
- Import paths: `internal/{service}/generated`
- Interface implementations in API handlers
- HTTP routing in server setup
- Type conversions where needed

## Extensibility

### Adding New Services

1. **Download API Definition:**
   ```bash
   # Add service to download script
   SERVICES=("ecs" "s3" "new-service")
   ```

2. **Generate Types:**
   ```bash
   go run cmd/codegen/main.go -service new-service -input new-service.json -output internal/new-service/generated
   ```

3. **Implement Integration:**
   - Create service integration package
   - Implement generated interfaces
   - Add routing configuration

### Customizing Generation

**Template Modifications:**
- Edit templates in `generator/templates/`
- Add new template functions
- Customize output formatting

**Parser Extensions:**
- Add new Smithy trait handlers
- Support additional type mappings
- Extend validation logic

**Generator Features:**
- Add new output file types
- Implement custom naming strategies
- Add validation generation

## Design Decisions

### 1. Smithy JSON as Source of Truth

**Decision:** Use AWS SDK Go v2's Smithy JSON definitions rather than original Smithy files.

**Rationale:**
- Preprocessed and validated by AWS
- Includes all necessary metadata
- Same source used by official AWS SDKs
- Easier parsing than raw Smithy syntax

### 2. Union Types as Structs

**Decision:** Implement Smithy unions as Go structs with optional pointer fields.

**Rationale:**
- Go doesn't have native union types
- Pointer fields allow detecting which variant is set
- Compatible with JSON marshaling/unmarshaling
- Type-safe at compilation

**Alternative Considered:** Interface-based unions
**Rejected:** More complex, less JSON-friendly

### 3. camelCase JSON Fields

**Decision:** Use camelCase for all JSON field names.

**Rationale:**
- AWS CLI expects camelCase
- Consistent with AWS API conventions
- Better compatibility with AWS tools
- Matches AWS SDK Go v2 behavior

### 4. Error Interface Implementation

**Decision:** All error types implement Go's `error` interface.

**Rationale:**
- Idiomatic Go error handling
- Enables type-safe error checking
- Preserves AWS error metadata
- Compatible with existing error handling patterns

### 5. Code Generation vs Runtime Reflection

**Decision:** Generate code at build time rather than using runtime reflection.

**Rationale:**
- Better performance (no reflection overhead)
- Compile-time type safety
- Easier debugging and IDE support
- Smaller binary size
- More predictable behavior

## Performance Considerations

### Generation Time

- Large APIs (S3) take ~1-2 seconds to generate
- Incremental generation not implemented (full regeneration)
- Template compilation happens once per service

### Runtime Performance

- Zero runtime overhead for type definitions
- No reflection or dynamic type creation
- Direct struct field access
- Efficient JSON marshaling/unmarshaling

### Memory Usage

- Generated types have minimal memory overhead
- Pointer fields only allocated when needed
- No hidden allocations or caching

## Future Enhancements

### Planned Features

1. **Validation Generation:**
   - Generate field validation based on Smithy constraints
   - Required field checking
   - Value range validation

2. **Documentation Generation:**
   - Extract API documentation from Smithy traits
   - Generate godoc comments
   - Create API reference documentation

3. **Client Generation:**
   - HTTP client implementations for each service
   - Request signing and authentication
   - Retry logic and error handling

4. **Performance Optimizations:**
   - Incremental generation (only changed types)
   - Parallel generation for multiple services
   - Template pre-compilation

### Potential Extensions

1. **OpenAPI Generation:**
   - Generate OpenAPI 3.0 specifications
   - Support for API documentation tools
   - Client generation for other languages

2. **Mock Generation:**
   - Generate mock implementations for testing
   - Configurable response generation
   - Integration with testing frameworks

3. **Validation Framework:**
   - Runtime validation based on Smithy constraints
   - Custom validation rules
   - Error message customization

## Testing Strategy

### Generator Testing

- Unit tests for parser components
- Template rendering tests
- Integration tests with real AWS API definitions
- Golden file testing for generated output

### Generated Code Testing

- Compilation tests for all generated types
- JSON marshaling/unmarshaling tests
- Error interface implementation tests
- Integration tests with AWS CLI compatibility

### Regression Testing

- Compare generated output across versions
- Validate against AWS API changes
- Ensure backward compatibility

This architecture provides a robust foundation for AWS API compatibility while maintaining the flexibility to adapt to future AWS service changes and KECS requirements.