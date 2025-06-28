# Contributing to KECS

Thank you for your interest in contributing to KECS! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker
- Make

### Getting Started

1. Fork and clone the repository:
```bash
git clone https://github.com/your-username/kecs.git
cd kecs
```

2. Set up git hooks (required for all contributors):
```bash
./scripts/setup-lefthook.sh
```

This installs [Lefthook](https://github.com/evilmartians/lefthook) and configures git hooks that will:
- **Pre-commit**: Run unit tests, `go fmt`, and `go vet` on changed files
- **Pre-push**: Run the full test suite with race detection

### Building

```bash
cd controlplane
make build
```

### Testing

Run all tests:
```bash
make test
```

Run tests with coverage:
```bash
make test-coverage
```

Run specific tests:
```bash
go test ./internal/storage/... -v
```

## Git Hooks

We use Lefthook to ensure code quality. The hooks are mandatory for all contributions.

### Pre-commit Hook
Runs automatically before each commit:
- Go formatting check (`gofmt`)
- Go vet analysis
- Unit tests for changed packages

### Pre-push Hook
Runs automatically before pushing:
- Full test suite with race detection
- Coverage report generation

### Skipping Hooks
In exceptional cases, you can skip hooks:
```bash
git commit --no-verify
git push --no-verify
```

**Note**: PRs that fail tests will not be merged, so skipping hooks is not recommended.

## Pull Request Process

1. Ensure all tests pass locally
2. Update documentation if needed
3. Add tests for new functionality
4. Ensure your code follows Go conventions
5. Create a pull request with a clear description

### PR Title Format
Use conventional commits format:
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Test additions or fixes
- `refactor:` Code refactoring
- `chore:` Maintenance tasks

Example: `feat: Add support for ECS service discovery`

## Code Style

- Follow standard Go conventions
- Run `gofmt` on all code
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small

## Testing Guidelines

- Write tests for all new functionality
- Maintain or improve code coverage
- Use table-driven tests where appropriate
- Mock external dependencies
- Test both success and error cases

## Reporting Issues

- Use GitHub Issues for bug reports and feature requests
- Provide clear reproduction steps for bugs
- Include relevant logs and error messages
- Specify your environment (OS, Go version, etc.)

## Questions?

Feel free to open an issue for any questions about contributing.