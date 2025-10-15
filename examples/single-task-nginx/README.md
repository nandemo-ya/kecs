# Single Task Nginx Example

This example demonstrates a simple nginx web server deployment using KECS with a single container task.

## Overview

- **Purpose**: Basic web server deployment
- **Components**: Single nginx container
- **Network**: Public IP with security group
- **Launch Type**: Fargate

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
- ECS Cluster: `single-task-nginx`
- CloudWatch Logs Log Group: `/ecs/single-task-nginx` (7 days retention)

### 3. Verify Resources

```bash
# Verify ECS cluster
aws ecs describe-clusters \
  --cluster single-task-nginx \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# Verify CloudWatch Logs log group
aws logs describe-log-groups \
  --log-group-name-prefix /ecs/single-task-nginx \
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
cluster_name   = "single-task-nginx"
service_name   = "single-task-nginx"
environment    = "development"
```

Or override via command line:

```bash
terraform apply -var="cluster_name=my-cluster" -var="environment=staging"
```

## Manual Setup (Alternative)

<details>
<summary>Click to expand manual setup instructions using AWS CLI</summary>

### 1. Create the ECS Cluster

```bash
aws ecs create-cluster --cluster-name single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 2. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

Note: The `ecsTaskExecutionRole` is automatically created by KECS when it starts LocalStack. No need to create it manually.

</details>

## Deployment

### Using AWS CLI

```bash
# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Create service
aws ecs create-service \
  --cli-input-json file://service_def.json \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

## Verification

### 1. Check Service Status

```bash
aws ecs describe-services \
  --cluster single-task-nginx \
  --services single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'
```

### 2. List Running Tasks

```bash
aws ecs list-tasks \
  --cluster single-task-nginx \
  --service-name single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 3. Get Task Details

```bash
# Get task ARN from list-tasks output
TASK_ARN=$(aws ecs list-tasks \
  --cluster single-task-nginx \
  --service-name single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'taskArns[0]' --output text)

# Describe task to check status
aws ecs describe-tasks \
  --cluster single-task-nginx \
  --tasks $TASK_ARN \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'tasks[0].{Status:lastStatus,DesiredStatus:desiredStatus,TaskArn:taskArn}'
```

### 4. Check CloudWatch Logs

```bash
# View recent logs
aws logs tail /ecs/single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Follow logs in real-time
aws logs tail /ecs/single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --follow
```

## Key Points to Verify

1. **Task Status**: Should be RUNNING
2. **Service Status**: desiredCount should match runningCount
3. **Health Checks**: Container should pass health checks
4. **Logs**: Check CloudWatch logs for any errors

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
terraform import aws_ecs_cluster.main single-task-nginx
terraform import aws_cloudwatch_log_group.nginx /ecs/single-task-nginx
```

### Task and Service Issues

#### Check Task Logs

```bash
aws logs tail /ecs/single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --follow
```

#### Check Service Events

```bash
aws ecs describe-services \
  --cluster single-task-nginx \
  --services single-task-nginx \
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
  --cluster single-task-nginx \
  --service single-task-nginx \
  --force \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition single-task-nginx:1 \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete ECS cluster
aws ecs delete-cluster \
  --cluster single-task-nginx \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

</details>
