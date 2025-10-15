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
            │      ECS Execution Role        │
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
2. Terraform installed (>= 1.0)
3. AWS CLI configured

## Quick Start with Terraform (Recommended)

### 1. Start KECS

```bash
# Start KECS instance
kecs start

# Wait for KECS to be ready
# Check that it's running (look for a running instance)
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
- ECS Cluster: `service-with-secrets`
- CloudWatch Logs Log Group: `/ecs/service-with-secrets` (7 days retention)
- SSM Parameters (3):
  - `/myapp/prod/database-url` - Database connection string
  - `/myapp/prod/api-key` - API key
  - `/myapp/prod/feature-flags` - Feature flags configuration
- Secrets Manager Secrets (3):
  - `myapp/prod/db` - Database credentials
  - `myapp/prod/jwt` - JWT signing key
  - `myapp/prod/encryption` - Encryption keys

### 3. Verify Resources

```bash
# Verify ECS cluster
aws ecs describe-clusters \
  --cluster service-with-secrets \
  --endpoint-url http://localhost:5373 \
  --region us-east-1

# Verify SSM parameters
aws ssm describe-parameters \
  --endpoint-url http://localhost:5373 \
  --region us-east-1 \
  --query "Parameters[*].[Name,Type]" \
  --output table

# Verify Secrets Manager secrets
aws secretsmanager list-secrets \
  --endpoint-url http://localhost:5373 \
  --region us-east-1 \
  --query "SecretList[*].[Name,Description]" \
  --output table

```

### 4. View Terraform Outputs

```bash
# Show all outputs
terraform output

# Show specific output
terraform output cluster_name
terraform output secret_arns_for_task_definition
```

After applying, Terraform provides several outputs:

- `cluster_name`: ECS cluster name
- `cluster_arn`: ECS cluster ARN
- `log_group_name`: CloudWatch Logs log group name
- `ssm_parameters`: Map of SSM parameter names
- `secrets_manager_secrets`: Map of Secrets Manager secret ARNs (sensitive)
- `secret_arns_for_task_definition`: Secret ARNs formatted for ECS task definitions
- `ssm_parameter_arns_for_task_definition`: SSM parameter ARNs formatted for ECS task definitions

### 5. Terraform Configuration

You can customize the configuration by creating a `terraform.tfvars` file:

```hcl
aws_region     = "us-east-1"
kecs_endpoint  = "http://localhost:5373"
cluster_name   = "service-with-secrets"
service_name   = "service-with-secrets"
environment    = "prod"
app_name       = "myapp"
```

Or override via command line:

```bash
terraform apply -var="cluster_name=my-cluster" -var="environment=dev"
```

### 6. Advanced Usage

#### Using with Different Environments

```bash
# Development environment
terraform apply -var="environment=dev" -var="app_name=myapp-dev"

# Staging environment
terraform apply -var="environment=staging" -var="app_name=myapp-staging"
```

#### Importing Existing Resources

If you already have resources created by `setup.sh`, you can import them:

```bash
# Import ECS cluster
terraform import aws_ecs_cluster.main service-with-secrets

# Import SSM parameter
terraform import aws_ssm_parameter.database_url /myapp/prod/database-url

# Import Secrets Manager secret
terraform import aws_secretsmanager_secret.db myapp/prod/db
```

#### Remote State (Optional)

For team collaboration, configure remote state in `provider.tf`:

```hcl
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "kecs/service-with-secrets/terraform.tfstate"
    region = "us-east-1"
  }
}
```

## Manual Setup (Alternative)

<details>
<summary>Click to expand manual setup instructions using AWS CLI</summary>

### 1. Create the ECS Cluster

```bash
aws ecs create-cluster --cluster-name service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

### 2. Create SSM Parameters

```bash
# Database URL
aws ssm put-parameter \
  --name "/myapp/prod/database-url" \
  --value "postgresql://app_user:password@db.example.com:5432/myapp" \
  --type "SecureString" \
  --description "Production database connection string" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# API Key
aws ssm put-parameter \
  --name "/myapp/prod/api-key" \
  --value "sk_live_abcdef123456789" \
  --type "SecureString" \
  --description "Production API key" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Feature Flags
aws ssm put-parameter \
  --name "/myapp/prod/feature-flags" \
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

### 5. Note about IAM Roles

**Note**: The `ecsTaskExecutionRole` is automatically created by KECS when it starts LocalStack, and it already has the necessary permissions to access secrets from Secrets Manager and SSM Parameter Store. No additional IAM configuration is required for this example.

### 6. Create CloudWatch Log Group

```bash
aws logs create-log-group \
  --log-group-name /ecs/service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

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
# Verify service is running
aws ecs describe-services \
  --cluster service-with-secrets \
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
  --name "/myapp/prod/api-key" \
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
  --cluster service-with-secrets \
  --service service-with-secrets \
  --force-new-deployment \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Wait for deployment to complete
aws ecs wait services-stable \
  --cluster service-with-secrets \
  --services service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

## Key Points to Verify

1. **Secret Injection**: All secrets from SSM and Secrets Manager are available as environment variables
2. **No Hardcoded Secrets**: Task definition contains only references, not actual secret values
3. **Proper Permissions**: Execution role has necessary permissions to retrieve secrets
4. **Secret Isolation**: Each environment (dev, staging, prod) uses different secret paths
5. **Audit Trail**: Secret access is logged in CloudTrail (in production AWS)

## Security Best Practices Demonstrated

1. **Least Privilege**: Execution role has controlled access to secrets
2. **Encryption at Rest**: Secrets are encrypted in both SSM and Secrets Manager
3. **Encryption in Transit**: Secrets are retrieved over TLS
4. **No Secret Logging**: Application doesn't log actual secret values
5. **Secret Rotation**: Supports updating secrets without code changes

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

If resources were created by `setup.sh`:

```bash
# Option 1: Destroy existing resources first using terraform
terraform destroy

# Option 2: Import existing resources
terraform import aws_ecs_cluster.main service-with-secrets
terraform import aws_ssm_parameter.database_url /myapp/prod/database-url
terraform import aws_secretsmanager_secret.db myapp/prod/db
```

#### Invalid Credentials

Terraform uses fake credentials for KECS. Make sure `provider.tf` includes:

```hcl
skip_credentials_validation = true
skip_requesting_account_id  = true
skip_metadata_api_check     = true
```

### Task and Service Issues

#### Check Task Logs for Secret Loading Issues

```bash
aws logs tail /ecs/service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373 \
  --follow
```

#### Verify Execution Role

```bash
# Verify execution role exists (auto-created by KECS)
aws iam get-role \
  --role-name ecsTaskExecutionRole \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

#### Debug Secret Access

```bash
# Check if secrets exist
aws secretsmanager describe-secret \
  --secret-id "myapp/prod/db" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Check if parameters exist
aws ssm get-parameter \
  --name "/myapp/prod/database-url" \
  --with-decryption \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
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
  --cluster service-with-secrets \
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
  --name "/myapp/prod/database-url" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws ssm delete-parameter \
  --name "/myapp/prod/api-key" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

aws ssm delete-parameter \
  --name "/myapp/prod/feature-flags" \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete log group
aws logs delete-log-group \
  --log-group-name /ecs/service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373

# Delete ECS cluster
aws ecs delete-cluster \
  --cluster service-with-secrets \
  --region us-east-1 \
  --endpoint-url http://localhost:5373
```

</details>