package performance_test

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("API Performance Tests", func() {
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
		clusterName = fmt.Sprintf("api-perf-%d", time.Now().Unix())
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

	Describe("API Response Times", func() {
		It("should handle concurrent API requests efficiently", func() {
			numConcurrentRequests := 50
			
			By("Registering task definition for API tests")
			taskDef, err := ecsClient.RegisterTaskDefinition("api-perf-task", createAPITestTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Making %d concurrent DescribeService requests", numConcurrentRequests))
			startTime := time.Now()
			
			var wg sync.WaitGroup
			var mu sync.Mutex
			var responseTimes []time.Duration
			var errors []error

			for i := 0; i < numConcurrentRequests; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					requestStart := time.Now()
					
					_, err := ecsClient.DescribeService(clusterName, "non-existent-service")
					requestDuration := time.Since(requestStart)
					
					mu.Lock()
					responseTimes = append(responseTimes, requestDuration)
					if err == nil {
						// We expect this to error for non-existent service, but track unexpected successes
						errors = append(errors, fmt.Errorf("unexpected success for non-existent service"))
					}
					mu.Unlock()
				}()
			}

			wg.Wait()
			totalDuration := time.Since(startTime)

			By("Analyzing response times")
			Expect(responseTimes).To(HaveLen(numConcurrentRequests))
			
			var totalResponseTime time.Duration
			var maxResponseTime time.Duration
			var minResponseTime time.Duration = time.Hour // Initialize to large value
			
			for _, rt := range responseTimes {
				totalResponseTime += rt
				if rt > maxResponseTime {
					maxResponseTime = rt
				}
				if rt < minResponseTime {
					minResponseTime = rt
				}
			}
			
			avgResponseTime := totalResponseTime / time.Duration(numConcurrentRequests)
			
			fmt.Printf("Concurrent API Performance Results:\n")
			fmt.Printf("  Total time: %v\n", totalDuration)
			fmt.Printf("  Average response time: %v\n", avgResponseTime)
			fmt.Printf("  Min response time: %v\n", minResponseTime)
			fmt.Printf("  Max response time: %v\n", maxResponseTime)
			fmt.Printf("  Requests per second: %.2f\n", float64(numConcurrentRequests)/totalDuration.Seconds())
			
			// API should respond within reasonable time even under load
			Expect(avgResponseTime).To(BeNumerically("<", 2*time.Second))
			Expect(maxResponseTime).To(BeNumerically("<", 5*time.Second))
		})

		It("should maintain performance with large numbers of services", func() {
			numServices := 20
			
			By("Registering task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("many-services-task", createAPITestTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %d services", numServices))
			var serviceNames []string
			for i := 0; i < numServices; i++ {
				serviceName := fmt.Sprintf("api-test-service-%d", i)
				serviceNames = append(serviceNames, serviceName)
				
				err := ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Measuring ListServices performance with many services")
			startTime := time.Now()
			services, err := ecsClient.ListServices(clusterName)
			listDuration := time.Since(startTime)
			
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(HaveLen(numServices))
			
			fmt.Printf("ListServices with %d services took: %v\n", numServices, listDuration)
			Expect(listDuration).To(BeNumerically("<", 3*time.Second))

			By("Measuring DescribeService performance for individual services")
			var describeTimes []time.Duration
			
			for _, serviceName := range serviceNames[:5] { // Test first 5 services
				startTime := time.Now()
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				describeDuration := time.Since(startTime)
				
				Expect(err).NotTo(HaveOccurred())
				Expect(service.ServiceName).To(Equal(serviceName))
				describeTimes = append(describeTimes, describeDuration)
			}
			
			var totalDescribeTime time.Duration
			for _, dt := range describeTimes {
				totalDescribeTime += dt
			}
			avgDescribeTime := totalDescribeTime / time.Duration(len(describeTimes))
			
			fmt.Printf("Average DescribeService time with %d services: %v\n", numServices, avgDescribeTime)
			Expect(avgDescribeTime).To(BeNumerically("<", 1*time.Second))

			By("Cleanup")
			for _, serviceName := range serviceNames {
				ecsClient.DeleteService(clusterName, serviceName)
			}
		})
	})

	Describe("Throughput Tests", func() {
		It("should handle rapid service operations", func() {
			operationCount := 100
			
			By("Registering task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("throughput-task", createAPITestTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Performing %d rapid create/delete cycles", operationCount))
			startTime := time.Now()
			
			for i := 0; i < operationCount; i++ {
				serviceName := fmt.Sprintf("throughput-test-%d", i)
				
				// Create service
				err := ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
				Expect(err).NotTo(HaveOccurred())
				
				// Immediately delete service
				err = ecsClient.DeleteService(clusterName, serviceName)
				Expect(err).NotTo(HaveOccurred())
			}
			
			totalDuration := time.Since(startTime)
			operationsPerSecond := float64(operationCount*2) / totalDuration.Seconds() // 2 operations per cycle
			
			fmt.Printf("Rapid operations performance:\n")
			fmt.Printf("  %d create/delete cycles in %v\n", operationCount, totalDuration)
			fmt.Printf("  Operations per second: %.2f\n", operationsPerSecond)
			
			// Should be able to handle a reasonable throughput
			Expect(operationsPerSecond).To(BeNumerically(">", 5.0))
		})

		It("should handle batch operations efficiently", func() {
			batchSize := 10
			
			By("Registering task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("batch-task", createAPITestTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %d services in rapid succession", batchSize))
			startTime := time.Now()
			var serviceNames []string
			
			for i := 0; i < batchSize; i++ {
				serviceName := fmt.Sprintf("batch-service-%d-%d", i, time.Now().UnixNano())
				serviceNames = append(serviceNames, serviceName)
				
				err := ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
				Expect(err).NotTo(HaveOccurred())
			}
			
			creationDuration := time.Since(startTime)
			fmt.Printf("Batch creation of %d services took: %v\n", batchSize, creationDuration)
			
			By(fmt.Sprintf("Deleting %d services in rapid succession", batchSize))
			startTime = time.Now()
			
			for _, serviceName := range serviceNames {
				err := ecsClient.DeleteService(clusterName, serviceName)
				Expect(err).NotTo(HaveOccurred())
			}
			
			deletionDuration := time.Since(startTime)
			fmt.Printf("Batch deletion of %d services took: %v\n", batchSize, deletionDuration)
			
			// Batch operations should be efficient
			avgCreationTime := creationDuration / time.Duration(batchSize)
			avgDeletionTime := deletionDuration / time.Duration(batchSize)
			
			Expect(avgCreationTime).To(BeNumerically("<", 2*time.Second))
			Expect(avgDeletionTime).To(BeNumerically("<", 1*time.Second))
		})
	})

	Describe("Memory and Resource Usage", func() {
		It("should handle large payloads efficiently", func() {
			By("Registering large task definition")
			largeTaskDef := createLargeTaskDefinition()
			
			startTime := time.Now()
			taskDef, err := ecsClient.RegisterTaskDefinition("large-task", largeTaskDef)
			registrationDuration := time.Since(startTime)
			
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("Large task definition registration took: %v\n", registrationDuration)
			
			// Should handle large payloads within reasonable time
			Expect(registrationDuration).To(BeNumerically("<", 5*time.Second))

			By("Creating service with large task definition")
			serviceName := fmt.Sprintf("large-service-%d", time.Now().Unix())
			
			startTime = time.Now()
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			creationDuration := time.Since(startTime)
			
			Expect(err).NotTo(HaveOccurred())
			fmt.Printf("Service creation with large task definition took: %v\n", creationDuration)
			
			Expect(creationDuration).To(BeNumerically("<", 5*time.Second))

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})
})

