# LocalStack Integration Tests

This directory contains integration tests for KECS LocalStack functionality.

## Overview

These tests verify that KECS correctly integrates with LocalStack to provide local AWS service emulation:

- **Lifecycle Tests**: LocalStack start/stop/restart and service management
- **AWS API Proxy Tests**: Routing of AWS API calls to LocalStack
- **Environment Injection Tests**: AWS SDK configuration in ECS tasks

## Prerequisites

- Docker installed and running
- Go 1.21+ with Ginkgo test framework
- KECS test image built (run `make build-image` in parent directory)

## Running Tests

### Run All Tests
```bash
make test
```

### Run Specific Test Suites
```bash
# Lifecycle tests only
make test-lifecycle

# AWS API proxy tests only
make test-proxy

# Environment injection tests only
make test-env
```

### Run with Debug Output
```bash
make test-verbose
```

### Run a Specific Test
```bash
make test-one TEST="should start LocalStack successfully"
```

## Test Structure

```
localstack/
├── localstack_suite_test.go    # Test suite setup
├── lifecycle_test.go           # LocalStack lifecycle management tests
├── aws_proxy_test.go          # AWS API routing tests
├── env_injection_test.go      # Environment variable injection tests
├── helpers/
│   └── localstack_helpers.go  # Test helper functions
├── Makefile                   # Test automation
└── README.md                  # This file
```

## Test Implementation Notes

### Lifecycle Tests
- Verify LocalStack container management
- Test service enable/disable functionality
- Check health monitoring
- Validate restart behavior

### AWS API Proxy Tests
- Test S3 API calls routed to LocalStack
- Test IAM API calls routed to LocalStack
- Verify ECS calls go to KECS, not LocalStack
- Check error handling for disabled services

### Environment Injection Tests
- Verify AWS SDK environment variables are injected
- Test multiple tasks get consistent configuration
- Ensure user environment variables are preserved
- Validate service task configuration

## Known Limitations

1. **GetLocalStackStatus**: The helper method assumes a KECS-specific endpoint that may not be implemented yet
2. **Container Logs**: Accessing task logs to verify environment variables requires additional implementation
3. **LocalStack Endpoint**: Tests assume LocalStack is accessible via a specific endpoint pattern

## Future Enhancements

1. Add tests for Phase 2 features (IAM integration, CloudWatch Logs, etc.)
2. Test sidecar proxy mode when implemented
3. Add performance benchmarks
4. Test LocalStack persistence across restarts
5. Add tests for custom LocalStack configuration