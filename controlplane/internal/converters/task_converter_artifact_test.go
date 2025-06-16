package converters

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertTaskToPod_WithArtifacts(t *testing.T) {
	// Create task converter with artifact manager
	converter := NewTaskConverter("us-east-1", "123456789012")
	
	// Mock artifact manager
	mockArtifactManager := &mockArtifactManager{}
	converter.SetArtifactManager(mockArtifactManager)

	// Create test task definition with artifacts
	containerDef := types.ContainerDefinition{
		Name:     stringPtr("webapp"),
		Image:    stringPtr("nginx:latest"),
		Cpu:      intPtr(512),
		Memory:   intPtr(1024),
		Essential: boolPtr(true),
		Artifacts: []types.Artifact{
			{
				Name:        stringPtr("app-config"),
				ArtifactUrl: stringPtr("s3://my-bucket/configs/app.conf"),
				Type:        stringPtr("s3"),
				TargetPath:  stringPtr("config/app.conf"),
				Permissions: stringPtr("0644"),
			},
			{
				Name:        stringPtr("static-assets"),
				ArtifactUrl: stringPtr("https://example.com/assets.tar.gz"),
				Type:        stringPtr("https"),
				TargetPath:  stringPtr("assets/assets.tar.gz"),
				Permissions: stringPtr("0755"),
			},
		},
	}

	containerDefsJSON, err := json.Marshal([]types.ContainerDefinition{containerDef})
	require.NoError(t, err)

	taskDef := &storage.TaskDefinition{
		ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/webapp-with-artifacts:1",
		Family:               "webapp-with-artifacts",
		Revision:             1,
		ContainerDefinitions: string(containerDefsJSON),
		CPU:                  "512",
		Memory:               "1024",
		NetworkMode:          "awsvpc",
		Status:               "ACTIVE",
	}

	cluster := &storage.Cluster{
		Name:   "test-cluster",
		Status: "ACTIVE",
	}

	runTaskReq := types.RunTaskRequest{
		Cluster:        stringPtr("test-cluster"),
		TaskDefinition: stringPtr("webapp-with-artifacts:1"),
		LaunchType:     stringPtr("FARGATE"),
	}
	runTaskReqJSON, err := json.Marshal(runTaskReq)
	require.NoError(t, err)

	// Convert task to pod
	pod, err := converter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "test-task-id")
	require.NoError(t, err)
	assert.NotNil(t, pod)

	// Verify init containers were created
	assert.Len(t, pod.Spec.InitContainers, 1)
	initContainer := pod.Spec.InitContainers[0]
	assert.Equal(t, "artifact-downloader-webapp", initContainer.Name)
	assert.Equal(t, "busybox:latest", initContainer.Image)
	assert.Equal(t, []string{"/bin/sh", "-c"}, initContainer.Command)

	// Verify artifact volume was created
	artifactVolumeFound := false
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == "artifacts-webapp" {
			artifactVolumeFound = true
			assert.NotNil(t, vol.VolumeSource.EmptyDir)
			break
		}
	}
	assert.True(t, artifactVolumeFound, "Artifact volume should be created")

	// Verify main container has artifact volume mount
	container := pod.Spec.Containers[0]
	artifactVolumeMountFound := false
	for _, mount := range container.VolumeMounts {
		if mount.Name == "artifacts-webapp" {
			artifactVolumeMountFound = true
			assert.Equal(t, "/artifacts", mount.MountPath)
			assert.True(t, mount.ReadOnly)
			break
		}
	}
	assert.True(t, artifactVolumeMountFound, "Container should have artifact volume mount")

	// Verify init container has correct environment variables
	assert.Len(t, initContainer.Env, 3)
	envMap := make(map[string]string)
	for _, env := range initContainer.Env {
		envMap[env.Name] = env.Value
	}
	assert.Equal(t, "test", envMap["AWS_ACCESS_KEY_ID"])
	assert.Equal(t, "test", envMap["AWS_SECRET_ACCESS_KEY"])
	assert.Equal(t, "us-east-1", envMap["AWS_DEFAULT_REGION"])
}

