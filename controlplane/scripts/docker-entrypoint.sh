#!/bin/bash
set -e

# Docker entrypoint script for KECS Control Plane
# The control plane runs inside a k3d cluster and doesn't need Docker socket access

# Show banner
echo "======================================"
echo "        KECS Control Plane"
echo "======================================"
echo ""

# Note about Docker socket for legacy compatibility
if [ -S /var/run/docker.sock ]; then
    echo "ℹ️  Docker socket detected (not required for control plane operation)"
    echo ""
fi

echo "Starting KECS Control Plane..."
echo ""

# Execute the control plane with all arguments
exec /controlplane "$@"