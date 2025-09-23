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

1. KECS running locally
2. AWS CLI configured

## Setup Instructions

### 1. Start KECS and LocalStack

```bash
# Start KECS (LocalStack is automatically started with KECS)
kecs start

# Wait for KECS and LocalStack to be ready
kecs status

# LocalStack is automatically available at port 4566
# No need to manually start LocalStack
```

### 2. Create the ECS Cluster

```bash
aws ecs create-cluster --cluster-name default \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 3. Create SSM Parameters

```bash
# Database URL
aws ssm put-parameter \
  --name "/myapp/prod/database_url" \
  --value "postgresql://app_user:password@db.example.com:5432/myapp" \
  --type "SecureString" \
  --description "Production database connection string" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# API Key
aws ssm put-parameter \
  --name "/myapp/prod/api_key" \
  --value "sk_live_abcdef123456789" \
  --type "SecureString" \
  --description "Production API key" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Feature Flags
aws ssm put-parameter \
  --name "/myapp/prod/feature_flags" \
  --value '{"new_ui": true, "beta_features": false, "maintenance_mode": false}' \
  --type "String" \
  --description "Feature flags configuration" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Verify parameters
aws ssm describe-parameters \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
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
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# JWT signing secret
aws secretsmanager create-secret \
  --name "myapp/prod/jwt" \
  --description "JWT signing secret" \
  --secret-string '{"secret": "jwt-signing-secret-key-here"}' \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Encryption key
aws secretsmanager create-secret \
  --name "myapp/prod/encryption" \
  --description "Data encryption key" \
  --secret-string '{"key": "AES256-encryption-key-32-bytes!!"}' \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# List secrets
aws secretsmanager list-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --query "SecretList[*].[Name,Description]" \
  --output table
```

### 5. Create IAM Roles with Proper Permissions

Note: The `ecsTaskExecutionRole` is automatically created by KECS when it starts LocalStack.

```bash
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
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Attach policies to execution role
# Note: ecsTaskExecutionRole is auto-created by KECS, we just need to attach additional policies
aws iam attach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

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
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 6. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

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
# Verify service is running
aws ecs describe-services \
  --cluster default \
  --services service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
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
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Update a secret in Secrets Manager
aws secretsmanager update-secret \
  --secret-id "myapp/prod/jwt" \
  --secret-string '{"secret": "new-jwt-signing-secret-key"}' \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Force service update to pick up new secrets
aws ecs update-service \
  --cluster default \
  --service service-with-secrets \
  --force-new-deployment \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Wait for deployment to complete
aws ecs wait services-stable \
  --cluster default \
  --services service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
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
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --follow
```

### Verify IAM Permissions

```bash
# Test if execution role can access secrets
aws sts assume-role \
  --role-arn arn:aws:iam::000000000000:role/ecsTaskExecutionRole \
  --role-session-name test-session \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# List attached policies
aws iam list-attached-role-policies \
  --role-name ecsTaskExecutionRole \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### Debug Secret Access

```bash
# Check if secrets exist
aws secretsmanager describe-secret \
  --secret-id "myapp/prod/db" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Check if parameters exist
aws ssm get-parameter \
  --name "/myapp/prod/database_url" \
  --with-decryption \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

## Cleanup

```bash
# Delete service
aws ecs delete-service \
  --cluster default \
  --service service-with-secrets \
  --force \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete secrets
aws secretsmanager delete-secret \
  --secret-id "myapp/prod/db" \
  --force-delete-without-recovery \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws secretsmanager delete-secret \
  --secret-id "myapp/prod/jwt" \
  --force-delete-without-recovery \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws secretsmanager delete-secret \
  --secret-id "myapp/prod/encryption" \
  --force-delete-without-recovery \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete SSM parameters
aws ssm delete-parameter \
  --name "/myapp/prod/database_url" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws ssm delete-parameter \
  --name "/myapp/prod/api_key" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws ssm delete-parameter \
  --name "/myapp/prod/feature_flags" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete IAM resources
# Note: Only detach the policy from ecsTaskExecutionRole, don't delete the role itself as it's managed by KECS
aws iam detach-role-policy \
  --role-name ecsTaskExecutionRole \
  --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws iam delete-policy \
  --policy-arn arn:aws:iam::000000000000:policy/ECSSecretsPolicy \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws iam delete-role \
  --role-name ecsTaskRole \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Stop LocalStack
docker stop localstack
docker rm localstack
```