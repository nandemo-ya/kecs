package networking_test

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage/duckdb"
)

var _ = Describe("AWSVPC Network Mode Integration", func() {
	var (
		ctx       context.Context
		ecsAPI    *api.DefaultECSAPI
		store     storage.Storage
		region    = "us-east-1"
		accountID = "123456789012"
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Set test mode to avoid requiring actual Kubernetes cluster
		os.Setenv("KECS_TEST_MODE", "true")

		// Initialize storage
		var err error
		store, err = duckdb.NewDuckDBStorage(":memory:")
		Expect(err).NotTo(HaveOccurred())

		// Initialize the database schema
		err = store.Initialize(ctx)
		Expect(err).NotTo(HaveOccurred())

		// Initialize ECS API with nil KindManager for test mode
		ecsAPI = api.NewDefaultECSAPIWithConfig(store, nil, region, accountID).(*api.DefaultECSAPI)

		// Create default cluster
		_, err = ecsAPI.CreateCluster(ctx, &generated.CreateClusterRequest{
			ClusterName: ptr.String("default"),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if store != nil {
			store.Close()
		}
	})

	Describe("Task with AWSVPC Network Mode", func() {
		var taskDefArn string

		BeforeEach(func() {
			// Register task definition with awsvpc mode
			containerDef := generated.ContainerDefinition{
				Name:  ptr.String("nginx"),
				Image: ptr.String("nginx:latest"),
				PortMappings: []generated.PortMapping{
					{
						ContainerPort: ptr.Int32(80),
						Protocol:      (*generated.TransportProtocol)(ptr.String("tcp")),
					},
				},
				Memory: ptr.Int32(512),
			}

			registerResp, err := ecsAPI.RegisterTaskDefinition(ctx, &generated.RegisterTaskDefinitionRequest{
				Family:      "test-awsvpc",
				NetworkMode: (*generated.NetworkMode)(ptr.String("awsvpc")),
				RequiresCompatibilities: []generated.Compatibility{
					generated.CompatibilityEC2,
				},
				Cpu:    ptr.String("256"),
				Memory: ptr.String("512"),
				ContainerDefinitions: []generated.ContainerDefinition{
					containerDef,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(registerResp.TaskDefinition).NotTo(BeNil())
			taskDefArn = *registerResp.TaskDefinition.TaskDefinitionArn
		})

		It("should run task with network configuration", func() {
			// Run task with network configuration
			runTaskResp, err := ecsAPI.RunTask(ctx, &generated.RunTaskRequest{
				TaskDefinition: taskDefArn,
				NetworkConfiguration: &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345", "subnet-67890"},
						SecurityGroups: []string{"sg-12345"},
						AssignPublicIp: (*generated.AssignPublicIp)(ptr.String("ENABLED")),
					},
				},
				LaunchType: (*generated.LaunchType)(ptr.String("EC2")),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskResp.Tasks).To(HaveLen(1))

			task := runTaskResp.Tasks[0]
			Expect(task.TaskArn).NotTo(BeNil())
			Expect(task.LastStatus).To(Equal(ptr.String("PROVISIONING")))

			// Verify container has network interfaces
			Expect(task.Containers).NotTo(BeEmpty())
			container := task.Containers[0]
			Expect(container.NetworkInterfaces).NotTo(BeEmpty())
			Expect(container.NetworkInterfaces[0].PrivateIpv4Address).NotTo(BeNil())

			// Verify attachments include ENI
			Expect(task.Attachments).NotTo(BeEmpty())
			eniAttachment := task.Attachments[0]
			Expect(*eniAttachment.Type).To(Equal("ElasticNetworkInterface"))
			Expect(*eniAttachment.Status).To(Equal("ATTACHED"))
		})

		It("should describe task with network details", func() {
			// Run task first
			runTaskResp, err := ecsAPI.RunTask(ctx, &generated.RunTaskRequest{
				TaskDefinition: taskDefArn,
				NetworkConfiguration: &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345"},
						SecurityGroups: []string{"sg-12345", "sg-67890"},
						AssignPublicIp: (*generated.AssignPublicIp)(ptr.String("DISABLED")),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskResp.Tasks).To(HaveLen(1))

			taskArn := *runTaskResp.Tasks[0].TaskArn

			// Describe task
			describeResp, err := ecsAPI.DescribeTasks(ctx, &generated.DescribeTasksRequest{
				Tasks: []string{taskArn},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(describeResp.Tasks).To(HaveLen(1))

			task := describeResp.Tasks[0]
			Expect(task.Containers).NotTo(BeEmpty())

			// Verify network interfaces are preserved
			container := task.Containers[0]
			Expect(container.NetworkInterfaces).NotTo(BeEmpty())
			Expect(container.NetworkInterfaces[0].AttachmentId).NotTo(BeNil())
			Expect(container.NetworkInterfaces[0].PrivateIpv4Address).NotTo(BeNil())
		})
	})

	Describe("Service with AWSVPC Network Mode", func() {
		var taskDefArn string

		BeforeEach(func() {
			// Register task definition
			containerDef := generated.ContainerDefinition{
				Name:  ptr.String("web"),
				Image: ptr.String("httpd:2.4"),
				PortMappings: []generated.PortMapping{
					{
						ContainerPort: ptr.Int32(80),
						Protocol:      (*generated.TransportProtocol)(ptr.String("tcp")),
					},
				},
				Memory: ptr.Int32(256),
			}

			registerResp, err := ecsAPI.RegisterTaskDefinition(ctx, &generated.RegisterTaskDefinitionRequest{
				Family:      "test-service-awsvpc",
				NetworkMode: (*generated.NetworkMode)(ptr.String("awsvpc")),
				RequiresCompatibilities: []generated.Compatibility{
					generated.CompatibilityEC2,
				},
				Cpu:    ptr.String("256"),
				Memory: ptr.String("512"),
				ContainerDefinitions: []generated.ContainerDefinition{
					containerDef,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			taskDefArn = *registerResp.TaskDefinition.TaskDefinitionArn
		})

		It("should create service with network configuration", func() {
			// Create service with network configuration
			createServiceResp, err := ecsAPI.CreateService(ctx, &generated.CreateServiceRequest{
				ServiceName:    "test-awsvpc-service",
				TaskDefinition: ptr.String(taskDefArn),
				DesiredCount:   ptr.Int32(2),
				LaunchType:     (*generated.LaunchType)(ptr.String("EC2")),
				NetworkConfiguration: &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345", "subnet-67890"},
						SecurityGroups: []string{"sg-web"},
						AssignPublicIp: (*generated.AssignPublicIp)(ptr.String("DISABLED")),
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(createServiceResp.Service).NotTo(BeNil())
			Expect(*createServiceResp.Service.ServiceName).To(Equal("test-awsvpc-service"))
			Expect(*createServiceResp.Service.DesiredCount).To(Equal(int32(2)))

			// Verify network configuration is stored
			Expect(createServiceResp.Service.NetworkConfiguration).NotTo(BeNil())
			Expect(createServiceResp.Service.NetworkConfiguration.AwsvpcConfiguration).NotTo(BeNil())
			Expect(createServiceResp.Service.NetworkConfiguration.AwsvpcConfiguration.Subnets).To(ConsistOf("subnet-12345", "subnet-67890"))
		})

		It("should create service with load balancer", func() {
			// Simulate target group ARN (would come from ELBv2 integration)
			targetGroupArn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:targetgroup/test-targets/1234567890123456", region, accountID)

			createServiceResp, err := ecsAPI.CreateService(ctx, &generated.CreateServiceRequest{
				ServiceName:    "test-lb-service",
				TaskDefinition: ptr.String(taskDefArn),
				DesiredCount:   ptr.Int32(2),
				LaunchType:     (*generated.LaunchType)(ptr.String("EC2")),
				LoadBalancers: []generated.LoadBalancer{
					{
						TargetGroupArn: ptr.String(targetGroupArn),
						ContainerName:  ptr.String("web"),
						ContainerPort:  ptr.Int32(80),
					},
				},
				NetworkConfiguration: &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345", "subnet-67890"},
						SecurityGroups: []string{"sg-web", "sg-alb"},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(createServiceResp.Service).NotTo(BeNil())
			Expect(createServiceResp.Service.LoadBalancers).To(HaveLen(1))
			Expect(*createServiceResp.Service.LoadBalancers[0].TargetGroupArn).To(Equal(targetGroupArn))
		})

		It("should create service with service discovery", func() {
			// Simulate service registry ARN (would come from Cloud Map integration)
			serviceRegistryArn := fmt.Sprintf("arn:aws:servicediscovery:%s:%s:service/srv-12345678", region, accountID)

			createServiceResp, err := ecsAPI.CreateService(ctx, &generated.CreateServiceRequest{
				ServiceName:    "test-sd-service",
				TaskDefinition: ptr.String(taskDefArn),
				DesiredCount:   ptr.Int32(1),
				LaunchType:     (*generated.LaunchType)(ptr.String("EC2")),
				ServiceRegistries: []generated.ServiceRegistry{
					{
						RegistryArn: ptr.String(serviceRegistryArn),
						Port:        ptr.Int32(80),
					},
				},
				NetworkConfiguration: &generated.NetworkConfiguration{
					AwsvpcConfiguration: &generated.AwsVpcConfiguration{
						Subnets:        []string{"subnet-private-1", "subnet-private-2"},
						SecurityGroups: []string{"sg-internal"},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(createServiceResp.Service).NotTo(BeNil())
			Expect(createServiceResp.Service.ServiceRegistries).To(HaveLen(1))
			Expect(*createServiceResp.Service.ServiceRegistries[0].RegistryArn).To(Equal(serviceRegistryArn))
		})
	})

	Describe("Network Mode Validation", func() {
		It("should handle bridge mode without network configuration", func() {
			// Register task definition with bridge mode
			containerDef := generated.ContainerDefinition{
				Name:  ptr.String("app"),
				Image: ptr.String("busybox:latest"),
				PortMappings: []generated.PortMapping{
					{
						ContainerPort: ptr.Int32(8080),
						HostPort:      ptr.Int32(0), // Dynamic port
						Protocol:      (*generated.TransportProtocol)(ptr.String("tcp")),
					},
				},
				Memory: ptr.Int32(128),
			}

			registerResp, err := ecsAPI.RegisterTaskDefinition(ctx, &generated.RegisterTaskDefinitionRequest{
				Family:      "test-bridge",
				NetworkMode: (*generated.NetworkMode)(ptr.String("bridge")),
				ContainerDefinitions: []generated.ContainerDefinition{
					containerDef,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Run task without network configuration
			runTaskResp, err := ecsAPI.RunTask(ctx, &generated.RunTaskRequest{
				TaskDefinition: *registerResp.TaskDefinition.TaskDefinitionArn,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskResp.Tasks).To(HaveLen(1))

			// Bridge mode containers should have network bindings but no interfaces
			task := runTaskResp.Tasks[0]
			Expect(task.Containers).NotTo(BeEmpty())
			container := task.Containers[0]
			Expect(container.NetworkBindings).NotTo(BeEmpty())
			Expect(container.NetworkInterfaces).To(BeEmpty())
		})

		It("should handle host mode", func() {
			// Register task definition with host mode
			containerDef := generated.ContainerDefinition{
				Name:  ptr.String("host-app"),
				Image: ptr.String("nginx:alpine"),
				PortMappings: []generated.PortMapping{
					{
						ContainerPort: ptr.Int32(80),
						Protocol:      (*generated.TransportProtocol)(ptr.String("tcp")),
					},
				},
				Memory: ptr.Int32(128),
			}

			registerResp, err := ecsAPI.RegisterTaskDefinition(ctx, &generated.RegisterTaskDefinitionRequest{
				Family:      "test-host",
				NetworkMode: (*generated.NetworkMode)(ptr.String("host")),
				ContainerDefinitions: []generated.ContainerDefinition{
					containerDef,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskResp, err := ecsAPI.RunTask(ctx, &generated.RunTaskRequest{
				TaskDefinition: *registerResp.TaskDefinition.TaskDefinitionArn,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskResp.Tasks).To(HaveLen(1))
		})
	})
})
