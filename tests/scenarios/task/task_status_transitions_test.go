package task_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Task Status Transitions", func() {
	var (
		kecs        *utils.KECSContainer
		client      *utils.ECSClient
		clusterName string
		checker     *utils.TaskStatusChecker
	)

	BeforeEach(func() {
		kecs = utils.StartKECS(GinkgoT())
		client = utils.NewECSClient(kecs.Endpoint())
		checker = utils.NewTaskStatusChecker(client)

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

	Context("when tracking normal task lifecycle", func() {
		It("should track transitions for successful task completion", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-success-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "success",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Task starting'; sleep 5; echo 'Task completed'; exit 0"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          1,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			if err != nil {
				// Get container logs for debugging
				logs, _ := kecs.GetLogs()
				GinkgoWriter.Printf("KECS Container Logs:\n%s\n", logs)
			}
			Expect(err).NotTo(HaveOccurred())

			tasks := result["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			taskArn := task["taskArn"].(string)

			// Track initial status
			initialStatus := task["lastStatus"].(string)
			Expect(initialStatus).To(BeElementOf("PROVISIONING", "PENDING"))

			// Wait for task to complete
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 60*time.Second)
			if err != nil {
				// Get container logs for debugging
				logs, _ := kecs.GetLogs()
				GinkgoWriter.Printf("KECS Container Logs:\n%s\n", logs)
			}
			Expect(err).NotTo(HaveOccurred())

			// Validate transitions
			err = checker.ValidateTransitions(taskArn)
			Expect(err).NotTo(HaveOccurred())

			// Check exit code
			exitCode, err := checker.GetTaskExitCode(clusterName, taskArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(*exitCode).To(Equal(0))

			// Verify status history
			history := checker.GetStatusHistory(taskArn)
			Expect(len(history)).To(BeNumerically(">=", 2))

			// Last status should be STOPPED
			Expect(history[len(history)-1].Status).To(Equal("STOPPED"))
		})

		It("should track transitions for failed task", func() {
			// Register task definition that will fail
			taskDefFamily := fmt.Sprintf("test-fail-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "fail",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Task will fail'; exit 1"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          1,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			Expect(err).NotTo(HaveOccurred())

			tasks := result["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			taskArn := task["taskArn"].(string)

			// Wait for task to fail
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 60*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Validate transitions
			err = checker.ValidateTransitions(taskArn)
			Expect(err).NotTo(HaveOccurred())

			// Check exit code (should be 1)
			exitCode, err := checker.GetTaskExitCode(clusterName, taskArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(*exitCode).To(Equal(1))

			// Check stopped reason
			descResult, err := client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			describedTasks := descResult["tasks"].([]interface{})
			stoppedTask := describedTasks[0].(map[string]interface{})
			Expect(stoppedTask["stoppedReason"]).To(ContainSubstring("Essential container"))
		})
	})

	Context("when tracking container health", func() {
		It("should track container health status", func() {
			// Register task definition with health check
			taskDefFamily := fmt.Sprintf("test-health-%d", time.Now().Unix())
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
							"interval":    10,
							"timeout":     5,
							"retries":     3,
							"startPeriod": 30,
						},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          1,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			Expect(err).NotTo(HaveOccurred())

			tasks := result["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			taskArn := task["taskArn"].(string)

			// Wait for task to start
			err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Check task health
			Eventually(func() bool {
				healthy, _ := checker.CheckTaskHealth(clusterName, taskArn)
				return healthy
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			// Stop task
			_, err = client.StopTask(clusterName, taskArn, "Test complete")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when handling edge cases", func() {
		It("should handle rapid state changes", func() {
			// Register a very quick task
			taskDefFamily := fmt.Sprintf("test-rapid-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "rapid",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"true"}, // Exits immediately with success
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          1,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			Expect(err).NotTo(HaveOccurred())

			tasks := result["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			taskArn := task["taskArn"].(string)

			// Task might stop very quickly
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Should still have valid transitions
			err = checker.ValidateTransitions(taskArn)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle tasks with multiple containers", func() {
			// Register multi-container task definition
			taskDefFamily := fmt.Sprintf("test-multi-container-%d", time.Now().Unix())
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
					},
					{
						"name":      "sidecar",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": false,
						"command":   []string{"sh", "-c", "while true; do echo 'Sidecar running'; sleep 10; done"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Run task
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          1,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			Expect(err).NotTo(HaveOccurred())

			tasks := result["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			taskArn := task["taskArn"].(string)

			// Wait for task to start
			err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Verify both containers are running
			descResult, err := client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			describedTasks := descResult["tasks"].([]interface{})
			runningTask := describedTasks[0].(map[string]interface{})
			containers := runningTask["containers"].([]interface{})
			Expect(containers).To(HaveLen(2))

			// Stop task
			_, err = client.StopTask(clusterName, taskArn, "Test complete")
			Expect(err).NotTo(HaveOccurred())

			// Wait for stop
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Validate transitions
			err = checker.ValidateTransitions(taskArn)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})