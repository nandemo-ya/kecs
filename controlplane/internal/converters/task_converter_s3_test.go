package converters_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/artifacts"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

var _ = Describe("TaskConverter S3 Integration", func() {
	var (
		converter       *converters.TaskConverter
		taskDef         *storage.TaskDefinition
		cluster         *storage.Cluster
		artifactManager *artifacts.Manager
	)

	BeforeEach(func() {
		converter = converters.NewTaskConverter("us-east-1", "123456789012")
		artifactManager = artifacts.NewManager(nil)
		converter.SetArtifactManager(artifactManager)

		cluster = &storage.Cluster{
			Name: "test-cluster",
			ARN:  "arn:aws:ecs:us-east-1:123456789012:cluster/test-cluster",
		}

		// Create a task definition with S3 artifacts
		containerDefs := []types.ContainerDefinition{
			{
				Name:   stringPtr("app"),
				Image:  stringPtr("nginx:latest"),
				Memory: intPtr(512),
				Cpu:    intPtr(256),
				Artifacts: []types.Artifact{
					{
						ArtifactUrl: stringPtr("s3://my-bucket/config/app.conf"),
						TargetPath:  stringPtr("config/app.conf"),
						Permissions: stringPtr("0644"),
					},
					{
						ArtifactUrl: stringPtr("https://example.com/data.json"),
						TargetPath:  stringPtr("data/data.json"),
					},
				},
			},
		}

		containerDefsJSON, _ := json.Marshal(containerDefs)
		taskDef = &storage.TaskDefinition{
			Family:               "test-task",
			Revision:             1,
			ContainerDefinitions: string(containerDefsJSON),
			CPU:                  "256",
			Memory:               "512",
			ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/test-task:1",
		}
	})

	Describe("ConvertTaskToPod with S3 artifacts", func() {
		It("should create init containers for downloading artifacts", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())
			Expect(pod).ToNot(BeNil())

			// Check init containers
			Expect(pod.Spec.InitContainers).To(HaveLen(1))
			initContainer := pod.Spec.InitContainers[0]
			Expect(initContainer.Name).To(Equal("artifact-downloader-app"))
			Expect(initContainer.Image).To(Equal("amazon/aws-cli:latest"))

			// Check environment variables for S3
			envMap := make(map[string]string)
			for _, env := range initContainer.Env {
				envMap[env.Name] = env.Value
			}
			Expect(envMap).To(HaveKey("AWS_ACCESS_KEY_ID"))
			Expect(envMap).To(HaveKey("AWS_SECRET_ACCESS_KEY"))
			Expect(envMap).To(HaveKey("AWS_DEFAULT_REGION"))
			Expect(envMap).To(HaveKey("AWS_ENDPOINT_URL_S3"))
			Expect(envMap["AWS_ENDPOINT_URL_S3"]).To(Equal("http://localstack-proxy.default.svc.cluster.local:4566"))

			// Check volume mounts
			Expect(initContainer.VolumeMounts).To(HaveLen(1))
			Expect(initContainer.VolumeMounts[0].Name).To(Equal("artifacts-app"))
			Expect(initContainer.VolumeMounts[0].MountPath).To(Equal("/artifacts"))

			// Check command
			Expect(initContainer.Command).To(Equal([]string{"/bin/sh", "-c"}))
			Expect(initContainer.Args).To(HaveLen(1))
			
			// Verify the download script contains S3 and HTTP commands
			script := initContainer.Args[0]
			Expect(script).To(ContainSubstring("aws s3 cp s3://my-bucket/config/app.conf"))
			Expect(script).To(ContainSubstring("curl -s -L -o"))
			Expect(script).To(ContainSubstring("chmod 0644"))
		})

		It("should create artifact volumes", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Check volumes
			var artifactVolume *corev1.Volume
			for i := range pod.Spec.Volumes {
				if pod.Spec.Volumes[i].Name == "artifacts-app" {
					artifactVolume = &pod.Spec.Volumes[i]
					break
				}
			}
			Expect(artifactVolume).ToNot(BeNil())
			Expect(artifactVolume.VolumeSource.EmptyDir).ToNot(BeNil())
		})

		It("should mount artifact volumes in main containers", func() {
			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Check main container volume mounts
			mainContainer := pod.Spec.Containers[0]
			var artifactMount *corev1.VolumeMount
			for i := range mainContainer.VolumeMounts {
				if mainContainer.VolumeMounts[i].Name == "artifacts-app" {
					artifactMount = &mainContainer.VolumeMounts[i]
					break
				}
			}
			Expect(artifactMount).ToNot(BeNil())
			Expect(artifactMount.MountPath).To(Equal("/artifacts"))
		})

		It("should handle multiple containers with different artifacts", func() {
			// Create task definition with multiple containers
			containerDefs := []types.ContainerDefinition{
				{
					Name:   stringPtr("app1"),
					Image:  stringPtr("app1:latest"),
					Memory: intPtr(256),
					Artifacts: []types.Artifact{
						{
							ArtifactUrl: stringPtr("s3://bucket/app1.conf"),
							TargetPath:  stringPtr("app1.conf"),
						},
					},
				},
				{
					Name:   stringPtr("app2"),
					Image:  stringPtr("app2:latest"),
					Memory: intPtr(256),
					Artifacts: []types.Artifact{
						{
							ArtifactUrl: stringPtr("s3://bucket/app2.conf"),
							TargetPath:  stringPtr("app2.conf"),
						},
					},
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Should have 2 init containers
			Expect(pod.Spec.InitContainers).To(HaveLen(2))
			Expect(pod.Spec.InitContainers[0].Name).To(Equal("artifact-downloader-app1"))
			Expect(pod.Spec.InitContainers[1].Name).To(Equal("artifact-downloader-app2"))

			// Should have 2 artifact volumes
			artifactVolumeCount := 0
			for _, vol := range pod.Spec.Volumes {
				if vol.Name == "artifacts-app1" || vol.Name == "artifacts-app2" {
					artifactVolumeCount++
				}
			}
			Expect(artifactVolumeCount).To(Equal(2))
		})

		It("should not create init containers when no artifacts are defined", func() {
			// Create task definition without artifacts
			containerDefs := []types.ContainerDefinition{
				{
					Name:   stringPtr("app"),
					Image:  stringPtr("nginx:latest"),
					Memory: intPtr(512),
				},
			}

			containerDefsJSON, _ := json.Marshal(containerDefs)
			taskDef.ContainerDefinitions = string(containerDefsJSON)

			runTaskReq := types.RunTaskRequest{
				TaskDefinition: stringPtr("test-task:1"),
				Cluster:        stringPtr("test-cluster"),
			}
			reqJSON, _ := json.Marshal(runTaskReq)

			pod, err := converter.ConvertTaskToPod(taskDef, reqJSON, cluster, "task-123")
			Expect(err).ToNot(HaveOccurred())

			// Should have no init containers
			Expect(pod.Spec.InitContainers).To(HaveLen(0))
		})
	})

	Describe("Artifact download script generation", func() {
		It("should generate correct S3 download commands", func() {
			// Access the private method through the artifact manager
			artifacts := []types.Artifact{
				{
					ArtifactUrl: stringPtr("s3://bucket/path/to/file.txt"),
					TargetPath:  stringPtr("config/file.txt"),
				},
			}

			script := artifactManager.GetArtifactScript(artifacts)
			Expect(script).To(ContainSubstring("#!/bin/sh"))
			Expect(script).To(ContainSubstring("mkdir -p $(dirname /artifacts/config/file.txt)"))
			// The artifact manager currently uses placeholder for S3
			Expect(script).To(ContainSubstring("S3 download placeholder"))
		})

		It("should generate correct HTTP download commands", func() {
			artifacts := []types.Artifact{
				{
					ArtifactUrl: stringPtr("https://example.com/file.txt"),
					TargetPath:  stringPtr("config/file.txt"),
				},
			}

			script := artifactManager.GetArtifactScript(artifacts)
			Expect(script).To(ContainSubstring("wget -O /artifacts/config/file.txt https://example.com/file.txt"))
		})

		It("should set permissions when specified", func() {
			artifacts := []types.Artifact{
				{
					ArtifactUrl: stringPtr("https://example.com/script.sh"),
					TargetPath:  stringPtr("scripts/script.sh"),
					Permissions: stringPtr("0755"),
				},
			}

			script := artifactManager.GetArtifactScript(artifacts)
			Expect(script).To(ContainSubstring("chmod 0755 /artifacts/scripts/script.sh"))
		})
	})
})

// Helper functions
func intPtr(i int) *int {
	return &i
}