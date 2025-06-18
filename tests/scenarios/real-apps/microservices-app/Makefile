.PHONY: build test clean run-local

# Build all service images
build:
	@echo "Building API Gateway image..."
	docker build -t microservices-api-gateway:latest services/api-gateway/
	@echo "Building User Service image..."
	docker build -t microservices-user-service:latest services/user-service/
	@echo "Building Order Service image..."
	# docker build -t microservices-order-service:latest services/order-service/
	@echo "Building Storage Service image..."
	# docker build -t microservices-storage-service:latest services/storage-service/
	@echo "Building Notification Service image..."
	# docker build -t microservices-notification-service:latest services/notification-service/

# Run tests
test: build
	@echo "Running microservices tests..."
	cd tests && go test -v -timeout 30m

# Run specific test
test-one: build
	cd tests && go test -v -timeout 30m -run "$(TEST)"

# Clean up
clean:
	docker rmi microservices-api-gateway:latest || true
	docker rmi microservices-user-service:latest || true
	docker rmi microservices-order-service:latest || true
	docker rmi microservices-storage-service:latest || true
	docker rmi microservices-notification-service:latest || true

# Run locally for development
run-local: build
	docker compose up -d

# Stop local development
stop-local:
	docker compose down

# View logs
logs:
	docker compose logs -f

# Test endpoints
test-endpoints:
	@echo "Testing API Gateway health..."
	@curl -f http://localhost:8080/health || echo "API Gateway unhealthy"
	@echo "\n\nTesting user creation..."
	@curl -X POST http://localhost:8080/api/users \
		-H "Content-Type: application/json" \
		-d '{"email":"test@example.com","password":"password123","name":"Test User"}' || echo "User creation failed"

# View specific service logs
logs-gateway:
	docker compose logs -f api-gateway

logs-user:
	docker compose logs -f user-service