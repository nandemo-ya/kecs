package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// TaskConverter converts ECS task definitions to Kubernetes resources
type TaskConverter struct {
	region    string
	accountID string
}

// NewTaskConverter creates a new task converter
func NewTaskConverter(region, accountID string) *TaskConverter {
	return &TaskConverter{
		region:    region,
		accountID: accountID,
	}
}

// ConvertTaskToPod converts an ECS task definition and RunTask request to a Kubernetes Pod
func (c *TaskConverter) ConvertTaskToPod(
	taskDef *storage.TaskDefinition,
	runTaskReqJSON []byte, // Accept JSON bytes to avoid circular import
	cluster *storage.Cluster,
	taskID string,
) (*corev1.Pod, error) {
	// Parse the request JSON
	var runTaskReq types.RunTaskRequest
	if err := json.Unmarshal(runTaskReqJSON, &runTaskReq); err != nil {
		return nil, fmt.Errorf("failed to parse RunTask request: %w", err)
	}
	// Parse container definitions
	var containerDefs []types.ContainerDefinition
	if err := json.Unmarshal([]byte(taskDef.ContainerDefinitions), &containerDefs); err != nil {
		return nil, fmt.Errorf("failed to parse container definitions: %w", err)
	}

	// Parse volumes if any
	var volumes []types.Volume
	if taskDef.Volumes != "" {
		if err := json.Unmarshal([]byte(taskDef.Volumes), &volumes); err != nil {
			return nil, fmt.Errorf("failed to parse volumes: %w", err)
		}
	}

	// Create pod spec
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("ecs-task-%s", taskID),
			Namespace: c.getNamespace(cluster),
			Labels: map[string]string{
				"kecs.dev/cluster":         cluster.Name,
				"kecs.dev/task-id":         taskID,
				"kecs.dev/task-family":     taskDef.Family,
				"kecs.dev/task-revision":   fmt.Sprintf("%d", taskDef.Revision),
				"kecs.dev/launch-type":     c.getLaunchType(&runTaskReq),
				"kecs.dev/managed-by":      "kecs",
			},
			Annotations: map[string]string{
				"kecs.dev/task-arn":            c.generateTaskARN(cluster.Name, taskID),
				"kecs.dev/task-definition-arn": taskDef.ARN,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever, // ECS tasks don't restart by default
			Containers:    c.convertContainers(containerDefs, taskDef, &runTaskReq),
			Volumes:       c.convertVolumes(volumes),
		},
	}

	// Apply network mode
	if taskDef.NetworkMode == "host" {
		pod.Spec.HostNetwork = true
	}

	// Apply PID mode
	if taskDef.PidMode == "host" {
		pod.Spec.HostPID = true
	}

	// Apply IPC mode
	if taskDef.IpcMode == "host" {
		pod.Spec.HostIPC = true
	}

	// Apply task-level resource constraints
	if taskDef.CPU != "" || taskDef.Memory != "" {
		c.applyResourceConstraints(pod, taskDef.CPU, taskDef.Memory)
	}

	// Apply overrides if any
	if runTaskReq.Overrides != nil {
		c.applyOverrides(pod, runTaskReq.Overrides)
	}

	// Add placement constraints as node selectors/affinity
	if runTaskReq.PlacementConstraints != nil {
		c.applyPlacementConstraints(pod, runTaskReq.PlacementConstraints)
	}

	// Add tags as labels
	if runTaskReq.Tags != nil {
		c.applyTags(pod, runTaskReq.Tags)
	}

	return pod, nil
}

