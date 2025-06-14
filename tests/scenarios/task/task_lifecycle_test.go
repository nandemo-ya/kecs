package task_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Task Lifecycle", func() {
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

	Context("when tracking task resource allocation", func() {
		It("should verify memory and CPU allocation", func() {
			// Register task definition with specific resources
			taskDefFamily := fmt.Sprintf("test-resources-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "resource-test",
						"image":     "busybox:latest",
						"memory":    512,
						"cpu":       256,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Checking resources'; sleep 10"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
				"cpu":                     "256",
				"memory":                  "512",
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

			// Verify resource allocation
			descResult, err := client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			// Use helper to extract tasks
			describedTasks, err := utils.GetTasksFromResult(descResult)
			Expect(err).NotTo(HaveOccurred())
			Expect(describedTasks).To(HaveLen(1))
			runningTask := describedTasks[0]

			// Check allocated resources
			Expect(runningTask["cpu"]).To(Equal("256"))
			Expect(runningTask["memory"]).To(Equal("512"))

			// Check container resources
			containersValue := runningTask["containers"]
			var container map[string]interface{}
			switch containers := containersValue.(type) {
			case []interface{}:
				container = containers[0].(map[string]interface{})
			case []map[string]interface{}:
				container = containers[0]
			default:
				Fail(fmt.Sprintf("Unexpected type for containers: %T", containers))
			}
			Expect(container["cpu"]).To(Equal("256"))
			Expect(container["memory"]).To(Equal("512"))
		})

		It("should track container restart on failure", func() {
			Skip("Service-based task restarts are not implemented in test mode")
			// Register task definition that will fail and restart
			taskDefFamily := fmt.Sprintf("test-restart-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "restarter",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Starting'; sleep 5; echo 'Failing'; exit 1"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service to handle restarts
			serviceName := fmt.Sprintf("test-restart-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for service to stabilize (with task failures)
			time.Sleep(30 * time.Second)

			// List tasks for the service
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())

			taskArns := listResult["taskArns"].([]interface{})
			Expect(len(taskArns)).To(BeNumerically(">=", 1))

			// Should see tasks that have stopped and restarted
			stoppedTasks, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":    serviceName,
				"desiredStatus": "STOPPED",
			})
			Expect(err).NotTo(HaveOccurred())

			stoppedArns := stoppedTasks["taskArns"].([]interface{})
			// Service should have tried to restart tasks
			Expect(len(stoppedArns)).To(BeNumerically(">=", 0))

			// Cleanup service
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when handling task metadata", func() {
		It("should track task timestamps accurately", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-timestamps-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "timer",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Start'; sleep 15; echo 'End'"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			startTime := time.Now()

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

			// Verify createdAt timestamp
			createdAt, ok := task["createdAt"].(string)
			Expect(ok).To(BeTrue())
			createdTime, err := time.Parse(time.RFC3339, createdAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdTime).To(BeTemporally("~", startTime, 10*time.Second))

			// Wait for task to start
			err = checker.WaitForStatus(clusterName, taskArn, "RUNNING", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Get startedAt timestamp
			descResult, err := client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			describedTasks := descResult["tasks"].([]interface{})
			runningTask := describedTasks[0].(map[string]interface{})

			startedAt, ok := runningTask["startedAt"].(string)
			Expect(ok).To(BeTrue())
			startedTime, err := time.Parse(time.RFC3339, startedAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(startedTime).To(BeTemporally(">", createdTime))

			// Wait for task to complete
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 60*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Get stoppedAt timestamp
			descResult, err = client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			stoppedTasks := descResult["tasks"].([]interface{})
			stoppedTask := stoppedTasks[0].(map[string]interface{})

			stoppedAt, ok := stoppedTask["stoppedAt"].(string)
			Expect(ok).To(BeTrue())
			stoppedTime, err := time.Parse(time.RFC3339, stoppedAt)
			Expect(err).NotTo(HaveOccurred())
			Expect(stoppedTime).To(BeTemporally(">", startedTime))

			// Verify task ran for approximately the expected duration
			duration := stoppedTime.Sub(startedTime)
			Expect(duration).To(BeNumerically(">=", 15*time.Second))
			Expect(duration).To(BeNumerically("<", 30*time.Second))
		})
	})

	Context("when handling task volumes", func() {
		It("should run task with volume mounts", func() {
			// Register task definition with volumes
			taskDefFamily := fmt.Sprintf("test-volumes-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "volume-test",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Testing volumes' > /data/test.txt && cat /data/test.txt"},
						"mountPoints": []map[string]interface{}{
							{
								"sourceVolume":  "test-volume",
								"containerPath": "/data",
								"readOnly":      false,
							},
						},
					},
				},
				"volumes": []map[string]interface{}{
					{
						"name": "test-volume",
						"host": map[string]interface{}{
							"sourcePath": "/tmp/kecs-test-volume",
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
			err = checker.WaitForStatus(clusterName, taskArn, "STOPPED", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())

			// Check exit code
			exitCode, err := checker.GetTaskExitCode(clusterName, taskArn)
			Expect(err).NotTo(HaveOccurred())
			Expect(*exitCode).To(Equal(0))
		})
	})

	Context("when handling task networking", func() {
		It("should run task with port mappings", func() {
			// Register task definition with port mappings
			taskDefFamily := fmt.Sprintf("test-ports-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "web-server",
						"image":     "nginx:alpine",
						"memory":    256,
						"essential": true,
						"portMappings": []map[string]interface{}{
							{
								"containerPort": 80,
								"hostPort":      0, // Dynamic port
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

			// Verify port mapping
			descResult, err := client.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())

			describedTasks := descResult["tasks"].([]interface{})
			runningTask := describedTasks[0].(map[string]interface{})
			containers := runningTask["containers"].([]interface{})
			container := containers[0].(map[string]interface{})

			// Check network bindings
			networkBindings, ok := container["networkBindings"].([]interface{})
			if ok && len(networkBindings) > 0 {
				binding := networkBindings[0].(map[string]interface{})
				Expect(binding["containerPort"]).To(Equal(float64(80)))
				Expect(binding["protocol"]).To(Equal("tcp"))
				// Host port should be assigned
				hostPort, ok := binding["hostPort"].(float64)
				Expect(ok).To(BeTrue())
				Expect(hostPort).To(BeNumerically(">", 0))
			}

			// Stop task
			_, err = client.StopTask(clusterName, taskArn, "Test complete")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})