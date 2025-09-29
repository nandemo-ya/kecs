# PostgreSQL Storage Implementation

This package provides a PostgreSQL-based implementation of the KECS storage interface.

## Features

- Full implementation of all storage interfaces
- Transaction support for atomic operations
- Proper NULL value handling
- Pagination support with nextToken
- Connection pooling and configuration
- Comprehensive error handling

## Testing

### Prerequisites

- Docker and Docker Compose
- Go 1.21+

### Running Tests

1. **Using Make (recommended)**:
   ```bash
   # From project root
   make test-postgres
   ```

2. **Manual setup**:
   ```bash
   # Start PostgreSQL container
   docker-compose -f controlplane/docker-compose.test.yml up -d

   # Run tests
   cd controlplane
   TEST_POSTGRES=true go test -v ./internal/storage/postgres/...

   # Stop PostgreSQL container
   docker-compose -f ../controlplane/docker-compose.test.yml down
   ```

3. **Using existing PostgreSQL**:
   ```bash
   # Set environment variables
   export TEST_POSTGRES=true
   export POSTGRES_HOST=localhost
   export POSTGRES_PORT=5432
   export POSTGRES_USER=kecs_test
   export POSTGRES_PASSWORD=kecs_test
   export POSTGRES_DB=kecs_test

   # Run tests
   go test -v ./internal/storage/postgres/...
   ```

## Environment Variables

The following environment variables can be used to configure the test database connection:

- `TEST_POSTGRES`: Set to "true" to enable PostgreSQL tests
- `POSTGRES_HOST`: Database host (default: localhost)
- `POSTGRES_PORT`: Database port (default: 5432)
- `POSTGRES_USER`: Database user (default: kecs_test)
- `POSTGRES_PASSWORD`: Database password (default: kecs_test)
- `POSTGRES_DB`: Database name (default: kecs_test)

## Implementation Status

### Completed Stores

- ✅ ClusterStore
- ✅ ServiceStore
- ✅ TaskStore
- ✅ TaskDefinitionStore
- ✅ AccountSettingStore
- ✅ TaskSetStore
- ✅ ContainerInstanceStore
- ✅ AttributeStore
- ✅ ELBv2Store (partial - load balancers and target groups)
- ✅ TaskLogStore

### Pending

- ⏳ ELBv2Store (listeners, rules, targets)
- ⏳ Complete test coverage for all stores
- ⏳ Integration with storage factory
- ⏳ PostgreSQL connection configuration in CLI

## Database Schema

The PostgreSQL storage uses the following tables:

- `clusters` - ECS cluster information
- `services` - ECS service definitions
- `tasks` - Running and stopped tasks
- `task_definitions` - Task definition revisions
- `account_settings` - Account-level settings
- `task_sets` - Task sets for services
- `container_instances` - Container instance registrations
- `attributes` - Custom attributes for resources
- `elbv2_load_balancers` - Load balancer configurations
- `elbv2_target_groups` - Target group configurations
- `task_logs` - Task execution logs

Each table includes appropriate indexes for query optimization and foreign key constraints where applicable.