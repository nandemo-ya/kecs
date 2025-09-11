# Greptile PR Review Workflow

## Overview
This document describes the workflow for handling Greptile automated code reviews in the KECS project.

## Workflow Steps

### 1. PR Creation
When a PR is created, Greptile automatically analyzes the code and provides:
- Summary of changes
- Confidence score (1-5)
- Inline code review comments
- Compilation error detection
- Logic issues and improvements

### 2. Review Greptile Comments
After PR creation, developers should:

```bash
# View PR with Greptile comments
gh pr view <PR_NUMBER> --comments

# Get detailed inline comments
gh api repos/nandemo-ya/kecs/pulls/<PR_NUMBER>/comments --jq '.[] | "File: \(.path)\nLine: \(.line)\nComment: \(.body)\n---"'
```

### 3. Comment Classification

#### Must Fix (Blocking Issues)
- **Compilation errors**: Code that won't compile
- **Missing methods/functions**: Undefined references
- **Security vulnerabilities**: Token exposure, unsafe operations
- **Data corruption risks**: Operations that could lose data

**Action**: Fix immediately before merge

#### Should Fix (Important Issues)
- **Logic errors**: Incorrect business logic
- **Resource leaks**: Unclosed connections, goroutine leaks
- **Race conditions**: Concurrent access issues
- **API contract violations**: Breaking changes

**Action**: Fix or create follow-up issue with justification

#### Consider Fixing (Improvements)
- **Code style**: Naming, formatting issues
- **Performance**: Non-critical optimizations
- **Code duplication**: Refactoring opportunities
- **Documentation**: Missing or unclear comments

**Action**: Evaluate and fix if it improves code quality

### 4. Response Actions

#### For Critical Issues (Confidence Score 1-2)
1. **Do NOT merge** until issues are resolved
2. Fix compilation errors immediately
3. Address all "Must Fix" items
4. Re-run tests after fixes

Example response:
```bash
# Fix critical issues
git checkout <branch>
# Make fixes based on Greptile comments
make test
git add .
git commit -m "fix: Address Greptile review comments"
git push
```

#### For Non-Critical Issues (Confidence Score 3-5)
1. Evaluate each comment's validity
2. Fix agreed-upon issues
3. Document why certain suggestions weren't implemented
4. Create follow-up issues for deferred improvements

### 5. Comment Resolution

When addressing Greptile comments:

1. **Fix in current PR**: For critical and simple fixes
   ```bash
   # Apply suggestion directly if provided
   gh api repos/nandemo-ya/kecs/pulls/<PR_NUMBER>/comments/<COMMENT_ID>/reactions \
     --method POST -f content='+1'
   ```

2. **Create follow-up issue**: For complex improvements
   ```bash
   gh issue create --title "Follow-up: <description>" \
     --body "Greptile suggested: <suggestion>\nPR: #<PR_NUMBER>" \
     --label "enhancement,greptile-suggestion"
   ```

3. **Dismiss with explanation**: For invalid suggestions
   - Add a comment explaining why the suggestion doesn't apply
   - Reference architecture decisions or project constraints

### 6. Automation Integration

#### GitHub Actions Workflow
Create `.github/workflows/greptile-check.yml`:

```yaml
name: Greptile Review Check

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  check-greptile:
    runs-on: ubuntu-latest
    steps:
      - name: Wait for Greptile Review
        run: sleep 30  # Give Greptile time to analyze
      
      - name: Check for Critical Issues
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          # Check if Greptile found compilation errors
          COMMENTS=$(gh api repos/${{ github.repository }}/pulls/${{ github.event.pull_request.number }}/comments)
          if echo "$COMMENTS" | grep -q "compilation error\|will cause a compilation error"; then
            echo "::error::Greptile found compilation errors. Please fix before merge."
            exit 1
          fi
```

## Best Practices

### 1. Pre-PR Checklist
Before creating a PR:
- Run `make test` locally
- Run `make vet` for static analysis
- Check for missing imports/methods
- Ensure all new methods are implemented

### 2. Greptile Comment Handling
- **Always review** Greptile comments, even with high confidence scores
- **Prioritize** compilation and logic errors
- **Document** why suggestions are not implemented
- **Learn** from repeated patterns in reviews

### 3. Team Collaboration
- Discuss Greptile suggestions in PR reviews
- Share knowledge about false positives
- Update this document with new patterns
- Adjust Greptile settings based on project needs

## Configuration

### Greptile Settings
Access at: https://app.greptile.com/review/github

Recommended settings for KECS:
- **Review Level**: Detailed
- **Focus Areas**: 
  - Compilation errors
  - Security issues
  - API compatibility
  - Resource management
- **Language-specific**:
  - Go: Check for goroutine leaks, proper error handling
  - TypeScript: Type safety, null checks

### Exemptions
Some files/patterns may generate false positives:
- Generated code (mocks, protobuf)
- Vendor directories
- Test fixtures

Add `.greptile.yml` to configure:
```yaml
ignore:
  - "**/vendor/**"
  - "**/*_generated.go"
  - "**/mocks/**"
```

## Metrics and Improvement

Track Greptile effectiveness:
1. **True positive rate**: Valid issues found
2. **False positive rate**: Invalid suggestions
3. **Fix rate**: Suggestions implemented
4. **Time saved**: Bugs caught before merge

Monthly review:
- Analyze patterns in Greptile comments
- Update coding standards based on common issues
- Adjust Greptile configuration
- Share learnings with team

## Examples from KECS

### Example 1: Compilation Error (PR #594)
```
File: controlplane/internal/restoration/service.go
Line: 119
Comment: Method `RestoreService` does not exist in ServiceManager. This will cause a compilation error.
```
**Action**: Must fix immediately - implement missing method

### Example 2: Logic Issue (PR #594)
```
File: controlplane/internal/kubernetes/task_manager.go
Line: 1133
Comment: Using `context.Background()` instead of the passed `ctx` parameter may prevent proper cancellation propagation.
```
**Action**: Should fix - use proper context for cancellation

### Example 3: Style Suggestion (PR #594)
```
File: controlplane/internal/host/instance/manager.go
Line: 225
Comment: This port tracking logic is duplicated. Consider extracting to a helper method.
```
**Action**: Consider - refactor if time permits or create follow-up issue