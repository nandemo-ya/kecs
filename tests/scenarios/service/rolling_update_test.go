package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Rolling Updates", func() {
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

	Context("when performing basic rolling updates", func() {
		It("should update service to new task definition version", func() {
			// Register initial task definition
			taskDefFamily := fmt.Sprintf("test-rolling-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:1.19",
						"memory":    256,
						"essential": true,
						"portMappings": []map[string]interface{}{
							{
								"containerPort": 80,
								"protocol":      "tcp",
							},
						},
						"environment": []map[string]interface{}{
							{
								"name":  "VERSION",
								"value": "1.0",
							},
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			result, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			var taskDefArn string
			if td, ok := result["taskDefinition"].(map[string]interface{}); ok {
				taskDefArn = td["taskDefinitionArn"].(string)
			}

			// Create service with initial version
			serviceName := fmt.Sprintf("test-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefArn,
				"desiredCount":   3,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        200,
					"minimumHealthyPercent": 100,
				},
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial deployment
			time.Sleep(10 * time.Second)

			// Verify initial tasks are running
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())

			initialTaskArns := listResult["taskArns"].([]interface{})
			Expect(initialTaskArns).To(HaveLen(3))

			// Register new version of task definition
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["image"] = "nginx:1.20"
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["environment"] = []map[string]interface{}{
				{
					"name":  "VERSION",
					"value": "2.0",
				},
			}

			result2, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			var newTaskDefArn string
			if td, ok := result2["taskDefinition"].(map[string]interface{}); ok {
				newTaskDefArn = td["taskDefinitionArn"].(string)
			}

			// Update service to new task definition
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": newTaskDefArn,
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Monitor rolling update progress
			Eventually(func() bool {
				// Check if all tasks are using new task definition
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return false
				}

				taskArns := listResult["taskArns"].([]interface{})
				if len(taskArns) != 3 {
					return false
				}

				// Describe tasks to check task definition
				descResult, err := client.DescribeTasks(clusterName, utils.InterfaceSliceToStringSlice(taskArns))
				if err != nil {
					return false
				}

				tasks := descResult["tasks"].([]interface{})
				for _, t := range tasks {
					task := t.(map[string]interface{})
					if taskDefArn, ok := task["taskDefinitionArn"].(string); ok {
						if taskDefArn != newTaskDefArn {
							return false
						}
					}
				}

				return true
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			// Verify old tasks were stopped
			stoppedResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "STOPPED",
			})
			Expect(err).NotTo(HaveOccurred())

			stoppedArns := stoppedResult["taskArns"].([]interface{})
			// Should have at least the original 3 tasks stopped
			Expect(len(stoppedArns)).To(BeNumerically(">=", 3))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should maintain minimum healthy percent during update", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-min-healthy-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "while true; do echo 'Running'; sleep 5; done"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with specific deployment configuration
			serviceName := fmt.Sprintf("test-min-healthy-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   4,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        150, // Max 6 tasks (150% of 4)
					"minimumHealthyPercent": 50,  // Min 2 tasks (50% of 4)
				},
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial deployment
			time.Sleep(10 * time.Second)

			// Register new version
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["command"] = []string{
				"sh", "-c", "echo 'New version'; while true; do echo 'Running v2'; sleep 5; done",
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

			// Monitor that minimum healthy tasks are maintained
			// Poll multiple times during the update
			for i := 0; i < 10; i++ {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err == nil {
					runningTasks := listResult["taskArns"].([]interface{})
					// Should always have at least 2 running tasks (50% of 4)
					Expect(len(runningTasks)).To(BeNumerically(">=", 2))
				}
				time.Sleep(3 * time.Second)
			}

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when handling zero-downtime deployments", func() {
		It("should ensure no downtime with 100% minimum healthy", func() {
			// Register task definition with health check
			taskDefFamily := fmt.Sprintf("test-zero-downtime-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "webapp",
						"image":     "nginx:alpine",
						"memory":    256,
						"essential": true,
						"portMappings": []map[string]interface{}{
							{
								"containerPort": 80,
								"protocol":      "tcp",
							},
						},
						"healthCheck": map[string]interface{}{
							"command":     []string{"CMD-SHELL", "curl -f http://localhost/ || exit 1"},
							"interval":    5,
							"timeout":     3,
							"retries":     2,
							"startPeriod": 10,
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with 100% minimum healthy
			serviceName := fmt.Sprintf("test-zero-downtime-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        200, // Allow double capacity
					"minimumHealthyPercent": 100, // Never go below desired count
				},
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial deployment
			time.Sleep(15 * time.Second)

			// Track initial healthy task count
			var healthyTaskCount int
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			healthyTaskCount = len(listResult["taskArns"].([]interface{}))
			Expect(healthyTaskCount).To(Equal(2))

			// Register new version
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["environment"] = []map[string]interface{}{
				{
					"name":  "DEPLOYMENT",
					"value": "v2",
				},
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

			// Monitor that we always have at least 2 healthy tasks
			downtime := false
			for i := 0; i < 20; i++ {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err == nil {
					runningTasks := listResult["taskArns"].([]interface{})
					if len(runningTasks) < 2 {
						downtime = true
						break
					}
				}
				time.Sleep(2 * time.Second)
			}

			Expect(downtime).To(BeFalse(), "Service experienced downtime during deployment")

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when handling update failures", func() {
		It("should rollback on deployment failure", func() {
			// Register working task definition
			taskDefFamily := fmt.Sprintf("test-rollback-%d", time.Now().Unix())
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

			var workingTaskDefArn string
			if td, ok := result["taskDefinition"].(map[string]interface{}); ok {
				workingTaskDefArn = td["taskDefinitionArn"].(string)
			}

			// Create service with circuit breaker
			serviceName := fmt.Sprintf("test-rollback-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": workingTaskDefArn,
				"desiredCount":   2,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        200,
					"minimumHealthyPercent": 100,
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

			// Register broken task definition (bad image)
			taskDef["containerDefinitions"].([]map[string]interface{})[0]["image"] = "invalid-image:nonexistent"

			result2, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			var brokenTaskDefArn string
			if td, ok := result2["taskDefinition"].(map[string]interface{}); ok {
				brokenTaskDefArn = td["taskDefinitionArn"].(string)
			}

			// Try to update to broken version
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": brokenTaskDefArn,
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for rollback to occur
			time.Sleep(30 * time.Second)

			// Check that service rolled back to working version
			descResult, err := client.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			service := descResult["service"].(map[string]interface{})
			// Task definition should have rolled back to working version
			Expect(service["taskDefinition"]).To(Equal(workingTaskDefArn))

			// Verify tasks are still running
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "RUNNING",
			})
			Expect(err).NotTo(HaveOccurred())

			runningTasks := listResult["taskArns"].([]interface{})
			Expect(len(runningTasks)).To(Equal(2))

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