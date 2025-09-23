# Contributing to KECS

Thank you for your interest in contributing to KECS! This guide will help you get started with contributing to the project.

## Code of Conduct

Please read and follow our [Code of Conduct](https://github.com/nandemo-ya/kecs/blob/main/CODE_OF_CONDUCT.md) to ensure a welcoming environment for all contributors.

## How to Contribute

### Reporting Issues

1. **Check existing issues** to avoid duplicates
2. **Use issue templates** when available
3. **Provide detailed information**:
   - KECS version
   - Operating system
   - Reproduction steps
   - Expected vs actual behavior
   - Relevant logs or error messages

### Suggesting Features

1. **Open a discussion** first for major features
2. **Explain the use case** and benefits
3. **Consider implementation complexity**
4. **Be open to feedback** and alternatives

### Contributing Code

1. **Fork the repository**
2. **Create a feature branch** from `main`
3. **Make your changes** following our guidelines
4. **Write tests** for new functionality
5. **Update documentation** as needed
6. **Submit a pull request**

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker Desktop
- Make
- Git

### Setting Up Your Development Environment

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/kecs.git
cd kecs

# Add upstream remote
git remote add upstream https://github.com/nandemo-ya/kecs.git

# Install dependencies
make deps

# Build the project
make build

# Run tests
make test
```

### Development Workflow

```bash
# Create a feature branch
git checkout -b feat/your-feature-name

# Make changes and test
make test

# Commit with conventional commits
git commit -m "feat: add new feature"

# Push to your fork
git push origin feat/your-feature-name

# Open a pull request
```

## Coding Standards

### Go Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Use `golint` for linting
- Write idiomatic Go code

```bash
# Format code
make fmt

# Run linter
make vet
```

### TypeScript/React Style

- Use TypeScript for type safety
- Follow React best practices
- Use functional components and hooks
- Format with Prettier

```bash

# Run linter
npm run lint
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Build process or auxiliary tool changes

Examples:
```
feat(api): add support for task definition tags

fix(storage): resolve connection leak in DuckDB pool

docs(api): update cluster API documentation
```

### Code Organization

```
kecs/
├── cmd/                    # Command-line interfaces
├── internal/               # Internal packages
│   ├── controlplane/      # Control plane logic
│   ├── storage/           # Storage implementations
│   ├── kubernetes/        # Kubernetes integration
│   └── converters/        # Type converters
├── pkg/                   # Public packages
├── api/                   # API definitions
├── tests/                # Integration tests
└── docs/                 # Documentation
```

## Testing Guidelines

### Unit Tests

- Write tests for all new code
- Maintain test coverage above 80%
- Use table-driven tests
- Mock external dependencies

```go
func TestCreateCluster(t *testing.T) {
    tests := []struct {
        name    string
        input   *CreateClusterInput
        want    *Cluster
        wantErr bool
    }{
        {
            name: "valid cluster",
            input: &CreateClusterInput{
                ClusterName: "test-cluster",
            },
            want: &Cluster{
                ClusterName: "test-cluster",
                Status:      "ACTIVE",
            },
            wantErr: false,
        },
        // Add more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CreateCluster(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateCluster() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("CreateCluster() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

```go
// +build integration

func TestClusterLifecycle(t *testing.T) {
    // Start KECS server
    server := startTestServer(t)
    defer server.Stop()
    
    // Test cluster creation
    cluster, err := server.Client.CreateCluster(&CreateClusterInput{
        ClusterName: "integration-test",
    })
    require.NoError(t, err)
    assert.Equal(t, "ACTIVE", cluster.Status)
    
    // Test cluster deletion
    err = server.Client.DeleteCluster(&DeleteClusterInput{
        Cluster: "integration-test",
    })
    require.NoError(t, err)
}
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Specific package
go test ./internal/storage/...

# With coverage
make test-coverage
```

## Documentation

### Code Documentation

- Add godoc comments to all exported types and functions
- Include examples where helpful
- Keep comments up-to-date with code changes

```go
// CreateCluster creates a new ECS cluster with the specified configuration.
// It returns the created cluster or an error if creation fails.
//
// Example:
//
//	cluster, err := client.CreateCluster(&CreateClusterInput{
//	    ClusterName: "production",
//	    Tags: []Tag{{Key: "env", Value: "prod"}},
//	})
func CreateCluster(input *CreateClusterInput) (*Cluster, error) {
    // Implementation
}
```

### User Documentation

- Update relevant guides when adding features
- Add examples for new functionality
- Keep API documentation accurate
- Update configuration references

## Pull Request Process

### Before Submitting

1. **Test your changes**
   ```bash
   make test
   make test-integration
   ```

2. **Update documentation**
   - API docs for new endpoints
   - Configuration guide for new options
   - Examples for new features

3. **Check code quality**
   ```bash
   make fmt
   make vet
   golangci-lint run
   ```

### PR Guidelines

1. **Title**: Use conventional commit format
2. **Description**: Explain what and why
3. **Testing**: Describe testing performed
4. **Breaking changes**: Clearly note any
5. **Issues**: Reference related issues

Example PR description:
```markdown
## Description
This PR adds support for ECS service auto-scaling by implementing the Application Auto Scaling APIs.

## Changes
- Add auto-scaling target registration
- Implement scaling policy management
- Add CloudWatch metrics integration

## Testing
- Unit tests for scaling logic
- Integration tests with Kind cluster
- Manual testing with sample application

## Breaking Changes
None

Fixes #123
```

### Review Process

1. **Automated checks** must pass
2. **Code review** by maintainers
3. **Address feedback** promptly
4. **Keep PR updated** with main branch
5. **Squash commits** when requested

## Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):
- MAJOR: Breaking changes
- MINOR: New features
- PATCH: Bug fixes

### Release Steps

1. **Update version** in relevant files
2. **Update CHANGELOG.md**
3. **Create release PR**
4. **Tag release** after merge
5. **Build and publish** artifacts

## Getting Help

### Resources

- [Documentation](https://kecs.dev)

### Communication

- **Questions**: Open an issue
- **Bugs**: Open an issue
- **Features**: Open an issue to discuss first
- **Security**: Open an issue with the security label

## Recognition

Contributors are recognized in:
- [CONTRIBUTORS.md](https://github.com/nandemo-ya/kecs/blob/main/CONTRIBUTORS.md)
- Release notes
- Project documentation

Thank you for contributing to KECS!