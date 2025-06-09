package service_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Service Scaling", func() {
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

	Context("when scaling services", func() {
		It("should scale up from 1 to 5 tasks", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-scale-up-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "nginx:alpine",
						"memory":    128,
						"essential": true,
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with 1 task
			serviceName := fmt.Sprintf("test-scale-up-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial task
			time.Sleep(5 * time.Second)

			// Verify we have 1 task
			listResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName": serviceName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listResult["taskArns"].([]interface{})).To(HaveLen(1))

			// Scale up to 5
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 5,
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for scale up
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName": serviceName,
				})
				if err != nil {
					return 0
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 30*time.Second, 2*time.Second).Should(Equal(5))

			// Verify all tasks are running
			listResult, err = client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "RUNNING",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(listResult["taskArns"].([]interface{})).To(HaveLen(5))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should scale down from 5 to 1 task", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-scale-down-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "while true; do echo 'Running'; sleep 5; done"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service with 5 tasks
			serviceName := fmt.Sprintf("test-scale-down-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   5,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for all tasks to start
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName": serviceName,
				})
				if err != nil {
					return 0
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 30*time.Second, 2*time.Second).Should(Equal(5))

			// Scale down to 1
			updateConfig := map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 1,
			}

			_, err = client.UpdateService(updateConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for scale down
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return -1
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 30*time.Second, 2*time.Second).Should(Equal(1))

			// Verify stopped tasks
			stoppedResult, err := client.ListTasks(clusterName, map[string]interface{}{
				"serviceName":   serviceName,
				"desiredStatus": "STOPPED",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(stoppedResult["taskArns"].([]interface{}))).To(Equal(4))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should handle rapid scale up/down cycles", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-rapid-scale-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "echo 'Starting'; sleep 10; echo 'Running'"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := fmt.Sprintf("test-rapid-scale-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   2,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Perform rapid scaling operations
			scalingSequence := []int{1, 4, 2, 5, 3, 1}
			for _, desiredCount := range scalingSequence {
				_, err = client.UpdateService(map[string]interface{}{
					"cluster":      clusterName,
					"service":      serviceName,
					"desiredCount": desiredCount,
				})
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(2 * time.Second)
			}

			// Eventually should stabilize at the last requested count
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return -1
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 60*time.Second, 3*time.Second).Should(Equal(1))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})

		It("should scale to zero and back", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-scale-zero-%d", time.Now().Unix())
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
			serviceName := fmt.Sprintf("test-scale-zero-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   3,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Wait for initial tasks
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName": serviceName,
				})
				if err != nil {
					return 0
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 30*time.Second, 2*time.Second).Should(Equal(3))

			// Scale to zero
			_, err = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for all tasks to stop
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return -1
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 30*time.Second, 2*time.Second).Should(Equal(0))

			// Scale back up
			_, err = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 2,
			})
			Expect(err).NotTo(HaveOccurred())

			// Wait for new tasks
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return 0
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 30*time.Second, 2*time.Second).Should(Equal(2))

			// Cleanup
			_, _ = client.UpdateService(map[string]interface{}{
				"cluster":      clusterName,
				"service":      serviceName,
				"desiredCount": 0,
			})
			_, _ = client.DeleteService(clusterName, serviceName)
		})
	})

	Context("when handling concurrent scaling operations", func() {
		It("should handle concurrent scale requests properly", func() {
			// Register task definition
			taskDefFamily := fmt.Sprintf("test-concurrent-scale-%d", time.Now().Unix())
			taskDef := map[string]interface{}{
				"family": taskDefFamily,
				"containerDefinitions": []map[string]interface{}{
					{
						"name":      "app",
						"image":     "busybox:latest",
						"memory":    128,
						"essential": true,
						"command":   []string{"sh", "-c", "sleep 300"},
					},
				},
				"requiresCompatibilities": []string{"EC2"},
				"networkMode":             "bridge",
			}

			_, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			// Create service
			serviceName := fmt.Sprintf("test-concurrent-scale-%d", time.Now().Unix())
			serviceConfig := map[string]interface{}{
				"cluster":        clusterName,
				"serviceName":    serviceName,
				"taskDefinition": taskDefFamily,
				"desiredCount":   1,
				"launchType":     "EC2",
			}

			_, err = client.CreateService(serviceConfig)
			Expect(err).NotTo(HaveOccurred())

			// Send multiple concurrent scale requests
			done := make(chan bool, 3)
			
			// Request 1: Scale to 4
			go func() {
				_, _ = client.UpdateService(map[string]interface{}{
					"cluster":      clusterName,
					"service":      serviceName,
					"desiredCount": 4,
				})
				done <- true
			}()

			// Request 2: Scale to 2
			go func() {
				time.Sleep(100 * time.Millisecond)
				_, _ = client.UpdateService(map[string]interface{}{
					"cluster":      clusterName,
					"service":      serviceName,
					"desiredCount": 2,
				})
				done <- true
			}()

			// Request 3: Scale to 3
			go func() {
				time.Sleep(200 * time.Millisecond)
				_, _ = client.UpdateService(map[string]interface{}{
					"cluster":      clusterName,
					"service":      serviceName,
					"desiredCount": 3,
				})
				done <- true
			}()

			// Wait for all requests to complete
			for i := 0; i < 3; i++ {
				<-done
			}

			// The final desired count should be from the last request (3)
			time.Sleep(5 * time.Second)
			
			descResult, err := client.DescribeService(clusterName, serviceName)
			Expect(err).NotTo(HaveOccurred())
			
			service := descResult["service"].(map[string]interface{})
			Expect(service["desiredCount"]).To(Equal(float64(3)))

			// Eventually should have 3 running tasks
			Eventually(func() int {
				listResult, err := client.ListTasks(clusterName, map[string]interface{}{
					"serviceName":   serviceName,
					"desiredStatus": "RUNNING",
				})
				if err != nil {
					return -1
				}
				return len(listResult["taskArns"].([]interface{}))
			}, 60*time.Second, 3*time.Second).Should(Equal(3))

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