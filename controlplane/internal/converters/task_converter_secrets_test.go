package converters_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("TaskConverter Secrets", func() {
	var (
		converter *converters.TaskConverter
		cluster   *storage.Cluster
		taskDef   *storage.TaskDefinition
	)

	BeforeEach(func() {
		converter = converters.NewTaskConverter("us-east-1", "123456789012")
		cluster = &storage.Cluster{
			Name:   "test-cluster",
			ARN:    "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster",
			Region: "us-east-1",
		}
	})

	Describe("ConvertTaskToPod with secrets", func() {
		Context("with Secrets Manager secrets", func() {
			It("should create environment variables referencing Kubernetes secrets", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name:  stringPtr("test-container"),
						Image: stringPtr("nginx:latest"),
						Secrets: []types.Secret{
							{
								Name:      stringPtr("DB_PASSWORD"),
								ValueFrom: stringPtr("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-password-AbCdEf"),
							},
							{
								Name:      stringPtr("API_KEY"),
								ValueFrom: stringPtr("arn:aws:secretsmanager:us-east-1:123456789012:secret:api-keys-XyZ123:api_key::"),
							},
						},
					},
				}

				containerDefsJSON, err := json.Marshal(containerDefs)
				Expect(err).NotTo(HaveOccurred())

				taskDef = &storage.TaskDefinition{
					Family:               "test-task",
					Revision:             1,
					ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
					ContainerDefinitions: string(containerDefsJSON),
					CPU:                  "256",
					Memory:               "512",
				}

				runTaskReq := &types.RunTaskRequest{
					TaskDefinition: stringPtr("test-task:1"),
				}
				runTaskReqJSON, err := json.Marshal(runTaskReq)
				Expect(err).NotTo(HaveOccurred())

				pod, err := converter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "task-123")
				Expect(err).NotTo(HaveOccurred())
				Expect(pod).NotTo(BeNil())

				// Check container environment variables
				Expect(pod.Spec.Containers).To(HaveLen(1))
				container := pod.Spec.Containers[0]

				// Find the secret environment variables
				var dbPasswordEnv, apiKeyEnv *corev1.EnvVar
				for i := range container.Env {
					if container.Env[i].Name == "DB_PASSWORD" {
						dbPasswordEnv = &container.Env[i]
					} else if container.Env[i].Name == "API_KEY" {
						apiKeyEnv = &container.Env[i]
					}
				}

				// Verify DB_PASSWORD references Kubernetes secret (Phase 2 implementation)
				Expect(dbPasswordEnv).NotTo(BeNil())
				Expect(dbPasswordEnv.Value).To(BeEmpty())
				Expect(dbPasswordEnv.ValueFrom).NotTo(BeNil())
				Expect(dbPasswordEnv.ValueFrom.SecretKeyRef).NotTo(BeNil())
				Expect(dbPasswordEnv.ValueFrom.SecretKeyRef.Name).To(Equal("sm-db-password"))
				Expect(dbPasswordEnv.ValueFrom.SecretKeyRef.Key).To(Equal("value"))

				// Verify API_KEY references Kubernetes secret (Phase 2 implementation)
				Expect(apiKeyEnv).NotTo(BeNil())
				Expect(apiKeyEnv.Value).To(BeEmpty())
				Expect(apiKeyEnv.ValueFrom).NotTo(BeNil())
				Expect(apiKeyEnv.ValueFrom.SecretKeyRef).NotTo(BeNil())
				Expect(apiKeyEnv.ValueFrom.SecretKeyRef.Name).To(Equal("sm-api-keys"))
				Expect(apiKeyEnv.ValueFrom.SecretKeyRef.Key).To(Equal("api_key"))

				// Check pod annotations for secret tracking
				Expect(pod.Annotations).To(HaveKeyWithValue("kecs.dev/secret-count", "2"))
				Expect(pod.Annotations).To(HaveKey("kecs.dev/secret-0-arn"))
				Expect(pod.Annotations).To(HaveKey("kecs.dev/secret-1-arn"))
			})
		})

		Context("with SSM Parameter Store secrets", func() {
			It("should create environment variables referencing Kubernetes secrets", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name:  stringPtr("test-container"),
						Image: stringPtr("nginx:latest"),
						Secrets: []types.Secret{
							{
								Name:      stringPtr("CONFIG_VALUE"),
								ValueFrom: stringPtr("arn:aws:ssm:us-east-1:123456789012:parameter/app/config/value"),
							},
						},
					},
				}

				containerDefsJSON, err := json.Marshal(containerDefs)
				Expect(err).NotTo(HaveOccurred())

				taskDef = &storage.TaskDefinition{
					Family:               "test-task",
					Revision:             1,
					ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
					ContainerDefinitions: string(containerDefsJSON),
					CPU:                  "256",
					Memory:               "512",
				}

				runTaskReq := &types.RunTaskRequest{
					TaskDefinition: stringPtr("test-task:1"),
				}
				runTaskReqJSON, err := json.Marshal(runTaskReq)
				Expect(err).NotTo(HaveOccurred())

				pod, err := converter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "task-124")
				Expect(err).NotTo(HaveOccurred())
				Expect(pod).NotTo(BeNil())

				// Check container environment variables
				container := pod.Spec.Containers[0]

				// Find the secret environment variable
				var configValueEnv *corev1.EnvVar
				for i := range container.Env {
					if container.Env[i].Name == "CONFIG_VALUE" {
						configValueEnv = &container.Env[i]
					}
				}

				// Verify CONFIG_VALUE references Kubernetes ConfigMap (Phase 2 implementation)
				// Since "config/value" doesn't contain sensitive keywords, it should use ConfigMap
				Expect(configValueEnv).NotTo(BeNil())
				Expect(configValueEnv.Value).To(BeEmpty())
				Expect(configValueEnv.ValueFrom).NotTo(BeNil())
				Expect(configValueEnv.ValueFrom.ConfigMapKeyRef).NotTo(BeNil())
				Expect(configValueEnv.ValueFrom.ConfigMapKeyRef.Name).To(Equal("ssm-cm-app-config-value"))
				Expect(configValueEnv.ValueFrom.ConfigMapKeyRef.Key).To(Equal("value"))
			})
		})

		Context("with mixed secrets", func() {
			It("should handle both Secrets Manager and SSM secrets", func() {
				containerDefs := []types.ContainerDefinition{
					{
						Name:  stringPtr("test-container"),
						Image: stringPtr("nginx:latest"),
						Secrets: []types.Secret{
							{
								Name:      stringPtr("DB_PASSWORD"),
								ValueFrom: stringPtr("arn:aws:secretsmanager:us-east-1:123456789012:secret:db-password-AbCdEf"),
							},
							{
								Name:      stringPtr("APP_CONFIG"),
								ValueFrom: stringPtr("arn:aws:ssm:us-east-1:123456789012:parameter/app/config"),
							},
						},
					},
				}

				containerDefsJSON, err := json.Marshal(containerDefs)
				Expect(err).NotTo(HaveOccurred())

				taskDef = &storage.TaskDefinition{
					Family:               "test-task",
					Revision:             1,
					ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
					ContainerDefinitions: string(containerDefsJSON),
					CPU:                  "256",
					Memory:               "512",
				}

				runTaskReq := &types.RunTaskRequest{
					TaskDefinition: stringPtr("test-task:1"),
				}
				runTaskReqJSON, err := json.Marshal(runTaskReq)
				Expect(err).NotTo(HaveOccurred())

				pod, err := converter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "task-125")
				Expect(err).NotTo(HaveOccurred())
				Expect(pod).NotTo(BeNil())

				// Check that both types of secrets are properly referenced
				container := pod.Spec.Containers[0]

				var dbPasswordEnv, appConfigEnv *corev1.EnvVar
				for i := range container.Env {
					if container.Env[i].Name == "DB_PASSWORD" {
						dbPasswordEnv = &container.Env[i]
					} else if container.Env[i].Name == "APP_CONFIG" {
						appConfigEnv = &container.Env[i]
					}
				}

				// Verify Secrets Manager secret references Kubernetes secret
				Expect(dbPasswordEnv).NotTo(BeNil())
				Expect(dbPasswordEnv.Value).To(BeEmpty())
				Expect(dbPasswordEnv.ValueFrom).NotTo(BeNil())
				Expect(dbPasswordEnv.ValueFrom.SecretKeyRef).NotTo(BeNil())
				Expect(dbPasswordEnv.ValueFrom.SecretKeyRef.Name).To(Equal("sm-db-password"))

				// Verify SSM parameter references Kubernetes ConfigMap
				// "app/config" doesn't contain sensitive keywords
				Expect(appConfigEnv).NotTo(BeNil())
				Expect(appConfigEnv.Value).To(BeEmpty())
				Expect(appConfigEnv.ValueFrom).NotTo(BeNil())
				Expect(appConfigEnv.ValueFrom.ConfigMapKeyRef).NotTo(BeNil())
				Expect(appConfigEnv.ValueFrom.ConfigMapKeyRef.Name).To(Equal("ssm-cm-app-config"))
				Expect(appConfigEnv.ValueFrom.ConfigMapKeyRef.Key).To(Equal("value"))
			})
		})
	})
})
