# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY awsproxy/ ./awsproxy/

# Build the binary
WORKDIR /build/awsproxy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev} -X main.CommitHash=${COMMIT_HASH:-unknown}" \
    -o /aws-proxy \
    .

# Runtime stage
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN adduser -D -u 1000 proxy

# Copy binary from builder
COPY --from=builder /aws-proxy /usr/local/bin/aws-proxy
RUN chmod +x /usr/local/bin/aws-proxy

# Switch to non-root user
USER proxy

# Expose port
EXPOSE 4566

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:4566/health || exit 1

# Set default environment variables
ENV LOCALSTACK_ENDPOINT=http://localstack.aws-services.svc.cluster.local:4566
ENV DEBUG=false

# Run the proxy
ENTRYPOINT ["/usr/local/bin/aws-proxy"]