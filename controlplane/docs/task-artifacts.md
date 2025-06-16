# Task Artifacts in KECS

KECS supports downloading artifacts (files) before starting task containers. This feature enables tasks to download configuration files, static assets, or other resources from S3, HTTP, or HTTPS sources.

## Overview

Task artifacts are downloaded using init containers in Kubernetes. When a task definition includes artifacts, KECS automatically:

1. Creates init containers to download the artifacts
2. Mounts shared volumes between init and main containers
3. Downloads artifacts to the specified paths
4. Validates checksums if specified
5. Sets file permissions if specified

## Supported Features

- **S3 Downloads**: Download files from S3 buckets (via LocalStack)
- **HTTP/HTTPS Downloads**: Download files from web servers
- **Checksum Validation**: SHA256 and MD5 checksum verification
- **File Permissions**: Set specific file permissions on downloaded files
- **LocalStack Integration**: Full support for LocalStack S3 service

## Task Definition Example

```json
{
  "family": "my-app",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "memory": 512,
      "artifacts": [
        {
          "name": "config",
          "artifactUrl": "s3://my-bucket/config/app.json",
          "targetPath": "/config/app.json",
          "type": "s3",
          "permissions": "0644"
        },
        {
          "name": "static-assets",
          "artifactUrl": "https://cdn.example.com/assets.tar.gz",
          "targetPath": "/static/assets.tar.gz",
          "type": "https",
          "checksum": "abc123...",
          "checksumType": "sha256"
        }
      ]
    }
  ]
}
```

## Artifact Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique name for the artifact |
| `artifactUrl` | string | Yes | URL to download the artifact from (s3://, http://, https://) |
| `targetPath` | string | Yes | Path within the container where the artifact will be placed |
| `type` | string | No | Type of artifact source (s3, http, https). Auto-detected from URL if not specified |
| `permissions` | string | No | File permissions in octal format (e.g., "0644") |
| `checksum` | string | No | Expected checksum of the file |
| `checksumType` | string | No | Type of checksum (sha256, md5). Required if checksum is specified |

## How It Works

### 1. Init Container Creation

For each container with artifacts, KECS creates an init container that:
- Downloads all artifacts for that container
- Validates checksums if specified
- Sets file permissions if specified
- Stores files in a shared volume

### 2. Volume Mounting

- Init containers mount an EmptyDir volume at `/artifacts`
- Main containers mount the same volume (read-only) at `/artifacts`
- Artifacts are accessible at `/artifacts/<targetPath>`

### 3. S3 Integration with LocalStack

When downloading from S3:
- Uses LocalStack S3 endpoint
- Credentials are automatically configured (AWS_ACCESS_KEY_ID=test, AWS_SECRET_ACCESS_KEY=test)
- Supports all S3 URL formats: `s3://bucket/key`

## Example: Configuration File

```json
{
  "containerDefinitions": [
    {
      "name": "web-server",
      "image": "nginx:latest",
      "artifacts": [
        {
          "name": "nginx-config",
          "artifactUrl": "s3://configs/nginx.conf",
          "targetPath": "/etc/nginx/nginx.conf",
          "permissions": "0644"
        }
      ],
      "command": ["nginx", "-c", "/artifacts/etc/nginx/nginx.conf"]
    }
  ]
}
```

## Example: Multiple Artifacts

```json
{
  "containerDefinitions": [
    {
      "name": "app",
      "image": "myapp:latest",
      "artifacts": [
        {
          "name": "app-config",
          "artifactUrl": "s3://configs/app/config.json",
          "targetPath": "/config/config.json"
        },
        {
          "name": "certificates",
          "artifactUrl": "https://vault.example.com/certs.tar.gz",
          "targetPath": "/certs/bundle.tar.gz",
          "checksum": "d2d2d2...",
          "checksumType": "sha256"
        },
        {
          "name": "static-data",
          "artifactUrl": "s3://data/static/data.db",
          "targetPath": "/data/app.db",
          "permissions": "0600"
        }
      ]
    }
  ]
}
```

## Testing with LocalStack

1. Upload a file to LocalStack S3:
```bash
aws --endpoint-url=http://localhost:4566 s3 mb s3://test-bucket
aws --endpoint-url=http://localhost:4566 s3 cp config.json s3://test-bucket/config.json
```

2. Create a task definition with the artifact:
```json
{
  "family": "test-app",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "busybox",
      "artifacts": [
        {
          "name": "config",
          "artifactUrl": "s3://test-bucket/config.json",
          "targetPath": "/config/app.json"
        }
      ],
      "command": ["cat", "/artifacts/config/app.json"]
    }
  ]
}
```

3. Run the task and verify the artifact was downloaded.

## Current Limitations

- **Init Container Image**: Currently uses `busybox:latest`. In production, a custom image with AWS CLI should be used.
- **S3 Authentication**: Uses hardcoded test credentials for LocalStack. Production would need proper IAM integration.
- **Download Script**: Uses basic wget/curl commands. Production should use AWS CLI for S3.
- **Error Handling**: Basic error handling. Production needs retry logic and better error reporting.

## Implementation Details

The artifact support is implemented in:
- `internal/types/task_definition.go`: Artifact type definition
- `internal/artifacts/manager.go`: Artifact download manager
- `internal/converters/task_converter.go`: Init container generation
- `internal/integrations/s3/`: S3 integration with LocalStack

The implementation provides a solid foundation for task artifacts, enabling configuration management and asset distribution in KECS.