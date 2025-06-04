# ADR-0008: SSG-based Documentation System

**Date:** 2025-06-04

## Status

Proposed

## Context

The KECS project is a complex system providing Amazon ECS-compatible APIs, requiring comprehensive documentation for developers. Currently, documentation is scattered across README files and ADRs, with the following challenges:

1. API specifications, usage examples, and configuration methods are not managed uniformly
2. Poor searchability of documentation
3. Difficulty in synchronizing version control with project code
4. Challenges in providing interactive examples and demos
5. Need for offline documentation access

## Decision

We will build a comprehensive documentation site for KECS using a Static Site Generator (SSG).

### Selected SSG: VitePress

Reasons for choosing VitePress:
- Vue.js-based, providing technical compatibility with existing Web UI
- Fast build and hot reload
- Markdown-centric content management
- Excellent built-in search functionality
- Customizable themes
- TypeScript support

### Documentation Structure

```
docs/
├── .vitepress/
│   ├── config.ts          # VitePress configuration
│   ├── theme/             # Custom theme
│   └── components/        # Custom Vue components
├── guide/
│   ├── getting-started.md # Quick start
│   ├── installation.md    # Installation guide
│   ├── configuration.md   # Configuration guide
│   └── examples/          # Usage examples
├── api/
│   ├── overview.md        # API overview
│   ├── cluster.md         # Cluster API
│   ├── service.md         # Service API
│   ├── task.md           # Task API
│   └── task-definition.md # Task Definition API
├── architecture/
│   ├── overview.md        # Architecture overview
│   ├── components.md      # Component details
│   └── kubernetes.md      # Kubernetes integration
├── operations/
│   ├── deployment.md      # Deployment
│   ├── monitoring.md      # Monitoring
│   └── troubleshooting.md # Troubleshooting
└── index.md              # Landing page
```

### Key Features

1. **Automated API Reference Generation**
   - Auto-generate API documentation from OpenAPI specifications
   - Embedded interactive API tester

2. **Code Example Execution Environment**
   - Sandboxed environment for code execution
   - Live demos connecting to actual KECS APIs

3. **Version Management**
   - Git tag-based versioning
   - Multi-version documentation support

4. **Search Functionality**
   - Full-text search
   - API endpoint search
   - Code example search

5. **Integration Features**
   - Direct links from Web UI
   - Quick access from CLI tools
   - IDE plugin integration

### Build Process

```makefile
# Makefile additions
docs-dev:
	cd docs && npm run dev

docs-build:
	cd docs && npm run build

docs-preview:
	cd docs && npm run preview
```

### Deployment

1. **Local Serving**
   - Embedded in KECS binary
   - Served at `http://localhost:8080/docs`

2. **Standalone**
   - Distributed as static files
   - Offline reference capability

3. **Hosting**
   - GitHub Pages
   - Custom domain publishing

## Consequences

### Benefits

1. **Improved Developer Experience**
   - Unified and searchable documentation
   - Interactive examples and demos
   - Offline access

2. **Maintainability**
   - Simple updates with Markdown
   - Version control synchronized with code
   - Automated build process

3. **Extensibility**
   - Easy addition of custom components
   - Multi-language support readiness
   - Plugin system

### Drawbacks

1. **Initial Setup**
   - VitePress configuration and customization
   - Migration of existing documentation

2. **Build Dependencies**
   - Requires Node.js environment
   - Additional build steps

3. **Size Increase**
   - Increased binary size (when embedded)
   - Additional distribution artifacts

### Implementation Plan

1. **Phase 1: Basic Setup** (1 week)
   - Create VitePress project
   - Configure basic structure
   - Integrate CI pipeline

2. **Phase 2: Content Migration** (2 weeks)
   - Organize and migrate existing documentation
   - Implement API reference generation
   - Create basic usage examples

3. **Phase 3: Advanced Features** (2 weeks)
   - Interactive components
   - Search optimization
   - Web UI integration

4. **Phase 4: Distribution and Integration** (1 week)
   - Binary embedding
   - Deployment process
   - Complete user guides

## References

- [VitePress Documentation](https://vitepress.dev/)
- [ADR-0005: Web UI](0005-web-ui.md)
- [ADR-0009: Local MCP Server](0009-local-mcp-server.md)