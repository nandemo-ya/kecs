# Advanced Networking Configuration

This document describes the advanced networking features implemented in KECS for ECS compatibility.

## Overview

KECS supports all ECS network modes and advanced networking configurations, mapping ECS networking concepts to Kubernetes networking primitives.

## Network Modes

### awsvpc Mode
- Each task gets its own network interface (ENI)
- Supports security groups and subnet configuration
- Maps to Kubernetes pods with specific network annotations
- Compatible with AWS VPC networking features via LocalStack

### bridge Mode
- Standard Docker bridge networking
- Dynamic port mapping support
- Default mode for backward compatibility

### host Mode
- Uses host network namespace
- Maps to Kubernetes `hostNetwork: true`
- No network isolation between containers and host

### none Mode
- No networking enabled
- Useful for batch jobs that don't require network access

## Implementation Details

### Core Types (`internal/types/networking.go`)

```go
type NetworkConfiguration struct {
    AwsvpcConfiguration *AwsvpcConfiguration `json:"awsvpcConfiguration,omitempty"`
}

type AwsvpcConfiguration struct {
    Subnets        []string       `json:"subnets"`
    SecurityGroups []string       `json:"securityGroups,omitempty"`
    AssignPublicIp AssignPublicIp `json:"assignPublicIp,omitempty"`
}
```

### Network Converter (`internal/converters/network_converter.go`)

Converts ECS network configurations to Kubernetes resources:
- Transforms NetworkConfiguration to pod annotations
- Maps security groups for future NetworkPolicy integration
- Handles subnet and VPC configurations
- Manages load balancer and service registry settings

### Kubernetes Annotations

Network configurations are stored as pod annotations:
- `ecs.amazonaws.com/network-mode`: Network mode (awsvpc, bridge, host, none)
- `ecs.amazonaws.com/subnets`: Comma-separated subnet IDs
- `ecs.amazonaws.com/security-groups`: Comma-separated security group IDs
- `ecs.amazonaws.com/assign-public-ip`: Public IP assignment (ENABLED/DISABLED)

## Service Integration

### CreateService API
- Accepts NetworkConfiguration in request
- Stores configuration with service definition
- Applies network settings to all tasks in the service

### RunTask API
- Accepts NetworkConfiguration for ad-hoc tasks
- Creates appropriate network interfaces
- Populates task details with network information

## Task Network Details

For tasks running in awsvpc mode:
- ENI (Elastic Network Interface) is created
- Private IP addresses are assigned
- Network bindings are populated
- Container network interfaces are tracked

## LocalStack Integration

The implementation is designed to work seamlessly with LocalStack:
- VPC and subnet IDs are preserved in annotations
- Security groups can be validated against LocalStack
- Network interfaces can be simulated
- Compatible with LocalStack's EC2/VPC services

## Load Balancer Support

### Application Load Balancer (ALB)
- Target group registration via annotations
- Health check configuration support
- Already implemented in previous phases

### Network Load Balancer (NLB)
- Similar to ALB but with TCP/UDP support
- Target group ARN stored in service metadata
- Container port mapping preserved

## Security Groups

While Kubernetes doesn't have a direct equivalent to AWS Security Groups:
- Security group IDs are stored in annotations
- Future integration with Kubernetes NetworkPolicy planned
- LocalStack can validate security group rules

## Testing

### Unit Tests
- `internal/types/networking_test.go`: Type validation tests
- `internal/converters/network_converter_test.go`: Converter logic tests

### Integration Tests
- `internal/integrations/networking/awsvpc_test.go`: awsvpc mode integration
- Tests task creation with different network modes
- Validates network interface creation

## Usage Examples

### Service with awsvpc Mode

```json
{
  "serviceName": "web-service",
  "taskDefinition": "web-task:1",
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-12345", "subnet-67890"],
      "securityGroups": ["sg-web"],
      "assignPublicIp": "ENABLED"
    }
  }
}
```

### Task with bridge Mode

```json
{
  "taskDefinition": "worker-task:1",
  "networkMode": "bridge",
  "count": 1
}
```

## Future Enhancements

1. **NetworkPolicy Integration**: Map AWS Security Groups to Kubernetes NetworkPolicies
2. **Service Mesh Support**: Integration with Istio/Linkerd for advanced networking
3. **IPv6 Support**: Dual-stack networking configuration
4. **Custom DNS Configuration**: Support for custom DNS servers and search domains
5. **Network Performance Metrics**: Integration with monitoring systems

## Troubleshooting

### Common Issues

1. **Task fails to get IP address**
   - Check subnet configuration
   - Verify Kubernetes cluster networking
   - Ensure CNI plugin supports multiple interfaces

2. **Security groups not enforced**
   - Currently stored as metadata only
   - NetworkPolicy integration pending

3. **Port conflicts in bridge mode**
   - Dynamic port allocation handles this
   - Check for host port conflicts

## References

- [ECS Task Networking](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking.html)
- [awsvpc Network Mode](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking-awsvpc.html)
- [Kubernetes Networking](https://kubernetes.io/docs/concepts/cluster-administration/networking/)