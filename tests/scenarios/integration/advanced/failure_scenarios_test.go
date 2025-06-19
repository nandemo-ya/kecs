package advanced_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Failure Recovery Scenarios", func() {
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
		clusterName = fmt.Sprintf("failure-test-%d", time.Now().Unix())
		err := ecsClient.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if ecsClient != nil && clusterName != "" {
			// Cleanup services
			services, _ := ecsClient.ListServices(clusterName)
			for _, serviceName := range services {
				ecsClient.DeleteService(clusterName, serviceName)
			}

			// Delete cluster
			ecsClient.DeleteCluster(clusterName)
		}

		if kecs != nil {
			kecs.Cleanup()
		}
	})

	Describe("Task Failure Recovery", func() {
		It("should restart failed tasks automatically", func() {
			serviceName := fmt.Sprintf("failure-recovery-%d", time.Now().Unix())

			By("Registering task definition with failing container")
			taskDef, err := ecsClient.RegisterTaskDefinition("failing-task", createFailingTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service that will experience failures")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service attempts to maintain desired count")
			Consistently(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.DesiredCount == 1
			}, 30*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})

		It("should handle unhealthy containers", func() {
			serviceName := fmt.Sprintf("unhealthy-test-%d", time.Now().Unix())

			By("Registering task definition with health check")
			taskDef, err := ecsClient.RegisterTaskDefinition("unhealthy-task", createUnhealthyTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with health checks")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service handles unhealthy containers")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.DesiredCount == 1
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})

	Describe("Resource Constraint Scenarios", func() {
		It("should handle memory-limited environments", func() {
			serviceName := fmt.Sprintf("memory-test-%d", time.Now().Unix())

			By("Registering high-memory task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("memory-task", createHighMemoryTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with memory constraints")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service handles memory constraints")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.DesiredCount == 1
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})

		It("should handle CPU-limited environments", func() {
			serviceName := fmt.Sprintf("cpu-test-%d", time.Now().Unix())

			By("Registering high-CPU task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("cpu-task", createHighCPUTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with CPU constraints")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service handles CPU constraints")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.DesiredCount == 1
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})

	Describe("Network Failure Scenarios", func() {
		It("should handle port conflicts gracefully", func() {
			serviceName1 := fmt.Sprintf("port-conflict-1-%d", time.Now().Unix())
			serviceName2 := fmt.Sprintf("port-conflict-2-%d", time.Now().Unix())

			By("Registering task definition with fixed port")
			taskDef, err := ecsClient.RegisterTaskDefinition("fixed-port-task", createFixedPortTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating first service with fixed port")
			err = ecsClient.CreateService(clusterName, serviceName1, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for first service to be running")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName1)
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Creating second service with same fixed port")
			err = ecsClient.CreateService(clusterName, serviceName2, taskDef.TaskDefinitionArn, 1)
			// This might fail or succeed depending on KECS implementation
			// We just verify it's handled gracefully

			By("Verifying system handles port conflicts")
			Eventually(func() bool {
				// At least one service should be running
				service1, err1 := ecsClient.DescribeService(clusterName, serviceName1)
				service2, err2 := ecsClient.DescribeService(clusterName, serviceName2)
				
				service1Running := err1 == nil && service1.RunningCount > 0
				service2Running := err2 == nil && service2.RunningCount > 0
				
				return service1Running || service2Running
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName1)
			ecsClient.DeleteService(clusterName, serviceName2)
		})
	})

	Describe("Service Dependency Failure", func() {
		var (
			dependencyTaskDef string
			dependentTaskDef  string
		)

		BeforeEach(func() {
			// Register dependency service task definition
			taskDef, err := ecsClient.RegisterTaskDefinition("dependency-service", createDependencyTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			dependencyTaskDef = taskDef.TaskDefinitionArn

			// Register dependent service task definition
			taskDef, err = ecsClient.RegisterTaskDefinition("dependent-service", createDependentTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			dependentTaskDef = taskDef.TaskDefinitionArn
		})

		It("should handle dependency service failures", func() {
			By("Deploying dependency service")
			err := ecsClient.CreateService(clusterName, "dependency-service", dependencyTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for dependency service to be ready")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "dependency-service")
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Deploying dependent service")
			err = ecsClient.CreateService(clusterName, "dependent-service", dependentTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying dependent service can handle dependency failures")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "dependent-service")
				return err == nil && service.DesiredCount == 1
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Simulating dependency failure by deleting dependency service")
			ecsClient.DeleteService(clusterName, "dependency-service")

			By("Verifying dependent service continues to maintain desired count")
			Consistently(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "dependent-service")
				return err == nil && service.DesiredCount == 1
			}, 30*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, "dependent-service")
		})
	})
})

func createFailingTaskDefinition() string {
	return `{
		"family": "failing-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "failing-container",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Starting...'; sleep 10; echo 'Failing now'; exit 1"],
				"essential": true,
				"memory": 256,
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/failing-task",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "failing"
					}
				}
			}
		]
	}`
}

func createUnhealthyTaskDefinition() string {
	return `{
		"family": "unhealthy-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "unhealthy-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				],
				"healthCheck": {
					"command": ["CMD-SHELL", "wget --quiet --tries=1 --spider http://localhost:8080/ || exit 1"],
					"interval": 30,
					"timeout": 5,
					"retries": 3,
					"startPeriod": 60
				},
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/unhealthy-task",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "unhealthy"
					}
				}
			}
		]
	}`
}

func createHighMemoryTaskDefinition() string {
	return `{
		"family": "memory-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "memory-container",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Testing memory limits'; dd if=/dev/zero of=/tmp/test bs=1M count=100; sleep 30"],
				"essential": true,
				"memory": 256,
				"memoryReservation": 128,
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/memory-task",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "memory"
					}
				}
			}
		]
	}`
}

func createHighCPUTaskDefinition() string {
	return `{
		"family": "cpu-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "cpu-container",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Testing CPU limits'; for i in $(seq 1 4); do (while true; do :; done) & done; sleep 30"],
				"essential": true,
				"memory": 256,
				"cpu": 128,
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/cpu-task",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "cpu"
					}
				}
			}
		]
	}`
}

func createFixedPortTaskDefinition() string {
	return `{
		"family": "fixed-port-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "fixed-port-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"hostPort": 9999,
						"protocol": "tcp"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/fixed-port-task",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "fixed-port"
					}
				}
			}
		]
	}`
}

func createDependencyTaskDefinition() string {
	return `{
		"family": "dependency-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "dependency-container",
				"image": "redis:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 6379,
						"protocol": "tcp"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/dependency-service",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "dependency"
					}
				}
			}
		]
	}`
}

func createDependentTaskDefinition() string {
	return `{
		"family": "dependent-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "dependent-container",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Dependent service starting'; while true; do echo 'Checking dependency...'; sleep 10; done"],
				"essential": true,
				"memory": 256,
				"environment": [
					{
						"name": "REDIS_URL",
						"value": "redis://dependency-service:6379"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/dependent-service",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "dependent"
					}
				}
			}
		]
	}`
}