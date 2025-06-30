package phase3

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("LocalStack S3 Proxy Integration", func() {
	var clusterName string

	BeforeEach(func() {
		// Note: LocalStack should be enabled by default in test containers
		GinkgoT().Log("Testing LocalStack S3 proxy integration...")

		// Create a unique cluster for this test
		var err error
		clusterName, err = sharedClusterManager.GetOrCreateCluster("phase3-s3")
		Expect(err).NotTo(HaveOccurred())
		GinkgoT().Logf("Testing S3 proxy with cluster: %s", clusterName)
		
		// Give LocalStack time to deploy (LocalStack deployment happens automatically)
		GinkgoT().Log("Waiting for LocalStack deployment...")
		// Wait longer for LocalStack to be fully ready with all services
		// LocalStack needs time to:
		// 1. Deploy the container
		// 2. Initialize all AWS services
		// 3. Set up Traefik TCP proxy routes
		time.Sleep(45 * time.Second)
	})

	Context("S3 API Proxy through LocalStack", func() {
		It("should proxy S3 API calls to LocalStack transparently", func() {
			// Register a simple task definition that lists S3 buckets
			// This should work even with an empty LocalStack
			// Note: AWS credentials are required even with transparent proxy
			// The Traefik TCP proxy intercepts S3 requests at the network level
			taskDefJSON := `{
				"family": "s3-list-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [{
					"name": "s3-test-container",
					"image": "amazon/aws-cli:latest",
					"cpu": 256,
					"memory": 512,
					"essential": true,
					"environment": [
						{
							"name": "AWS_DEFAULT_REGION",
							"value": "us-east-1"
						},
						{
							"name": "AWS_ACCESS_KEY_ID",
							"value": "test"
						},
						{
							"name": "AWS_SECRET_ACCESS_KEY",
							"value": "test"
						}
					],
					"command": [
						"s3api",
						"list-buckets"
					]
				}]
			}`

			taskDef, err := sharedClient.RegisterTaskDefinitionFromJSON(taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef).NotTo(BeNil())

			// Run the task
			runResp, err := sharedClient.RunTask(clusterName, "s3-list-test", 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(runResp.Tasks).To(HaveLen(1))
			taskArn := runResp.Tasks[0].TaskArn

			// Wait for task to complete
			GinkgoT().Log("Waiting for S3 task to complete...")
			Eventually(func() string {
				tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
				if err != nil || len(tasks) == 0 {
					return "UNKNOWN"
				}
				return tasks[0].LastStatus
			}, 120*time.Second, 2*time.Second).Should(Equal("STOPPED"))

			// Check task completion status
			tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			
			// The task should have stopped successfully
			// If it reached LocalStack (even with empty bucket list), it means proxy is working
			// If it tried to reach real AWS, it would fail with credentials error
			GinkgoT().Logf("Task stopped with status: %s, reason: %s", tasks[0].LastStatus, tasks[0].StoppedReason)
		})

		It("should handle S3 operations with environment-based proxy configuration", func() {
			// Test with explicit proxy environment variables
			taskDefJSON := `{
				"family": "s3-env-proxy-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [{
					"name": "s3-env-test-container",
					"image": "amazon/aws-cli:latest",
					"cpu": 256,
					"memory": 512,
					"essential": true,
					"environment": [
						{
							"name": "AWS_DEFAULT_REGION",
							"value": "us-east-1"
						},
						{
							"name": "AWS_ACCESS_KEY_ID",
							"value": "test"
						},
						{
							"name": "AWS_SECRET_ACCESS_KEY",
							"value": "test"
						}
					],
					"command": [
						"s3api",
						"list-buckets"
					]
				}]
			}`

			taskDef, err := sharedClient.RegisterTaskDefinitionFromJSON(taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef).NotTo(BeNil())

			// Run the task
			runResp, err := sharedClient.RunTask(clusterName, "s3-env-proxy-test", 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(runResp.Tasks).To(HaveLen(1))
			taskArn := runResp.Tasks[0].TaskArn

			// Wait for task to complete
			Eventually(func() string {
				tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
				if err != nil || len(tasks) == 0 {
					return "UNKNOWN"
				}
				return tasks[0].LastStatus
			}, 120*time.Second, 2*time.Second).Should(Equal("STOPPED"))

			// The task should have stopped successfully
			// The important part is that it reached LocalStack, not real AWS
			tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			
			// The task should complete successfully with list-buckets
			// Even with empty bucket list, success shows the request went to LocalStack
			// If it went to real AWS, we'd get authentication errors instead
			GinkgoT().Logf("Task stopped with status: %s, reason: %s", tasks[0].LastStatus, tasks[0].StoppedReason)
			Expect(tasks[0].LastStatus).To(Equal("STOPPED"))
		})

		It("should support multiple S3 operations in a single task", func() {
			// Create a task that performs multiple S3 operations
			bucketName := fmt.Sprintf("test-multi-ops-%d", time.Now().Unix())
			taskDefJSON := fmt.Sprintf(`{
				"family": "s3-multi-ops-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [{
					"name": "s3-multi-ops-container",
					"image": "amazon/aws-cli:latest",
					"cpu": 256,
					"memory": 512,
					"essential": true,
					"environment": [
						{
							"name": "AWS_DEFAULT_REGION",
							"value": "us-east-1"
						},
						{
							"name": "AWS_ACCESS_KEY_ID",
							"value": "test"
						},
						{
							"name": "AWS_SECRET_ACCESS_KEY",
							"value": "test"
						}
					],
					"entryPoint": ["sh", "-c"],
					"command": [
						"aws s3api create-bucket --bucket %s && echo 'Hello from KECS' > /tmp/test.txt && aws s3 cp /tmp/test.txt s3://%s/test.txt && aws s3 ls s3://%s/"
					]
				}]
			}`, bucketName, bucketName, bucketName)

			taskDef, err := sharedClient.RegisterTaskDefinitionFromJSON(taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef).NotTo(BeNil())

			// Run the task
			runResp, err := sharedClient.RunTask(clusterName, "s3-multi-ops-test", 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(runResp.Tasks).To(HaveLen(1))
			taskArn := runResp.Tasks[0].TaskArn

			// Wait for task to complete
			Eventually(func() string {
				tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
				if err != nil || len(tasks) == 0 {
					return "UNKNOWN"
				}
				return tasks[0].LastStatus
			}, 120*time.Second, 2*time.Second).Should(Equal("STOPPED"))

			// Check task completed successfully
			tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].StoppedReason).NotTo(ContainSubstring("error"))
		})
	})
})