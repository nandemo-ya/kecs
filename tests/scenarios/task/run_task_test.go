package task_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("RunTask Operations", func() {
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

	Context("when running a simple task", func() {
		It("should successfully run a task with simple container", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-simple-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "busybox",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"echo", "Hello from KECS"},
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

			// Verify response
			tasks, ok := result["tasks"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(tasks).To(HaveLen(1))

			task := tasks[0].(map[string]interface{})
			Expect(task["taskDefinitionArn"]).To(ContainSubstring(taskDefFamily))
			Expect(task["clusterArn"]).To(ContainSubstring(clusterName))
			Expect(task["lastStatus"]).To(BeElementOf("PROVISIONING", "PENDING"))
			Expect(task["desiredStatus"]).To(Equal("RUNNING"))

			// Extract task ARN
			taskArn := task["taskArn"].(string)
			Expect(taskArn).NotTo(BeEmpty())

			// Wait for task to start
			err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Verify task is running
			descResult, err := client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			describedTasks := descResult["tasks"].([]interface{})
			Expect(describedTasks).To(HaveLen(1))

			runningTask := describedTasks[0].(map[string]interface{})
			Expect(runningTask["lastStatus"]).To(Equal("RUNNING"))
		})

		It("should run multiple tasks with count parameter", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-multi-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "nginx",
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
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Run multiple tasks
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          3,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			if err != nil {
				// Get container logs for debugging
				logs, _ := kecs.GetLogs()
				GinkgoWriter.Printf("KECS Container Logs:\n%s\n", logs)
			}
			Expect(err).NotTo(HaveOccurred())

			// Verify 3 tasks were created
			tasks := result["tasks"].([]interface{})
			if len(tasks) != 3 {
				// Get container logs for debugging
				logs, _ := kecs.GetLogs()
				GinkgoWriter.Printf("Expected 3 tasks but got %d. KECS Container Logs:\n%s\n", len(tasks), logs)
			}
			Expect(tasks).To(HaveLen(3))

			// Collect task ARNs
			taskArns := []string{}
			for _, t := range tasks {
				task := t.(map[string]interface{})
				taskArns = append(taskArns, task["taskArn"].(string))
			}

			// Wait for all tasks to start
			for _, taskArn := range taskArns {
				err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle task with environment variables", func() {
			// Register task definition with env vars
			taskDefFamily := fmt.Sprintf("test-env-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo $MESSAGE && echo $VERSION"},
						"environment": []map[string]interface{}{
							{
								"name":  "MESSAGE",
								"value": "Hello from KECS",
							},
							{
								"name":  "VERSION",
								"value": "1.0.0",
							},
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

			// Wait for task to complete
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 60*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Check exit code
			exitCode, err := checker.GetTaskExitCode(clusterName, taskArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(*exitCode).To(Equal(0))
		})
	})

	Context("when handling errors", func() {
		It("should reject running task with non-existent task definition", func() {
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": "non-existent-task-def",
				"count":          1,
			}

			_, err := client.RunTask(runTaskConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should reject running task without required fields", func() {
			// Missing taskDefinition
			runTaskConfig := map[string]interface{}{
				"cluster": clusterName,
				"count":   1,
			}

			_, err := client.RunTask(runTaskConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("taskDefinition"))
		})

		It("should handle task placement failures gracefully", func() {
			// Register task definition with excessive resource requirements
			taskDefFamily := fmt.Sprintf("test-no-resources-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "resource-hog",
						"image":     "busybox:latest",
						"memory":    999999, // Excessive memory
						"cpu":       999999, // Excessive CPU
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Try to run task
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          1,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			Expect(err).NotTo(HaveOccurred())

			// Check for failures
			failures, ok := result["failures"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(failures).To(HaveLen(1))

			failure := failures[0].(map[string]interface{})
			Expect(failure["reason"]).To(ContainSubstring("RESOURCE"))
		})
	})
})