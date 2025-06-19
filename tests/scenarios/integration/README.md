# KECS Integration Test Suite

This directory contains comprehensive integration tests for KECS (Kubernetes-based ECS Compatible Service).

## Test Categories

### Basic Integration Tests (`basic/`)

Core ECS API functionality and basic resource lifecycle tests:

- **cluster_lifecycle_test.go**: Cluster creation, deletion, attributes, and tagging
- **service_lifecycle_test.go**: Service CRUD operations, scaling, and updates
- **localstack_integration_test.go**: LocalStack-specific integrations (S3, SSM, CloudWatch)

### Advanced Integration Tests (`advanced/`)

Complex deployment scenarios and advanced features:

- **multi_service_test.go**: Multi-tier application deployment and service interactions
- **networking_test.go**: Network configuration scenarios (bridge, host, awsvpc modes)
- **service_discovery_test.go**: Service discovery and DNS-based communication tests
- **failure_scenarios_test.go**: Failure recovery, resource constraints, and resilience testing

### Performance Tests (`performance/`)

Load testing, scalability, and performance benchmarks:

- **cluster_scaling_test.go**: Service creation performance, scaling performance, resource utilization
- **api_performance_test.go**: API response times, throughput tests, large payload handling

## Test Structure

```
integration/
├── basic/
│   ├── cluster_lifecycle_test.go
│   ├── service_lifecycle_test.go
│   ├── task_lifecycle_test.go
│   └── localstack_integration_test.go
├── advanced/
│   ├── multi_service_test.go
│   ├── networking_test.go
│   ├── service_discovery_test.go
│   └── failure_scenarios_test.go
├── performance/
│   ├── load_test.go
│   ├── scalability_test.go
│   └── benchmark_test.go
└── utils/
    ├── test_helpers.go
    ├── performance_utils.go
    └── assertions.go
```

## Running Tests

### All Integration Tests
```bash
cd tests/scenarios/integration
go test -v ./...
```

### Specific Test Categories
```bash
# Basic tests
go test -v ./basic/...

# Advanced tests
go test -v ./advanced/...

# Performance tests
go test -v -timeout 30m ./performance/...
```

### With Tags
```bash
# Run only fast tests
go test -v -tags=fast ./...

# Run only slow tests (includes performance)
go test -v -tags=slow ./...
```

## Test Requirements

### Prerequisites
- Docker and Docker Compose
- Go 1.21+
- KECS binary or Docker image
- LocalStack (automatically started)

### Environment Variables
- `KECS_IMAGE`: Docker image for KECS (default: kecs:test)
- `KECS_LOG_LEVEL`: Log level (debug, info, warn, error)
- `TEST_TIMEOUT`: Test timeout duration (default: 10m)
- `LOCALSTACK_VERSION`: LocalStack version (default: latest)

### Test Configuration
```yaml
# test-config.yaml
test:
  timeout: 10m
  parallel: true
  cleanup: true
  
kecs:
  image: kecs:test
  logLevel: debug
  
localstack:
  version: latest
  services:
    - iam
    - s3
    - dynamodb
    - sqs
    - sns
    - logs
    - servicediscovery
```

## Test Scenarios

### Basic Integration
1. **Cluster Management**
   - Create/delete clusters
   - List clusters
   - Cluster attributes

2. **Service Management**
   - Create/update/delete services
   - Service scaling
   - Service discovery registration

3. **Task Management**
   - Run tasks
   - Task lifecycle
   - Task networking

4. **LocalStack Integration**
   - S3 integration
   - DynamoDB integration
   - CloudWatch Logs

### Advanced Integration
1. **Multi-Service Applications**
   - Service dependencies
   - Inter-service communication
   - Load balancing

2. **Networking Scenarios**
   - VPC configuration
   - Security groups
   - Service mesh patterns

3. **Service Discovery**
   - Cloud Map integration
   - DNS resolution
   - Health checks

4. **Failure Scenarios**
   - Service failures
   - Network partitions
   - Resource constraints

### Performance Testing
1. **Load Testing**
   - Concurrent service creation
   - High-throughput task execution
   - API response times

2. **Scalability Testing**
   - Cluster scaling limits
   - Service instance scaling
   - Resource utilization

3. **Benchmark Testing**
   - API performance benchmarks
   - Memory and CPU usage
   - Network throughput

## Metrics and Reporting

### Test Metrics
- Execution time
- Success/failure rates
- Resource utilization
- Error patterns

### Reporting
- JUnit XML output
- Coverage reports
- Performance benchmarks
- HTML reports

### CI/CD Integration
```yaml
# .github/workflows/integration-tests.yml
name: Integration Tests
on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      
      - name: Run Integration Tests
        run: |
          cd tests/scenarios/integration
          go test -v -timeout 30m ./...
```

## Troubleshooting

### Common Issues
1. **Container startup failures**
   - Check Docker daemon
   - Verify image availability
   - Check port conflicts

2. **LocalStack connection issues**
   - Verify LocalStack is running
   - Check endpoint configuration
   - Validate AWS credentials

3. **Test timeouts**
   - Increase timeout values
   - Check resource constraints
   - Monitor system load

### Debug Mode
```bash
# Enable debug logging
export KECS_LOG_LEVEL=debug

# Run specific test with verbose output
go test -v -run TestSpecificScenario ./basic/
```

### Log Analysis
```bash
# View KECS logs
docker logs <kecs-container>

# View LocalStack logs
docker logs <localstack-container>

# View test output
go test -v ./... 2>&1 | tee test-output.log
```