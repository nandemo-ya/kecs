# Phase 1: Cluster Operations Test Plan

## Objective

Comprehensively test all ECS cluster operations to ensure KECS provides full compatibility with AWS ECS cluster APIs.

## Scope

### In Scope
- All cluster CRUD operations (Create, Read, Update, Delete)
- Cluster settings and configuration
- Cluster tagging operations
- Capacity provider management
- Pagination for list operations
- Error handling and validation
- AWS ECS behavioral compatibility

### Out of Scope
- Performance testing (removed per request)
- Container instance operations (Phase 2)
- Service operations (Phase 2)
- Task operations (Phase 3)

## Test Categories

### 1. Basic Operations (cluster_basic_operations_test.go)

#### Create Cluster
- **Default Cluster**: Create without name (should create "default")
- **Named Cluster**: Create with specific name
- **Special Characters**: Handle hyphens, numbers in names
- **Idempotency**: Create same cluster twice (should succeed)

#### Describe Clusters
- **By Name**: Describe using cluster name
- **By ARN**: Describe using full ARN
- **Multiple Clusters**: Describe multiple in one request
- **Non-existent**: Handle cluster not found

#### Delete Cluster
- **Empty Cluster**: Delete cluster with no resources
- **By Name**: Delete using cluster name
- **By ARN**: Delete using full ARN
- **Non-existent**: Handle deletion of non-existent cluster

#### List Clusters
- **Empty List**: List when no clusters exist
- **With Clusters**: List after creating multiple clusters
- **ARN Format**: Verify proper ARN format in results

### 2. Advanced Features (cluster_advanced_features_test.go)

#### Cluster Settings
- **Container Insights**: Enable/disable container insights
- **Update Settings**: Modify individual settings
- **Describe with SETTINGS**: Include settings in describe response

#### Cluster Configuration
- **Execute Command Config**: Configure execute command settings
- **Update Configuration**: Modify cluster configuration
- **Service Connect Defaults**: Configure service connect

#### Cluster Tags
- **Add Tags**: Tag cluster with multiple key-value pairs
- **Remove Tags**: Remove specific tags
- **List Tags**: Retrieve all tags for a cluster
- **Describe with TAGS**: Include tags in describe response

#### Capacity Providers
- **Set Providers**: Configure FARGATE, FARGATE_SPOT
- **Default Strategy**: Set default capacity provider strategy
- **Update Strategy**: Modify existing strategy

### 3. Pagination (cluster_pagination_test.go)

#### Page Sizes
- **maxResults=1**: Single item per page
- **maxResults=10**: Small pages
- **maxResults=50**: Medium pages
- **maxResults=100**: Maximum page size

#### Token Handling
- **Next Token Flow**: Follow pagination through multiple pages
- **Invalid Token**: Handle invalid next token
- **Token Consistency**: Ensure tokens work correctly

#### Large Scale
- **150+ Clusters**: Create and paginate through large set
- **No Duplicates**: Verify no duplicate results
- **Complete Coverage**: Ensure all items are returned

### 4. Error Scenarios (cluster_error_scenarios_test.go)

#### Invalid Operations
- **Long Names**: Names exceeding 255 characters
- **Invalid Characters**: Special characters not allowed
- **Empty Names**: Missing required name
- **Malformed ARNs**: Invalid ARN formats

#### Resource Conflicts
- **Active Services**: Delete cluster with services
- **Running Tasks**: Delete cluster with tasks
- **Resource Dependencies**: Handle dependent resources

#### Validation Errors
- **Invalid Settings**: Unknown setting names/values
- **Invalid Providers**: Non-existent capacity providers
- **Invalid Strategy**: Negative weights/base values
- **Malformed JSON**: Invalid configuration format

## Test Implementation Details

### Test Environment
- **Container**: KECS running in TestContainers
- **Client**: AWS CLI v2 (no curl)
- **Region**: us-east-1 (default)
- **Credentials**: Dummy values for local testing

### Resource Naming
```go
clusterName := utils.GenerateTestName("test-cluster")
// Format: test-cluster-20060102-150405-123
```

### Cleanup Strategy
- Use `DeferCleanup` for automatic cleanup
- Clean up in reverse order of creation
- Handle cleanup failures gracefully

### Validation Approach
1. Verify operation success/failure
2. Validate response structure
3. Check resource state changes
4. Verify AWS compatibility

## Success Criteria

### Functional
- [x] All basic CRUD operations work correctly
- [x] Advanced features function as expected
- [x] Pagination handles all edge cases
- [x] Error scenarios return appropriate errors

### Compatibility
- [x] Response formats match AWS ECS
- [x] Error messages are consistent
- [x] Behavioral compatibility maintained
- [x] ARN formats are correct

### Quality
- [x] Tests are maintainable and clear
- [x] Comprehensive test coverage
- [x] Proper resource cleanup
- [x] Good error diagnostics

## Test Execution

### Prerequisites
- Docker installed and running
- AWS CLI v2 installed
- Go 1.21+ and Ginkgo installed
- Sufficient resources for TestContainers

### Running Tests
```bash
# All Phase 1 tests
cd tests/scenarios
ginkgo -v ./phase1/...

# Specific category
ginkgo -v ./phase1/cluster_basic_operations_test.go

# With coverage
ginkgo -v -cover ./phase1/...
```

### Expected Duration
- Basic Operations: ~2 minutes
- Advanced Features: ~3 minutes
- Pagination: ~2 minutes (5+ minutes with large scale)
- Error Scenarios: ~2 minutes
- **Total**: ~10-15 minutes

## Known Limitations

1. **AWS CLI Limitations**: Some operations may not be fully supported
2. **TestContainers**: Requires Docker daemon access
3. **Serial Execution**: Tests run serially to avoid conflicts
4. **Resource Limits**: Large scale tests may hit system limits

## Future Enhancements

1. **Metrics Collection**: Add test execution metrics
2. **Parallel Execution**: Optimize for parallel test runs
3. **Extended Validation**: Add more comprehensive checks
4. **Integration Points**: Test with other AWS services