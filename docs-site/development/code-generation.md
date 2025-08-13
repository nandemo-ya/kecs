# Code Generation

KECS uses code generation to create AWS API-compatible types from official AWS API definitions. This approach ensures full compatibility with AWS CLI and SDKs while maintaining type safety.

## Overview

Instead of using AWS SDK directly, KECS generates its own types from AWS Smithy JSON definitions. This provides:

- **AWS CLI Compatibility**: Generated types use proper JSON field names
- **Type Safety**: Compile-time checking for all API operations  
- **No SDK Dependencies**: Reduces binary size and complexity
- **Consistent Interface**: All services follow the same pattern

## Quick Start

### Generate Types for a Service

```bash
# Download AWS API definition
cd controlplane
./scripts/download-aws-api-definitions.sh ecs

# Generate Go code
cd cmd/codegen
go run . -input ecs.json -output ../../internal/ecs/generated
```

### Use Generated Types

```go
import api "github.com/nandemo-ya/kecs/controlplane/internal/ecs/generated"

// Create request
req := &api.CreateClusterRequest{
    ClusterName: aws.String("my-cluster"),
}

// Call API
resp, err := handler.CreateCluster(ctx, req)
```

## Generated Code Structure

Each service generates three files:

### types.go
All request/response types with JSON tags:
```go
type CreateClusterRequest struct {
    ClusterName *string `json:"clusterName,omitempty"`
    Tags []Tag `json:"tags,omitempty"`
}
```

### operations.go
Service interface definition:
```go
type AmazonECSAPI interface {
    CreateCluster(ctx context.Context, input *CreateClusterRequest) (*CreateClusterResponse, error)
    DeleteCluster(ctx context.Context, input *DeleteClusterRequest) (*DeleteClusterResponse, error)
    // ... other operations
}
```

### routing.go
HTTP request routing and marshaling:
```go
func (r *Router) Route(w http.ResponseWriter, req *http.Request) {
    action := r.extractAction(req)
    switch action {
    case "CreateCluster":
        r.handleCreateCluster(w, req)
    // ... other actions
    }
}
```

## Implementing a Service

### 1. Implement the Interface

```go
type ecsHandler struct {
    storage storage.Storage
}

func (h *ecsHandler) CreateCluster(ctx context.Context, input *api.CreateClusterRequest) (*api.CreateClusterResponse, error) {
    // Validate input
    if input.ClusterName == nil || *input.ClusterName == "" {
        return nil, errors.New("cluster name is required")
    }

    // Create cluster in storage
    cluster := &model.Cluster{
        Name:   *input.ClusterName,
        Status: "ACTIVE",
    }
    
    if err := h.storage.CreateCluster(ctx, cluster); err != nil {
        return nil, err
    }

    // Return response
    return &api.CreateClusterResponse{
        Cluster: &api.Cluster{
            ClusterArn:  aws.String(cluster.ARN),
            ClusterName: aws.String(cluster.Name),
            Status:      aws.String(cluster.Status),
        },
    }, nil
}
```

### 2. Set Up HTTP Server

```go
// Create handler
handler := &ecsHandler{
    storage: storage,
}

// Create router
router := api.NewRouter(handler)

// Set up HTTP server
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    router.Route(w, r)
})

http.ListenAndServe(":8080", nil)
```

## Supported Services

| Service | Status | Notes |
|---------|--------|-------|
| ECS | ✅ Complete | All operations implemented |
| STS | ✅ Generated | Types ready for implementation |
| Secrets Manager | ✅ Generated | Types ready for implementation |
| IAM | ⚠️ Partial | Some union types missing |
| CloudWatch Logs | ⚠️ Partial | Union and streaming types missing |
| S3 | ⚠️ Partial | Union and streaming types missing |
| SSM | ⚠️ Partial | Union types missing |

## Adding a New Service

### 1. Download API Definition

```bash
cd controlplane
./scripts/download-aws-api-definitions.sh <service-name>
```

Available services:
- `cloudwatch-logs`
- `iam`
- `s3`
- `secretsmanager`
- `ssm`
- `sts`

### 2. Generate Code

```bash
cd cmd/codegen
go run . -input <service>.json -output ../../internal/<service>/generated
```

### 3. Verify Compilation

```bash
cd ../../internal/<service>/generated
go build ./...
```

### 4. Create Tests

```go
package generated_test

import (
    "encoding/json"
    "testing"
    
    api "github.com/nandemo-ya/kecs/controlplane/internal/<service>/generated"
)

func TestJSONMarshaling(t *testing.T) {
    // Test that JSON fields use camelCase
    req := &api.SomeRequest{
        FieldName: aws.String("value"),
    }
    
    data, _ := json.Marshal(req)
    var m map[string]interface{}
    json.Unmarshal(data, &m)
    
    if _, ok := m["fieldName"]; !ok {
        t.Error("Expected camelCase field name")
    }
}
```

## Troubleshooting

### Common Issues

**1. Compilation Errors**

If generated code doesn't compile, check for:
- Missing union type definitions
- Unsupported streaming types
- Circular dependencies

**2. Missing Types**

Some complex types may not generate correctly:
```bash
# Check the error
go build ./...

# Look for undefined type errors
# Add manual type definitions if needed
```

**3. JSON Field Names**

Verify field names are camelCase:
```bash
# Test with curl
curl -X POST http://localhost:8080/ \
  -H "X-Amz-Target: AmazonECS.ListClusters" \
  -d '{}' | jq .
```

## Advanced Topics

### Custom Type Handling

For types that don't generate correctly, create manual definitions:

```go
// internal/<service>/generated/custom_types.go
package api

// Union type example
type FilterType struct {
    Name   *string
    Values []string
}

// Document type example  
type DocumentValue map[string]interface{}
```

### Extending Generated Code

Never modify generated files directly. Instead:

1. Create wrapper types
2. Use embedding
3. Add helper functions in separate files

```go
// internal/<service>/helpers.go
package service

import api "github.com/nandemo-ya/kecs/controlplane/internal/<service>/generated"

// Helper function
func NewCreateRequest(name string) *api.CreateRequest {
    return &api.CreateRequest{
        Name: aws.String(name),
    }
}
```

## Best Practices

1. **Regenerate Regularly**: Keep API definitions up to date
2. **Test JSON Output**: Verify AWS CLI compatibility
3. **Document Limitations**: Note any missing operations
4. **Use Pointers**: Follow AWS SDK patterns for optional fields
5. **Handle Errors**: Return appropriate AWS error codes

## Next Steps

- [Architecture Overview](./architecture.md) - Understand KECS architecture
- [Building KECS](./building.md) - Build from source
- [Testing Guide](./testing.md) - Write tests for your implementation