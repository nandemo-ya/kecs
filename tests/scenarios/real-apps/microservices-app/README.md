# Microservices Application Test Suite

This test suite demonstrates KECS capabilities with a microservices architecture using LocalStack services and service discovery.

## Architecture

```
┌─────────────────┐
│   API Gateway   │
│     (Node.js)   │
└────────┬────────┘
         │
    ┌────┴────┬─────────┬──────────┐
    │         │         │          │
┌───▼───┐ ┌──▼───┐ ┌──▼───┐ ┌────▼────┐
│ User  │ │Order │ │Storage│ │Notification│
│Service│ │Service│ │Service│ │  Service   │
└───┬───┘ └──┬───┘ └──┬───┘ └────┬────┘
    │        │        │          │
┌───▼───┐ ┌──▼───┐ ┌──▼───┐ ┌────▼────┐
│DynamoDB│ │DynamoDB│ │  S3  │ │SQS/SNS │
└────────┘ └────────┘ └──────┘ └─────────┘
```

## Services

### 1. API Gateway Service
- **Purpose**: Central entry point for all client requests
- **Technology**: Node.js with Express
- **Features**:
  - Request routing to appropriate microservices
  - Authentication/authorization
  - Rate limiting
  - Service discovery integration

### 2. User Service
- **Purpose**: User management and authentication
- **Technology**: Node.js
- **Storage**: DynamoDB (LocalStack)
- **Features**:
  - User registration/login
  - Profile management
  - JWT token generation

### 3. Order Service
- **Purpose**: Order processing and management
- **Technology**: Python with FastAPI
- **Storage**: DynamoDB (LocalStack)
- **Features**:
  - Order creation
  - Order status tracking
  - Integration with User Service for validation
  - Sends notifications via SNS

### 4. Storage Service
- **Purpose**: File storage and management
- **Technology**: Go
- **Storage**: S3 (LocalStack)
- **Features**:
  - File upload/download
  - Pre-signed URL generation
  - Metadata management

### 5. Notification Service
- **Purpose**: Asynchronous notification processing
- **Technology**: Python
- **Integration**: SQS/SNS (LocalStack)
- **Features**:
  - Email notifications
  - SMS notifications (simulated)
  - Webhook delivery

## Test Scenarios

### 1. Service Discovery and Communication
- Services register with Cloud Map
- Services discover each other dynamically
- Health checks and circuit breakers

### 2. Distributed Transaction Flow
- User creates an account
- User places an order
- Order triggers notification
- Files are uploaded and processed

### 3. Resilience Testing
- Service failure and recovery
- Message queue processing during failures
- Circuit breaker activation

### 4. Scaling and Load Distribution
- Auto-scaling based on metrics
- Load balancing across instances
- Performance under load

## LocalStack Services Used

- **DynamoDB**: User and order data storage
- **S3**: File storage
- **SQS**: Asynchronous message processing
- **SNS**: Notification distribution
- **Cloud Map**: Service discovery
- **CloudWatch**: Metrics and logging
- **IAM**: Service authentication

## Running Tests

### Prerequisites
- Docker
- Go 1.21+
- Node.js 18+
- Python 3.9+

### Build and Test
```bash
# Build all service images
make build

# Run all tests
make test

# Run specific test scenario
make test-one TEST=TestServiceDiscovery

# Run locally for development
make run-local

# Test endpoints
make test-endpoints
```

### Local Development
```bash
# Start all services
make run-local

# View logs
make logs

# Test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/users
curl http://localhost:8080/api/orders

# Stop services
make stop-local
```

## API Endpoints

### API Gateway (Port 8080)
- `GET /health` - Health check
- `POST /api/users` - Create user
- `GET /api/users/:id` - Get user
- `POST /api/orders` - Create order
- `GET /api/orders/:id` - Get order
- `POST /api/storage/upload` - Upload file
- `GET /api/storage/:key` - Download file

## Environment Variables

### Common
- `SERVICE_NAME` - Service identifier
- `SERVICE_PORT` - Service port
- `LOCALSTACK_ENDPOINT` - LocalStack endpoint URL
- `SERVICE_DISCOVERY_NAMESPACE` - Cloud Map namespace

### Service-Specific
- `DYNAMODB_TABLE` - DynamoDB table name
- `S3_BUCKET` - S3 bucket name
- `SQS_QUEUE_URL` - SQS queue URL
- `SNS_TOPIC_ARN` - SNS topic ARN

## Troubleshooting

### Service Discovery Issues
1. Check Cloud Map registration:
   ```bash
   aws servicediscovery list-services --endpoint-url=$LOCALSTACK_ENDPOINT
   ```

2. Verify DNS resolution:
   ```bash
   docker exec <container> nslookup user-service.microservices.local
   ```

### LocalStack Connection Issues
1. Verify LocalStack is running:
   ```bash
   curl http://localhost:4566/_localstack/health
   ```

2. Check service logs:
   ```bash
   docker logs <service-container>
   ```

### Message Queue Issues
1. Check SQS messages:
   ```bash
   aws sqs receive-message --queue-url <queue-url> --endpoint-url=$LOCALSTACK_ENDPOINT
   ```

2. Verify SNS subscriptions:
   ```bash
   aws sns list-subscriptions --endpoint-url=$LOCALSTACK_ENDPOINT
   ```