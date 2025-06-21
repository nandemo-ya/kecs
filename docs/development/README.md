# KECS Development Documentation

This directory contains technical documentation for KECS developers and contributors.

## Contents

### Troubleshooting & Known Issues

- [AWS CLI Compatibility Issue](./AWS_CLI_COMPATIBILITY_ISSUE.md) - Field name casing issues when migrating to AWS SDK Go v2

### Development Guides

- [Worktree Workflow](./worktree-workflow.md) - Git worktree based development workflow

## For Contributors

When encountering technical issues or discovering important implementation details:

1. Document the issue and solution in this directory
2. Use descriptive filenames (e.g., `AWS_SDK_V2_MIGRATION_NOTES.md`)
3. Include:
   - Problem description
   - Root cause analysis
   - Solution/workaround
   - Test results
   - Recommendations for permanent fixes

## Categories

- **Migration Notes**: Issues and solutions when migrating between libraries/versions
- **Compatibility**: Cross-client/cross-language compatibility issues
- **Performance**: Performance optimization findings
- **Architecture**: Important architectural decisions and their rationale