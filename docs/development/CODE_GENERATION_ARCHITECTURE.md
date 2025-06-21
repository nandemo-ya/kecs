# Code Generation Architecture for AWS API Compatibility

## Overview

KECS uses a custom code generation approach to create AWS API-compatible types from AWS Smithy JSON definitions. This architecture was developed to address the AWS CLI compatibility issues discovered when using AWS SDK Go v2 types directly (see [AWS_CLI_COMPATIBILITY_ISSUE.md](./AWS_CLI_COMPATIBILITY_ISSUE.md)).

## Background

### The Problem

AWS SDK Go v2 uses PascalCase field names without JSON tags, which causes AWS CLI to fail silently when parsing responses:

```go
// AWS SDK v2 type (doesn't work with AWS CLI)
type ListClustersOutput struct {
    ClusterArns []string  // Marshals as "ClusterArns", AWS CLI expects "clusterArns"
    NextToken *string
}
```

### The Solution

Generate our own types from AWS API definitions with proper JSON tags:

```go
// Generated type (works with AWS CLI)
type ListClustersResponse struct {
    ClusterArns []string `json:"clusterArns"`
    NextToken   *string  `json:"nextToken,omitempty"`
}
```

## Architecture Components

### 1. AWS API Definitions

AWS provides Smithy JSON files that define all API operations and types. These files are the source of truth for AWS APIs.

**Location**: `controlplane/cmd/codegen/*.json`

**Download Script**: `controlplane/scripts/download-aws-api-definitions.sh`

```bash
# Download API definitions for a service
./scripts/download-aws-api-definitions.sh
```

### 2. Code Generator

The code generator reads Smithy JSON files and generates Go code with proper JSON tags.

**Location**: `controlplane/cmd/codegen/`

**Key Components**:
- `main.go` - Entry point and CLI interface
- `parser/smithy.go` - Parses Smithy JSON format
- `generator/types.go` - Generates Go type definitions
- `generator/operations.go` - Generates service interfaces
- `generator/routing.go` - Generates HTTP routing handlers

**Usage**:
```bash
cd controlplane/cmd/codegen
go run . -input ecs.json -output ../../internal/ecs/generated
```

### 3. Generated Code Structure

For each AWS service, three files are generated:

#### types.go
Contains all request/response types with proper JSON tags:
```go
type CreateClusterRequest struct {
    ClusterName *string `json:"clusterName,omitempty"`
    Tags []Tag `json:"tags,omitempty"`
}
```

#### operations.go
Defines the service interface:
```go
type AmazonECSAPI interface {
    CreateCluster(ctx context.Context, input *CreateClusterRequest) (*CreateClusterResponse, error)
    // ... other operations
}
```

#### routing.go
Handles HTTP routing and request/response marshaling:
```go
func (r *Router) handleCreateCluster(w http.ResponseWriter, req *http.Request) {
    var input CreateClusterRequest
    if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
        writeError(w, http.StatusBadRequest, "InvalidParameterException", err.Error())
        return
    }
    // ... call service implementation
}
```

## Supported AWS Services

| Service | Status | Package Location | Notes |
|---------|--------|------------------|-------|
| ECS | ✅ Complete | `internal/ecs/generated` | Fully functional |
| STS | ✅ Complete | `internal/sts/generated` | All operations compile |
| Secrets Manager | ✅ Complete | `internal/secretsmanager/generated` | All operations compile |
| IAM | ⚠️ Partial | `internal/iam/generated` | Missing some union types |
| CloudWatch Logs | ⚠️ Partial | `internal/cloudwatchlogs/generated` | Missing union and streaming types |
| S3 | ⚠️ Partial | `internal/s3/generated` | Missing union and streaming types |
| SSM | ⚠️ Partial | `internal/ssm/generated` | Missing union types |

## Known Limitations

### 1. Union Types
Smithy union types are not yet implemented. These are used for polymorphic fields in some APIs.

Example from CloudWatch Logs:
```json
"IntegrationDetails": {
    "target": "com.amazonaws.cloudwatchlogs#IntegrationDetails"
}
```

### 2. Streaming Types
Streaming responses (e.g., S3 SelectObjectContent) are not yet supported.

### 3. Document Types
DynamoDB-style document types need special handling.

## Integration with KECS

### 1. Service Implementation

Implement the generated interface in your service handler:

```go
type ecsHandler struct {
    storage storage.Storage
    k8s     kubernetes.Manager
}

func (h *ecsHandler) CreateCluster(ctx context.Context, input *api.CreateClusterRequest) (*api.CreateClusterResponse, error) {
    // Implementation
    return &api.CreateClusterResponse{
        Cluster: &api.Cluster{
            ClusterArn:  aws.String(arn),
            ClusterName: input.ClusterName,
            Status:      aws.String("ACTIVE"),
        },
    }, nil
}
```

### 2. HTTP Server Setup

Wire up the generated router in your HTTP server:

```go
router := api.NewRouter(handler)

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    router.Route(w, r)
})
```

### 3. AWS HTTP Client

For calling other AWS services, use the custom HTTP client with Signature V4:

```go
client := awshttp.NewClient(awshttp.Config{
    Region:      "us-east-1",
    Credentials: creds,
})

// Make API call
resp, err := client.Do(ctx, awshttp.Request{
    Service:   "ecs",
    Operation: "ListClusters",
    Body:      requestBody,
})
```

## Development Workflow

### 1. Adding a New Service

```bash
# 1. Download the API definition
./scripts/download-aws-api-definitions.sh <service-name>

# 2. Generate the code
cd cmd/codegen
go run . -input <service>.json -output ../../internal/<service>/generated

# 3. Check if it compiles
cd ../../internal/<service>/generated
go build ./...
```

### 2. Updating an Existing Service

```bash
# 1. Re-download the latest API definition
./scripts/download-aws-api-definitions.sh <service-name>

# 2. Regenerate the code
cd cmd/codegen
go run . -input <service>.json -output ../../internal/<service>/generated

# 3. Check for breaking changes
git diff ../../internal/<service>/generated/
```

### 3. Testing Generated Code

Create example tests to verify JSON marshaling:

```go
func TestGeneratedTypes(t *testing.T) {
    req := &CreateClusterRequest{
        ClusterName: aws.String("test-cluster"),
    }
    
    data, err := json.Marshal(req)
    require.NoError(t, err)
    
    var jsonMap map[string]interface{}
    err = json.Unmarshal(data, &jsonMap)
    require.NoError(t, err)
    
    // Verify camelCase field names
    assert.Equal(t, "test-cluster", jsonMap["clusterName"])
}
```

## Benefits

1. **AWS CLI Compatibility**: Generated types work correctly with AWS CLI and all AWS SDKs
2. **Type Safety**: Compile-time type checking for all API operations
3. **No SDK Dependencies**: Pure Go types without AWS SDK baggage
4. **Consistent Interface**: All services follow the same pattern
5. **Easy Updates**: Regenerate from latest AWS API definitions

## Future Improvements

1. **Union Type Support**: Implement Smithy union types for full API compatibility
2. **Streaming Support**: Add support for streaming responses
3. **Code Generation Optimizations**: 
   - Remove unused imports automatically
   - Generate only required types (tree shaking)
   - Better handling of recursive types
4. **Validation**: Add request validation based on Smithy constraints
5. **Documentation**: Generate Go doc comments from Smithy documentation

## Related Documents

- [AWS CLI Compatibility Issue](./AWS_CLI_COMPATIBILITY_ISSUE.md) - The original issue that led to this architecture
- [Generated Types Summary](../controlplane/docs/GENERATED_TYPES_SUMMARY.md) - Current status of generated services