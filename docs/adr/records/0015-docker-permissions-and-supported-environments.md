# ADR-0015. Docker Permissions and Supported Environments

Date: 2025-06-26

## Status

Accepted

## Context

KECS requires the ability to create and manage local Kubernetes clusters (k3d/kind) to provide full Amazon ECS compatibility. This functionality requires access to the Docker daemon, which has significant security implications.

We needed to decide:
1. Whether to support Docker access in all distribution methods
2. How to handle the security implications
3. Which environments to officially support

### Options Considered

1. **Separate images with/without Docker access**
   - `kecs:latest` without Docker access (distroless)
   - `kecs:docker` with Docker access
   - Pros: Clear security boundaries
   - Cons: Confusing for users, limited functionality in default image

2. **Binary-only distribution**
   - No container images
   - Pros: Simpler permissions model
   - Cons: Loses Testcontainers support, harder CI/CD integration

3. **All images with Docker access + clear disclaimer**
   - All distribution methods support full functionality
   - Clear documentation about security implications
   - Pros: Consistent experience, full functionality
   - Cons: Requires elevated permissions

## Decision

We will provide Docker daemon access in all distribution methods (binary and container images) with clear disclaimers about the security implications and supported environments.

### Supported Environments

- ✅ Local development machines
- ✅ CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins, etc.)
- ✅ Isolated test environments
- ❌ Production environments
- ❌ Public-facing deployments
- ❌ Multi-tenant systems

### Security Model

1. **Explicit Acknowledgment**: Users must acknowledge the security implications on first run
2. **Clear Documentation**: README and documentation clearly state the permission requirements
3. **Environment Restrictions**: Explicitly document that KECS is not for production use

### Implementation

All container images will:
- Include docker-cli for k3d management
- Run as non-root user in docker group
- Include entrypoint script to handle Docker socket permissions
- Show disclaimer on first run

## Consequences

### Positive

- **Consistent Experience**: All users get full functionality regardless of installation method
- **Testcontainers Support**: Container images work seamlessly with Testcontainers
- **Simple Mental Model**: One product, one set of features
- **Development Focus**: Optimized for the primary use case (local development)

### Negative

- **Security Concerns**: Requires elevated permissions (Docker socket access)
- **Limited Use Cases**: Cannot be safely used in production or multi-tenant environments
- **Trust Requirement**: Users must trust KECS with Docker daemon access

### Mitigation

- Clear and prominent disclaimers in documentation
- First-run acknowledgment requirement
- Option to disable k3d features (though Docker access still required)
- Comparison with similar tools (Docker Desktop, kind, etc.) to set expectations

## References

- [Docker Socket Security](https://docs.docker.com/engine/security/#docker-daemon-attack-surface)
- [Kind Security Model](https://kind.sigs.k8s.io/docs/user/security/)
- [Testcontainers Requirements](https://www.testcontainers.org/)