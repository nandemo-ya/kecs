package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Creation", func() {
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

	Context("when creating a new service", func() {
		It("should successfully create a service with single task", func() {
			// Register task definition first
			taskDefFamily := fmt.Sprintf("test-nginx-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "nginx",
						"image":     "nginx:alpine",
						"memory":    256,
						"cpu":       128,
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

			// Create service
			serviceName := fmt.Sprintf("test-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			createResult, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
			Expect(createResult).NotTo(BeNil())

			// Verify service response
			service, ok := createResult["service"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(service["serviceName"]).To(Equal(serviceName))
			Expect(service["status"]).To(Equal("ACTIVE"))
			Expect(service["desiredCount"]).To(Equal(float64(1)))
			Expect(service["runningCount"]).To(Equal(float64(0))) // Initially 0
			Expect(service["pendingCount"]).To(Equal(float64(0))) // Initially 0

			// Verify task definition is set correctly
			Expect(service["taskDefinition"]).To(ContainSubstring(taskDefFamily))
		})

		It("should successfully create a service with multiple replicas", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-webapp-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "webapp",
						"image":     "httpd:alpine",
						"memory":    512,
						"cpu":       256,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with 3 replicas
			serviceName := fmt.Sprintf("test-multi-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   3,
				"launchType":     "EC2",
			}

			createResult, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			service := createResult["service"].(map[string]interface{})
			Expect(service["desiredCount"]).To(Equal(float64(3)))
		})

		It("should handle service with placement constraints", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-constraint-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox",
						"memory":    128,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with placement constraints
			serviceName := fmt.Sprintf("test-placement-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
				"placementConstraints": []map[string]interface{}{
					{
						"type":       "memberOf",
						"expression": "attribute:ecs.instance-type == t2.micro",
					},
				},
			}

			createResult, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			service := createResult["service"].(map[string]interface{})
			constraints := service["placementConstraints"].([]interface{})
			Expect(constraints).To(HaveLen(1))

			constraint := constraints[0].(map[string]interface{})
			Expect(constraint["type"]).To(Equal("memberOf"))
			Expect(constraint["expression"]).To(Equal("attribute:ecs.instance-type == t2.micro"))
		})

		It("should create service with deployment configuration", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-deployment-%d", time.Now().Unix())
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

			// Create service with deployment configuration
			serviceName := fmt.Sprintf("test-deploy-config-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        200,
					"minimumHealthyPercent": 100,
				},
			}

			createResult, err := client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			service := createResult["service"].(map[string]interface{})
			deployConfig := service["deploymentConfiguration"].(map[string]interface{})
			Expect(deployConfig["maximumPercent"]).To(Equal(float64(200)))
			Expect(deployConfig["minimumHealthyPercent"]).To(Equal(float64(100)))
		})
	})

	Context("when handling errors", func() {
		It("should reject service creation without task definition", func() {
			serviceConfig := map[string]interface{}{
				"cluster":      clusterName,
				"serviceName":  "invalid-service",
				"desiredCount": 1,
			}

			_, err := client.CreateService(serviceConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("taskDefinition"))
		})

		It("should reject service with non-existent task definition", func() {
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    "invalid-service",
				"taskDefinition": "non-existent-task-def",
				"desiredCount":   1,
			}

			_, err := client.CreateService(serviceConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should reject duplicate service names in same cluster", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-dup-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "nginx",
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

			// Create first service
			serviceName := fmt.Sprintf("test-dup-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Try to create duplicate service
			_, err = client.CreateService(serviceConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already exists"))
		})

		It("should reject invalid desired count", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-count-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "nginx",
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

			// Try negative desired count
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    "invalid-count-service",
				"taskDefinition": taskDefFamily,
				"desiredCount":   -1,
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).To(HaveOccurred())
		})
	})
})