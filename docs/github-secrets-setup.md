# GitHub Secrets Setup for Code Signing

## Required Secrets

Add these secrets to your GitHub repository:
`Settings` → `Secrets and variables` → `Actions` → `New repository secret`

### 1. MACOS_DEVELOPER_ID
- **Value**: Full certificate name
- **Example**: `Developer ID Application: Akinori Yamada (ZTPL5R597W)`
- **How to find**: Run `security find-identity -v -p codesigning`

### 2. APPLE_ID
- **Value**: Your Apple ID email
- **Example**: `developer@example.com`

### 3. NOTARIZATION_PASSWORD
- **Value**: App-specific password
- **Format**: `xxxx-xxxx-xxxx-xxxx`
- **How to create**:
  1. Go to [appleid.apple.com](https://appleid.apple.com)
  2. Sign in and go to "App-Specific Passwords"
  3. Generate a new password for "KECS Notarization"

### 4. APPLE_TEAM_ID
- **Value**: Your Team ID
- **Example**: `ZTPL5R597W`
- **How to find**: Check your certificate details or Apple Developer account

### 5. HOMEBREW_TAP_GITHUB_TOKEN (Already configured)
- **Value**: Personal Access Token with repo scope
- **Purpose**: Update homebrew-kecs repository

## Verification

After setting up the secrets, you can verify they're configured by:

1. Going to the Actions tab
2. Checking the GoReleaser workflow
3. The secrets should appear as `***` in logs

## Security Notes

- Never commit these values to the repository
- Rotate app-specific passwords periodically
- Use repository environments for additional security if needed

## Testing

You can test the setup by creating a test tag:

```bash
# Create and push test tag
git tag v0.0.1-test
git push origin v0.0.1-test

# After verification, clean up the test tag
git push --delete origin v0.0.1-test
git tag -d v0.0.1-test
```

Then monitor the GoReleaser workflow in the Actions tab.