func createAPITestTaskDefinition() string {
	return `{
		"family": "api-test-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "api-test-container",
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

func createLargeTaskDefinition() string {
	// Create a task definition with many containers and environment variables
	return `{
		"family": "large-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "1024",
		"memory": "2048",
		"containerDefinitions": [
			{
				"name": "main-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 512,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					},
					{
						"containerPort": 443,
						"protocol": "tcp"
					}
				],
				"environment": [
					{"name": "ENV_VAR_1", "value": "value1"},
					{"name": "ENV_VAR_2", "value": "value2"},
					{"name": "ENV_VAR_3", "value": "value3"},
					{"name": "ENV_VAR_4", "value": "value4"},
					{"name": "ENV_VAR_5", "value": "value5"},
					{"name": "ENV_VAR_6", "value": "value6"},
					{"name": "ENV_VAR_7", "value": "value7"},
					{"name": "ENV_VAR_8", "value": "value8"},
					{"name": "ENV_VAR_9", "value": "value9"},
					{"name": "ENV_VAR_10", "value": "value10"}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/large-task-main",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "main"
					}
				}
			},
			{
				"name": "sidecar-container-1",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Sidecar 1'; sleep 300"],
				"essential": false,
				"memory": 256,
				"environment": [
					{"name": "SIDECAR_TYPE", "value": "logging"},
					{"name": "SIDECAR_VERSION", "value": "1.0.0"}
				]
			},
			{
				"name": "sidecar-container-2",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Sidecar 2'; sleep 300"],
				"essential": false,
				"memory": 256,
				"environment": [
					{"name": "SIDECAR_TYPE", "value": "monitoring"},
					{"name": "SIDECAR_VERSION", "value": "2.0.0"}
				]
			},
			{
				"name": "sidecar-container-3",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Sidecar 3'; sleep 300"],
				"essential": false,
				"memory": 256,
				"environment": [
					{"name": "SIDECAR_TYPE", "value": "proxy"},
					{"name": "SIDECAR_VERSION", "value": "3.0.0"}
				]
			}
		]
	}`
}