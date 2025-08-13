# KECS Configuration Guide

KECS supports multiple configuration methods: configuration files, environment variables, and command-line flags.

## Default Configuration

As of the latest version, KECS has the following default settings:

### LocalStack
- **Enabled by default**: LocalStack is now enabled by default to provide AWS service emulation
- **Traefik integration**: Traefik is enabled by default for LocalStack routing

### Features
- **Traefik**: Enabled by default for advanced routing capabilities
- **Auto-recovery**: Enabled by default to recover state after restarts

## Disabling LocalStack and Traefik

If you need to disable LocalStack or Traefik (e.g., for testing or specific deployments), you can use the following methods:

### Method 1: Environment Variables

```bash
# Disable LocalStack
export KECS_LOCALSTACK_ENABLED=false

# Disable Traefik
export KECS_FEATURES_TRAEFIK=false

# Disable both
export KECS_LOCALSTACK_ENABLED=false
export KECS_FEATURES_TRAEFIK=false

# Run KECS
kecs server
```

### Method 2: Configuration File

Create a configuration file (e.g., `kecs.yaml`):

```yaml
# Disable LocalStack and Traefik
localstack:
  enabled: false
  useTraefik: false

features:
  traefik: false
```

Then run KECS with the config file:

```bash
kecs server --config kecs.yaml
```

### Method 3: Command-line Flags

```bash
# Disable LocalStack via flag
kecs server --localstack-enabled=false
```

## Configuration Priority

Configuration is applied in the following order (highest priority first):
1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Common Configuration Scenarios

### Testing/CI Environment

For testing environments where you don't need LocalStack:

```yaml
# test-config.yaml
localstack:
  enabled: false

features:
  traefik: false
  testMode: true
```

### Production-like Environment

For a production-like setup with all features:

```yaml
# prod-config.yaml
server:
  port: 8080
  adminPort: 8081
  logLevel: info

localstack:
  enabled: true
  useTraefik: true
  services:
    - iam
    - logs
    - ssm
    - secretsmanager
    - s3
  persistence: true

features:
  traefik: true
  autoRecoverState: true

kubernetes:
  keepClustersOnShutdown: true
```

### Minimal Setup

For a minimal setup without external dependencies:

```bash
# Set environment variables
export KECS_LOCALSTACK_ENABLED=false
export KECS_FEATURES_TRAEFIK=false
export KECS_FEATURES_AUTO_RECOVER_STATE=false

# Run KECS
kecs server
```

## Verifying Configuration

You can verify the current configuration using the admin API:

```bash
# Check current configuration
curl http://localhost:8081/config | jq .

# Check specific settings
curl -s http://localhost:8081/config | jq '.localstack.enabled'
curl -s http://localhost:8081/config | jq '.features.traefik'
```

## Environment Variable Reference

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `KECS_LOCALSTACK_ENABLED` | Enable/disable LocalStack | `true` |
| `KECS_LOCALSTACK_USE_TRAEFIK` | Enable/disable Traefik for LocalStack | `true` |
| `KECS_FEATURES_TRAEFIK` | Enable/disable Traefik feature | `true` |
| `KECS_FEATURES_TEST_MODE` | Enable/disable test mode | `false` |
| `KECS_FEATURES_AUTO_RECOVER_STATE` | Enable/disable state recovery | `true` |
| `KECS_SERVER_PORT` | API server port | `8080` |
| `KECS_SERVER_ADMIN_PORT` | Admin server port | `8081` |
| `KECS_SERVER_DATA_DIR` | Data directory path | `~/.kecs/data` |
| `KECS_SERVER_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |

## Notes

- LocalStack provides AWS service emulation for local development
- Traefik enables advanced routing capabilities for ELBv2 features
- Disabling these features may limit some functionality but can be useful for testing or lightweight deployments
- Configuration changes require a server restart to take effect