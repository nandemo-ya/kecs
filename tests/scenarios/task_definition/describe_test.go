package task_definition_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/tests/scenarios/utils"
)

var _ = Describe("Task Definition Describe Operations", func() {
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

	Context("when describing task definitions", func() {
		It("should describe a registered task definition", func() {
			// Register a task definition first
			family := fmt.Sprintf("test-describe-%d", time.Now().Unix())
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

			// Describe the task definition
			descResult, err := client.DescribeTaskDefinition(family)
			Expect(err).NotTo(HaveOccurred())
			Expect(descResult).NotTo(BeNil())

			// Verify the described task definition matches registered one
			describedTd := descResult["taskDefinition"].(map[string]interface{})
			registeredTd := regResult["taskDefinition"].(map[string]interface{})

			Expect(describedTd["family"]).To(Equal(registeredTd["family"]))
			Expect(describedTd["revision"]).To(Equal(registeredTd["revision"]))
			Expect(describedTd["status"]).To(Equal("ACTIVE"))
			Expect(describedTd["taskDefinitionArn"]).To(Equal(registeredTd["taskDefinitionArn"]))
		})

		It("should describe a specific revision of task definition", func() {
			family := fmt.Sprintf("test-revision-describe-%d", time.Now().Unix())

			// Register multiple revisions
			for i := 1; i <= 3; i++ {
				taskDef := map[string]interface{}{
					"family": family,
					"containerDefinitions": []map[string]interface{}{
						{
							"name":      "nginx",
							"image":     fmt.Sprintf("nginx:1.%d", 20+i),
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

			// Describe specific revision
			taskDefId := fmt.Sprintf("%s:2", family)
			result, err := client.DescribeTaskDefinition(taskDefId)
			Expect(err).NotTo(HaveOccurred())

			td := result["taskDefinition"].(map[string]interface{})
			Expect(td["family"]).To(Equal(family))
			Expect(td["revision"]).To(Equal(float64(2)))

			// Verify correct image version
			containerDefs := td["containerDefinitions"].([]interface{})
			container := containerDefs[0].(map[string]interface{})
			Expect(container["image"]).To(Equal("nginx:1.22"))
		})

		It("should return error for non-existent task definition", func() {
			_, err := client.DescribeTaskDefinition("non-existent-family")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})

	Context("when listing task definitions", func() {
		It("should list all registered task definition families", func() {
			// Register multiple task definitions
			families := []string{}
			for i := 0; i < 3; i++ {
				family := fmt.Sprintf("test-list-%d-%d", time.Now().Unix(), i)
				families = append(families, family)

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
				_, err := client.RegisterTaskDefinition(taskDef)
				Expect(err).NotTo(HaveOccurred())
			}

			// List task definition families
			result, err := client.ListTaskDefinitionFamilies()
			Expect(err).NotTo(HaveOccurred())

			listedFamilies, ok := result["families"].([]interface{})
			Expect(ok).To(BeTrue())

			// Verify all registered families are listed
			listedFamilyNames := make([]string, 0)
			for _, f := range listedFamilies {
				listedFamilyNames = append(listedFamilyNames, f.(string))
			}

			for _, family := range families {
				Expect(listedFamilyNames).To(ContainElement(family))
			}
		})

		It("should list task definitions with pagination", func() {
			// Register multiple task definitions
			baseTime := time.Now().Unix()
			for i := 0; i < 5; i++ {
				family := fmt.Sprintf("test-paginate-%d-%d", baseTime, i)
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
				_, err := client.RegisterTaskDefinition(taskDef)
				Expect(err).NotTo(HaveOccurred())
			}

			// List with max results
			result, err := client.ListTaskDefinitionsWithOptions(map[string]interface{}{
				"maxResults": 2,
			})
			Expect(err).NotTo(HaveOccurred())

			taskDefArns, ok := result["taskDefinitionArns"].([]interface{})
			Expect(ok).To(BeTrue())
			Expect(len(taskDefArns)).To(BeNumerically("<=", 2))

			// Check if nextToken is provided when there are more results
			if len(taskDefArns) == 2 {
				_, hasNextToken := result["nextToken"]
				Expect(hasNextToken).To(BeTrue())
			}
		})

		It("should list task definitions filtered by family", func() {
			// Register task definitions in different families
			family1 := fmt.Sprintf("test-filter-app-%d", time.Now().Unix())
			family2 := fmt.Sprintf("test-filter-web-%d", time.Now().Unix())

			for _, family := range []string{family1, family2} {
				taskDef := map[string]interface{}{
					"family": family,
					"containerDefinitions": []map[string]interface{}{
						{
							"name":      "container",
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
			}

			// List filtered by family prefix
			result, err := client.ListTaskDefinitionsWithOptions(map[string]interface{}{
				"familyPrefix": family1,
			})
			Expect(err).NotTo(HaveOccurred())

			taskDefArns, ok := result["taskDefinitionArns"].([]interface{})
			Expect(ok).To(BeTrue())

			// Verify only matching family is returned
			for _, arnInterface := range taskDefArns {
				arn := arnInterface.(string)
				Expect(arn).To(ContainSubstring(family1))
				Expect(arn).NotTo(ContainSubstring(family2))
			}
		})

		It("should list task definitions filtered by status", func() {
			family := fmt.Sprintf("test-status-%d", time.Now().Unix())

			// Register and then deregister a task definition
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
			_, err = client.DeregisterTaskDefinition(taskDefArn)
			Expect(err).NotTo(HaveOccurred())

			// List only INACTIVE task definitions
			result, err := client.ListTaskDefinitionsWithOptions(map[string]interface{}{
				"status": "INACTIVE",
			})
			Expect(err).NotTo(HaveOccurred())

			taskDefArns, ok := result["taskDefinitionArns"].([]interface{})
			Expect(ok).To(BeTrue())

			// Verify the deregistered task definition is listed
			arnStrings := make([]string, len(taskDefArns))
			for i, arn := range taskDefArns {
				arnStrings[i] = arn.(string)
			}
			Expect(arnStrings).To(ContainElement(taskDefArn))
		})
	})
})