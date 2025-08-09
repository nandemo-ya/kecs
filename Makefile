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

# Build TUI v2 mock application
.PHONY: build-tui2
build-tui2:
	@echo "Building KECS TUI v2 (mock)..."
	cd $(CONTROLPLANE_DIR) && $(GO) build -o ../bin/kecs-tui2 ./cmd/kecs-tui2

# Run TUI v2 mock
.PHONY: run-tui2
run-tui2: build-tui2
	@echo "Running KECS TUI v2 (mock)..."
	./bin/kecs-tui2

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

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	$(DOCKER) build -t $(DOCKER_IMAGE):$(VERSION) $(CONTROLPLANE_DIR)
	$(DOCKER) tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_IMAGE):latest

# Build Docker image for local k3d registry (dev mode)
.PHONY: docker-build-dev
docker-build-dev:
	@echo "Building Docker image for local k3d registry..."
	$(DOCKER) build -t k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION) $(CONTROLPLANE_DIR)
	$(DOCKER) tag k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION) k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:latest

# Push Docker image to local k3d registry (dev mode)
.PHONY: docker-push-dev
docker-push-dev: docker-build-dev
	@echo "Pushing to k3d registry..."
	$(DOCKER) push k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION)
	$(DOCKER) push k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:latest

# Hot reload: Build and replace controlplane in running KECS instance
.PHONY: hot-reload
hot-reload: docker-push-dev
	@echo "Hot reloading controlplane in KECS..."
	@# Get the instance name (default to 'default' if not specified)
	@INSTANCE_NAME=$${KECS_INSTANCE:-default}; \
	CLUSTER_NAME="kecs-$$INSTANCE_NAME"; \
	echo "Updating controlplane in cluster: $$CLUSTER_NAME"; \
	if kubectl config get-contexts -o name | grep -q "k3d-$$CLUSTER_NAME"; then \
		kubectl config use-context "k3d-$$CLUSTER_NAME" && \
		kubectl set image deployment/kecs-controlplane kecs=k3d-kecs-registry.localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION) -n kecs-system && \
		kubectl rollout status deployment/kecs-controlplane -n kecs-system && \
		echo "✅ Controlplane updated successfully!"; \
	else \
		echo "❌ Error: KECS cluster '$$CLUSTER_NAME' not found."; \
		echo "Available clusters:"; \
		kubectl config get-contexts -o name | grep "k3d-kecs-" | sed 's/k3d-kecs-/  - /'; \
		exit 1; \
	fi

# Dev workflow: Build and hot reload in one command
.PHONY: dev
dev: build hot-reload
	@echo "✅ Development build and deploy completed!"

# Dev workflow with logs: Build, reload and tail logs
.PHONY: dev-logs
dev-logs: dev
	@# Auto-detect KECS cluster if not specified (same logic as dev)
	@if [ -n "$${KECS_INSTANCE}" ]; then \
		CLUSTER_NAME="kecs-$${KECS_INSTANCE}"; \
	else \
		CLUSTERS=$$(kubectl config get-contexts -o name | grep "^k3d-kecs-" | sed 's/^k3d-//'); \
		CLUSTER_COUNT=$$(echo "$$CLUSTERS" | grep -c "^kecs-"); \
		if [ "$$CLUSTER_COUNT" -eq 1 ]; then \
			CLUSTER_NAME=$$(echo "$$CLUSTERS" | head -1); \
		else \
			echo "❌ Error: Cannot determine cluster for logs."; \
			exit 1; \
		fi; \
	fi; \
	kubectl config use-context "k3d-$$CLUSTER_NAME" && \
	kubectl logs -f deployment/kecs-controlplane -n kecs-system

