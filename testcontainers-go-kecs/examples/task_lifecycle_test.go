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
	"github.com/testcontainers/testcontainers-go"
)

var _ = Describe("Task Lifecycle", func() {
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

	Describe("Task Operations", func() {
		var (
			clusterName      string
			taskDefOutput    *ecs.RegisterTaskDefinitionOutput
		)

		BeforeEach(func() {
			clusterName = "task-test-cluster"

			// Setup cluster
			_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())

			// Register task definition
			taskDefOutput, err = client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("lifecycle-task"),
				NetworkMode: types.NetworkModeBridge,
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:  aws.String("app"),
						Image: aws.String("busybox:latest"),
						Command: []string{
							"sh",
							"-c",
							"echo 'Starting task' && sleep 30 && echo 'Task completed'",
						},
						Memory:    aws.Int32(128),
						Essential: aws.Bool(true),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should run and stop a task", func() {
			// Run task
			runTaskOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(clusterName),
				TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
				Count:          aws.Int32(1),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskOutput.Tasks).To(HaveLen(1))

			taskArn := aws.ToString(runTaskOutput.Tasks[0].TaskArn)

			// Wait for task to be running
			err = kecs.WaitForTask(ctx, client, clusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Describe task
			describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(clusterName),
				Tasks:   []string{taskArn},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(describeOutput.Tasks).To(HaveLen(1))
			Expect(aws.ToString(describeOutput.Tasks[0].LastStatus)).To(Equal("RUNNING"))

			// Stop task
			stopOutput, err := client.StopTask(ctx, &ecs.StopTaskInput{
				Cluster: aws.String(clusterName),
				Task:    aws.String(taskArn),
				Reason:  aws.String("Test completed"),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(aws.ToString(stopOutput.Task.TaskArn)).To(Equal(taskArn))

			// Wait for task to be stopped
			err = kecs.WaitForTask(ctx, client, clusterName, taskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should track task status transitions", func() {
			// Register a quick task definition
			quickTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("quick-task"),
				NetworkMode: types.NetworkModeBridge,
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:  aws.String("quick"),
						Image: aws.String("busybox:latest"),
						Command: []string{
							"sh",
							"-c",
							"echo 'Quick task' && sleep 5",
						},
						Memory:    aws.Int32(128),
						Essential: aws.Bool(true),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(clusterName),
				TaskDefinition: quickTaskDef.TaskDefinition.TaskDefinitionArn,
				Count:          aws.Int32(1),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskOutput.Tasks).To(HaveLen(1))

			taskArn := aws.ToString(runTaskOutput.Tasks[0].TaskArn)

			// Track status transitions
			var statuses []string
			lastStatus := ""

			Eventually(func() bool {
				describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
					Cluster: aws.String(clusterName),
					Tasks:   []string{taskArn},
				})
				Expect(err).NotTo(HaveOccurred())

				if len(describeOutput.Tasks) > 0 {
					currentStatus := aws.ToString(describeOutput.Tasks[0].LastStatus)
					if currentStatus != lastStatus {
						statuses = append(statuses, currentStatus)
						lastStatus = currentStatus
						GinkgoWriter.Printf("Task status: %s\n", currentStatus)
					}

					if currentStatus == "STOPPED" {
						return true
					}
				}
				return false
			}, 20*time.Second, 1*time.Second).Should(BeTrue())

			// Verify we saw expected status transitions
			Expect(statuses).To(ContainElement("PENDING"))
			Expect(statuses).To(ContainElement("RUNNING"))
			Expect(statuses).To(ContainElement("STOPPED"))
		})
	})

	Describe("Multi-Container Tasks", func() {
		var clusterName string

		BeforeEach(func() {
			clusterName = "multi-container-cluster"

			// Setup cluster
			_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(clusterName),
			})
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := kecs.CleanupCluster(ctx, client, clusterName)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should run a task with multiple containers", func() {
			// Start KECS container with logging
			logContainer, err := kecs.StartContainer(ctx,
				kecs.WithTestMode(),
				kecs.WithLogConsumer(&kecsLogConsumer{}),
			)
			Expect(err).NotTo(HaveOccurred())
			defer logContainer.Cleanup(ctx)

			// Create ECS client
			logClient, err := logContainer.NewECSClient(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Create cluster for this test
			logClusterName := "log-multi-container-cluster"
			_, err = logClient.CreateCluster(ctx, &ecs.CreateClusterInput{
				ClusterName: aws.String(logClusterName),
			})
			Expect(err).NotTo(HaveOccurred())
			defer kecs.CleanupCluster(ctx, logClient, logClusterName)

			// Register task definition with multiple containers
			taskDefOutput, err := logClient.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
				Family:      aws.String("multi-container-task"),
				NetworkMode: types.NetworkModeBridge,
				ContainerDefinitions: []types.ContainerDefinition{
					{
						Name:      aws.String("web"),
						Image:     aws.String("nginx:alpine"),
						Memory:    aws.Int32(256),
						Essential: aws.Bool(true),
						PortMappings: []types.PortMapping{
							{
								ContainerPort: aws.Int32(80),
								HostPort:      aws.Int32(8080),
								Protocol:      types.TransportProtocolTcp,
							},
						},
					},
					{
						Name:  aws.String("sidecar"),
						Image: aws.String("busybox:latest"),
						Command: []string{
							"sh",
							"-c",
							"while true; do echo 'Sidecar running'; sleep 10; done",
						},
						Memory:    aws.Int32(128),
						Essential: aws.Bool(false),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskOutput, err := logClient.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(logClusterName),
				TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
				Count:          aws.Int32(1),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(runTaskOutput.Tasks).To(HaveLen(1))

			taskArn := aws.ToString(runTaskOutput.Tasks[0].TaskArn)

			// Wait for task to be running
			err = kecs.WaitForTask(ctx, logClient, logClusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Verify both containers are running
			describeOutput, err := logClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(logClusterName),
				Tasks:   []string{taskArn},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(describeOutput.Tasks).To(HaveLen(1))

			task := describeOutput.Tasks[0]
			Expect(task.Containers).To(HaveLen(2))

			// Verify container statuses
			for _, container := range task.Containers {
				GinkgoWriter.Printf("Container %s status: %s\n", aws.ToString(container.Name), aws.ToString(container.LastStatus))
				Expect(aws.ToString(container.LastStatus)).To(Equal("RUNNING"))
			}

			// Clean up
			_, err = logClient.StopTask(ctx, &ecs.StopTaskInput{
				Cluster: aws.String(logClusterName),
				Task:    aws.String(taskArn),
				Reason:  aws.String("Test cleanup"),
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

// kecsLogConsumer implements testcontainers.LogConsumer interface
type kecsLogConsumer struct{}

func (lc *kecsLogConsumer) Accept(log testcontainers.Log) {
	GinkgoWriter.Printf("KECS: %s", log.Content)
}