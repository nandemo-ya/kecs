# Migration from AWS SDK v2 to Generated Types

This document describes the migration from aws-sdk-go-v2 to our custom generated types and HTTP client.

## Overview

We've created a code generation tool that:
1. Parses AWS API definition files (Smithy JSON format)
2. Generates Go types with proper JSON tags for AWS CLI compatibility
3. Provides a custom AWS HTTP client with signature v4 support

## Key Components

### 1. Code Generation Tool (`cmd/codegen/`)
- Parses AWS service definitions from JSON files
- Generates Go structs with camelCase JSON tags
- Handles special cases like empty response types
- Supports all AWS data types (strings, numbers, lists, maps, etc.)

### 2. AWS HTTP Client (`internal/awsclient/`)
- Implements AWS Signature Version 4
- Handles retries and exponential backoff
- Supports custom endpoints for local testing
- No dependency on aws-sdk-go-v2

### 3. Generated Types (`internal/controlplane/api/generated_v2/`)
- All ECS API types with proper JSON marshaling
- Compatible with AWS CLI's case-sensitive requirements
- Matches AWS API documentation exactly

## Migration Steps

### Step 1: Replace SDK Imports

Before:
```go
import (
    "github.com/aws/aws-sdk-go-v2/service/ecs"
    "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)
```

After:
```go
import (
    generated_v2 "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated_v2"
    "github.com/nandemo-ya/kecs/controlplane/internal/awsclient"
)
```

### Step 2: Update Type Usage

Before:
```go
// Using SDK types
req := &ecs.CreateClusterInput{
    ClusterName: aws.String("my-cluster"),
}
```

After:
```go
// Using generated types
req := &generated_v2.CreateClusterRequest{
    ClusterName: stringPtr("my-cluster"),
}
```

### Step 3: Replace Client Calls

Before:
```go
// Using SDK client
client := ecs.NewFromConfig(cfg)
resp, err := client.CreateCluster(ctx, req)
```

After:
```go
// Using custom client
client := awsclient.New(awsclient.Config{
    EndpointURL: endpoint,
    Region:      "ap-northeast-1",
    Service:     "ecs",
})

reqBody, _ := json.Marshal(req)
respBody, err := client.Do(ctx, "CreateCluster", reqBody)

var resp generated_v2.CreateClusterResponse
json.Unmarshal(respBody, &resp)
```

## Example Implementation

See `cmd/example-generated/main.go` for a complete example of using the generated types and custom client.

## Benefits

1. **AWS CLI Compatibility**: Generated types use camelCase JSON tags, ensuring compatibility with case-sensitive clients
2. **No SDK Dependencies**: Removes the entire aws-sdk-go-v2 dependency tree
3. **Smaller Binary Size**: Significantly reduces binary size by avoiding SDK overhead
4. **Custom Control**: Full control over HTTP client behavior and error handling
5. **Local Testing**: Easy to test against local endpoints without SDK configuration complexity

## Future Work

1. Generate types for other AWS services (IAM, S3, CloudWatch, etc.)
2. Add streaming support for CloudWatch Logs
3. Implement service-specific error types
4. Add request/response middleware support
5. Create migration tool to automatically update existing code

## Testing

The generated types maintain full compatibility with AWS APIs. All existing tests should continue to work after migration, requiring only import and client initialization changes.