.PHONY: test test-cluster test-verbose deps clean docker-build

# Default test timeout
TIMEOUT ?= 30m

# Test specific clusters
test-cluster:
	ginkgo -v -timeout $(TIMEOUT) ./cluster/...

# Run all tests
test:
	ginkgo -v -timeout $(TIMEOUT) ./...

# Run tests with detailed output
test-verbose:
	KECS_LOG_LEVEL=debug ginkgo -v -timeout $(TIMEOUT) ./...

# Install dependencies
deps:
	go mod download
	go mod tidy
	@which ginkgo > /dev/null || go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4

# Clean test artifacts
clean:
	rm -rf results/
	go clean -testcache
	rm -f *.coverprofile coverage.out coverage.html

# Build KECS docker image for testing
docker-build:
	cd ../../controlplane && docker build -t kecs:test .

# Run specific test
test-one:
	ginkgo -v -timeout $(TIMEOUT) --focus $(TEST) ./...

# Run tests with coverage
test-coverage:
	ginkgo -v -timeout $(TIMEOUT) --cover --coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Check AWS CLI is installed
check-aws:
	@which aws > /dev/null || (echo "AWS CLI not found. Please install it first." && exit 1)

# Setup test environment
setup: check-aws deps docker-build
	@echo "Test environment setup complete"