#!/bin/bash

# Production build script for KECS

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
VERSION=${VERSION:-$(git describe --tags --always --dirty)}
DOCKER_REGISTRY=${DOCKER_REGISTRY:-"ghcr.io/nandemo-ya/kecs"}
PLATFORMS=${PLATFORMS:-"linux/amd64,linux/arm64"}

echo -e "${GREEN}Building KECS Production Image${NC}"
echo "Version: $VERSION"
echo "Registry: $DOCKER_REGISTRY"
echo "Platforms: $PLATFORMS"
echo ""

# Check prerequisites
command -v docker >/dev/null 2>&1 || { echo -e "${RED}Docker is required but not installed.${NC}" >&2; exit 1; }

# Check if buildx is available
if ! docker buildx version >/dev/null 2>&1; then
    echo -e "${YELLOW}Docker buildx not found. Setting up...${NC}"
    docker buildx create --name kecs-builder --use
fi

# Build multi-platform Docker image
cd "$PROJECT_ROOT"

echo -e "${GREEN}Building Docker image...${NC}"

# Build flags
BUILD_FLAGS="--platform=$PLATFORMS"
BUILD_FLAGS="$BUILD_FLAGS --build-arg VERSION=$VERSION"
BUILD_FLAGS="$BUILD_FLAGS -f controlplane/Dockerfile"
BUILD_FLAGS="$BUILD_FLAGS -t $DOCKER_REGISTRY:$VERSION"
BUILD_FLAGS="$BUILD_FLAGS -t $DOCKER_REGISTRY:latest"

# Add push flag if requested
if [ "$PUSH" = "true" ]; then
    BUILD_FLAGS="$BUILD_FLAGS --push"
else
    BUILD_FLAGS="$BUILD_FLAGS --load"
    echo -e "${YELLOW}Note: Building for local use only. Use PUSH=true to push to registry.${NC}"
fi

# Build the image
docker buildx build $BUILD_FLAGS .

echo -e "${GREEN}Build complete!${NC}"

# Print image info
echo ""
echo "Image tags:"
echo "  - $DOCKER_REGISTRY:$VERSION"
echo "  - $DOCKER_REGISTRY:latest"

# Print run instructions
echo ""
echo "To run locally:"
echo "  docker run -p 8080:8080 -p 8081:8081 $DOCKER_REGISTRY:$VERSION"
echo ""
echo "To run with custom data directory:"
echo "  docker run -p 8080:8080 -p 8081:8081 -v /path/to/data:/data $DOCKER_REGISTRY:$VERSION"