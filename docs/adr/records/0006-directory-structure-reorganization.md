# ADR-0006: Directory Structure Reorganization

**Date:** 2025-06-01

## Status

Proposed

## Context

KECS (Kubernetes-based ECS Compatible Service) is expanding to include a Web UI component as outlined in ADR-0005. The current directory structure places all Go code at the repository root, which will become problematic when we add the Web UI implementation. We need a clear separation between the control plane (Go) and web UI (React/TypeScript) components.

Current structure:
```
kecs/
├── cmd/           # Go command line tools
├── internal/      # Go internal packages
├── go.mod         # Go module file
├── Dockerfile     # Docker configuration
└── ...
```

This flat structure doesn't provide clear boundaries between different components of the system.

## Decision

We will reorganize the directory structure to separate the control plane and web UI components into distinct directories at the repository root.

New structure:
```
kecs/
├── controlplane/          # Go-based control plane
│   ├── cmd/              # Command line tools
│   ├── internal/         # Internal packages
│   ├── go.mod           # Go module configuration
│   ├── go.sum           # Go dependencies
│   └── Dockerfile       # Control plane Docker image
├── web-ui/               # React/TypeScript Web UI
│   ├── src/             # Source code
│   ├── public/          # Static assets
│   ├── package.json     # npm configuration
│   └── tsconfig.json    # TypeScript configuration
├── api-models/           # Shared API definitions (unchanged)
├── docs/                 # Documentation (unchanged)
├── examples/             # Example configurations (unchanged)
├── bin/                  # Build outputs (unchanged)
├── Makefile             # Root build orchestration
├── README.md            # Project documentation
└── LICENSE              # License file
```

### Implementation Details

1. **Module Path Update**: The Go module path changes from `github.com/nandemo-ya/kecs` to `github.com/nandemo-ya/kecs/controlplane`

2. **Import Path Updates**: All Go imports must be updated to reflect the new module path

3. **Build Process**: The root Makefile orchestrates builds for both components

4. **Docker Context**: The Dockerfile moves into the controlplane directory since it's specific to that component

## Consequences

### Positive Outcomes

- **Clear Component Separation**: Each major component has its own directory with appropriate tooling
- **Independent Development**: Teams can work on control plane and web UI independently
- **Better Organization**: Related files are grouped together (e.g., Dockerfile with the code it builds)
- **Scalability**: Easy to add more components in the future (e.g., CLI tools, SDKs)
- **Tooling Isolation**: Each component can have its own dependencies and build tools

### Challenges and Trade-offs

- **Import Path Changes**: All existing Go imports need to be updated
- **Build Complexity**: The root Makefile becomes more complex to handle multiple components
- **Development Setup**: Developers need to understand the new structure
- **CI/CD Updates**: Build pipelines need to be updated for the new paths

### Migration Plan

1. Create new directory structure
2. Move Go source files to `controlplane/`
3. Update Go module path in `go.mod`
4. Update all import paths in Go files
5. Move Dockerfile to `controlplane/`
6. Update Makefile targets for new paths
7. Update `.gitignore` for both Go and Node.js artifacts
8. Test the build process
9. Update documentation

## Alternatives Considered

### Keep Flat Structure
- **Pros**: No migration needed, simpler for small projects
- **Cons**: Mixing of Go and JavaScript tooling, unclear boundaries
- **Rejected**: Doesn't scale well with multiple components

### Monorepo Tools (Nx, Lerna)
- **Pros**: Sophisticated dependency management, shared tooling
- **Cons**: Additional complexity, learning curve
- **Rejected**: Overkill for a two-component system

### Separate Repositories
- **Pros**: Complete isolation, independent versioning
- **Cons**: Complex coordination, difficult local development
- **Rejected**: Goes against the integrated nature of KECS

## References

- [Go Modules Documentation](https://golang.org/ref/mod)
- [Monorepo Best Practices](https://monorepo.tools/)
- [ADR-0005: Web UI Integration](./0005-web-ui.md)