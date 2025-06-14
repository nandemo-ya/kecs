package converters

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
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

// SecretInfo holds parsed information from a secret ARN
type SecretInfo struct {
	SecretName string
	Key        string
	Source     string // "secretsmanager" or "ssm"
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
				"kecs.dev/cluster":       cluster.Name,
				"kecs.dev/task-id":       taskID,
				"kecs.dev/task-family":   taskDef.Family,
				"kecs.dev/task-revision": fmt.Sprintf("%d", taskDef.Revision),
				"kecs.dev/launch-type":   c.getLaunchType(&runTaskReq),
				"kecs.dev/managed-by":    "kecs",
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

	// Add volume configuration annotations
	c.addVolumeAnnotations(pod, volumes)

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
	envVars := make([]corev1.EnvVar, 0, len(secrets))

	for _, secret := range secrets {
		if secret.Name == nil || secret.ValueFrom == nil {
			continue
		}

		// Parse the valueFrom ARN to extract secret information
		// Format: arn:aws:secretsmanager:region:account:secret:name-xxxx:key::
		// or arn:aws:ssm:region:account:parameter/path
		secretInfo := c.parseSecretARN(*secret.ValueFrom)

		if secretInfo != nil {
			envVar := corev1.EnvVar{
				Name: *secret.Name,
			}

			// Reference the Kubernetes secret
			envVar.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretInfo.SecretName,
					},
					Key: secretInfo.Key,
				},
			}

			envVars = append(envVars, envVar)
		}
	}

	return envVars
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

		// Handle different volume types
		switch {
		case vol.Host != nil && vol.Host.SourcePath != nil:
			// Host volume
			k8sVol.VolumeSource = corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: *vol.Host.SourcePath,
				},
			}

		case vol.EfsVolumeConfiguration != nil:
			// EFS volume - convert to NFS in Kubernetes
			k8sVol.VolumeSource = c.convertEFSVolume(vol.EfsVolumeConfiguration)

		case vol.DockerVolumeConfiguration != nil:
			// Docker volume - convert based on driver
			k8sVol.VolumeSource = c.convertDockerVolume(vol.DockerVolumeConfiguration, *vol.Name)

		case vol.FsxWindowsFileServerVolumeConfiguration != nil:
			// FSx Windows File Server - convert to CIFS/SMB
			k8sVol.VolumeSource = c.convertFSxWindowsVolume(vol.FsxWindowsFileServerVolumeConfiguration)

		default:
			// Default to emptyDir for volumes without specific configuration
			k8sVol.VolumeSource = corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}
		}

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
	// Apply task-level resource constraints to containers
	if cpu == "" && memory == "" {
		return
	}

	// Parse task-level resources
	var taskCPUMillis int64
	var taskMemoryMi int64

	if cpu != "" {
		// ECS CPU units: 1024 = 1 vCPU
		cpuUnits, err := strconv.ParseInt(cpu, 10, 64)
		if err == nil {
			taskCPUMillis = cpuUnits * 1000 / 1024
		}
	}

	if memory != "" {
		// ECS memory is in MiB
		memMiB, err := strconv.ParseInt(memory, 10, 64)
		if err == nil {
			taskMemoryMi = memMiB
		}
	}

	// Calculate total requested resources by containers
	var totalRequestedCPU int64
	var totalRequestedMemory int64

	for _, container := range pod.Spec.Containers {
		if container.Resources.Requests != nil {
			if cpuReq, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				totalRequestedCPU += cpuReq.MilliValue()
			}
			if memReq, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				// Convert to MiB for calculation
				totalRequestedMemory += memReq.Value() / (1024 * 1024)
			}
		}
	}

	// If no containers have requested resources, distribute evenly
	if totalRequestedCPU == 0 && totalRequestedMemory == 0 {
		c.distributeResourcesEvenly(pod, taskCPUMillis, taskMemoryMi)
		return
	}

	// Apply proportional distribution based on requested resources
	c.distributeResourcesProportionally(pod, taskCPUMillis, taskMemoryMi, totalRequestedCPU, totalRequestedMemory)
}

