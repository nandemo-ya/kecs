# Generated AWS Types Usage Guide

This document provides comprehensive guidance on using the automatically generated AWS API types in KECS. These types replace AWS SDK v2 dependencies and provide full AWS CLI compatibility.

## Overview

KECS automatically generates AWS API types from official Smithy JSON specifications, providing:

- **AWS CLI Compatibility**: JSON field names use camelCase as expected by AWS CLI
- **Type Safety**: Full Go type definitions with proper error handling
- **No AWS SDK Dependencies**: Self-contained types with no external dependencies
- **Union Type Support**: Properly structured union types for complex AWS APIs
- **Error Interface Implementation**: All error types implement Go's `error` interface

## Generated Services

The following AWS services have generated types available:

### Core Services
- **ECS** (`internal/controlplane/api/generated`) - Amazon Elastic Container Service
- **S3** (`internal/s3/generated`) - Amazon Simple Storage Service  
- **CloudWatch Logs** (`internal/cloudwatchlogs/generated`) - Amazon CloudWatch Logs
- **IAM** (`internal/iam/generated`) - AWS Identity and Access Management
- **Secrets Manager** (`internal/secretsmanager/generated`) - AWS Secrets Manager
- **Systems Manager** (`internal/ssm/generated`) - AWS Systems Manager
- **STS** (`internal/sts/generated`) - AWS Security Token Service

## Basic Usage

### Import Generated Types

```go
import (
    ecsapi "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
    s3api "github.com/nandemo-ya/kecs/controlplane/internal/s3/generated"
    iamapi "github.com/nandemo-ya/kecs/controlplane/internal/iam/generated"
)
```

### Create Request Structures

```go
// ECS CreateCluster request
createClusterReq := &ecsapi.CreateClusterRequest{
    ClusterName: "my-cluster",
    Tags: []ecsapi.Tag{
        {
            Key:   "Environment", 
            Value: "production",
        },
    },
}

// S3 GetObject request  
getObjectReq := &s3api.GetObjectRequest{
    Bucket: "my-bucket",
    Key:    "my-file.txt",
}
```

### JSON Marshaling (AWS CLI Compatible)

All generated types use camelCase JSON field names for AWS CLI compatibility:

```go
import "encoding/json"

// Create a task definition
taskDef := &ecsapi.RegisterTaskDefinitionRequest{
    Family: "my-app",
    ContainerDefinitions: []ecsapi.ContainerDefinition{
        {
            Name:      "web",
            Image:     "nginx:latest", 
            Memory:    512,
            Essential: true,
        },
    },
}

// Marshal to JSON (produces camelCase field names)
jsonData, err := json.Marshal(taskDef)
// Output: {"family":"my-app","containerDefinitions":[{"name":"web","image":"nginx:latest","memory":512,"essential":true}]}
```

## Working with Union Types

Union types represent AWS API structures that can contain one of several possible field types:

### S3 Analytics Filter Example

```go
// Create an analytics filter with prefix
filter := &s3api.AnalyticsFilter{
    Prefix: stringPtr("logs/"),
}

// Or with tag
filter = &s3api.AnalyticsFilter{
    Tag: &s3api.Tag{
        Key:   "Environment",
        Value: "production", 
    },
}

// Or with complex AND operation
filter = &s3api.AnalyticsFilter{
    And: &s3api.AnalyticsAndOperator{
        Prefix: stringPtr("logs/"),
        Tags: []s3api.Tag{
            {Key: "Environment", Value: "production"},
            {Key: "Application", Value: "web"},
        },
    },
}
```

### CloudWatch Logs Response Stream Example

```go
// Handle different types of events in a Live Tail stream
responseStream := &cloudwatchlogsapi.StartLiveTailResponseStream{
    SessionStart: &cloudwatchlogsapi.LiveTailSessionStart{
        // Session initialization data
    },
}

// Or handle session updates
responseStream = &cloudwatchlogsapi.StartLiveTailResponseStream{
    SessionUpdate: &cloudwatchlogsapi.LiveTailSessionUpdate{
        // Log events data
    },
}
```

## Error Handling

All AWS error types implement Go's `error` interface with additional AWS-specific methods:

### Basic Error Handling

```go
// S3 NoSuchUpload error
err := s3api.NoSuchUpload{}
fmt.Printf("Error: %v\n", err)
// Output: "Error: NoSuchUpload: AWS client error (HTTP 404)"

// Use as standard Go error
var e error = err
fmt.Printf("Standard error: %v\n", e)
```

