# syntax=docker/dockerfile:1
# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24.3 AS builder
WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH

# Install cross-compilers with cache
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    case "$TARGETARCH" in \
        arm64) apt-get install -y gcc-aarch64-linux-gnu g++-aarch64-linux-gnu ;; \
        amd64) apt-get install -y gcc-x86-64-linux-gnu g++-x86-64-linux-gnu ;; \
    esac

# Copy go mod files for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# Copy source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build with cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    case "$TARGETARCH" in \
        arm64) export CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++ ;; \
        amd64) export CC=x86_64-linux-gnu-gcc CXX=x86_64-linux-gnu-g++ ;; \
    esac && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o controlplane ./cmd/controlplane

# Stage 3: Final image - use debian slim for docker-cli support
FROM debian:stable-slim

# Install docker-cli and required utilities
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get install -y docker-ce-cli \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user and docker group if needed
RUN groupadd -f -g 999 docker || true && \
    groupadd -g 1000 kecs && \
    useradd -u 1000 -g kecs -G docker -m -s /bin/bash kecs && \
    mkdir -p /data && \
    chown -R kecs:kecs /data

# Copy the binary
COPY --from=builder /app/controlplane /controlplane
RUN chmod +x /controlplane

# Copy entrypoint script
COPY scripts/docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER kecs
WORKDIR /data

ENV PORT=8080
ENV ADMIN_PORT=8081
ENV KECS_DATA_DIR=/data

EXPOSE 8080 8081

ENTRYPOINT ["/entrypoint.sh"]