func (c *TaskConverter) applyOverrides(pod *corev1.Pod, overrides *types.TaskOverride) {
	if overrides == nil {
		return
	}

	// Apply task-level resource overrides
	if overrides.Cpu != nil || overrides.Memory != nil {
		cpu := ""
		memory := ""
		if overrides.Cpu != nil {
			cpu = *overrides.Cpu
		}
		if overrides.Memory != nil {
			memory = *overrides.Memory
		}
		c.applyResourceConstraints(pod, cpu, memory)
	}

	// Apply container-specific overrides
	if overrides.ContainerOverrides != nil {
		for _, containerOverride := range overrides.ContainerOverrides {
			if containerOverride.Name == nil {
				continue
			}

			// Find the container by name
			for i := range pod.Spec.Containers {
				if pod.Spec.Containers[i].Name == *containerOverride.Name {
					c.applyContainerOverride(&pod.Spec.Containers[i], &containerOverride)
					break
				}
			}
		}
	}

	// Apply task role ARN as annotation
	if overrides.TaskRoleArn != nil {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations["kecs.dev/task-role-arn"] = *overrides.TaskRoleArn
	}

	// Apply execution role ARN as annotation
	if overrides.ExecutionRoleArn != nil {
		if pod.Annotations == nil {
			pod.Annotations = make(map[string]string)
		}
		pod.Annotations["kecs.dev/execution-role-arn"] = *overrides.ExecutionRoleArn
	}
}

// applyContainerOverride applies overrides to a specific container
func (c *TaskConverter) applyContainerOverride(container *corev1.Container, override *types.ContainerOverride) {
	// Override command
	if override.Command != nil && len(override.Command) > 0 {
		container.Command = override.Command
	}

	// Override or add environment variables
	if override.Environment != nil {
		envMap := make(map[string]string)

		// First, collect existing environment variables
		for _, env := range container.Env {
			envMap[env.Name] = env.Value
		}

		// Then, apply overrides
		for _, envVar := range override.Environment {
			if envVar.Name != nil && envVar.Value != nil {
				envMap[*envVar.Name] = *envVar.Value
			}
		}

		// Rebuild the environment variables list
		container.Env = make([]corev1.EnvVar, 0, len(envMap))
		for name, value := range envMap {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  name,
				Value: value,
			})
		}
	}

	// Override CPU and memory
	if override.Cpu != nil || override.Memory != nil || override.MemoryReservation != nil {
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = corev1.ResourceList{}
		}

		// Override CPU
		if override.Cpu != nil {
			cpuMillis := *override.Cpu * 1000 / 1024
			cpuQuantity := resource.NewMilliQuantity(int64(cpuMillis), resource.DecimalSI)
			container.Resources.Requests[corev1.ResourceCPU] = *cpuQuantity
			container.Resources.Limits[corev1.ResourceCPU] = *cpuQuantity
		}

		// Override Memory
		if override.Memory != nil {
			memQuantity := resource.MustParse(fmt.Sprintf("%dMi", *override.Memory))
			container.Resources.Requests[corev1.ResourceMemory] = memQuantity
			container.Resources.Limits[corev1.ResourceMemory] = memQuantity
		} else if override.MemoryReservation != nil {
			// Use memory reservation as request if memory limit not set
			memQuantity := resource.MustParse(fmt.Sprintf("%dMi", *override.MemoryReservation))
			container.Resources.Requests[corev1.ResourceMemory] = memQuantity
		}
	}
}

