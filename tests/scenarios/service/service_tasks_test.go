package service_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Tasks", func() {
	var (
		kecs    *utils.KECSContainer
		client  *utils.ECSClient
		checker *utils.TaskStatusChecker
		
		clusterName string
		taskDefArn  string
		serviceName string
	)

	BeforeEach(func() {
		// Start KECS container
		kecs = utils.StartKECS(GinkgoT())
		client = utils.NewECSClient(kecs.Endpoint())
		checker = utils.NewTaskStatusChecker(client)
		
		// Create a test cluster
		clusterName = "test-cluster-" + utils.GenerateRandomString(8)
		err := client.CreateCluster(clusterName)
		Expect(err).NotTo(HaveOccurred())
		
		// Register a task definition
		taskDef := map[string]interface{}{
			"family": "test-service-task",
			"networkMode": "bridge",
			"containerDefinitions": []interface{}{
				map[string]interface{}{
					"name":   "test-container",
					"image":  "nginx:alpine",
					"memory": 128,
					"cpu":    256,
					"essential": true,
					"portMappings": []interface{}{
						map[string]interface{}{
							"containerPort": 80,
							"protocol":      "tcp",
						},
					},
				},
			},
		}
		
		result, err := client.RegisterTaskDefinition(taskDef)
		Expect(err).NotTo(HaveOccurred())
		
		// Extract task definition ARN
		if taskDefInfo, ok := result["taskDefinition"].(map[string]interface{}); ok {
			taskDefArn = taskDefInfo["taskDefinitionArn"].(string)
		}
		Expect(taskDefArn).NotTo(BeEmpty())
		
		serviceName = "test-service-" + utils.GenerateRandomString(8)
	})

	AfterEach(func() {
		// Force delete service if exists
		_, _ = client.DeleteServiceForce(clusterName, serviceName)
		
		// Delete cluster
		_ = client.DeleteCluster(clusterName)
		
		// Cleanup container
		kecs.Cleanup()
	})

	Context("service task management", func() {
		It("should launch correct number of tasks for service", func() {
			// Create service with desired count of 3
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefArn,
				"desiredCount":   3,
			}
			
			result, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Extract service information
			service, ok := result["service"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(service["serviceName"]).To(Equal(serviceName))
			Expect(service["desiredCount"]).To(Equal(float64(3)))
			
			// Wait for service to stabilize
			time.Sleep(5 * time.Second)
			
			// List tasks for the service
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			taskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(taskArns).To(HaveLen(3))
			
			// Wait for all tasks to reach RUNNING status
			for _, arn := range taskArns {
				taskArn := arn.(string)
				err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 60*time.Second)
				Expect(err).NotTo(HaveOccurred())
			}
			
			// Verify all tasks have correct configuration
			descResult, err := client.DescribeTasks(clusterName, utils.InterfaceSliceToStringSlice(taskArns))
			Expect(err).NotTo(HaveOccurred())
			
			tasks, ok := descResult["tasks"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(tasks).To(HaveLen(3))
			
			for _, t := range tasks {
				task := t.(map[string]interface{})
				// Verify task belongs to our service
				if group, ok := task["group"].(string); ok {
					Expect(group).To(Equal("service:" + serviceName))
				}
				// Verify task is using correct task definition
				if taskDef, ok := task["taskDefinitionArn"].(string); ok {
					Expect(taskDef).To(Equal(taskDefArn))
				}
			}
		})
		
		It("should replace failed tasks automatically", func() {
			// Create service with 2 tasks
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefArn,
				"desiredCount":   2,
			}
			
			_, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for service to stabilize
			time.Sleep(5 * time.Second)
			
			// List initial tasks
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			initialTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(initialTaskArns).To(HaveLen(2))
			
			// Wait for all initial tasks to reach RUNNING status
			for _, arn := range initialTaskArns {
				taskArn := arn.(string)
				err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 60*time.Second)
				Expect(err).NotTo(HaveOccurred())
			}
			
			// Stop one of the tasks to simulate failure
			stoppedTaskArn := initialTaskArns[0].(string)
			_, err = client.StopTask(clusterName, stoppedTaskArn, "Simulating task failure")
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for the stopped task to reach STOPPED status
			err = checker.WaitForStatus(clusterName, stoppedTaskArn, "STOPPED", 60*time.Second)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for service to launch replacement task
			time.Sleep(10 * time.Second)
			
			// List tasks again
			listResult, err = client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "RUNNING",
			})
			Expect(err).NotTo(HaveOccurred())
			
			runningTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			// Should have 2 running tasks (1 original + 1 replacement)
			Expect(runningTaskArns).To(HaveLen(2))
			
			// Verify the replacement task is different from the stopped one
			Expect(runningTaskArns).NotTo(ContainElement(stoppedTaskArn))
			
			// Wait for all running tasks to be in RUNNING status
			for _, arn := range runningTaskArns {
				taskArn := arn.(string)
				status, err := checker.GetCurrentStatus(clusterName, taskArn)
				Expect(err).NotTo(HaveOccurred())
				if status.Status != "RUNNING" {
					err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 60*time.Second)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
		
		It("should handle scaling up service tasks", func() {
			// Create service with 1 task initially
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefArn,
				"desiredCount":   1,
			}
			
			_, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for initial task to start
			time.Sleep(5 * time.Second)
			
			// Verify initial task count
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			initialTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(initialTaskArns).To(HaveLen(1))
			
			// Wait for initial task to be running
			err = checker.WaitForStatus(clusterName, initialTaskArns[0].(string), "RUNNING", 60*time.Second)
			Expect(err).NotTo(HaveOccurred())
			
			// Scale up to 4 tasks
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 4,
			}
			
			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for scale up to complete
			time.Sleep(10 * time.Second)
			
			// List tasks again
			listResult, err = client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			scaledTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(scaledTaskArns).To(HaveLen(4))
			
			// Wait for all tasks to reach RUNNING status
			for _, arn := range scaledTaskArns {
				taskArn := arn.(string)
				status, err := checker.GetCurrentStatus(clusterName, taskArn)
				Expect(err).NotTo(HaveOccurred())
				if status.Status != "RUNNING" {
					err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 60*time.Second)
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
		
		It("should handle scaling down service tasks", func() {
			// Create service with 4 tasks initially
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefArn,
				"desiredCount":   4,
			}
			
			_, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for all tasks to start
			time.Sleep(10 * time.Second)
			
			// Verify initial task count
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			initialTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(initialTaskArns).To(HaveLen(4))
			
			// Wait for all initial tasks to be running
			for _, arn := range initialTaskArns {
				taskArn := arn.(string)
				err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 60*time.Second)
				Expect(err).NotTo(HaveOccurred())
			}
			
			// Scale down to 1 task
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 1,
			}
			
			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for scale down to complete
			time.Sleep(10 * time.Second)
			
			// List running tasks
			listResult, err = client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "RUNNING",
			})
			Expect(err).NotTo(HaveOccurred())
			
			runningTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(runningTaskArns).To(HaveLen(1))
			
			// List stopped tasks
			listResult, err = client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "STOPPED",
			})
			Expect(err).NotTo(HaveOccurred())
			
			stoppedTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(stoppedTaskArns).To(HaveLen(3))
		})
		
		It("should maintain service tasks across task definition updates", func() {
			// Create service with 2 tasks
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefArn,
				"desiredCount":   2,
			}
			
			_, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Wait for tasks to start
			time.Sleep(5 * time.Second)
			
			// List initial tasks
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			initialTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(initialTaskArns).To(HaveLen(2))
			
			// Register a new version of the task definition
			newTaskDef := map[string]interface{}{
				"family": "test-service-task",
				"networkMode": "bridge",
				"containerDefinitions": []interface{}{
					map[string]interface{}{
						"name":   "test-container",
						"image":  "nginx:alpine",
						"memory": 256, // Changed memory
						"cpu":    256,
						"essential": true,
						"portMappings": []interface{}{
							map[string]interface{}{
								"containerPort": 80,
								"protocol":      "tcp",
							},
						},
						"environment": []interface{}{
							map[string]interface{}{
								"name":  "VERSION",
								"value": "v2",
							},
						},
					},
				},
			}
			
			newResult, err := client.RegisterTaskDefinition(newTaskDef)
			Expect(err).NotTo(HaveOccurred())
			
			var newTaskDefArn string
			if taskDefInfo, ok := newResult["taskDefinition"].(map[string]interface{}); ok {
				newTaskDefArn = taskDefInfo["taskDefinitionArn"].(string)
			}
			Expect(newTaskDefArn).NotTo(BeEmpty())
			Expect(newTaskDefArn).NotTo(Equal(taskDefArn))
			
			// Update service with new task definition
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": newTaskDefArn,
			}
			
			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())
			
			// Service should maintain desired count during update
			// Wait for update to progress
			time.Sleep(15 * time.Second)
			
			// List all tasks (running and pending)
			listResult, err = client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			
			allTaskArns, ok := listResult["taskArns"].([]interface{})
			Expect(ok).To(BeTrue())
			// Should have at least 2 tasks (might have more during rolling update)
			Expect(len(allTaskArns)).To(BeNumerically(">=", 2))
			
			// Eventually all tasks should use the new task definition
			Eventually(func() bool {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return false
				}
				
				runningArns, ok := listResult["taskArns"].([]interface{})
				if !ok || len(runningArns) != 2 {
					return false
				}
				
				// Check if all running tasks use new task definition
				descResult, err := client.DescribeTasks(clusterName, utils.InterfaceSliceToStringSlice(runningArns))
				if err != nil {
					return false
				}
				
				tasks, ok := descResult["tasks"].([]interface{})
				if !ok {
					return false
				}
				
				for _, t := range tasks {
					task := t.(map[string]interface{})
					if taskDef, ok := task["taskDefinitionArn"].(string); ok {
						if taskDef != newTaskDefArn {
							return false
						}
					}
				}
				
				return true
			}, 60*time.Second, 5*time.Second).Should(BeTrue())
		})
	})
})