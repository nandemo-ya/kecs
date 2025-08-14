package converters

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("TaskConverter", func() {
	var converter *TaskConverter

	BeforeEach(func() {
		converter = NewTaskConverter("us-east-1", "123456789012")
	})

	Describe("parseSecretARN", func() {
		It("should parse Secrets Manager ARN with JSON key", func() {
			arn := "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf:username::"
			result, err := converter.parseSecretArn(arn)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.SecretName).To(Equal("my-secret-AbCdEf"))
			Expect(result.Key).To(Equal("username"))
			Expect(result.Source).To(Equal("secretsmanager"))
		})

		It("should parse Secrets Manager ARN without JSON key", func() {
			arn := "arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf"
			result, err := converter.parseSecretArn(arn)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.SecretName).To(Equal("my-secret-AbCdEf"))
			Expect(result.Key).To(Equal("value"))
			Expect(result.Source).To(Equal("secretsmanager"))
		})

		It("should parse SSM Parameter Store ARN", func() {
			arn := "arn:aws:ssm:us-east-1:123456789012:parameter/app/database/password"
			result, err := converter.parseSecretArn(arn)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.SecretName).To(Equal("app/database/password"))
			Expect(result.Key).To(Equal("value"))
			Expect(result.Source).To(Equal("ssm"))
		})

		It("should return error for invalid ARN", func() {
			result, err := converter.parseSecretArn("invalid-arn")
			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Describe("ConvertTaskToPod", func() {
		var (
			taskDef     *storage.TaskDefinition
			runTaskJSON []byte
			cluster     *storage.Cluster
			taskID      string
		)

		BeforeEach(func() {
			// Create minimal task definition for testing
			containerDefs := []types.ContainerDefinition{
				{
					Name:   ptr.To("nginx"),
					Image:  ptr.To("nginx:latest"),
					Memory: ptr.To(int(512)),
					PortMappings: []types.PortMapping{
						{
							ContainerPort: ptr.To(int(80)),
							Protocol:      ptr.To("tcp"),
						},
					},
					Essential: ptr.To(true),
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)

			taskDef = &storage.TaskDefinition{
				Family:               "test-task",
				Revision:             1,
				ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
				ContainerDefinitions: string(containerDefsJSON),
				TaskRoleARN:          "",
				Memory:               "1024",
				CPU:                  "512",
				NetworkMode:          "awsvpc",
				Status:               "ACTIVE",
			}

			runTaskJSON = []byte(`{}`)
			cluster = &storage.Cluster{
				Name:   "test-cluster",
				Region: "us-east-1",
			}
			taskID = "test-task-id"
		})

		It("should convert simple task definition to Kubernetes pod", func() {
			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, taskID)

			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())
			Expect(pod.Name).To(Equal("test-task-id"))
			Expect(pod.Namespace).To(Equal("test-cluster-us-east-1"))
			Expect(pod.Labels).To(HaveKeyWithValue("kecs.dev/task-family", "test-task"))
			Expect(pod.Labels).To(HaveKeyWithValue("kecs.dev/task-revision", "1"))

			// Check container
			Expect(pod.Spec.Containers).To(HaveLen(1))
			container := pod.Spec.Containers[0]
			Expect(container.Name).To(Equal("nginx"))
			Expect(container.Image).To(Equal("nginx:latest"))
			Expect(container.Ports).To(HaveLen(1))
			Expect(container.Ports[0].ContainerPort).To(Equal(int32(80)))
		})

		It("should handle task with environment variables", func() {
			containerDefs := []types.ContainerDefinition{
				{
					Name:  ptr.To("app"),
					Image: ptr.To("myapp:latest"),
					Environment: []types.KeyValuePair{
						{
							Name:  ptr.To("NODE_ENV"),
							Value: ptr.To("production"),
						},
						{
							Name:  ptr.To("PORT"),
							Value: ptr.To("3000"),
						},
					},
					Essential: ptr.To(true),
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, taskID)

			Expect(err).NotTo(HaveOccurred())
			container := pod.Spec.Containers[0]
			Expect(container.Env).To(HaveLen(2))
			Expect(container.Env[0].Name).To(Equal("NODE_ENV"))
			Expect(container.Env[0].Value).To(Equal("production"))
			Expect(container.Env[1].Name).To(Equal("PORT"))
			Expect(container.Env[1].Value).To(Equal("3000"))
		})

		It("should handle task with secrets", func() {
			containerDefs := []types.ContainerDefinition{
				{
					Name:  ptr.To("app"),
					Image: ptr.To("myapp:latest"),
					Secrets: []types.Secret{
						{
							Name:      ptr.To("DB_PASSWORD"),
							ValueFrom: ptr.To("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-pass-XyZ123"),
						},
					},
					Essential: ptr.To(true),
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, taskID)

			Expect(err).NotTo(HaveOccurred())
			container := pod.Spec.Containers[0]
			Expect(container.Env).To(HaveLen(1))
			Expect(container.Env[0].Name).To(Equal("DB_PASSWORD"))
			// Phase 2 implementation uses Kubernetes secret references
			Expect(container.Env[0].Value).To(BeEmpty())
			Expect(container.Env[0].ValueFrom).NotTo(BeNil())
			Expect(container.Env[0].ValueFrom.SecretKeyRef).NotTo(BeNil())
			Expect(container.Env[0].ValueFrom.SecretKeyRef.Name).To(Equal("sm-db-pass"))
			Expect(container.Env[0].ValueFrom.SecretKeyRef.Key).To(Equal("value"))
		})

		It("should handle overrides from RunTask request", func() {
			runTaskJSON = []byte(`{
				"overrides": {
					"containerOverrides": [{
						"name": "nginx",
						"environment": [
							{"name": "OVERRIDE_ENV", "value": "override_value"}
						],
						"command": ["sh", "-c", "echo hello"]
					}]
				}
			}`)

			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, taskID)

			Expect(err).NotTo(HaveOccurred())
			container := pod.Spec.Containers[0]
			Expect(container.Env).To(HaveLen(1))
			Expect(container.Env[0].Name).To(Equal("OVERRIDE_ENV"))
			Expect(container.Command).To(Equal([]string{"sh", "-c", "echo hello"}))
		})

		It("should handle task with multiple containers", func() {
			containerDefs := []types.ContainerDefinition{
				{
					Name:      ptr.To("nginx"),
					Image:     ptr.To("nginx:latest"),
					Essential: ptr.To(true),
				},
				{
					Name:      ptr.To("sidecar"),
					Image:     ptr.To("busybox:latest"),
					Essential: ptr.To(false),
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, taskID)

			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Spec.Containers).To(HaveLen(2))
			Expect(pod.Spec.Containers[0].Name).To(Equal("nginx"))
			Expect(pod.Spec.Containers[1].Name).To(Equal("sidecar-nonessential"))
		})
	})
})
