# Three-Tier Application Test Suite

This test suite demonstrates KECS capabilities with a real-world three-tier application consisting of:
- **Frontend**: Nginx serving static HTML with API proxy
- **Backend**: Node.js Express API with database and cache integration
- **Database**: PostgreSQL for persistent storage and Redis for caching

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│   Backend   │────▶│  Database   │
│   (Nginx)   │     │  (Node.js)  │     │ (PostgreSQL)│
│             │     │             │     │   + Redis   │
└─────────────┘     └─────────────┘     └─────────────┘
       │                    │                    │
       └────────────────────┴────────────────────┘
                            │
                     ┌──────────────┐
                     │  LocalStack  │
                     │ (S3, SSM,    │
                     │  CloudWatch) │
                     └──────────────┘
```

## Features Tested

### 1. Multi-Container Task Definitions
- Database task with PostgreSQL and Redis containers
- Backend task with environment variables and secrets
- Frontend task with Nginx configuration

### 2. Service Discovery
- Database service registered as `database-service.kecs.local`
- Backend service registered as `backend-service.kecs.local`
- Service-to-service communication via DNS

### 3. LocalStack Integration
- **S3**: File upload and listing
- **SSM Parameter Store**: Secure password storage
- **CloudWatch Logs**: Centralized logging for all containers
- **Service Discovery**: Cloud Map compatibility

### 4. Advanced Scenarios
- **Rolling Updates**: Zero-downtime deployments
- **Auto-scaling**: Horizontal scaling under load
- **Failure Recovery**: Automatic task restart on failure
- **Load Distribution**: Traffic routing across multiple instances

## Running Tests

### Prerequisites
- Docker installed and running
- Go 1.21 or later
- AWS CLI v2 (for LocalStack interaction)

### Build and Test
```bash
# Build Docker images
make build

# Run all tests
make test

# Run specific test
make test-one TEST=TestThreeTierApp

# Run locally for development (uses compose.yaml)
make run-local

# Test local deployment
make test-local
```

### Local Development
```bash
# Start services
make run-local

# View logs
make logs

# Test endpoints
curl http://localhost:80/health         # Frontend health
curl http://localhost:80/api/health     # Backend health via proxy
curl http://localhost:80/api/users      # User API
curl http://localhost:80/api/services   # Service discovery

# Stop services
make stop-local
```

## Test Scenarios

### 1. Full Stack Deployment
Tests the deployment and integration of all three tiers:
- Creates database service with PostgreSQL and Redis
- Creates backend service that connects to database
- Creates frontend service with load balancer
- Verifies end-to-end connectivity
- Tests data persistence through cache

### 2. Rolling Updates
Tests zero-downtime deployment:
- Updates backend to new version
- Monitors health during update
- Verifies no request failures
- Confirms new version is running

### 3. Auto-scaling
Tests horizontal scaling:
- Scales backend service to 3 instances
- Verifies all instances are running
- Tests load distribution across instances
- Confirms service discovery updates

### 4. Failure Recovery
Tests resilience and recovery:
- Simulates database failure
- Verifies backend reports unhealthy
- Waits for automatic recovery
- Confirms system returns to healthy state

## Application Endpoints

### Frontend (Port 80)
- `/` - Interactive web UI
- `/health` - Health check

### Backend (Port 3000)
- `/health` - Health check with dependency status
- `/api/users` - User CRUD operations
- `/api/files` - S3 file operations
- `/api/services` - Service discovery information

## Environment Variables

### Backend
- `DB_HOST` - PostgreSQL host
- `DB_PORT` - PostgreSQL port
- `DB_NAME` - Database name
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password (from SSM)
- `REDIS_HOST` - Redis host
- `REDIS_PORT` - Redis port
- `AWS_ENDPOINT_URL` - LocalStack endpoint
- `S3_BUCKET` - S3 bucket name

## Troubleshooting

### Common Issues

1. **Docker build fails**
   - Ensure Docker daemon is running
   - Check available disk space
   - Verify network connectivity

2. **Tests timeout**
   - Increase timeout in test configuration
   - Check container logs with `make logs`
   - Verify LocalStack is healthy

3. **Service discovery not working**
   - Ensure KECS is running with LocalStack integration
   - Check service registration in CloudWatch logs
   - Verify DNS resolution in containers

### Debug Commands
```bash
# Check running containers
docker ps

# View container logs
docker logs <container-id>

# Access container shell
docker exec -it <container-id> /bin/sh

# Test service discovery
docker exec <backend-container> nslookup database-service.kecs.local
```