package kecs_test

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/nandemo-ya/kecs/testcontainers-go-kecs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("KECS Container", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("StartContainer", func() {
		Context("with default configuration", func() {
			var container *kecs.Container

			BeforeEach(func() {
				var err error
				container, err = kecs.StartContainer(ctx, kecs.WithTestMode())
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				if container != nil {
					Expect(container.Cleanup(ctx)).To(Succeed())
				}
			})

			It("should provide valid endpoints", func() {
				Expect(container.Endpoint()).NotTo(BeEmpty())
				Expect(container.AdminEndpoint()).NotTo(BeEmpty())
			})

			It("should use default region", func() {
				Expect(container.Region()).To(Equal(kecs.DefaultRegion))
			})

			It("should create a valid ECS client", func() {
				client, err := container.NewECSClient(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(client).NotTo(BeNil())
			})
		})

		Context("with custom configuration", func() {
			It("should apply custom settings", func() {
				container, err := kecs.StartContainer(ctx,
					kecs.WithTestMode(),
					kecs.WithRegion("eu-west-1"),
					kecs.WithAPIPort("9090"),
					kecs.WithAdminPort("9091"),
					kecs.WithEnv(map[string]string{
						"CUSTOM_VAR": "test",
					}),
				)
				Expect(err).NotTo(HaveOccurred())
				defer container.Cleanup(ctx)

				Expect(container.Region()).To(Equal("eu-west-1"))
			})
		})

		Describe("ECS Operations", func() {
			var (
				container *kecs.Container
				client    *ecs.Client
			)

			BeforeEach(func() {
				var err error
				container, err = kecs.StartContainer(ctx, kecs.WithTestMode())
				Expect(err).NotTo(HaveOccurred())

				client, err = container.NewECSClient(ctx)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				if container != nil {
					Expect(container.Cleanup(ctx)).To(Succeed())
				}
			})

			It("should handle cluster operations", func() {
				clusterName := "test-cluster"

				// Create cluster
				createOutput, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
					ClusterName: aws.String(clusterName),
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(aws.ToString(createOutput.Cluster.ClusterName)).To(Equal(clusterName))

				// List clusters
				listOutput, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
				Expect(err).NotTo(HaveOccurred())
				Expect(listOutput.ClusterArns).To(ContainElement(clusterName))

				// Delete cluster
				_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
					Cluster: aws.String(clusterName),
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Describe("Helper Functions", func() {
		var (
			container   *kecs.Container
			client      *ecs.Client
			clusterName string
		)

		BeforeEach(func() {
			var err error
			container, err = kecs.StartContainer(ctx, kecs.WithTestMode())
			Expect(err).NotTo(HaveOccurred())

			client, err = container.NewECSClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Create cluster
			clusterName = "helper-test-cluster"
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

		Describe("CreateTestTaskDefinition", func() {
			It("should create a valid task definition", func() {
				taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "test-family")
				Expect(err).NotTo(HaveOccurred())
				Expect(aws.ToString(taskDef.Family)).To(Equal("test-family"))
				Expect(taskDef.Revision).To(Equal(int32(1)))
				Expect(taskDef.ContainerDefinitions).To(HaveLen(1))
			})
		})

		Describe("CreateTestService", func() {
			It("should create a valid service", func() {
				// Create task definition first
				taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "service-test-family")
				Expect(err).NotTo(HaveOccurred())

				// Create service
				service, err := kecs.CreateTestService(ctx, client, clusterName, "test-service", aws.ToString(taskDef.TaskDefinitionArn))
				Expect(err).NotTo(HaveOccurred())
				Expect(aws.ToString(service.ServiceName)).To(Equal("test-service"))
				Expect(service.DesiredCount).To(Equal(int32(1)))

				// Clean up service
				_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
					Cluster:      aws.String(clusterName),
					Service:      aws.String("test-service"),
					DesiredCount: aws.Int32(0),
				})
				Expect(err).NotTo(HaveOccurred())

				_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
					Cluster: aws.String(clusterName),
					Service: aws.String("test-service"),
				})
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("WaitForCluster", func() {
			It("should successfully wait for an active cluster", func() {
				// Cluster should already be active
				err := kecs.WaitForCluster(ctx, client, clusterName, "ACTIVE", 5*time.Second)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should timeout for non-existent cluster", func() {
				err := kecs.WaitForCluster(ctx, client, "non-existent-cluster", "ACTIVE", 1*time.Second)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("timeout"))
			})
		})
	})
})