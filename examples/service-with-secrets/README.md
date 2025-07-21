# Service with Secrets Example

This example demonstrates how to securely inject secrets from AWS Secrets Manager and SSM Parameter Store into ECS tasks.

## Overview

- **Purpose**: Show secure secret management in ECS tasks
- **Components**: 
  - Python web service that uses secrets
  - AWS Secrets Manager integration
  - SSM Parameter Store integration
- **Features**:
  - Environment variables from SSM parameters
  - Secrets from Secrets Manager
  - Secure credential injection
  - No hardcoded secrets

## Architecture

```
┌─────────────────────────┐     ┌─────────────────────────┐
│   SSM Parameter Store   │     │    Secrets Manager      │
│                         │     │                         │
│ • /myapp/prod/database  │     │ • myapp/prod/db         │
│ • /myapp/prod/api_key   │     │ • myapp/prod/jwt        │
│ • /myapp/prod/features  │     │ • myapp/prod/encryption │
└───────────┬─────────────┘     └───────────┬─────────────┘
            │                                 │
            │         ECS Task Role          │
            └─────────────┬──────────────────┘
                          │
                    ┌─────▼─────┐
                    │ ECS Task  │
                    │           │
                    │ Environment Variables:
                    │ • DATABASE_URL (from SSM)
                    │ • API_KEY (from SSM)
                    │ • DB_PASSWORD (from Secrets Manager)
                    │ • JWT_SECRET (from Secrets Manager)
                    └───────────┘
```

## Prerequisites

1. KECS running locally (port 8080)
2. LocalStack (for Secrets Manager and SSM, port 4566)
3. AWS CLI configured
4. ecspresso installed

### Endpoint URLs

