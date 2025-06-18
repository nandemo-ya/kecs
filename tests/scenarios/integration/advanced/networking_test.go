package advanced_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Networking Scenarios", func() {
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
		clusterName = fmt.Sprintf("networking-test-%d", time.Now().Unix())
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

	Describe("Network Mode Configurations", func() {
		It("should deploy services with different network modes", func() {
			networkModes := []struct {
				mode        string
				description string
			}{
				{"bridge", "Bridge network mode"},
				{"host", "Host network mode"},
				{"awsvpc", "AWS VPC network mode"},
			}

			for _, config := range networkModes {
				serviceName := fmt.Sprintf("network-%s-%d", config.mode, time.Now().Unix())
				taskDefName := fmt.Sprintf("network-task-%s", config.mode)

				By(fmt.Sprintf("Registering task definition with %s", config.description))
				taskDef, err := ecsClient.RegisterTaskDefinition(taskDefName, createNetworkTaskDefinition(config.mode))
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Creating service with %s", config.description))
				err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Verifying service with %s is running", config.description))
				Eventually(func() bool {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					return err == nil && service.RunningCount > 0
				}, 90*time.Second, 10*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Cleanup %s service", config.description))
				ecsClient.DeleteService(clusterName, serviceName)
			}
		})
	})

	Describe("Port Mapping Scenarios", func() {
		It("should handle multiple port mappings", func() {
			serviceName := fmt.Sprintf("multiport-service-%d", time.Now().Unix())

			By("Registering task definition with multiple port mappings")
			taskDef, err := ecsClient.RegisterTaskDefinition("multiport-task", createMultiPortTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with multiple ports")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service is running")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})

		It("should handle dynamic port allocation", func() {
			serviceName := fmt.Sprintf("dynamic-port-%d", time.Now().Unix())

			By("Registering task definition with dynamic ports")
			taskDef, err := ecsClient.RegisterTaskDefinition("dynamic-port-task", createDynamicPortTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with dynamic port allocation")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 2)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying multiple instances can run with dynamic ports")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.RunningCount == 2
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})

	Describe("Inter-Service Communication", func() {
		var (
			backendTaskDef  string
			frontendTaskDef string
		)

		BeforeEach(func() {
			// Register backend service task definition
			taskDef, err := ecsClient.RegisterTaskDefinition("backend-comm", createBackendCommTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			backendTaskDef = taskDef.TaskDefinitionArn

			// Register frontend service task definition
			taskDef, err = ecsClient.RegisterTaskDefinition("frontend-comm", createFrontendCommTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			frontendTaskDef = taskDef.TaskDefinitionArn
		})

		It("should enable communication between services", func() {
			By("Deploying backend service")
			err := ecsClient.CreateService(clusterName, "backend-comm-service", backendTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for backend to be ready")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "backend-comm-service")
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Deploying frontend service")
			err = ecsClient.CreateService(clusterName, "frontend-comm-service", frontendTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying both services are running")
			Eventually(func() bool {
				backendService, err1 := ecsClient.DescribeService(clusterName, "backend-comm-service")
				frontendService, err2 := ecsClient.DescribeService(clusterName, "frontend-comm-service")
				return err1 == nil && err2 == nil &&
					backendService.RunningCount > 0 && frontendService.RunningCount > 0
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Cleanup services")
			ecsClient.DeleteService(clusterName, "frontend-comm-service")
			ecsClient.DeleteService(clusterName, "backend-comm-service")
		})
	})
})

func createNetworkTaskDefinition(networkMode string) string {
	return fmt.Sprintf(`{
		"family": "network-task-%s",
		"networkMode": "%s",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "network-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				]
			}
		]
	}`, networkMode, networkMode)
}

func createMultiPortTaskDefinition() string {
	return `{
		"family": "multiport-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "multiport-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"hostPort": 8080,
						"protocol": "tcp"
					},
					{
						"containerPort": 443,
						"hostPort": 8443,
						"protocol": "tcp"
					}
				]
			}
		]
	}`
}

func createDynamicPortTaskDefinition() string {
	return `{
		"family": "dynamic-port-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "dynamic-port-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				]
			}
		]
	}`
}

func createBackendCommTaskDefinition() string {
	return `{
		"family": "backend-comm-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "backend-comm",
				"image": "node:18-alpine",
				"command": ["node", "-e", "require('http').createServer((req,res)=>{res.writeHead(200);res.end('Backend API')}).listen(3000,()=>console.log('Backend listening on 3000'))"],
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 3000,
						"protocol": "tcp"
					}
				],
				"environment": [
					{
						"name": "NODE_ENV",
						"value": "test"
					}
				]
			}
		]
	}`
}

func createFrontendCommTaskDefinition() string {
	return `{
		"family": "frontend-comm-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "frontend-comm",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				],
				"environment": [
					{
						"name": "BACKEND_URL",
						"value": "http://backend-comm-service:3000"
					}
				]
			}
		]
	}`
}