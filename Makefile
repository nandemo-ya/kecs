# KECS (Kubernetes-based ECS Compatible Service) Makefile

# Variables
CLI_BINARY_NAME=kecs
SERVER_BINARY_NAME=kecs-server
MAIN_PKG=./controlplane/cmd/controlplane
GO=go
GOFMT=gofmt
GOIMPORTS=goimports
GOLANGCI_LINT=golangci-lint
DOCKER=docker
DOCKER_IMAGE=ghcr.io/nandemo-ya/kecs

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_VERSION := $(shell go version | cut -d' ' -f3)

# Build flags
LDFLAGS := -ldflags "\
  -X github.com/nandemo-ya/kecs/controlplane/internal/version.Version=$(VERSION) \
  -X github.com/nandemo-ya/kecs/controlplane/internal/version.GitCommit=$(GIT_COMMIT) \
  -X github.com/nandemo-ya/kecs/controlplane/internal/version.BuildDate=$(BUILD_DATE) \
  -X github.com/nandemo-ya/kecs/controlplane/internal/version.GoVersion=$(GO_VERSION)"

GOTEST=$(GO) test
GOVET=$(GO) vet
PLATFORMS=linux/amd64 linux/arm64
CONTROLPLANE_DIR=./controlplane

# Default target
.PHONY: all
all: clean fmt vet test build

# Build both CLI and server
.PHONY: build
build: build-cli build-server

# Build CLI (without DuckDB/CGO)
.PHONY: build-cli
build-cli:
	@echo "Building $(CLI_BINARY_NAME) (CLI only, no DuckDB)..."
	cd $(CONTROLPLANE_DIR) && CGO_ENABLED=0 $(GO) build -tags nocli $(LDFLAGS) -o ../bin/$(CLI_BINARY_NAME) ./cmd/controlplane

# Build server (with DuckDB/CGO)
.PHONY: build-server
build-server:
	@echo "Building $(SERVER_BINARY_NAME) (with DuckDB support)..."
	cd $(CONTROLPLANE_DIR) && CGO_ENABLED=1 $(GO) build $(LDFLAGS) -o ../bin/$(SERVER_BINARY_NAME) ./cmd/controlplane

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

# Generate CREDITS file for dependencies
.PHONY: credits
credits:
	@echo "Generating CREDITS file..."
	@if ! command -v gocredits > /dev/null 2>&1; then \
		echo "Installing gocredits..."; \
		go install github.com/Songmu/gocredits/cmd/gocredits@latest; \
	fi
	@cd $(CONTROLPLANE_DIR) && gocredits -skip-missing . > ../CREDITS 2>/dev/null || true
	@echo "CREDITS file generated successfully"


# Run the CLI
.PHONY: run
run: build-cli
	@echo "Running $(CLI_BINARY_NAME) (CLI)..."
	./bin/$(CLI_BINARY_NAME)

# Run the server
.PHONY: run-server
run-server: build-server
	@echo "Running $(SERVER_BINARY_NAME) (Server with DuckDB)..."
	./bin/$(SERVER_BINARY_NAME) server

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
	$(DOCKER) build -t localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION) $(CONTROLPLANE_DIR)
	$(DOCKER) tag localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION) localhost:5000/nandemo-ya/kecs-controlplane:latest

# Push Docker image to local k3d registry (dev mode)
.PHONY: docker-push-dev
docker-push-dev: docker-build-dev
	@echo "Pushing to k3d registry..."
	$(DOCKER) push localhost:5000/nandemo-ya/kecs-controlplane:$(VERSION)
	$(DOCKER) push localhost:5000/nandemo-ya/kecs-controlplane:latest

# Hot reload: Build and replace controlplane in running KECS instance
.PHONY: hot-reload
hot-reload: docker-push-dev
	@echo "Hot reloading controlplane in KECS..."
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
			echo "Please specify one with: KECS_INSTANCE=<name> make dev"; \
			exit 1; \
		fi; \
	fi; \
	echo "Updating controlplane in cluster: $$CLUSTER_NAME"; \
	kubectl config use-context "k3d-$$CLUSTER_NAME" && \
	kubectl set image deployment/kecs-controlplane controlplane=registry.kecs.local:5000/nandemo-ya/kecs-controlplane:$(VERSION) -n kecs-system && \
	kubectl rollout status deployment/kecs-controlplane -n kecs-system && \
	echo "✅ Controlplane updated successfully!"

# Dev workflow: Build server and hot reload in one command
.PHONY: dev
dev: build-server hot-reload
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
	$(DOCKER) build -t $(DOCKER_IMAGE)-awsproxy:$(VERSION) -f $(CONTROLPLANE_DIR)/awsproxy/Dockerfile $(CONTROLPLANE_DIR)
	$(DOCKER) tag $(DOCKER_IMAGE)-awsproxy:$(VERSION) $(DOCKER_IMAGE)-awsproxy:latest

# Push AWS Proxy Docker image
.PHONY: docker-push-awsproxy
docker-push-awsproxy: docker-build-awsproxy
	@echo "Pushing AWS Proxy Docker image..."
	$(DOCKER) push $(DOCKER_IMAGE)-awsproxy:$(VERSION)
	$(DOCKER) push $(DOCKER_IMAGE)-awsproxy:latest


# Help target
.PHONY: help
help:
	@echo "KECS Makefile targets:"
	@echo ""
	@echo "Building:"
	@echo "  all            - Run clean, fmt, vet, test, and build"
	@echo "  build          - Build both CLI and server binaries"
	@echo "  build-cli      - Build CLI binary (no DuckDB/CGO)"
	@echo "  build-server   - Build server binary (with DuckDB/CGO)"
	@echo "  build-tui2     - Build TUI v2 mock application"
	@echo ""
	@echo "Running:"
	@echo "  run            - Build and run CLI"
	@echo "  run-server     - Build and run server with DuckDB"
	@echo "  run-tui2       - Build and run TUI v2 mock"
	@echo ""
	@echo "Code Quality:"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format code and organize imports"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  vet            - Vet code"
	@echo "  lint           - Run golangci-lint"
	@echo "  lint-fix       - Run golangci-lint and fix issues automatically"
	@echo "  deps           - Install dependencies"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build   - Build Docker image (server)"
	@echo "  docker-push    - Push Docker image"
	@echo "  docker-build-dev - Build Docker image for k3d registry (dev mode)"
	@echo "  docker-push-dev - Push Docker image to k3d registry (dev mode)"
	@echo "  docker-build-awsproxy - Build AWS Proxy Docker image"
	@echo "  docker-push-awsproxy  - Push AWS Proxy Docker image"
	@echo ""
	@echo "Development:"
	@echo "  hot-reload     - Build and replace controlplane in running KECS instance"
	@echo "  dev            - Build server and hot reload controlplane"
	@echo "  dev-logs       - Same as 'dev' but also tail controlplane logs"
	@echo "  generate       - Generate code from AWS API definitions"
	@echo "  credits        - Generate CREDITS file for dependencies"
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Development workflow (Docker hot-reload):"
	@echo "  1. Start KECS: ./bin/kecs start"
	@echo "  2. Make code changes"
	@echo "  3. Run: make dev"
	@echo "  4. Or run with logs: make dev-logs"
	@echo ""
	@echo "For specific instance: KECS_INSTANCE=myinstance make dev"
