#!/bin/bash
# Test script for task artifacts functionality

set -e

echo "=== Task Artifacts Test Script ==="
echo

# Test S3 integration
echo "1. Testing S3 integration..."
go test ./internal/integrations/s3/... -v -count=1
echo "✓ S3 integration tests passed"
echo

# Test artifact manager
echo "2. Testing artifact manager..."
go test ./internal/artifacts/... -v -count=1
echo "✓ Artifact manager tests passed"
echo

# Test task converter with artifacts
echo "3. Testing task converter artifact support..."
cd internal/converters && ginkgo -r --focus="TaskConverter Artifact Support" && cd ../..
echo "✓ Task converter artifact tests passed"
echo

echo "=== All task artifact tests passed! ==="
echo
echo "Example task definition with artifacts:"
echo
cat << 'EOF'
{
  "family": "test-app",
  "containerDefinitions": [
    {
      "name": "app",
      "image": "busybox",
      "memory": 256,
      "artifacts": [
        {
          "name": "config",
          "artifactUrl": "s3://test-bucket/config.json",
          "targetPath": "/config/app.json",
          "permissions": "0644"
        }
      ],
      "command": ["cat", "/artifacts/config/app.json"]
    }
  ]
}
EOF