# Hybrid Test Client for KECS Scenario Tests

This document describes the hybrid test client implementation that supports both `curl` and AWS CLI for testing KECS.

## Overview

The hybrid test client allows scenario tests to run using either:
- **Curl mode** (default): Direct HTTP calls using curl commands
- **AWS CLI mode**: Using the official AWS CLI tool

This approach ensures:
1. Backward compatibility with existing tests
2. Validation of AWS CLI compatibility
3. Better test coverage by testing both access methods

## Usage

### Basic Usage

```go
// Default mode (curl)
client := utils.NewECSClient(endpoint)

// Explicit curl mode
client := utils.NewECSClient(endpoint, utils.CurlMode)

// AWS CLI mode
client := utils.NewECSClient(endpoint, utils.AWSCLIMode)
```

### Running Tests with Both Clients

Use the `TestWithBothClients` helper to automatically test with both modes:

```go
func TestMyFeature(t *testing.T) {
    utils.TestWithBothClients(t, "MyFeature", func(t *testing.T, client utils.ECSClientInterface, mode utils.ClientMode) {
        // Your test code here
        err := client.CreateCluster("test-cluster")
        require.NoError(t, err)
    })
}
```

### Environment Variables

- `TEST_WITH_AWS_CLI=true`: Enable AWS CLI tests (disabled by default)
- `USE_AWS_CLI=true`: Use AWS CLI as the default client mode

### Running Tests

```bash
# Run tests with curl only (default)
make test

# Run tests with both curl and AWS CLI
TEST_WITH_AWS_CLI=true make test

# Run tests with AWS CLI as default
USE_AWS_CLI=true make test
```

## Implementation Details

### Client Interface

All clients implement the `ECSClientInterface`:

```go
type ECSClientInterface interface {
    CreateCluster(name string) error
    DescribeCluster(name string) (*Cluster, error)
    ListClusters() ([]string, error)
    DeleteCluster(name string) error
    // ... other ECS operations
}
```

### CurlClient

- Uses direct HTTP calls with curl
- Fast and reliable for local testing
- No external dependencies

### AWSCLIClient

- Uses the official AWS CLI
- Validates real-world AWS CLI compatibility
- Requires AWS CLI v2 to be installed
- Uses dummy credentials for local testing

## Benefits

1. **Comprehensive Testing**: Tests both direct API access and AWS CLI access
2. **Real-world Validation**: Ensures KECS works with actual AWS CLI commands
3. **Backward Compatibility**: Existing tests continue to work without changes
4. **Flexible**: Easy to add new client implementations in the future

## Limitations

### AWS CLI Mode Limitations

Some operations are not directly supported by AWS CLI commands:
- `PutAttributes`
- `ListAttributes`
- `DeleteAttributes`

These operations return an error in AWS CLI mode.

## Future Enhancements

1. Add AWS SDK Go v2 client implementation
2. Support for performance comparison between clients
3. Automatic retry logic for transient failures
4. Better error reporting and diagnostics