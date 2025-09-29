# syntax=docker/dockerfile:1
# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25.0 AS builder
WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH

# No longer need cross-compilers since we don't use CGO

# Copy go mod files for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build with cache mount (no CGO required)
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o kecs-server ./cmd/controlplane

# Stage 3: Final image (can use scratch or alpine since no CGO)
FROM alpine:3.19

# Install required runtime
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 kecs && \
    adduser -u 1000 -G kecs -h /home/kecs -D kecs && \
    mkdir -p /data && \
    chown -R kecs:kecs /data

# Copy the binary
COPY --from=builder /app/kecs-server /kecs-server
RUN chmod +x /kecs-server

USER kecs
WORKDIR /data

ENV PORT=8080
ENV ADMIN_PORT=8081
ENV KECS_DATA_DIR=/data

EXPOSE 8080 8081

ENTRYPOINT ["/kecs-server", "server"]