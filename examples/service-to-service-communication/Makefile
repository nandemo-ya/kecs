# Makefile for Service-to-Service Communication Example

.PHONY: build deploy test clean help

# Configuration
KECS_ENDPOINT ?= http://localhost:5373
CLUSTER_NAME ?= default

help: ## Show this help message
	@echo "Service-to-Service Communication Example"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build Docker images for both services
	@echo "Building backend image..."
	@docker build -t backend-api:latest ./backend
	@echo "Building frontend image..."
	@docker build -t frontend-web:latest ./frontend

deploy: build ## Deploy services to KECS with service discovery
	@echo "Deploying services to KECS..."
	@./scripts/deploy.sh

test: ## Test service discovery and communication
	@echo "Testing service communication..."
	@./scripts/test-communication.sh

logs-backend: ## Show backend service logs
	@kubectl logs -l app=backend-api -f

logs-frontend: ## Show frontend service logs
	@kubectl logs -l app=frontend-web -f

port-forward: ## Port forward frontend service to localhost:3000
	@echo "Port forwarding frontend service to http://localhost:3000"
	@kubectl port-forward service/frontend-web-service 3000:3000

status: ## Check status of deployed services
	@echo "Checking ECS services..."
	@aws ecs list-services --cluster $(CLUSTER_NAME) --endpoint-url $(KECS_ENDPOINT)
	@echo ""
	@echo "Checking Service Discovery..."
	@aws servicediscovery list-namespaces --endpoint-url $(KECS_ENDPOINT)

clean: ## Remove all deployed resources
	@echo "Cleaning up..."
	@./scripts/cleanup.sh

run-backend-local: ## Run backend service locally for development
	@cd backend && go run main.go

run-frontend-local: ## Run frontend service locally for development
	@cd frontend && BACKEND_URL=http://localhost:8080 go run main.go