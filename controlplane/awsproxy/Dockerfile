# syntax=docker/dockerfile:1
# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24.3 AS builder
WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT_HASH=unknown

# Copy go mod files for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source code
COPY awsproxy/ ./awsproxy/

# Build with cache mount
WORKDIR /app/awsproxy
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s -X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH}" \
    -o /aws-proxy .

# Final stage - use distroless for minimal image
FROM gcr.io/distroless/static-debian12:nonroot

# Copy binary from builder
COPY --from=builder /aws-proxy /aws-proxy

# Set default environment variables
ENV LOCALSTACK_ENDPOINT=http://localstack.aws-services.svc.cluster.local:4566
ENV DEBUG=false

# Expose port
EXPOSE 4566

# Use nonroot user (65532)
USER nonroot

# Run the proxy
ENTRYPOINT ["/aws-proxy"]