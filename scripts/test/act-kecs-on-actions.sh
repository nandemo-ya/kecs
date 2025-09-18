#!/bin/bash

# E2E Test KECS workflow with act

echo "Running E2E Test KECS workflow with act..."
echo "This will test KECS deployment in a k3d cluster using Docker containers"
echo ""

# Default value (use 'latest' to avoid local build)
KECS_IMAGE_TAG="${1:-latest}"

echo "Parameters:"
echo "  KECS Image Tag: $KECS_IMAGE_TAG"
echo ""

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
  x86_64)
    CONTAINER_ARCH="linux/amd64"
    ;;
  arm64|aarch64)
    # Use amd64 for better emulation compatibility on M1/M2/M3 Macs
    echo "Note: Running on ARM64 architecture (Apple Silicon)"
    echo "Using linux/amd64 for better compatibility with act"
    CONTAINER_ARCH="linux/amd64"
    ;;
  *)
    echo "Warning: Unknown architecture $ARCH, defaulting to linux/amd64"
    CONTAINER_ARCH="linux/amd64"
    ;;
esac

echo "  Container Architecture: $CONTAINER_ARCH"
echo ""

# Execute act command
# --container-architecture: Specify container architecture
# -j e2e-test: Run specific job
# --input: Set workflow_dispatch input parameters
# -P ubuntu-latest=catthehacker/ubuntu:act-latest: Specify Ubuntu environment
# --rm: Remove containers after execution

act workflow_dispatch \
  -W .github/workflows/kecs-on-actions.yml \
  --container-architecture $CONTAINER_ARCH \
  -j e2e-test \
  --input kecs_image_tag="$KECS_IMAGE_TAG" \
  --input debug=false \
  -P ubuntu-latest=catthehacker/ubuntu:act-latest \
  --rm

# Notes:
# 1. Docker Desktop must be running
# 2. Initial run will take time to download Docker images
# 3. ghcr.io/nandemo-ya/kecs image must exist
#    (or use locally built image)