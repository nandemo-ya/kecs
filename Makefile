# KECS (Kubernetes-based ECS Compatible Service) Makefile

# Variables
BINARY_NAME=kecs
MAIN_PKG=./controlplane/cmd/controlplane
GO=go
GOFMT=gofmt
GOIMPORTS=goimports
GOLANGCI_LINT=golangci-lint
DOCKER=docker
DOCKER_IMAGE=ghcr.io/nandemo-ya/kecs
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd.Version=$(VERSION)"
GOTEST=$(GO) test
GOVET=$(GO) vet
PLATFORMS=linux/amd64 linux/arm64
CONTROLPLANE_DIR=./controlplane

# Default target
.PHONY: all
all: clean fmt vet test build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	cd $(CONTROLPLANE_DIR) && $(GO) build $(LDFLAGS) -o ../bin/$(BINARY_NAME) ./cmd/controlplane

# Generate code from AWS API definitions
.PHONY: generate
generate:
	@echo "Generating code from AWS API definitions..."
	cd $(CONTROLPLANE_DIR) && $(GO) build -o ../bin/codegen ./cmd/codegen
	cd $(CONTROLPLANE_DIR) && ../bin/codegen -service ecs -input cmd/codegen/ecs.json -output internal/controlplane/api/generated_v2 -package api


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
	$(GOFMT) -s -w $(CONTROLPLANE_DIR)
	@echo "Organizing imports..."
	@if command -v $(GOIMPORTS) > /dev/null; then \
		$(GOIMPORTS) -w -local "github.com/nandemo-ya/kecs" $(CONTROLPLANE_DIR); \
	else \
		echo "goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	cd $(CONTROLPLANE_DIR) && $(GOTEST) -v -race ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	cd $(CONTROLPLANE_DIR) && $(GOTEST) -v -race -coverprofile=../coverage.txt -covermode=atomic ./...

# Vet code
.PHONY: vet
vet:
	@echo "Vetting code..."
	cd $(CONTROLPLANE_DIR) && $(GOVET) ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run; \
	else \
		echo "golangci-lint not found. Install with:"; \
		echo "  brew install golangci-lint (macOS)"; \
		echo "  or download from https://github.com/golangci/golangci-lint/releases"; \
	fi

# Fix linting issues automatically
.PHONY: lint-fix
lint-fix:
	@echo "Fixing linting issues..."
	@if command -v $(GOLANGCI_LINT) > /dev/null; then \
		$(GOLANGCI_LINT) run --fix; \
	else \
		echo "golangci-lint not found. Install with:"; \
		echo "  brew install golangci-lint (macOS)"; \
		echo "  or download from https://github.com/golangci/golangci-lint/releases"; \
	fi

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	cd $(CONTROLPLANE_DIR) && $(GO) mod download
	cd $(CONTROLPLANE_DIR) && $(GO) mod verify

# Scenario Tests
.PHONY: test-scenarios-simple
test-scenarios-simple: build
	@echo "Cleaning up any existing k3d clusters..."
	@k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-' | xargs -r -I {} k3d cluster delete {} || true
	@echo "Starting KECS server in background..."
	@KECS_SECURITY_ACKNOWLEDGED=true ./bin/$(BINARY_NAME) server --port 8080 --admin-port 8081 > kecs-test.log 2>&1 & echo $$! > kecs.pid
	@echo "Waiting for KECS to be ready..."
	@sleep 5
	@echo "Running scenario tests with single KECS instance..."
	@cd tests/scenarios && \
		KECS_ENDPOINT=http://localhost:8080 \
		KECS_ADMIN_ENDPOINT=http://localhost:8081 \
		KECS_TEST_MODE=simple \
		go test -v ./phase1 ./phase2 ./phase3 -p 1 -timeout 30m || true
	@echo "Stopping KECS server..."
	@if [ -f kecs.pid ]; then kill `cat kecs.pid` || true; rm kecs.pid; fi
	@echo "Cleaning up k3d clusters..."
	@k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-' | xargs -r -I {} k3d cluster delete {} || true
	@echo "Test log available in kecs-test.log"

# Clean up k3d clusters
.PHONY: clean-k3d
clean-k3d:
	@echo "Cleaning up all KECS k3d clusters..."
	@k3d cluster list -o json | jq -r '.[].name' | grep '^kecs-' | xargs -r -I {} k3d cluster delete {} || true
	@echo "K3d cleanup complete"

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	$(DOCKER) build -t $(DOCKER_IMAGE):$(VERSION) $(CONTROLPLANE_DIR)
	$(DOCKER) tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

# Build API-only Docker image
.PHONY: docker-build-api
docker-build-api:
	@echo "Building API-only Docker image..."
	$(DOCKER) build -t $(DOCKER_IMAGE)-api:$(VERSION) -f $(CONTROLPLANE_DIR)/Dockerfile.api $(CONTROLPLANE_DIR)
	$(DOCKER) tag $(DOCKER_IMAGE)-api:$(VERSION) $(DOCKER_IMAGE)-api:latest

# Build separated image (API only)
.PHONY: docker-build-separated
docker-build-separated: docker-build-api

# Push Docker image
.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker image..."
	$(DOCKER) push $(DOCKER_IMAGE):$(VERSION)
	$(DOCKER) push $(DOCKER_IMAGE):latest

# Push separated Docker image (API only)
.PHONY: docker-push-separated
docker-push-separated: docker-build-separated
	@echo "Pushing separated Docker image..."
	$(DOCKER) push $(DOCKER_IMAGE)-api:$(VERSION)
	$(DOCKER) push $(DOCKER_IMAGE)-api:latest

# Build AWS Proxy Docker image
.PHONY: docker-build-awsproxy
docker-build-awsproxy:
	@echo "Building AWS Proxy Docker image..."
	$(DOCKER) build -t $(DOCKER_REGISTRY)/aws-proxy:$(VERSION) -f $(CONTROLPLANE_DIR)/awsproxy/Dockerfile $(CONTROLPLANE_DIR)
	$(DOCKER) tag $(DOCKER_REGISTRY)/aws-proxy:$(VERSION) $(DOCKER_REGISTRY)/aws-proxy:latest

# Push AWS Proxy Docker image
.PHONY: docker-push-awsproxy
docker-push-awsproxy: docker-build-awsproxy
	@echo "Pushing AWS Proxy Docker image..."
	$(DOCKER) push $(DOCKER_REGISTRY)/aws-proxy:$(VERSION)
	$(DOCKER) push $(DOCKER_REGISTRY)/aws-proxy:latest


# Help target
.PHONY: help
help:
	@echo "KECS Makefile targets:"
	@echo "  all            - Run clean, fmt, vet, test, and build"
	@echo "  build          - Build the application"
	@echo "  run            - Run the application"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code and organize imports"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-scenarios-simple - Run scenario tests with single KECS instance"
	@echo "  clean-k3d      - Clean up all KECS k3d clusters"
	@echo "  vet            - Vet code"
	@echo "  lint           - Run golangci-lint"
	@echo "  lint-fix       - Run golangci-lint and fix issues automatically"
	@echo "  deps           - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Push Docker image"
	@echo "  docker-build-awsproxy - Build AWS Proxy Docker image"
	@echo "  docker-push-awsproxy  - Push AWS Proxy Docker image"
	@echo "  help           - Show this help message"
