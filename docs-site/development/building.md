# Building Guide

This guide covers building KECS from source, creating releases, and packaging for distribution.

## Prerequisites

### Required Tools

- **Go 1.21+**: Programming language
- **Node.js 18+**: For Web UI development
- **Docker**: For container builds
- **Make**: Build automation
- **Git**: Version control

### Optional Tools

- **goreleaser**: For release automation
- **upx**: For binary compression
- **act**: For testing GitHub Actions locally

## Building from Source

### Quick Build

```bash
# Clone repository
git clone https://github.com/nandemo-ya/kecs.git
cd kecs

# Build everything
make all

# Binary will be at ./bin/kecs
./bin/kecs version
```

### Detailed Build Steps

#### 1. Control Plane Only

```bash
# Build control plane binary
make build

# Or manually
cd controlplane
go build -o ../bin/kecs ./cmd/controlplane
```

#### 2. Web UI Only

```bash
# Build Web UI
cd web-ui
npm install
npm run build

# Output in web-ui/dist/
```

#### 3. Complete Build with Embedded UI

```bash
# Build everything
./scripts/build-webui.sh

# Or step by step
cd web-ui
npm install
npm run build

cd ../controlplane
go generate ./...
go build -tags webui -o ../bin/kecs ./cmd/controlplane
```

## Build Options

### Build Tags

```bash
# Build with Web UI embedded
go build -tags webui

# Build with experimental features
go build -tags experimental

# Build with all features
go build -tags "webui experimental"
```

### Build Variables

```bash
# Set version information
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE"
```

### Cross-Compilation

```bash
# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o bin/kecs-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o bin/kecs-darwin-amd64
GOOS=darwin GOARCH=arm64 go build -o bin/kecs-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o bin/kecs-windows-amd64.exe

# Or use make
make build-all-platforms
```

## Docker Build

### Standard Docker Build

```bash
# Build Docker image
docker build -t kecs:latest .

# Multi-stage build
docker build --target runtime -t kecs:latest .

# With build arguments
docker build \
  --build-arg VERSION=v1.0.0 \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  -t kecs:v1.0.0 .
```

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make nodejs npm

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build Web UI
WORKDIR /build/web-ui
RUN npm ci && npm run build

# Build control plane
WORKDIR /build
RUN make build-webui

# Runtime stage
FROM alpine:latest AS runtime

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/bin/kecs /usr/local/bin/kecs

EXPOSE 8080 8081

ENTRYPOINT ["kecs"]
CMD ["server"]
```

### Multi-Architecture Build

```bash
# Setup buildx
docker buildx create --name kecs-builder --use

# Build for multiple architectures
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag ghcr.io/nandemo-ya/kecs:latest \
  --push .
```

## Release Process

### Manual Release

```bash
# Tag version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Build release artifacts
./scripts/build-release.sh v1.0.0

# Output will be in dist/
ls -la dist/
# kecs-v1.0.0-darwin-amd64.tar.gz
# kecs-v1.0.0-darwin-arm64.tar.gz
# kecs-v1.0.0-linux-amd64.tar.gz
# kecs-v1.0.0-windows-amd64.zip
```

### Using GoReleaser

Create `.goreleaser.yml`:

```yaml
project_name: kecs

before:
  hooks:
    - go mod tidy
    - ./scripts/build-webui.sh

