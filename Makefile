# KECS (Kubernetes-based ECS Compatible Service) Makefile

# Variables
BINARY_NAME=kecs
MAIN_PKG=./cmd/controlplane
GO=go
GOFMT=gofmt
DOCKER=docker
DOCKER_IMAGE=ghcr.io/nandemo-ya/kecs
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X github.com/nandemo-ya/kecs/internal/controlplane/cmd.Version=$(VERSION)"
GOTEST=$(GO) test
GOVET=$(GO) vet
PLATFORMS=linux/amd64 linux/arm64

# Default target
.PHONY: all
all: clean fmt vet test build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(LDFLAGS) -o bin/$(BINARY_NAME) $(MAIN_PKG)

# Run the application
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	./bin/$(BINARY_NAME)

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.txt

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	$(GOVET) ./...

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod verify

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	$(DOCKER) build -t $(DOCKER_IMAGE):$(VERSION) .
	$(DOCKER) tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

# Push Docker image
.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker image..."
	$(DOCKER) push $(DOCKER_IMAGE):$(VERSION)
	$(DOCKER) push $(DOCKER_IMAGE):latest

# Generate OpenAPI code
.PHONY: gen-api
gen-api:
	@echo "Generating API code from OpenAPI spec..."
	# TODO: Add OpenAPI code generation command here

# Help target
.PHONY: help
help:
	@echo "KECS Makefile targets:"
	@echo "  all            - Run clean, fmt, vet, test, and build"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  vet            - Vet code"
	@echo "  deps           - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Push Docker image"
	@echo "  gen-api        - Generate API code from OpenAPI spec"
	@echo "  help           - Show this help message"