func (c *TaskConverter) applyPlacementConstraints(pod *corev1.Pod, constraints []types.PlacementConstraint) {
	if len(constraints) == 0 {
		return
	}

	// Initialize node selector if not present
	if pod.Spec.NodeSelector == nil {
		pod.Spec.NodeSelector = make(map[string]string)
	}

	// Initialize affinity if needed
	var nodeAffinity *corev1.NodeAffinity
	var podAffinity *corev1.PodAffinity
	var podAntiAffinity *corev1.PodAntiAffinity

	if pod.Spec.Affinity != nil {
		nodeAffinity = pod.Spec.Affinity.NodeAffinity
		podAffinity = pod.Spec.Affinity.PodAffinity
		podAntiAffinity = pod.Spec.Affinity.PodAntiAffinity
	}

	for _, constraint := range constraints {
		if constraint.Type == nil {
			continue
		}

		switch *constraint.Type {
		case "memberOf":
			// Handle memberOf constraints
			c.applyMemberOfConstraint(pod, constraint, &nodeAffinity)

		case "distinctInstance":
			// Ensure tasks run on different instances
			c.applyDistinctInstanceConstraint(pod, &podAntiAffinity)

		default:
			// Unknown constraint type, add as annotation
			if pod.Annotations == nil {
				pod.Annotations = make(map[string]string)
			}
			pod.Annotations[fmt.Sprintf("kecs.dev/constraint-%s", *constraint.Type)] =
				fmt.Sprintf("%v", constraint.Expression)
		}
	}

	// Apply affinity if any rules were added
	if nodeAffinity != nil || podAffinity != nil || podAntiAffinity != nil {
		pod.Spec.Affinity = &corev1.Affinity{
			NodeAffinity:    nodeAffinity,
			PodAffinity:     podAffinity,
			PodAntiAffinity: podAntiAffinity,
		}
	}
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

// parseSecretARN parses an AWS secret ARN and returns secret information
func (c *TaskConverter) parseSecretARN(arn string) *SecretInfo {
	// Pattern for Secrets Manager ARN: arn:aws:secretsmanager:region:account:secret:name-xxxx:jsonkey::
	secretsManagerPattern := regexp.MustCompile(`^arn:aws:secretsmanager:[^:]+:[^:]+:secret:([^:]+)(?::([^:]+))?`)

	// Pattern for SSM Parameter Store ARN: arn:aws:ssm:region:account:parameter/path/to/param
	ssmPattern := regexp.MustCompile(`^arn:aws:ssm:[^:]+:[^:]+:parameter/(.+)$`)

	// Check Secrets Manager pattern
	if matches := secretsManagerPattern.FindStringSubmatch(arn); matches != nil {
		info := &SecretInfo{
			Source: "secretsmanager",
		}

		// Extract secret name (remove the random suffix like -AbCdEf)
		secretFullName := matches[1]
		parts := strings.Split(secretFullName, "-")
		if len(parts) > 1 && len(parts[len(parts)-1]) == 6 {
			// Remove the random suffix
			info.SecretName = strings.Join(parts[:len(parts)-1], "-")
		} else {
			info.SecretName = secretFullName
		}

		// JSON key is optional
		if len(matches) > 2 && matches[2] != "" {
			info.Key = matches[2]
		} else {
			info.Key = "value" // Default key
		}

		// Convert to Kubernetes-compatible secret name
		info.SecretName = c.sanitizeSecretName(info.SecretName)

		return info
	}

	// Check SSM pattern
	if matches := ssmPattern.FindStringSubmatch(arn); matches != nil {
		paramPath := matches[1]

		info := &SecretInfo{
			Source: "ssm",
			Key:    "value", // Default key for SSM parameters
		}

		// Convert path to secret name (replace / with -)
		info.SecretName = c.sanitizeSecretName(strings.ReplaceAll(paramPath, "/", "-"))

		return info
	}

	return nil
}

// sanitizeSecretName converts a name to be Kubernetes-compatible
func (c *TaskConverter) sanitizeSecretName(name string) string {
	// Kubernetes names must be lowercase alphanumeric or '-'
	// and must start and end with alphanumeric
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	// Prefix with kecs to avoid conflicts
	return fmt.Sprintf("kecs-secret-%s", name)
}

// CollectSecrets collects all secrets used in a task definition
func (c *TaskConverter) CollectSecrets(containerDefs []types.ContainerDefinition) map[string]*SecretInfo {
	secrets := make(map[string]*SecretInfo)

	for _, def := range containerDefs {
		if def.Secrets != nil {
			for _, secret := range def.Secrets {
				if secret.ValueFrom != nil {
					if info := c.parseSecretARN(*secret.ValueFrom); info != nil {
						// Use the ARN as key to avoid duplicates
						secrets[*secret.ValueFrom] = info
					}
				}
			}
		}
	}

	return secrets
}

// distributeResourcesEvenly distributes task-level resources evenly among containers
func (c *TaskConverter) distributeResourcesEvenly(pod *corev1.Pod, taskCPUMillis, taskMemoryMi int64) {
	if len(pod.Spec.Containers) == 0 {
		return
	}

	// Divide resources evenly
	cpuPerContainer := taskCPUMillis / int64(len(pod.Spec.Containers))
	memoryPerContainer := taskMemoryMi / int64(len(pod.Spec.Containers))

	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Resources.Requests == nil {
			pod.Spec.Containers[i].Resources.Requests = corev1.ResourceList{}
		}
		if pod.Spec.Containers[i].Resources.Limits == nil {
			pod.Spec.Containers[i].Resources.Limits = corev1.ResourceList{}
		}

		// Set CPU
		if cpuPerContainer > 0 {
			cpuQuantity := resource.NewMilliQuantity(cpuPerContainer, resource.DecimalSI)
			pod.Spec.Containers[i].Resources.Requests[corev1.ResourceCPU] = *cpuQuantity
			pod.Spec.Containers[i].Resources.Limits[corev1.ResourceCPU] = *cpuQuantity
		}

		// Set Memory
		if memoryPerContainer > 0 {
			memQuantity := resource.MustParse(fmt.Sprintf("%dMi", memoryPerContainer))
			pod.Spec.Containers[i].Resources.Requests[corev1.ResourceMemory] = memQuantity
			pod.Spec.Containers[i].Resources.Limits[corev1.ResourceMemory] = memQuantity
		}
	}
}

