# Docker Compose Examples for KECS

This directory contains Docker Compose examples for running KECS.

## Basic Usage

### Single Instance

Run a single KECS instance:

```bash
docker-compose up -d
```

This starts KECS on ports 8080 (API) and 8081 (Admin).

### Multiple Instances

Run multiple KECS instances using profiles:

```bash
docker-compose --profile multi up -d
```

This starts:
- `kecs-dev` on ports 8080/8081
- `kecs-staging` on ports 8090/8091

### With Application

Run KECS with an example application:

```bash
docker-compose --profile with-app up -d
```

## Environment Variables

### Required for KECS

- `KECS_CONTAINER_MODE=true` - Enables container mode
- `KECS_DATA_DIR=/data` - Data directory inside container

### Optional Configuration

- `KECS_LOG_LEVEL` - Log level (debug, info, warn, error)
- `KECS_TEST_MODE` - Enable test mode
- `KECS_DEFAULT_REGION` - Default AWS region
- `KECS_ACCOUNT_ID` - AWS account ID

### For Applications Using KECS

- `AWS_ENDPOINT_URL=http://kecs:8080` - KECS endpoint
- `AWS_DEFAULT_REGION=us-east-1` - AWS region
- `AWS_ACCESS_KEY_ID=dummy` - Dummy credentials
- `AWS_SECRET_ACCESS_KEY=dummy` - Dummy credentials

## Volumes

Data is persisted in Docker volumes:
- `kecs-data` - Main instance data
- `kecs-dev-data` - Dev instance data  
- `kecs-staging-data` - Staging instance data

## Health Checks

KECS includes health checks that:
- Test the admin endpoint at `/health`
- Run every 30 seconds
- Allow 40 seconds for startup
- Retry 3 times before marking unhealthy

## Networking

All services run in the `kecs-network` network, allowing:
- Service discovery by container name
- Isolation from other Docker networks
- Communication between KECS and applications

## Commands

### View Logs

```bash
docker-compose logs -f kecs
```

### Check Status

```bash
docker-compose ps
```

### Stop Services

```bash
docker-compose down
```

### Remove Volumes

```bash
docker-compose down -v
```

## Customization

### Using Local Build

To use a locally built image:

1. Build the image:
   ```bash
   cd ../../controlplane
   docker build -t kecs:local .
   ```

2. Update `docker-compose.yml`:
   ```yaml
   image: kecs:local
   ```

### Custom Ports

Modify the ports section:
```yaml
ports:
  - "9080:8080"  # Custom API port
  - "9081:8081"  # Custom Admin port
```

### Additional Environment Variables

Add more environment variables:
```yaml
environment:
  - KECS_CONTAINER_MODE=true
  - KECS_LOG_LEVEL=debug
  - KECS_FEATURE_X=enabled
```

## Troubleshooting

### Container Won't Start

Check logs:
```bash
docker-compose logs kecs
```

### Port Conflicts

Change ports in `docker-compose.yml` or stop conflicting services.

### Permission Issues

Ensure Docker daemon is accessible:
```bash
sudo usermod -aG docker $USER
```

## Integration with CI/CD

This compose file can be used in CI/CD:

```bash
# Start services
docker-compose up -d

# Wait for health
docker-compose exec kecs curl -f http://localhost:8081/health

# Run tests
./run-tests.sh

# Cleanup
docker-compose down -v
```