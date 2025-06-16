# LocalStack Integration Tests

.PHONY: test test-lifecycle test-proxy test-env clean build-image help

# Default target
all: test

# Build KECS test image if needed
build-image:
	@echo "Building KECS test image..."
	@cd ../../../ && make docker-build-test

# Run all LocalStack tests
test:
	@echo "Running all LocalStack integration tests..."
	ginkgo -v .

# Run lifecycle tests only
test-lifecycle:
	@echo "Running LocalStack lifecycle tests..."
	ginkgo -v --focus="LocalStack Lifecycle" .

# Run AWS API proxy tests only
test-proxy:
	@echo "Running AWS API proxy tests..."
	ginkgo -v --focus="AWS API Proxy" .

# Run environment injection tests only
test-env:
	@echo "Running environment variable injection tests..."
	ginkgo -v --focus="Environment Variable Injection" .

# Run tests with debug output
test-verbose:
	@echo "Running tests with verbose output..."
	KECS_LOG_LEVEL=debug ginkgo -v --trace .

# Run a specific test
test-one:
	@echo "Running test: $(TEST)"
	ginkgo -v --focus="$(TEST)" .

# Clean up test artifacts
clean:
	@echo "Cleaning up test artifacts..."
	@rm -f *.log
	@rm -f *.out

# Help
help:
	@echo "LocalStack Integration Test Targets:"
	@echo "  make test           - Run all LocalStack tests"
	@echo "  make test-lifecycle - Run lifecycle tests only"
	@echo "  make test-proxy     - Run AWS API proxy tests only"
	@echo "  make test-env       - Run environment injection tests only"
	@echo "  make test-verbose   - Run tests with debug output"
	@echo "  make test-one TEST=<name> - Run specific test by name"
	@echo "  make build-image    - Build KECS test image"
	@echo "  make clean          - Clean up test artifacts"
	@echo "  make help           - Show this help message"