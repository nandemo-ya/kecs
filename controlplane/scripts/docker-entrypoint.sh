#!/bin/bash
set -e

# Docker entrypoint script for KECS
# Handles Docker socket permissions and starts the control plane

# Function to check Docker socket access
check_docker_access() {
    if [ -S /var/run/docker.sock ]; then
        # Check if we can read the Docker socket
        if [ -r /var/run/docker.sock ]; then
            echo "✓ Docker socket found and readable"
            return 0
        else
            echo "⚠️  Docker socket found but cannot access it"
            echo "   This might be a permissions issue"
            
            # Get Docker socket group
            DOCKER_GID=$(stat -c '%g' /var/run/docker.sock 2>/dev/null || stat -f '%g' /var/run/docker.sock 2>/dev/null || echo "")
            if [ -n "$DOCKER_GID" ]; then
                echo "   Docker socket group ID: $DOCKER_GID"
                echo "   Current user groups: $(id -G)"
                echo ""
                echo "   To fix this, you can:"
                echo "   1. Run the container with --group-add=$DOCKER_GID"
                echo "   2. Or mount the Docker socket with appropriate permissions"
            fi
            return 1
        fi
    else
        echo "⚠️  Docker socket not found at /var/run/docker.sock"
        echo "   KECS requires Docker socket access to manage k3d clusters"
        echo ""
        echo "   To fix this, mount the Docker socket:"
        echo "   docker run -v /var/run/docker.sock:/var/run/docker.sock ..."
        return 1
    fi
}

# Show banner
echo "======================================"
echo "        KECS Control Plane"
echo "======================================"
echo ""

# Check Docker access
if ! check_docker_access; then
    echo ""
    echo "❌ Cannot proceed without Docker access"
    exit 1
fi

echo ""
echo "Starting KECS Control Plane..."
echo ""

# Execute the control plane with all arguments
exec /controlplane "$@"