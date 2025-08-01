.PHONY: test test-cluster test-verbose deps clean docker-build clean-test-resources

# Default test timeout
TIMEOUT ?= 30m

# Path to controlplane binary (can be overridden)
CONTROLPLANE_BINARY ?= ../../controlplane/bin/controlplane

# Test Phase 1: Cluster Operations
test-phase1:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./phase1/...

# Test Phase 2: Core Operations
test-phase2:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./phase2/...

# Test Phase 3: LocalStack Transparent Communication
test-phase3:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./phase3/...

# Test tasks only
test-task:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./task/...

# Test task definitions only
test-task-definition:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./task_definition/...

# Test services only
test-service:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./service/...

# Run all tests
test:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./...

# Run tests with detailed output
test-verbose:
	KECS_LOG_LEVEL=debug KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) ./...

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

# Clean up orphaned test resources (containers, temp dirs, k3d clusters)
clean-test-resources:
	@echo "Cleaning up orphaned test resources..."
	@docker ps -a --filter "label=com.kecs.managed=true" --format "{{.Names}}" | grep "^kecs-test-" | xargs -r docker stop || true
	@docker ps -a --filter "label=com.kecs.managed=true" --format "{{.Names}}" | grep "^kecs-test-" | xargs -r docker rm || true
	@k3d cluster list -o json 2>/dev/null | jq -r '.[].name' 2>/dev/null | grep '^kecs-test-' | xargs -r -I {} k3d cluster delete {} || true
	@find /tmp -maxdepth 1 -type d -name "kecs-test-*" -exec rm -rf {} + 2>/dev/null || true
	@echo "Cleanup complete"

# Build KECS docker image for testing
docker-build:
	cd ../../controlplane && docker build -f Dockerfile.test -t kecs:test .

# Run specific test
test-one:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) --focus $(TEST) ./...

# Run tests with coverage
test-coverage:
	KECS_CONTROLPLANE_BINARY=$(CONTROLPLANE_BINARY) go run github.com/onsi/ginkgo/v2/ginkgo -v -timeout $(TIMEOUT) --cover --coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Check AWS CLI is installed
check-aws:
	@which aws > /dev/null || (echo "AWS CLI not found. Please install it first." && exit 1)

# Setup test environment
setup: check-aws deps docker-build
	@echo "Test environment setup complete"
	@echo ""
	@echo "Available targets:"
	@echo "  make test              # Run all tests"
	@echo "  make test-phase1       # Run Phase 1 tests (Cluster Operations)"
	@echo "  make test-phase2       # Run Phase 2 tests (Task Definitions and Services)"
	@echo "  make test-phase3       # Run Phase 3 tests (LocalStack Integration)"
	@echo "  make test-verbose      # Run with debug logging"
	@echo ""
	@echo "Clean up test resources:"
	@echo "  make clean-test-resources"