// distributeResourcesProportionally distributes task-level resources proportionally based on container requests
func (c *TaskConverter) distributeResourcesProportionally(pod *corev1.Pod, taskCPUMillis, taskMemoryMi, totalRequestedCPU, totalRequestedMemory int64) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		// Get current container requests
		var containerCPU int64
		var containerMemory int64

		if container.Resources.Requests != nil {
			if cpuReq, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				containerCPU = cpuReq.MilliValue()
			}
			if memReq, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				containerMemory = memReq.Value() / (1024 * 1024) // Convert to MiB
			}
		}

		// Calculate proportional resources
		if taskCPUMillis > 0 && totalRequestedCPU > 0 {
			proportionalCPU := (containerCPU * taskCPUMillis) / totalRequestedCPU
			if proportionalCPU > 0 {
				cpuQuantity := resource.NewMilliQuantity(proportionalCPU, resource.DecimalSI)
				if container.Resources.Requests == nil {
					container.Resources.Requests = corev1.ResourceList{}
				}
				if container.Resources.Limits == nil {
					container.Resources.Limits = corev1.ResourceList{}
				}
				container.Resources.Requests[corev1.ResourceCPU] = *cpuQuantity
				container.Resources.Limits[corev1.ResourceCPU] = *cpuQuantity
			}
		}

		if taskMemoryMi > 0 && totalRequestedMemory > 0 {
			proportionalMemory := (containerMemory * taskMemoryMi) / totalRequestedMemory
			if proportionalMemory > 0 {
				memQuantity := resource.MustParse(fmt.Sprintf("%dMi", proportionalMemory))
				if container.Resources.Requests == nil {
					container.Resources.Requests = corev1.ResourceList{}
				}
				if container.Resources.Limits == nil {
					container.Resources.Limits = corev1.ResourceList{}
				}
				container.Resources.Requests[corev1.ResourceMemory] = memQuantity
				container.Resources.Limits[corev1.ResourceMemory] = memQuantity
			}
		}
	}
}

// applyMemberOfConstraint applies ECS memberOf placement constraints
func (c *TaskConverter) applyMemberOfConstraint(pod *corev1.Pod, constraint types.PlacementConstraint, nodeAffinity **corev1.NodeAffinity) {
	if constraint.Expression == nil {
		return
	}

	expr := *constraint.Expression

	// Parse common ECS memberOf expressions
	// Examples:
	// - "attribute:ecs.instance-type =~ t2.*"
	// - "attribute:ecs.availability-zone in [us-east-1a, us-east-1b]"
	// - "attribute:ecs.instance-type == t2.micro"

	// Simple parsing for common patterns
	if strings.HasPrefix(expr, "attribute:") {
		// Extract attribute name and condition
		parts := strings.SplitN(expr[10:], " ", 2) // Skip "attribute:"
		if len(parts) != 2 {
			return
		}

		attribute := parts[0]
		condition := parts[1]

		// Convert ECS attributes to Kubernetes labels
		k8sLabel := c.convertECSAttributeToK8sLabel(attribute)

		// Parse the condition
		if strings.HasPrefix(condition, "=~ ") {
			// Regex match - use node affinity with In operator
			pattern := strings.TrimSpace(condition[3:])
			c.addNodeAffinityRule(nodeAffinity, k8sLabel, pattern, "regex")
		} else if strings.HasPrefix(condition, "== ") {
			// Exact match - use node selector
			value := strings.TrimSpace(condition[3:])
			pod.Spec.NodeSelector[k8sLabel] = value
		} else if strings.HasPrefix(condition, "in ") {
			// In list - use node affinity
			valueList := strings.TrimSpace(condition[3:])
			c.addNodeAffinityRule(nodeAffinity, k8sLabel, valueList, "in")
		}
	}
}

