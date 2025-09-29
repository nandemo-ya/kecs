# PostgreSQL Storage Backend

KECS supports PostgreSQL as an alternative storage backend to the default DuckDB. This enables better scalability and integration with existing PostgreSQL infrastructure.

## Configuration

### Environment Variables

The storage backend is selected using the `KECS_STORAGE_TYPE` environment variable:

```bash
# Use PostgreSQL backend
export KECS_STORAGE_TYPE=postgresql

# Use DuckDB backend (default)
export KECS_STORAGE_TYPE=duckdb
```

### PostgreSQL Connection

There are two ways to configure the PostgreSQL connection:

#### Method 1: Database URL

Set the complete connection URL using `KECS_DATABASE_URL`:

```bash
export KECS_DATABASE_URL="postgres://username:password@host:port/database?sslmode=disable"
```

Example:
```bash
export KECS_DATABASE_URL="postgres://kecs:secret@localhost:5432/kecs?sslmode=disable"
```

#### Method 2: Individual Environment Variables

Configure the connection using separate environment variables:

```bash
export KECS_POSTGRES_HOST=localhost      # Default: localhost
export KECS_POSTGRES_PORT=5432          # Default: 5432
export KECS_POSTGRES_USER=kecs          # Default: kecs
export KECS_POSTGRES_PASSWORD=secret    # Default: kecs
export KECS_POSTGRES_DATABASE=kecs      # Default: kecs
export KECS_POSTGRES_SSLMODE=disable    # Default: disable
```

If `KECS_DATABASE_URL` is not set, KECS will automatically build the connection URL from these individual variables.

## Running KECS with PostgreSQL

### Prerequisites

1. PostgreSQL server (version 12 or later recommended)
2. Database and user with appropriate permissions

### Setup PostgreSQL

```sql
-- Create database
CREATE DATABASE kecs;

-- Create user
CREATE USER kecs WITH PASSWORD 'your_secure_password';

-- Grant permissions
GRANT ALL PRIVILEGES ON DATABASE kecs TO kecs;
```

### Start KECS with PostgreSQL

```bash
# Using database URL
export KECS_STORAGE_TYPE=postgresql
export KECS_DATABASE_URL="postgres://kecs:your_secure_password@localhost:5432/kecs?sslmode=disable"
./bin/kecs start

# Using individual variables
export KECS_STORAGE_TYPE=postgresql
export KECS_POSTGRES_HOST=localhost
export KECS_POSTGRES_USER=kecs
export KECS_POSTGRES_PASSWORD=your_secure_password
export KECS_POSTGRES_DATABASE=kecs
./bin/kecs start
```

### Using with Docker

When running KECS in Docker with PostgreSQL:

```bash
docker run -d \
  -e KECS_STORAGE_TYPE=postgresql \
  -e KECS_DATABASE_URL="postgres://kecs:password@postgres:5432/kecs" \
  -p 8080:8080 \
  kecs:latest
```

Or with docker-compose:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: kecs
      POSTGRES_USER: kecs
      POSTGRES_PASSWORD: kecs_password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  kecs:
    image: kecs:latest
    environment:
      KECS_STORAGE_TYPE: postgresql
      KECS_DATABASE_URL: postgres://kecs:kecs_password@postgres:5432/kecs?sslmode=disable
    ports:
      - "8080:8080"
      - "5374:5374"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

## Migration from DuckDB

To migrate existing data from DuckDB to PostgreSQL:

1. Export data from DuckDB (future feature - not yet implemented)
2. Import data to PostgreSQL (future feature - not yet implemented)

Currently, migration between storage backends requires recreating all ECS resources.

## Performance Considerations

- **DuckDB**: Better for single-instance deployments, embedded database, no external dependencies
- **PostgreSQL**: Better for multi-instance deployments, centralized storage, existing PostgreSQL infrastructure

## Troubleshooting

### Connection Issues

If you see connection errors:

1. Verify PostgreSQL is running: `pg_isready -h localhost -p 5432`
2. Check credentials: `psql -h localhost -U kecs -d kecs -c "SELECT 1"`
3. Verify network connectivity
4. Check PostgreSQL logs

### SSL/TLS Configuration

For production environments, use SSL:

```bash
export KECS_POSTGRES_SSLMODE=require  # or 'verify-ca', 'verify-full'
```

### Password Security

The password is masked in logs for security. You'll see output like:
```
Using PostgreSQL storage url=postgres://kecs:***@localhost:5432/kecs
```

## Features

Both DuckDB and PostgreSQL backends support the full KECS feature set:

- Clusters management
- Task definitions
- Services
- Tasks
- Container instances
- Account settings
- Task sets
- ELBv2 integration
- Attributes
- Task logs

The storage backend is abstracted through a common interface, ensuring consistent behavior regardless of the chosen backend.