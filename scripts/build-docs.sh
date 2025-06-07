#!/bin/bash
set -e

echo "Building KECS documentation site..."

# Change to docs-site directory
cd docs-site

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Build the documentation
echo "Building documentation..."
npm run docs:build

echo "Documentation built successfully!"
echo "Output is in docs-site/.vitepress/dist/"
echo ""
echo "To preview the built site, run:"
echo "  cd docs-site && npm run docs:preview"