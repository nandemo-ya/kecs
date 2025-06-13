package api

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/mocks"
)

var _ = Describe("Task Definition ECS API", func() {
	var (
		server *Server
		ctx    context.Context
		mockStorage *mocks.MockStorage
		mockTaskDefStore *mocks.MockTaskDefinitionStore
	)

	BeforeEach(func() {
		mockStorage = mocks.NewMockStorage()
		mockTaskDefStore = mocks.NewMockTaskDefinitionStore()
		mockStorage.SetTaskDefinitionStore(mockTaskDefStore)
		
		server = &Server{
			storage:     mockStorage,
			kindManager: nil, // Skip actual kind cluster creation in tests
			ecsAPI:      NewDefaultECSAPI(mockStorage, nil),
		}
		ctx = context.Background()
	})

	Describe("RegisterTaskDefinition", func() {
		Context("when registering a task definition", func() {
			It("should register a new task definition", func() {
				family := "nginx"
				containerDefs := []generated.ContainerDefinition{
					{
						Name:      ptr.String("nginx"),
						Image:     ptr.String("nginx:latest"),
						Memory:    ptr.Int32(512),
						Essential: ptr.Bool(true),
					},
				}

				req := &generated.RegisterTaskDefinitionRequest{
					Family:               &family,
					ContainerDefinitions: containerDefs,
				}

				resp, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinition).NotTo(BeNil())
				Expect(*resp.TaskDefinition.Family).To(Equal("nginx"))
				Expect(*resp.TaskDefinition.Revision).To(Equal(int32(1)))
				Expect(*resp.TaskDefinition.Status).To(Equal(generated.TaskDefinitionStatusActive))
			})

			It("should increment revision for existing family", func() {
				family := "webapp"
				containerDefs := []generated.ContainerDefinition{
					{
						Name:      ptr.String("app"),
						Image:     ptr.String("app:v1"),
						Memory:    ptr.Int32(256),
						Essential: ptr.Bool(true),
					},
				}

				// Register first version
				req1 := &generated.RegisterTaskDefinitionRequest{
					Family:               &family,
					ContainerDefinitions: containerDefs,
				}
				resp1, err := server.ecsAPI.RegisterTaskDefinition(ctx, req1)
				Expect(err).NotTo(HaveOccurred())
				Expect(*resp1.TaskDefinition.Revision).To(Equal(int32(1)))

				// Update container definition
				containerDefs[0].Image = ptr.String("app:v2")
				req2 := &generated.RegisterTaskDefinitionRequest{
					Family:               &family,
					ContainerDefinitions: containerDefs,
				}
				resp2, err := server.ecsAPI.RegisterTaskDefinition(ctx, req2)
				Expect(err).NotTo(HaveOccurred())
				Expect(*resp2.TaskDefinition.Revision).To(Equal(int32(2)))
			})

			It("should fail without required fields", func() {
				req := &generated.RegisterTaskDefinitionRequest{}
				_, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("family is required"))
			})
		})
	})

	Describe("DescribeTaskDefinition", func() {
		Context("when describing a task definition", func() {
			BeforeEach(func() {
				// Register a test task definition
				family := "describe-test"
				containerDefs := []generated.ContainerDefinition{
					{
						Name:      ptr.String("app"),
						Image:     ptr.String("app:latest"),
						Memory:    ptr.Int32(512),
						Essential: ptr.Bool(true),
					},
				}
				req := &generated.RegisterTaskDefinitionRequest{
					Family:               &family,
					ContainerDefinitions: containerDefs,
				}
				_, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should describe by family:revision", func() {
				taskDef := "describe-test:1"
				req := &generated.DescribeTaskDefinitionRequest{
					TaskDefinition: &taskDef,
				}

				resp, err := server.ecsAPI.DescribeTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinition).NotTo(BeNil())
				Expect(*resp.TaskDefinition.Family).To(Equal("describe-test"))
				Expect(*resp.TaskDefinition.Revision).To(Equal(int32(1)))
			})

			It("should describe latest by family name only", func() {
				taskDef := "describe-test"
				req := &generated.DescribeTaskDefinitionRequest{
					TaskDefinition: &taskDef,
				}

				resp, err := server.ecsAPI.DescribeTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinition).NotTo(BeNil())
				Expect(*resp.TaskDefinition.Family).To(Equal("describe-test"))
			})

			It("should describe by ARN", func() {
				// Get the ARN from storage
				taskDef, err := server.storage.TaskDefinitionStore().Get(ctx, "describe-test", 1)
				Expect(err).NotTo(HaveOccurred())

				req := &generated.DescribeTaskDefinitionRequest{
					TaskDefinition: &taskDef.ARN,
				}

				resp, err := server.ecsAPI.DescribeTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinition).NotTo(BeNil())
				Expect(*resp.TaskDefinition.Family).To(Equal("describe-test"))
			})

			It("should fail for non-existent task definition", func() {
				taskDef := "non-existent:1"
				req := &generated.DescribeTaskDefinitionRequest{
					TaskDefinition: &taskDef,
				}

				_, err := server.ecsAPI.DescribeTaskDefinition(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})

	Describe("DeregisterTaskDefinition", func() {
		Context("when deregistering a task definition", func() {
			BeforeEach(func() {
				// Register a test task definition
				family := "deregister-test"
				containerDefs := []generated.ContainerDefinition{
					{
						Name:      ptr.String("app"),
						Image:     ptr.String("app:latest"),
						Memory:    ptr.Int32(512),
						Essential: ptr.Bool(true),
					},
				}
				req := &generated.RegisterTaskDefinitionRequest{
					Family:               &family,
					ContainerDefinitions: containerDefs,
				}
				_, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should deregister a task definition", func() {
				taskDef := "deregister-test:1"
				req := &generated.DeregisterTaskDefinitionRequest{
					TaskDefinition: &taskDef,
				}

				resp, err := server.ecsAPI.DeregisterTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinition).NotTo(BeNil())
				Expect(*resp.TaskDefinition.Status).To(Equal(generated.TaskDefinitionStatusInactive))
			})

			It("should be idempotent", func() {
				taskDef := "deregister-test:1"
				req := &generated.DeregisterTaskDefinitionRequest{
					TaskDefinition: &taskDef,
				}

				// First deregister
				_, err := server.ecsAPI.DeregisterTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Second deregister should succeed (idempotent)
				resp, err := server.ecsAPI.DeregisterTaskDefinition(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).NotTo(BeNil())
			})

			It("should require revision number", func() {
				taskDef := "deregister-test"
				req := &generated.DeregisterTaskDefinitionRequest{
					TaskDefinition: &taskDef,
				}

				_, err := server.ecsAPI.DeregisterTaskDefinition(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must include revision"))
			})
		})
	})

	Describe("ListTaskDefinitionFamilies", func() {
		Context("when listing task definition families", func() {
			BeforeEach(func() {
				// Register multiple task definitions
				families := []string{"app-web", "app-api", "worker-batch", "worker-stream"}
				for _, family := range families {
					fam := family
					containerDefs := []generated.ContainerDefinition{
						{
							Name:      ptr.String("container"),
							Image:     ptr.String("image:latest"),
							Memory:    ptr.Int32(512),
							Essential: ptr.Bool(true),
						},
					}
					req := &generated.RegisterTaskDefinitionRequest{
						Family:               &fam,
						ContainerDefinitions: containerDefs,
					}
					_, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should list all families", func() {
				req := &generated.ListTaskDefinitionFamiliesRequest{}

				resp, err := server.ecsAPI.ListTaskDefinitionFamilies(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Families).To(HaveLen(4))
			})

			It("should filter by family prefix", func() {
				prefix := "app-"
				req := &generated.ListTaskDefinitionFamiliesRequest{
					FamilyPrefix: &prefix,
				}

				resp, err := server.ecsAPI.ListTaskDefinitionFamilies(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Families).To(HaveLen(2))
				Expect(resp.Families).To(ConsistOf("app-web", "app-api"))
			})

			It("should respect max results", func() {
				maxResults := int32(2)
				req := &generated.ListTaskDefinitionFamiliesRequest{
					MaxResults: &maxResults,
				}

				resp, err := server.ecsAPI.ListTaskDefinitionFamilies(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.Families).To(HaveLen(2))
				Expect(resp.NextToken).NotTo(BeNil())
			})
		})
	})

	Describe("ListTaskDefinitions", func() {
		Context("when listing task definitions", func() {
			BeforeEach(func() {
				// Register multiple revisions
				family := "list-test"
				for i := 1; i <= 3; i++ {
					fam := family
					containerDefs := []generated.ContainerDefinition{
						{
							Name:      ptr.String("container"),
							Image:     ptr.String("image:v" + string(rune(i+'0'))),
							Memory:    ptr.Int32(512),
							Essential: ptr.Bool(true),
						},
					}
					req := &generated.RegisterTaskDefinitionRequest{
						Family:               &fam,
						ContainerDefinitions: containerDefs,
					}
					_, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should list all task definitions", func() {
				req := &generated.ListTaskDefinitionsRequest{}

				resp, err := server.ecsAPI.ListTaskDefinitions(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinitionArns).To(HaveLen(3))
			})

			It("should filter by family prefix", func() {
				prefix := "list-test"
				req := &generated.ListTaskDefinitionsRequest{
					FamilyPrefix: &prefix,
				}

				resp, err := server.ecsAPI.ListTaskDefinitions(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinitionArns).To(HaveLen(3))
			})
		})
	})

	Describe("DeleteTaskDefinitions", func() {
		Context("when deleting task definitions", func() {
			BeforeEach(func() {
				// Register test task definitions
				families := []string{"delete-test-1", "delete-test-2"}
				for _, family := range families {
					fam := family
					containerDefs := []generated.ContainerDefinition{
						{
							Name:      ptr.String("container"),
							Image:     ptr.String("image:latest"),
							Memory:    ptr.Int32(512),
							Essential: ptr.Bool(true),
						},
					}
					req := &generated.RegisterTaskDefinitionRequest{
						Family:               &fam,
						ContainerDefinitions: containerDefs,
					}
					_, err := server.ecsAPI.RegisterTaskDefinition(ctx, req)
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("should delete multiple task definitions", func() {
				taskDefs := []string{"delete-test-1:1", "delete-test-2:1"}
				req := &generated.DeleteTaskDefinitionsRequest{
					TaskDefinitions: taskDefs,
				}

				resp, err := server.ecsAPI.DeleteTaskDefinitions(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinitions).To(HaveLen(2))
				Expect(resp.Failures).To(BeEmpty())

				// Verify all are inactive
				for _, td := range resp.TaskDefinitions {
					Expect(*td.Status).To(Equal(generated.TaskDefinitionStatusInactive))
				}
			})

			It("should handle partial failures", func() {
				taskDefs := []string{"delete-test-1:1", "non-existent:1"}
				req := &generated.DeleteTaskDefinitionsRequest{
					TaskDefinitions: taskDefs,
				}

				resp, err := server.ecsAPI.DeleteTaskDefinitions(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskDefinitions).To(HaveLen(1))
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})

			It("should require task definitions", func() {
				req := &generated.DeleteTaskDefinitionsRequest{}

				_, err := server.ecsAPI.DeleteTaskDefinitions(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("taskDefinitions is required"))
			})
		})
	})
})