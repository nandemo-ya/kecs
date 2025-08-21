# KECS Versioning Strategy

## Version Numbering

KECS follows Semantic Versioning (SemVer) with a simplified approach:

### Development Phase (v0.0.x)
- **v0.0.1 - v0.0.9**: Early development, alpha quality
- Features may change rapidly
- Not recommended for production use
- Not published to Homebrew stable channel

### Pre-release Phase (v0.0.10+)
- **v0.0.10 - v0.0.x**: Beta quality
- API stabilizing
- Can be used for testing and development
- Still not in Homebrew stable

### Stable Releases (v0.1.0+)
- **v0.1.0**: First stable release
- Published to Homebrew
- Recommended for production use
- Backward compatibility maintained within minor versions

### Version Examples
```
v0.0.1   # Initial development release
v0.0.2   # Bug fixes and early features
v0.0.10  # Beta-quality release
v0.1.0   # First stable release (GA)
v0.1.1   # Patch release (bug fixes)
v0.2.0   # Minor release (new features)
v1.0.0   # Major release (after proven stability)
```

## Release Process

### Development Releases (v0.0.x)
1. Tag and push:
   ```bash
   git tag v0.0.1
   git push origin v0.0.1
   ```
2. GitHub Release created automatically
3. Binaries available for download
4. Docker image published
5. **NOT** published to Homebrew stable

### Stable Releases (v0.1.0+)
1. Tag and push:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
2. GitHub Release created automatically
3. Binaries available for download
4. Docker image published
5. **Homebrew formula automatically updated**

## Installation Methods by Version

### For Development Versions (v0.0.x)
```bash
# Direct download from GitHub Releases
curl -L https://github.com/nandemo-ya/kecs/releases/download/v0.0.1/kecs_v0.0.1_Darwin_arm64.tar.gz | tar xz

# Docker
docker pull ghcr.io/nandemo-ya/kecs:v0.0.1

# Build from source
git clone https://github.com/nandemo-ya/kecs
cd kecs
git checkout v0.0.1
make build
```

### For Stable Versions (v0.1.0+)
```bash
# Homebrew (recommended)
brew tap nandemo-ya/kecs
brew install kecs

# Direct download
curl -L https://github.com/nandemo-ya/kecs/releases/download/v0.1.0/kecs_v0.1.0_Darwin_arm64.tar.gz | tar xz

# Docker
docker pull ghcr.io/nandemo-ya/kecs:v0.1.0
# or use 'latest' for the newest stable
docker pull ghcr.io/nandemo-ya/kecs:latest
```

## Version Guarantees

### v0.0.x (Development)
- No compatibility guarantees
- Features may be added, changed, or removed
- Configuration format may change
- Use for testing and development only

### v0.x.y (Pre-1.0 Stable)
- Backward compatibility within minor versions (0.1.x)
- Breaking changes only in minor version bumps (0.1 → 0.2)
- Bug fixes in patch versions (0.1.0 → 0.1.1)
- Production use is acceptable with caution

### v1.0.0+ (Future)
- Full semantic versioning guarantees
- Breaking changes only in major versions
- Enterprise-ready

## Docker Tag Strategy

| Version | Docker Tags |
|---------|------------|
| v0.0.1 | `ghcr.io/nandemo-ya/kecs:v0.0.1` |
| v0.1.0 | `ghcr.io/nandemo-ya/kecs:v0.1.0`, `ghcr.io/nandemo-ya/kecs:0.1`, `ghcr.io/nandemo-ya/kecs:latest` |
| v0.1.1 | `ghcr.io/nandemo-ya/kecs:v0.1.1`, `ghcr.io/nandemo-ya/kecs:0.1`, `ghcr.io/nandemo-ya/kecs:latest` |
| v1.0.0 | `ghcr.io/nandemo-ya/kecs:v1.0.0`, `ghcr.io/nandemo-ya/kecs:1.0`, `ghcr.io/nandemo-ya/kecs:1`, `ghcr.io/nandemo-ya/kecs:latest` |

## FAQ

### Why not use -alpha/-beta/-rc tags?
- Simplicity: Easier to manage and understand
- Homebrew compatibility: Works seamlessly with Homebrew
- Clear progression: Obvious which versions are stable (>= 0.1.0)

### When will v1.0.0 be released?
After v0.x has proven stability in production environments and the API is fully stable.

### Can I use v0.0.x in production?
Not recommended. These are development versions with no stability guarantees.

### Can I use v0.1.0+ in production?
Yes, with appropriate testing. These are stable releases with backward compatibility within minor versions.