# Use debian-based image for CGO support
FROM --platform=$BUILDPLATFORM golang:1.24.3 AS builder

WORKDIR /app

# Install build dependencies for DuckDB
RUN apt-get update && apt-get install -y \
    gcc-aarch64-linux-gnu \
    gcc-x86-64-linux-gnu \
    g++-aarch64-linux-gnu \
    g++-x86-64-linux-gnu \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS=linux
ARG TARGETARCH

# Set cross-compilation variables
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        export CC=aarch64-linux-gnu-gcc CXX=aarch64-linux-gnu-g++; \
    elif [ "$TARGETARCH" = "amd64" ]; then \
        export CC=x86_64-linux-gnu-gcc CXX=x86_64-linux-gnu-g++; \
    fi && \
    CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -o controlplane ./cmd/controlplane

# Use debian-slim for runtime (needed for CGO dependencies)
FROM --platform=$TARGETPLATFORM debian:bookworm-slim
# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -u 65532 -m nonroot

COPY --from=builder /app/controlplane /controlplane

USER nonroot

ENV PORT=8080
ENV ADMIN_PORT=8081

EXPOSE 8080 8081
ENTRYPOINT ["/controlplane", "server"]
