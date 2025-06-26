# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy source code
COPY . .

# Build the application
RUN npm run build

# Production stage with Traefik
FROM traefik:v3.0

# Install necessary tools
RUN apk add --no-cache nodejs npm

# Create app directory
RUN mkdir -p /app/build

# Copy built files from builder
COPY --from=builder /app/build /app/build

# Create a simple static file server
RUN npm install -g serve

# Copy Traefik configuration
COPY traefik.yml /etc/traefik/traefik.yml
COPY traefik-dynamic.yml /etc/traefik/dynamic/traefik-dynamic.yml

# Add runtime script for environment variable injection
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN chmod +x /docker-entrypoint.sh

# Expose ports
EXPOSE 80 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

# Start Traefik with static file server
ENTRYPOINT ["/docker-entrypoint.sh"]