builds:
  - id: kecs
    main: ./controlplane/cmd/controlplane
    binary: kecs
    tags:
      - webui
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.date={{.Date}}
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - id: kecs-archive
    name_template: "kecs-v{{ .Version }}-{{ .Os }}-{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - LICENSE
      - README.md
      - docs/*

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

dockers:
  - image_templates:
      - "ghcr.io/nandemo-ya/kecs:{{ .Tag }}"
      - "ghcr.io/nandemo-ya/kecs:v{{ .Major }}"
      - "ghcr.io/nandemo-ya/kecs:v{{ .Major }}.{{ .Minor }}"
      - "ghcr.io/nandemo-ya/kecs:latest"
    dockerfile: Dockerfile
    build_flag_templates:
      - "--pull"
      - "--label=org.opencontainers.image.created={{.Date}}"
      - "--label=org.opencontainers.image.title={{.ProjectName}}"
      - "--label=org.opencontainers.image.revision={{.FullCommit}}"
      - "--label=org.opencontainers.image.version={{.Version}}"

release:
  github:
    owner: nandemo-ya
    name: kecs
  name_template: "v{{.Version}}"
  draft: true
  prerelease: auto
```

Run release:

```bash
# Dry run
goreleaser release --snapshot --clean

# Create release
goreleaser release
```

## CI/CD Build Pipeline

### GitHub Actions

```yaml
# .github/workflows/build.yml
name: Build

on:
  push:
    branches: [main]
    tags: ['v*']
  pull_request:

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      
      - name: Cache Go modules
        uses: actions/cache@v3
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
          restore-keys: |
            ${{ runner.os }}-go-
      
      - name: Cache npm dependencies
        uses: actions/cache@v3
        with:
          path: ~/.npm
          key: ${{ runner.os }}-node-${{ hashFiles('**/package-lock.json') }}
          restore-keys: |
            ${{ runner.os }}-node-
      
      - name: Build
        run: make all
      
      - name: Test
        run: make test
      
      - name: Upload artifacts
        uses: actions/upload-artifact@v3
        with:
          name: kecs-binary
          path: bin/kecs

  docker:
    runs-on: ubuntu-latest
    needs: build
    if: startsWith(github.ref, 'refs/tags/')
    steps:
      - uses: actions/checkout@v3
      
      - uses: docker/setup-qemu-action@v2
      
      - uses: docker/setup-buildx-action@v2
      
      - uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - uses: docker/build-push-action@v4
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            ghcr.io/nandemo-ya/kecs:latest
            ghcr.io/nandemo-ya/kecs:${{ github.ref_name }}
```

## Build Optimization

### Binary Size Reduction

```bash
# Strip debug information
go build -ldflags="-s -w"

# Use UPX compression
upx --best bin/kecs

# Comparison
ls -lh bin/
# -rwxr-xr-x  50M  kecs
# -rwxr-xr-x  35M  kecs (stripped)
# -rwxr-xr-x  12M  kecs (compressed)
```

### Build Performance

```makefile
# Parallel builds
GOMAXPROCS := $(shell nproc)

# Enable build cache
export GOCACHE := $(HOME)/.cache/go-build

# Incremental builds
.PHONY: build-fast
build-fast:
	go build -i -o bin/kecs ./cmd/controlplane
```

### Conditional Compilation

```go
// +build webui

package main

import (
    "embed"
    "net/http"
)

//go:embed all:web-ui/dist
var webUI embed.FS

func init() {
    http.Handle("/ui/", http.FileServer(http.FS(webUI)))
}
```

## Development Builds

### Hot Reload

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Configure .air.toml
air -c .air.toml
```

### Debug Builds

```bash
# Build with debug symbols
go build -gcflags="all=-N -l" -o bin/kecs-debug

# Build with race detector
go build -race -o bin/kecs-race

# Build with coverage
go test -c -covermode=atomic -o bin/kecs-test
```

## Package Management

### Homebrew Formula

```ruby
class Kecs < Formula
  desc "Kubernetes-based ECS Compatible Service"
  homepage "https://github.com/nandemo-ya/kecs"
  url "https://github.com/nandemo-ya/kecs/archive/v1.0.0.tar.gz"
  sha256 "..."
  license "MIT"

  depends_on "go" => :build
  depends_on "node" => :build

  def install
    system "make", "build"
    bin.install "bin/kecs"
  end

  test do
    assert_match "KECS version", shell_output("#{bin}/kecs version")
  end
end
```

### Debian Package

```bash
# Create package structure
mkdir -p debian/usr/local/bin
cp bin/kecs debian/usr/local/bin/

# Create control file
cat > debian/DEBIAN/control << EOF
Package: kecs
Version: 1.0.0
Architecture: amd64
Maintainer: KECS Team <team@kecs.dev>
Description: Kubernetes-based ECS Compatible Service
EOF

# Build package
dpkg-deb --build debian kecs_1.0.0_amd64.deb
```

## Troubleshooting Builds

### Common Issues

1. **Module errors**
   ```bash
   go mod download
   go mod tidy
   ```

2. **Node dependencies**
   ```bash
   cd web-ui
   rm -rf node_modules package-lock.json
   npm install
   ```

3. **Build cache issues**
   ```bash
   go clean -cache
   go clean -modcache
   ```

4. **Cross-compilation errors**
   ```bash
   # Install required tools
   apt-get install gcc-aarch64-linux-gnu
   ```

### Build Verification

```bash
# Verify binary
file bin/kecs
./bin/kecs version

# Verify Docker image
docker run --rm kecs:latest version

# Check dependencies
go mod graph
npm list --depth=0
```

## Next Steps

- [Testing Guide](./testing) - Running tests
- [Contributing](./contributing) - Contribution guidelines
- [Release Process](./release) - Detailed release procedures