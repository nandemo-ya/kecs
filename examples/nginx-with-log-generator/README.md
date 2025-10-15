# Nginx with Log Generator Example

This example demonstrates log generation and viewing in KECS using a multi-container task definition with:
- An nginx web server container
- A sidecar container that continuously sends HTTP requests to nginx every 5 seconds

This setup ensures continuous log generation without needing port forwarding or external access, making it ideal for testing CloudWatch Logs integration.

## Overview

- **Purpose**: Demonstrate CloudWatch Logs integration with multi-container tasks
- **Components**:
  - nginx web server
  - curl-based log generator (sidecar)
- **Features**:
  - Automatic log generation every 5 seconds
  - CloudWatch Logs integration
  - Multi-container task coordination

## Prerequisites

1. KECS running locally
2. Terraform installed (>= 1.0)
3. AWS CLI configured

## Quick Start with Terraform (Recommended)

### 1. Start KECS

```bash
# Start KECS instance
kecs start

# Wait for KECS to be ready
kecs list
```

### 2. Deploy Infrastructure with Terraform

```bash
# Initialize Terraform
terraform init

# Review the planned changes
terraform plan

# Apply the configuration
terraform apply

# Type 'yes' when prompted
```

This will create:
- ECS Cluster: `nginx-with-log-generator`
- CloudWatch Logs Log Group: `/ecs/nginx-with-log-generator` (7 days retention)

### 3. Verify Resources

```bash
# Verify ECS cluster
aws ecs describe-clusters \
  --cluster nginx-with-log-generator \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# Verify CloudWatch Logs log group
aws logs describe-log-groups \
  --log-group-name-prefix /ecs/nginx-with-log-generator \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# View Terraform outputs
terraform output
```

### 4. View Terraform Outputs

```bash
# Show all outputs
terraform output

# Show specific output
terraform output cluster_name
terraform output log_group_name
```

After applying, Terraform provides several outputs:

- `cluster_name`: ECS cluster name
- `cluster_arn`: ECS cluster ARN
- `log_group_name`: CloudWatch Logs log group name
- `log_group_arn`: CloudWatch Logs log group ARN

### 5. Terraform Configuration

You can customize the configuration by creating a `terraform.tfvars` file:

```hcl
aws_region     = "us-east-1"
kecs_endpoint  = "http://localhost:5373"
cluster_name   = "nginx-with-log-generator"
service_name   = "nginx-with-log-generator"
environment    = "development"
```

Or override via command line:

```bash
terraform apply -var="cluster_name=my-cluster" -var="environment=staging"
```

## Deployment

### Using AWS CLI

```bash
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1

# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --region us-east-1

# Create service
aws ecs create-service \
  --cli-input-json file://service_def.json \
  --region us-east-1
```

## Viewing Logs

### Using KECS TUI

```bash
kecs
```

Navigate to the task and press 'l' to view logs.

### Using AWS CLI (CloudWatch Logs)

```bash
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1

# List log streams to find the stream name
aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1

# Get nginx log stream name
LOG_STREAM=$(aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1 \
  --query 'logStreams[?contains(logStreamName, `nginx/`)].logStreamName' \
  --output text)

# View nginx logs
aws logs get-log-events \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --log-stream-name "$LOG_STREAM" \
  --region us-east-1 \
  --limit 20

# Get log-generator log stream name
LOG_STREAM=$(aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1 \
  --query 'logStreams[?contains(logStreamName, `log-generator/`)].logStreamName' \
  --output text)

# View log-generator logs
aws logs get-log-events \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --log-stream-name "$LOG_STREAM" \
  --region us-east-1 \
  --limit 20

# Tail logs (using --start-time with recent timestamp)
aws logs get-log-events \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --log-stream-name "$LOG_STREAM" \
  --region us-east-1 \
  --start-time $(date -u -v-5M +%s)000
```

### Using kubectl

```bash
# Get the pod name
kubectl get pods -n nginx-with-log-generator-us-east-1

# View nginx logs
kubectl logs -n nginx-with-log-generator-us-east-1 <pod-name> nginx

# View log-generator logs
kubectl logs -n nginx-with-log-generator-us-east-1 <pod-name> log-generator

# Follow logs in real-time
kubectl logs -n nginx-with-log-generator-us-east-1 <pod-name> nginx -f
kubectl logs -n nginx-with-log-generator-us-east-1 <pod-name> log-generator -f
```

## Expected Behavior

The log-generator container will:
1. Send an HTTP request to the nginx container every 5 seconds
2. Log "Sending request to nginx..." before each request
3. Log "Request successful" after a successful request

The nginx container will log access logs for each request:
- Standard nginx access logs showing the internal requests from the log-generator

## Verification

### Check Service Status

```bash
aws ecs describe-services \
  --cluster nginx-with-log-generator \
  --services nginx-with-log-generator \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'
```

### Check Logs are Being Generated

```bash
# Check log streams exist
aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Verify logs are being written (should see recent timestamps)
aws logs describe-log-streams \
  --log-group-name "/ecs/nginx-with-log-generator" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'logStreams[*].{Stream:logStreamName,LastEvent:lastEventTime}' \
  --output table
```

## Troubleshooting

### Terraform Issues

#### Connection Refused

If Terraform can't connect to KECS:

```bash
# Check KECS is running
kecs status

# Verify endpoint is accessible
curl http://localhost:5373/health
```

#### Resource Already Exists

If resources were manually created:

```bash
# Option 1: Destroy existing resources first using terraform
terraform destroy

# Option 2: Import existing resources
terraform import aws_ecs_cluster.main nginx-with-log-generator
terraform import aws_cloudwatch_log_group.nginx_logs /ecs/nginx-with-log-generator
```

### Task and Service Issues

#### No Logs Appearing

```bash
# Check if log group exists
aws logs describe-log-groups \
  --log-group-name-prefix /ecs/nginx-with-log-generator \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Check pod status
kubectl get pods -n nginx-with-log-generator-us-east-1

# Check container logs directly
kubectl logs -n nginx-with-log-generator-us-east-1 <pod-name> nginx
kubectl logs -n nginx-with-log-generator-us-east-1 <pod-name> log-generator
```

#### Service Not Starting

```bash
# Check service events
aws ecs describe-services \
  --cluster nginx-with-log-generator \
  --services nginx-with-log-generator \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'services[0].events[0:5]'
```

## Cleanup

### Using Terraform (Recommended)

```bash
# Destroy all infrastructure
terraform destroy

# Type 'yes' when prompted
```

This will remove all resources created by Terraform.

### Manual Cleanup (Alternative)

<details>
<summary>Click to expand manual cleanup instructions</summary>

```bash
# Delete service
aws ecs delete-service \
  --cluster nginx-with-log-generator \
  --service nginx-with-log-generator \
  --force \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/nginx-with-log-generator \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete ECS cluster
aws ecs delete-cluster \
  --cluster nginx-with-log-generator \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

</details>
