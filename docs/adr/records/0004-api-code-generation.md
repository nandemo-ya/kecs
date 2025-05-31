# ECS API Code Generation from Smithy Models

**Date:** 2025-01-31

## Status

Proposed

## Context

KECS aims to provide Amazon ECS compatible APIs. Currently, the API endpoints and data structures are manually implemented based on ECS documentation. This approach has several challenges:

1. **Maintenance Burden**: Keeping up with AWS ECS API updates requires manual tracking and implementation
2. **Accuracy**: Manual implementation may introduce inconsistencies with the official ECS API
3. **Completeness**: Ensuring all API operations and data structures are properly implemented is time-consuming
4. **Type Safety**: Manually maintaining request/response structures is error-prone

AWS uses Smithy IDL (Interface Definition Language) to define their service APIs. These definitions are publicly available in AWS SDK repositories and contain comprehensive API specifications.

## Decision

We will implement an automated code generation pipeline that:

1. **Sources API Definitions** from AWS SDK repositories (specifically `aws-sdk-go-v2`)
2. **Uses Smithy Models** as the source of truth for API specifications
3. **Generates Go Code** for:
   - API operation handlers
   - Request/Response data structures
   - Input validation logic
   - API documentation

The code generation will use the following approach:

### Source Model
- Primary source: `https://github.com/aws/aws-sdk-go-v2/blob/main/codegen/sdk-codegen/aws-models/ecs.json`
- Format: Smithy 2.0 JSON
- Protocol: `awsJson1.1`

### Generation Process
1. Fetch the latest ECS Smithy model
2. Parse the Smithy JSON to extract:
   - Service metadata
   - Operation definitions
   - Shape (type) definitions
   - Validation rules
3. Generate Go code using templates
4. Integrate with existing KECS architecture

### Generated Artifacts
- API handler interfaces in `internal/controlplane/api/generated/`
- Type definitions maintaining AWS ECS compatibility
- Validation logic based on Smithy constraints
- OpenAPI specification for additional tooling

## Consequences

### Positive
- **Accuracy**: Direct generation from official AWS models ensures API compatibility
- **Maintainability**: Updates to ECS API can be incorporated by regenerating code
- **Completeness**: All ECS operations and types will be available
- **Type Safety**: Generated code includes proper validation and type constraints
- **Documentation**: API documentation is generated from Smithy descriptions
- **Reduced Manual Work**: Developers focus on implementation logic rather than API contracts

### Negative
- **Build Complexity**: Adds a code generation step to the build process
- **Customization**: Generated code may need hooks for custom logic
- **Dependencies**: Requires smithy-go or custom Smithy parser
- **Learning Curve**: Team needs to understand Smithy models and generation process

## Alternatives Considered

### 1. Manual Implementation (Current Approach)
- **Pros**: Full control, no generation complexity
- **Cons**: High maintenance burden, prone to inconsistencies
- **Rejected**: Does not scale with ECS API evolution

### 2. OpenAPI-Based Generation
- **Pros**: Standard tooling, wide ecosystem support
- **Cons**: AWS doesn't provide official OpenAPI specs for ECS
- **Rejected**: Would require manual OpenAPI creation, defeating the purpose

### 3. Direct AWS SDK Usage
- **Pros**: Always up-to-date, no generation needed
- **Cons**: Includes AWS-specific logic, not suitable for local implementation
- **Rejected**: KECS needs to implement the server side, not client side

### 4. Protocol Buffer Definitions
- **Pros**: Efficient, good tooling
- **Cons**: ECS uses JSON protocol, not gRPC
- **Rejected**: Mismatched with ECS wire protocol

## Implementation Plan

1. Create a `cmd/codegen` tool for code generation
2. Add Makefile target: `make gen-api`
3. Store generated code in `internal/controlplane/api/generated/`
4. Keep manual implementations in `internal/controlplane/api/`
5. Use generated types and interfaces in manual implementations

## References

- [Smithy Homepage](https://smithy.io/)
- [AWS SDK Go v2 ECS Models](https://github.com/aws/aws-sdk-go-v2/tree/main/codegen/sdk-codegen/aws-models)
- [Smithy Go Code Generation](https://github.com/aws/smithy-go)
- [AWS ECS API Reference](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/)