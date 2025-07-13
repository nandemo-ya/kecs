# KECS Configuration Files

This directory contains symbolic links to the actual configuration files located in `controlplane/configs/`.

## Available Configurations

- `production.yaml` - Production configuration with LocalStack enabled by default
- `development.yaml` - Development configuration (symlinked to production.yaml)
- `test.yaml` - Test configuration with LocalStack disabled for unit tests

## Why Symbolic Links?

To avoid confusion and maintenance issues with duplicate configuration files, we use symbolic links to point to the canonical configuration files in the controlplane directory. This ensures:

1. Single source of truth for configurations
2. Consistency across all usage patterns
3. Easy maintenance and updates

## Usage

```bash
# Start KECS with production configuration
./bin/kecs server --config configs/production.yaml

# Start KECS with test configuration
./bin/kecs server --config configs/test.yaml

# Or use the full path
./bin/kecs server --config controlplane/configs/production.yaml
```

## Modifying Configurations

To modify configurations, edit the actual files in `controlplane/configs/`. The changes will be automatically reflected through the symbolic links.

## Configuration Structure

See `controlplane/configs/production.yaml` for the full configuration structure with comments.