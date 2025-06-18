.PHONY: build test clean run-local

# Build all Docker images
build:
	@echo "Building backend image..."
	docker build -t three-tier-backend:latest backend/
	@echo "Building frontend image..."
	docker build -t three-tier-frontend:latest frontend/

# Run tests
test: build
	@echo "Running three-tier application tests..."
	cd tests && go test -v -timeout 30m

# Run specific test
test-one: build
	cd tests && go test -v -timeout 30m -run "$(TEST)"

# Clean up
clean:
	docker rmi three-tier-backend:latest || true
	docker rmi three-tier-frontend:latest || true

# Run locally for development
run-local: build
	docker compose up -d

# Stop local development
stop-local:
	docker compose down

# View logs
logs:
	docker compose logs -f

# Quick test of services locally
test-local:
	@echo "Testing frontend health..."
	@curl -f http://localhost:80/health || echo "Frontend unhealthy"
	@echo "\nTesting backend health..."
	@curl -f http://localhost:3000/health || echo "Backend unhealthy"
	@echo "\nTesting API through frontend..."
	@curl -f http://localhost:80/api/health || echo "API proxy not working"