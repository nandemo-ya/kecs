# Task Artifacts in KECS

KECS supports downloading artifacts for ECS tasks, allowing containers to access configuration files, static assets, or other resources from S3 or HTTP/HTTPS sources before the main container starts.

## Overview

Task artifacts enable you to:
- Download configuration files from S3 before container startup
- Fetch static assets or binaries from HTTP/HTTPS URLs
- Set proper file permissions on downloaded artifacts
- Validate artifact integrity with checksums
- Access artifacts through a shared volume in your containers

## How It Works

When a task definition includes artifacts:

1. KECS creates init containers that run before the main containers
2. Each init container downloads the specified artifacts to a shared volume
3. The main containers can access these artifacts at `/artifacts/*`
4. Artifacts are downloaded once per task and shared among containers

## Task Definition Format

Add an `artifacts` array to any container definition:

```json
{
  "containerDefinitions": [
    {
      "name": "webapp",
      "image": "nginx:latest",
      "artifacts": [
        {
          "name": "app-config",
          "artifactUrl": "s3://my-bucket/configs/app.conf",
          "type": "s3",
          "targetPath": "config/app.conf",
          "permissions": "0644",
          "checksum": "sha256:abc123...",
          "checksumType": "sha256"
        },
        {
          "name": "static-assets",
          "artifactUrl": "https://example.com/assets.tar.gz",
          "type": "https",
          "targetPath": "assets/assets.tar.gz",
          "permissions": "0755"
        }
      ]
    }
  ]
}
```

### Artifact Fields

- **name** (required): A unique name for the artifact
- **artifactUrl** (required): The URL to download from (s3://, http://, or https://)
- **targetPath** (required): Where to place the artifact within `/artifacts/`
- **type** (optional): Explicitly specify the type ("s3", "http", "https")
- **permissions** (optional): Unix file permissions (e.g., "0644")
- **checksum** (optional): Expected checksum for validation
- **checksumType** (optional): Type of checksum ("sha256" or "md5")

## S3 Integration

### Using LocalStack

KECS integrates with LocalStack for local S3 emulation:

1. Ensure LocalStack is running with S3 enabled
2. Configure KECS to use LocalStack (see LocalStack integration docs)
3. Upload artifacts to LocalStack S3:
   ```bash
   aws s3 cp app.conf s3://my-bucket/configs/ --endpoint-url http://localhost:4566
   ```

### AWS Credentials

When using LocalStack, KECS automatically sets:
- `AWS_ACCESS_KEY_ID=test`
- `AWS_SECRET_ACCESS_KEY=test`
- `AWS_DEFAULT_REGION=<your-configured-region>`

For production use with real S3, configure appropriate IAM roles.

## Implementation Details

### Init Containers

For each container with artifacts, KECS creates an init container:
- Name: `artifact-downloader-<container-name>`
- Image: `busybox:latest` (configurable in production)
- Volume: `artifacts-<container-name>` mounted at `/artifacts`

### Download Scripts

The init container runs a shell script that:
1. Creates necessary directories
2. Downloads artifacts using wget (HTTP/HTTPS) or AWS CLI (S3)
3. Sets file permissions if specified
4. Validates checksums if provided

### Volume Sharing

Each container with artifacts gets:
- An EmptyDir volume for storing artifacts
- A volume mount at `/artifacts` (read-only)
- Access to all artifacts defined for that container

## Examples

### Configuration File from S3

```json
{
  "artifacts": [
    {
      "name": "nginx-config",
      "artifactUrl": "s3://config-bucket/nginx/nginx.conf",
      "targetPath": "nginx/nginx.conf",
      "permissions": "0644"
    }
  ]
}
```

Access in container: `/artifacts/nginx/nginx.conf`

### Binary from HTTPS

```json
{
  "artifacts": [
    {
      "name": "custom-tool",
      "artifactUrl": "https://releases.example.com/tool-v1.0.0-linux-amd64",
      "targetPath": "bin/tool",
      "permissions": "0755",
      "checksum": "sha256:1234567890abcdef...",
      "checksumType": "sha256"
    }
  ]
}
```

Access in container: `/artifacts/bin/tool`

### Multiple Artifacts

```json
{
  "artifacts": [
    {
      "name": "app-config",
      "artifactUrl": "s3://configs/app/config.yaml",
      "targetPath": "config/app.yaml",
      "permissions": "0640"
    },
    {
      "name": "ssl-cert",
      "artifactUrl": "s3://certs/app.crt",
      "targetPath": "certs/app.crt",
      "permissions": "0644"
    },
    {
      "name": "ssl-key",
      "artifactUrl": "s3://certs/app.key",
      "targetPath": "certs/app.key",
      "permissions": "0600"
    }
  ]
}
```

## Limitations

Current limitations in KECS:

1. **S3 Download**: Currently shows placeholder output. Full AWS CLI integration planned.
2. **Init Container Image**: Uses `busybox:latest`. Production deployments should use a custom image with AWS CLI.
3. **Checksum Validation**: Implemented in artifact manager but not yet in init containers.
4. **Error Handling**: Basic error handling. Production use requires more robust retry logic.

## Future Enhancements

Planned improvements:

1. Custom init container image with pre-installed AWS CLI
2. Support for additional artifact sources (Git, GCS, etc.)
3. Artifact caching across tasks
4. Progress reporting for large downloads
5. Integration with ECS task IAM roles for S3 access
6. Support for artifact encryption/decryption

## Testing

Test artifact functionality:

```bash
# Register task definition
cd examples
./test-task-with-artifacts.sh

# Upload test file to LocalStack S3
aws s3 mb s3://my-bucket --endpoint-url http://localhost:4566
echo "test config" | aws s3 cp - s3://my-bucket/configs/app.conf --endpoint-url http://localhost:4566

# Run task
curl -X POST "http://localhost:8080/v1/RunTask" \
  -H "Content-Type: application/x-amz-json-1.1" \
  -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.RunTask" \
  -d '{
    "cluster": "default",
    "taskDefinition": "webapp-with-artifacts"
  }'
```

Check Kubernetes pods to see init containers and artifact volumes:

```bash
kubectl get pods -n kecs-default
kubectl describe pod ecs-task-<task-id> -n kecs-default
```