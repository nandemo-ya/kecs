# GitHub Actions Integration

KECS provides an official GitHub Action that simplifies setting up KECS in your CI/CD workflows. This guide covers how to use the [KECS Action](https://github.com/marketplace/actions/setup-kecs) for testing ECS workflows in GitHub Actions.

## Overview

The KECS Action automates the entire setup process:
- Installs KECS CLI
- Creates a k3d cluster with KECS control plane
- Configures environment variables
- Sets up kubeconfig
- Provides cleanup capabilities with optional log collection

## Quick Start

Add KECS to your workflow with just a few lines:

```yaml
name: ECS Workflow Test

on: [push]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup KECS
        uses: nandemo-ya/kecs-action@v1
        id: kecs

      - name: Run ECS Tests
        run: |
          # AWS_ENDPOINT_URL is automatically set
          aws ecs create-cluster --cluster-name test
          aws ecs list-clusters

      - name: Cleanup KECS
        if: always()
        uses: nandemo-ya/kecs-action/cleanup@v1
        with:
          instance-name: ${{ steps.kecs.outputs.instance-name }}
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `kecs-version` | KECS version to install | No | `latest` |
| `instance-name` | KECS instance name | No | Auto-generated |
| `api-port` | AWS API port | No | `5373` |
| `admin-port` | Admin API port | No | `5374` |
| `additional-localstack-services` | Additional LocalStack services (comma-separated) | No | `""` |
| `timeout` | Timeout for cluster creation | No | `10m` |
| `debug` | Enable debug logging | No | `false` |

## Outputs

| Output | Description |
|--------|-------------|
| `instance-name` | Actual KECS instance name |
| `endpoint` | KECS API endpoint URL |
| `admin-endpoint` | KECS admin endpoint URL |
| `kubeconfig` | Path to kubeconfig file |

## Environment Variables

The action automatically exports these environment variables for use in subsequent steps:

- `AWS_ENDPOINT_URL`: KECS API endpoint (for AWS CLI)
- `KUBECONFIG`: Path to kubeconfig file (for kubectl access)

For direct API access to KECS, use the `endpoint` output: <code v-pre>${{ steps.kecs.outputs.endpoint }}</code>

## Usage Examples

### Basic ECS Workflow

Test cluster, service, and task operations:

```yaml
- name: Setup KECS
  uses: nandemo-ya/kecs-action@v1
  id: kecs

- name: Configure AWS CLI
  run: |
    export AWS_ACCESS_KEY_ID=test
    export AWS_SECRET_ACCESS_KEY=test

- name: Test ECS Workflow
  run: |
    # Create cluster
    aws ecs create-cluster --cluster-name my-cluster --region us-east-1

    # Register task definition
    aws ecs register-task-definition \
      --family my-app \
      --requires-compatibilities FARGATE \
      --cpu 256 \
      --memory 512 \
      --container-definitions '[{
        "name": "app",
        "image": "nginx:latest",
        "essential": true
      }]' \
      --region us-east-1

    # Create service
    aws ecs create-service \
      --cluster my-cluster \
      --service-name my-service \
      --task-definition my-app \
      --desired-count 1 \
      --launch-type FARGATE \
      --region us-east-1

- name: Cleanup
  if: always()
  uses: nandemo-ya/kecs-action/cleanup@v1
  with:
    instance-name: ${{ steps.kecs.outputs.instance-name }}
```

### With Additional LocalStack Services

Enable S3, DynamoDB, and SQS alongside ECS:

```yaml
- name: Setup KECS with LocalStack
  uses: nandemo-ya/kecs-action@v1
  with:
    additional-localstack-services: s3,dynamodb,sqs

- name: Test with Multiple AWS Services
  run: |
    # Use ECS
    aws ecs create-cluster --cluster-name test --region us-east-1

    # Use S3
    aws s3 mb s3://my-bucket --region us-east-1

    # Use DynamoDB
    aws dynamodb create-table \
      --table-name my-table \
      --attribute-definitions AttributeName=id,AttributeType=S \
      --key-schema AttributeName=id,KeyType=HASH \
      --billing-mode PAY_PER_REQUEST \
      --region us-east-1
```

### Custom Configuration

Specify version, instance name, and ports:

```yaml
- name: Setup KECS
  uses: nandemo-ya/kecs-action@v1
  with:
    kecs-version: v0.0.1-beta.10
    instance-name: my-test-instance
    api-port: 6000
    admin-port: 6001
```

### With kubectl Access

Access Kubernetes resources directly:

```yaml
- name: Setup KECS
  uses: nandemo-ya/kecs-action@v1
  id: kecs

- name: Verify Kubernetes Resources
  run: |
    # Check KECS system pods
    kubectl get pods -n kecs-system

    # Check cluster namespaces
    kubectl get namespaces
```

### Debugging

Enable debug mode for troubleshooting:

```yaml
- name: Setup KECS
  uses: nandemo-ya/kecs-action@v1
  with:
    debug: true
