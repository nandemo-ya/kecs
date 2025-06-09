package task_definition_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Task Definition Revision and Deregister", func() {
	var (
		kecs   *utils.KECSContainer
		client *utils.ECSClient
	)

	BeforeEach(func() {
		kecs = utils.StartKECS(GinkgoT())
		client = utils.NewECSClient(kecs.Endpoint())
	})

	AfterEach(func() {
		kecs.Cleanup()
	})

	Context("when managing task definition revisions", func() {
		It("should maintain revision history correctly", func() {
			family := fmt.Sprintf("test-history-%d", time.Now().Unix())
			revisions := []string{}

			// Create 5 revisions
			for i := 1; i <= 5; i++ {
				taskDef := map[string]interface{}{
					"family": family,
					"containerDefinitions": []map[string]interface{}{
						{
							"name":      "app",
							"image":     fmt.Sprintf("myapp:v%d", i),
							"memory":    256,
							"essential": true,
							"environment": []map[string]interface{}{
								{
									"name":  "VERSION",
									"value": fmt.Sprintf("v%d", i),
								},
							},
						},
					},
					"requiresCompatibilities": []string{"EC2"},
					"networkMode":             "bridge",
				}

				result, err := client.RegisterTaskDefinition(taskDef)
				Expect(err).NotTo(HaveOccurred())

				td := result["taskDefinition"].(map[string]interface{})
				Expect(td["revision"]).To(Equal(float64(i)))
				revisions = append(revisions, td["taskDefinitionArn"].(string))
			}

			// List all revisions
			result, err := client.ListTaskDefinitionsWithOptions(map[string]interface{}{
				"familyPrefix": family,
				"sort":         "DESC",
			})
			Expect(err).NotTo(HaveOccurred())

			taskDefArns := result["taskDefinitionArns"].([]interface{})
			Expect(taskDefArns).To(HaveLen(5))

			// Verify latest revision is listed first (DESC sort)
			latestArn := taskDefArns[0].(string)
			Expect(latestArn).To(ContainSubstring(":5"))
		})

		It("should use latest ACTIVE revision when no revision specified", func() {
			family := fmt.Sprintf("test-latest-%d", time.Now().Unix())

			// Create multiple revisions
			for i := 1; i <= 3; i++ {
				taskDef := map[string]interface{}{
					"family": family,
					"containerDefinitions": []map[string]interface{}{
						{
							"name":      "app",
							"image":     fmt.Sprintf("myapp:v%d", i),
							"memory":    256,
							"essential": true,
						},
					},
					"requiresCompatibilities": []string{"EC2"},
					"networkMode":             "bridge",
				}
				_, err := client.RegisterTaskDefinition(taskDef)
				Expect(err).NotTo(HaveOccurred())
			}

			// Describe without revision (should get latest)
			result, err := client.DescribeTaskDefinition(family)
			Expect(err).NotTo(HaveOccurred())

			td := result["taskDefinition"].(map[string]interface{})
			Expect(td["revision"]).To(Equal(float64(3)))

			// Verify it's the latest version
			containerDefs := td["containerDefinitions"].([]interface{})
			container := containerDefs[0].(map[string]interface{})
			Expect(container["image"]).To(Equal("myapp:v3"))
		})
	})

	Context("when deregistering task definitions", func() {
		It("should successfully deregister a task definition", func() {
			family := fmt.Sprintf("test-deregister-%d", time.Now().Unix())

			// Register task definition
			taskDef := map[string]interface{}{
				"family": family,
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

			regResult, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			td := regResult["taskDefinition"].(map[string]interface{})
			taskDefArn := td["taskDefinitionArn"].(string)

			// Deregister the task definition
			deregResult, err := client.DeregisterTaskDefinition(taskDefArn)
			Expect(err).NotTo(HaveOccurred())

			// Verify response
			deregTd := deregResult["taskDefinition"].(map[string]interface{})
			Expect(deregTd["status"]).To(Equal("INACTIVE"))
			Expect(deregTd["taskDefinitionArn"]).To(Equal(taskDefArn))

			// Verify task definition is now INACTIVE
			descResult, err := client.DescribeTaskDefinition(taskDefArn)
			Expect(err).NotTo(HaveOccurred())

			descTd := descResult["taskDefinition"].(map[string]interface{})
			Expect(descTd["status"]).To(Equal("INACTIVE"))
		})

		It("should not affect other revisions when deregistering", func() {
			family := fmt.Sprintf("test-deregister-multi-%d", time.Now().Unix())

			// Register 3 revisions
			var revision2Arn string
			for i := 1; i <= 3; i++ {
				taskDef := map[string]interface{}{
					"family": family,
					"containerDefinitions": []map[string]interface{}{
						{
							"name":      "app",
							"image":     fmt.Sprintf("myapp:v%d", i),
							"memory":    256,
							"essential": true,
						},
					},
					"requiresCompatibilities": []string{"EC2"},
					"networkMode":             "bridge",
				}

				result, err := client.RegisterTaskDefinition(taskDef)
				Expect(err).NotTo(HaveOccurred())

				if i == 2 {
					td := result["taskDefinition"].(map[string]interface{})
					revision2Arn = td["taskDefinitionArn"].(string)
				}
			}

			// Deregister revision 2
			_, err := client.DeregisterTaskDefinition(revision2Arn)
			Expect(err).NotTo(HaveOccurred())

			// Verify revision 1 is still ACTIVE
			result1, err := client.DescribeTaskDefinition(fmt.Sprintf("%s:1", family))
			Expect(err).NotTo(HaveOccurred())
			td1 := result1["taskDefinition"].(map[string]interface{})
			Expect(td1["status"]).To(Equal("ACTIVE"))

			// Verify revision 2 is INACTIVE
			result2, err := client.DescribeTaskDefinition(revision2Arn)
			Expect(err).NotTo(HaveOccurred())
			td2 := result2["taskDefinition"].(map[string]interface{})
			Expect(td2["status"]).To(Equal("INACTIVE"))

			// Verify revision 3 is still ACTIVE
			result3, err := client.DescribeTaskDefinition(fmt.Sprintf("%s:3", family))
			Expect(err).NotTo(HaveOccurred())
			td3 := result3["taskDefinition"].(map[string]interface{})
			Expect(td3["status"]).To(Equal("ACTIVE"))

			// Verify latest active revision is now 3 (not 2)
			latestResult, err := client.DescribeTaskDefinition(family)
			Expect(err).NotTo(HaveOccurred())
			latestTd := latestResult["taskDefinition"].(map[string]interface{})
			Expect(latestTd["revision"]).To(Equal(float64(3)))
		})

		It("should handle deregistering already inactive task definition", func() {
			family := fmt.Sprintf("test-double-deregister-%d", time.Now().Unix())

			// Register and deregister
			taskDef := map[string]interface{}{
				"family": family,
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

			regResult, err := client.RegisterTaskDefinition(taskDef)
			Expect(err).NotTo(HaveOccurred())

			td := regResult["taskDefinition"].(map[string]interface{})
			taskDefArn := td["taskDefinitionArn"].(string)

			// First deregister
			_, err = client.DeregisterTaskDefinition(taskDefArn)
			Expect(err).NotTo(HaveOccurred())

			// Second deregister should still succeed (idempotent)
			deregResult, err := client.DeregisterTaskDefinition(taskDefArn)
			Expect(err).NotTo(HaveOccurred())

			deregTd := deregResult["taskDefinition"].(map[string]interface{})
			Expect(deregTd["status"]).To(Equal("INACTIVE"))
		})

		It("should error when deregistering non-existent task definition", func() {
			nonExistentArn := "arn:aws:ecs:us-east-1:123456789012:task-definition/non-existent:1"
			_, err := client.DeregisterTaskDefinition(nonExistentArn)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})
})