# Testing with Generated Types

This document describes how to use the generated types mode in test scenarios.

## Overview

KECS test scenarios support three client modes:
1. **CurlMode** (default) - Uses curl commands for direct HTTP API calls
2. **AWSCLIMode** - Uses AWS CLI commands  
3. **GeneratedMode** - Uses generated types with custom AWS client (NEW)

## Using Generated Mode

To use generated types in your tests:

```go
// Create client with generated mode
client := utils.NewECSClientInterface(kecs.Endpoint(), utils.GeneratedMode)

// Use the client normally
err := client.CreateCluster("test-cluster")
```

## Example Test

See `cluster/cluster_generated_test.go` for a complete example of testing with generated types.

## Benefits

1. **Type Safety**: Generated types provide compile-time type checking
2. **AWS CLI Compatibility**: Generated types use camelCase JSON tags
3. **No SDK Dependencies**: Removes dependency on aws-sdk-go-v2

## Current Implementation Status

### Implemented Operations
- ✅ CreateCluster
- ✅ DescribeCluster  
- ✅ ListClusters
- ✅ DeleteCluster

### TODO Operations
- ⏳ RegisterTaskDefinition
- ⏳ DescribeTaskDefinition
- ⏳ ListTaskDefinitions
- ⏳ CreateService
- ⏳ DescribeService
- ⏳ UpdateService
- ⏳ DeleteService
- ⏳ RunTask
- ⏳ DescribeTasks
- ⏳ StopTask
- ⏳ TagResource
- ⏳ UntagResource

## Migration Guide

To migrate existing tests to use generated types:

1. Change client creation:
```go
// Before
client := utils.NewECSClient(endpoint)

// After  
client := utils.NewECSClientInterface(endpoint, utils.GeneratedMode)
```

2. Update type assertions if needed (most operations return the same interface)

3. Run tests to ensure compatibility

## Integration Tests

The generated types are tested in:
- `controlplane/internal/controlplane/api/generated_integration_test.go` - Unit tests
- `tests/scenarios/cluster/cluster_generated_test.go` - E2E tests with real KECS container