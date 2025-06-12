package examples_test

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/testcontainers-go-kecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Advanced Scenarios", func() {
	var (
		ctx         context.Context
		container   *kecs.Container
		client      *ecs.Client
		clusterName string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Start KECS with custom configuration
		var err error
		container, err = kecs.StartContainer(ctx,
			kecs.WithTestMode(),
			kecs.WithRegion("us-east-1"),
			kecs.WithWaitTimeout(2*time.Minute),
			kecs.WithEnv(map[string]string{
				"LOG_LEVEL": "info",
			}),
		)
		Expect(err).NotTo(HaveOccurred())

		// Create ECS client
		client, err = container.NewECSClient(ctx)
		Expect(err).NotTo(HaveOccurred())

		// Create main cluster
		clusterName = "advanced-test-cluster"
		_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: aws.String(clusterName),
			Tags: []types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("test")},
				{Key: aws.String("Team"), Value: aws.String("platform")},
			},
			Settings: []types.ClusterSetting{
				{
					Name:  types.ClusterSettingNameContainerInsights,
					Value: aws.String("enabled"),
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if client != nil && clusterName != "" {
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		}
		if container != nil {
			Expect(container.Cleanup(ctx)).To(Succeed())
		}
	})

	Describe("Microservices with Service Discovery", func() {
		It("should deploy and manage microservices with service discovery", func() {
			// Register API service task definition
			apiTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("api-service"),
				NetworkMode: types.NetworkModeAwsvpc,
				RequiresCompatibilities: []types.Compatibility{
					types.CompatibilityFargate,
				},
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("api"),
						Image:     aws.String("node:14-alpine"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
						Command: []string{
							"sh", "-c",
							"echo 'API server running on port 3000' && node -e 'require(\"http\").createServer((req,res)=>res.end(\"API v1\")).listen(3000)'",
						},
						PortMappings: []types.PortMapping{
							{
								ContainerPort: aws.Int32(3000),
								Protocol:      types.TransportProtocolTcp,
							},
						},
						Environment: []types.KeyValuePair{
							{Name: aws.String("SERVICE_NAME"), Value: aws.String("api")},
							{Name: aws.String("VERSION"), Value: aws.String("1.0.0")},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Register worker service task definition
			workerTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("worker-service"),
				NetworkMode: types.NetworkModeAwsvpc,
				RequiresCompatibilities: []types.Compatibility{
					types.CompatibilityFargate,
				},
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("worker"),
						Image:     aws.String("busybox:latest"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
						Command: []string{
							"sh", "-c",
							"while true; do echo 'Processing jobs...'; sleep 10; done",
						},
						Environment: []types.KeyValuePair{
							{Name: aws.String("SERVICE_NAME"), Value: aws.String("worker")},
							{Name: aws.String("API_ENDPOINT"), Value: aws.String("http://api.local:3000")},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Create API service
			apiService, err := client.CreateService(ctx, &ecs.CreateServiceInput{
				Cluster:        aws.String(clusterName),
				ServiceName:    aws.String("api-service"),
				TaskDefinition: apiTaskDef.TaskDefinition.TaskDefinitionArn,
				DesiredCount:   aws.Int32(2),
				LaunchType:     types.LaunchTypeFargate,
				NetworkConfiguration: &types.NetworkConfiguration{
					AwsvpcConfiguration: &types.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345"},
						SecurityGroups: []string{"sg-12345"},
						AssignPublicIp: types.AssignPublicIpEnabled,
					},
				},
				ServiceRegistries: []types.ServiceRegistry{
					{
						RegistryArn: aws.String("arn:aws:servicediscovery:us-east-1:123456789012:service/srv-123"),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(aws.ToString(apiService.Service.ServiceName)).To(Equal("api-service"))

			// Create worker service
			workerService, err := client.CreateService(ctx, &ecs.CreateServiceInput{
				Cluster:        aws.String(clusterName),
				ServiceName:    aws.String("worker-service"),
				TaskDefinition: workerTaskDef.TaskDefinition.TaskDefinitionArn,
				DesiredCount:   aws.Int32(3),
				LaunchType:     types.LaunchTypeFargate,
				NetworkConfiguration: &types.NetworkConfiguration{
					AwsvpcConfiguration: &types.AwsVpcConfiguration{
						Subnets:        []string{"subnet-12345"},
						SecurityGroups: []string{"sg-12345"},
						AssignPublicIp: types.AssignPublicIpEnabled,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(aws.ToString(workerService.Service.ServiceName)).To(Equal("worker-service"))

			// List all services
			listOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
				Cluster: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listOutput.ServiceArns).To(HaveLen(2))

			// Clean up services
			for _, serviceName := range []string{"api-service", "worker-service"} {
				_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
					Cluster:      aws.String(clusterName),
					Service:      aws.String(serviceName),
					DesiredCount: aws.Int32(0),
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
					Cluster: aws.String(clusterName),
					Service: aws.String(serviceName),
				})
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Describe("Batch Job Processing", func() {
		It("should process multiple batch jobs concurrently", func() {
			// Register batch job task definition
			batchTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("batch-processor"),
				NetworkMode: types.NetworkModeBridge,
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("processor"),
						Image:     aws.String("busybox:latest"),
						Memory:    aws.Int32(256),
						Essential: aws.Bool(true),
						Command: []string{
							"sh", "-c",
							"echo 'Starting batch job'; for i in $(seq 1 5); do echo \"Processing item $i\"; sleep 2; done; echo 'Batch job completed'",
						},
						LogConfiguration: &types.LogConfiguration{
							LogDriver: types.LogDriverAwslogs,
							Options: map[string]string{
								"awslogs-group":         "/ecs/batch-processor",
								"awslogs-region":        "us-east-1",
								"awslogs-stream-prefix": "batch",
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Run multiple batch jobs
			var taskArns []string
			for i := 0; i < 3; i++ {
				runOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
					Cluster:        aws.String(clusterName),
					TaskDefinition: batchTaskDef.TaskDefinition.TaskDefinitionArn,
					Count:          aws.Int32(1),
					Overrides: &types.TaskOverride{
						ContainerOverrides: []types.ContainerOverride{
							{
								Name: aws.String("processor"),
								Environment: []types.KeyValuePair{
									{Name: aws.String("JOB_ID"), Value: aws.String(fmt.Sprintf("job-%d", i+1))},
									{Name: aws.String("BATCH_SIZE"), Value: aws.String("100")},
								},
							},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(runOutput.Tasks).To(HaveLen(1))
				taskArns = append(taskArns, aws.ToString(runOutput.Tasks[0].TaskArn))
			}

			// Monitor batch jobs
			completedJobs := 0
			timeout := time.After(30 * time.Second)
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()

			for completedJobs < len(taskArns) {
				select {
				case <-timeout:
					Fail("Timeout waiting for batch jobs to complete")
				case <-ticker.C:
					describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
						Cluster: aws.String(clusterName),
						Tasks:   taskArns,
					})
					Expect(err).NotTo(HaveOccurred())

					completedJobs = 0
					for _, task := range describeOutput.Tasks {
						if aws.ToString(task.LastStatus) == "STOPPED" {
							completedJobs++
							GinkgoWriter.Printf("Task %s completed with exit code: %d\n",
								aws.ToString(task.TaskArn),
								aws.ToInt32(task.Containers[0].ExitCode))
						}
					}
				}
			}

			Expect(completedJobs).To(Equal(len(taskArns)))
		})
	})
})

var _ = Describe("Complex Service Dependencies", func() {
	var (
		ctx         context.Context
		container   *kecs.Container
		client      *ecs.Client
		clusterName string
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Start KECS with detailed logging
		var err error
		container, err = kecs.StartContainer(ctx,
			kecs.WithTestMode(),
			// Note: Log consumer can be added if needed for debugging
			// kecs.WithLogConsumer(customLogConsumer),
		)
		Expect(err).NotTo(HaveOccurred())

		// Create ECS client
		client, err = container.NewECSClient(ctx)
		Expect(err).NotTo(HaveOccurred())

		// Create cluster
		clusterName = "complex-deps-cluster"
		_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: aws.String(clusterName),
		})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if client != nil && clusterName != "" {
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		}
		if container != nil {
			Expect(container.Cleanup(ctx)).To(Succeed())
		}
	})

	Describe("Application Stack with Dependencies", func() {
		It("should deploy a complex application stack with service dependencies", func() {
			// Create a complex application stack: Database -> API -> Frontend

			// 1. Database service
			dbTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("database"),
				NetworkMode: types.NetworkModeAwsvpc,
				RequiresCompatibilities: []types.Compatibility{
					types.CompatibilityFargate,
				},
				Cpu:    aws.String("512"),
				Memory: aws.String("1024"),
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("postgres"),
						Image:     aws.String("postgres:13-alpine"),
						Memory:    aws.Int32(1024),
						Essential: aws.Bool(true),
						PortMappings: []types.PortMapping{
							{
								ContainerPort: aws.Int32(5432),
								Protocol:      types.TransportProtocolTcp,
							},
						},
						Environment: []types.KeyValuePair{
							{Name: aws.String("POSTGRES_DB"), Value: aws.String("testdb")},
							{Name: aws.String("POSTGRES_USER"), Value: aws.String("testuser")},
							{Name: aws.String("POSTGRES_PASSWORD"), Value: aws.String("testpass")},
						},
						HealthCheck: &types.HealthCheck{
							Command: []string{
								"CMD-SHELL",
								"pg_isready -U testuser",
							},
							Interval:    aws.Int32(30),
							Timeout:     aws.Int32(5),
							Retries:     aws.Int32(3),
							StartPeriod: aws.Int32(60),
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// 2. API service with dependency on database
			apiTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("api"),
				NetworkMode: types.NetworkModeAwsvpc,
				RequiresCompatibilities: []types.Compatibility{
					types.CompatibilityFargate,
				},
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("api"),
						Image:     aws.String("node:14-alpine"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
						Command: []string{
							"sh", "-c",
							"echo 'Waiting for database...' && sleep 10 && echo 'API server starting' && node -e 'require(\"http\").createServer((req,res)=>res.end(JSON.stringify({status:\"ok\",db:\"connected\"}))).listen(8080)'",
						},
						PortMappings: []types.PortMapping{
							{
								ContainerPort: aws.Int32(8080),
								Protocol:      types.TransportProtocolTcp,
							},
						},
						Environment: []types.KeyValuePair{
							{Name: aws.String("DATABASE_HOST"), Value: aws.String("database.local")},
							{Name: aws.String("DATABASE_PORT"), Value: aws.String("5432")},
							{Name: aws.String("DATABASE_NAME"), Value: aws.String("testdb")},
						},
						DependsOn: []types.ContainerDependency{
							{
								ContainerName: aws.String("postgres"),
								Condition:     types.ContainerConditionHealthy,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// 3. Frontend service with dependency on API
			frontendTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("frontend"),
				NetworkMode: types.NetworkModeAwsvpc,
				RequiresCompatibilities: []types.Compatibility{
					types.CompatibilityFargate,
				},
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("nginx"),
						Image:     aws.String("nginx:alpine"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
						PortMappings: []types.PortMapping{
							{
								ContainerPort: aws.Int32(80),
								Protocol:      types.TransportProtocolTcp,
							},
						},
						Environment: []types.KeyValuePair{
							{Name: aws.String("API_URL"), Value: aws.String("http://api.local:8080")},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Create services in dependency order
			services := []struct {
				name           string
				taskDefinition string
				desiredCount   int32
			}{
				{"database-service", *dbTaskDef.TaskDefinition.TaskDefinitionArn, 1},
				{"api-service", *apiTaskDef.TaskDefinition.TaskDefinitionArn, 2},
				{"frontend-service", *frontendTaskDef.TaskDefinition.TaskDefinitionArn, 3},
			}

			for _, svc := range services {
				_, err = client.CreateService(ctx, &ecs.CreateServiceInput{
					Cluster:        aws.String(clusterName),
					ServiceName:    aws.String(svc.name),
					TaskDefinition: aws.String(svc.taskDefinition),
					DesiredCount:   aws.Int32(svc.desiredCount),
					LaunchType:     types.LaunchTypeFargate,
					NetworkConfiguration: &types.NetworkConfiguration{
						AwsvpcConfiguration: &types.AwsVpcConfiguration{
							Subnets:        []string{"subnet-12345"},
							SecurityGroups: []string{"sg-12345"},
							AssignPublicIp: types.AssignPublicIpEnabled,
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				// Wait for service to stabilize
				err = kecs.WaitForService(ctx, client, clusterName, svc.name, "ACTIVE", 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
			}

			// Verify all services are running
			listOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
				Cluster: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listOutput.ServiceArns).To(HaveLen(3))

			// Get service details
			describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  aws.String(clusterName),
				Services: []string{"database-service", "api-service", "frontend-service"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(describeOutput.Services).To(HaveLen(3))

			for _, service := range describeOutput.Services {
				GinkgoWriter.Printf("Service %s: desired=%d, running=%d, pending=%d\n",
					aws.ToString(service.ServiceName),
					service.DesiredCount,
					service.RunningCount,
					service.PendingCount)
				Expect(aws.ToString(service.Status)).To(Equal("ACTIVE"))
			}

			// Clean up in reverse order
			for i := len(services) - 1; i >= 0; i-- {
				_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
					Cluster:      aws.String(clusterName),
					Service:      aws.String(services[i].name),
					DesiredCount: aws.Int32(0),
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
					Cluster: aws.String(clusterName),
					Service: aws.String(services[i].name),
				})
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})
})