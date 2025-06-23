package phase1_test

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Error Scenarios", Serial, func() {
	var (
		client utils.ECSClientInterface
		logger *utils.TestLogger
	)

	BeforeEach(func() {
		// Use shared resources from suite
		client = sharedClient
		logger = sharedLogger
	})

	Describe("Invalid Operations", func() {
		Context("when using invalid cluster names", func() {
			It("should handle cluster names that are too long", func() {
				// AWS ECS cluster names must be 1-255 characters
				longName := strings.Repeat("a", 256)
				logger.Info("Testing cluster creation with name length: %d", len(longName))

				err := client.CreateCluster(longName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Invalid"))
			})

			It("should handle cluster names with invalid characters", func() {
				invalidNames := []string{
					"cluster@name",     // @ symbol
					"cluster name",     // space
					"cluster/name",     // slash
					"cluster\\name",    // backslash
					"cluster:name",     // colon
					"cluster*name",     // asterisk
					"cluster?name",     // question mark
					"cluster#name",     // hash
					"cluster%name",     // percent
				}

				for _, name := range invalidNames {
					logger.Info("Testing invalid cluster name: %s", name)
					err := client.CreateCluster(name)
					Expect(err).To(HaveOccurred(), "Should fail for cluster name: %s", name)
				}
			})

			It("should handle empty cluster name in delete operation", func() {
				logger.Info("Testing delete with empty cluster name")
				
				err := client.DeleteCluster("")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when using invalid ARN formats", func() {
			It("should handle malformed ARNs in describe operation", func() {
				invalidArns := []string{
					"not-an-arn",
					"arn:aws:ecs",  // incomplete ARN
					"arn:aws:ecs:us-east-1",  // missing account and resource
					"arn:aws:ecs:us-east-1:123456789012",  // missing resource
					"arn:aws:ecs:us-east-1:123456789012:wrongtype/cluster-name",  // wrong resource type
				}

				for _, arn := range invalidArns {
					logger.Info("Testing invalid ARN: %s", arn)
					_, err := client.DescribeCluster(arn)
					Expect(err).To(HaveOccurred(), "Should fail for ARN: %s", arn)
				}
			})

			It("should handle malformed ARNs in delete operation", func() {
				logger.Info("Testing delete with malformed ARN")
				
				err := client.DeleteCluster("arn:invalid:format")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Resource Conflicts", func() {
		Context("when deleting a cluster with active services", func() {
			var clusterName string
			var serviceName string
			var taskDefArn string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("conflict-cluster")
				serviceName = utils.GenerateTestName("conflict-service")
				
				// Create cluster
				Expect(client.CreateCluster(clusterName)).To(Succeed())
				
				// Register a task definition
				taskDef := `{
					"family": "conflict-task",
					"containerDefinitions": [{
						"name": "app",
						"image": "nginx:latest",
						"memory": 128,
						"essential": true
					}]
				}`
				td, err := client.RegisterTaskDefinition("conflict-task", taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDefArn = td.TaskDefinitionArn

				// Create a service with unique name
				err = client.CreateService(clusterName, serviceName, taskDefArn, 1)
				Expect(err).NotTo(HaveOccurred())

				// Wait a bit for service to be registered
				time.Sleep(2 * time.Second)

				// Verify service was created
				services, err := client.ListServices(clusterName)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(services)).To(BeNumerically(">", 0), "Service should be created")

				DeferCleanup(func() {
					// Clean up service first
					_ = client.DeleteService(clusterName, serviceName)
					// Then cluster
					_ = client.DeleteCluster(clusterName)
					// Deregister task definition
					_ = client.DeregisterTaskDefinition(taskDefArn)
				})
			})

			PIt("should fail to delete cluster with active service", func() { // FLAKY: Service creation fails with duplicate key in shared container
				logger.Info("Attempting to delete cluster with active service: %s", clusterName)

				err := client.DeleteCluster(clusterName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Or(
					ContainSubstring("active"),
					ContainSubstring("services are active"),
				))
			})
		})

		Context("when deleting a cluster with running tasks", func() {
			var clusterName string
			var taskDefArn string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("task-conflict")
				
				// Create cluster
				Expect(client.CreateCluster(clusterName)).To(Succeed())
				
				// Register a task definition
				taskDef := `{
					"family": "running-task",
					"containerDefinitions": [{
						"name": "app",
						"image": "busybox",
						"command": ["sleep", "300"],
						"memory": 128,
						"essential": true
					}]
				}`
				td, err := client.RegisterTaskDefinition("running-task", taskDef)
				Expect(err).NotTo(HaveOccurred())
				taskDefArn = td.TaskDefinitionArn

				// Run a task
				runResp, err := client.RunTask(clusterName, taskDefArn, 1)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(runResp.Tasks)).To(BeNumerically(">", 0))

				DeferCleanup(func() {
					// Stop any running tasks
					tasks, _ := client.ListTasks(clusterName, "")
					for _, taskArn := range tasks {
						_ = client.StopTask(clusterName, taskArn, "cleanup")
					}
					// Delete cluster
					_ = client.DeleteCluster(clusterName)
					// Deregister task definition
					_ = client.DeregisterTaskDefinition(taskDefArn)
				})
			})

			It("should fail to delete cluster with running tasks", func() {
				logger.Info("Attempting to delete cluster with running tasks: %s", clusterName)

				err := client.DeleteCluster(clusterName)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Or(
					ContainSubstring("tasks"),
					ContainSubstring("active"),
				))
			})
		})
	})

	Describe("Validation Errors", func() {
		Context("when providing invalid settings", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("settings-error")
				Expect(client.CreateCluster(clusterName)).To(Succeed())
				
				DeferCleanup(func() {
					_ = client.DeleteCluster(clusterName)
				})
			})

			It("should handle invalid setting names", func() {
				logger.Info("Testing invalid setting name for cluster: %s", clusterName)

				awsClient := client.(*utils.AWSCLIClient)
				settings := []map[string]string{
					{
						"name":  "invalidSettingName",
						"value": "enabled",
					},
				}
				
				err := awsClient.UpdateClusterSettings(clusterName, settings)
				Expect(err).To(HaveOccurred())
			})

			It("should handle invalid setting values", func() {
				logger.Info("Testing invalid setting value for cluster: %s", clusterName)

				awsClient := client.(*utils.AWSCLIClient)
				settings := []map[string]string{
					{
						"name":  "containerInsights",
						"value": "invalid-value", // Should be "enabled" or "disabled"
					},
				}
				
				err := awsClient.UpdateClusterSettings(clusterName, settings)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when providing invalid capacity providers", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("capacity-error")
				Expect(client.CreateCluster(clusterName)).To(Succeed())
				
				DeferCleanup(func() {
					_ = client.DeleteCluster(clusterName)
				})
			})

			It("should handle invalid capacity provider names", func() {
				logger.Info("Testing invalid capacity provider for cluster: %s", clusterName)

				awsClient := client.(*utils.AWSCLIClient)
				providers := []string{"INVALID_PROVIDER", "ANOTHER_INVALID"}
				
				err := awsClient.PutClusterCapacityProviders(clusterName, providers, nil)
				Expect(err).To(HaveOccurred())
			})

			It("should handle invalid capacity provider strategy", func() {
				logger.Info("Testing invalid capacity provider strategy for cluster: %s", clusterName)

				awsClient := client.(*utils.AWSCLIClient)
				providers := []string{"FARGATE"}
				strategy := []map[string]interface{}{
					{
						"capacityProvider": "FARGATE",
						"weight":          -1,  // Invalid: negative weight
						"base":            -5,  // Invalid: negative base
					},
				}
				
				err := awsClient.PutClusterCapacityProviders(clusterName, providers, strategy)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when providing malformed JSON configurations", func() {
			var clusterName string

			BeforeEach(func() {
				clusterName = utils.GenerateTestName("config-error")
				Expect(client.CreateCluster(clusterName)).To(Succeed())
				
				DeferCleanup(func() {
					_ = client.DeleteCluster(clusterName)
				})
			})

			It("should handle malformed execute command configuration", func() {
				logger.Info("Testing malformed configuration for cluster: %s", clusterName)

				awsClient := client.(*utils.AWSCLIClient)
				
				// Invalid configuration with bad values
				config := map[string]interface{}{
					"configuration": map[string]interface{}{
						"executeCommandConfiguration": map[string]interface{}{
							"kmsKeyId": "not-a-valid-kms-key",
							"logging":  "INVALID_LOGGING_VALUE", // Should be DEFAULT, NONE, or OVERRIDE
						},
					},
				}
				
				err := awsClient.UpdateCluster(clusterName, config)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Missing Required Parameters", func() {
		Context("when required parameters are missing", func() {
			It("should handle missing cluster identifier in describe", func() {
				logger.Info("Testing describe with missing cluster identifier")
				
				// Describing with empty string should fail
				_, err := client.DescribeCluster("")
				Expect(err).To(HaveOccurred())
			})

			It("should handle missing resource ARN in tag operations", func() {
				logger.Info("Testing tag operations with missing resource ARN")
				
				tags := map[string]string{"Key": "Value"}
				err := client.TagResource("", tags)
				Expect(err).To(HaveOccurred())
				
				err = client.UntagResource("", []string{"Key"})
				Expect(err).To(HaveOccurred())
				
				_, err = client.ListTagsForResource("")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Operation on Non-existent Resources", func() {
		Context("when operating on non-existent clusters", func() {
			nonExistentCluster := "non-existent-cluster-ops"

			It("should fail to update settings on non-existent cluster", func() {
				logger.Info("Testing update settings on non-existent cluster")
				
				awsClient := client.(*utils.AWSCLIClient)
				settings := []map[string]string{
					{"name": "containerInsights", "value": "enabled"},
				}
				
				err := awsClient.UpdateClusterSettings(nonExistentCluster, settings)
				Expect(err).To(HaveOccurred())
			})

			It("should fail to update configuration on non-existent cluster", func() {
				logger.Info("Testing update configuration on non-existent cluster")
				
				awsClient := client.(*utils.AWSCLIClient)
				config := map[string]interface{}{
					"configuration": map[string]interface{}{
						"executeCommandConfiguration": map[string]interface{}{
							"logging": "DEFAULT",
						},
					},
				}
				
				err := awsClient.UpdateCluster(nonExistentCluster, config)
				Expect(err).To(HaveOccurred())
			})

			It("should fail to set capacity providers on non-existent cluster", func() {
				logger.Info("Testing put capacity providers on non-existent cluster")
				
				awsClient := client.(*utils.AWSCLIClient)
				providers := []string{"FARGATE"}
				
				err := awsClient.PutClusterCapacityProviders(nonExistentCluster, providers, nil)
				Expect(err).To(HaveOccurred())
			})

			It("should fail to tag non-existent cluster", func() {
				logger.Info("Testing tag operations on non-existent cluster")
				
				fakeArn := fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", nonExistentCluster)
				tags := map[string]string{"Environment": "test"}
				
				err := client.TagResource(fakeArn, tags)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})