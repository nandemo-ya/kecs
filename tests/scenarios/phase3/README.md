# Phase 3: Container Communication Tests

This phase tests basic container communication and functionality within KECS.

## Overview

These tests verify that containers can run successfully within KECS, with proper network connectivity, environment variable handling, and logging.

## Test Scenarios

### Container Communication Tests (`container_communication_test.go`)

1. **Network Connectivity Test**
   - Verifies containers can access external networks
   - Tests basic HTTP connectivity
   - Ensures network isolation works correctly

2. **Environment Variable Test**
   - Verifies environment variables are properly set
   - Tests custom environment variable handling
   - Ensures AWS-related variables are available

3. **Container Logging Test**
   - Verifies container logs are captured
   - Tests multi-line log output
   - Ensures logs are accessible through KECS

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

## Future LocalStack Integration

In future phases, KECS will support:
- Automatic LocalStack deployment within clusters
- Transparent proxy for AWS API calls
- Routing of AWS service calls to LocalStack
- No manual endpoint configuration in containers

The tests verify basic container functionality as a foundation for more advanced features like LocalStack integration in future phases.