package performance_test

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Cluster Scaling Performance", func() {
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
		clusterName = fmt.Sprintf("perf-cluster-%d", time.Now().Unix())
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

	Describe("Service Creation Performance", func() {
		It("should handle creating multiple services concurrently", func() {
			numServices := 10
			taskDefArn := ""

			By("Registering task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("perf-task", createPerformanceTaskDefinition())
			Expect(err).NotTo(HaveOccurred())
			taskDefArn = taskDef.TaskDefinitionArn

			By(fmt.Sprintf("Creating %d services concurrently", numServices))
			startTime := time.Now()
			
			var wg sync.WaitGroup
			var mu sync.Mutex
			var errors []error
			var serviceNames []string

			for i := 0; i < numServices; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					serviceName := fmt.Sprintf("perf-service-%d-%d", index, time.Now().Unix())
					
					mu.Lock()
					serviceNames = append(serviceNames, serviceName)
					mu.Unlock()
					
					err := ecsClient.CreateService(clusterName, serviceName, taskDefArn, 1)
					if err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
					}
				}(i)
			}

			wg.Wait()
			creationDuration := time.Since(startTime)

			By("Verifying all services were created successfully")
			Expect(errors).To(BeEmpty(), "All services should be created without errors")
			
			By(fmt.Sprintf("Measuring creation performance: %v for %d services", creationDuration, numServices))
			avgCreationTime := creationDuration / time.Duration(numServices)
			fmt.Printf("Average service creation time: %v\n", avgCreationTime)
			
			// Should be able to create services reasonably quickly
			Expect(avgCreationTime).To(BeNumerically("<", 5*time.Second))

			By("Verifying all services reach running state")
			Eventually(func() bool {
				runningCount := 0
				for _, serviceName := range serviceNames {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					if err == nil && service.RunningCount > 0 {
						runningCount++
					}
				}
				return runningCount == numServices
			}, 300*time.Second, 10*time.Second).Should(BeTrue())

			By("Cleanup")
			for _, serviceName := range serviceNames {
				ecsClient.DeleteService(clusterName, serviceName)
			}
		})

		It("should handle rapid service scaling", func() {
			serviceName := fmt.Sprintf("scale-perf-%d", time.Now().Unix())

			By("Registering task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("scale-perf-task", createPerformanceTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By("Creating service with 1 instance")
			err = ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, 1)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for initial service to be running")
			Eventually(func() bool {
				service, err := ecsClient.DescribeService(clusterName, serviceName)
				return err == nil && service.RunningCount == 1
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			scaleTargets := []int{5, 10, 15, 20}
			
			for _, target := range scaleTargets {
				By(fmt.Sprintf("Scaling to %d instances", target))
				startTime := time.Now()
				
				err = ecsClient.UpdateService(clusterName, serviceName, &target, "")
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Waiting for scale to %d to complete", target))
				Eventually(func() int {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					if err != nil {
						return 0
					}
					return service.RunningCount
				}, 300*time.Second, 10*time.Second).Should(Equal(target))

				scaleDuration := time.Since(startTime)
				fmt.Printf("Scale to %d instances took: %v\n", target, scaleDuration)
				
				// Scaling should complete within reasonable time
				Expect(scaleDuration).To(BeNumerically("<", 120*time.Second))
			}

			By("Cleanup")
			ecsClient.DeleteService(clusterName, serviceName)
		})
	})

	Describe("Task Definition Performance", func() {
		It("should handle registering many task definitions", func() {
			numTaskDefs := 50
			
			By(fmt.Sprintf("Registering %d task definitions", numTaskDefs))
			startTime := time.Now()
			
			var wg sync.WaitGroup
			var mu sync.Mutex
			var errors []error
			var taskDefArns []string

			for i := 0; i < numTaskDefs; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					taskDefName := fmt.Sprintf("perf-taskdef-%d", index)
					
					taskDef, err := ecsClient.RegisterTaskDefinition(taskDefName, createPerformanceTaskDefinition())
					if err != nil {
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
					} else {
						mu.Lock()
						taskDefArns = append(taskDefArns, taskDef.TaskDefinitionArn)
						mu.Unlock()
					}
				}(i)
			}

			wg.Wait()
			registrationDuration := time.Since(startTime)

			By("Verifying all task definitions were registered successfully")
			Expect(errors).To(BeEmpty(), "All task definitions should be registered without errors")
			Expect(taskDefArns).To(HaveLen(numTaskDefs))
			
			By(fmt.Sprintf("Measuring registration performance: %v for %d task definitions", registrationDuration, numTaskDefs))
			avgRegistrationTime := registrationDuration / time.Duration(numTaskDefs)
			fmt.Printf("Average task definition registration time: %v\n", avgRegistrationTime)
			
			// Should be able to register task definitions quickly
			Expect(avgRegistrationTime).To(BeNumerically("<", 2*time.Second))
		})
	})

	Describe("Resource Utilization", func() {
		It("should efficiently handle high-density deployments", func() {
			numServices := 20
			instancesPerService := 3
			totalInstances := numServices * instancesPerService

			By("Registering lightweight task definition")
			taskDef, err := ecsClient.RegisterTaskDefinition("lightweight-task", createLightweightTaskDefinition())
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Creating %d services with %d instances each (%d total)", 
				numServices, instancesPerService, totalInstances))
			
			startTime := time.Now()
			var serviceNames []string

			for i := 0; i < numServices; i++ {
				serviceName := fmt.Sprintf("density-service-%d-%d", i, time.Now().Unix())
				serviceNames = append(serviceNames, serviceName)
				
				err := ecsClient.CreateService(clusterName, serviceName, taskDef.TaskDefinitionArn, instancesPerService)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Waiting for all services to reach desired state")
			Eventually(func() int {
				totalRunning := 0
				for _, serviceName := range serviceNames {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					if err == nil {
						totalRunning += service.RunningCount
					}
				}
				return totalRunning
			}, 600*time.Second, 15*time.Second).Should(Equal(totalInstances))

			deploymentDuration := time.Since(startTime)
			fmt.Printf("High-density deployment (%d instances) took: %v\n", totalInstances, deploymentDuration)

			By("Verifying all services are stable")
			Consistently(func() int {
				totalRunning := 0
				for _, serviceName := range serviceNames {
					service, err := ecsClient.DescribeService(clusterName, serviceName)
					if err == nil {
						totalRunning += service.RunningCount
					}
				}
				return totalRunning
			}, 60*time.Second, 10*time.Second).Should(Equal(totalInstances))

			By("Cleanup")
			for _, serviceName := range serviceNames {
				ecsClient.DeleteService(clusterName, serviceName)
			}
		})
	})
})

func createPerformanceTaskDefinition() string {
	return `{
		"family": "performance-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "256",
		"memory": "512",
		"containerDefinitions": [
			{
				"name": "perf-container",
				"image": "nginx:alpine",
				"essential": true,
				"memory": 128,
				"cpu": 64,
				"portMappings": [
					{
						"containerPort": 80,
						"protocol": "tcp"
					}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/performance-test",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "perf"
					}
				}
			}
		]
	}`
}

func createLightweightTaskDefinition() string {
	return `{
		"family": "lightweight-task",
		"networkMode": "bridge",
		"requiresCompatibilities": ["EC2"],
		"cpu": "128",
		"memory": "256",
		"containerDefinitions": [
			{
				"name": "lightweight-container",
				"image": "alpine:latest",
				"command": ["sh", "-c", "echo 'Lightweight container'; sleep 300"],
				"essential": true,
				"memory": 64,
				"cpu": 32,
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/lightweight-test",
						"awslogs-region": "us-east-1",
						"awslogs-stream-prefix": "lightweight"
					}
				}
			}
		]
	}`
}