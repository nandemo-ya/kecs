# CLI Commands Reference

KECS provides a comprehensive CLI for managing local ECS environments. This reference covers all available commands and their options.

## Core Commands

### kecs start

Starts a new KECS instance with a k3d cluster.

```bash
kecs start [flags]
```

**Flags:**
- `--instance string`: Instance name (default: auto-generated)
- `--api-port int`: API port for ECS/ELBv2 APIs (default: 5373)
- `--admin-port int`: Admin port for health/metrics (default: 5374)
- `--data-dir string`: Data directory (default: ~/.kecs/data)
- `--config string`: Configuration file path
- `--additional-localstack-services string`: Additional LocalStack services to enable (comma-separated, e.g., `s3,dynamodb,sqs`)
- `--timeout duration`: Timeout for cluster creation (default: 10m)

**LocalStack Services:**

KECS always enables these core AWS services by default:
- `iam` - Identity and Access Management
- `logs` - CloudWatch Logs
- `ssm` - Systems Manager Parameter Store
- `secretsmanager` - Secrets Manager
- `elbv2` - Elastic Load Balancing v2
- `s3` - Simple Storage Service (note: included in defaults)

You can enable additional services using `--additional-localstack-services`:
- `dynamodb` - DynamoDB
- `sqs` - Simple Queue Service
- `sns` - Simple Notification Service
- `kinesis` - Kinesis Data Streams
- `lambda` - Lambda
- `rds` - Relational Database Service
- `ec2` - Elastic Compute Cloud
- And many more (see [LocalStack documentation](https://docs.localstack.cloud/user-guide/aws/feature-coverage/) for full list)

**Examples:**
```bash
# Start with default settings (includes iam, logs, ssm, secretsmanager, elbv2, s3)
kecs start

# Start with custom instance name
kecs start --instance dev

# Start with additional services (DynamoDB and SQS)
kecs start --instance dev --additional-localstack-services dynamodb,sqs

# Start with S3 and DynamoDB for data processing workload
kecs start --instance data-pipeline --additional-localstack-services s3,dynamodb

# Start with custom ports and Lambda support
kecs start --instance staging --api-port 6373 --admin-port 6374 --additional-localstack-services lambda,sns
```

**Using the TUI (Interactive Mode):**

When using `kecs tui`, you can configure additional LocalStack services through the instance creation dialog:

![TUI LocalStack Services Configuration](../images/tui-localstack-services.png)

1. Navigate to "Create New Instance"
2. Enter instance name
3. In "Additional LocalStack Services" field, enter comma-separated service names
4. The UI shows helper text with examples and indicates which services are always enabled
5. Press Create to start the instance

### kecs stop

Stops and removes a KECS instance.

```bash
kecs stop [flags]
```

**Flags:**
- `--instance string`: Instance name to stop (required)

**Examples:**
```bash
# Stop specific instance
kecs stop --instance dev

# Stop another instance
kecs stop --instance staging
```

## Kubernetes Integration

### kecs kubeconfig

Manages kubeconfig for KECS clusters.

```bash
kecs kubeconfig [subcommand] [flags]
```

**Subcommands:**
- `list`: List all available KECS clusters
- `get`: Get kubeconfig for an instance

**Examples:**
```bash
# List all available KECS clusters
kecs kubeconfig list

# Get kubeconfig path
kecs kubeconfig get --instance dev
```

## Utility Commands

### kecs version

Shows version information.

```bash
kecs version [flags]
```

**Flags:**
- `--short`: Show only version number
- `--json`: Output in JSON format

**Example output:**
```
KECS Version: v0.5.0
Git Commit: abc123def
Build Date: 2024-01-15
Go Version: go1.21.5
Platform: darwin/arm64
```

### kecs doctor

Checks system requirements and diagnoses issues.

```bash
kecs doctor [flags]
```

**Checks:**
- Docker daemon status
- k3d installation
- Port availability
- Storage permissions
- Network connectivity

**Example output:**
```
✅ Docker daemon is running
✅ k3d is installed (v5.4.6)
✅ Port 5373 is available
✅ Storage directory is writable
✅ Network connectivity OK

All checks passed!
```

### kecs cleanup

Cleans up orphaned resources.

```bash
kecs cleanup [flags]
```

**Flags:**
- `--dry-run`: Show what would be cleaned
- `--force`: Skip confirmation
- `--all`: Clean all resources including data

**Examples:**
```bash
# Preview cleanup
kecs cleanup --dry-run

# Clean orphaned clusters
kecs cleanup

# Clean everything including data
kecs cleanup --all --force
```

## Environment Variables

KECS respects the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|  
| `KECS_INSTANCE` | Default instance name | - |
| `KECS_API_PORT` | Default API port | 5373 |
| `KECS_ADMIN_PORT` | Default admin port | 5374 |
| `KECS_NAMESPACE` | Kubernetes namespace | kecs-system |
| `KECS_LOG_LEVEL` | Log level (debug, info, warn, error) | info |
| `KECS_STORAGE_PATH` | Storage directory | ~/.kecs/data |
| `KECS_CONFIG_PATH` | Config file path | ~/.kecs/config.yaml |
| `KECS_LOCALSTACK_ENABLED` | Enable LocalStack | true |
| `KECS_FEATURES_TRAEFIK` | Enable Traefik | true |

## AWS CLI Integration

To use AWS CLI with KECS:

```bash
# Set endpoint URL
export AWS_ENDPOINT_URL=http://localhost:5373

# Configure dummy credentials
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_REGION=us-east-1

# Use AWS CLI normally
aws ecs list-clusters
aws elbv2 describe-load-balancers
```

## Troubleshooting

### Common Issues

**Port already in use:**
```bash
# Check what's using the port
lsof -i :5373

# Use different port
kecs start --api-port 6373
```

**k3d cluster not found:**
```bash
# List k3d clusters
k3d cluster list

# Clean up orphaned cluster
k3d cluster delete kecs-<instance>
```

**Permission denied:**
```bash
# Fix Docker permissions
sudo usermod -aG docker $USER
newgrp docker
```

## Port Forwarding Commands

### kecs port-forward start

Creates a port forward to access services or tasks locally.

```bash
kecs port-forward start <type> <target> [flags]
```

**Types:**
- `service` - Forward to an ECS service
- `task` - Forward to a specific task

**Flags:**
- `--local-port int`: Local port to bind (default: auto-assign)
- `--target-port int`: Target container port (default: 80 for services, 8080 for tasks)
- `--tags stringToString`: Tags for task selection (task type only)
- `--no-auto-reconnect`: Disable automatic reconnection on failure

**Examples:**
```bash
# Forward a service
kecs port-forward start service default/nginx --local-port 8080

# Forward to newest task with tags
kecs port-forward start task production --tags app=api,version=v2 --local-port 3000

# Auto-assign local port
kecs port-forward start service staging/web
```

### kecs port-forward list

Lists all active port forwards.

```bash
kecs port-forward list [flags]
```

**Flags:**
- `--format string`: Output format (table, json, yaml)
- `--watch`: Watch for changes in real-time

**Example output:**
```
ID                          TYPE     CLUSTER    TARGET      LOCAL   TARGET   STATUS
svc-default-nginx-1234      service  default    nginx       8080    80       active
task-prod-api-5678          task     production api-task    3000    8080     active
```

### kecs port-forward stop

Stops one or more port forwards.

```bash
kecs port-forward stop <forward-id|--all> [flags]
```

**Flags:**
- `--all`: Stop all active port forwards

**Examples:**
```bash
# Stop specific forward
kecs port-forward stop svc-default-nginx-1234

# Stop all forwards
kecs port-forward stop --all
```


### Debug Mode

Enable debug logging for troubleshooting:

```bash
# Via flag
kecs start --debug

# Via environment variable
export KECS_LOG_LEVEL=debug
kecs start
```

## Next Steps

- [Port Forwarding Guide](/guides/port-forward) - Access services locally
- [Services Guide](/guides/services) - Deploy ECS services
- [ELBv2 Integration](/guides/elbv2-integration) - Configure load balancers
- [TUI Interface](/guides/tui-interface) - Interactive management
- [Troubleshooting Guide](/guides/troubleshooting) - Resolve common issues