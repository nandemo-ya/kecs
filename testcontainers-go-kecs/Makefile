.PHONY: test test-examples test-coverage deps clean help

# Default target
default: test

# Run all tests
test:
	go test -v -race ./...

# Run example tests
test-examples:
	cd examples && go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Download dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	go clean -testcache

# Show help
help:
	@echo "Available targets:"
	@echo "  test          - Run all tests"
	@echo "  test-examples - Run example tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  help          - Show this help message"