// applyDistinctInstanceConstraint ensures tasks run on different instances
func (c *TaskConverter) applyDistinctInstanceConstraint(pod *corev1.Pod, podAntiAffinity **corev1.PodAntiAffinity) {
	if *podAntiAffinity == nil {
		*podAntiAffinity = &corev1.PodAntiAffinity{}
	}

	// Add anti-affinity rule to avoid scheduling on the same node
	antiAffinityTerm := corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"kecs.dev/task-family": pod.Labels["kecs.dev/task-family"],
			},
		},
		TopologyKey: "kubernetes.io/hostname", // Ensure different nodes
	}

	// Use preferred anti-affinity to allow scheduling if needed
	(*podAntiAffinity).PreferredDuringSchedulingIgnoredDuringExecution = append(
		(*podAntiAffinity).PreferredDuringSchedulingIgnoredDuringExecution,
		corev1.WeightedPodAffinityTerm{
			Weight:          100,
			PodAffinityTerm: antiAffinityTerm,
		},
	)
}

// convertECSAttributeToK8sLabel converts ECS attribute names to Kubernetes label format
func (c *TaskConverter) convertECSAttributeToK8sLabel(attribute string) string {
	// Common ECS attributes mapping
	attributeMap := map[string]string{
		"ecs.instance-type":     "node.kubernetes.io/instance-type",
		"ecs.availability-zone": "topology.kubernetes.io/zone",
		"ecs.ami-id":            "kecs.dev/ami-id",
		"ecs.instance-id":       "kecs.dev/instance-id",
		"ecs.os-type":           "kubernetes.io/os",
		"ecs.cpu-architecture":  "kubernetes.io/arch",
	}

	if k8sLabel, ok := attributeMap[attribute]; ok {
		return k8sLabel
	}

	// Default: convert to kecs.dev namespace
	sanitized := strings.ReplaceAll(attribute, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	return fmt.Sprintf("kecs.dev/%s", sanitized)
}

// addNodeAffinityRule adds a node affinity rule
func (c *TaskConverter) addNodeAffinityRule(nodeAffinity **corev1.NodeAffinity, key, value, matchType string) {
	if *nodeAffinity == nil {
		*nodeAffinity = &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{},
			},
		}
	}

	var requirement corev1.NodeSelectorRequirement
	requirement.Key = key

	switch matchType {
	case "regex":
		// For regex patterns, we'll use In operator with expanded values
		// This is a simplified approach - in production, you might want to
		// use a webhook or operator to handle regex matching
		requirement.Operator = corev1.NodeSelectorOpIn
		requirement.Values = c.expandRegexPattern(value)

	case "in":
		// Parse list like "[value1, value2, value3]"
		requirement.Operator = corev1.NodeSelectorOpIn
		requirement.Values = c.parseValueList(value)

	default:
		requirement.Operator = corev1.NodeSelectorOpIn
		requirement.Values = []string{value}
	}

	// Add to existing terms or create new one
	if len((*nodeAffinity).RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 0 {
		(*nodeAffinity).RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions =
			append((*nodeAffinity).RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions, requirement)
	} else {
		(*nodeAffinity).RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
			(*nodeAffinity).RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
			corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{requirement},
			},
		)
	}
}

