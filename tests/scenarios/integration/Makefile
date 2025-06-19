.PHONY: test test-basic test-advanced test-performance test-all clean

# Run all integration tests
test: test-all

# Run all tests with timeout
test-all:
	@echo "Running all integration tests..."
	go test -v -timeout 30m ./...

# Run basic integration tests
test-basic:
	@echo "Running basic integration tests..."
	go test -v -timeout 15m ./basic/...

# Run advanced integration tests
test-advanced:
	@echo "Running advanced integration tests..."
	go test -v -timeout 20m ./advanced/...

# Run performance tests
test-performance:
	@echo "Running performance tests..."
	go test -v -timeout 45m ./performance/...

# Run specific test
test-one:
	@echo "Running specific test: $(TEST)"
	go test -v -timeout 15m -run "$(TEST)" ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -timeout 30m -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -v -race -timeout 30m ./...

# Run tests in parallel
test-parallel:
	@echo "Running tests in parallel..."
	go test -v -timeout 30m -parallel 4 ./...

# Run fast tests only
test-fast:
	@echo "Running fast tests..."
	go test -v -timeout 10m -tags=fast ./...

# Run slow tests only (includes performance)
test-slow:
	@echo "Running slow tests..."
	go test -v -timeout 45m -tags=slow ./...

# Benchmark tests
benchmark:
	@echo "Running benchmark tests..."
	go test -v -timeout 30m -bench=. -benchmem ./...

# Clean up test artifacts
clean:
	@echo "Cleaning up test artifacts..."
	rm -f coverage.out coverage.html
	go clean -testcache

# Install dependencies
deps:
	@echo "Installing test dependencies..."
	go mod download
	go mod verify

# Lint tests
lint:
	@echo "Linting test code..."
	golangci-lint run ./...

# Format test code
fmt:
	@echo "Formatting test code..."
	go fmt ./...

# Vet test code
vet:
	@echo "Vetting test code..."
	go vet ./...

# Run all quality checks
check: fmt vet lint

# Help
help:
	@echo "Available targets:"
	@echo "  test           - Run all integration tests"
	@echo "  test-basic     - Run basic integration tests"
	@echo "  test-advanced  - Run advanced integration tests"
	@echo "  test-performance - Run performance tests"
	@echo "  test-one       - Run specific test (set TEST variable)"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-parallel  - Run tests in parallel"
	@echo "  test-fast      - Run fast tests only"
	@echo "  test-slow      - Run slow tests only"
	@echo "  benchmark      - Run benchmark tests"
	@echo "  clean          - Clean up test artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  check          - Run all quality checks"
	@echo "  help           - Show this help"