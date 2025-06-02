# Stage 1: Build dependencies cache
FROM --platform=$BUILDPLATFORM golang:1.24.3 AS deps
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Stage 2: Build the application
FROM --platform=$BUILDPLATFORM golang:1.24.3 AS builder
WORKDIR /app

ARG TARGETOS=linux
ARG TARGETARCH

# Install only the required cross-compiler based on target architecture
RUN apt-get update && \
    if [ "$TARGETARCH" = "arm64" ]; then \
        apt-get install -y gcc-aarch64-linux-gnu g++-aarch64-linux-gnu; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
        apt-get install -y gcc-x86-64-linux-gnu g++-x86-64-linux-gnu; \
    fi && \
    rm -rf /var/lib/apt/lists/*

# Copy dependencies from cache
COPY --from=deps /go/pkg /go/pkg

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build with proper cross-compilation
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
        export CC=x86_64-linux-gnu-gcc CXX=x86_64-linux-gnu-g++; \
    fi && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o controlplane ./cmd/controlplane

# Stage 3: Prepare runtime base (can run in parallel)
FROM debian:bookworm-slim AS runtime-base
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -u 65532 -m nonroot

# Stage 4: Final image
FROM runtime-base
COPY --from=builder /app/controlplane /controlplane

USER nonroot

ENV PORT=8080
ENV ADMIN_PORT=8081

EXPOSE 8080 8081
ENTRYPOINT ["/controlplane", "server"]