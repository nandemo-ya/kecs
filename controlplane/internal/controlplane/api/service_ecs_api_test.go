package api

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockServiceStore for service operations
type MockServiceStore struct {
	storage *MockStorage
	services map[string]*storage.Service
}

func NewMockServiceStore(mockStorage *MockStorage) *MockServiceStore {
	return &MockServiceStore{
		storage: mockStorage,
		services: make(map[string]*storage.Service),
	}
}

func (m *MockServiceStore) Create(ctx context.Context, service *storage.Service) error {
	if m.services == nil {
		m.services = make(map[string]*storage.Service)
	}
	key := fmt.Sprintf("%s:%s", service.ClusterARN, service.ServiceName)
	if _, exists := m.services[key]; exists {
		return errors.New("service already exists")
	}
	m.services[key] = service
	return nil
}

func (m *MockServiceStore) Get(ctx context.Context, cluster, serviceName string) (*storage.Service, error) {
	key := fmt.Sprintf("%s:%s", cluster, serviceName)
	service, exists := m.services[key]
	if !exists {
		return nil, errors.New("service not found")
	}
	return service, nil
}

func (m *MockServiceStore) List(ctx context.Context, cluster string, serviceName string, launchType string, limit int, nextToken string) ([]*storage.Service, string, error) {
	var services []*storage.Service
	for _, service := range m.services {
		// Apply filters
		if cluster != "" && service.ClusterARN != cluster {
			continue
		}
		if serviceName != "" && service.ServiceName != serviceName {
			continue
		}
		if launchType != "" && service.LaunchType != launchType {
			continue
		}
		services = append(services, service)
		// Simple pagination for testing
		if limit > 0 && len(services) >= limit {
			break
		}
	}
	return services, "", nil
}

func (m *MockServiceStore) Update(ctx context.Context, service *storage.Service) error {
	key := fmt.Sprintf("%s:%s", service.ClusterARN, service.ServiceName)
	if _, exists := m.services[key]; !exists {
		return errors.New("service not found")
	}
	m.services[key] = service
	return nil
}

func (m *MockServiceStore) Delete(ctx context.Context, cluster, serviceName string) error {
	key := fmt.Sprintf("%s:%s", cluster, serviceName)
	delete(m.services, key)
	return nil
}

func (m *MockServiceStore) GetByARN(ctx context.Context, arn string) (*storage.Service, error) {
	for _, service := range m.services {
		if service.ARN == arn {
			return service, nil
		}
	}
	return nil, errors.New("service not found")
}

var _ = Describe("Service ECS API", func() {
	var (
		server *Server
		ctx    context.Context
		mockServiceStore *MockServiceStore
	)

	BeforeEach(func() {
		mockStorage := NewMockStorage()
		mockServiceStore = NewMockServiceStore(mockStorage)
		
		// Add mock service store to storage
		mockStorage.services = mockServiceStore.services
		
		server = &Server{
			storage:     mockStorage,
			kindManager: nil,
			ecsAPI:      NewDefaultECSAPI(mockStorage, nil),
		}
		ctx = context.Background()
		
		// Pre-populate with test data
		// Add a default cluster
		cluster := &storage.Cluster{
			Name:       "default",
			ARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
			Status:     "ACTIVE",
			Region:     "ap-northeast-1",
			AccountID:  "123456789012",
		}
		mockStorage.clusters["default"] = cluster
		
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
		mockServiceStore.services["arn:aws:ecs:ap-northeast-1:123456789012:cluster/default:test-service"] = testService
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
				mockServiceStore.services["arn:aws:ecs:ap-northeast-1:123456789012:cluster/default:test-service-2"] = service2
				
				service3 := &storage.Service{
					ID:          "service-3",
					ARN:         "arn:aws:ecs:ap-northeast-1:123456789012:service/default/other-service",
					ServiceName: "other-service",
					ClusterARN:  "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
					Namespace:   "other-namespace",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				mockServiceStore.services["arn:aws:ecs:ap-northeast-1:123456789012:cluster/default:other-service"] = service3
			})

			It("should list services in specified namespace", func() {
				namespace := "test-namespace"
				req := &generated.ListServicesByNamespaceRequest{
					Namespace: &namespace,
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
					Namespace: &namespace,
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
		Context("when updating primary task set", func() {
			It("should update primary task set successfully", func() {
				serviceName := "test-service"
				taskSetId := "new-task-set"
				req := &generated.UpdateServicePrimaryTaskSetRequest{
					Service:        &serviceName,
					PrimaryTaskSet: &taskSetId,
				}
				
				resp, err := server.ecsAPI.UpdateServicePrimaryTaskSet(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskSet).NotTo(BeNil())
				Expect(*resp.TaskSet.Id).To(Equal(taskSetId))
				Expect(*resp.TaskSet.Status).To(Equal("PRIMARY"))
			})

			It("should fail without service name", func() {
				taskSetId := "new-task-set"
				req := &generated.UpdateServicePrimaryTaskSetRequest{
					PrimaryTaskSet: &taskSetId,
				}
				
				_, err := server.ecsAPI.UpdateServicePrimaryTaskSet(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("service is required"))
			})

			It("should fail without primary task set", func() {
				serviceName := "test-service"
				req := &generated.UpdateServicePrimaryTaskSetRequest{
					Service: &serviceName,
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
				Expect(*deployment.Status).To(Equal(generated.ServiceDeploymentStatusSuccessful))
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
					Service: &serviceName,
				}
				
				resp, err := server.ecsAPI.ListServiceDeployments(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.ServiceDeployments).To(HaveLen(2)) // Current and previous
				
				// Check current deployment
				current := resp.ServiceDeployments[0]
				Expect(*current.ServiceDeploymentArn).To(ContainSubstring("current"))
				Expect(*current.Status).To(Equal(generated.ServiceDeploymentStatusSuccessful))
			})

			It("should filter by status", func() {
				serviceName := "test-service"
				status := []generated.ServiceDeploymentStatus{generated.ServiceDeploymentStatusSuccessful}
				req := &generated.ListServiceDeploymentsRequest{
					Service: &serviceName,
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
					ServiceDeploymentArn: &deploymentArn,
				}
				
				resp, err := server.ecsAPI.StopServiceDeployment(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(*resp.ServiceDeploymentArn).To(Equal(deploymentArn))
			})

			It("should fail with invalid ARN format", func() {
				deploymentArn := "invalid-arn"
				req := &generated.StopServiceDeploymentRequest{
					ServiceDeploymentArn: &deploymentArn,
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