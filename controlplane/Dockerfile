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

# Stage 3: Final image (distroless for minimal attack surface)
FROM gcr.io/distroless/static:nonroot

# Create data directory (distroless images have nonroot user with UID 65532)
USER 65532
WORKDIR /data

# Copy the binary (already executable from builder stage)
COPY --from=builder --chown=65532:65532 /app/kecs-server /kecs-server

ENV PORT=8080
ENV ADMIN_PORT=8081
ENV KECS_DATA_DIR=/data

EXPOSE 8080 8081

ENTRYPOINT ["/kecs-server", "server"]