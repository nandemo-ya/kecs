package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("ServiceHandler", func() {
	var (
		dbStorage *duckdb.DuckDBStorage
		server    *Server
		ctx       context.Context
		cluster   *storage.Cluster
		taskDef   *storage.TaskDefinition
	)

	BeforeEach(func() {
		var err error
		// Initialize in-memory storage for testing
		dbStorage, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).NotTo(HaveOccurred())

		ctx = context.Background()

		// Initialize database schema
		err = dbStorage.Initialize(ctx)
		Expect(err).NotTo(HaveOccurred())

		// Create server
		server = &Server{
			storage:     dbStorage,
			kindManager: kubernetes.NewKindManager(),
			region:      "us-east-1",
			accountID:   "123456789012",
		}

		// Create test cluster
		cluster = &storage.Cluster{
			Name:            "test-cluster",
			ARN:             fmt.Sprintf("arn:aws:ecs:%s:%s:cluster/test-cluster", server.region, server.accountID),
			Status:          "ACTIVE",
			Region:          server.region,
			AccountID:       server.accountID,
			KindClusterName: "kecs-test-cluster",
		}
		err = dbStorage.ClusterStore().Create(ctx, cluster)
		Expect(err).NotTo(HaveOccurred())

		// Create test task definition
		taskDef = &storage.TaskDefinition{
			Family:               "test-task",
			Revision:             1,
			ARN:                  fmt.Sprintf("arn:aws:ecs:%s:%s:task-definition/test-task:1", server.region, server.accountID),
			ContainerDefinitions: `[{"name":"test-container","image":"nginx:latest"}]`,
			Status:               "ACTIVE",
			Region:               server.region,
			AccountID:            server.accountID,
		}
		_, err = dbStorage.TaskDefinitionStore().Register(ctx, taskDef)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if dbStorage != nil {
			dbStorage.Close()
		}
	})

	Describe("DeleteServiceWithStorage", func() {
		var service *storage.Service

		BeforeEach(func() {
			// Create test service
			service = &storage.Service{
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
			err := dbStorage.ServiceStore().Create(ctx, service)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when deleting a service without force", func() {
			It("should fail when desired count > 0", func() {
				req := DeleteServiceRequest{
					Cluster: "test-cluster",
					Service: "test-service",
					Force:   false,
				}

				_, err := server.DeleteServiceWithStorage(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("desired count of 0"))
			})
		})

		Context("when deleting a service with force", func() {
			It("should succeed", func() {
				req := DeleteServiceRequest{
					Cluster: "test-cluster",
					Service: "test-service",
					Force:   true,
				}

				resp, err := server.DeleteServiceWithStorage(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify response
				Expect(resp.Service.ServiceName).To(Equal("test-service"))
				Expect(resp.Service.Status).To(Equal("DRAINING"))
				Expect(resp.Service.DesiredCount).To(Equal(int(0)))

				// Verify service is deleted from storage
				_, err = dbStorage.ServiceStore().Get(ctx, cluster.ARN, "test-service")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when deleting a non-existent service", func() {
			It("should fail", func() {
				req := DeleteServiceRequest{
					Cluster: "test-cluster",
					Service: "non-existent-service",
					Force:   true,
				}

				_, err := server.DeleteServiceWithStorage(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service not found"))
			})
		})

		Context("when deleting from a non-existent cluster", func() {
			It("should fail", func() {
				req := DeleteServiceRequest{
					Cluster: "non-existent-cluster",
					Service: "test-service",
					Force:   true,
				}

				_, err := server.DeleteServiceWithStorage(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("cluster not found"))
			})
		})
	})

	Describe("DeleteService API Endpoint", func() {
		var service *storage.Service

		BeforeEach(func() {
			// Create test service with desired count 0 for easy deletion
			service = &storage.Service{
				ServiceName:       "api-test-service",
				ARN:               fmt.Sprintf("arn:aws:ecs:%s:%s:service/test-cluster/api-test-service", server.region, server.accountID),
				ClusterARN:        cluster.ARN,
				TaskDefinitionARN: taskDef.ARN,
				DesiredCount:      0,
				RunningCount:      0,
				PendingCount:      0,
				Status:            "ACTIVE",
				LaunchType:        "FARGATE",
				Region:            server.region,
				AccountID:         server.accountID,
				DeploymentName:    "ecs-service-api-test-service",
				Namespace:         "test-cluster-us-east-1",
			}
			err := dbStorage.ServiceStore().Create(ctx, service)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should delete service via API endpoint", func() {
			req := DeleteServiceRequest{
				Cluster: "test-cluster",
				Service: "api-test-service",
			}

			body, _ := json.Marshal(req)
			httpReq := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			httpReq.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.DeleteService")
			httpReq.Header.Set("Content-Type", "application/x-amz-json-1.1")

			w := httptest.NewRecorder()
			server.handleECSRequest(w, httpReq)

			// Check response
			Expect(w.Code).To(Equal(http.StatusOK))

			var resp DeleteServiceResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.Service.ServiceName).To(Equal("api-test-service"))
			Expect(resp.Service.Status).To(Equal("DRAINING"))
		})
	})
})