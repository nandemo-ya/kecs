package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Deployment Configuration", func() {
	var (
		kecs        *utils.KECSContainer
		client      *utils.ECSClient
		clusterName string
	)

	BeforeEach(func() {
		kecs = utils.StartKECS(GinkgoT())
		client = utils.NewECSClient(kecs.Endpoint())

		// Create a test cluster
		clusterName = fmt.Sprintf("test-cluster-%d", time.Now().Unix())
		err := client.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// Cleanup cluster
		_ = client.DeleteCluster(clusterName)
		kecs.Cleanup()
	})

	Context("when configuring deployment strategies", func() {
		It("should handle custom deployment configuration", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-deploy-config-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:alpine",
						"memory":    256,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with custom deployment configuration
			serviceName := fmt.Sprintf("test-deploy-config-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   3,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        150, // Can have up to 150% of desired count
					"minimumHealthyPercent": 75,  // Must maintain at least 75% healthy
					"deploymentCircuitBreaker": map[string]interface{}{
						"enable":   true,
						"rollback": true,
					},
				},
			}

			result, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Verify deployment configuration was applied
			service := result["service"].(map[string]interface{})
			deployConfig := service["deploymentConfiguration"].(map[string]interface{})
			Expect(deployConfig["maximumPercent"]).To(Equal(float64(150)))
			Expect(deployConfig["minimumHealthyPercent"]).To(Equal(float64(75)))

			circuitBreaker := deployConfig["deploymentCircuitBreaker"].(map[string]interface{})
			Expect(circuitBreaker["enable"]).To(BeTrue())
			Expect(circuitBreaker["rollback"]).To(BeTrue())

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should track deployment progress", func() {
			// Register initial task definition
			taskDefFamily := fmt.Sprintf("test-deploy-progress-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Starting v1'; sleep 10; echo 'Running v1'"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := fmt.Sprintf("test-deploy-progress-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial deployment
			time.Sleep(10 * time.Second)

			// Register new version
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["command"] = []string{
				"sh", "-c", "echo 'Starting v2'; sleep 10; echo 'Running v2'",
			}

			_, err = client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Update service
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": taskDefFamily + ":2",
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Track deployment states
			deploymentStates := []string{}
			for i := 0; i < 10; i++ {
				descResult, err := client.DescribeService(clusterName, serviceName)
				if err == nil {
					service := descResult["service"].(map[string]interface{})
					if deployments, ok := service["deployments"].([]interface{}); ok && len(deployments) > 0 {
						primaryDeployment := deployments[0].(map[string]interface{})
						state := primaryDeployment["status"].(string)
						deploymentStates = append(deploymentStates, state)
						
						// Check if deployment is complete
						if state == "PRIMARY" && len(deployments) == 1 {
							break
						}
					}
				}
				time.Sleep(3 * time.Second)
			}

			// Should have seen deployment progress
			Expect(len(deploymentStates)).To(BeNumerically(">", 0))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when handling circuit breaker", func() {
		It("should stop deployment on repeated failures", func() {
			// Register task definition that will fail
			taskDefFamily := fmt.Sprintf("test-circuit-breaker-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "failing-app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "exit 1"}, // Will always fail
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with circuit breaker
			serviceName := fmt.Sprintf("test-circuit-breaker-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"deploymentCircuitBreaker": map[string]interface{}{
						"enable":   true,
						"rollback": false, // Don't rollback, just stop
					},
				},
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for circuit breaker to trigger
			time.Sleep(30 * time.Second)

			// Check deployment status
			descResult, err := client.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			service := descResult["service"].(map[string]interface{})
			deployments := service["deployments"].([]interface{})
			
			// Should have a failed deployment
			var failedDeployment map[string]interface{}
			for _, d := range deployments {
				deployment := d.(map[string]interface{})
				if deployment["status"] == "FAILED" {
					failedDeployment = deployment
					break
				}
			}

			Expect(failedDeployment).NotTo(BeNil(), "Expected to find a FAILED deployment")

			// Cleanup
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should automatically rollback on failure when configured", func() {
			// Register working task definition
			taskDefFamily := fmt.Sprintf("test-auto-rollback-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:alpine",
						"memory":    256,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			result, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			var workingRevision string
			if td, ok := result["taskDefinition"].(map[string]interface{}); ok {
				workingRevision = fmt.Sprintf("%s:%v", taskDefFamily, td["revision"])
			}

			// Create service with auto-rollback
			serviceName := fmt.Sprintf("test-auto-rollback-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": workingRevision,
				"desiredCount":   2,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"deploymentCircuitBreaker": map[string]interface{}{
						"enable":   true,
						"rollback": true,
					},
				},
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial deployment
			time.Sleep(10 * time.Second)

			// Register failing task definition
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["command"] = []string{
				"sh", "-c", "exit 1",
			}

			result2, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			var failingRevision string
			if td, ok := result2["taskDefinition"].(map[string]interface{}); ok {
				failingRevision = fmt.Sprintf("%s:%v", taskDefFamily, td["revision"])
			}

			// Try to update to failing version
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": failingRevision,
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for rollback
			time.Sleep(30 * time.Second)

			// Check that service rolled back
			descResult, err := client.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			service := descResult["service"].(map[string]interface{})
			// Should have rolled back to working version
			currentTaskDef := service["taskDefinition"].(string)
			Expect(currentTaskDef).To(ContainSubstring(workingRevision))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when handling multiple deployments", func() {
		It("should queue deployments properly", func() {
			// Register task definitions
			taskDefFamily := fmt.Sprintf("test-queue-%d", time.Now().Unix())
			
			// V1
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:1.19",
						"memory":    256,
						"essential": true,
						"environment": []map[string]interface{}{
							{"name": "VERSION", "value": "1"},
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}
			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// V2
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["image"] = "nginx:1.20"
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["environment"] = []map[string]interface{}{
				{"name": "VERSION", "value": "2"},
			}
			_, err = client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// V3
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["image"] = "nginx:1.21"
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["environment"] = []map[string]interface{}{
				{"name": "VERSION", "value": "3"},
			}
			_, err = client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := fmt.Sprintf("test-queue-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily + ":1",
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial deployment
			time.Sleep(5 * time.Second)

			// Rapid updates
			_, err = client.UpdateService(map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": taskDefFamily + ":2",
			})
			Expect(err).NotTo(HaveOccurred())

			// Immediately update again
			_, err = client.UpdateService(map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": taskDefFamily + ":3",
			})
			Expect(err).NotTo(HaveOccurred())

			// Eventually should end up on version 3
			Eventually(func() string {
				descResult, err := client.DescribeService(clusterName, serviceName)
				if err != nil {
					return ""
				}
				service := descResult["service"].(map[string]interface{})
				return service["taskDefinition"].(string)
			}, 60*time.Second, 5*time.Second).Should(ContainSubstring(":3"))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})
})