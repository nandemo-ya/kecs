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

# Stage 3: Final image (distroless)
FROM gcr.io/distroless/base-debian12:nonroot
COPY --from=builder /app/controlplane /controlplane

ENV PORT=8080
ENV ADMIN_PORT=8081

EXPOSE 8080 8081
ENTRYPOINT ["/controlplane", "server"]