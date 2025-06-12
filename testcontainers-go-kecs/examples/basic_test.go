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

var _ = Describe("Basic ECS Operations", func() {
	var (
		ctx       context.Context
		container *kecs.Container
		client    *ecs.Client
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Start KECS container in test mode
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

	Describe("Cluster Operations", func() {
		It("should create and describe a cluster", func() {
			clusterName := "test-cluster"

			// Create cluster
			createOutput, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(aws.ToString(createOutput.Cluster.ClusterName)).To(Equal(clusterName))
			Expect(aws.ToString(createOutput.Cluster.Status)).To(Equal("ACTIVE"))

			// Describe cluster
			describeOutput, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
				Clusters: []string{clusterName},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(describeOutput.Clusters).To(HaveLen(1))
			Expect(aws.ToString(describeOutput.Clusters[0].ClusterName)).To(Equal(clusterName))
		})
	})

	Describe("Task Definition Operations", func() {
		It("should register a task definition", func() {
			// Register task definition
			registerOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family: aws.String("test-task"),
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("nginx"),
						Image:     aws.String("nginx:latest"),
						Memory:    aws.Int32(512),
						Essential: aws.Bool(true),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(aws.ToString(registerOutput.TaskDefinition.Family)).To(Equal("test-task"))
			Expect(registerOutput.TaskDefinition.Revision).To(Equal(int32(1)))
		})
	})

	Describe("Service Operations", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = "service-test-cluster"

			// Create cluster first
			_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			// Clean up cluster
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create and list services", func() {
			// Create task definition
			taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "service-task")
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := "test-service"
			createServiceOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
				Cluster:        aws.String(clusterName),
				ServiceName:    aws.String(serviceName),
				TaskDefinition: taskDef.TaskDefinitionArn,
				DesiredCount:   aws.Int32(2),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(aws.ToString(createServiceOutput.Service.ServiceName)).To(Equal(serviceName))

			// List services
			listOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
				Cluster: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listOutput.ServiceArns).To(HaveLen(1))
		})
	})

	Describe("Cluster Lifecycle", func() {
		It("should manage cluster lifecycle with custom configuration", func() {
			// Start KECS with custom configuration
			customContainer, err := kecs.StartContainer(ctx,
				kecs.WithTestMode(),
				kecs.WithRegion("us-west-2"),
				kecs.WithEnv(map[string]string{
					"LOG_LEVEL": "debug",
				}),
			)
			Expect(err).NotTo(HaveOccurred())
			defer customContainer.Cleanup(ctx)

			// Verify configuration
			Expect(customContainer.Region()).To(Equal("us-west-2"))

			// Create ECS client
			customClient, err := customContainer.NewECSClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			clusterName := "lifecycle-test"

			// Create cluster
			_, err = customClient.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
				Tags: []types.Tag{
					{
						Key:   aws.String("Environment"),
						Value: aws.String("test"),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for cluster to be active
			err = kecs.WaitForCluster(ctx, customClient, clusterName, "ACTIVE", 10*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// List clusters
			listOutput, err := customClient.ListClusters(ctx, &ecs.ListClustersInput{})
			Expect(err).NotTo(HaveOccurred())
			Expect(listOutput.ClusterArns).To(ContainElement(ContainSubstring(clusterName)))

			// Delete cluster
			_, err = customClient.DeleteCluster(ctx, &ecs.DeleteClusterInput{
				Cluster: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())

			// Verify cluster is deleted
			describeOutput, err := customClient.DescribeClusters(ctx, &ecs.DescribeClustersInput{
				Clusters: []string{clusterName},
			})
			Expect(err).NotTo(HaveOccurred())
			if len(describeOutput.Clusters) > 0 {
				Expect(aws.ToString(describeOutput.Clusters[0].Status)).To(Equal("INACTIVE"))
			}
		})
	})
})