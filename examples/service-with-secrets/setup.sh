#!/bin/bash

# Setup script for creating secrets in LocalStack
# This script creates SSM parameters and Secrets Manager secrets for testing

ENDPOINT_URL=${ENDPOINT_URL:-http://localhost:8080}
REGION=${REGION:-us-east-1}

echo "Creating SSM Parameters..."

# Create SSM parameters
aws ssm put-parameter \
    --name "/myapp/prod/database-url" \
    --value "postgresql://app_user:password@db.example.com:5432/myapp" \
    --type "SecureString" \
    --description "Production database connection string" \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    --overwrite

aws ssm put-parameter \
    --name "/myapp/prod/api-key" \
    --value "sk_live_abcdef123456789" \
    --type "SecureString" \
    --description "Production API key" \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    --overwrite

aws ssm put-parameter \
    --name "/myapp/prod/feature-flags" \
    --value '{"new_ui": true, "beta_features": false, "maintenance_mode": false}' \
    --type "String" \
    --description "Feature flags configuration" \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    --overwrite

echo "Creating Secrets Manager Secrets..."

# Create Secrets Manager secrets
aws secretsmanager create-secret \
    --name "myapp/prod/db" \
    --description "Database credentials" \
    --secret-string '{"username":"admin","password":"super-secret-password"}' \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    2>/dev/null || \
aws secretsmanager put-secret-value \
    --secret-id "myapp/prod/db" \
    --secret-string '{"username":"admin","password":"super-secret-password"}' \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION

aws secretsmanager create-secret \
    --name "myapp/prod/jwt" \
    --description "JWT signing key" \
    --secret-string '{"key":"very-secret-jwt-key-12345"}' \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    2>/dev/null || \
aws secretsmanager put-secret-value \
    --secret-id "myapp/prod/jwt" \
    --secret-string '{"key":"very-secret-jwt-key-12345"}' \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION

aws secretsmanager create-secret \
    --name "myapp/prod/encryption" \
    --description "Encryption keys" \
    --secret-string '{"aes_key":"256bit-encryption-key-example"}' \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    2>/dev/null || \
aws secretsmanager put-secret-value \
    --secret-id "myapp/prod/encryption" \
    --secret-string '{"aes_key":"256bit-encryption-key-example"}' \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION

echo "Verifying created resources..."

echo ""
echo "SSM Parameters:"
aws ssm describe-parameters \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    --query "Parameters[*].[Name,Type,Description]" \
    --output table

echo ""
echo "Secrets Manager Secrets:"
aws secretsmanager list-secrets \
    --endpoint-url $ENDPOINT_URL \
    --region $REGION \
    --query "SecretList[*].[Name,Description]" \
    --output table

echo ""
echo "Setup complete!"