package api_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("TaskDefinition ECS API V2", func() {
	var (
		ecsAPIV2    *api.DefaultECSAPIV2
		testStorage storage.Storage
		ctx         context.Context
	)

	BeforeEach(func() {
		// Create test storage
		var err error
		testStorage, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).ToNot(HaveOccurred())

		// Initialize tables
		err = testStorage.Initialize(context.Background())
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()

		// Initialize V2 API
		ecsAPIV2 = api.NewDefaultECSAPIV2(testStorage, nil)
	})

	AfterEach(func() {
		if testStorage != nil {
			testStorage.Close()
		}
	})

	Describe("RegisterTaskDefinitionV2", func() {
		It("should register a new task definition", func() {
			req := &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-family"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:latest"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
			}
			resp, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskDefinition).ToNot(BeNil())
			Expect(*resp.TaskDefinition.Family).To(Equal("test-family"))
			Expect(resp.TaskDefinition.Revision).To(Equal(int32(1)))
			Expect(resp.TaskDefinition.Status).To(Equal(ecstypes.TaskDefinitionStatusActive))
			Expect(resp.TaskDefinition.ContainerDefinitions).To(HaveLen(1))
		})

		It("should increment revision for same family", func() {
			// Register first revision
			req1 := &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-family"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:1.0"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
			}
			resp1, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req1)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp1.TaskDefinition.Revision).To(Equal(int32(1)))

			// Register second revision
			req2 := &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-family"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:2.0"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
			}
			resp2, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req2)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp2.TaskDefinition.Revision).To(Equal(int32(2)))
		})

		It("should register task definition with all fields", func() {
			req := &ecs.RegisterTaskDefinitionInput{
				Family:           aws.String("complex-family"),
				TaskRoleArn:      aws.String("arn:aws:iam::123456789012:role/TaskRole"),
				ExecutionRoleArn: aws.String("arn:aws:iam::123456789012:role/ExecutionRole"),
				NetworkMode:      ecstypes.NetworkModeAwsvpc,
				Cpu:              aws.String("256"),
				Memory:           aws.String("512"),
				PidMode:          ecstypes.PidModeTask,
				IpcMode:          ecstypes.IpcModeTask,
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:latest"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
						PortMappings: []ecstypes.PortMapping{
							{
								ContainerPort: aws.Int32(80),
								Protocol:      ecstypes.TransportProtocolTcp,
							},
						},
						Environment: []ecstypes.KeyValuePair{
							{
								Name:  aws.String("ENV"),
								Value: aws.String("production"),
							},
						},
					},
				},
				Volumes: []ecstypes.Volume{
					{
						Name: aws.String("data"),
						Host: &ecstypes.HostVolumeProperties{
							SourcePath: aws.String("/data"),
						},
					},
				},
				PlacementConstraints: []ecstypes.TaskDefinitionPlacementConstraint{
					{
						Type:       ecstypes.TaskDefinitionPlacementConstraintTypeMemberOf,
						Expression: aws.String("attribute:ecs.instance-type =~ t2.*"),
					},
				},
				RequiresCompatibilities: []ecstypes.Compatibility{
					ecstypes.CompatibilityFargate,
				},
				Tags: []ecstypes.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				},
			}
			resp, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.TaskDefinition.TaskRoleArn).To(Equal("arn:aws:iam::123456789012:role/TaskRole"))
			Expect(*resp.TaskDefinition.ExecutionRoleArn).To(Equal("arn:aws:iam::123456789012:role/ExecutionRole"))
			Expect(resp.TaskDefinition.NetworkMode).To(Equal(ecstypes.NetworkModeAwsvpc))
			Expect(*resp.TaskDefinition.Cpu).To(Equal("256"))
			Expect(*resp.TaskDefinition.Memory).To(Equal("512"))
			Expect(resp.TaskDefinition.Volumes).To(HaveLen(1))
			Expect(resp.TaskDefinition.PlacementConstraints).To(HaveLen(1))
			Expect(resp.TaskDefinition.RequiresCompatibilities).To(HaveLen(1))
			Expect(resp.Tags).To(HaveLen(1))
		})

		It("should fail when family is missing", func() {
			req := &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:   aws.String("app"),
						Image:  aws.String("nginx"),
						Memory: aws.Int32(512),
					},
				},
			}
			_, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("family is required"))
		})

		It("should fail when container definitions are missing", func() {
			req := &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-family"),
			}
			_, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("containerDefinitions is required"))
		})
	})

	Describe("DeregisterTaskDefinitionV2", func() {
		var taskDefArn string

		BeforeEach(func() {
			// Register a task definition first
			req := &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-family"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:latest"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
			}
			resp, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			taskDefArn = *resp.TaskDefinition.TaskDefinitionArn
		})

		It("should deregister a task definition", func() {
			req := &ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: aws.String(taskDefArn),
			}
			resp, err := ecsAPIV2.DeregisterTaskDefinitionV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskDefinition).ToNot(BeNil())
			Expect(resp.TaskDefinition.Status).To(Equal(ecstypes.TaskDefinitionStatusInactive))
		})

		It("should fail when task definition not found", func() {
			req := &ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: aws.String("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/non-existent:1"),
			}
			_, err := ecsAPIV2.DeregisterTaskDefinitionV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to deregister task definition"))
		})
	})

	Describe("DescribeTaskDefinitionV2", func() {
		var taskDefArn string

		BeforeEach(func() {
			// Register a task definition first
			req := &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("test-family"),
				TaskRoleArn: aws.String("arn:aws:iam::123456789012:role/TaskRole"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:latest"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
				Tags: []ecstypes.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				},
			}
			resp, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)
			Expect(err).ToNot(HaveOccurred())
			taskDefArn = *resp.TaskDefinition.TaskDefinitionArn
		})

		It("should describe task definition by ARN", func() {
			req := &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String(taskDefArn),
			}
			resp, err := ecsAPIV2.DescribeTaskDefinitionV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.TaskDefinition.Family).To(Equal("test-family"))
			Expect(resp.TaskDefinition.Revision).To(Equal(int32(1)))
			Expect(*resp.TaskDefinition.TaskRoleArn).To(Equal("arn:aws:iam::123456789012:role/TaskRole"))
			Expect(resp.Tags).To(HaveLen(1))
		})

		It("should describe task definition by family:revision", func() {
			req := &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String("test-family:1"),
			}
			resp, err := ecsAPIV2.DescribeTaskDefinitionV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.TaskDefinition.Family).To(Equal("test-family"))
			Expect(resp.TaskDefinition.Revision).To(Equal(int32(1)))
		})

		It("should describe latest task definition by family", func() {
			// Register second revision
			req2 := &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-family"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:2.0"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
			}
			_, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req2)
			Expect(err).ToNot(HaveOccurred())

			req := &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String("test-family"),
			}
			resp, err := ecsAPIV2.DescribeTaskDefinitionV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.TaskDefinition.Family).To(Equal("test-family"))
			Expect(resp.TaskDefinition.Revision).To(Equal(int32(2)))
		})

		It("should fail when task definition not found", func() {
			req := &ecs.DescribeTaskDefinitionInput{
				TaskDefinition: aws.String("non-existent:1"),
			}
			_, err := ecsAPIV2.DescribeTaskDefinitionV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("task definition not found"))
		})
	})

	Describe("ListTaskDefinitionFamiliesV2", func() {
		BeforeEach(func() {
			// Register task definitions with different families
			families := []string{"web-app", "web-api", "backend-service", "data-processor"}
			for _, family := range families {
				req := &ecs.RegisterTaskDefinitionInput{
					Family: aws.String(family),
					ContainerDefinitions: []ecstypes.ContainerDefinition{
						{
							Name:      aws.String("app"),
							Image:     aws.String("nginx:latest"),
							Memory:    aws.Int32(512),
							Essential: aws.Bool(true),
						},
					},
				}
				_, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should list all task definition families", func() {
			req := &ecs.ListTaskDefinitionFamiliesInput{}
			resp, err := ecsAPIV2.ListTaskDefinitionFamiliesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Families).To(HaveLen(4))
			Expect(resp.Families).To(ContainElements("web-app", "web-api", "backend-service", "data-processor"))
		})

		It("should filter families by prefix", func() {
			req := &ecs.ListTaskDefinitionFamiliesInput{
				FamilyPrefix: aws.String("web-"),
			}
			resp, err := ecsAPIV2.ListTaskDefinitionFamiliesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Families).To(HaveLen(2))
			Expect(resp.Families).To(ContainElements("web-app", "web-api"))
		})

		It("should respect max results", func() {
			req := &ecs.ListTaskDefinitionFamiliesInput{
				MaxResults: aws.Int32(2),
			}
			resp, err := ecsAPIV2.ListTaskDefinitionFamiliesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Families).To(HaveLen(2))
			Expect(resp.NextToken).ToNot(BeNil())
		})
	})

	Describe("ListTaskDefinitionsV2", func() {
		BeforeEach(func() {
			// Register multiple revisions of different families
			families := []struct {
				family    string
				revisions int
			}{
				{"web-app", 3},
				{"backend-service", 2},
			}

			for _, f := range families {
				for i := 1; i <= f.revisions; i++ {
					req := &ecs.RegisterTaskDefinitionInput{
						Family: aws.String(f.family),
						ContainerDefinitions: []ecstypes.ContainerDefinition{
							{
								Name:      aws.String("app"),
								Image:     aws.String(fmt.Sprintf("nginx:%d.0", i)),
								Memory:    aws.Int32(512),
								Essential: aws.Bool(true),
							},
						},
					}
					_, err := ecsAPIV2.RegisterTaskDefinitionV2(ctx, req)
					Expect(err).ToNot(HaveOccurred())
				}
			}
		})

		It("should list all task definitions", func() {
			req := &ecs.ListTaskDefinitionsInput{}
			resp, err := ecsAPIV2.ListTaskDefinitionsV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskDefinitionArns).To(HaveLen(5)) // 3 + 2 revisions
		})

		It("should list task definitions for specific family", func() {
			req := &ecs.ListTaskDefinitionsInput{
				FamilyPrefix: aws.String("web-app"),
			}
			resp, err := ecsAPIV2.ListTaskDefinitionsV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskDefinitionArns).To(HaveLen(3))
			for _, arn := range resp.TaskDefinitionArns {
				Expect(arn).To(ContainSubstring("web-app"))
			}
		})

		It("should filter by status", func() {
			// Deregister one task definition
			deregReq := &ecs.DeregisterTaskDefinitionInput{
				TaskDefinition: aws.String("web-app:1"),
			}
			_, err := ecsAPIV2.DeregisterTaskDefinitionV2(ctx, deregReq)
			Expect(err).ToNot(HaveOccurred())

			// List only INACTIVE
			req := &ecs.ListTaskDefinitionsInput{
				Status: ecstypes.TaskDefinitionStatusInactive,
			}
			resp, err := ecsAPIV2.ListTaskDefinitionsV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.TaskDefinitionArns).To(HaveLen(1))
			Expect(resp.TaskDefinitionArns[0]).To(ContainSubstring("web-app:1"))
		})
	})
})