#!/bin/bash

# Script to download AWS API definitions for code generation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CODEGEN_DIR="$SCRIPT_DIR/../cmd/codegen"

echo "Downloading AWS API definitions..."

# Create directory if it doesn't exist
mkdir -p "$CODEGEN_DIR"

# Download API definitions from AWS SDK Go v2 repository
# These are Smithy JSON files from the SDK's codegen/sdk-codegen/aws-models directory

SERVICES=(
    "ecs"
    "cloudwatch-logs"
    "elastic-load-balancing-v2"
    "iam"
    "s3"
    "secrets-manager"
    "ssm"
    "sts"
)

BASE_URL="https://raw.githubusercontent.com/aws/aws-sdk-go-v2/main/codegen/sdk-codegen/aws-models"

for SERVICE in "${SERVICES[@]}"; do
    echo "Downloading $SERVICE.json..."
    OUTPUT_FILE="$CODEGEN_DIR/${SERVICE//-/}.json"
    
    # Try to download the service definition
    if curl -f -L -o "$OUTPUT_FILE" "$BASE_URL/$SERVICE.json" 2>/dev/null; then
        echo "✓ Downloaded $SERVICE.json"
    else
        echo "✗ Failed to download $SERVICE.json"
        rm -f "$OUTPUT_FILE"
    fi
done

# Check if we have the files
echo ""
echo "Downloaded API definitions:"
ls -la "$CODEGEN_DIR"/*.json 2>/dev/null || echo "No JSON files found"

echo ""
echo "Done!"