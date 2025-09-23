# macOS Code Signing and Notarization

This document describes how KECS binaries are signed and notarized for macOS to prevent Gatekeeper warnings.

## Overview

Starting from version 0.1.0, KECS macOS binaries are:
1. **Code signed** with a Developer ID certificate
2. **Notarized** by Apple
3. **Stapled** with the notarization ticket

This ensures users can run KECS without security warnings on macOS 10.15 (Catalina) and later.

## Requirements

### For Release Managers

1. Apple Developer Program membership ($99/year)
2. Developer ID Application certificate
3. App-specific password for notarization

### GitHub Secrets Required

Configure these secrets in the repository settings:

- `MACOS_DEVELOPER_ID`: Full certificate name (e.g., "Developer ID Application: Your Name (TEAM_ID)")
- `APPLE_ID`: Your Apple ID email
- `NOTARIZATION_PASSWORD`: App-specific password (format: xxxx-xxxx-xxxx-xxxx)
- `APPLE_TEAM_ID`: Your team ID (e.g., "ZTPL5R597W")

## Local Testing

### 1. Build and Sign

```bash
# Build the binary
make build-cli

# Sign the binary
codesign --force --options=runtime \
  --sign "Developer ID Application: Your Name (TEAM_ID)" \
  --timestamp \
  bin/kecs

# Verify signature
codesign -dv --verbose=4 bin/kecs
```

### 2. Notarize

```bash
# Create ZIP for notarization
ditto -c -k --keepParent bin/kecs kecs.zip

# Submit for notarization
xcrun notarytool submit kecs.zip \
  --apple-id "your-email@example.com" \
  --password "xxxx-xxxx-xxxx-xxxx" \
  --team-id "TEAM_ID" \
  --wait

# Staple the notarization (for DMG/PKG, not needed for direct downloads)
xcrun stapler staple bin/kecs
```

### 3. Test

```bash
# Add quarantine attribute (simulate download)
xattr -w com.apple.quarantine "0083;00000000;Chrome;" bin/kecs

# Try to run - should work without warning
./bin/kecs version
```

## CI/CD Process

The GoReleaser workflow automatically handles signing and notarization:

1. **Build**: Creates binaries for all platforms
2. **Sign**: macOS binaries are code signed
3. **Archive**: Creates tar.gz archives
4. **Notarize**: macOS archives are submitted to Apple
5. **Release**: Publishes to GitHub Releases

## Troubleshooting

### "Developer cannot be verified" error

If users still see this error:
1. The binary might not be properly notarized
2. The user might be on an older macOS version
3. Network issues prevented notarization status check

### Manual workaround for users

```bash
# Remove quarantine attribute
xattr -d com.apple.quarantine ./kecs

# Or right-click and select "Open" in Finder
```

## Alternative Installation

For users who prefer not to deal with Gatekeeper:

```bash
# Install via Homebrew (recommended)
brew tap nandemo-ya/kecs
brew install kecs
```

## References

- [Notarizing macOS Software](https://developer.apple.com/documentation/security/notarizing_macos_software)
- [Code Signing Guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/CodeSigningGuide/)
- [GoReleaser Code Signing](https://goreleaser.com/customization/sign/)