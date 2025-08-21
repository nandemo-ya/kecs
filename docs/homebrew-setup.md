# Homebrew Setup for KECS

This document describes how to set up Homebrew distribution for KECS.

## Prerequisites

1. **Create Homebrew Tap Repository**
   - Create a new repository named `homebrew-kecs` in the nandemo-ya organization
   - Repository URL: `https://github.com/nandemo-ya/homebrew-kecs`

2. **Set up GitHub Token**
   - Create a Personal Access Token with `repo` scope
   - Add it as a secret named `HOMEBREW_TAP_GITHUB_TOKEN` in the main KECS repository

## Repository Structure

### homebrew-kecs repository structure:
```
homebrew-kecs/
├── Formula/
│   └── kecs.rb        # Homebrew formula
└── README.md
```

### Initial Formula Template

Create `Formula/kecs.rb` in the homebrew-kecs repository:

```ruby
class Kecs < Formula
  desc "Kubernetes-based ECS Compatible Service"
  homepage "https://github.com/nandemo-ya/kecs"
  version "0.1.0"
  license "Apache-2.0"

  # URLs will be automatically updated by GitHub Actions
  on_macos do
    if Hardware::CPU.intel?
      url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Darwin_x86_64.tar.gz"
      sha256 "PLACEHOLDER"
    else
      url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    if Hardware::CPU.intel?
      url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Linux_x86_64.tar.gz"
      sha256 "PLACEHOLDER"
    else
      url "https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Linux_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "kecs"
  end

  test do
    system "#{bin}/kecs", "version"
  end
end
```

## Release Process

### Automated Release (Recommended)

1. **Tag and Push**
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. **GitHub Actions will automatically:**
   - Build binaries for all platforms
   - Create GitHub Release with binaries
   - Update Homebrew formula with correct URLs and SHA256

3. **Users can install via:**
   ```bash
   brew tap nandemo-ya/kecs
   brew install kecs
   ```

### Manual Release Process

If you need to release manually:

1. **Build binaries for each platform:**
   ```bash
   # macOS AMD64
   CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o kecs ./controlplane/cmd/controlplane
   tar czf kecs_v0.1.0_Darwin_x86_64.tar.gz kecs README.md LICENSE

   # macOS ARM64
   CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o kecs ./controlplane/cmd/controlplane
   tar czf kecs_v0.1.0_Darwin_arm64.tar.gz kecs README.md LICENSE

   # Linux AMD64
   CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o kecs ./controlplane/cmd/controlplane
   tar czf kecs_v0.1.0_Linux_x86_64.tar.gz kecs README.md LICENSE

   # Linux ARM64
   CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -o kecs ./controlplane/cmd/controlplane
   tar czf kecs_v0.1.0_Linux_arm64.tar.gz kecs README.md LICENSE
   ```

2. **Generate checksums:**
   ```bash
   shasum -a 256 kecs_*.tar.gz > checksums.txt
   ```

3. **Create GitHub Release and upload binaries**

4. **Update Homebrew formula with correct SHA256 values**

## Testing

### Local Testing

Before releasing, test the formula locally:

```bash
# Clone the tap
git clone https://github.com/nandemo-ya/homebrew-kecs
cd homebrew-kecs

# Test installation
brew install --build-from-source Formula/kecs.rb

# Test the binary
kecs version
```

### CI Testing

The GitHub Actions workflow includes tests for:
- Binary execution
- Version command
- Basic functionality

## Troubleshooting

### Common Issues

1. **CGO Dependencies**
   - KECS requires CGO for DuckDB integration
   - Ensure build environment has proper C/C++ compilers

2. **Cross-compilation**
   - macOS binaries must be built on macOS runners
   - Linux ARM64 requires cross-compilation tools

3. **Formula Updates**
   - Formula must be updated with each release
   - SHA256 must match exactly

## Alternative: GoReleaser

For more complex release management, consider using GoReleaser:

1. Use the `.goreleaser.yml` configuration
2. Install GoReleaser: `brew install goreleaser`
3. Test locally: `goreleaser release --snapshot --clean`
4. Release: `goreleaser release`

Note: GoReleaser with CGO requires additional setup for cross-compilation.