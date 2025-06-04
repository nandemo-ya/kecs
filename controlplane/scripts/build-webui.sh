#!/bin/bash

# Build Web UI for embedding in Control Plane

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/../.." && pwd )"
WEBUI_DIR="$PROJECT_ROOT/web-ui"
CONTROLPLANE_DIR="$PROJECT_ROOT/controlplane"
WEBUI_DIST_DIR="$CONTROLPLANE_DIR/internal/controlplane/api/webui_dist"

echo "Building Web UI..."

# Check if web-ui directory exists
if [ ! -d "$WEBUI_DIR" ]; then
    echo "Error: web-ui directory not found at $WEBUI_DIR"
    exit 1
fi

# Change to web-ui directory
cd "$WEBUI_DIR"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Build the React app
echo "Building React app..."
npm run build

# Create webui_dist directory
echo "Preparing Web UI for embedding..."
rm -rf "$WEBUI_DIST_DIR"
mkdir -p "$WEBUI_DIST_DIR"

# Copy build files
cp -r build/* "$WEBUI_DIST_DIR/"

echo "Web UI build complete!"
echo "Files copied to: $WEBUI_DIST_DIR"

# Build Control Plane with embedded Web UI
cd "$CONTROLPLANE_DIR"
echo "Building Control Plane with embedded Web UI..."
go build -tags embed_webui -o bin/kecs cmd/controlplane/main.go

echo "Build complete!"