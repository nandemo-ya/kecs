// Package phase1 provides comprehensive scenario tests for ECS cluster operations.
//
// Overview
//
// Phase 1 tests focus on validating all aspects of ECS cluster management including
// basic CRUD operations, advanced features, and error handling.
// These tests use TestContainers to run KECS in isolation and AWS CLI v2 for
// all API interactions.
//
// Test Organization
//
// The tests are organized into three main categories:
//
//   - Basic Operations: Fundamental cluster CRUD operations
//   - Advanced Features: Settings, configuration, tags, and capacity providers
//   - Error Scenarios: Invalid operations, validation errors, and edge cases
//
// Running Tests
//
// Run all Phase 1 tests:
//
//	cd tests/scenarios
//	ginkgo -v ./phase1/...
//
// Run specific test file:
//
//	ginkgo -v ./phase1/cluster_basic_operations_test.go
//
// Run with focus on specific tests:
//
//	ginkgo -v --focus="Create Cluster" ./phase1/...
//
// Skip large scale tests:
//
//	ginkgo -v --skip="Large Scale" ./phase1/...
//
// Test Structure
//
// Each test file follows a consistent structure:
//   - BeforeEach: Start KECS container and initialize AWS CLI client
//   - Test scenarios: Organized by Describe/Context/It blocks
//   - DeferCleanup: Automatic resource cleanup after each test
//   - Serial execution: Tests run serially to avoid conflicts
//
// AWS CLI Integration
//
// All tests use AWS CLI v2 through the AWSCLIClient. The client has been
// extended with additional operations to support advanced cluster features:
//   - UpdateClusterSettings
//   - UpdateCluster
//   - PutClusterCapacityProviders
//   - DescribeClustersWithInclude
//
// Test Data
//
// Tests use unique resource names with timestamps to avoid conflicts:
//
//	clusterName := utils.GenerateTestName("test-cluster")
//	// Results in: test-cluster-20060102-150405-123
//
// Validation
//
// Comprehensive validation is performed for all operations:
//   - Response structure validation
//   - ARN format verification
//   - Resource state verification
//   - Error message validation
//
// Coverage Goals
//
// Phase 1 aims for 100% coverage of:
//   - All cluster API operations
//   - All request/response fields
//   - All error conditions
//   - AWS ECS compatibility behaviors
//
package phase1