package phase3

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("KECS Basic Functionality", func() {
	var clusterName string

	BeforeEach(func() {
		// Create a unique cluster for this test
		var err error
		clusterName, err = sharedClusterManager.GetOrCreateCluster("phase3")
		Expect(err).NotTo(HaveOccurred())
		GinkgoT().Logf("Testing basic functionality for cluster: %s", clusterName)
	})

	Context("Basic Operations", func() {
		It("should register and describe task definitions", func() {
			// Create a simple task definition
			taskDefJSON := `{
				"family": "basic-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [{
					"name": "simple-container",
					"image": "busybox:latest",
					"cpu": 256,
					"memory": 512,
					"essential": true,
					"command": ["echo", "Hello KECS"]
				}]
			}`

			// Register task definition
			taskDef, err := sharedClient.RegisterTaskDefinitionFromJSON(taskDefJSON)
			Expect(err).NotTo(HaveOccurred())
			Expect(taskDef).NotTo(BeNil())
			Expect(taskDef.Family).To(Equal("basic-test"))
			Expect(taskDef.Revision).To(Equal(1))

			// Verify task definition was registered
			// In test mode, we just verify the registration succeeded

			// Describe the task definition
			describedTaskDef, err := sharedClient.DescribeTaskDefinition("basic-test:1")
			Expect(err).NotTo(HaveOccurred())
			Expect(describedTaskDef).NotTo(BeNil())
			Expect(describedTaskDef.Family).To(Equal("basic-test"))
		})

		It("should create and list services", func() {
			// First register a task definition
			taskDefJSON := `{
				"family": "service-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [{
					"name": "service-container",
					"image": "nginx:alpine",
					"cpu": 256,
					"memory": 512,
					"essential": true,
					"portMappings": [{
						"containerPort": 80,
						"protocol": "tcp"
					}]
				}]
			}`

			_, err := sharedClient.RegisterTaskDefinitionFromJSON(taskDefJSON)
			Expect(err).NotTo(HaveOccurred())

			// Create a service
			err = sharedClient.CreateService(clusterName, "test-service", "service-test", 1)
			Expect(err).NotTo(HaveOccurred())

			// List services
			services, err := sharedClient.ListServices(clusterName)
			Expect(err).NotTo(HaveOccurred())
			Expect(services).To(ContainElement(ContainSubstring("test-service")))

			// Verify service was created by checking the list
			// Service details verification is limited in test mode
		})

		It("should handle task operations", func() {
			// Register a task definition
			taskDefJSON := `{
				"family": "task-ops-test",
				"networkMode": "bridge",
				"requiresCompatibilities": ["EC2"],
				"cpu": "256",
				"memory": "512",
				"containerDefinitions": [{
					"name": "task-container",
					"image": "busybox:latest",
					"cpu": 256,
					"memory": 512,
					"essential": true,
					"command": ["sleep", "300"]
				}]
			}`

			_, err := sharedClient.RegisterTaskDefinitionFromJSON(taskDefJSON)
			Expect(err).NotTo(HaveOccurred())

			// Run a task
			runResp, err := sharedClient.RunTask(clusterName, "task-ops-test", 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(runResp.Tasks).To(HaveLen(1))
			Expect(runResp.Tasks[0].TaskArn).NotTo(BeEmpty())

			taskArn := runResp.Tasks[0].TaskArn

			// List tasks
			taskArns, err := sharedClient.ListTasks(clusterName, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(taskArns).To(ContainElement(taskArn))

			// Describe the task
			tasks, err := sharedClient.DescribeTasks(clusterName, []string{taskArn})
			Expect(err).NotTo(HaveOccurred())
			Expect(tasks).To(HaveLen(1))
			Expect(tasks[0].TaskArn).To(Equal(taskArn))
			Expect(tasks[0].LastStatus).To(Or(Equal("PENDING"), Equal("RUNNING")))

			// In test mode, tasks won't actually transition to STOPPED
			// Just verify the task was created successfully
			Expect(tasks[0].TaskDefinitionArn).To(ContainSubstring("task-ops-test"))
		})
	})
})
