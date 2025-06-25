package converters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

func TestConvertVolumes(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name            string
		volumes         []types.Volume
		expectedVolumes int
		validate        func(t *testing.T, volumes []corev1.Volume)
	}{
		{
			name: "host volume",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-host-volume"),
					Host: &types.HostVolumeProperties{
						SourcePath: ptr.To("/var/lib/data"),
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-host-volume", volumes[0].Name)
				require.NotNil(t, volumes[0].HostPath)
				assert.Equal(t, "/var/lib/data", volumes[0].HostPath.Path)
			},
		},
		{
			name: "EFS volume",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-efs"),
					EfsVolumeConfiguration: &types.EFSVolumeConfiguration{
						FileSystemId:  ptr.To("fs-12345678"),
						RootDirectory: ptr.To("/export"),
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-efs", volumes[0].Name)
				require.NotNil(t, volumes[0].NFS)
				assert.Equal(t, "fs-12345678.efs.us-east-1.amazonaws.com", volumes[0].NFS.Server)
				assert.Equal(t, "/export", volumes[0].NFS.Path)
			},
		},
		{
			name: "EFS volume with default root",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-efs-default"),
					EfsVolumeConfiguration: &types.EFSVolumeConfiguration{
						FileSystemId: ptr.To("fs-87654321"),
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-efs-default", volumes[0].Name)
				require.NotNil(t, volumes[0].NFS)
				assert.Equal(t, "fs-87654321.efs.us-east-1.amazonaws.com", volumes[0].NFS.Server)
				assert.Equal(t, "/", volumes[0].NFS.Path)
			},
		},
		{
			name: "Docker volume - local task scope",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-docker-local"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Scope:  ptr.To("task"),
						Driver: ptr.To("local"),
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-docker-local", volumes[0].Name)
				require.NotNil(t, volumes[0].EmptyDir)
			},
		},
		{
			name: "Docker volume - local shared scope",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-docker-shared"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Scope:  ptr.To("shared"),
						Driver: ptr.To("local"),
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-docker-shared", volumes[0].Name)
				require.NotNil(t, volumes[0].PersistentVolumeClaim)
				assert.Equal(t, "kecs-volume-my-docker-shared", volumes[0].PersistentVolumeClaim.ClaimName)
			},
		},
		{
			name: "Docker volume - EBS driver",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-ebs"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Driver: ptr.To("rexray/ebs"),
						DriverOpts: map[string]string{
							"volumeID": "vol-12345678",
						},
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-ebs", volumes[0].Name)
				require.NotNil(t, volumes[0].AWSElasticBlockStore)
				assert.Equal(t, "vol-12345678", volumes[0].AWSElasticBlockStore.VolumeID)
				assert.Equal(t, "ext4", volumes[0].AWSElasticBlockStore.FSType)
			},
		},
		{
			name: "Docker volume - NFS driver",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-nfs"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Driver: ptr.To("nfs"),
						DriverOpts: map[string]string{
							"server": "nfs.example.com",
							"path":   "/data/shared",
						},
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-nfs", volumes[0].Name)
				require.NotNil(t, volumes[0].NFS)
				assert.Equal(t, "nfs.example.com", volumes[0].NFS.Server)
				assert.Equal(t, "/data/shared", volumes[0].NFS.Path)
			},
		},
		{
			name: "FSx Windows volume",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-fsx"),
					FsxWindowsFileServerVolumeConfiguration: &types.FSxWindowsFileServerVolumeConfiguration{
						FileSystemId:  ptr.To("fs-windows123"),
						RootDirectory: ptr.To("\\share\\data"),
					},
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-fsx", volumes[0].Name)
				// FSx Windows is not directly supported, should fallback to emptyDir
				require.NotNil(t, volumes[0].EmptyDir)
			},
		},
		{
			name: "empty volume (no configuration)",
			volumes: []types.Volume{
				{
					Name: ptr.To("my-empty"),
				},
			},
			expectedVolumes: 1,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 1)
				assert.Equal(t, "my-empty", volumes[0].Name)
				require.NotNil(t, volumes[0].EmptyDir)
			},
		},
		{
			name: "multiple volumes",
			volumes: []types.Volume{
				{
					Name: ptr.To("host-vol"),
					Host: &types.HostVolumeProperties{
						SourcePath: ptr.To("/host/path"),
					},
				},
				{
					Name: ptr.To("efs-vol"),
					EfsVolumeConfiguration: &types.EFSVolumeConfiguration{
						FileSystemId: ptr.To("fs-multi123"),
					},
				},
				{
					Name: ptr.To("docker-vol"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Scope: ptr.To("task"),
					},
				},
			},
			expectedVolumes: 3,
			validate: func(t *testing.T, volumes []corev1.Volume) {
				require.Len(t, volumes, 3)

				// Find each volume by name
				var hostVol, efsVol, dockerVol *corev1.Volume
				for i := range volumes {
					switch volumes[i].Name {
					case "host-vol":
						hostVol = &volumes[i]
					case "efs-vol":
						efsVol = &volumes[i]
					case "docker-vol":
						dockerVol = &volumes[i]
					}
				}

				require.NotNil(t, hostVol)
				require.NotNil(t, hostVol.HostPath)
				assert.Equal(t, "/host/path", hostVol.HostPath.Path)

				require.NotNil(t, efsVol)
				require.NotNil(t, efsVol.NFS)
				assert.Equal(t, "fs-multi123.efs.us-east-1.amazonaws.com", efsVol.NFS.Server)

				require.NotNil(t, dockerVol)
				require.NotNil(t, dockerVol.EmptyDir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertVolumes(tt.volumes)
			assert.Len(t, result, tt.expectedVolumes)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestConvertEFSVolume(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name      string
		efsConfig *types.EFSVolumeConfiguration
		validate  func(t *testing.T, volumeSource corev1.VolumeSource)
	}{
		{
			name: "basic EFS configuration",
			efsConfig: &types.EFSVolumeConfiguration{
				FileSystemId: ptr.To("fs-12345678"),
			},
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.NFS)
				assert.Equal(t, "fs-12345678.efs.us-east-1.amazonaws.com", vs.NFS.Server)
				assert.Equal(t, "/", vs.NFS.Path)
			},
		},
		{
			name: "EFS with custom root directory",
			efsConfig: &types.EFSVolumeConfiguration{
				FileSystemId:  ptr.To("fs-87654321"),
				RootDirectory: ptr.To("/my/custom/path"),
			},
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.NFS)
				assert.Equal(t, "fs-87654321.efs.us-east-1.amazonaws.com", vs.NFS.Server)
				assert.Equal(t, "/my/custom/path", vs.NFS.Path)
			},
		},
		{
			name: "EFS with empty root directory",
			efsConfig: &types.EFSVolumeConfiguration{
				FileSystemId:  ptr.To("fs-empty"),
				RootDirectory: ptr.To(""),
			},
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.NFS)
				assert.Equal(t, "/", vs.NFS.Path)
			},
		},
		{
			name:      "nil filesystem ID",
			efsConfig: &types.EFSVolumeConfiguration{},
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.EmptyDir)
				assert.Nil(t, vs.NFS)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertEFSVolume(tt.efsConfig)
			tt.validate(t, result)
		})
	}
}

