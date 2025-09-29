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

The tests use [Testcontainers](https://www.testcontainers.org/) to automatically spin up a PostgreSQL container for testing. This ensures tests run against a real PostgreSQL instance without requiring any manual setup.

### Prerequisites

- Docker must be installed and running
- Go 1.21+

### Running Tests

```bash
# From project root
make test-postgres

# Or directly with go test
cd controlplane
go test -v ./internal/storage/postgres/...

# Run with specific timeout
go test -v -timeout 60s ./internal/storage/postgres/...
```

The tests will automatically:
1. Start a PostgreSQL container using Testcontainers
2. Initialize the database schema
3. Run all tests against the real PostgreSQL instance
4. Clean up the container after tests complete

### Test Architecture

- **BeforeSuite**: Starts a single PostgreSQL container shared across all tests
- **BeforeEach**: Cleans up test data before each test
- **AfterSuite**: Terminates the PostgreSQL container

This approach ensures:
- Fast test execution (single container for all tests)
- Test isolation (data cleaned between tests)
- Real PostgreSQL compatibility testing
- No manual database setup required

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

## Using with KECS

To use PostgreSQL storage with KECS (when implemented):

```bash
# Start KECS with PostgreSQL storage
kecs start --storage postgres --postgres-url "postgres://user:pass@localhost:5432/kecs"

# Or use environment variable
export KECS_STORAGE=postgres
export KECS_POSTGRES_URL="postgres://user:pass@localhost:5432/kecs"
kecs start
```