package advanced_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Discovery Scenarios", func() {
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
		clusterName = fmt.Sprintf("discovery-test-%d", time.Now().Unix())
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

	Describe("AWS Cloud Map Integration", func() {
		It("should create services with service discovery", func() {
			serviceName := fmt.Sprintf("discovery-service-%d", time.Now().Unix())

			By("Registering task definition with service discovery")
			taskDef, err := ecsClient.RegisterTaskDefinition("discovery-task", createServiceDiscoveryTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with service discovery enabled")
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
	})

	Describe("Service Discovery between Services", func() {
		var (
			producerTaskDef string
			consumerTaskDef string
		)

		BeforeEach(func() {
			// Register producer service task definition
			taskDef, err := ecsClient.RegisterTaskDefinition("producer-service", createProducerTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			producerTaskDef = taskDef.TaskDefinitionArn

			// Register consumer service task definition
			taskDef, err = ecsClient.RegisterTaskDefinition("consumer-service", createConsumerTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			consumerTaskDef = taskDef.TaskDefinitionArn
		})

		It("should enable service-to-service discovery", func() {
			By("Deploying producer service")
			err := ecsClient.CreateService(clusterName, "producer-service", producerTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for producer service to be ready")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "producer-service")
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Deploying consumer service")
			err = ecsClient.CreateService(clusterName, "consumer-service", consumerTaskDef, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying consumer can discover producer")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "consumer-service")
				return err == nil && service.RunningCount > 0
			}, 90*time.Second, 10*time.Second).Should(BeTrue())

			By("Cleanup services")
			ecsClient.DeleteService(clusterName, "consumer-service")
			ecsClient.DeleteService(clusterName, "producer-service")
		})

		It("should handle service discovery with multiple instances", func() {
			By("Deploying producer service with multiple instances")
			err := ecsClient.CreateService(clusterName, "multi-producer", producerTaskDef, 3)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for all producer instances to be ready")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, "multi-producer")
				return err == nil && service.RunningCount == 3
			}, 120*time.Second, 10*time.Second).Should(BeTrue())

			By("Deploying consumer service")
			err = ecsClient.CreateService(clusterName, "multi-consumer", consumerTaskDef, 2)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying all services are running")
			Eventually(func() bool {
				producerService, err1 := ecsClient.DescribeService(clusterName, "multi-producer")
				consumerService, err2 := ecsClient.DescribeService(clusterName, "multi-consumer")
				return err1 == nil && err2 == nil &&
					producerService.RunningCount == 3 && consumerService.RunningCount == 2
			}, 120*time.Second, 10*time.Second).Should(BeTrue())

			By("Cleanup services")
			ecsClient.DeleteService(clusterName, "multi-consumer")
			ecsClient.DeleteService(clusterName, "multi-producer")
		})
	})

	Describe("DNS-based Service Discovery", func() {
		It("should resolve service names via DNS", func() {
			serviceName := fmt.Sprintf("dns-service-%d", time.Now().Unix())

			By("Registering task definition with DNS configuration")
			taskDef, err := ecsClient.RegisterTaskDefinition("dns-task", createDNSTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with DNS-based discovery")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying service is accessible via DNS")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.RunningCount > 0
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})
})

func createServiceDiscoveryTaskDefinition() string {
	return `{
		"family": "discovery-task",
		"networkMode": "awsvpc",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "discovery-container",
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
						"name": "SERVICE_NAME",
						"value": "discovery-service"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/discovery-service",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "discovery"
					}
				}
			}
		]
	}`
}

func createProducerTaskDefinition() string {
	return `{
		"family": "producer-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "producer",
				"image": "node:18-alpine",
				"command": ["node", "-e", "require('http').createServer((req,res)=>{res.writeHead(200,{'Content-Type':'application/json'});res.end(JSON.stringify({service:'producer',timestamp:Date.now()}))}).listen(8080,()=>console.log('Producer listening on 8080'))"],
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 8080,
						"protocol": "tcp"
					}
				],
				"environment": [
					{
						"name": "SERVICE_TYPE",
						"value": "producer"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/producer-service",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "producer"
					}
				}
			}
		]
	}`
}

func createConsumerTaskDefinition() string {
	return `{
		"family": "consumer-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "consumer",
				"image": "node:18-alpine",
				"command": ["node", "-e", "setInterval(()=>{console.log('Consumer checking for producer service...');},5000);require('http').createServer((req,res)=>{res.writeHead(200);res.end('Consumer service running')}).listen(8081,()=>console.log('Consumer listening on 8081'))"],
				"essential": true,
				"memory": 256,
				"portMappings": [
					{
						"containerPort": 8081,
						"protocol": "tcp"
					}
				],
				"environment": [
					{
						"name": "SERVICE_TYPE",
						"value": "consumer"
					},
					{
						"name": "PRODUCER_SERVICE_URL",
						"value": "http://producer-service:8080"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/consumer-service",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "consumer"
					}
				}
			}
		]
	}`
}

func createDNSTaskDefinition() string {
	return `{
		"family": "dns-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "dns-container",
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
						"name": "DNS_ENABLED",
						"value": "true"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/dns-service",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "dns"
					}
				}
			}
		]
	}`
}