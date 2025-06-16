package converters_test

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/artifacts"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// Mock artifact manager for testing
type mockArtifactManager struct{}

func (m *mockArtifactManager) DownloadArtifact(ctx context.Context, artifact *types.Artifact) error {
	return nil
}

func (m *mockArtifactManager) CleanupArtifacts(artifacts []types.Artifact) {}

var _ = Describe("TaskConverter Artifact Support", func() {
	var (
		converter *converters.TaskConverter
		taskDef   *storage.TaskDefinition
		cluster   *storage.Cluster
	)

	BeforeEach(func() {
		converter = converters.NewTaskConverter("us-east-1", "123456789012")
		// Set mock artifact manager
		converter.SetArtifactManager(artifacts.NewManager(nil)) // In production, pass real S3 integration
		
		cluster = &storage.Cluster{
			Name:   "test-cluster",
			Region: "us-east-1",
		}
	})

	Describe("ConvertTaskToPod with Artifacts", func() {
		It("should create init containers for artifacts", func() {
			// Task definition with artifacts
			containerDefs := []types.ContainerDefinition{
				{
					Name:   ptr.To("app"),
					Image:  ptr.To("myapp:latest"),
					Memory: ptr.To(int(512)),
					Artifacts: []types.Artifact{
						{
							Name:        ptr.To("config"),
							ArtifactUrl: ptr.To("s3://my-bucket/config.json"),
							TargetPath:  ptr.To("/config/app.json"),
							Type:        ptr.To("s3"),
						},
						{
							Name:        ptr.To("static-files"),
							ArtifactUrl: ptr.To("https://example.com/static.tar.gz"),
							TargetPath:  ptr.To("/static/files.tar.gz"),
							Type:        ptr.To("https"),
						},
					},
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef = &storage.TaskDefinition{
				Family:               "test-task",
				Revision:             1,
				ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
				ContainerDefinitions: string(containerDefsJSON),
				Memory:               "1024",
				CPU:                  "512",
				NetworkMode:          "awsvpc",
				Status:               "ACTIVE",
			}

			runTaskJSON := []byte(`{}`)
			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, "task-123")

			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			// Check init containers
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			initContainer := pod.Spec.InitContainers[0]
			Expect(initContainer.Name).To(Equal("artifact-downloader-app"))
			Expect(initContainer.Image).To(Equal("busybox:latest"))

			// Check volumes
			Expect(pod.Spec.Volumes).To(HaveLen(1))
			Expect(pod.Spec.Volumes[0].Name).To(Equal("artifacts-app"))

			// Check main container has artifact volume mount
			Expect(pod.Spec.Containers[0].VolumeMounts).To(HaveLen(1))
			volumeMount := pod.Spec.Containers[0].VolumeMounts[0]
			Expect(volumeMount.Name).To(Equal("artifacts-app"))
			Expect(volumeMount.MountPath).To(Equal("/artifacts"))
			Expect(volumeMount.ReadOnly).To(BeTrue())

			// Check environment variables for LocalStack
			Expect(initContainer.Env).To(HaveLen(3))
			envMap := make(map[string]string)
			for _, env := range initContainer.Env {
				envMap[env.Name] = env.Value
			}
			Expect(envMap["AWS_ACCESS_KEY_ID"]).To(Equal("test"))
			Expect(envMap["AWS_SECRET_ACCESS_KEY"]).To(Equal("test"))
			Expect(envMap["AWS_DEFAULT_REGION"]).To(Equal("us-east-1"))
		})

		It("should handle containers without artifacts", func() {
			// Task definition without artifacts
			containerDefs := []types.ContainerDefinition{
				{
					Name:   ptr.To("app"),
					Image:  ptr.To("myapp:latest"),
					Memory: ptr.To(int(512)),
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef = &storage.TaskDefinition{
				Family:               "test-task",
				Revision:             1,
				ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
				ContainerDefinitions: string(containerDefsJSON),
				Memory:               "1024",
				CPU:                  "512",
				NetworkMode:          "awsvpc",
				Status:               "ACTIVE",
			}

			runTaskJSON := []byte(`{}`)
			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, "task-123")

			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			// No init containers should be created
			Expect(pod.Spec.InitContainers).To(BeEmpty())

			// No artifact volumes
			Expect(pod.Spec.Volumes).To(BeEmpty())

			// No artifact volume mounts
			Expect(pod.Spec.Containers[0].VolumeMounts).To(BeEmpty())
		})

		It("should handle mixed containers with and without artifacts", func() {
			// Task definition with mixed containers
			containerDefs := []types.ContainerDefinition{
				{
					Name:   ptr.To("app"),
					Image:  ptr.To("myapp:latest"),
					Memory: ptr.To(int(512)),
					Artifacts: []types.Artifact{
						{
							Name:        ptr.To("config"),
							ArtifactUrl: ptr.To("s3://my-bucket/config.json"),
							TargetPath:  ptr.To("/config/app.json"),
						},
					},
				},
				{
					Name:   ptr.To("sidecar"),
					Image:  ptr.To("sidecar:latest"),
					Memory: ptr.To(int(256)),
					// No artifacts
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef = &storage.TaskDefinition{
				Family:               "test-task",
				Revision:             1,
				ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
				ContainerDefinitions: string(containerDefsJSON),
				Memory:               "1024",
				CPU:                  "512",
				NetworkMode:          "awsvpc",
				Status:               "ACTIVE",
			}

			runTaskJSON := []byte(`{}`)
			pod, err := converter.ConvertTaskToPod(taskDef, runTaskJSON, cluster, "task-123")

			Expect(err).NotTo(HaveOccurred())
			Expect(pod).NotTo(BeNil())

			// Only one init container for the app container
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			Expect(pod.Spec.InitContainers[0].Name).To(Equal("artifact-downloader-app"))

			// Only one volume for artifacts
			Expect(pod.Spec.Volumes).To(HaveLen(1))

			// Check container volume mounts
			Expect(pod.Spec.Containers[0].VolumeMounts).To(HaveLen(1)) // app has artifact mount
			Expect(pod.Spec.Containers[1].VolumeMounts).To(BeEmpty())  // sidecar has no artifact mount
		})
	})
})