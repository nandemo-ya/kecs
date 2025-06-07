package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

// TestDeleteServiceWithStorage tests the DeleteService API implementation
func TestDeleteServiceWithStorage(t *testing.T) {
	// Initialize in-memory storage for testing
	dbStorage, err := duckdb.NewDuckDBStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer dbStorage.Close()

	ctx := context.Background()
	
	// Initialize database schema
	if err := dbStorage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create server
	server := &Server{
		storage:     dbStorage,
		kindManager: kubernetes.NewKindManager(),
		region:      "us-east-1",
		accountID:   "123456789012",
	}

	// Create test cluster
	cluster := &storage.Cluster{
		Name:            "test-cluster",
		ARN:             fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/test-cluster", server.region, server.accountID),
		Status:          "ACTIVE",
		Region:          server.region,
		AccountID:       server.accountID,
		KindClusterName: "kecs-test-cluster",
	}
	if err := dbStorage.ClusterStore().Create(ctx, cluster); err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}

	// Create test task definition
	taskDef := &storage.TaskDefinition{
		Family:               "test-task",
		Revision:             1,
		ARN:                  fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/test-task:1", server.region, server.accountID),
		ContainerDefinitions: `[{"name":"test-container","image":"nginx:latest"}]`,
		Status:               "ACTIVE",
		Region:               server.region,
		AccountID:            server.accountID,
	}
	if _, err := dbStorage.TaskDefinitionStore().Register(ctx, taskDef); err != nil {
		t.Fatalf("Failed to register task definition: %v", err)
	}

	// Create test service
	service := &storage.Service{
		ServiceName:       "test-service",
		ARN:               fmt.Sprintf("arn:aws:ecs:%s:%s:service/test-cluster/test-service", server.region, server.accountID),
		ClusterARN:        cluster.ARN,
		TaskDefinitionARN: taskDef.ARN,
		DesiredCount:      2,
		RunningCount:      2,
		PendingCount:      0,
		Status:            "ACTIVE",
		LaunchType:        "FARGATE",
		Region:            server.region,
		AccountID:         server.accountID,
		DeploymentName:    "ecs-service-test-service",
		Namespace:         "test-cluster-us-east-1",
	}
	if err := dbStorage.ServiceStore().Create(ctx, service); err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	t.Run("DeleteService without force should fail when desired count > 0", func(t *testing.T) {
		req := DeleteServiceRequest{
			Cluster: "test-cluster",
			Service: "test-service",
			Force:   false,
		}

		_, err := server.DeleteServiceWithStorage(ctx, req)
		if err == nil {
			t.Error("Expected error when deleting service with desired count > 0 without force")
		}
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("desired count of 0")) {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("DeleteService with force should succeed", func(t *testing.T) {
		req := DeleteServiceRequest{
			Cluster: "test-cluster",
			Service: "test-service",
			Force:   true,
		}

		resp, err := server.DeleteServiceWithStorage(ctx, req)
		if err != nil {
			t.Fatalf("Failed to delete service with force: %v", err)
		}

		// Verify response
		if resp.Service.ServiceName != "test-service" {
			t.Errorf("Expected service name 'test-service', got '%s'", resp.Service.ServiceName)
		}
		if resp.Service.Status != "DRAINING" {
			t.Errorf("Expected status 'DRAINING', got '%s'", resp.Service.Status)
		}
		if resp.Service.DesiredCount != 0 {
			t.Errorf("Expected desired count 0, got %d", resp.Service.DesiredCount)
		}

		// Verify service is deleted from storage
		_, err = dbStorage.ServiceStore().Get(ctx, cluster.ARN, "test-service")
		if err == nil {
			t.Error("Expected service to be deleted from storage")
		}
	})

	t.Run("DeleteService non-existent service should fail", func(t *testing.T) {
		req := DeleteServiceRequest{
			Cluster: "test-cluster",
			Service: "non-existent-service",
			Force:   true,
		}

		_, err := server.DeleteServiceWithStorage(ctx, req)
		if err == nil {
			t.Error("Expected error when deleting non-existent service")
		}
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("service not found")) {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("DeleteService non-existent cluster should fail", func(t *testing.T) {
		req := DeleteServiceRequest{
			Cluster: "non-existent-cluster",
			Service: "test-service",
			Force:   true,
		}

		_, err := server.DeleteServiceWithStorage(ctx, req)
		if err == nil {
			t.Error("Expected error when deleting service from non-existent cluster")
		}
		if err != nil && !bytes.Contains([]byte(err.Error()), []byte("cluster not found")) {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

// TestDeleteServiceAPIEndpoint tests the HTTP endpoint for DeleteService
func TestDeleteServiceAPIEndpoint(t *testing.T) {
	// Initialize in-memory storage for testing
	dbStorage, err := duckdb.NewDuckDBStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer dbStorage.Close()

	ctx := context.Background()
	
	// Initialize database schema
	if err := dbStorage.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize storage: %v", err)
	}

	// Create server
	server := &Server{
		storage:     dbStorage,
		kindManager: kubernetes.NewKindManager(),
		region:      "us-east-1",
		accountID:   "123456789012",
	}

	// Setup test data
	cluster := &storage.Cluster{
		Name:            "default",
		ARN:             fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/default", server.region, server.accountID),
		Status:          "ACTIVE",
		Region:          server.region,
		AccountID:       server.accountID,
		KindClusterName: "kecs-default",
	}
	if err := dbStorage.ClusterStore().Create(ctx, cluster); err != nil {
		t.Fatalf("Failed to create cluster: %v", err)
	}

	taskDef := &storage.TaskDefinition{
		Family:               "api-test-task",
		Revision:             1,
		ARN:                  fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/api-test-task:1", server.region, server.accountID),
		ContainerDefinitions: `[{"name":"test-container","image":"nginx:latest"}]`,
		Status:               "ACTIVE",
		Region:               server.region,
		AccountID:            server.accountID,
	}
	if _, err := dbStorage.TaskDefinitionStore().Register(ctx, taskDef); err != nil {
		t.Fatalf("Failed to register task definition: %v", err)
	}

	service := &storage.Service{
		ServiceName:       "api-test-service",
		ARN:               fmt.Sprintf("arn:aws:ecs:%s:%s:service/default/api-test-service", server.region, server.accountID),
		ClusterARN:        cluster.ARN,
		TaskDefinitionARN: taskDef.ARN,
		DesiredCount:      0, // Set to 0 for easy deletion
		RunningCount:      0,
		PendingCount:      0,
		Status:            "ACTIVE",
		LaunchType:        "FARGATE",
		Region:            server.region,
		AccountID:         server.accountID,
		DeploymentName:    "ecs-service-api-test-service",
		Namespace:         "default-us-east-1",
	}
	if err := dbStorage.ServiceStore().Create(ctx, service); err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}

	// Test the API endpoint
	t.Run("DeleteService via API endpoint", func(t *testing.T) {
		req := DeleteServiceRequest{
			Service: "api-test-service",
		}
		
		body, _ := json.Marshal(req)
		httpReq := httptest.NewRequest("POST", "/", bytes.NewReader(body))
		httpReq.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.DeleteService")
		httpReq.Header.Set("Content-Type", "application/x-amz-json-1.1")
		
		w := httptest.NewRecorder()
		server.handleECSRequest(w, httpReq)
		
		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", w.Code)
			t.Logf("Response body: %s", w.Body.String())
		}
		
		var resp DeleteServiceResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		
		if resp.Service.ServiceName != "api-test-service" {
			t.Errorf("Expected service name 'api-test-service', got '%s'", resp.Service.ServiceName)
		}
		if resp.Service.Status != "DRAINING" {
			t.Errorf("Expected status 'DRAINING', got '%s'", resp.Service.Status)
		}
	})
}