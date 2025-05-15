# Development Environment, Tools, and Libraries

**Date:** 2025-05-15

## Status

Proposed

## Context

KECS (Kubernetes-based ECS Compatible Service) requires a well-defined development environment, tools, and libraries to ensure consistent development, testing, and deployment. This ADR outlines the decisions regarding the runtime environment, development language, libraries, data stores, and development tools to be used in the KECS project.

## Decision

### Runtime Environment
- **Docker**: Container runtime for local development and testing
- **Docker Compose**: Multi-container orchestration for local development
- **Kubernetes**: Production-grade container orchestration platform
- **Kind**: Kubernetes in Docker for local Kubernetes development and testing

### Development Language
- **Go 1.24+**: Primary development language, chosen for its performance, concurrency model, and strong ecosystem for cloud-native applications

### Libraries and Frameworks
- **HTTP Server**: 
  - `net/http`: Go's standard HTTP package
  - **Swagger/OpenAPI**: For API specification and code generation, enabling consistent API development and documentation

- **Testing**:
  - **Ginkgo**: BDD-style testing framework for Go, providing a rich testing vocabulary and parallel test execution

- **Command Line Interface**:
  - **Cobra**: CLI framework for Go, providing a structured approach to building command-line applications

- **Logging**:
  - **slog**: Go's standard structured logging package, providing a consistent logging interface

- **Database Access**:
  - **sqlc**: SQL compiler for Go, generating type-safe code from SQL

- **AWS Integration**:
  - **aws-sdk-go-v2**: AWS SDK for Go v2, providing a modern, idiomatic interface to AWS services

### Data Store
- **DuckDB**: Embedded analytical database, chosen for its performance and simplicity for local development and testing

### Development Tools
- **Tilt**: Development environment orchestration, providing fast feedback loops during development
- **TestContainers**: For running E2E tests against KECS instances in isolated containers

## Consequences

### Advantages
- Consistent development environment across the team
- Strong type safety with Go and sqlc
- Comprehensive testing capabilities with Ginkgo and TestContainers
- Streamlined API development with Swagger/OpenAPI
- Fast feedback loops with Tilt
- Simplified local development with Kind and Docker

### Challenges
- Learning curve for developers new to these tools and libraries
- Maintenance of OpenAPI specifications alongside code
- Integration complexity between Kubernetes and ECS-compatible APIs

## Alternatives Considered

### Programming Language
- **Rust**: Considered for its performance and safety guarantees but rejected due to the team's expertise in Go and Go's mature ecosystem for cloud-native applications
- **Java/Kotlin**: Considered for its widespread use in enterprise applications but rejected due to resource overhead and deployment complexity

### Data Store
- **PostgreSQL**: Considered for its robustness but rejected in favor of DuckDB's simplicity for local development
- **SQLite**: Considered for its embedded nature but rejected due to DuckDB's better performance for analytical workloads

### API Framework
- **gRPC**: Considered for its performance but rejected in favor of REST APIs with Swagger/OpenAPI for better compatibility with ECS API clients

## References

- [Go Programming Language](https://golang.org/)
- [Swagger/OpenAPI Specification](https://swagger.io/specification/)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)
- [Cobra CLI Framework](https://github.com/spf13/cobra)
- [Go slog Package](https://pkg.go.dev/log/slog)
- [sqlc](https://sqlc.dev/)
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- [DuckDB](https://duckdb.org/)
- [Tilt](https://tilt.dev/)
- [TestContainers](https://www.testcontainers.org/)
