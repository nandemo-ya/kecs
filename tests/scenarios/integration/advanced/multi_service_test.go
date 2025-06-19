package advanced_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Multi-Service Scenarios", func() {
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
		clusterName = fmt.Sprintf("multi-service-test-%d", time.Now().Unix())
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

	Describe("Multi-Tier Application Deployment", func() {
		var (
			frontendTaskDef string
			backendTaskDef  string
			databaseTaskDef string
		)

		BeforeEach(func() {
			// Register task definitions for a multi-tier application
			var err error
			var taskDef *utils.TaskDefinition

			// Frontend task definition
			taskDef, err = ecsClient.RegisterTaskDefinition("frontend", createFrontendTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			frontendTaskDef = taskDef.TaskDefinitionArn

			// Backend task definition
			taskDef, err = ecsClient.RegisterTaskDefinition("backend", createBackendTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			backendTaskDef = taskDef.TaskDefinitionArn

			// Database task definition
			taskDef, err = ecsClient.RegisterTaskDefinition("database", createDatabaseTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			databaseTaskDef = taskDef.TaskDefinitionArn
		})

		It("should deploy a complete multi-tier application", func() {
			By("Deploying database service")
			err := ecsClient.CreateService(clusterName, "database-service", databaseTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for database service to be stable")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "database-service")
				return err == nil && service.RunningCount == service.DesiredCount
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Deploying backend service")
			err = ecsClient.CreateService(clusterName, "backend-service", backendTaskDef, 2)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for backend service to be stable")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "backend-service")
				return err == nil && service.RunningCount == service.DesiredCount
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Deploying frontend service")
			err = ecsClient.CreateService(clusterName, "frontend-service", frontendTaskDef, 2)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for frontend service to be stable")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "frontend-service")
				return err == nil && service.RunningCount == service.DesiredCount
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Verifying all services are running")
			services, err := ecsClient.ListServices(clusterName)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(3))
			Expect(services).To(ContainElements(
				ContainSubstring("database-service"),
				ContainSubstring("backend-service"),
				ContainSubstring("frontend-service"),
			))

			By("Verifying service configurations")
			for _, serviceName := range []string{"database-service", "backend-service", "frontend-service"} {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				Expect(err).NotTo(HaveOccurred())
				Expect(service.RunningCount).To(BeNumerically(">", 0))
			}
		})

		It("should handle service dependencies", func() {
			By("Deploying services in dependency order")
			
			// Deploy database first
			err := ecsClient.CreateService(clusterName, "database-service", databaseTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			// Wait for database to be ready before deploying backend
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "database-service")
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			// Deploy backend
			err = ecsClient.CreateService(clusterName, "backend-service", backendTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			// Wait for backend to be ready before deploying frontend
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "backend-service")
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			// Deploy frontend
			err = ecsClient.CreateService(clusterName, "frontend-service", frontendTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying all services reach stable state")
			Eventually(func() bool {
				services := []string{"database-service", "backend-service", "frontend-service"}
				for _, serviceName := range services {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					if err != nil || service.RunningCount != service.DesiredCount {
						return false
					}
				}
				return true
			}, 120*time.Second, 10*time.Second).Should(BeTrue())
		})

		It("should handle rolling updates across services", func() {
			By("Initial deployment")
			services := map[string]string{
				"database-service": databaseTaskDef,
				"backend-service":  backendTaskDef,
				"frontend-service": frontendTaskDef,
			}

			for name, taskDef := range services {
				err := ecsClient.CreateService(clusterName, name, taskDef, 1)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Waiting for initial deployment to stabilize")
			Eventually(func() bool {
				for serviceName := range services {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					if err != nil || service.RunningCount != 1 {
						return false
					}
				}
				return true
			}, 120*time.Second, 10*time.Second).Should(BeTrue())

			By("Updating backend service")
			newBackendTaskDef, err := ecsClient.RegisterTaskDefinition("backend-v2", createBackendTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			err = ecsClient.UpdateService(clusterName, "backend-service", nil, newBackendTaskDef.TaskDefinitionArn)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying backend update completes")
			Eventually(func() string {
				service, err := ecsClient.DescribeService(clusterName, "backend-service")
				if err != nil {
					return ""
				}
				return service.TaskDefinition
			}, 60*time.Second, 5*time.Second).Should(Equal(newBackendTaskDef.TaskDefinitionArn))
		})
	})

	Describe("Service Scaling Scenarios", func() {
		var taskDefArn string

		BeforeEach(func() {
			taskDef, err := ecsClient.RegisterTaskDefinition("scaling-test", createSimpleTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			taskDefArn = taskDef.TaskDefinitionArn
		})

		It("should scale multiple services independently", func() {
			serviceConfigs := []struct {
				name         string
				initialCount int
				targetCount  int
			}{
				{"service-a", 1, 3},
				{"service-b", 2, 5},
				{"service-c", 1, 2},
			}

			By("Creating services with initial counts")
			for _, config := range serviceConfigs {
				err := ecsClient.CreateService(clusterName, config.name, taskDefArn, config.initialCount)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Waiting for initial deployment")
			Eventually(func() bool {
				for _, config := range serviceConfigs {
					service, err := ecsClient.DescribeService(clusterName, config.name)
					if err != nil || service.RunningCount != config.initialCount {
						return false
					}
				}
				return true
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Scaling all services to target counts")
			for _, config := range serviceConfigs {
				err := ecsClient.UpdateService(clusterName, config.name, &config.targetCount, "")
				Expect(err).NotTo(HaveOccurred())
			}

			By("Verifying all services reach target counts")
			Eventually(func() bool {
				for _, config := range serviceConfigs {
					service, err := ecsClient.DescribeService(clusterName, config.name)
					if err != nil || service.RunningCount != config.targetCount {
						return false
					}
				}
				return true
			}, 120*time.Second, 10*time.Second).Should(BeTrue())
		})
	})
})

func createFrontendTaskDefinition() string {
	return `{
		"family": "frontend-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "frontend",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/frontend",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "frontend"
					}
				}
			}
		]
	}`
}

func createBackendTaskDefinition() string {
	return `{
		"family": "backend-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "512",
		"memory": "1024",
		"containerDefinitions": [
			{
				"name": "backend",
				"image": "node:18-alpine",
				"command": ["node", "-e", "console.log('Backend service'); setInterval(() => {}, 1000);"],
				"essential": true,
				"memory": 512,
				"portMappings": [
					{
						"containerPort": 3000,
						"protocol": "tcp"
					}
				],
				"environment": [
					{
						"name": "NODE_ENV",
						"value": "production"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/backend",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "backend"
					}
				}
			}
		]
	}`
}

func createDatabaseTaskDefinition() string {
	return `{
		"family": "database-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "512",
		"memory": "1024",
		"containerDefinitions": [
			{
				"name": "database",
				"image": "postgres:13-alpine",
				"essential": true,
				"memory": 512,
				"portMappings": [
					{
						"containerPort": 5432,
						"protocol": "tcp"
					}
				],
				"environment": [
					{
						"name": "POSTGRES_DB",
						"value": "testdb"
					},
					{
						"name": "POSTGRES_USER",
						"value": "testuser"
					},
					{
						"name": "POSTGRES_PASSWORD",
						"value": "testpass"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/database",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "database"
					}
				}
			}
		]
	}`
}

func createSimpleTaskDefinition() string {
	return `{
		"family": "simple-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "simple-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256
			}
		]
	}`
}