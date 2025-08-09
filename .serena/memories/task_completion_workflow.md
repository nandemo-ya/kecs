# Task Completion Workflow for KECS

## Before Starting Implementation
1. **Create feature branch**:
   ```bash
   git checkout -b feat/feature-name  # For features
   git checkout -b fix/bug-name       # For bug fixes
   ```

## During Development
1. **Code formatting and quality**:
   ```bash
   make fmt    # Format code and organize imports
   make vet    # Run go vet
   make lint   # Run golangci-lint (optional but recommended)
   ```

2. **Testing during development**:
   ```bash
   make test   # Run all tests with race detection
   # or for specific components
   cd controlplane && ginkgo -r
   ```

## Before Completing Task
1. **Run full test suite**:
   ```bash
   cd controlplane && ginkgo -r         # Control plane tests
   cd controlplane && go test ./...     # Alternative test command
   ```

2. **Verify code quality**:
   ```bash
   make all    # Runs clean, fmt, vet, test, and build
   ```

3. **Test CI/CD changes locally** (if applicable):
   ```bash
   act -W .github/workflows/workflow-name.yml -j job-name --container-architecture linux/amd64
   ```

## Task Completion Requirements
- **All tests must pass** before creating PR
- **Code must be formatted** (make fmt)
- **No linting errors** (if using make lint)
- **All changes committed** to feature branch
- **Hot reload tested** if applicable (make dev)

## For ECS API Endpoint Implementation
1. Add type definitions to appropriate file in `internal/controlplane/api/`
2. Implement handler function following existing patterns
3. Register handler in `api/server.go`
4. Follow AWS ECS API naming conventions exactly
5. Write Ginkgo tests covering:
   - Success cases
   - Error cases  
   - Edge cases (idempotency, empty inputs)
   - AWS ECS compatibility behavior

## Final Checks
- Tests pass: `make test`
- Code builds: `make build`
- Hot reload works: `make dev` (if applicable)
- Documentation updated if needed