# Ko development: Ultra-fast hot reload using ko
.PHONY: ko-dev
ko-dev:
	@echo "🚀 Using ko for ultra-fast deployment..."
	@# Auto-detect KECS cluster if not specified
	@if [ -n "$${KECS_INSTANCE}" ]; then \
		CLUSTER_NAME="kecs-$${KECS_INSTANCE}"; \
	else \
		CLUSTERS=$$(kubectl config get-contexts -o name | grep "^k3d-kecs-" | sed 's/^k3d-//'); \
		CLUSTER_COUNT=$$(echo "$$CLUSTERS" | grep -c "^kecs-"); \
		if [ "$$CLUSTER_COUNT" -eq 0 ]; then \
			echo "❌ Error: No KECS clusters found."; \
			echo "Start a KECS instance with: ./bin/kecs start"; \
			exit 1; \
		elif [ "$$CLUSTER_COUNT" -eq 1 ]; then \
			CLUSTER_NAME=$$(echo "$$CLUSTERS" | head -1); \
			echo "Auto-detected cluster: $$CLUSTER_NAME"; \
		else \
			echo "Multiple KECS clusters found:"; \
			echo "$$CLUSTERS" | sed 's/^/  - /'; \
			echo ""; \
			echo "Please specify one with: KECS_INSTANCE=<name> make ko-dev"; \
			exit 1; \
		fi; \
	fi; \
	kubectl config use-context "k3d-$$CLUSTER_NAME" && \
	export KO_DOCKER_REPO=k3d-kecs-registry.localhost:5000 && \
	export VERSION=$$(git describe --tags --always --dirty) && \
	kubectl get deployment kecs-controlplane -n kecs-system -o yaml | \
		sed 's|image: .*kecs-controlplane.*|image: ko://github.com/nandemo-ya/kecs/controlplane/cmd/controlplane|' | \
		ko apply -f - --bare -t latest && \
	kubectl rollout status deployment/kecs-controlplane -n kecs-system && \
	echo "✅ Deployment updated with ko!"

# Ko development with logs
.PHONY: ko-dev-logs
ko-dev-logs: ko-dev
	@# Auto-detect KECS cluster if not specified (same logic as ko-dev)
	@if [ -n "$${KECS_INSTANCE}" ]; then \
		CLUSTER_NAME="kecs-$${KECS_INSTANCE}"; \
	else \
		CLUSTERS=$$(kubectl config get-contexts -o name | grep "^k3d-kecs-" | sed 's/^k3d-//'); \
		CLUSTER_COUNT=$$(echo "$$CLUSTERS" | grep -c "^kecs-"); \
		if [ "$$CLUSTER_COUNT" -eq 1 ]; then \
			CLUSTER_NAME=$$(echo "$$CLUSTERS" | head -1); \
		else \
			echo "❌ Error: Cannot determine cluster for logs."; \
			exit 1; \
		fi; \
	fi; \
	kubectl config use-context "k3d-$$CLUSTER_NAME" && \
	kubectl logs -f deployment/kecs-controlplane -n kecs-system


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
	@echo "  vet            - Vet code"
	@echo "  lint           - Run golangci-lint"
	@echo "  lint-fix       - Run golangci-lint and fix issues automatically"
	@echo "  deps           - Install dependencies"
	@echo "  docker-build   - Build Docker image"
	@echo "  docker-push    - Push Docker image"
	@echo "  docker-build-dev - Build Docker image for k3d registry (dev mode)"
	@echo "  docker-push-dev - Push Docker image to k3d registry (dev mode)"
	@echo "  hot-reload     - Build and replace controlplane in running KECS instance"
	@echo "  dev            - Build binary and hot reload controlplane (development workflow)"
	@echo "  dev-logs       - Same as 'dev' but also tail controlplane logs"
	@echo "  ko-dev         - Ultra-fast deployment using ko (no Docker build)"
	@echo "  ko-dev-logs    - Same as 'ko-dev' but also tail controlplane logs"
	@echo "  docker-build-awsproxy - Build AWS Proxy Docker image"
	@echo "  docker-push-awsproxy  - Push AWS Proxy Docker image"
	@echo "  build-tui2     - Build TUI v2 mock application"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Development workflow (ko - Recommended):"
	@echo "  1. Start KECS: ./bin/kecs start"
	@echo "  2. Make code changes"
	@echo "  3. Run: make ko-dev"
	@echo "  4. Or run with logs: make ko-dev-logs"
	@echo ""
	@echo "Development workflow (Docker hot-reload):"
	@echo "  1. Start KECS: ./bin/kecs start"
	@echo "  2. Make code changes"
	@echo "  3. Run: make dev"
	@echo "  4. Or run with logs: make dev-logs"
	@echo ""
	@echo "For specific instance: KECS_INSTANCE=myinstance make ko-dev"
