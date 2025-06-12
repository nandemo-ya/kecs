# Pointer Helper Package

This package provides helper functions for converting between pointer and non-pointer values, inspired by the AWS SDK v2 pointer helpers.

## Why Use Pointer Helpers?

The ECS API uses pointer types for optional fields to distinguish between:
- A field that is not set (nil)
- A field that is explicitly set to its zero value (e.g., "", 0, false)

Working with pointers in Go can be verbose, so this package provides convenient helper functions.

## Usage

### Creating Pointers

```go
import "github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"

// Instead of:
serviceName := "my-service"
req := &CreateServiceRequest{
    ServiceName: &serviceName,
}

// Use:
req := &CreateServiceRequest{
    ServiceName: ptr.String("my-service"),
}
```

### Dereferencing Pointers Safely

```go
// Instead of:
var name string
if response.ServiceName != nil {
    name = *response.ServiceName
} else {
    name = ""
}

// Use:
name := ptr.ToString(response.ServiceName)
```

## Available Functions

### Basic Types

| Function | Description |
|----------|-------------|
| `String(v string) *string` | Returns a pointer to the string value |
| `ToString(p *string) string` | Returns the value or "" if nil |
| `Bool(v bool) *bool` | Returns a pointer to the bool value |
| `ToBool(p *bool) bool` | Returns the value or false if nil |
| `Int32(v int32) *int32` | Returns a pointer to the int32 value |
| `ToInt32(p *int32) int32` | Returns the value or 0 if nil |
| `Int64(v int64) *int64` | Returns a pointer to the int64 value |
| `ToInt64(p *int64) int64` | Returns the value or 0 if nil |
| `Float64(v float64) *float64` | Returns a pointer to the float64 value |
| `ToFloat64(p *float64) float64` | Returns the value or 0 if nil |
| `Time(v time.Time) *time.Time` | Returns a pointer to the time value |
| `ToTime(p *time.Time) time.Time` | Returns the value or zero time if nil |

### Collection Types

| Function | Description |
|----------|-------------|
| `StringSlice(v []string) []*string` | Converts a string slice to a slice of string pointers |
| `ToStringSlice(p []*string) []string` | Converts a slice of string pointers to a string slice (skips nils) |
| `StringMap(v map[string]string) map[string]*string` | Converts a string map to a map with string pointer values |
| `ToStringMap(p map[string]*string) map[string]string` | Converts a map with string pointer values to a string map (skips nils) |

## Examples

### Creating a Service Request

```go
req := &generated.CreateServiceRequest{
    ServiceName:          ptr.String("web-service"),
    Cluster:              ptr.String("production"),
    TaskDefinition:       ptr.String("web-app:latest"),
    DesiredCount:         ptr.Int32(3),
    EnableECSManagedTags: ptr.Bool(true),
}
```

### Working with Responses

```go
// Safe dereferencing
if response.Service != nil {
    name := ptr.ToString(response.Service.ServiceName)
    count := ptr.ToInt32(response.Service.DesiredCount)
    
    fmt.Printf("Service %s has %d desired tasks\n", name, count)
}
```

### VPC Configuration

```go
vpcConfig := &generated.AwsVpcConfiguration{
    Subnets:        ptr.StringSlice([]string{"subnet-123", "subnet-456"}),
    SecurityGroups: ptr.StringSlice([]string{"sg-789"}),
    AssignPublicIp: ptr.String("ENABLED"),
}
```

## Best Practices

1. **Use for Optional Fields**: Use pointer helpers for fields that are optional in the API
2. **Nil Checks**: When reading responses, always check if the parent object is nil before dereferencing fields
3. **Zero Values**: Remember that `To*` functions return zero values for nil pointers
4. **Collections**: Use slice helpers when working with arrays of strings or other basic types