// expandRegexPattern expands common regex patterns to explicit values
func (c *TaskConverter) expandRegexPattern(pattern string) []string {
	// Handle common patterns
	// This is a simplified implementation - in production, you'd want
	// more sophisticated regex handling

	if strings.HasPrefix(pattern, "t2.") {
		return []string{"t2.micro", "t2.small", "t2.medium", "t2.large", "t2.xlarge", "t2.2xlarge"}
	}

	if strings.HasPrefix(pattern, "m5.") {
		return []string{"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge", "m5.8xlarge", "m5.12xlarge", "m5.16xlarge", "m5.24xlarge"}
	}

	// Default: return as-is
	return []string{pattern}
}

// parseValueList parses a list string like "[value1, value2]" into a slice
func (c *TaskConverter) parseValueList(valueList string) []string {
	// Remove brackets and split by comma
	valueList = strings.Trim(valueList, "[]")
	values := strings.Split(valueList, ",")

	// Trim whitespace from each value
	result := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// convertEFSVolume converts ECS EFS volume to Kubernetes NFS volume
func (c *TaskConverter) convertEFSVolume(efsConfig *types.EFSVolumeConfiguration) corev1.VolumeSource {
	if efsConfig.FileSystemId == nil {
		return corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}
	}

	// EFS is essentially NFS v4, so we can use NFS volume in Kubernetes
	// The server would be: <file-system-id>.efs.<region>.amazonaws.com
	server := fmt.Sprintf("%s.efs.%s.amazonaws.com", *efsConfig.FileSystemId, c.region)

	// Root directory defaults to "/"
	path := "/"
	if efsConfig.RootDirectory != nil && *efsConfig.RootDirectory != "" {
		path = *efsConfig.RootDirectory
	}

	nfsVolume := &corev1.NFSVolumeSource{
		Server: server,
		Path:   path,
	}

	// Note: Transit encryption and IAM authorization would need to be handled
	// at the pod level with appropriate sidecars or init containers

	return corev1.VolumeSource{
		NFS: nfsVolume,
	}
}

// convertDockerVolume converts ECS Docker volume to appropriate Kubernetes volume
func (c *TaskConverter) convertDockerVolume(dockerConfig *types.DockerVolumeConfiguration, volumeName string) corev1.VolumeSource {
	// Check the scope of the volume
	scope := "task"
	if dockerConfig.Scope != nil {
		scope = *dockerConfig.Scope
	}

	// Check the driver
	driver := "local"
	if dockerConfig.Driver != nil {
		driver = *dockerConfig.Driver
	}

	switch driver {
	case "local":
		// Local Docker volumes map to PersistentVolumeClaim or emptyDir
		if scope == "shared" {
			// Shared volumes should use PVC
			return corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: c.sanitizeVolumeName(volumeName),
				},
			}
		}
		// Task-scoped local volumes use emptyDir
		return corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		}

	case "rexray/ebs":
		// EBS volume - use AWS EBS volume source
		if ebsVolumeId, ok := dockerConfig.DriverOpts["volumeID"]; ok {
			return corev1.VolumeSource{
				AWSElasticBlockStore: &corev1.AWSElasticBlockStoreVolumeSource{
					VolumeID: ebsVolumeId,
					FSType:   "ext4", // Default filesystem
				},
			}
		}

	case "nfs":
		// NFS driver
		if server, ok := dockerConfig.DriverOpts["server"]; ok {
			path := "/"
			if p, ok := dockerConfig.DriverOpts["path"]; ok {
				path = p
			}
			return corev1.VolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server: server,
					Path:   path,
				},
			}
		}
	}

	// Default fallback
	return corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}
}

// convertFSxWindowsVolume converts FSx Windows File Server to CIFS/SMB volume
func (c *TaskConverter) convertFSxWindowsVolume(fsxConfig *types.FSxWindowsFileServerVolumeConfiguration) corev1.VolumeSource {
	// FSx Windows File Server is not directly supported in Kubernetes
	// We'll need to use a FlexVolume or CSI driver that supports SMB/CIFS
	// For now, we'll store the configuration as annotations

	// In a real implementation, you would use a CSI driver like:
	// - Azure File CSI driver (supports SMB)
	// - AWS FSx CSI driver

	// Fallback to emptyDir with a note that this needs proper CSI driver
	return corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumDefault,
		},
	}
}