func TestGenerateArtifactDownloadScript(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	artifacts := []types.Artifact{
		{
			Name:        stringPtr("s3-artifact"),
			ArtifactUrl: stringPtr("s3://my-bucket/file.txt"),
			TargetPath:  stringPtr("data/file.txt"),
			Permissions: stringPtr("0644"),
		},
		{
			Name:        stringPtr("http-artifact"),
			ArtifactUrl: stringPtr("https://example.com/archive.tar.gz"),
			TargetPath:  stringPtr("downloads/archive.tar.gz"),
			Permissions: stringPtr("0755"),
		},
	}

	script := converter.generateArtifactDownloadScript(artifacts)
	assert.NotEmpty(t, script)

	// Verify script contains expected commands
	assert.Contains(t, script, "mkdir -p $(dirname /artifacts/data/file.txt)")
	assert.Contains(t, script, "mkdir -p $(dirname /artifacts/downloads/archive.tar.gz)")
	assert.Contains(t, script, "wget -q -O /artifacts/downloads/archive.tar.gz https://example.com/archive.tar.gz")
	assert.Contains(t, script, "chmod 0644 /artifacts/data/file.txt")
	assert.Contains(t, script, "chmod 0755 /artifacts/downloads/archive.tar.gz")
	assert.Contains(t, script, "S3 download would happen here") // Placeholder for S3
}

func TestConvertTaskToPod_NoArtifacts(t *testing.T) {
	// Create task converter without artifact manager
	converter := NewTaskConverter("us-east-1", "123456789012")

	// Create test task definition without artifacts
	containerDef := types.ContainerDefinition{
		Name:      stringPtr("webapp"),
		Image:     stringPtr("nginx:latest"),
		Cpu:       intPtr(512),
		Memory:    intPtr(1024),
		Essential: boolPtr(true),
	}

	containerDefsJSON, err := json.Marshal([]types.ContainerDefinition{containerDef})
	require.NoError(t, err)

	taskDef := &storage.TaskDefinition{
		ARN:                  "arn:aws:ecs:us-east-1:123456789012:task-definition/webapp:1",
		Family:               "webapp",
		Revision:             1,
		ContainerDefinitions: string(containerDefsJSON),
		CPU:                  "512",
		Memory:               "1024",
		NetworkMode:          "awsvpc",
		Status:               "ACTIVE",
	}

	cluster := &storage.Cluster{
		Name:   "test-cluster",
		Status: "ACTIVE",
	}

	runTaskReq := types.RunTaskRequest{
		Cluster:        stringPtr("test-cluster"),
		TaskDefinition: stringPtr("webapp:1"),
		LaunchType:     stringPtr("FARGATE"),
	}
	runTaskReqJSON, err := json.Marshal(runTaskReq)
	require.NoError(t, err)

	// Convert task to pod
	pod, err := converter.ConvertTaskToPod(taskDef, runTaskReqJSON, cluster, "test-task-id")
	require.NoError(t, err)
	assert.NotNil(t, pod)

	// Verify no init containers were created
	assert.Len(t, pod.Spec.InitContainers, 0)

	// Verify no artifact volumes were created
	for _, vol := range pod.Spec.Volumes {
		assert.NotContains(t, vol.Name, "artifacts-")
	}

	// Verify container has no artifact volume mounts
	container := pod.Spec.Containers[0]
	for _, mount := range container.VolumeMounts {
		assert.NotEqual(t, "/artifacts", mount.MountPath)
	}
}

// Mock artifact manager for testing
type mockArtifactManager struct{}

func (m *mockArtifactManager) DownloadArtifacts(ctx context.Context, artifacts []types.Artifact, workDir string) error {
	return nil
}

func (m *mockArtifactManager) CleanupArtifacts(workDir string) error {
	return nil
}

func (m *mockArtifactManager) SetS3Endpoint(endpoint string) {}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}