#!/bin/bash

# Create S3 bucket
awslocal s3 mb s3://test-bucket

# Create SSM parameter
awslocal ssm put-parameter \
  --name /three-tier/db-password \
  --value mysecretpassword \
  --type SecureString

# Create CloudWatch log groups
awslocal logs create-log-group --log-group-name /ecs/three-tier-backend
awslocal logs create-log-group --log-group-name /ecs/three-tier-frontend
awslocal logs create-log-group --log-group-name /ecs/three-tier-database

echo "LocalStack initialization complete"