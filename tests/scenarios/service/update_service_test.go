package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Updates", func() {
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

	Context("when updating service configurations", func() {
		var (
			serviceName   string
			taskDefFamily string
		)

		BeforeEach(func() {
			// Register initial task definition
			taskDefFamily = fmt.Sprintf("test-update-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:1.24",
						"memory":    256,
						"cpu":       128,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create initial service
			serviceName = fmt.Sprintf("test-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should update service desired count", func() {
			// Update desired count from 2 to 5
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 5,
			}

			updateResult, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			service := updateResult["service"].(map[string]interface{})
			Expect(service["desiredCount"]).To(Equal(float64(5)))
			Expect(service["serviceName"]).To(Equal(serviceName))
		})

		It("should scale down service", func() {
			// Scale down from 2 to 0
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			}

			updateResult, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			service := updateResult["service"].(map[string]interface{})
			Expect(service["desiredCount"]).To(Equal(float64(0)))
		})

		It("should update service with new task definition revision", func() {
			// Register new revision with updated image
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:1.25", // Updated version
						"memory":    256,
						"cpu":       128,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			regResult, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			newTaskDef := regResult["taskDefinition"].(map[string]interface{})
			newTaskDefArn := newTaskDef["taskDefinitionArn"].(string)

			// Update service to use new task definition
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": newTaskDefArn,
			}

			updateResult, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			service := updateResult["service"].(map[string]interface{})
			Expect(service["taskDefinition"]).To(Equal(newTaskDefArn))

			// Verify deployments show new version being deployed
			deployments := service["deployments"].([]interface{})
			Expect(len(deployments)).To(BeNumerically(">=", 1))

			// Find the new deployment
			var newDeployment map[string]interface{}
			for _, d := range deployments {
				deployment := d.(map[string]interface{})
				if deployment["taskDefinition"] == newTaskDefArn {
					newDeployment = deployment
					break
				}
			}
			Expect(newDeployment).NotTo(BeNil())
			Expect(newDeployment["status"]).To(Equal("PRIMARY"))
		})

		It("should update deployment configuration", func() {
			// Update deployment configuration
			updateConfig := map[string]interface{}{
				"cluster": clusterName,
				"service": serviceName,
				"deploymentConfiguration": map[string]interface{}{
					"maximumPercent":        150,
					"minimumHealthyPercent": 50,
				},
			}

			updateResult, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			service := updateResult["service"].(map[string]interface{})
			deployConfig := service["deploymentConfiguration"].(map[string]interface{})
			Expect(deployConfig["maximumPercent"]).To(Equal(float64(150)))
			Expect(deployConfig["minimumHealthyPercent"]).To(Equal(float64(50)))
		})

		It("should force new deployment without changes", func() {
			// Force new deployment
			updateConfig := map[string]interface{}{
				"cluster":            clusterName,
				"service":            serviceName,
				"forceNewDeployment": true,
			}

			updateResult, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			service := updateResult["service"].(map[string]interface{})
			deployments := service["deployments"].([]interface{})

			// Should have at least 2 deployments (original + forced)
			Expect(len(deployments)).To(BeNumerically(">=", 2))
		})

		It("should update placement constraints", func() {
			// Update with new placement constraints
			updateConfig := map[string]interface{}{
				"cluster": clusterName,
				"service": serviceName,
				"placementConstraints": []map[string]interface{}{
					{
						"type":       "distinctInstance",
						"expression": "",
					},
				},
			}

			updateResult, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			service := updateResult["service"].(map[string]interface{})
			constraints := service["placementConstraints"].([]interface{})
			Expect(constraints).To(HaveLen(1))

			constraint := constraints[0].(map[string]interface{})
			Expect(constraint["type"]).To(Equal("distinctInstance"))
		})
	})

	Context("when handling update errors", func() {
		It("should error when updating non-existent service", func() {
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      "non-existent-service",
				"desiredCount": 5,
			}

			_, err := client.UpdateService(updateConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should error when updating with invalid task definition", func() {
			// Create a service first
			taskDefFamily := fmt.Sprintf("test-invalid-%d", time.Now().Unix())
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

			serviceName := fmt.Sprintf("test-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Try to update with non-existent task definition
			updateConfig := map[string]interface{}{
				"cluster":        clusterName,
				"service":        serviceName,
				"taskDefinition": "non-existent-task-def:99",
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})
})