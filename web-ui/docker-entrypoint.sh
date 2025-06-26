#!/bin/sh
set -e

# Default API endpoint if not provided
API_ENDPOINT="${KECS_API_ENDPOINT:-http://localhost:8080}"

# Create runtime config file
cat > /app/build/config.js <<EOF
window.KECS_CONFIG = {
  API_ENDPOINT: "${API_ENDPOINT}",
  WS_ENDPOINT: "${API_ENDPOINT}".replace(/^http/, 'ws') + "/ws"
};
EOF

# Update Traefik dynamic config with actual API endpoint
sed -i "s|http://localhost:8080|${API_ENDPOINT}|g" /etc/traefik/dynamic/traefik-dynamic.yml

# Start static file server in background
serve -s /app/build -l 3001 &

# Start Traefik
exec traefik