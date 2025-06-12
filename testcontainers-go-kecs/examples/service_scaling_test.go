package examples_test

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/testcontainers-go-kecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service Scaling", func() {
	var (
		ctx       context.Context
		container *kecs.Container
		client    *ecs.Client
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Start KECS container
		var err error
		container, err = kecs.StartContainer(ctx, kecs.WithTestMode())
		Expect(err).NotTo(HaveOccurred())

		// Create ECS client
		client, err = container.NewECSClient(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if container != nil {
			Expect(container.Cleanup(ctx)).To(Succeed())
		}
	})

	Describe("Service Scaling Operations", func() {
		var (
			clusterName string
			taskDef     *types.TaskDefinition
		)

		BeforeEach(func() {
			clusterName = "scaling-test-cluster"

			// Create cluster
			_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())

			// Register task definition
			taskDef, err = kecs.CreateTestTaskDefinition(ctx, client, "scaling-task")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up cluster
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when scaling up and down", func() {
			It("should update desired count correctly", func() {
				serviceName := "scaling-service"

				// Create service with 1 task
				createOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
					Cluster:        aws.String(clusterName),
					ServiceName:    aws.String(serviceName),
					TaskDefinition: taskDef.TaskDefinitionArn,
					DesiredCount:   aws.Int32(1),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createOutput.Service.DesiredCount).To(Equal(int32(1)))

				// Wait for service to be active
				err = kecs.WaitForService(ctx, client, clusterName, serviceName, "ACTIVE", 30*time.Second)
				Expect(err).NotTo(HaveOccurred())

				// Scale up to 3 tasks
				updateOutput, err := client.UpdateService(ctx, &ecs.UpdateServiceInput{
					Cluster:      aws.String(clusterName),
					Service:      aws.String(serviceName),
					DesiredCount: aws.Int32(3),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(updateOutput.Service.DesiredCount).To(Equal(int32(3)))

				// Wait a bit for scaling to take effect
				time.Sleep(2 * time.Second)

				// Verify running count
				describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
					Cluster:  aws.String(clusterName),
					Services: []string{serviceName},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(describeOutput.Services).To(HaveLen(1))
				Expect(describeOutput.Services[0].DesiredCount).To(Equal(int32(3)))

				// Scale down to 0
				updateOutput, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
					Cluster:      aws.String(clusterName),
					Service:      aws.String(serviceName),
					DesiredCount: aws.Int32(0),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(updateOutput.Service.DesiredCount).To(Equal(int32(0)))

				// Delete service
				_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
					Cluster: aws.String(clusterName),
					Service: aws.String(serviceName),
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when using placement strategies", func() {
			It("should apply placement strategies and constraints", func() {
				serviceName := "placement-service"

				// Create service with placement strategies
				createOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
					Cluster:        aws.String(clusterName),
					ServiceName:    aws.String(serviceName),
					TaskDefinition: taskDef.TaskDefinitionArn,
					DesiredCount:   aws.Int32(2),
					PlacementStrategy: []types.PlacementStrategy{
						{
							Type:  types.PlacementStrategyTypeSpread,
							Field: aws.String("attribute:ecs.availability-zone"),
						},
					},
					PlacementConstraints: []types.PlacementConstraint{
						{
							Type:       types.PlacementConstraintTypeDistinctInstance,
							Expression: aws.String(""),
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createOutput.Service.DesiredCount).To(Equal(int32(2)))
				Expect(createOutput.Service.PlacementStrategy).To(HaveLen(1))
				Expect(createOutput.Service.PlacementConstraints).To(HaveLen(1))

				// Clean up
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
			})
		})
	})

	Describe("Service Deployment Configuration", func() {
		var (
			clusterName string
			v1Output    *ecs.RegisterTaskDefinitionOutput
		)

		BeforeEach(func() {
			clusterName = "deployment-test-cluster"

			// Create cluster
			_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())

			// Register initial task definition
			v1Output, err = client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("deployment-task"),
				NetworkMode: types.NetworkModeBridge,
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("app"),
						Image:     aws.String("nginx:1.19"),
						Memory:    aws.Int32(256),
						Essential: aws.Bool(true),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up cluster
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when performing rolling update deployment", func() {
			It("should update service with new task definition version", func() {
				serviceName := "rolling-update-service"

				// Create service with custom deployment configuration
				createOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
					Cluster:        aws.String(clusterName),
					ServiceName:    aws.String(serviceName),
					TaskDefinition: v1Output.TaskDefinition.TaskDefinitionArn,
					DesiredCount:   aws.Int32(3),
					DeploymentConfiguration: &types.DeploymentConfiguration{
						MaximumPercent:        aws.Int32(200),
						MinimumHealthyPercent: aws.Int32(50),
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(createOutput.Service.DeploymentConfiguration.MaximumPercent).To(Equal(int32(200)))
				Expect(createOutput.Service.DeploymentConfiguration.MinimumHealthyPercent).To(Equal(int32(50)))

				// Wait for initial deployment
				err = kecs.WaitForService(ctx, client, clusterName, serviceName, "ACTIVE", 30*time.Second)
				Expect(err).NotTo(HaveOccurred())

				// Register new version of task definition
				v2Output, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
					Family:      aws.String("deployment-task"),
					NetworkMode: types.NetworkModeBridge,
					ContainerDefinitions: []types.ContainerDefinition{
						{
							Name:      aws.String("app"),
							Image:     aws.String("nginx:1.20"), // Updated version
							Memory:    aws.Int32(256),
							Essential: aws.Bool(true),
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				// Update service to use new task definition
				updateOutput, err := client.UpdateService(ctx, &ecs.UpdateServiceInput{
					Cluster:        aws.String(clusterName),
					Service:        aws.String(serviceName),
					TaskDefinition: v2Output.TaskDefinition.TaskDefinitionArn,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(aws.ToString(updateOutput.Service.TaskDefinition)).To(ContainSubstring("deployment-task:2"))

				// Check deployments
				describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
					Cluster:  aws.String(clusterName),
					Services: []string{serviceName},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(describeOutput.Services).To(HaveLen(1))

				service := describeOutput.Services[0]
				GinkgoWriter.Printf("Service has %d deployments\n", len(service.Deployments))
				for _, deployment := range service.Deployments {
					GinkgoWriter.Printf("Deployment: %s, Status: %s, Running: %d, Pending: %d, Desired: %d\n",
						aws.ToString(deployment.Id),
						aws.ToString(deployment.Status),
						deployment.RunningCount,
						deployment.PendingCount,
						deployment.DesiredCount,
					)
				}

				// Clean up
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
			})
		})
	})
})