// sanitizeVolumeName converts a volume name to be Kubernetes-compatible
func (c *TaskConverter) sanitizeVolumeName(name string) string {
	// Kubernetes names must be lowercase alphanumeric or '-'
	// and must start and end with alphanumeric
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	// Prefix with kecs to avoid conflicts
	return fmt.Sprintf("kecs-volume-%s", name)
}

// addVolumeAnnotations adds annotations for volume configurations that need special handling
func (c *TaskConverter) addVolumeAnnotations(pod *corev1.Pod, volumes []types.Volume) {
	for _, vol := range volumes {
		if vol.Name == nil {
			continue
		}

		// Add EFS-specific annotations
		if vol.EfsVolumeConfiguration != nil {
			efsConfig := vol.EfsVolumeConfiguration
			if efsConfig.FileSystemId != nil {
				annotationPrefix := fmt.Sprintf("kecs.dev/volume-%s-efs", *vol.Name)

				pod.Annotations[annotationPrefix+"-filesystem-id"] = *efsConfig.FileSystemId

				if efsConfig.TransitEncryption != nil {
					pod.Annotations[annotationPrefix+"-transit-encryption"] = *efsConfig.TransitEncryption
				}

				if efsConfig.TransitEncryptionPort != nil {
					pod.Annotations[annotationPrefix+"-transit-encryption-port"] = fmt.Sprintf("%d", *efsConfig.TransitEncryptionPort)
				}

				if efsConfig.AuthorizationConfig != nil {
					if efsConfig.AuthorizationConfig.AccessPointId != nil {
						pod.Annotations[annotationPrefix+"-access-point-id"] = *efsConfig.AuthorizationConfig.AccessPointId
					}
					if efsConfig.AuthorizationConfig.Iam != nil {
						pod.Annotations[annotationPrefix+"-iam"] = *efsConfig.AuthorizationConfig.Iam
					}
				}
			}
		}

		// Add Docker volume annotations
		if vol.DockerVolumeConfiguration != nil {
			dockerConfig := vol.DockerVolumeConfiguration
			annotationPrefix := fmt.Sprintf("kecs.dev/volume-%s-docker", *vol.Name)

			if dockerConfig.Scope != nil {
				pod.Annotations[annotationPrefix+"-scope"] = *dockerConfig.Scope
			}

			if dockerConfig.Driver != nil {
				pod.Annotations[annotationPrefix+"-driver"] = *dockerConfig.Driver
			}

			if dockerConfig.Autoprovision != nil {
				pod.Annotations[annotationPrefix+"-autoprovision"] = fmt.Sprintf("%t", *dockerConfig.Autoprovision)
			}

			// Store driver options as JSON
			if len(dockerConfig.DriverOpts) > 0 {
				if optsJSON, err := json.Marshal(dockerConfig.DriverOpts); err == nil {
					pod.Annotations[annotationPrefix+"-driver-opts"] = string(optsJSON)
				}
			}

			// Store labels as JSON
			if len(dockerConfig.Labels) > 0 {
				if labelsJSON, err := json.Marshal(dockerConfig.Labels); err == nil {
					pod.Annotations[annotationPrefix+"-labels"] = string(labelsJSON)
				}
			}
		}

		// Add FSx Windows annotations
		if vol.FsxWindowsFileServerVolumeConfiguration != nil {
			fsxConfig := vol.FsxWindowsFileServerVolumeConfiguration
			annotationPrefix := fmt.Sprintf("kecs.dev/volume-%s-fsx", *vol.Name)

			if fsxConfig.FileSystemId != nil {
				pod.Annotations[annotationPrefix+"-filesystem-id"] = *fsxConfig.FileSystemId
			}

			if fsxConfig.RootDirectory != nil {
				pod.Annotations[annotationPrefix+"-root-directory"] = *fsxConfig.RootDirectory
			}

			if fsxConfig.AuthorizationConfig != nil {
				if fsxConfig.AuthorizationConfig.CredentialsParameter != nil {
					pod.Annotations[annotationPrefix+"-credentials-parameter"] = *fsxConfig.AuthorizationConfig.CredentialsParameter
				}
				if fsxConfig.AuthorizationConfig.Domain != nil {
					pod.Annotations[annotationPrefix+"-domain"] = *fsxConfig.AuthorizationConfig.Domain
				}
			}
		}
	}
}
