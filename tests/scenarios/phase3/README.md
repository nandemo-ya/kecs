# Phase 3: Container Communication Tests

This phase tests basic container communication and functionality within KECS.

## Overview

These tests verify that containers can run successfully within KECS, with proper network connectivity, environment variable handling, and logging.

## Test Scenarios

### Basic Functionality Tests (`basic_functionality_test.go`)

1. **Task Definition Management**
   - Register and describe task definitions
   - Verify task definition persistence

2. **Service Management**
   - Create and list services
   - Basic service lifecycle operations

3. **Task Operations**
   - Run tasks and track their status
   - List and describe running tasks

### LocalStack S3 Proxy Tests (`localstack_s3_proxy_test.go`)

1. **S3 API Proxy Test**
   - Verifies S3 API calls are transparently proxied to LocalStack
   - Tests bucket creation and listing operations
   - Ensures no manual endpoint configuration is needed

2. **Environment-based Proxy Test**
   - Tests S3 operations with proxy environment variables
   - Verifies requests reach LocalStack, not real AWS

3. **Multiple S3 Operations Test**
   - Tests complex S3 workflows (create bucket, upload, list)
   - Verifies all operations are proxied correctly

## Running Tests

```bash
# Run all Phase 3 tests
make test

# Run with verbose output
make test-verbose

# Run a specific test
make test-one TEST=TestSpecificScenario

# Run with race detection
make test-race

# Run with coverage
make test-coverage
```

## Prerequisites

- Docker must be running
- KECS must be properly configured
- Go 1.24.3 installed

## Test Architecture

The tests use:
- Shared KECS container instance
- Shared cluster manager for efficient resource usage
- Basic container images (busybox) for testing
- Standard ECS task definitions

## Key Features Tested

1. **Container Execution**
   - Basic container lifecycle management
   - Task state transitions
   - Error handling and reporting

2. **Network Connectivity**
   - External network access
   - DNS resolution
   - HTTP/HTTPS connectivity

3. **Environment Variables**
   - Variable injection into containers
   - AWS credential handling
   - Custom variable support

4. **Logging**
   - Container output capture
   - Multi-line log handling
   - Log accessibility

## LocalStack Integration

KECS now supports automatic LocalStack integration with:
- Automatic LocalStack deployment within clusters
- Transparent proxy for AWS API calls through Traefik
- Routing of AWS service calls to LocalStack
- No manual endpoint configuration needed in containers

The LocalStack S3 proxy tests verify that:
1. AWS SDK calls from containers are automatically routed to LocalStack
2. No explicit endpoint configuration is required
3. Multiple AWS services can be accessed transparently
4. The proxy works with standard AWS CLI and SDKs