### AWS Error Methods

```go
// Check error details
if awsErr, ok := err.(interface{
    ErrorCode() string
    ErrorFault() string  
}); ok {
    fmt.Printf("AWS Error Code: %s\n", awsErr.ErrorCode())
    fmt.Printf("Error Type: %s\n", awsErr.ErrorFault()) // "client" or "server"
}
```

### Secrets Manager Error Examples

```go
// Handle different error types
switch err := someSecretsManagerCall().(type) {
case secretsmanagerapi.DecryptionFailure:
    fmt.Printf("Decryption failed: %v\n", err)
case secretsmanagerapi.InternalServiceError:
    fmt.Printf("AWS service error: %v\n", err) 
case secretsmanagerapi.ResourceNotFoundException:
    fmt.Printf("Resource not found: %v\n", err)
default:
    fmt.Printf("Unknown error: %v\n", err)
}
```

## Pointer Utilities

Many AWS API fields are optional and require pointers. Helper functions make this easier:

```go
// Utility functions for creating pointers
func stringPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
func boolPtr(b bool) *bool { return &b }

// Usage example
taskDefinition := &ecsapi.RegisterTaskDefinitionRequest{
    Family:                   "my-app",
    Cpu:                     stringPtr("256"),
    Memory:                  stringPtr("512"), 
    RequiresCompatibilities: []ecsapi.Compatibility{
        ecsapi.CompatibilityFargate,
    },
    NetworkMode: ecsapi.NetworkModeAwsvpc,
    ContainerDefinitions: []ecsapi.ContainerDefinition{
        {
            Name:      "app",
            Image:     "my-app:latest",
            Essential: boolPtr(true),
            Memory:    int32Ptr(512),
            PortMappings: []ecsapi.PortMapping{
                {
                    ContainerPort: int32Ptr(8080),
                    Protocol:      ecsapi.TransportProtocolTcp,
                },
            },
        },
    },
}
```

## Enum Usage

Enums are generated as string types with constants:

### ECS Launch Types

```go
// Using ECS launch type enums
service := &ecsapi.CreateServiceRequest{
    ServiceName: "my-service",
    LaunchType:  ecsapi.LaunchTypeFargate, // or ecsapi.LaunchTypeEc2
}
```

### S3 Storage Classes

```go
// Using S3 storage class enums
putRequest := &s3api.PutObjectRequest{
    Bucket:       "my-bucket",
    Key:          "my-file.txt",
    StorageClass: s3api.StorageClassStandardIa,
}
```

## Advanced Usage Patterns

### Building Complex Requests

```go
// Complex ECS service creation with all options
serviceRequest := &ecsapi.CreateServiceRequest{
    ServiceName:    "web-service",
    Cluster:        "production-cluster", 
    TaskDefinition: "web-app:5",
    DesiredCount:   int32Ptr(3),
    LaunchType:     ecsapi.LaunchTypeFargate,
    
    NetworkConfiguration: &ecsapi.NetworkConfiguration{
        AwsvpcConfiguration: &ecsapi.AwsVpcConfiguration{
            Subnets:        []string{"subnet-123", "subnet-456"},
            SecurityGroups: []string{"sg-789"},
            AssignPublicIp: ecsapi.AssignPublicIpEnabled,
        },
    },
    
    LoadBalancers: []ecsapi.LoadBalancer{
        {
            TargetGroupArn: stringPtr("arn:aws:elasticloadbalancing:..."),
            ContainerName:  stringPtr("web"),
            ContainerPort:  int32Ptr(8080),
        },
    },
    
    ServiceRegistries: []ecsapi.ServiceRegistry{
        {
            RegistryArn:   stringPtr("arn:aws:servicediscovery:..."),
            ContainerName: stringPtr("web"),
            ContainerPort: int32Ptr(8080),
        },
    },
    
    Tags: []ecsapi.Tag{
        {Key: "Environment", Value: "production"},
        {Key: "Application", Value: "web"},
    },
    
    EnableExecuteCommand: boolPtr(true),
}
```

### Working with Complex Container Definitions

