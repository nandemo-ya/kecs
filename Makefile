# KECS (Kubernetes-based ECS Compatible Service) Makefile

# Variables
BINARY_NAME=kecs
MAIN_PKG=./controlplane/cmd/controlplane
GO=go
GOFMT=gofmt
DOCKER=docker
DOCKER_IMAGE=ghcr.io/nandemo-ya/kecs
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X github.com/nandemo-ya/kecs/controlplane/internal/controlplane/cmd.Version=$(VERSION)"
GOTEST=$(GO) test
GOVET=$(GO) vet
PLATFORMS=linux/amd64 linux/arm64
CONTROLPLANE_DIR=./controlplane
WEBUI_DIR=./web-ui

# Default target
.PHONY: all
all: clean fmt vet test build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	cd $(CONTROLPLANE_DIR) && $(GO) build $(LDFLAGS) -o ../bin/$(BINARY_NAME) ./cmd/controlplane

# Build with Web UI embedded
.PHONY: build-with-ui
build-with-ui: build-webui
	@echo "Building $(BINARY_NAME) with embedded Web UI..."
	cd $(CONTROLPLANE_DIR) && $(GO) build -tags embed_webui $(LDFLAGS) -o ../bin/$(BINARY_NAME) ./cmd/controlplane

# Build Web UI
.PHONY: build-webui
build-webui:
	@echo "Building Web UI..."
	@if [ -d "$(WEBUI_DIR)" ]; then \
		cd $(WEBUI_DIR) && npm install && npm run build; \
		rm -rf $(CONTROLPLANE_DIR)/internal/controlplane/api/webui_dist; \
		mkdir -p $(CONTROLPLANE_DIR)/internal/controlplane/api/webui_dist; \
		cp -r $(WEBUI_DIR)/build/* $(CONTROLPLANE_DIR)/internal/controlplane/api/webui_dist/; \
	else \
		echo "Web UI directory not found, skipping..."; \
	fi

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
	rm -rf $(CONTROLPLANE_DIR)/internal/controlplane/api/webui_dist

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w $(CONTROLPLANE_DIR)

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

# Push Docker image
.PHONY: docker-push
docker-push: docker-build
	@echo "Pushing Docker image..."
	$(DOCKER) push $(DOCKER_IMAGE):$(VERSION)
	$(DOCKER) push $(DOCKER_IMAGE):latest

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

# Generate API code from Smithy models
.PHONY: gen-api
gen-api:
	@echo "Downloading latest ECS Smithy model..."
	curl -s https://raw.githubusercontent.com/aws/aws-sdk-go-v2/main/codegen/sdk-codegen/aws-models/ecs.json -o api-models/ecs.json
	@echo "Generating API code from Smithy models..."
	cd $(CONTROLPLANE_DIR) && $(GO) run ./cmd/codegen -model=../api-models/ecs.json -output=internal/controlplane/api/generated

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
	@echo "  docker-build-awsproxy - Build AWS Proxy Docker image"
	@echo "  docker-push-awsproxy  - Push AWS Proxy Docker image"
	@echo "  gen-api        - Generate API code from Smithy models"
	@echo "  build-webui    - Build Web UI"
	@echo "  build-with-ui  - Build with embedded Web UI"
	@echo "  help           - Show this help message"