This example uses two different endpoints:
- **KECS (http://localhost:8080)**: For ECS APIs (clusters, services, tasks)
- **LocalStack (http://localhost:4566)**: For AWS services (Secrets Manager, SSM, IAM)

## Setup Instructions

### 1. Start KECS and LocalStack

```bash
# Start KECS
kecs start

# Start LocalStack with required services
docker run -d \
  --name localstack \
  -p 4566:4566 \
  -e SERVICES=secretsmanager,ssm,iam,logs,ecs \
  -e DEBUG=1 \
  localstack/localstack

# Wait for LocalStack to be ready
until curl -s http://localhost:4566/_localstack/health | grep -q '"ssm": "available"'; do
  echo "Waiting for LocalStack..."
  sleep 2
done
```

### 2. Create the ECS Cluster

```bash
aws ecs create-cluster --cluster-name default \
  --endpoint-url http://localhost:8080
```

### 3. Create SSM Parameters

```bash
# Database URL
aws ssm put-parameter \
  --name "/myapp/prod/database_url" \
  --value "postgresql://app_user:password@db.example.com:5432/myapp" \
  --type "SecureString" \
  --description "Production database connection string" \
  --endpoint-url http://localhost:4566  # LocalStack for SSM

# API Key
aws ssm put-parameter \
  --name "/myapp/prod/api_key" \
  --value "sk_live_abcdef123456789" \
  --type "SecureString" \
  --description "Production API key" \
  --endpoint-url http://localhost:4566

# Feature Flags
aws ssm put-parameter \
  --name "/myapp/prod/feature_flags" \
  --value '{"new_ui": true, "beta_features": false, "maintenance_mode": false}' \
  --type "String" \
  --description "Feature flags configuration" \
  --endpoint-url http://localhost:4566

# Verify parameters
aws ssm describe-parameters \
  --endpoint-url http://localhost:4566 \
  --query "Parameters[*].[Name,Type,Description]" \
  --output table
```

### 4. Create Secrets in Secrets Manager

```bash
# Database password
aws secretsmanager create-secret \
  --name "myapp/prod/db" \
  --description "Production database password" \
  --secret-string '{"password": "super-secret-db-password"}' \
  --endpoint-url http://localhost:4566

# JWT signing secret
aws secretsmanager create-secret \
  --name "myapp/prod/jwt" \
  --description "JWT signing secret" \
  --secret-string '{"secret": "jwt-signing-secret-key-here"}' \
  --endpoint-url http://localhost:4566

# Encryption key
aws secretsmanager create-secret \
  --name "myapp/prod/encryption" \
  --description "Data encryption key" \
  --secret-string '{"key": "AES256-encryption-key-32-bytes!!"}' \
  --endpoint-url http://localhost:4566

# List secrets
aws secretsmanager list-secrets \
  --endpoint-url http://localhost:4566 \
  --query "SecretList[*].[Name,Description]" \
  --output table
```

### 5. Create IAM Roles with Proper Permissions

```bash
# Create Task Execution Role
aws iam create-role \
  --role-name ecsTaskExecutionRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ecs-tasks.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }' \
  --endpoint-url http://localhost:4566

# Create policy for accessing secrets
aws iam create-policy \
  --policy-name ECSSecretsPolicy \
  --policy-document '{
    "Version": "2012-10-17",
    "Statement": [
      {
        "Effect": "Allow",
        "Action": [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParameterHistory"
        ],
        "Resource": [
          "arn:aws:ssm:us-east-1:000000000000:parameter/myapp/prod/*"
        ]
      },
      {
        "Effect": "Allow",
        "Action": [
          "secretsmanager:GetSecretValue"
        ],
        "Resource": [
          "arn:aws:secretsmanager:us-east-1:000000000000:secret:myapp/prod/*"
        ]
      },
      {
        "Effect": "Allow",
        "Action": [
          "kms:Decrypt"
        ],
        "Resource": "*"
      }
    ]
  }' \
  --endpoint-url http://localhost:4566

# Attach policies to execution role
aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy \
  --endpoint-url http://localhost:4566

# Create Task Role
aws iam create-role \
  --role-name ecsTaskRole \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ecs-tasks.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }' \
  --endpoint-url http://localhost:4566
```

### 6. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/service-with-secrets \
  --endpoint-url http://localhost:8080
```

## Deployment

### Using ecspresso

```bash
# Deploy the service
ecspresso deploy --config ecspresso.yml

# Check deployment status
ecspresso status --config ecspresso.yml

# View logs
ecspresso logs --config ecspresso.yml
```

### Using AWS CLI

```bash
# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://task_def.json \
  --endpoint-url http://localhost:8080

# Create service
aws ecs create-service \
  --cli-input-json file://service_def.json \
  --endpoint-url http://localhost:8080
```

## Verification

### 1. Check Service Status

```bash
# Verify service is running
aws ecs describe-services \
  --cluster default \
  --services service-with-secrets \
  --endpoint-url http://localhost:8080 \
  --query 'services[0].{Status:status,Desired:desiredCount,Running:runningCount}'
```

### 2. Test Secret Injection

```bash
# Get a running task's pod
POD_NAME=$(kubectl get pods -n default -l app=service-with-secrets -o jsonpath='{.items[0].metadata.name}')

# Port forward to access the service
kubectl port-forward -n default $POD_NAME 8080:8080 &
PF_PID=$!

# Test health endpoint
curl http://localhost:8080/health
# Expected: {"status": "healthy"}

# Check configuration (non-secret values)
curl http://localhost:8080/config | jq
# Expected output showing configuration values:
# {
#   "database_url": "postgresql://app_user:password@db.example.com:5432/myapp",
#   "api_key_present": true,
#   "app_config": "server=app.example.com;timeout=30;retries=3",
#   "feature_flags": "{\"new_ui\": true, \"beta_features\": false, \"maintenance_mode\": false}",
#   "environment": "production"
# }

# Verify secrets are loaded (but not exposed)
curl http://localhost:8080/secrets | jq
# Expected output confirming secrets are loaded:
# {
#   "db_password_loaded": true,
#   "jwt_secret_loaded": true,
#   "encryption_key_loaded": true
# }

# Clean up port forward
kill $PF_PID
```

### 3. Verify Environment Variables in Container

```bash
# Check that secrets are injected as environment variables
kubectl exec -n default $POD_NAME -- env | grep -E "(DATABASE_URL|API_KEY|DB_PASSWORD)" | wc -l
# Should return 3 (number of secrets injected)

# Verify specific environment variable exists (without showing value)
kubectl exec -n default $POD_NAME -- sh -c 'if [ -n "$DB_PASSWORD" ]; then echo "DB_PASSWORD is set"; else echo "DB_PASSWORD is NOT set"; fi'
# Expected: "DB_PASSWORD is set"
```

### 4. Test Secret Rotation

```bash
# Update a parameter in SSM
aws ssm put-parameter \
  --name "/myapp/prod/api_key" \
  --value "sk_live_new_key_987654321" \
  --type "SecureString" \
  --overwrite \
  --endpoint-url http://localhost:4566

# Update a secret in Secrets Manager
aws secretsmanager update-secret \
  --secret-id "myapp/prod/jwt" \
  --secret-string '{"secret": "new-jwt-signing-secret-key"}' \
  --endpoint-url http://localhost:4566

# Force service update to pick up new secrets
aws ecs update-service \
  --cluster default \
  --service service-with-secrets \
  --force-new-deployment \
  --endpoint-url http://localhost:8080

# Wait for deployment to complete
aws ecs wait services-stable \
  --cluster default \
  --services service-with-secrets \
  --endpoint-url http://localhost:8080
```

## Key Points to Verify

1. **Secret Injection**: All secrets from SSM and Secrets Manager are available as environment variables
2. **No Hardcoded Secrets**: Task definition contains only references, not actual secret values
3. **Proper Permissions**: IAM roles have necessary permissions to retrieve secrets
4. **Secret Isolation**: Each environment (dev, staging, prod) uses different secret paths
5. **Audit Trail**: Secret access is logged in CloudTrail (in production AWS)

## Security Best Practices Demonstrated

1. **Least Privilege**: IAM roles only have access to specific secret paths
2. **Encryption at Rest**: Secrets are encrypted in both SSM and Secrets Manager
3. **Encryption in Transit**: Secrets are retrieved over TLS
4. **No Secret Logging**: Application doesn't log actual secret values
5. **Secret Rotation**: Supports updating secrets without code changes

## Troubleshooting

### Check Task Logs for Secret Loading Issues

```bash
aws logs tail /ecs/service-with-secrets \
  --endpoint-url http://localhost:8080 \
  --follow
```

### Verify IAM Permissions

```bash
# Test if execution role can access secrets
aws sts assume-role \
  --role-arn arn:aws:iam::000000000000:role/ecsTaskExecutionRole \
  --role-session-name test-session \
  --endpoint-url http://localhost:4566

# List attached policies
aws iam list-attached-role-policies \
  --role-name ecsTaskExecutionRole \
  --endpoint-url http://localhost:4566
```

### Debug Secret Access

```bash
# Check if secrets exist
aws secretsmanager describe-secret \
  --secret-id "myapp/prod/db" \
  --endpoint-url http://localhost:4566

# Check if parameters exist
aws ssm get-parameter \
  --name "/myapp/prod/database_url" \
  --with-decryption \
  --endpoint-url http://localhost:4566
```

## Cleanup

```bash
# Delete service
aws ecs delete-service \
  --cluster default \
  --service service-with-secrets \
  --force \
  --endpoint-url http://localhost:8080

# Delete secrets
aws secretsmanager delete-secret \
  --secret-id "myapp/prod/db" \
  --force-delete-without-recovery \
  --endpoint-url http://localhost:4566

aws secretsmanager delete-secret \
  --secret-id "myapp/prod/jwt" \
  --force-delete-without-recovery \
  --endpoint-url http://localhost:4566

aws secretsmanager delete-secret \
  --secret-id "myapp/prod/encryption" \
  --force-delete-without-recovery \
  --endpoint-url http://localhost:4566

# Delete SSM parameters
aws ssm delete-parameter \
  --name "/myapp/prod/database_url" \
  --endpoint-url http://localhost:4566

aws ssm delete-parameter \
  --name "/myapp/prod/api_key" \
  --endpoint-url http://localhost:4566

aws ssm delete-parameter \
  --name "/myapp/prod/feature_flags" \
  --endpoint-url http://localhost:4566

# Delete IAM resources
aws iam detach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy \
  --endpoint-url http://localhost:4566

aws iam delete-policy \
  --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy \
  --endpoint-url http://localhost:4566

aws iam delete-role \
  --role-name ecsTaskExecutionRole \
  --endpoint-url http://localhost:4566

aws iam delete-role \
  --role-name ecsTaskRole \
  --endpoint-url http://localhost:4566

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/service-with-secrets \
  --endpoint-url http://localhost:8080

# Stop LocalStack
docker stop localstack
docker rm localstack
```