#!/bin/bash
# Development script for testing Japanese documentation

echo "Starting VitePress development server with Japanese documentation..."
echo "Japanese documentation will be available at: http://localhost:5173/kecs/ja/"
echo ""

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "Installing dependencies..."
    npm install
fi

# Start the development server
npm run docs:dev