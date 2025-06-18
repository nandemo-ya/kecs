package basic_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("LocalStack Integration", func() {
	var (
		kecs        *utils.KECSContainer
		ecsClient   utils.ECSClientInterface
		clusterName string
	)

	BeforeEach(func() {
		// Start KECS
		kecs = utils.StartKECS(GinkgoT())
		ecsClient = utils.NewCurlClient(kecs.Endpoint())

		// Create test cluster
		clusterName = fmt.Sprintf("localstack-test-%d", time.Now().Unix())
		err := ecsClient.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if ecsClient != nil && clusterName != "" {
			ecsClient.DeleteCluster(clusterName)
		}

		if kecs != nil {
			kecs.Cleanup()
		}
	})

	Describe("LocalStack Status", func() {
		It("should report LocalStack status", func() {
			By("Checking LocalStack status")
			status, err := ecsClient.GetLocalStackStatus(clusterName)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).NotTo(BeEmpty())
		})
	})

	Describe("Task Definitions with LocalStack Services", func() {
		It("should register task definition with S3 integration", func() {
			taskDefJSON := `{
				"family": "s3-task",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [
					{
						"name": "s3-container",
						"image": "amazonlinux:latest",
						"command": ["sh", "-c", "echo 'S3 integration test'"],
						"essential": true,
						"memory": 256,
						"environment": [
							{
								"name": "AWS_DEFAULT_REGION",
								"value": "us-east-1"
							},
							{
								"name": "S3_BUCKET",
								"value": "test-bucket"
							}
						],
						"logConfiguration": {
							"logDriver": "awslogs",
							"options": {
								"awslogs-group": "/ecs/s3-task",
								"awslogs-region": "us-east-1",
								"awslogs-stream-prefix": "s3"
							}
						}
					}
				]
			}`

			By("Registering task definition with S3 configuration")
			taskDef, err := ecsClient.RegisterTaskDefinition("s3-integration", taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef.Family).To(Equal("s3-task"))
		})

		It("should register task definition with SSM secrets", func() {
			taskDefJSON := `{
				"family": "ssm-task",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [
					{
						"name": "ssm-container",
						"image": "amazonlinux:latest",
						"command": ["sh", "-c", "echo 'SSM integration test'"],
						"essential": true,
						"memory": 256,
						"secrets": [
							{
								"name": "DB_PASSWORD",
								"valueFrom": "arn:aws:ssm:us-east-1:123456789012:parameter/test/db-password"
							}
						],
						"logConfiguration": {
							"logDriver": "awslogs",
							"options": {
								"awslogs-group": "/ecs/ssm-task",
								"awslogs-region": "us-east-1",
								"awslogs-stream-prefix": "ssm"
							}
						}
					}
				]
			}`

			By("Registering task definition with SSM secrets")
			taskDef, err := ecsClient.RegisterTaskDefinition("ssm-integration", taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef.Family).To(Equal("ssm-task"))
		})

		It("should register task definition with CloudWatch Logs", func() {
			taskDefJSON := `{
				"family": "logs-task",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [
					{
						"name": "logs-container",
						"image": "nginx:alpine",
						"essential": true,
						"memory": 256,
						"logConfiguration": {
							"logDriver": "awslogs",
							"options": {
								"awslogs-group": "/ecs/integration-test",
								"awslogs-region": "us-east-1",
								"awslogs-stream-prefix": "logs-test",
								"awslogs-create-group": "true"
							}
						}
					}
				]
			}`

			By("Registering task definition with CloudWatch Logs")
			taskDef, err := ecsClient.RegisterTaskDefinition("logs-integration", taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef.Family).To(Equal("logs-task"))
		})
	})

	Describe("Service with LocalStack Integration", func() {
		var taskDefArn string

		BeforeEach(func() {
			// Register a task definition with LocalStack services
			taskDefJSON := `{
				"family": "integration-task",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [
					{
						"name": "integration-container",
						"image": "nginx:alpine",
						"essential": true,
						"memory": 256,
						"environment": [
							{
								"name": "AWS_DEFAULT_REGION",
								"value": "us-east-1"
							}
						],
						"logConfiguration": {
							"logDriver": "awslogs",
							"options": {
								"awslogs-group": "/ecs/integration-service",
								"awslogs-region": "us-east-1",
								"awslogs-stream-prefix": "integration"
							}
						}
					}
				]
			}`

			taskDef, err := ecsClient.RegisterTaskDefinition("integration-test", taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			taskDefArn = taskDef.TaskDefinitionArn
		})

		It("should create service with LocalStack logging", func() {
			serviceName := fmt.Sprintf("localstack-service-%d", time.Now().Unix())

			By("Creating service with LocalStack integration")
			err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service is running")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})

	Describe("Task Execution with LocalStack", func() {
		var taskDefArn string

		BeforeEach(func() {
			// Register a simple task definition
			taskDefJSON := `{
				"family": "run-task-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [
					{
						"name": "test-container",
						"image": "hello-world",
						"essential": true,
						"memory": 256,
						"logConfiguration": {
							"logDriver": "awslogs",
							"options": {
								"awslogs-group": "/ecs/run-task-test",
								"awslogs-region": "us-east-1",
								"awslogs-stream-prefix": "task"
							}
						}
					}
				]
			}`

			taskDef, err := ecsClient.RegisterTaskDefinition("run-task-integration", taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			taskDefArn = taskDef.TaskDefinitionArn
		})

		It("should run task with LocalStack logging", func() {
			By("Running a task")
			response, err := ecsClient.RunTask(clusterName, taskDefArn, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Tasks).To(HaveLen(1))

			taskArn := response.Tasks[0].TaskArn

			By("Verifying task execution")
			Eventually(func() string {
				tasks, err := ecsClient.DescribeTasks(clusterName, []string{taskArn})
				if err != nil || len(tasks) == 0 {
					return ""
				}
				return tasks[0].LastStatus
			}, 60*time.Second, 5*time.Second).Should(Or(Equal("STOPPED"), Equal("RUNNING")))
		})
	})
})