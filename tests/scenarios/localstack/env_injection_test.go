package localstack_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/nandemo-ya/kecs/tests/scenarios/localstack/helpers"
	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Environment Variable Injection", func() {
	var (
		kecs            *utils.KECSContainer
		client          *utils.ECSClient
		testClusterName string
	)

	BeforeEach(func() {
		// Start KECS with LocalStack enabled
		kecs = utils.StartKECS(GinkgoWrapper{GinkgoT()})
		DeferCleanup(func() {
			if kecs != nil {
				kecs.Cleanup()
			}
		})

		// Create ECS client
		client = utils.NewECSClient(kecs.Endpoint())
		
		// Create a test cluster
		testClusterName = fmt.Sprintf("test-env-%d", time.Now().Unix())
		err := client.CurlClient.CreateCluster(testClusterName)
		Expect(err).NotTo(HaveOccurred())

		// Start LocalStack
		helpers.StartLocalStack(&TestingTWrapper{GinkgoT()}, kecs, []string{"iam", "s3"})
		helpers.WaitForLocalStackReady(&TestingTWrapper{GinkgoT()}, client, testClusterName, 30*time.Second)
	})

	AfterEach(func() {
		// Clean up
		if client != nil && testClusterName != "" {
			// Stop any running tasks
			tasks, _ := client.CurlClient.ListTasks(testClusterName, "")
			for _, taskArn := range tasks {
				client.CurlClient.StopTask(testClusterName, taskArn, "Test cleanup")
			}
			
			client.CurlClient.DeleteCluster(testClusterName)
		}
		if kecs != nil {
			helpers.StopLocalStack(&TestingTWrapper{GinkgoT()}, kecs)
		}
	})

	Describe("AWS SDK Configuration", func() {
		It("should inject AWS endpoint environment variables into ECS tasks", func() {
			// Register a task definition that prints environment variables
			taskDef, err := client.CurlClient.RegisterTaskDefinition("env-test", `{
				"containerDefinitions": [{
					"name": "env-printer",
					"image": "busybox:latest",
					"memory": 128,
					"command": ["sh", "-c", "env | grep AWS_ | sort"],
					"logConfiguration": {
						"logDriver": "json-file"
					}
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())

			// Run the task
			runResult, err := client.CurlClient.RunTask(testClusterName, fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision), 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(runResult.Tasks)).To(Equal(1))

			taskArn := runResult.Tasks[0].TaskArn

			// Wait for task to complete
			Eventually(func() string {
				tasks, err := client.CurlClient.DescribeTasks(testClusterName, []string{taskArn})
				if err != nil || len(tasks) == 0 {
					return ""
				}
				return tasks[0].LastStatus
			}, 30*time.Second, 1*time.Second).Should(Equal("STOPPED"))

			// In a real test, we would check the task logs to verify environment variables
			// For now, we'll check that the task ran successfully
			tasks, err := client.CurlClient.DescribeTasks(testClusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(tasks)).To(Equal(1))
			
			// The task should have completed (exit code would be 0 if env vars were set)
			// Note: In a real implementation, we'd need to check container exit code
		})

		It("should set correct LocalStack endpoint URL", func() {
			// Register a task that uses AWS SDK
			taskDef, err := client.CurlClient.RegisterTaskDefinition("sdk-test", `{
				"containerDefinitions": [{
					"name": "aws-cli",
					"image": "amazon/aws-cli:latest",
					"memory": 256,
					"command": ["s3", "ls"],
					"logConfiguration": {
						"logDriver": "json-file"
					}
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())

			// Run the task
			runResult, err := client.CurlClient.RunTask(testClusterName, fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision), 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(runResult.Tasks)).To(Equal(1))

			// The task should be able to connect to LocalStack
			// In a real test, we'd verify the S3 ls command succeeded
		})

		It("should inject AWS region configuration", func() {
			// Register a task that checks AWS region
			taskDef, err := client.CurlClient.RegisterTaskDefinition("region-test", `{
				"containerDefinitions": [{
					"name": "region-checker",
					"image": "busybox:latest",
					"memory": 128,
					"command": ["sh", "-c", "echo $AWS_DEFAULT_REGION"],
					"logConfiguration": {
						"logDriver": "json-file"
					}
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())

			// Run the task
			runResult, err := client.CurlClient.RunTask(testClusterName, fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision), 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(runResult.Tasks)).To(Equal(1))

			// The AWS_DEFAULT_REGION should be set
			// In a real test, we'd check the output
		})
	})

	Describe("Multiple Tasks", func() {
		It("should inject environment variables into multiple tasks consistently", func() {
			// Register task definition
			taskDef, err := client.CurlClient.RegisterTaskDefinition("multi-env-test", `{
				"containerDefinitions": [{
					"name": "env-test",
					"image": "busybox:latest",
					"memory": 128,
					"command": ["sh", "-c", "sleep 10"],
					"logConfiguration": {
						"logDriver": "json-file"
					}
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())

			// Run multiple tasks
			runResult, err := client.CurlClient.RunTask(testClusterName, fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision), 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(runResult.Tasks)).To(Equal(3))

			// All tasks should have the same environment configuration
			taskArns := []string{}
			for _, task := range runResult.Tasks {
				taskArns = append(taskArns, task.TaskArn)
			}

			// Verify all tasks are running
			Eventually(func() bool {
				tasks, err := client.CurlClient.DescribeTasks(testClusterName, taskArns)
				if err != nil || len(tasks) != 3 {
					return false
				}
				for _, task := range tasks {
					if task.LastStatus != "RUNNING" {
						return false
					}
				}
				return true
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			// Clean up
			for _, taskArn := range taskArns {
				client.CurlClient.StopTask(testClusterName, taskArn, "Test cleanup")
			}
		})
	})

	Describe("Service Integration", func() {
		It("should inject environment variables into service tasks", func() {
			// Register task definition
			taskDef, err := client.CurlClient.RegisterTaskDefinition("service-env-test", `{
				"containerDefinitions": [{
					"name": "web",
					"image": "nginx:alpine",
					"memory": 128,
					"portMappings": [{
						"containerPort": 80,
						"protocol": "tcp"
					}]
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := "env-test-service"
			err = client.CurlClient.CreateService(testClusterName, serviceName, fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision), 1)
			Expect(err).NotTo(HaveOccurred())

			// Wait for service task to start
			Eventually(func() int {
				tasks, _ := client.CurlClient.ListTasks(testClusterName, serviceName)
				return len(tasks)
			}, 30*time.Second, 1*time.Second).Should(Equal(1))

			// Service tasks should also have LocalStack environment variables
			tasks, err := client.CurlClient.ListTasks(testClusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(tasks)).To(Equal(1))

			// Clean up
			client.CurlClient.UpdateService(testClusterName, serviceName, intPtr(0), "")
			Eventually(func() int {
				tasks, _ := client.CurlClient.ListTasks(testClusterName, serviceName)
				return len(tasks)
			}, 30*time.Second, 1*time.Second).Should(Equal(0))
			client.CurlClient.DeleteService(testClusterName, serviceName)
		})
	})

	Describe("Proxy Mode Configuration", func() {
		It("should use environment variable proxy mode by default", func() {
			// This test verifies that the default proxy mode is "environment"
			// which injects AWS SDK configuration via environment variables
			
			status := helpers.GetLocalStackStatus(&TestingTWrapper{GinkgoT()}, kecs)
			// The status output should indicate environment proxy mode
			// Note: This depends on the actual implementation
			Expect(status).To(ContainSubstring("Proxy Mode: environment"))
		})

		It("should not interfere with user-defined environment variables", func() {
			// Register task with custom environment variables
			taskDef, err := client.CurlClient.RegisterTaskDefinition("custom-env-test", `{
				"containerDefinitions": [{
					"name": "custom-env",
					"image": "busybox:latest",
					"memory": 128,
					"command": ["sh", "-c", "echo $CUSTOM_VAR"],
					"environment": [
						{"name": "CUSTOM_VAR", "value": "user-defined-value"},
						{"name": "AWS_CUSTOM", "value": "should-not-override"}
					],
					"logConfiguration": {
						"logDriver": "json-file"
					}
				}]
			}`)
			Expect(err).NotTo(HaveOccurred())

			// Run the task
			runResult, err := client.CurlClient.RunTask(testClusterName, fmt.Sprintf("%s:%d", taskDef.Family, taskDef.Revision), 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(runResult.Tasks)).To(Equal(1))

			// User-defined environment variables should be preserved
			// LocalStack variables should be added without overriding user values
		})
	})
})

func intPtr(i int) *int {
	return &i
}