func TestConvertDockerVolume(t *testing.T) {
	converter := NewTaskConverter("us-west-2", "123456789012")

	tests := []struct {
		name         string
		dockerConfig *types.DockerVolumeConfiguration
		volumeName   string
		validate     func(t *testing.T, volumeSource corev1.VolumeSource)
	}{
		{
			name:         "default configuration",
			dockerConfig: &types.DockerVolumeConfiguration{},
			volumeName:   "test-vol",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.EmptyDir)
			},
		},
		{
			name: "local driver task scope",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("local"),
				Scope:  ptr.To("task"),
			},
			volumeName: "task-vol",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.EmptyDir)
			},
		},
		{
			name: "local driver shared scope",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("local"),
				Scope:  ptr.To("shared"),
			},
			volumeName: "shared-vol",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.PersistentVolumeClaim)
				assert.Equal(t, "kecs-volume-shared-vol", vs.PersistentVolumeClaim.ClaimName)
			},
		},
		{
			name: "EBS driver with volume ID",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("rexray/ebs"),
				DriverOpts: map[string]string{
					"volumeID": "vol-abcdef123",
				},
			},
			volumeName: "ebs-vol",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.AWSElasticBlockStore)
				assert.Equal(t, "vol-abcdef123", vs.AWSElasticBlockStore.VolumeID)
				assert.Equal(t, "ext4", vs.AWSElasticBlockStore.FSType)
			},
		},
		{
			name: "EBS driver without volume ID",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("rexray/ebs"),
			},
			volumeName: "ebs-vol-no-id",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.EmptyDir)
			},
		},
		{
			name: "NFS driver",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("nfs"),
				DriverOpts: map[string]string{
					"server": "192.168.1.100",
					"path":   "/exports/data",
				},
			},
			volumeName: "nfs-vol",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.NFS)
				assert.Equal(t, "192.168.1.100", vs.NFS.Server)
				assert.Equal(t, "/exports/data", vs.NFS.Path)
			},
		},
		{
			name: "NFS driver without server",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("nfs"),
			},
			volumeName: "nfs-vol-no-server",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.EmptyDir)
			},
		},
		{
			name: "unknown driver",
			dockerConfig: &types.DockerVolumeConfiguration{
				Driver: ptr.To("custom-driver"),
			},
			volumeName: "custom-vol",
			validate: func(t *testing.T, vs corev1.VolumeSource) {
				require.NotNil(t, vs.EmptyDir)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.convertDockerVolume(tt.dockerConfig, tt.volumeName)
			tt.validate(t, result)
		})
	}
}