```

## Cleanup Action

The cleanup action ensures resources are properly deleted and optionally collects logs for debugging:

```yaml
- name: Cleanup KECS
  if: always()
  uses: nandemo-ya/kecs-action/cleanup@v1
  with:
    instance-name: ${{ steps.kecs.outputs.instance-name }}
    collect-logs: 'true'
    log-directory: '.kecs-logs'
```

### Cleanup Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `instance-name` | KECS instance name to cleanup | Yes | - |
| `collect-logs` | Collect logs before cleanup | No | `false` |
| `log-directory` | Directory to save logs | No | `.kecs-logs` |

When `collect-logs` is enabled:
- Collects KECS control plane logs
- Collects LocalStack logs
- Collects Traefik logs
- Collects pod statuses and cluster events
- Uploads logs as workflow artifacts (7-day retention)

## Best Practices

### Always Use Cleanup

Use `if: always()` to ensure cleanup runs even when tests fail:

```yaml
- name: Cleanup KECS
  if: always()
  uses: nandemo-ya/kecs-action/cleanup@v1
  with:
    instance-name: ${{ steps.kecs.outputs.instance-name }}
```

### Enable Log Collection on Failure

Collect logs when tests fail for debugging:

```yaml
- name: Cleanup KECS
  if: always()
  uses: nandemo-ya/kecs-action/cleanup@v1
  with:
    instance-name: ${{ steps.kecs.outputs.instance-name }}
    collect-logs: ${{ failure() }}
```

### Use Specific Versions

Pin to specific versions for reproducible builds:

```yaml
- uses: nandemo-ya/kecs-action@v1.0.0  # Specific version
- uses: nandemo-ya/kecs-action@v1      # Latest v1.x.x (recommended)
```

### Configure AWS Credentials

Even though KECS runs locally, AWS CLI requires credentials:

```yaml
- name: Configure AWS credentials
  run: |
    export AWS_ACCESS_KEY_ID=test
    export AWS_SECRET_ACCESS_KEY=test
```

## Matrix Testing

Test across multiple KECS versions:

```yaml
strategy:
  matrix:
    kecs-version: [v0.0.1-beta.9, v0.0.1-beta.10, latest]

steps:
  - uses: nandemo-ya/kecs-action@v1
    with:
      kecs-version: ${{ matrix.kecs-version }}
```

## Troubleshooting

### Port Conflicts

If default ports conflict, specify custom ports:

```yaml
- uses: nandemo-ya/kecs-action@v1
  with:
    api-port: 6000
    admin-port: 6001
```

### Timeout Issues

Increase timeout for slower environments:

```yaml
- uses: nandemo-ya/kecs-action@v1
  with:
    timeout: 15m
```

### Debugging Failures

Enable debug mode and collect logs:

```yaml
- name: Setup KECS
  uses: nandemo-ya/kecs-action@v1
  with:
    debug: true

- name: Cleanup with logs
  if: always()
  uses: nandemo-ya/kecs-action/cleanup@v1
  with:
    instance-name: ${{ steps.kecs.outputs.instance-name }}
    collect-logs: 'true'
```

## Advanced Usage

### Multiple Instances

Run multiple KECS instances in parallel jobs:

```yaml
jobs:
  test-cluster-1:
    runs-on: ubuntu-latest
    steps:
      - uses: nandemo-ya/kecs-action@v1
        with:
          instance-name: cluster-1
          api-port: 5373

  test-cluster-2:
    runs-on: ubuntu-latest
    steps:
      - uses: nandemo-ya/kecs-action@v1
        with:
          instance-name: cluster-2
          api-port: 6373
```

### Reusable Workflows

Create a reusable workflow for KECS testing:

```yaml
# .github/workflows/kecs-test.yml
name: Reusable KECS Test

on:
  workflow_call:
    inputs:
      kecs-version:
        type: string
        default: latest

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: nandemo-ya/kecs-action@v1
        id: kecs
        with:
          kecs-version: ${{ inputs.kecs-version }}

      - run: |
          # Your tests here
          aws ecs list-clusters

      - uses: nandemo-ya/kecs-action/cleanup@v1
        if: always()
        with:
          instance-name: ${{ steps.kecs.outputs.instance-name }}
```

Use in other workflows:

```yaml
# .github/workflows/ci.yml
name: CI

on: [push]

jobs:
  test:
    uses: ./.github/workflows/kecs-test.yml
    with:
      kecs-version: latest
```

## Requirements

- **Runner**: Ubuntu (ubuntu-latest recommended)
- **Docker**: Pre-installed on GitHub Actions runners
- **Resources**: Sufficient for k3d cluster (~2GB memory recommended)

## Links

- [KECS Action on GitHub Marketplace](https://github.com/marketplace/actions/setup-kecs)
- [KECS Action Repository](https://github.com/nandemo-ya/kecs-action)
- [KECS Project](https://github.com/nandemo-ya/kecs)
- [Example Workflows](https://github.com/nandemo-ya/kecs-action/tree/main/.github/workflows)

## Contributing

Found an issue or have a suggestion for the KECS Action? Please open an issue or PR in the [kecs-action repository](https://github.com/nandemo-ya/kecs-action).
