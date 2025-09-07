# Task Set Examples

This directory contains examples demonstrating ECS Task Set functionality in KECS.

## Overview

Task Sets enable advanced deployment patterns in ECS, allowing you to:
- Run multiple versions of a task definition within a single service
- Perform blue-green deployments
- Implement canary deployments
- Control traffic distribution between different versions

## Examples

### 1. Blue-Green Deployment (`task-set-blue-green/`)
Demonstrates how to perform blue-green deployments using task sets.
- Seamless version switching
- Zero-downtime deployments
- Rollback capabilities

### 2. Load Balancer Integration (`task-set-load-balancer/`)
Shows how to use task sets with Elastic Load Balancer (ELB).
- Traffic distribution
- Health check integration
- Target group management

### 3. Service Discovery (`task-set-service-discovery/`)
Illustrates task sets with service discovery integration.
- DNS-based service discovery
- Route53 integration
- Multi-version service endpoints

## Test Script

The `test-task-sets.sh` script provides automated testing for task set functionality:

```bash
# Run all task set tests
./test-task-sets.sh

# Test specific functionality
./test-task-sets.sh blue-green
./test-task-sets.sh load-balancer
./test-task-sets.sh service-discovery
```

## Prerequisites

- KECS instance running
- AWS CLI configured
- Kubernetes cluster with KECS deployed

## Getting Started

1. Choose an example directory based on your use case
2. Review the README and configuration files in that directory
3. Deploy the example using the provided scripts or commands
4. Use the test script to validate the deployment

## Common Use Cases

- **Progressive Rollouts**: Gradually shift traffic from old to new version
- **A/B Testing**: Run multiple versions simultaneously for testing
- **Disaster Recovery**: Quick rollback to previous stable version
- **Multi-Region**: Different task sets for different regions

## Resources

- [AWS ECS Task Sets Documentation](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-sets.html)
- [KECS Documentation](../../docs/)