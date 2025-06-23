package api

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

var _ = Describe("Service ECS API", func() {
	var (
		server           *Server
		ctx              context.Context
		mockStorage      *mocks.MockStorage
		mockServiceStore *mocks.MockServiceStore
		mockClusterStore *mocks.MockClusterStore
	)

	BeforeEach(func() {
		mockStorage = mocks.NewMockStorage()
		mockServiceStore = mocks.NewMockServiceStore()
		mockClusterStore = mocks.NewMockClusterStore()

		// Set stores on mock storage
		mockStorage.SetServiceStore(mockServiceStore)
		mockStorage.SetClusterStore(mockClusterStore)

		server = &Server{
			storage:     mockStorage,
			ecsAPI:      NewDefaultECSAPI(mockStorage),
		}
		ctx = context.Background()

		// Pre-populate with test data
		// Add a default cluster
		cluster := &storage.Cluster{
			Name:      "default",
			ARN:       "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
			Status:    "ACTIVE",
			Region:    "ap-northeast-1",
			AccountID: "123456789012",
		}
		err := mockClusterStore.Create(ctx, cluster)
		Expect(err).To(BeNil())

		// Add test services
		testService := &storage.Service{
			ID:                "service-1",
			ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:service/default/test-service",
			ServiceName:       "test-service",
			ClusterARN:        cluster.ARN,
			TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
			DesiredCount:      2,
			RunningCount:      2,
			PendingCount:      0,
			LaunchType:        "FARGATE",
			Status:            "ACTIVE",
			Namespace:         "test-namespace",
			Region:            "ap-northeast-1",
			AccountID:         "123456789012",
			CreatedAt:         time.Now().Add(-1 * time.Hour),
			UpdatedAt:         time.Now().Add(-30 * time.Minute),
		}
		err = mockServiceStore.Create(ctx, testService)
		Expect(err).To(BeNil())
	})

	Describe("ListServicesByNamespace", func() {
		Context("when listing services by namespace", func() {
			BeforeEach(func() {
				// Add more services with different namespaces
				service2 := &storage.Service{
					ID:          "service-2",
					ARN:         "arn:aws:ecs:ap-northeast-1:123456789012:service/default/test-service-2",
					ServiceName: "test-service-2",
					ClusterARN:  "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
					Namespace:   "test-namespace",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err := mockServiceStore.Create(ctx, service2)
				Expect(err).To(BeNil())

				service3 := &storage.Service{
					ID:          "service-3",
					ARN:         "arn:aws:ecs:ap-northeast-1:123456789012:service/default/other-service",
					ServiceName: "other-service",
					ClusterARN:  "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
					Namespace:   "other-namespace",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				err = mockServiceStore.Create(ctx, service3)
				Expect(err).To(BeNil())
			})

			It("should list services in specified namespace", func() {
				namespace := "test-namespace"
				req := &generated.ListServicesByNamespaceRequest{
					Namespace: namespace,
				}

				resp, err := server.ecsAPI.ListServicesByNamespace(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceArns).To(HaveLen(2))
				for _, arn := range resp.ServiceArns {
					Expect(arn).To(ContainSubstring("test-service"))
				}
			})

			It("should return empty list for non-existent namespace", func() {
				namespace := "non-existent"
				req := &generated.ListServicesByNamespaceRequest{
					Namespace: namespace,
				}

				resp, err := server.ecsAPI.ListServicesByNamespace(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceArns).To(BeEmpty())
			})

			It("should fail without namespace", func() {
				req := &generated.ListServicesByNamespaceRequest{}

				_, err := server.ecsAPI.ListServicesByNamespace(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("namespace is required"))
			})
		})
	})

	Describe("UpdateServicePrimaryTaskSet", func() {
		var mockTaskSetStore *mocks.MockTaskSetStore

		BeforeEach(func() {
			mockTaskSetStore = mocks.NewMockTaskSetStore()
			mockStorage.SetTaskSetStore(mockTaskSetStore)

			// Create a task set for testing
			taskSet := &storage.TaskSet{
				ID:                   "new-task-set",
				ARN:                  "arn:aws:ecs:ap-northeast-1:123456789012:task-set/default/test-service/new-task-set",
				ServiceARN:           "arn:aws:ecs:ap-northeast-1:123456789012:service/default/test-service",
				ClusterARN:           "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
				Status:               "ACTIVE",
				TaskDefinition:       "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
				LaunchType:           "FARGATE",
				Scale:                `{"value":100.0,"unit":"PERCENT"}`,
				StabilityStatus:      "STEADY_STATE",
				ComputedDesiredCount: 2,
				RunningCount:         2,
				PendingCount:         0,
				CreatedAt:            time.Now(),
				UpdatedAt:            time.Now(),
			}
			err := mockTaskSetStore.Create(ctx, taskSet)
			Expect(err).To(BeNil())
		})

		Context("when updating primary task set", func() {
			It("should update primary task set successfully", func() {
				serviceName := "test-service"
				taskSetId := "new-task-set"
				req := &generated.UpdateServicePrimaryTaskSetRequest{
					Service:        serviceName,
					PrimaryTaskSet: taskSetId,
				}

				resp, err := server.ecsAPI.UpdateServicePrimaryTaskSet(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskSet).NotTo(BeNil())
				Expect(*resp.TaskSet.Id).To(Equal(taskSetId))
				Expect(*resp.TaskSet.Status).To(Equal("ACTIVE"))
			})

			It("should fail without service name", func() {
				taskSetId := "new-task-set"
				req := &generated.UpdateServicePrimaryTaskSetRequest{
					PrimaryTaskSet: taskSetId,
				}

				_, err := server.ecsAPI.UpdateServicePrimaryTaskSet(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service is required"))
			})

			It("should fail without primary task set", func() {
				serviceName := "test-service"
				req := &generated.UpdateServicePrimaryTaskSetRequest{
					Service: serviceName,
				}

				_, err := server.ecsAPI.UpdateServicePrimaryTaskSet(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("primaryTaskSet is required"))
			})
		})
	})

	Describe("DescribeServiceDeployments", func() {
		Context("when describing service deployments", func() {
			It("should describe deployment successfully", func() {
				deploymentArn := "arn:aws:ecs:ap-northeast-1:123456789012:service-deployment/default/test-service/deployment-1"
				req := &generated.DescribeServiceDeploymentsRequest{
					ServiceDeploymentArns: []string{deploymentArn},
				}

				resp, err := server.ecsAPI.DescribeServiceDeployments(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceDeployments).To(HaveLen(1))
				Expect(resp.Failures).To(BeEmpty())

				deployment := resp.ServiceDeployments[0]
				Expect(*deployment.ServiceDeploymentArn).To(Equal(deploymentArn))
				Expect(*deployment.Status).To(Equal(generated.ServiceDeploymentStatusSUCCESSFUL))
			})

			It("should report failure for invalid ARN format", func() {
				deploymentArn := "invalid-arn"
				req := &generated.DescribeServiceDeploymentsRequest{
					ServiceDeploymentArns: []string{deploymentArn},
				}

				resp, err := server.ecsAPI.DescribeServiceDeployments(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceDeployments).To(BeEmpty())
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("INVALID_ARN"))
			})

			It("should fail without deployment ARNs", func() {
				req := &generated.DescribeServiceDeploymentsRequest{}

				_, err := server.ecsAPI.DescribeServiceDeployments(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("serviceDeploymentArns is required"))
			})
		})
	})

	Describe("DescribeServiceRevisions", func() {
		Context("when describing service revisions", func() {
			It("should describe revision successfully", func() {
				revisionArn := "arn:aws:ecs:ap-northeast-1:123456789012:service-revision/default/test-service/revision-1"
				req := &generated.DescribeServiceRevisionsRequest{
					ServiceRevisionArns: []string{revisionArn},
				}

				resp, err := server.ecsAPI.DescribeServiceRevisions(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceRevisions).To(HaveLen(1))
				Expect(resp.Failures).To(BeEmpty())

				revision := resp.ServiceRevisions[0]
				Expect(*revision.ServiceRevisionArn).To(Equal(revisionArn))
				Expect(*revision.TaskDefinition).To(ContainSubstring("nginx:1"))
			})

			It("should fail without revision ARNs", func() {
				req := &generated.DescribeServiceRevisionsRequest{}

				_, err := server.ecsAPI.DescribeServiceRevisions(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("serviceRevisionArns is required"))
			})
		})
	})

	Describe("ListServiceDeployments", func() {
		Context("when listing service deployments", func() {
			It("should list deployments for a service", func() {
				serviceName := "test-service"
				req := &generated.ListServiceDeploymentsRequest{
					Service: serviceName,
				}

				resp, err := server.ecsAPI.ListServiceDeployments(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceDeployments).To(HaveLen(2)) // Current and previous

				// Check current deployment
				current := resp.ServiceDeployments[0]
				Expect(*current.ServiceDeploymentArn).To(ContainSubstring("current"))
				Expect(*current.Status).To(Equal(generated.ServiceDeploymentStatusSUCCESSFUL))
			})

			It("should filter by status", func() {
				serviceName := "test-service"
				status := []generated.ServiceDeploymentStatus{generated.ServiceDeploymentStatusSUCCESSFUL}
				req := &generated.ListServiceDeploymentsRequest{
					Service: serviceName,
					Status:  status,
				}

				resp, err := server.ecsAPI.ListServiceDeployments(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceDeployments).To(HaveLen(2)) // Both are successful
			})

			It("should fail without service name", func() {
				req := &generated.ListServiceDeploymentsRequest{}

				_, err := server.ecsAPI.ListServiceDeployments(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service is required"))
			})
		})
	})

	Describe("StopServiceDeployment", func() {
		Context("when stopping a deployment", func() {
			It("should stop deployment successfully", func() {
				deploymentArn := "arn:aws:ecs:ap-northeast-1:123456789012:service-deployment/default/test-service/deployment-1"
				req := &generated.StopServiceDeploymentRequest{
					ServiceDeploymentArn: deploymentArn,
				}

				resp, err := server.ecsAPI.StopServiceDeployment(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(*resp.ServiceDeploymentArn).To(Equal(deploymentArn))
			})

			It("should fail with invalid ARN format", func() {
				deploymentArn := "invalid-arn"
				req := &generated.StopServiceDeploymentRequest{
					ServiceDeploymentArn: deploymentArn,
				}

				_, err := server.ecsAPI.StopServiceDeployment(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid deployment ARN format"))
			})

			It("should fail without deployment ARN", func() {
				req := &generated.StopServiceDeploymentRequest{}

				_, err := server.ecsAPI.StopServiceDeployment(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("serviceDeploymentArn is required"))
			})
		})
	})
})
