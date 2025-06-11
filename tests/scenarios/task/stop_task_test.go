package task_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("StopTask Operations", func() {
	var (
		kecs        *utils.KECSContainer
		client      *utils.ECSClient
		clusterName string
		checker     *utils.TaskStatusChecker
	)

	BeforeEach(func() {
		kecs = utils.StartKECS(GinkgoT())
		client = utils.NewECSClient(kecs.Endpoint())
		checker = utils.NewTaskStatusChecker(client.CurlClient)

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

	Context("when stopping running tasks", func() {
		var (
			taskDefFamily string
			runningTaskArn string
		)

		BeforeEach(func() {
			// Register a long-running task definition
			taskDefFamily = fmt.Sprintf("test-long-running-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "sleeper",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "while true; do echo 'Running...'; sleep 5; done"},
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
			runningTaskArn = task["taskArn"].(string)

			// Wait for task to start
			err = checker.WaitForStatus(clusterName, runningTaskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should successfully stop a running task", func() {
			// Stop the task
			stopResult, err := client.StopTask(clusterName, runningTaskArn, "Test stop")
			Expect(err).NotTo(HaveOccurred())

			// Verify response
			stoppedTask, ok := stopResult["task"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(stoppedTask["taskArn"]).To(Equal(runningTaskArn))
			Expect(stoppedTask["desiredStatus"]).To(Equal("STOPPED"))

			// Wait for task to stop
			err = checker.WaitForStatus(clusterName, runningTaskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Verify task stopped
			descResult, err := client.DescribeTasks(clusterName, []string{runningTaskArn})
			Expect(err).NotTo(HaveOccurred())

			tasks := descResult["tasks"].([]interface{})
			task := tasks[0].(map[string]interface{})
			Expect(task["lastStatus"]).To(Equal("STOPPED"))
			Expect(task["stoppedReason"]).To(ContainSubstring("Test stop"))
		})

		It("should stop multiple tasks", func() {
			// Run multiple tasks
			runTaskConfig := map[string]interface{}{
				"cluster":        clusterName,
				"taskDefinition": taskDefFamily,
				"count":          3,
				"launchType":     "EC2",
			}

			result, err := client.RunTask(runTaskConfig)
			Expect(err).NotTo(HaveOccurred())

			tasks := result["tasks"].([]interface{})
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

			// Stop all tasks
			for i, taskArn := range taskArns {
				reason := fmt.Sprintf("Stopping task %d", i+1)
				_, err = client.StopTask(clusterName, taskArn, reason)
				Expect(err).NotTo(HaveOccurred())
			}

			// Wait for all tasks to stop
			for _, taskArn := range taskArns {
				err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 30*time.Second)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should handle stop without reason", func() {
			// Stop without providing reason
			stopResult, err := client.StopTask(clusterName, runningTaskArn, "")
			Expect(err).NotTo(HaveOccurred())

			task := stopResult["task"].(map[string]interface{})
			Expect(task["desiredStatus"]).To(Equal("STOPPED"))

			// Wait for task to stop
			err = checker.WaitForStatus(clusterName, runningTaskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when handling stopping states", func() {
		It("should track status transitions during stop", func() {
			// Register and run a task
			taskDefFamily := fmt.Sprintf("test-stop-transition-%d", time.Now().Unix())
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

			// Wait for running state
			err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Stop the task
			_, err = client.StopTask(clusterName, taskArn, "Testing transitions")
			Expect(err).NotTo(HaveOccurred())

			// Wait for stopped state
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Validate status transitions
			err = checker.ValidateTransitions(taskArn)
			Expect(err).NotTo(HaveOccurred())

			// Check status history includes expected states
			history := checker.GetStatusHistory(taskArn)
			Expect(len(history)).To(BeNumerically(">=", 2))

			// Should have transitioned through some states
			statuses := []string{}
			for _, status := range history {
				statuses = append(statuses, status.Status)
			}
			Expect(statuses).To(ContainElement("RUNNING"))
			Expect(statuses[len(statuses)-1]).To(Equal("STOPPED"))
		})
	})

	Context("when handling errors", func() {
		It("should error when stopping non-existent task", func() {
			nonExistentArn := "arn:aws:ecs:us-east-1:123456789012:task/non-existent-task"
			_, err := client.StopTask(clusterName, nonExistentArn, "Test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should handle stopping already stopped task", func() {
			// Register and run a quick task
			taskDefFamily := fmt.Sprintf("test-quick-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "quick",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"echo", "Done"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

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

			// Wait for task to stop naturally
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 60*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Try to stop already stopped task
			stopResult, err := client.StopTask(clusterName, taskArn, "Already stopped")
			// This might succeed (idempotent) or return an error
			if err == nil {
				task := stopResult["task"].(map[string]interface{})
				Expect(task["lastStatus"]).To(Equal("STOPPED"))
			} else {
				Expect(err.Error()).To(ContainSubstring("already stopped"))
			}
		})
	})
})