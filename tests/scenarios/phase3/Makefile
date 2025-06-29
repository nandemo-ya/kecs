.PHONY: test test-verbose test-one clean

# Default test command
test:
	@echo "Running Phase 3 LocalStack Communication Tests..."
	@go test -v -timeout 10m -run="LocalStack"

# Run tests with verbose output
test-verbose:
	@echo "Running Phase 3 Tests with verbose output..."
	@go test -v -timeout 10m -run="LocalStack" -ginkgo.v

# Run a specific test
test-one:
	@echo "Running specific test: $(TEST)"
	@go test -v -timeout 10m -run="$(TEST)"

# Clean test artifacts
clean:
	@echo "Cleaning test artifacts..."
	@rm -f *.test
	@rm -f *.out
	@rm -f *.log

# Run tests with race detection
test-race:
	@echo "Running Phase 3 Tests with race detection..."
	@go test -v -race -timeout 10m -run="LocalStack"

# Run tests with coverage
test-coverage:
	@echo "Running Phase 3 Tests with coverage..."
	@go test -v -timeout 10m -run="LocalStack" -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"