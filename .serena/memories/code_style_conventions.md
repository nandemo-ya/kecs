# KECS Code Style and Conventions

## Go Code Style
- **Formatting**: Uses `gofmt` for automatic formatting
- **Imports**: Uses `goimports` with local import prefix `github.com/nandemo-ya/kecs`
- **Linting**: Uses `golangci-lint` for code quality checks
- **Naming**: Follows standard Go naming conventions (PascalCase for exports, camelCase for internal)

## Testing Conventions
- **Framework**: Uses Ginkgo for BDD-style testing with Gomega matchers
- **Test Structure**: 
  ```go
  var _ = Describe("ComponentName", func() {
      Context("when condition", func() {
          It("should behavior", func() {
              Expect(actual).To(Equal(expected))
          })
      })
  })
  ```
- **File Naming**: Test files end with `*_test.go`
- **Location**: Tests are placed alongside the code they test

## API Implementation Pattern
- Each ECS resource type has its own file in `internal/controlplane/api/`
- Request/Response struct definitions match AWS ECS API exactly
- Handler functions are registered in `api/server.go`
- Follow AWS ECS API naming conventions exactly

## Code Organization
- **Clean Architecture**: Clear separation between API, business logic, and infrastructure layers
- **Context Usage**: Context-based cancellation throughout the codebase
- **Error Handling**: Proper error handling and propagation
- **Interfaces**: Use interfaces for dependency injection and testing

## Documentation
- **Comments**: Go doc-style comments for exported functions and types
- **README**: Project documentation in markdown
- **ADRs**: Architectural Decision Records in `docs/adr/records/`

## Git Conventions
- **Branching**: 
  - Feature branches: `feat/feature-name`
  - Bug fixes: `fix/bug-name`
- **Commits**: Clear, descriptive commit messages
- **PR Requirements**: All tests must pass before creating PR