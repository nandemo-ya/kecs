package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Deletion", func() {
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

	Context("when deleting services", func() {
		var (
			serviceName   string
			taskDefFamily string
		)

		BeforeEach(func() {
			// Register task definition
			taskDefFamily = fmt.Sprintf("test-delete-%d", time.Now().Unix())
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

			// Create service
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

		It("should delete a service successfully", func() {
			// First scale down to 0
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			}
			_, err := client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Delete the service
			deleteResult, err := client.DeleteService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteResult).NotTo(BeNil())

			service := deleteResult["service"].(map[string]interface{})
			Expect(service["serviceName"]).To(Equal(serviceName))
			Expect(service["status"]).To(Equal("DRAINING"))

			// Verify service is no longer in list
			listResult, err := client.ListServices(clusterName)
			Expect(err).NotTo(HaveOccurred())

			serviceArns := listResult["serviceArns"].([]interface{})
			for _, arnInterface := range serviceArns {
				arn := arnInterface.(string)
				Expect(arn).NotTo(ContainSubstring(serviceName))
			}
		})

		It("should force delete a service with running tasks", func() {
			// Delete the service with force flag
			deleteResult, err := client.DeleteServiceForce(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			service := deleteResult["service"].(map[string]interface{})
			Expect(service["status"]).To(Equal("DRAINING"))
			// Desired count should be set to 0 when force deleting
			Expect(service["desiredCount"]).To(Equal(float64(0)))
		})

		It("should handle deleting already deleted service", func() {
			// Delete once
			_, err := client.DeleteServiceForce(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Try to delete again
			_, err = client.DeleteService(clusterName, serviceName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should delete multiple services independently", func() {
			// Create another service
			serviceName2 := fmt.Sprintf("test-service-2-%d", time.Now().Unix())
			serviceConfig2 := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName2,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err := client.CreateService(serviceConfig2)
			Expect(err).NotTo(HaveOccurred())

			// Delete first service
			_, err = client.DeleteServiceForce(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())

			// Second service should still exist
			descResult, err := client.DescribeServices(clusterName, []string{serviceName2})
			Expect(err).NotTo(HaveOccurred())

			services := descResult["services"].([]interface{})
			Expect(services).To(HaveLen(1))

			service2 := services[0].(map[string]interface{})
			Expect(service2["serviceName"]).To(Equal(serviceName2))
			Expect(service2["status"]).To(Equal("ACTIVE"))

			// Clean up second service
			_, err = client.DeleteServiceForce(clusterName, serviceName2)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when handling deletion errors", func() {
		It("should error when deleting non-existent service", func() {
			_, err := client.DeleteService(clusterName, "non-existent-service")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should error when cluster doesn't exist", func() {
			_, err := client.DeleteService("non-existent-cluster", "some-service")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("should require scaling down before deletion without force", func() {
			// Create service with running tasks
			taskDefFamily := fmt.Sprintf("test-running-%d", time.Now().Unix())
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

			serviceName := fmt.Sprintf("test-running-service-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   3,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Try to delete without scaling down or force
			_, err = client.DeleteService(clusterName, serviceName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scale down"))
		})
	})

	Context("when listing services", func() {
		It("should list all active services in cluster", func() {
			// Create multiple services
			var serviceNames []string
			for i := 0; i < 3; i++ {
				// Register task definition
				taskDefFamily := fmt.Sprintf("test-list-%d-%d", time.Now().Unix(), i)
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

				// Create service
				serviceName := fmt.Sprintf("test-list-service-%d-%d", time.Now().Unix(), i)
				serviceNames = append(serviceNames, serviceName)

				serviceConfig := map[string]interface{}{
					"cluster":        clusterName,
					"serviceName":    serviceName,
					"taskDefinition": taskDefFamily,
					"desiredCount":   1,
					"launchType":     "EC2",
				}

				_, err = client.CreateService(serviceConfig)
				Expect(err).NotTo(HaveOccurred())
			}

			// List services
			listResult, err := client.ListServices(clusterName)
			Expect(err).NotTo(HaveOccurred())

			serviceArns := listResult["serviceArns"].([]interface{})
			Expect(len(serviceArns)).To(BeNumerically(">=", 3))

			// Verify all created services are in the list
			arnStrings := make([]string, len(serviceArns))
			for i, arn := range serviceArns {
				arnStrings[i] = arn.(string)
			}

			for _, serviceName := range serviceNames {
				found := false
				for _, arn := range arnStrings {
					if containsString(arn, serviceName) {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), fmt.Sprintf("Service %s not found in list", serviceName))
			}
		})

		It("should describe multiple services", func() {
			// Create services
			var serviceNames []string
			for i := 0; i < 2; i++ {
				taskDefFamily := fmt.Sprintf("test-desc-%d-%d", time.Now().Unix(), i)
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

				serviceName := fmt.Sprintf("test-desc-service-%d-%d", time.Now().Unix(), i)
				serviceNames = append(serviceNames, serviceName)

				serviceConfig := map[string]interface{}{
					"cluster":        clusterName,
					"serviceName":    serviceName,
					"taskDefinition": taskDefFamily,
					"desiredCount":   i + 1,
					"launchType":     "EC2",
				}

				_, err = client.CreateService(serviceConfig)
				Expect(err).NotTo(HaveOccurred())
			}

			// Describe services
			descResult, err := client.DescribeServices(clusterName, serviceNames)
			Expect(err).NotTo(HaveOccurred())

			services := descResult["services"].([]interface{})
			Expect(services).To(HaveLen(2))

			// Verify service details
			service1 := services[0].(map[string]interface{})
			service2 := services[1].(map[string]interface{})

			// Services might be returned in any order
			if service1["serviceName"] == serviceNames[1] {
				service1, service2 = service2, service1
			}

			Expect(service1["serviceName"]).To(Equal(serviceNames[0]))
			Expect(service1["desiredCount"]).To(Equal(float64(1)))

			Expect(service2["serviceName"]).To(Equal(serviceNames[1]))
			Expect(service2["desiredCount"]).To(Equal(float64(2)))
		})
	})
})

func containsString(str, substr string) bool {
	return len(str) >= len(substr) && str[len(str)-len(substr):] == substr
}