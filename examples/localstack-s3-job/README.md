# LocalStack S3 Job Example

This example demonstrates a one-off batch job that interacts with S3 using AWS CLI within a KECS task. The task is executed via `RunTask` (not a service) and performs S3 operations as a batch job.

## Overview

- **Purpose**: Demonstrate S3 integration with batch job pattern
- **Components**: Single container with AWS CLI
- **Pattern**: One-off task execution (RunTask, not CreateService)
- **Integration**: KECS transparently proxies S3 requests to internal LocalStack
- **Launch Type**: Fargate

## What This Example Does

The task performs the following S3 operations:
1. Creates an S3 bucket (`test-bucket`)
2. Generates a test file with content
3. Uploads the file to S3
4. Lists bucket contents
5. Exits successfully

All S3 operations work just like real AWS ECS - **no endpoint URL configuration needed**. KECS automatically injects the necessary environment variables and proxies requests to its internal LocalStack instance.

## Prerequisites

Before running this example, ensure you have:

1. KECS running locally
2. AWS CLI configured to point to KECS endpoint

## Setup Instructions

### 1. Start KECS (if not already running)

```bash
kecs start
```

KECS automatically starts LocalStack internally, so you don't need to manage LocalStack separately.

### 2. Set Environment Variables (Optional)

```bash
export AWS_ENDPOINT_URL=http://localhost:5373
export AWS_REGION=us-east-1
export CLUSTER_NAME=default
```

## Deployment

### Using the Run Script

The simplest way to run this example:

```bash
cd examples/localstack-s3-job
./run-task.sh
```

The script will:
1. Create the ECS cluster
2. Create CloudWatch log group
3. Register the task definition
4. Run the task
5. Wait for completion
6. Show logs
7. Verify S3 bucket contents

### Manual Step-by-Step

If you prefer to run commands manually:

#### 1. Create Cluster

```bash
aws ecs create-cluster \
  --cluster-name default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

#### 2. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/localstack-s3-job \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

#### 3. Register Task Definition

```bash
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

#### 4. Run Task

```bash
aws ecs run-task \
  --cluster default \
  --task-definition localstack-s3-job \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-12345],assignPublicIp=ENABLED}" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

## Verification

### 1. Check Task Status

```bash
# List tasks
aws ecs list-tasks \
  --cluster default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Get task details
TASK_ARN=$(aws ecs list-tasks \
  --cluster default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query 'taskArns[0]' --output text)

aws ecs describe-tasks \
  --cluster default \
  --tasks $TASK_ARN \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 2. Check CloudWatch Logs

```bash
# View task logs
aws logs tail /ecs/localstack-s3-job \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --since 10m
```

### 3. Verify S3 Operations

All S3 operations go through KECS endpoint - no need to use LocalStack endpoint directly:

```bash
# List buckets
aws s3 ls \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# List objects in test-bucket
aws s3 ls s3://test-bucket/ \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# Download the output file
aws s3 cp s3://test-bucket/output.txt - \
  --endpoint-url http://localhost:5373 \
  --region us-east-1
```

Expected output: `Hello from KECS task!`

## Key Features

### Transparent LocalStack Integration

KECS provides a seamless experience identical to real AWS ECS:

- **No endpoint configuration in task definition**: Just like real ECS, your tasks don't need `AWS_ENDPOINT_URL` environment variables
- **Automatic credential injection**: KECS automatically injects `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and `AWS_ENDPOINT_URL` into your tasks
- **Single endpoint for all services**: Both ECS API and AWS service APIs (S3, DynamoDB, etc.) go through KECS endpoint (port 5373)
- **Transparent proxying**: KECS intelligently routes requests to either its ECS implementation or internal LocalStack

This design mirrors the real AWS ECS experience where tasks naturally have access to AWS services through IAM roles without explicit endpoint configuration.

### Task Definition Simplicity

Notice how the task definition is clean and AWS-compatible:

```json
{
  "command": [
    "sh", "-c",
    "aws s3 mb s3://test-bucket || true; ..."
  ]
  // No AWS_ENDPOINT_URL needed!
}
```

This is identical to how you would write task definitions for production AWS ECS.

## Architecture Details

### Request Routing

```
Your AWS CLI → KECS (port 5373) → [ECS API Handler OR LocalStack Proxy]
Task Container → Auto-injected AWS env vars → KECS → LocalStack
```

KECS automatically determines whether a request is:
- ECS API call (CreateService, RunTask, etc.) → handled by KECS
- AWS service API call (S3, DynamoDB, etc.) → proxied to LocalStack

### IAM Roles

- **executionRoleArn**: Used by ECS to pull images and write logs
- **taskRoleArn**: Used by the task itself to access AWS services (S3)

Both roles are automatically created by KECS when it starts.

### Task vs Service

This example uses **RunTask** instead of **CreateService**:
- **RunTask**: One-off task execution (batch jobs, scheduled tasks)
- **CreateService**: Long-running services with desired count and auto-restart

## Troubleshooting

### Task Fails to Start

Check task definition and cluster status:

```bash
aws ecs describe-tasks \
  --cluster default \
  --tasks <TASK_ARN> \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### S3 Operations Fail

Check task logs to see actual error messages:

```bash
aws logs tail /ecs/localstack-s3-job \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --follow
```

### Verify KECS is Running

```bash
kecs status
```

## Cleanup

```bash
# Deregister task definition
aws ecs deregister-task-definition \
  --task-definition localstack-s3-job:1 \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/localstack-s3-job \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Clean up S3 bucket (optional)
aws s3 rb s3://test-bucket --force \
  --endpoint-url http://localhost:5373 \
  --region us-east-1
```

## Use Cases

This pattern is suitable for:

- **Batch processing**: Data processing jobs, ETL tasks
- **Scheduled tasks**: Nightly reports, data cleanup
- **CI/CD jobs**: Build tasks, deployment scripts
- **Data migration**: One-time data transfer operations
- **Testing**: Integration tests with AWS services

## Next Steps

To build upon this example:

1. **Add more AWS services**: DynamoDB, SQS, SNS, etc.
2. **Error handling**: Implement retry logic with exponential backoff
3. **Notifications**: Send SNS notifications on completion/failure
4. **Parameterization**: Use environment variables for bucket names, regions
5. **Scheduled execution**: Use CloudWatch Events to trigger tasks periodically
6. **Multi-step workflows**: Chain multiple tasks with Step Functions

## Related Examples

- `single-task-nginx`: Basic one-off task example
- `service-with-secrets`: Service with secrets management
- `multi-container-alb`: Multi-container service with load balancer
