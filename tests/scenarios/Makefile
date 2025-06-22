.PHONY: test test-cluster test-verbose deps clean docker-build

# Default test timeout
TIMEOUT ?= 30m

# Test Phase 1: Cluster Operations
test-phase1:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./phase1/...

# Test Phase 2: Core Operations
test-phase2:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./phase2/...

# Test Phase 3: Task Lifecycle & Status
test-phase3:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./task/... ./service/service_tasks_test.go

# Test tasks only
test-task:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./task/...

# Test task definitions only
test-task-definition:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./task_definition/...

# Test services only
test-service:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./service/...

# Run all tests
test:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./...

# Run tests with detailed output
test-verbose:
	KECS_LOG_LEVEL=debug go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./...

# Install dependencies
deps:
	go mod download
	go mod tidy
	# Ginkgo is automatically available via Go 1.24 tool directive

# Clean test artifacts
clean:
	rm -rf results/
	go clean -testcache
	rm -f *.coverprofile coverage.out coverage.html

# Build KECS docker image for testing
docker-build:
	cd ../../controlplane && docker build -f Dockerfile.test -t kecs:test .

# Run specific test
test-one:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) --focus $(TEST) ./...

# Run tests with coverage
test-coverage:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) --cover --coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Check AWS CLI is installed
check-aws:
	@which aws > /dev/null || (echo "AWS CLI not found. Please install it first." && exit 1)

# Setup test environment
setup: check-aws deps docker-build
	@echo "Test environment setup complete"