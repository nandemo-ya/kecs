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
	@INSTANCE_NAME=$${KECS_INSTANCE:-default}; \
	CLUSTER_NAME="kecs-$$INSTANCE_NAME"; \
	kubectl config use-context "k3d-$$CLUSTER_NAME" && \
	kubectl logs -f deployment/kecs-controlplane -n kecs-system

# Telepresence: Connect to cluster for local development
.PHONY: telepresence-connect
telepresence-connect:
	@echo "Connecting to cluster with Telepresence..."
	@# Try to use KECS_INSTANCE if set, otherwise auto-detect
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
			echo "Please specify one with: KECS_INSTANCE=<name> make telepresence-connect"; \
			exit 1; \
		fi; \
	fi; \
	if kubectl config get-contexts -o name | grep -q "k3d-$$CLUSTER_NAME"; then \
		kubectl config use-context "k3d-$$CLUSTER_NAME" && \
		telepresence connect --namespace kecs-system && \
		echo "✅ Connected to $$CLUSTER_NAME cluster"; \
	else \
		echo "❌ Error: KECS cluster '$$CLUSTER_NAME' not found."; \
		exit 1; \
	fi

# Telepresence: Intercept controlplane traffic for local development
.PHONY: telepresence-intercept
telepresence-intercept: build
	@echo "Intercepting controlplane traffic..."
	@if ! telepresence status | grep -q "Connected"; then \
		echo "Telepresence not connected. Running telepresence-connect..."; \
		$(MAKE) telepresence-connect; \
	fi && \
	echo "Setting up intercept for kecs-controlplane API service..." && \
	telepresence intercept kecs-controlplane \
		--service kecs-api \
		--port 8080:http \
		--env-file .telepresence.env && \
	echo "✅ Intercept active for API service (port 8080)." && \
	echo "" && \
	echo "⚠️  Important: Traffic routing depends on how you access KECS:" && \
	echo "  - Cluster internal: Automatically intercepted to local controlplane" && \
	echo "  - Via TUI port (e.g., 8080): Still goes to cluster (use port-forward instead)" && \
	echo "  - Via port-forward: Goes to local controlplane" && \
	echo "" && \
	echo "Run the following to start local controlplane:" && \
	echo "  source .telepresence.env && KECS_DATA_DIR=/tmp/kecs-data ./bin/kecs server"

# Telepresence: Run local controlplane with intercept
.PHONY: telepresence-run
telepresence-run: telepresence-intercept
	@echo "Starting local controlplane with intercepted traffic..."
	@echo ""
	@echo "📝 Note: When using Telepresence, access the API through:"
	@echo "  - Cluster internal traffic: automatically intercepted"
	@echo "  - External access: kubectl port-forward service/kecs-api 9080:80 -n kecs-system"
	@echo "  - Then use: http://localhost:9080"
	@echo ""
	@if [ -f .telepresence.env ]; then \
		source .telepresence.env && \
		KECS_DATA_DIR=/tmp/kecs-data \
		./bin/$(BINARY_NAME) server; \
	else \
		echo "❌ Error: .telepresence.env not found. Run 'make telepresence-intercept' first."; \
		exit 1; \
	fi

# Telepresence: Stop intercept and disconnect
.PHONY: telepresence-stop
telepresence-stop:
	@echo "Stopping Telepresence intercept..."
	@telepresence leave kecs-controlplane 2>/dev/null || true
	@echo "Disconnecting from cluster..."
	@telepresence quit 2>/dev/null || true
	@rm -f .telepresence.env
	@echo "✅ Telepresence stopped"

# Telepresence: Show current status
.PHONY: telepresence-status
telepresence-status:
	@echo "Telepresence status:"
	@telepresence status
	@echo ""
	@echo "Active intercepts:"
	@telepresence list

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
	@echo "  docker-build-awsproxy - Build AWS Proxy Docker image"
	@echo "  docker-push-awsproxy  - Push AWS Proxy Docker image"
	@echo "  build-tui2     - Build TUI v2 mock application"
	@echo ""
	@echo "Telepresence targets (for local development):"
	@echo "  telepresence-connect   - Connect to KECS cluster with Telepresence"
	@echo "  telepresence-intercept - Build and intercept controlplane traffic"
	@echo "  telepresence-run       - Run local controlplane with intercepted traffic"
	@echo "  telepresence-stop      - Stop intercept and disconnect"
	@echo "  telepresence-status    - Show Telepresence connection status"
	@echo ""
	@echo "  help           - Show this help message"
	@echo ""
	@echo "Development workflow (Docker hot-reload):"
	@echo "  1. Start KECS: ./bin/kecs start"
	@echo "  2. Make code changes"
	@echo "  3. Run: make dev"
	@echo "  4. Or run with logs: make dev-logs"
	@echo ""
	@echo "Development workflow (Telepresence):"
	@echo "  1. Start KECS: ./bin/kecs start"
	@echo "  2. Run: make telepresence-run"
	@echo "  3. Make code changes and restart local binary"
	@echo "  4. When done: make telepresence-stop"
	@echo ""
	@echo "For specific instance: KECS_INSTANCE=myinstance make dev"