// convertContainers converts ECS container definitions to Kubernetes containers
func (c *TaskConverter) convertContainers(
	containerDefs []types.ContainerDefinition,
	taskDef *storage.TaskDefinition,
	runTaskReq *types.RunTaskRequest,
) []corev1.Container {
	containers := make([]corev1.Container, 0, len(containerDefs))

	for _, def := range containerDefs {
		container := corev1.Container{
			Name:            *def.Name,
			Image:           *def.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
		}

		// Command and arguments
		if def.Command != nil && len(def.Command) > 0 {
			container.Command = def.Command
		}
		if def.EntryPoint != nil && len(def.EntryPoint) > 0 {
			container.Command = def.EntryPoint
			if def.Command != nil {
				container.Args = def.Command
			}
		}

		// Environment variables
		if def.Environment != nil {
			container.Env = c.convertEnvironment(def.Environment)
		}

		// Secrets from environment
		if def.Secrets != nil {
			container.Env = append(container.Env, c.convertSecrets(def.Secrets)...)
		}

		// Port mappings
		if def.PortMappings != nil {
			container.Ports = c.convertPortMappings(def.PortMappings)
		}

		// Resources (CPU and Memory)
		resources := corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}

		if def.Cpu != nil {
			// ECS CPU units: 1024 = 1 vCPU
			cpuMillis := *def.Cpu * 1000 / 1024
			resources.Requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(cpuMillis), resource.DecimalSI)
			resources.Limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(int64(cpuMillis), resource.DecimalSI)
		}

		if def.Memory != nil {
			// ECS memory is in MiB
			memoryMi := resource.MustParse(fmt.Sprintf("%dMi", *def.Memory))
			resources.Requests[corev1.ResourceMemory] = memoryMi
			resources.Limits[corev1.ResourceMemory] = memoryMi
		} else if def.MemoryReservation != nil {
			// Use memory reservation as request if memory limit not set
			memoryMi := resource.MustParse(fmt.Sprintf("%dMi", *def.MemoryReservation))
			resources.Requests[corev1.ResourceMemory] = memoryMi
		}

		if len(resources.Requests) > 0 || len(resources.Limits) > 0 {
			container.Resources = resources
		}

		// Volume mounts
		if def.MountPoints != nil {
			container.VolumeMounts = c.convertMountPoints(def.MountPoints)
		}

		// Working directory
		if def.WorkingDirectory != nil {
			container.WorkingDir = *def.WorkingDirectory
		}

		// Essential containers (all containers are essential by default in K8s)
		// If Essential is false, we might want to handle this differently
		if def.Essential != nil && !*def.Essential {
			// Add annotation to track non-essential containers
			container.Name = container.Name + "-nonessential"
		}

		// Health check
		if def.HealthCheck != nil {
			container.LivenessProbe = c.convertHealthCheck(def.HealthCheck)
		}

		// User
		if def.User != nil {
			container.SecurityContext = c.parseUser(*def.User)
		}

		// Privileged
		if def.Privileged != nil && *def.Privileged {
			if container.SecurityContext == nil {
				container.SecurityContext = &corev1.SecurityContext{}
			}
			container.SecurityContext.Privileged = ptr.To(true)
		}

		// ReadonlyRootFilesystem
		if def.ReadonlyRootFilesystem != nil && *def.ReadonlyRootFilesystem {
			if container.SecurityContext == nil {
				container.SecurityContext = &corev1.SecurityContext{}
			}
			container.SecurityContext.ReadOnlyRootFilesystem = ptr.To(true)
		}

		containers = append(containers, container)
	}

	return containers
}

// convertEnvironment converts ECS environment variables to Kubernetes
func (c *TaskConverter) convertEnvironment(env []types.KeyValuePair) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0, len(env))
	for _, e := range env {
		if e.Name != nil && e.Value != nil {
			envVars = append(envVars, corev1.EnvVar{
				Name:  *e.Name,
				Value: *e.Value,
			})
		}
	}
	return envVars
}

// convertSecrets converts ECS secrets to Kubernetes environment variables
func (c *TaskConverter) convertSecrets(secrets []types.Secret) []corev1.EnvVar {
	// TODO: Implement proper secret handling with Kubernetes secrets
	// For now, we'll skip secrets as they need proper integration
	return []corev1.EnvVar{}
}

// convertPortMappings converts ECS port mappings to Kubernetes
func (c *TaskConverter) convertPortMappings(mappings []types.PortMapping) []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, 0, len(mappings))
	for _, mapping := range mappings {
		port := corev1.ContainerPort{}
		
		if mapping.ContainerPort != nil {
			port.ContainerPort = int32(*mapping.ContainerPort)
		}
		
		if mapping.Protocol != nil {
			switch strings.ToLower(*mapping.Protocol) {
			case "tcp":
				port.Protocol = corev1.ProtocolTCP
			case "udp":
				port.Protocol = corev1.ProtocolUDP
			default:
				port.Protocol = corev1.ProtocolTCP
			}
		} else {
			port.Protocol = corev1.ProtocolTCP
		}
		
		if mapping.Name != nil {
			port.Name = *mapping.Name
		}
		
		// Note: HostPort is not set by default in Kubernetes
		// It would require special handling for host network mode
		
		ports = append(ports, port)
	}
	return ports
}