```go
containerDef := &ecsapi.ContainerDefinition{
    Name:      "web-server",
    Image:     "nginx:1.21",
    Essential: boolPtr(true),
    Memory:    int32Ptr(512),
    
    PortMappings: []ecsapi.PortMapping{
        {
            ContainerPort: int32Ptr(80),
            Protocol:      ecsapi.TransportProtocolTcp,
        },
    },
    
    Environment: []ecsapi.KeyValuePair{
        {Name: stringPtr("ENV"), Value: stringPtr("production")},
        {Name: stringPtr("DEBUG"), Value: stringPtr("false")},
    },
    
    Secrets: []ecsapi.Secret{
        {
            Name:      stringPtr("DB_PASSWORD"),
            ValueFrom: stringPtr("arn:aws:secretsmanager:..."),
        },
    },
    
    LogConfiguration: &ecsapi.LogConfiguration{
        LogDriver: ecsapi.LogDriverAwslogs,
        Options: map[string]string{
            "awslogs-group":  "/ecs/web-service",
            "awslogs-region": "us-east-1",
        },
    },
    
    HealthCheck: &ecsapi.HealthCheck{
        Command: []string{"CMD-SHELL", "curl -f http://localhost:80/ || exit 1"},
        Interval: int32Ptr(30),
        Timeout:  int32Ptr(5),
        Retries:  int32Ptr(3),
    },
}
```

## Migration from AWS SDK v2

When migrating from AWS SDK v2 types to generated types:

### Type Name Changes

| AWS SDK v2 | Generated Types |
|------------|-----------------|
| `ecs.CreateClusterInput` | `ecsapi.CreateClusterRequest` |
| `ecs.CreateClusterOutput` | `ecsapi.CreateClusterOutput` |
| `s3.GetObjectInput` | `s3api.GetObjectRequest` |
| `s3.GetObjectOutput` | `s3api.GetObjectOutput` |

### Import Changes

```go
// Before (AWS SDK v2)
import (
    "github.com/aws/aws-sdk-go-v2/service/ecs"
    "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// After (Generated types)
import (
    ecsapi "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)
```

### Function Signature Changes

```go
// Before (AWS SDK v2)
func CreateCluster(ctx context.Context, req *ecs.CreateClusterInput) (*ecs.CreateClusterOutput, error)

// After (Generated types)  
func CreateCluster(ctx context.Context, req *ecsapi.CreateClusterRequest) (*ecsapi.CreateClusterOutput, error)
```

## Best Practices

### 1. Use Type-Safe Constants

Always use generated enum constants instead of string literals:

```go
// Good
service.LaunchType = ecsapi.LaunchTypeFargate

// Avoid
service.LaunchType = "FARGATE"
```

### 2. Handle Errors Properly

Take advantage of typed error handling:

```go
if err != nil {
    if awsErr, ok := err.(interface{ ErrorCode() string }); ok {
        switch awsErr.ErrorCode() {
        case "ClusterNotFoundException":
            // Handle missing cluster
        case "InvalidParameterException": 
            // Handle invalid parameters
        default:
            // Handle other errors
        }
    }
}
```

### 3. Use Helper Functions for Pointers

Create utility functions to avoid repetitive pointer creation:

```go
func ptr[T any](v T) *T { return &v }

// Usage
taskDef := &ecsapi.RegisterTaskDefinitionRequest{
    Family: "my-app",
    Cpu:    ptr("256"),
    Memory: ptr("512"),
    ContainerDefinitions: []ecsapi.ContainerDefinition{
        {
            Name:      "app",
            Essential: ptr(true),
            Memory:    ptr(int32(512)),
        },
    },
}
```

### 4. Validate Required Fields

Check for required fields before making API calls:

```go
func validateCreateServiceRequest(req *ecsapi.CreateServiceRequest) error {
    if req.ServiceName == "" {
        return fmt.Errorf("ServiceName is required")
    }
    if req.TaskDefinition == "" {
        return fmt.Errorf("TaskDefinition is required")  
    }
    return nil
}
```

## Troubleshooting

### Common Issues

1. **Missing Pointer Fields**: Remember that optional fields require pointers
2. **Enum Values**: Use generated constants, not string literals
3. **JSON Field Names**: Generated types use camelCase for AWS CLI compatibility
4. **Union Types**: Only set one field per union type instance

### Debugging JSON Output

To verify JSON output matches AWS CLI expectations:

```go
data, _ := json.MarshalIndent(request, "", "  ")
fmt.Printf("Request JSON:\n%s\n", data)
```

This should produce camelCase field names that match AWS CLI requirements.