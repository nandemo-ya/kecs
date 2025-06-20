package api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("Service ECS API V2", func() {
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

		// Create a test cluster
		cluster := &storage.Cluster{
			Name:   "test-cluster",
			ARN:    "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
			Status: "ACTIVE",
		}
		err = testStorage.ClusterStore().Create(ctx, cluster)
		Expect(err).ToNot(HaveOccurred())

		// Create a test task definition
		taskDef := &storage.TaskDefinition{
			Family:               "test-task",
			Revision:             1,
			ARN:                  "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
			ContainerDefinitions: `[{"name":"app","image":"nginx:latest","memory":512,"essential":true}]`,
			Status:               "ACTIVE",
		}
		_, err = testStorage.TaskDefinitionStore().Register(ctx, taskDef)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		if testStorage != nil {
			testStorage.Close()
		}
	})

	Describe("CreateServiceV2", func() {
		It("should create a new service", func() {
			req := &ecs.CreateServiceInput{
				Cluster:        aws.String("test-cluster"),
				ServiceName:    aws.String("test-service"),
				TaskDefinition: aws.String("test-task:1"),
				DesiredCount:   aws.Int32(2),
			}
			resp, err := ecsAPIV2.CreateServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Service).ToNot(BeNil())
			Expect(*resp.Service.ServiceName).To(Equal("test-service"))
			Expect(resp.Service.DesiredCount).To(Equal(int32(2)))
			Expect(*resp.Service.Status).To(Equal("ACTIVE"))
			Expect(*resp.Service.TaskDefinition).To(ContainSubstring("test-task:1"))
		})

		It("should use default cluster when not specified", func() {
			// Create default cluster
			cluster := &storage.Cluster{
				Name:   "default",
				ARN:    "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
				Status: "ACTIVE",
			}
			err := testStorage.ClusterStore().Create(ctx, cluster)
			Expect(err).ToNot(HaveOccurred())

			req := &ecs.CreateServiceInput{
				ServiceName:    aws.String("default-service"),
				TaskDefinition: aws.String("test-task"),
			}
			resp, err := ecsAPIV2.CreateServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.Service.ClusterArn).To(ContainSubstring("default"))
		})

		It("should fail when cluster not found", func() {
			req := &ecs.CreateServiceInput{
				Cluster:        aws.String("non-existent"),
				ServiceName:    aws.String("test-service"),
				TaskDefinition: aws.String("test-task"),
			}
			_, err := ecsAPIV2.CreateServiceV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cluster not found"))
		})

		It("should fail when task definition not found", func() {
			req := &ecs.CreateServiceInput{
				Cluster:        aws.String("test-cluster"),
				ServiceName:    aws.String("test-service"),
				TaskDefinition: aws.String("non-existent:1"),
			}
			_, err := ecsAPIV2.CreateServiceV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("task definition not found"))
		})

		It("should create service with advanced configuration", func() {
			req := &ecs.CreateServiceInput{
				Cluster:        aws.String("test-cluster"),
				ServiceName:    aws.String("advanced-service"),
				TaskDefinition: aws.String("test-task:1"),
				DesiredCount:   aws.Int32(3),
				LaunchType:     ecstypes.LaunchTypeFargate,
				NetworkConfiguration: &ecstypes.NetworkConfiguration{
					AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
						Subnets:        []string{"subnet-123", "subnet-456"},
						SecurityGroups: []string{"sg-123"},
						AssignPublicIp: ecstypes.AssignPublicIpEnabled,
					},
				},
				LoadBalancers: []ecstypes.LoadBalancer{
					{
						TargetGroupArn: aws.String("arn:aws:elasticloadbalancing:region:account:targetgroup/tg/123"),
						ContainerName:  aws.String("app"),
						ContainerPort:  aws.Int32(80),
					},
				},
				EnableExecuteCommand: true,
				Tags: []ecstypes.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				},
			}
			resp, err := ecsAPIV2.CreateServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Service.LaunchType).To(Equal(ecstypes.LaunchTypeFargate))
			Expect(resp.Service.NetworkConfiguration).ToNot(BeNil())
			Expect(resp.Service.LoadBalancers).To(HaveLen(1))
			Expect(resp.Service.EnableExecuteCommand).To(Equal(true))
			Expect(resp.Service.Tags).To(HaveLen(1))
		})
	})

	Describe("ListServicesV2", func() {
		BeforeEach(func() {
			// Create test services
			for i := 1; i <= 3; i++ {
				service := &storage.Service{
					ServiceName:       fmt.Sprintf("service-%d", i),
					ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/service-%d", i),
					ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
					TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
					Status:            "ACTIVE",
					LaunchType:        "FARGATE",
				}
				err := testStorage.ServiceStore().Create(ctx, service)
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("should list all services in cluster", func() {
			req := &ecs.ListServicesInput{
				Cluster: aws.String("test-cluster"),
			}
			resp, err := ecsAPIV2.ListServicesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ServiceArns).To(HaveLen(3))
			Expect(resp.ServiceArns).To(ContainElements(
				ContainSubstring("service-1"),
				ContainSubstring("service-2"),
				ContainSubstring("service-3"),
			))
		})

		It("should filter by launch type", func() {
			// Create EC2 service
			ec2Service := &storage.Service{
				ServiceName:       "ec2-service",
				ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/ec2-service",
				ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				Status:            "ACTIVE",
				LaunchType:        "EC2",
			}
			err := testStorage.ServiceStore().Create(ctx, ec2Service)
			Expect(err).ToNot(HaveOccurred())

			req := &ecs.ListServicesInput{
				Cluster:    aws.String("test-cluster"),
				LaunchType: ecstypes.LaunchTypeEc2,
			}
			resp, err := ecsAPIV2.ListServicesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ServiceArns).To(HaveLen(1))
			Expect(resp.ServiceArns[0]).To(ContainSubstring("ec2-service"))
		})

		It("should handle pagination", func() {
			req := &ecs.ListServicesInput{
				Cluster:    aws.String("test-cluster"),
				MaxResults: aws.Int32(2),
			}
			resp, err := ecsAPIV2.ListServicesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.ServiceArns).To(HaveLen(2))
			Expect(resp.NextToken).ToNot(BeNil())
		})
	})

	Describe("DescribeServicesV2", func() {
		BeforeEach(func() {
			// Create test service with details
			service := &storage.Service{
				ServiceName:               "describe-test",
				ARN:                       "arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/describe-test",
				ClusterARN:                "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN:         "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				Status:                    "ACTIVE",
				DesiredCount:              3,
				RunningCount:              2,
				PendingCount:              1,
				LaunchType:                "FARGATE",
				NetworkConfiguration:      `{"awsvpcConfiguration":{"subnets":["subnet-123"],"securityGroups":["sg-123"]}}`,
				LoadBalancers:             `[{"targetGroupArn":"arn:aws:elasticloadbalancing:region:account:targetgroup/tg/123","containerName":"app","containerPort":80}]`,
				Tags:                      `[{"key":"Environment","value":"test"}]`,
				EnableExecuteCommand:      true,
				HealthCheckGracePeriodSeconds: 60,
			}
			err := testStorage.ServiceStore().Create(ctx, service)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should describe specific service", func() {
			req := &ecs.DescribeServicesInput{
				Cluster:  aws.String("test-cluster"),
				Services: []string{"describe-test"},
			}
			resp, err := ecsAPIV2.DescribeServicesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Services).To(HaveLen(1))
			
			service := resp.Services[0]
			Expect(*service.ServiceName).To(Equal("describe-test"))
			Expect(service.DesiredCount).To(Equal(int32(3)))
			Expect(service.RunningCount).To(Equal(int32(2)))
			Expect(service.PendingCount).To(Equal(int32(1)))
			Expect(service.NetworkConfiguration).ToNot(BeNil())
			Expect(service.LoadBalancers).To(HaveLen(1))
			Expect(service.Tags).To(HaveLen(1))
			Expect(service.EnableExecuteCommand).To(BeTrue())
			Expect(*service.HealthCheckGracePeriodSeconds).To(Equal(int32(60)))
		})

		It("should describe all services when none specified", func() {
			req := &ecs.DescribeServicesInput{
				Cluster: aws.String("test-cluster"),
			}
			resp, err := ecsAPIV2.DescribeServicesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Services).To(HaveLen(1))
		})

		It("should handle non-existent services", func() {
			req := &ecs.DescribeServicesInput{
				Cluster:  aws.String("test-cluster"),
				Services: []string{"describe-test", "non-existent"},
			}
			resp, err := ecsAPIV2.DescribeServicesV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Services).To(HaveLen(1))
			Expect(resp.Failures).To(HaveLen(1))
			Expect(*resp.Failures[0].Arn).To(Equal("non-existent"))
			Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
		})
	})

	Describe("UpdateServiceV2", func() {
		BeforeEach(func() {
			// Create test service
			service := &storage.Service{
				ServiceName:       "update-test",
				ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/update-test",
				ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				Status:            "ACTIVE",
				DesiredCount:      2,
				LaunchType:        "FARGATE",
			}
			err := testStorage.ServiceStore().Create(ctx, service)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should update desired count", func() {
			req := &ecs.UpdateServiceInput{
				Cluster:      aws.String("test-cluster"),
				Service:      aws.String("update-test"),
				DesiredCount: aws.Int32(5),
			}
			resp, err := ecsAPIV2.UpdateServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Service.DesiredCount).To(Equal(int32(5)))
		})

		It("should update task definition", func() {
			// Create new task definition revision
			taskDef := &storage.TaskDefinition{
				Family:               "test-task",
				Revision:             2,
				ARN:                  "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:2",
				ContainerDefinitions: `[{"name":"app","image":"nginx:1.21","memory":512,"essential":true}]`,
				Status:               "ACTIVE",
			}
			_, err := testStorage.TaskDefinitionStore().Register(ctx, taskDef)
			Expect(err).ToNot(HaveOccurred())

			req := &ecs.UpdateServiceInput{
				Cluster:        aws.String("test-cluster"),
				Service:        aws.String("update-test"),
				TaskDefinition: aws.String("test-task:2"),
			}
			resp, err := ecsAPIV2.UpdateServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.Service.TaskDefinition).To(ContainSubstring("test-task:2"))
		})

		It("should update network configuration", func() {
			req := &ecs.UpdateServiceInput{
				Cluster: aws.String("test-cluster"),
				Service: aws.String("update-test"),
				NetworkConfiguration: &ecstypes.NetworkConfiguration{
					AwsvpcConfiguration: &ecstypes.AwsVpcConfiguration{
						Subnets:        []string{"subnet-789"},
						SecurityGroups: []string{"sg-456"},
					},
				},
			}
			resp, err := ecsAPIV2.UpdateServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Service.NetworkConfiguration).ToNot(BeNil())
			Expect(resp.Service.NetworkConfiguration.AwsvpcConfiguration.Subnets).To(ContainElement("subnet-789"))
		})

		It("should fail when service not found", func() {
			req := &ecs.UpdateServiceInput{
				Cluster:      aws.String("test-cluster"),
				Service:      aws.String("non-existent"),
				DesiredCount: aws.Int32(3),
			}
			_, err := ecsAPIV2.UpdateServiceV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("service not found"))
		})
	})

	Describe("DeleteServiceV2", func() {
		BeforeEach(func() {
			// Create test service
			service := &storage.Service{
				ServiceName:       "delete-test",
				ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/delete-test",
				ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				Status:            "ACTIVE",
				DesiredCount:      0,
				RunningCount:      0,
			}
			err := testStorage.ServiceStore().Create(ctx, service)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should delete service with zero desired count", func() {
			req := &ecs.DeleteServiceInput{
				Cluster: aws.String("test-cluster"),
				Service: aws.String("delete-test"),
			}
			resp, err := ecsAPIV2.DeleteServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.Service.ServiceName).To(Equal("delete-test"))
			Expect(*resp.Service.Status).To(Equal("INACTIVE"))

			// Verify service is deleted
			_, err = testStorage.ServiceStore().Get(ctx, "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster", "delete-test")
			Expect(err).To(HaveOccurred())
		})

		It("should force delete service with running tasks", func() {
			// Create service with running tasks
			busyService := &storage.Service{
				ServiceName:       "busy-service",
				ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/busy-service",
				ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				Status:            "ACTIVE",
				DesiredCount:      2,
				RunningCount:      2,
			}
			err := testStorage.ServiceStore().Create(ctx, busyService)
			Expect(err).ToNot(HaveOccurred())

			req := &ecs.DeleteServiceInput{
				Cluster: aws.String("test-cluster"),
				Service: aws.String("busy-service"),
				Force:   aws.Bool(true),
			}
			resp, err := ecsAPIV2.DeleteServiceV2(ctx, req)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(*resp.Service.Status).To(Equal("INACTIVE"))
		})

		It("should fail to delete service with desired count > 0 without force", func() {
			// Create service with desired count > 0
			activeService := &storage.Service{
				ServiceName:       "active-service",
				ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:service/test-cluster/active-service",
				ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/test-cluster",
				TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/test-task:1",
				Status:            "ACTIVE",
				DesiredCount:      1,
			}
			err := testStorage.ServiceStore().Create(ctx, activeService)
			Expect(err).ToNot(HaveOccurred())

			req := &ecs.DeleteServiceInput{
				Cluster: aws.String("test-cluster"),
				Service: aws.String("active-service"),
			}
			_, err = ecsAPIV2.DeleteServiceV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("desired count > 0"))
		})

		It("should fail when service not found", func() {
			req := &ecs.DeleteServiceInput{
				Cluster: aws.String("test-cluster"),
				Service: aws.String("non-existent"),
			}
			_, err := ecsAPIV2.DeleteServiceV2(ctx, req)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("service not found"))
		})
	})
})

func TestServiceECSAPIV2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service ECS API V2 Suite")
}