// convertVolumes converts ECS volumes to Kubernetes volumes
func (c *TaskConverter) convertVolumes(volumes []types.Volume) []corev1.Volume {
	k8sVolumes := make([]corev1.Volume, 0, len(volumes))
	
	for _, vol := range volumes {
		if vol.Name == nil {
			continue
		}
		
		k8sVol := corev1.Volume{
			Name: *vol.Name,
		}
		
		// Handle host volumes
		if vol.Host != nil && vol.Host.SourcePath != nil {
			k8sVol.VolumeSource = corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: *vol.Host.SourcePath,
				},
			}
		} else {
			// Default to emptyDir for volumes without host path
			k8sVol.VolumeSource = corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}
		}
		
		// TODO: Handle other volume types like EFS, Docker volumes, etc.
		
		k8sVolumes = append(k8sVolumes, k8sVol)
	}
	
	return k8sVolumes
}

// convertMountPoints converts ECS mount points to Kubernetes volume mounts
func (c *TaskConverter) convertMountPoints(mountPoints []types.MountPoint) []corev1.VolumeMount {
	mounts := make([]corev1.VolumeMount, 0, len(mountPoints))
	
	for _, mp := range mountPoints {
		if mp.SourceVolume == nil || mp.ContainerPath == nil {
			continue
		}
		
		mount := corev1.VolumeMount{
			Name:      *mp.SourceVolume,
			MountPath: *mp.ContainerPath,
		}
		
		if mp.ReadOnly != nil {
			mount.ReadOnly = *mp.ReadOnly
		}
		
		mounts = append(mounts, mount)
	}
	
	return mounts
}

// convertHealthCheck converts ECS health check to Kubernetes liveness probe
func (c *TaskConverter) convertHealthCheck(hc *types.HealthCheck) *corev1.Probe {
	if hc.Command == nil || len(hc.Command) == 0 {
		return nil
	}
	
	probe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: hc.Command,
			},
		},
	}
	
	if hc.Interval != nil {
		probe.PeriodSeconds = int32(*hc.Interval)
	}
	
	if hc.Timeout != nil {
		probe.TimeoutSeconds = int32(*hc.Timeout)
	}
	
	if hc.Retries != nil {
		probe.FailureThreshold = int32(*hc.Retries)
	}
	
	if hc.StartPeriod != nil {
		probe.InitialDelaySeconds = int32(*hc.StartPeriod)
	}
	
	return probe
}

// parseUser parses ECS user string to SecurityContext
func (c *TaskConverter) parseUser(user string) *corev1.SecurityContext {
	// ECS user format: "uid:gid" or just "uid"
	parts := strings.Split(user, ":")
	
	sc := &corev1.SecurityContext{}
	
	// Try to parse as integer
	if uid, err := parseInt64(parts[0]); err == nil {
		sc.RunAsUser = ptr.To(uid)
	}
	
	if len(parts) > 1 {
		if gid, err := parseInt64(parts[1]); err == nil {
			sc.RunAsGroup = ptr.To(gid)
		}
	}
	
	return sc
}

// Helper functions

func (c *TaskConverter) getNamespace(cluster *storage.Cluster) string {
	// Format: <cluster-name>-<region>
	return fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
}

func (c *TaskConverter) getLaunchType(req *types.RunTaskRequest) string {
	if req.LaunchType != nil {
		return *req.LaunchType
	}
	return "FARGATE" // Default
}

func (c *TaskConverter) generateTaskARN(clusterName, taskID string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s",
		c.region, c.accountID, clusterName, taskID)
}

func (c *TaskConverter) applyResourceConstraints(pod *corev1.Pod, cpu, memory string) {
	// Apply to all containers proportionally if task-level constraints are set
	// This is a simplified implementation
	// TODO: Implement proper resource distribution logic
}

func (c *TaskConverter) applyOverrides(pod *corev1.Pod, overrides *types.TaskOverride) {
	// TODO: Implement override logic
	// This would modify container commands, environment variables, etc.
}

func (c *TaskConverter) applyPlacementConstraints(pod *corev1.Pod, constraints []types.PlacementConstraint) {
	// TODO: Convert ECS placement constraints to Kubernetes node selectors/affinity
}

func (c *TaskConverter) applyTags(pod *corev1.Pod, tags []types.Tag) {
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			// Convert tags to labels, ensuring they're valid K8s label format
			key := strings.ReplaceAll(*tag.Key, "/", "-")
			key = strings.ReplaceAll(key, ".", "-")
			pod.Labels[fmt.Sprintf("kecs.dev/tag-%s", key)] = *tag.Value
		}
	}
}

func parseInt64(s string) (int64, error) {
	var i int64
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}