# Phase 1: Cluster Operations Tests

This directory contains comprehensive tests for ECS cluster operations using TestContainers and AWS CLI v2.

## Test Files

### 1. `cluster_basic_operations_test.go`
Tests fundamental cluster operations:
- **Create Cluster**: Default cluster, named clusters, special characters, idempotent creation
- **Describe Clusters**: By name, by ARN, multiple clusters, non-existent clusters
- **Delete Cluster**: Empty clusters, by name, by ARN, non-existent clusters
- **List Clusters**: Empty list, multiple clusters

### 2. `cluster_advanced_features_test.go`
Tests advanced cluster features:
- **Cluster Settings**: Container Insights configuration
- **Cluster Configuration**: Execute command configuration
- **Cluster Tags**: Add, remove, list tags
- **Capacity Providers**: FARGATE, FARGATE_SPOT configuration
- **Describe with Include**: SETTINGS, CONFIGURATIONS, TAGS

### 3. `cluster_pagination_test.go`
Tests pagination functionality:
- **Various Page Sizes**: maxResults=1, 10, 50, 100
- **Next Token Handling**: Token flow, invalid tokens
- **Pagination Consistency**: No duplicates, complete coverage
- **Large Scale Testing**: 150+ clusters (optional)

### 4. `cluster_error_scenarios_test.go`
Tests error handling:
- **Invalid Operations**: Long names, invalid characters, malformed ARNs
- **Resource Conflicts**: Delete with active services/tasks
- **Validation Errors**: Invalid settings, capacity providers
- **Missing Parameters**: Required fields validation

## Running the Tests

### Run all Phase 1 tests:
```bash
cd tests/scenarios
ginkgo -v ./phase1/...
```

### Run specific test file:
```bash
ginkgo -v ./phase1/cluster_basic_operations_test.go
```

### Run with specific focus:
```bash
ginkgo -v --focus="Create Cluster" ./phase1/...
```

### Skip large scale tests:
```bash
ginkgo -v --skip="Large Scale" ./phase1/...
```

## Test Utilities

Enhanced utilities in `utils/cluster_helpers.go`:
- `AssertClusterHasSettings()` - Verify cluster settings
- `AssertClusterHasConfiguration()` - Verify cluster configuration
- `AssertClusterHasTags()` - Verify cluster tags
- `AssertClusterHasCapacityProviders()` - Verify capacity providers
- `CreateClusterWithSettings()` - Create cluster with settings
- `CreateClusterWithTags()` - Create cluster with tags
- `ValidateClusterResponse()` - Comprehensive response validation
- `GetAllClustersWithPagination()` - Handle pagination automatically

## AWS CLI Integration

The tests use AWS CLI v2 for all operations. New operations added:
- `UpdateClusterSettings()` - Update cluster settings
- `UpdateCluster()` - Update cluster configuration
- `PutClusterCapacityProviders()` - Set capacity providers
- `DescribeClustersWithInclude()` - Describe with additional fields
- `ListClustersWithPagination()` - List with pagination support

## Test Coverage

Phase 1 provides comprehensive coverage of:
- ✅ All cluster CRUD operations
- ✅ Advanced features (settings, configuration, tags, capacity providers)
- ✅ Pagination logic
- ✅ Error handling and validation
- ✅ AWS ECS compatibility behaviors

## Notes

- Tests run serially to avoid resource conflicts
- Each test cleans up its resources using `DeferCleanup`
- Large scale tests (150+ clusters) can be skipped for faster runs
- All tests use unique cluster names with timestamps