func TestSanitizeVolumeName(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "my-volume",
			expected: "kecs-volume-my-volume",
		},
		{
			name:     "name with underscores",
			input:    "my_volume_name",
			expected: "kecs-volume-my-volume-name",
		},
		{
			name:     "name with special characters",
			input:    "my@volume#123",
			expected: "kecs-volume-my-volume-123",
		},
		{
			name:     "name with uppercase",
			input:    "MyVolume",
			expected: "kecs-volume-myvolume",
		},
		{
			name:     "name with dots",
			input:    "my.volume.name",
			expected: "kecs-volume-my-volume-name",
		},
		{
			name:     "name starting with hyphen",
			input:    "-volume-",
			expected: "kecs-volume-volume",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := converter.sanitizeVolumeName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAddVolumeAnnotations(t *testing.T) {
	converter := NewTaskConverter("us-east-1", "123456789012")

	tests := []struct {
		name        string
		volumes     []types.Volume
		validatePod func(t *testing.T, pod *corev1.Pod)
	}{
		{
			name: "EFS volume with all options",
			volumes: []types.Volume{
				{
					Name: ptr.To("efs-vol"),
					EfsVolumeConfiguration: &types.EFSVolumeConfiguration{
						FileSystemId:          ptr.To("fs-12345678"),
						TransitEncryption:     ptr.To("ENABLED"),
						TransitEncryptionPort: ptr.To(2049),
						AuthorizationConfig: &types.EFSAuthorizationConfig{
							AccessPointId: ptr.To("fsap-12345678"),
							Iam:           ptr.To("ENABLED"),
						},
					},
				},
			},
			validatePod: func(t *testing.T, pod *corev1.Pod) {
				assert.Equal(t, "fs-12345678", pod.Annotations["kecs.dev/volume-efs-vol-efs-filesystem-id"])
				assert.Equal(t, "ENABLED", pod.Annotations["kecs.dev/volume-efs-vol-efs-transit-encryption"])
				assert.Equal(t, "2049", pod.Annotations["kecs.dev/volume-efs-vol-efs-transit-encryption-port"])
				assert.Equal(t, "fsap-12345678", pod.Annotations["kecs.dev/volume-efs-vol-efs-access-point-id"])
				assert.Equal(t, "ENABLED", pod.Annotations["kecs.dev/volume-efs-vol-efs-iam"])
			},
		},
		{
			name: "Docker volume with driver options",
			volumes: []types.Volume{
				{
					Name: ptr.To("docker-vol"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Scope:         ptr.To("shared"),
						Driver:        ptr.To("rexray/ebs"),
						Autoprovision: ptr.To(true),
						DriverOpts: map[string]string{
							"volumeID": "vol-12345",
							"size":     "100",
						},
						Labels: map[string]string{
							"env":  "prod",
							"team": "platform",
						},
					},
				},
			},
			validatePod: func(t *testing.T, pod *corev1.Pod) {
				assert.Equal(t, "shared", pod.Annotations["kecs.dev/volume-docker-vol-docker-scope"])
				assert.Equal(t, "rexray/ebs", pod.Annotations["kecs.dev/volume-docker-vol-docker-driver"])
				assert.Equal(t, "true", pod.Annotations["kecs.dev/volume-docker-vol-docker-autoprovision"])

				// Check JSON encoded driver opts
				driverOpts := pod.Annotations["kecs.dev/volume-docker-vol-docker-driver-opts"]
				assert.Contains(t, driverOpts, `"volumeID":"vol-12345"`)
				assert.Contains(t, driverOpts, `"size":"100"`)

				// Check JSON encoded labels
				labels := pod.Annotations["kecs.dev/volume-docker-vol-docker-labels"]
				assert.Contains(t, labels, `"env":"prod"`)
				assert.Contains(t, labels, `"team":"platform"`)
			},
		},
		{
			name: "FSx Windows volume",
			volumes: []types.Volume{
				{
					Name: ptr.To("fsx-vol"),
					FsxWindowsFileServerVolumeConfiguration: &types.FSxWindowsFileServerVolumeConfiguration{
						FileSystemId:  ptr.To("fs-windows123"),
						RootDirectory: ptr.To("\\share\\data"),
						AuthorizationConfig: &types.FSxWindowsFileServerAuthorizationConfig{
							CredentialsParameter: ptr.To("arn:aws:ssm:region:account:parameter/fsx/creds"),
							Domain:               ptr.To("corp.example.com"),
						},
					},
				},
			},
			validatePod: func(t *testing.T, pod *corev1.Pod) {
				assert.Equal(t, "fs-windows123", pod.Annotations["kecs.dev/volume-fsx-vol-fsx-filesystem-id"])
				assert.Equal(t, "\\share\\data", pod.Annotations["kecs.dev/volume-fsx-vol-fsx-root-directory"])
				assert.Equal(t, "arn:aws:ssm:region:account:parameter/fsx/creds", pod.Annotations["kecs.dev/volume-fsx-vol-fsx-credentials-parameter"])
				assert.Equal(t, "corp.example.com", pod.Annotations["kecs.dev/volume-fsx-vol-fsx-domain"])
			},
		},
		{
			name: "multiple volumes",
			volumes: []types.Volume{
				{
					Name: ptr.To("vol1"),
					EfsVolumeConfiguration: &types.EFSVolumeConfiguration{
						FileSystemId: ptr.To("fs-111"),
					},
				},
				{
					Name: ptr.To("vol2"),
					DockerVolumeConfiguration: &types.DockerVolumeConfiguration{
						Driver: ptr.To("local"),
					},
				},
			},
			validatePod: func(t *testing.T, pod *corev1.Pod) {
				assert.Equal(t, "fs-111", pod.Annotations["kecs.dev/volume-vol1-efs-filesystem-id"])
				assert.Equal(t, "local", pod.Annotations["kecs.dev/volume-vol2-docker-driver"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: make(map[string]string),
				},
			}

			converter.addVolumeAnnotations(pod, tt.volumes)

			if tt.validatePod != nil {
				tt.validatePod(t, pod)
			}
		})
	}
}
