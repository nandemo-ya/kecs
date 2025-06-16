# LocalStack Version Management

KECS allows flexible version management for LocalStack deployment, enabling users to specify particular versions for testing and compatibility purposes.

## Configuration Options

### Via Configuration File

In your `production.yaml` or cluster configuration:

```yaml
localstack:
  enabled: true
  image: "localstack/localstack"
  version: "4.5"  # Specify the desired version
  # version: "latest"  # Default if not specified
```

### Via CLI Commands

When using the `kecs localstack` commands, you can specify the version:

```bash
# Start LocalStack with a specific version
kecs localstack start --version 4.5

# Use a custom image repository
kecs localstack start --image my-registry/localstack --version custom-tag
```

## Version Compatibility

### Recommended Versions

- **Production**: Use specific version tags (e.g., `4.5`) for stability
- **Development**: `latest` tag is acceptable but may introduce breaking changes
- **Testing**: Match the version used in your CI/CD pipeline

### Version Features

Different LocalStack versions support different AWS service features:

- **v4.0+**: Latest features with improved AWS parity and performance
- **v3.0+**: Full support for all services with improved performance
- **v2.3+**: Stable support for core services (IAM, S3, Lambda, etc.)
- **v1.x**: Legacy version, not recommended for new deployments

## Managing Version Updates

### Checking Current Version

```bash
kecs localstack status
```

Output includes version information:
```
LocalStack Status:
  Running: true
  Healthy: true
  Endpoint: http://localstack.aws-services.svc.cluster.local:4566
  Version: 4.5
```

### Upgrading LocalStack

To upgrade to a new version:

```bash
# Stop the current instance
kecs localstack stop

# Start with new version
kecs localstack start --version 4.6
```

### Rolling Back

If issues occur with a new version:

```bash
# Revert to previous version
kecs localstack restart --version 4.5
```

## Best Practices

1. **Version Pinning**: Always pin to specific versions in production
2. **Testing**: Test version upgrades in development environments first
3. **Persistence**: Enable persistence when upgrading to preserve data
4. **Documentation**: Document the LocalStack version in your project README

## Troubleshooting

### Version Conflicts

If you encounter compatibility issues:

1. Check the LocalStack changelog for breaking changes
2. Verify service-specific version requirements
3. Review KECS logs for detailed error messages

### Image Pull Issues

For custom registries or specific versions:

```bash
# Pre-pull the image
docker pull localstack/localstack:4.5

# Or use a private registry
kecs localstack start --image private.registry.com/localstack --version 4.5
```