# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the proxy binary
RUN CGO_ENABLED=0 GOOS=linux go build -o aws-sdk-proxy ./cmd/aws-sdk-proxy

# Runtime stage
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/aws-sdk-proxy /app/

# Create non-root user
RUN adduser -D -u 1000 proxy
USER proxy

# Expose proxy port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/aws-sdk-proxy"]