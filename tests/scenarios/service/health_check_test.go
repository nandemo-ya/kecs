package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Health Checks", func() {
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

	Context("when services have container health checks", func() {
		It("should respect health check configuration", func() {
			// Register task definition with health check
			taskDefFamily := fmt.Sprintf("test-health-check-%d", time.Now().Unix())
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
							"interval":    30,
							"timeout":     5,
							"retries":     3,
							"startPeriod": 60,
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			result, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Verify health check was registered
			taskDefinition := result["taskDefinition"].(map[string]interface{})
			containerDefs := taskDefinition["containerDefinitions"].([]interface{})
			containerDef := containerDefs[0].(map[string]interface{})
			healthCheck := containerDef["healthCheck"].(map[string]interface{})

			Expect(healthCheck["interval"]).To(Equal(float64(30)))
			Expect(healthCheck["timeout"]).To(Equal(float64(5)))
			Expect(healthCheck["retries"]).To(Equal(float64(3)))
			Expect(healthCheck["startPeriod"]).To(Equal(float64(60)))

			// Create service
			serviceName := fmt.Sprintf("test-health-check-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for tasks to start
			time.Sleep(10 * time.Second)

			// List tasks and verify they're running
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())

			taskArns := listResult["taskArns"].([]interface{})
			Expect(taskArns).To(HaveLen(2))

			// Describe tasks to check health status
			descResult, err := client.DescribeTasks(clusterName, utils.InterfaceSliceToStringSlice(taskArns))
			Expect(err).NotTo(HaveOccurred())

			tasks := descResult["tasks"].([]interface{})
			for _, t := range tasks {
				task := t.(map[string]interface{})
				// During start period, health status should be UNKNOWN
				if healthStatus, ok := task["healthStatus"].(string); ok {
					Expect([]string{"UNKNOWN", "HEALTHY"}).To(ContainElement(healthStatus))
				}
			}

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should handle health check grace period", func() {
			// Register task definition with short grace period
			taskDefFamily := fmt.Sprintf("test-grace-period-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command": []string{
							"sh", "-c",
							"echo 'Starting...'; sleep 5; echo 'Ready'; while true; do echo 'Healthy'; sleep 5; done",
						},
						"healthCheck": map[string]interface{}{
							"command":     []string{"CMD-SHELL", "test -f /tmp/ready || exit 1"},
							"interval":    5,
							"timeout":     2,
							"retries":     1,
							"startPeriod": 30, // 30 second grace period
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with health check grace period
			serviceName := fmt.Sprintf("test-grace-period-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":               clusterName,
				"serviceName":           serviceName,
				"taskDefinition":        taskDefFamily,
				"desiredCount":          1,
				"launchType":            "EC2",
				"healthCheckGracePeriod": 60, // Service-level grace period
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Check task health status during grace period
			time.Sleep(5 * time.Second)

			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())

			taskArns := listResult["taskArns"].([]interface{})
			Expect(taskArns).To(HaveLen(1))

			// During grace period, task should not be replaced even if unhealthy
			descResult, err := client.DescribeTasks(clusterName, utils.InterfaceSliceToStringSlice(taskArns))
			Expect(err).NotTo(HaveOccurred())

			tasks := descResult["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			lastStatus := task["lastStatus"].(string)
			Expect(lastStatus).To(Equal("RUNNING"))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when tasks fail health checks", func() {
		It("should replace unhealthy tasks", func() {
			// Register task definition that will fail health check
			taskDefFamily := fmt.Sprintf("test-unhealthy-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "failing-app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command": []string{
							"sh", "-c",
							"echo 'Starting...'; sleep 10; echo 'Will fail health checks'; while true; do sleep 5; done",
						},
						"healthCheck": map[string]interface{}{
							"command":     []string{"CMD-SHELL", "exit 1"}, // Always fails
							"interval":    5,
							"timeout":     2,
							"retries":     2,
							"startPeriod": 15, // Short start period
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := fmt.Sprintf("test-unhealthy-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Get initial tasks
			time.Sleep(5 * time.Second)
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())

			initialTaskArns := listResult["taskArns"].([]interface{})

			// Wait for health checks to fail and tasks to be replaced
			time.Sleep(30 * time.Second)

			// Check for stopped tasks (should be the initial unhealthy ones)
			stoppedResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "STOPPED",
			})
			Expect(err).NotTo(HaveOccurred())

			stoppedTaskArns := stoppedResult["taskArns"].([]interface{})
			// Should have stopped tasks due to health check failures
			Expect(len(stoppedTaskArns)).To(BeNumerically(">", 0))

			// Verify stopped tasks include initial tasks
			for _, arn := range initialTaskArns {
				Expect(stoppedTaskArns).To(ContainElement(arn))
			}

			// Should still have desired count running (replacements)
			runningResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "RUNNING",
			})
			Expect(err).NotTo(HaveOccurred())

			runningTaskArns := runningResult["taskArns"].([]interface{})
			Expect(runningTaskArns).To(HaveLen(2))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when using multiple health check types", func() {
		It("should support different health check commands", func() {
			// Register task definition with multiple containers and health checks
			taskDefFamily := fmt.Sprintf("test-multi-health-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "web",
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
							"interval":    10,
							"timeout":     3,
							"retries":     2,
							"startPeriod": 20,
						},
					},
					{
						"name":      "sidecar",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": false,
						"command":   []string{"sh", "-c", "while true; do echo 'Sidecar running'; sleep 5; done"},
						"healthCheck": map[string]interface{}{
							"command":     []string{"CMD", "true"}, // Simple command health check
							"interval":    15,
							"timeout":     5,
							"retries":     1,
							"startPeriod": 0,
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			result, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Verify both health checks were registered
			taskDefinition := result["taskDefinition"].(map[string]interface{})
			containerDefs := taskDefinition["containerDefinitions"].([]interface{})
			Expect(containerDefs).To(HaveLen(2))

			// Check web container health check
			webContainer := containerDefs[0].(map[string]interface{})
			webHealthCheck := webContainer["healthCheck"].(map[string]interface{})
			Expect(webHealthCheck["command"].([]interface{})[0]).To(Equal("CMD-SHELL"))

			// Check sidecar container health check
			sidecarContainer := containerDefs[1].(map[string]interface{})
			sidecarHealthCheck := sidecarContainer["healthCheck"].(map[string]interface{})
			Expect(sidecarHealthCheck["command"].([]interface{})[0]).To(Equal("CMD"))

			// Create service
			serviceName := fmt.Sprintf("test-multi-health-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for task to start
			time.Sleep(10 * time.Second)

			// Verify task is running
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())

			taskArns := listResult["taskArns"].([]interface{})
			Expect(taskArns).To(HaveLen(1))

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