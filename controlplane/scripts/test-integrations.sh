#!/bin/bash
# Run integration tests for LocalStack integrations

set -e

echo "Running LocalStack integration tests..."

# Test IAM integration
echo "Testing IAM integration..."
go test ./internal/integrations/iam/... -v

# Test CloudWatch integration
echo "Testing CloudWatch integration..."
go test ./internal/integrations/cloudwatch/... -v

# Test S3 integration
echo "Testing S3 integration..."
go test ./internal/integrations/s3/... -v

# Test Artifacts manager
echo "Testing Artifacts manager..."
go test ./internal/artifacts/... -v

# Future integration tests can be added here

# echo "Testing ELBv2 integration..."
# go test ./internal/integrations/elbv2/... -v

echo "All integration tests passed!"