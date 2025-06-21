# AWS CLI Compatibility Issue with AWS SDK Go v2 Migration

## Issue Description

After migrating from custom generated types to AWS SDK Go v2 types (commit `90a31b0`), AWS CLI stopped displaying output for KECS API responses, although the responses were correctly returned to the client.

### Symptoms
- `curl` commands worked correctly and displayed responses
- HTTP proxy tools (Proxyman) showed correct responses being sent
- AWS CLI received the responses but displayed nothing (0 bytes output)
- The issue affected both `--endpoint-url` and `--profile` usage

### Investigation Timeline
1. Confirmed KECS was returning correct responses via curl
2. Verified responses were being sent correctly via HTTP proxy
3. Discovered AWS CLI was receiving responses but not displaying them
4. Compared generated types with AWS SDK v2 types
5. Found field name casing differences

## Root Cause

The issue was caused by field name casing differences between AWS SDK Go v2 and AWS CLI expectations:

### AWS SDK Go v2 Behavior
- Uses `PascalCase` field names in struct definitions (e.g., `ClusterArns`)
- No JSON tags on the struct fields
- Marshals to JSON using the exact field names
- Accepts both `PascalCase` and `camelCase` in requests (case insensitive)

### AWS CLI (Python/boto3) Behavior
- Expects `camelCase` field names in responses (e.g., `clusterArns`)
- Strictly case sensitive when parsing responses
- Silently fails to display output if field names don't match expectations

### Example

Generated types (working):
```go
type ListClustersResponse struct {
    ClusterArns []string `json:"clusterArns"`  // Note: lowercase 'c'
    NextToken   *string  `json:"nextToken,omitempty"`
}
```

AWS SDK v2 types (not working with AWS CLI):
```go
type ListClustersOutput struct {
    ClusterArns []string  // No JSON tag, marshals as "ClusterArns"
    NextToken *string
    ResultMetadata middleware.Metadata
    // Has unexported fields
}
```

## Solution

Created a response cleaning function that:
1. Removes AWS SDK v2 specific fields (`ResultMetadata`)
2. Converts field names from `PascalCase` to `camelCase`
3. Removes nil fields to match AWS API behavior

```go
// cleanAWSResponse removes AWS SDK v2 specific fields that might interfere with AWS CLI
// and converts field names to match AWS API conventions (e.g., ClusterArns -> clusterArns)
func cleanAWSResponse(v interface{}) map[string]interface{} {
    // Convert the response to a map
    data, err := json.Marshal(v)
    if err != nil {
        return nil
    }
    
    var result map[string]interface{}
    if err := json.Unmarshal(data, &result); err != nil {
        return nil
    }
    
    // Remove ResultMetadata if it exists
    delete(result, "ResultMetadata")
    
    // Convert field names to match AWS API conventions
    // AWS SDK v2 uses PascalCase, but AWS API expects camelCase
    cleanedResult := make(map[string]interface{})
    for k, v := range result {
        if v != nil {
            // Convert first letter to lowercase for AWS API compatibility
            newKey := k
            if len(k) > 0 {
                newKey = strings.ToLower(k[:1]) + k[1:]
            }
            
            // Check if the value is a slice and if it's not nil/empty
            if reflect.TypeOf(v).Kind() == reflect.Slice {
                s := reflect.ValueOf(v)
                if s.Len() > 0 {
                    cleanedResult[newKey] = v
                }
            } else {
                cleanedResult[newKey] = v
            }
        }
    }
    
    return cleanedResult
}
```

## Testing Results

Before fix:
```bash
$ aws --endpoint-url http://localhost:8080 ecs list-clusters
# No output

$ curl -X POST http://localhost:8080/ -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" -d '{}'
{"ClusterArns":["arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster"],"NextToken":null,"ResultMetadata":{}}
```

After fix:
```bash
$ aws --endpoint-url http://localhost:8080 ecs list-clusters
{
    "clusterArns": [
        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster"
    ]
}

$ curl -X POST http://localhost:8080/ -H "X-Amz-Target: AmazonEC2ContainerServiceV20141113.ListClusters" -d '{}'
{"clusterArns":["arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster"]}
```

## Recommendations

1. **For Production Implementation**: Consider implementing a more robust solution that:
   - Uses custom JSON marshaling for AWS SDK v2 types
   - Adds proper JSON tags to wrapper structs
   - Handles nested objects and complex types correctly

2. **Alternative Approaches**:
   - Create wrapper types with proper JSON tags
   - Use a custom JSON encoder that handles field name transformation
   - Implement a middleware that transforms responses at the HTTP level

3. **Testing**: When migrating to AWS SDK v2, always test with:
   - AWS CLI
   - boto3/Python SDK
   - Other language SDKs
   - Direct HTTP/curl requests

## Lessons Learned

1. AWS SDK Go v2 doesn't include JSON tags by default, unlike the generated types
2. AWS CLI/boto3 are strict about field name casing in responses
3. HTTP proxies are invaluable for debugging API compatibility issues
4. Always test with actual AWS CLI, not just curl or unit tests