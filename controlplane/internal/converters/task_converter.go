package converters

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/nandemo-ya/kecs/controlplane/internal/artifacts"
	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
	"github.com/nandemo-ya/kecs/controlplane/internal/proxy"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
	"github.com/nandemo-ya/kecs/controlplane/internal/types"
)

// TaskConverter converts ECS task definitions to Kubernetes resources
type TaskConverter struct {
	region                string
	accountID             string
	cloudWatchIntegration cloudwatch.Integration
	artifactManager       *artifacts.Manager
	proxyManager          *proxy.Manager
	networkConverter      *NetworkConverter
}

// NewTaskConverter creates a new task converter
func NewTaskConverter(region, accountID string) *TaskConverter {
	return &TaskConverter{
		region:    region,
		accountID: accountID,
	}
}

// NewTaskConverterWithCloudWatch creates a new task converter with CloudWatch integration
func NewTaskConverterWithCloudWatch(region, accountID string, cwIntegration cloudwatch.Integration) *TaskConverter {
	return &TaskConverter{
		region:                region,
		accountID:             accountID,
		cloudWatchIntegration: cwIntegration,
		networkConverter:      NewNetworkConverter(region, accountID),
	}
}

// SetArtifactManager sets the artifact manager for the task converter
func (c *TaskConverter) SetArtifactManager(am *artifacts.Manager) {
	c.artifactManager = am
}

// SetProxyManager sets the proxy manager for the task converter
func (c *TaskConverter) SetProxyManager(pm *proxy.Manager) {
	c.proxyManager = pm
}

// ConvertTaskToPod converts an ECS task definition and RunTask request to a Kubernetes Pod
func (c *TaskConverter) ConvertTaskToPod(
	taskDef *storage.TaskDefinition,
	runTaskReqJSON []byte, // Accept JSON bytes to avoid circular import
	cluster *storage.Cluster,
	taskID string,
) (*corev1.Pod, error) {
	// Import the generated types to properly handle network configuration
	var runTaskReq struct {
		Cluster              *string `json:"cluster,omitempty"`
		TaskDefinition       *string `json:"taskDefinition"`
		Count                *int    `json:"count,omitempty"`
		Group                *string `json:"group,omitempty"`
		StartedBy            *string `json:"startedBy,omitempty"`
		LaunchType           *string `json:"launchType,omitempty"`
		NetworkConfiguration *struct {
			AwsvpcConfiguration *struct {
				Subnets        []string `json:"subnets"`
				SecurityGroups []string `json:"securityGroups,omitempty"`
				AssignPublicIp *string  `json:"assignPublicIp,omitempty"`
			} `json:"awsvpcConfiguration,omitempty"`
		} `json:"networkConfiguration,omitempty"`
		PlacementConstraints []types.PlacementConstraint `json:"placementConstraints,omitempty"`
		PlacementStrategy    []types.PlacementStrategy   `json:"placementStrategy,omitempty"`
		PlatformVersion      *string                     `json:"platformVersion,omitempty"`
		EnableECSManagedTags *bool                       `json:"enableECSManagedTags,omitempty"`
		PropagateTags        *string                     `json:"propagateTags,omitempty"`
		ReferenceId          *string                     `json:"referenceId,omitempty"`
		Tags                 []types.Tag                 `json:"tags,omitempty"`
		EnableExecuteCommand *bool                       `json:"enableExecuteCommand,omitempty"`
		Overrides            *types.TaskOverride         `json:"overrides,omitempty"`
	}
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
			Name:      taskID,
			Namespace: c.getNamespace(cluster),
			Labels: map[string]string{
				"kecs.dev/cluster":       cluster.Name,
				"kecs.dev/task-id":       taskID,
				"kecs.dev/task-family":   taskDef.Family,
				"kecs.dev/task-revision": fmt.Sprintf("%d", taskDef.Revision),
				"kecs.dev/launch-type":   c.getLaunchTypeFromRequest(runTaskReq.LaunchType),
				"kecs.dev/managed-by":    "kecs",
			},
			Annotations: map[string]string{
				"kecs.dev/task-arn":            c.generateTaskARN(cluster.Name, taskID),
				"kecs.dev/task-definition-arn": taskDef.ARN,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever, // ECS tasks don't restart by default
			Containers:    c.convertContainersWithOverrides(containerDefs, taskDef, runTaskReq.Overrides),
			Volumes:       c.convertVolumes(volumes),
		},
	}

	// Add init containers and volumes for artifacts if needed
	if c.artifactManager != nil {
		initContainers, artifactVolumes := c.createArtifactInitContainers(containerDefs)
		if len(initContainers) > 0 {
			pod.Spec.InitContainers = initContainers
			pod.Spec.Volumes = append(pod.Spec.Volumes, artifactVolumes...)
		}
	}

	// Apply network mode
	networkMode := types.GetNetworkMode(&taskDef.NetworkMode)
	if networkMode == types.NetworkModeHost {
		pod.Spec.HostNetwork = true
	}

	// Add network configuration annotations
	if runTaskReq.NetworkConfiguration != nil && c.networkConverter != nil {
		// Add network mode annotation
		pod.Annotations["ecs.amazonaws.com/network-mode"] = string(networkMode)

		// Add awsvpc configuration if present
		if runTaskReq.NetworkConfiguration.AwsvpcConfiguration != nil {
			awsvpc := runTaskReq.NetworkConfiguration.AwsvpcConfiguration
			if len(awsvpc.Subnets) > 0 {
				pod.Annotations["ecs.amazonaws.com/subnets"] = strings.Join(awsvpc.Subnets, ",")
			}
			if len(awsvpc.SecurityGroups) > 0 {
				pod.Annotations["ecs.amazonaws.com/security-groups"] = strings.Join(awsvpc.SecurityGroups, ",")
			}
			if awsvpc.AssignPublicIp != nil {
				pod.Annotations["ecs.amazonaws.com/assign-public-ip"] = *awsvpc.AssignPublicIp
			}
		}
	} else {
		// Just add the network mode from task definition
		pod.Annotations["ecs.amazonaws.com/network-mode"] = string(networkMode)
	}

	// Apply PID mode
	if taskDef.PidMode == "host" {
		pod.Spec.HostPID = true
	}

	// Apply task role via ServiceAccount only if IAM integration is enabled
	if taskDef.TaskRoleARN != "" && config.GetBool("features.iamIntegration") {
		// Extract role name from ARN
		roleName := c.extractRoleNameFromARN(taskDef.TaskRoleARN)
		if roleName != "" {
			// ServiceAccount name is typically rolename-sa
			serviceAccountName := fmt.Sprintf("%s-sa", roleName)
			pod.Spec.ServiceAccountName = serviceAccountName

			// Add annotations for tracking
			pod.Annotations["kecs.dev/task-role-arn"] = taskDef.TaskRoleARN
			pod.Annotations["kecs.dev/task-role-name"] = roleName
		}
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

	// Add secret annotations
	c.addSecretAnnotations(pod, containerDefs)

	// Apply IAM role annotations if specified (for tracking purposes)
	if taskDef.ExecutionRoleARN != "" {
		pod.ObjectMeta.Annotations["kecs.dev/execution-role-arn"] = taskDef.ExecutionRoleARN
	}

	// Apply CloudWatch logs configuration
	if c.cloudWatchIntegration != nil {
		c.applyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)
		// Note: FluentBit sidecar is no longer needed as we use Vector DaemonSet
		// The annotations set by applyCloudWatchLogsConfiguration will be read by Vector
	}

	// Add AWS proxy sidecar if proxy manager is available
	if c.proxyManager != nil && c.proxyManager.GetSidecarProxy() != nil {
		sidecarProxy := c.proxyManager.GetSidecarProxy()
		if sidecarProxy.ShouldInjectSidecar(pod) {
			if err := sidecarProxy.InjectSidecar(pod); err != nil {
				logging.Warn("Failed to inject AWS proxy sidecar", "error", err)
			}
		}
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

	// taskArn is no longer needed since we removed FluentBit sidecar
	// taskArn := taskDef.ARN

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

		// Add artifact volume mounts
		if c.artifactManager != nil && def.Artifacts != nil && len(def.Artifacts) > 0 {
			artifactVolumeMount := corev1.VolumeMount{
				Name:      fmt.Sprintf("artifacts-%s", *def.Name),
				MountPath: "/artifacts",
				ReadOnly:  true,
			}
			container.VolumeMounts = append(container.VolumeMounts, artifactVolumeMount)
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
			// Use the same probe for both liveness and readiness
			container.LivenessProbe = c.convertHealthCheck(def.HealthCheck)
			// Create a copy for readiness probe with potentially different settings
			readinessProbe := c.convertHealthCheck(def.HealthCheck)
			// Readiness probe can have shorter initial delay for faster service availability
			if readinessProbe.InitialDelaySeconds > 10 {
				readinessProbe.InitialDelaySeconds = 10
			}
			container.ReadinessProbe = readinessProbe
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

		// Note: Log configuration is now handled by annotations set in applyCloudWatchLogsConfiguration
		// Vector DaemonSet will read these annotations to route logs to CloudWatch

		containers = append(containers, container)
	}

	// Apply container overrides if any
	if runTaskReq.Overrides != nil && runTaskReq.Overrides.ContainerOverrides != nil {
		for i := range containers {
			for _, override := range runTaskReq.Overrides.ContainerOverrides {
				if override.Name != nil && containers[i].Name == *override.Name {
					c.applyContainerOverride(&containers[i], &override)
				}
			}
		}
	}

	return containers
}

// convertEnvironment converts ECS environment variables to Kubernetes
func (c *TaskConverter) convertEnvironment(env []types.KeyValuePair) []corev1.EnvVar {
	k8sEnv := make([]corev1.EnvVar, 0, len(env))
	for _, e := range env {
		if e.Name != nil && e.Value != nil {
			k8sEnv = append(k8sEnv, corev1.EnvVar{
				Name:  *e.Name,
				Value: *e.Value,
			})
		}
	}
	return k8sEnv
}

// convertPortMappings converts ECS port mappings to Kubernetes
func (c *TaskConverter) convertPortMappings(mappings []types.PortMapping) []corev1.ContainerPort {
	ports := make([]corev1.ContainerPort, 0, len(mappings))

	for _, m := range mappings {
		if m.ContainerPort == nil {
			continue
		}

		port := corev1.ContainerPort{
			ContainerPort: int32(*m.ContainerPort),
		}

		if m.HostPort != nil {
			port.HostPort = int32(*m.HostPort)
		}

		if m.Protocol != nil {
			switch strings.ToLower(*m.Protocol) {
			case "tcp":
				port.Protocol = corev1.ProtocolTCP
			case "udp":
				port.Protocol = corev1.ProtocolUDP
			case "sctp":
				port.Protocol = corev1.ProtocolSCTP
			default:
				port.Protocol = corev1.ProtocolTCP
			}
		} else {
			port.Protocol = corev1.ProtocolTCP
		}

		if m.Name != nil {
			port.Name = *m.Name
		}

		ports = append(ports, port)
	}

	return ports
}

// convertContainersWithOverrides converts ECS container definitions to Kubernetes containers with overrides
func (c *TaskConverter) convertContainersWithOverrides(
	containerDefs []types.ContainerDefinition,
	taskDef *storage.TaskDefinition,
	overrides *types.TaskOverride,
) []corev1.Container {
	// Create a minimal RunTaskRequest with just the overrides
	runTaskReq := &types.RunTaskRequest{
		Overrides: overrides,
	}
	return c.convertContainers(containerDefs, taskDef, runTaskReq)
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

// convertHealthCheck converts ECS health check to Kubernetes probe
func (c *TaskConverter) convertHealthCheck(hc *types.HealthCheck) *corev1.Probe {
	probe := &corev1.Probe{}

	// Process command based on format
	if hc.Command != nil && len(hc.Command) > 0 {
		cmdType := hc.Command[0]

		switch cmdType {
		case "CMD-SHELL":
			// Shell command - use exec probe with sh -c
			if len(hc.Command) > 1 {
				probe.Exec = &corev1.ExecAction{
					Command: []string{"sh", "-c", hc.Command[1]},
				}
			}
		case "CMD":
			// Direct command - use exec probe
			if len(hc.Command) > 1 {
				probe.Exec = &corev1.ExecAction{
					Command: hc.Command[1:],
				}
			}
		case "HTTP":
			// HTTP health check
			if len(hc.Command) > 1 {
				// Parse URL from command
				path := hc.Command[1]
				port := int32(80) // Default port

				// Extract port if specified
				if len(hc.Command) > 2 {
					// Try to parse port from command
					if p, err := strconv.Atoi(hc.Command[2]); err == nil {
						port = int32(p)
					}
				}

				probe.HTTPGet = &corev1.HTTPGetAction{
					Path: path,
					Port: intstr.FromInt(int(port)),
				}
			}
		default:
			// Assume direct command format
			probe.Exec = &corev1.ExecAction{
				Command: hc.Command,
			}
		}
	}

	// Set timing parameters
	if hc.Interval != nil {
		probe.PeriodSeconds = int32(*hc.Interval)
	} else {
		probe.PeriodSeconds = 30 // Default
	}

	if hc.Timeout != nil {
		probe.TimeoutSeconds = int32(*hc.Timeout)
	} else {
		probe.TimeoutSeconds = 5 // Default
	}

	if hc.Retries != nil {
		probe.FailureThreshold = int32(*hc.Retries)
	} else {
		probe.FailureThreshold = 3 // Default
	}

	if hc.StartPeriod != nil {
		probe.InitialDelaySeconds = int32(*hc.StartPeriod)
	} else {
		probe.InitialDelaySeconds = 30 // Default
	}

	// Kubernetes successThreshold is always 1 for liveness probe
	probe.SuccessThreshold = 1

	return probe
}

// parseUser parses user string and returns SecurityContext
func (c *TaskConverter) parseUser(user string) *corev1.SecurityContext {
	sc := &corev1.SecurityContext{}

	// Parse user:group format
	parts := strings.Split(user, ":")
	if len(parts) > 0 && parts[0] != "" {
		// Try to parse as number
		if uid, err := strconv.ParseInt(parts[0], 10, 64); err == nil {
			sc.RunAsUser = ptr.To(uid)
		}
	}

	if len(parts) > 1 && parts[1] != "" {
		// Try to parse as number
		if gid, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			sc.RunAsGroup = ptr.To(gid)
		}
	}

	return sc
}

// getNamespace determines the namespace for the pod
func (c *TaskConverter) getNamespace(cluster *storage.Cluster) string {
	// Create namespace based on cluster name and region
	return fmt.Sprintf("%s-%s", cluster.Name, cluster.Region)
}

// getLaunchType determines the launch type from the request
func (c *TaskConverter) getLaunchType(req *types.RunTaskRequest) string {
	if req.LaunchType != nil {
		return *req.LaunchType
	}
	if req.CapacityProviderStrategy != nil && len(req.CapacityProviderStrategy) > 0 {
		return "CAPACITY_PROVIDER"
	}
	return "EC2"
}

// getLaunchTypeFromRequest determines the launch type from a string pointer
func (c *TaskConverter) getLaunchTypeFromRequest(launchType *string) string {
	if launchType != nil {
		return *launchType
	}
	return "EC2"
}

// generateTaskARN generates a task ARN
func (c *TaskConverter) generateTaskARN(clusterName, taskID string) string {
	return fmt.Sprintf("arn:aws:ecs:%s:%s:task/%s/%s", c.region, c.accountID, clusterName, taskID)
}

// applyResourceConstraints applies task-level resource constraints to the pod
func (c *TaskConverter) applyResourceConstraints(pod *corev1.Pod, taskCPU, taskMemory string) {
	var cpuMillis int64
	var memoryMi int64

	// Parse CPU (can be "256", "0.25 vCPU", "1 vCPU", etc.)
	if taskCPU != "" {
		if strings.Contains(taskCPU, "vCPU") {
			// Parse vCPU format
			cpuStr := strings.TrimSuffix(strings.TrimSpace(taskCPU), " vCPU")
			if cpuFloat, err := strconv.ParseFloat(cpuStr, 64); err == nil {
				cpuMillis = int64(cpuFloat * 1000)
			}
		} else {
			// Parse CPU units (1024 = 1 vCPU)
			if cpuUnits, err := strconv.ParseInt(taskCPU, 10, 64); err == nil {
				cpuMillis = cpuUnits * 1000 / 1024
			}
		}
	}

	// Parse Memory (in MiB)
	if taskMemory != "" {
		if mem, err := strconv.ParseInt(taskMemory, 10, 64); err == nil {
			memoryMi = mem
		}
	}

	// Apply constraints to containers
	if cpuMillis > 0 || memoryMi > 0 {
		// Count containers with existing resource requests
		var totalRequestedCPU int64
		var totalRequestedMemory int64
		containersWithoutResources := 0

		for i := range pod.Spec.Containers {
			container := &pod.Spec.Containers[i]
			if container.Resources.Requests != nil {
				if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					totalRequestedCPU += cpu.MilliValue()
				}
				if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					// Convert to MiB
					totalRequestedMemory += mem.Value() / (1024 * 1024)
				}
			} else {
				containersWithoutResources++
			}
		}

		// If some containers don't have resources defined, distribute remaining resources
		if containersWithoutResources > 0 {
			remainingCPU := cpuMillis - totalRequestedCPU
			remainingMemory := memoryMi - totalRequestedMemory

			if remainingCPU > 0 || remainingMemory > 0 {
				cpuPerContainer := remainingCPU / int64(containersWithoutResources)
				memoryPerContainer := remainingMemory / int64(containersWithoutResources)

				for i := range pod.Spec.Containers {
					container := &pod.Spec.Containers[i]
					if container.Resources.Requests == nil {
						container.Resources.Requests = corev1.ResourceList{}
					}
					if container.Resources.Limits == nil {
						container.Resources.Limits = corev1.ResourceList{}
					}

					// Only set if not already set
					if _, ok := container.Resources.Requests[corev1.ResourceCPU]; !ok && cpuPerContainer > 0 {
						container.Resources.Requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpuPerContainer, resource.DecimalSI)
						container.Resources.Limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(cpuPerContainer, resource.DecimalSI)
					}
					if _, ok := container.Resources.Requests[corev1.ResourceMemory]; !ok && memoryPerContainer > 0 {
						memQuantity := resource.MustParse(fmt.Sprintf("%dMi", memoryPerContainer))
						container.Resources.Requests[corev1.ResourceMemory] = memQuantity
						container.Resources.Limits[corev1.ResourceMemory] = memQuantity
					}
				}
			}
		} else if totalRequestedCPU > 0 || totalRequestedMemory > 0 {
			// Scale existing resources proportionally to fit task constraints
			if cpuMillis > 0 && totalRequestedCPU > 0 {
				cpuScale := float64(cpuMillis) / float64(totalRequestedCPU)
				for i := range pod.Spec.Containers {
					container := &pod.Spec.Containers[i]
					if container.Resources.Requests != nil {
						if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
							scaledCPU := int64(float64(cpu.MilliValue()) * cpuScale)
							if container.Resources.Limits == nil {
								container.Resources.Limits = corev1.ResourceList{}
							}
							container.Resources.Requests[corev1.ResourceCPU] = *resource.NewMilliQuantity(scaledCPU, resource.DecimalSI)
							container.Resources.Limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(scaledCPU, resource.DecimalSI)
						}
					}
				}
			}

			if memoryMi > 0 && totalRequestedMemory > 0 {
				memScale := float64(memoryMi) / float64(totalRequestedMemory)
				for i := range pod.Spec.Containers {
					container := &pod.Spec.Containers[i]
					if container.Resources.Requests != nil {
						if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
							scaledMemMiB := int64(float64(mem.Value()/(1024*1024)) * memScale)
							memQuantity := resource.MustParse(fmt.Sprintf("%dMi", scaledMemMiB))
							if container.Resources.Limits == nil {
								container.Resources.Limits = corev1.ResourceList{}
							}
							container.Resources.Requests[corev1.ResourceMemory] = memQuantity
							container.Resources.Limits[corev1.ResourceMemory] = memQuantity
						}
					}
				}
			}
		}
	}
}

// applyOverrides applies task overrides to the pod
func (c *TaskConverter) applyOverrides(pod *corev1.Pod, overrides *types.TaskOverride) {
	// Apply task-level overrides
	if overrides.Cpu != nil {
		c.applyResourceConstraints(pod, *overrides.Cpu, "")
	}
	if overrides.Memory != nil {
		c.applyResourceConstraints(pod, "", *overrides.Memory)
	}

	if overrides.TaskRoleArn != nil {
		pod.Annotations["kecs.dev/task-role-arn"] = *overrides.TaskRoleArn
	}

	if overrides.ExecutionRoleArn != nil {
		pod.Annotations["kecs.dev/execution-role-arn"] = *overrides.ExecutionRoleArn
	}

	// Container overrides are handled in convertContainers
}

// applyContainerOverride applies container-specific overrides
func (c *TaskConverter) applyContainerOverride(container *corev1.Container, override *types.ContainerOverride) {
	if override.Command != nil {
		container.Command = override.Command
	}

	if override.Environment != nil {
		// Merge or replace environment variables
		envMap := make(map[string]string)
		for _, env := range container.Env {
			envMap[env.Name] = env.Value
		}

		for _, envVar := range override.Environment {
			if envVar.Name != nil && envVar.Value != nil {
				envMap[*envVar.Name] = *envVar.Value
			}
		}

		// Rebuild env array
		container.Env = make([]corev1.EnvVar, 0, len(envMap))
		for name, value := range envMap {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  name,
				Value: value,
			})
		}
	}

	// Apply resource overrides
	if override.Cpu != nil {
		cpuMillis := *override.Cpu * 1000 / 1024
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = corev1.ResourceList{}
		}
		cpuQuantity := resource.NewMilliQuantity(int64(cpuMillis), resource.DecimalSI)
		container.Resources.Requests[corev1.ResourceCPU] = *cpuQuantity
		container.Resources.Limits[corev1.ResourceCPU] = *cpuQuantity
	}

	if override.Memory != nil {
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = corev1.ResourceList{}
		}
		memQuantity := resource.MustParse(fmt.Sprintf("%dMi", *override.Memory))
		container.Resources.Requests[corev1.ResourceMemory] = memQuantity
		container.Resources.Limits[corev1.ResourceMemory] = memQuantity
	}

	if override.MemoryReservation != nil {
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		memQuantity := resource.MustParse(fmt.Sprintf("%dMi", *override.MemoryReservation))
		container.Resources.Requests[corev1.ResourceMemory] = memQuantity
	}
}

// applyPlacementConstraints converts ECS placement constraints to Kubernetes node affinity
func (c *TaskConverter) applyPlacementConstraints(pod *corev1.Pod, constraints []types.PlacementConstraint) {
	if len(constraints) == 0 {
		return
	}

	// Initialize node affinity if needed
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}

	// Handle different constraint types
	for _, constraint := range constraints {
		if constraint.Type == nil {
			continue
		}

		switch *constraint.Type {
		case "memberOf":
			if constraint.Expression != nil {
				c.parseMemberOfExpression(*constraint.Expression, pod)
			}

		case "distinctInstance":
			// Implement anti-affinity to ensure tasks run on different nodes
			if pod.Spec.Affinity.PodAntiAffinity == nil {
				pod.Spec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			}

			// Add anti-affinity rule as preferred (not required)
			pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
				pod.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
				corev1.WeightedPodAffinityTerm{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kecs.dev/task-family": pod.Labels["kecs.dev/task-family"],
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			)
		}
	}
}

// parseMemberOfExpression parses ECS memberOf expressions and converts to node selector or affinity
func (c *TaskConverter) parseMemberOfExpression(expression string, pod *corev1.Pod) {
	// ECS expressions examples:
	// - attribute:ecs.instance-type == t2.micro
	// - attribute:ecs.availability-zone in [us-west-2a, us-west-2b]
	// - attribute:custom-attribute =~ pattern*

	// Simple parser for common patterns
	parts := strings.Fields(expression)
	if len(parts) < 3 {
		return
	}

	attribute := strings.TrimPrefix(parts[0], "attribute:")
	operator := parts[1]
	value := strings.Join(parts[2:], " ")

	// Convert ECS attribute to Kubernetes label
	k8sLabel := c.convertECSAttributeToK8sLabel(attribute)

	// For simple equality, use node selector
	if operator == "==" {
		if pod.Spec.NodeSelector == nil {
			pod.Spec.NodeSelector = make(map[string]string)
		}
		pod.Spec.NodeSelector[k8sLabel] = value
		return
	}

	// For other operators, use node affinity
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}
	nodeAffinity := pod.Spec.Affinity.NodeAffinity

	// Initialize node selector term
	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{},
		}
	}

	requirement := corev1.NodeSelectorRequirement{
		Key: k8sLabel,
	}

	switch operator {

	case "!=":
		requirement.Operator = corev1.NodeSelectorOpNotIn
		requirement.Values = []string{value}

	case "=~":
		// Regex matching - Kubernetes doesn't support regex in node selectors
		// We'll need to expand common patterns or use a different approach
		// For now, treat as exact match but in production you might want to
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

// addSecretAnnotations adds annotations for secrets used by the task
func (c *TaskConverter) addSecretAnnotations(pod *corev1.Pod, containerDefs []types.ContainerDefinition) {
	secretIndex := 0
	for _, containerDef := range containerDefs {
		if containerDef.Secrets != nil {
			for _, secret := range containerDef.Secrets {
				if secret.Name != nil && secret.ValueFrom != nil {
					// Add annotation for each secret with container and environment variable info
					annotationKey := fmt.Sprintf("kecs.dev/secret-%d-arn", secretIndex)
					annotationValue := fmt.Sprintf("%s:%s:%s", *containerDef.Name, *secret.Name, *secret.ValueFrom)
					pod.Annotations[annotationKey] = annotationValue
					secretIndex++
				}
			}
		}
	}

	// Add total count of secrets
	if secretIndex > 0 {
		pod.Annotations["kecs.dev/secret-count"] = fmt.Sprintf("%d", secretIndex)
	}
}

// applyIAMRole applies IAM role configuration to the pod
func (c *TaskConverter) applyIAMRole(pod *corev1.Pod, roleArn string) {
	// Add role ARN annotation
	pod.ObjectMeta.Annotations["kecs.dev/task-role-arn"] = roleArn

	// Extract role name from ARN
	// ARN format: arn:aws:iam::account-id:role/role-name
	parts := strings.Split(roleArn, "/")
	if len(parts) >= 2 {
		roleName := parts[len(parts)-1]

		// ServiceAccount name would be created by IAM integration
		serviceAccountName := fmt.Sprintf("%s-sa", roleName)

		// Set ServiceAccount on the pod
		pod.Spec.ServiceAccountName = serviceAccountName

		// Add label for easier querying
		pod.ObjectMeta.Labels["kecs.dev/iam-role"] = roleName

		// Inject AWS credentials for LocalStack
		c.injectAWSCredentials(pod)
	}
}

// injectAWSCredentials adds AWS credential environment variables for LocalStack
func (c *TaskConverter) injectAWSCredentials(pod *corev1.Pod) {
	// AWS credentials for LocalStack
	awsEnvVars := []corev1.EnvVar{
		{
			Name:  "AWS_ACCESS_KEY_ID",
			Value: "test",
		},
		{
			Name:  "AWS_SECRET_ACCESS_KEY",
			Value: "test",
		},
		{
			Name:  "AWS_DEFAULT_REGION",
			Value: c.region,
		},
	}

	// Add credentials to all containers
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		// Check if env vars already exist to avoid duplicates
		envMap := make(map[string]bool)
		for _, env := range container.Env {
			envMap[env.Name] = true
		}

		// Add AWS env vars if not already present
		for _, envVar := range awsEnvVars {
			if !envMap[envVar.Name] {
				container.Env = append(container.Env, envVar)
			}
		}
	}
}

// applyCloudWatchLogsConfiguration applies CloudWatch logs configuration to the pod
func (c *TaskConverter) applyCloudWatchLogsConfiguration(pod *corev1.Pod, containerDefs []types.ContainerDefinition, taskDef *storage.TaskDefinition) {
	if c.cloudWatchIntegration == nil {
		return
	}

	// Check if any container has awslogs driver
	hasAwslogs := false
	for _, def := range containerDefs {
		if def.LogConfiguration != nil && def.LogConfiguration.LogDriver != nil && *def.LogConfiguration.LogDriver == "awslogs" {
			hasAwslogs = true
			break
		}
	}

	if !hasAwslogs {
		return
	}

	// Add CloudWatch logs annotations to the pod
	for _, def := range containerDefs {
		if def.LogConfiguration != nil && def.LogConfiguration.LogDriver != nil && *def.LogConfiguration.LogDriver == "awslogs" {
			if def.Name == nil {
				continue
			}

			options := def.LogConfiguration.Options
			if options == nil {
				options = make(map[string]string)
			}

			// Get log configuration from task definition
			logGroupName := options["awslogs-group"]
			if logGroupName == "" {
				logGroupName = c.cloudWatchIntegration.GetLogGroupForTask(taskDef.ARN)
			}

			// Handle stream prefix - this is what Vector will use
			streamPrefix := options["awslogs-stream-prefix"]
			if streamPrefix == "" {
				streamPrefix = *def.Name
			}

			// Get region from options or use default
			region := options["awslogs-region"]
			if region == "" {
				region = "us-east-1"
			}

			// Add annotations for each container's log configuration
			// These annotations will be read by Vector to determine log routing
			annotationPrefix := fmt.Sprintf("kecs.dev/container-%s-logs", *def.Name)
			pod.Annotations[annotationPrefix+"-driver"] = "awslogs"
			pod.Annotations[annotationPrefix+"-group"] = logGroupName
			pod.Annotations[annotationPrefix+"-stream-prefix"] = streamPrefix
			pod.Annotations[annotationPrefix+"-region"] = region

			// Add any additional options as annotations
			for key, value := range options {
				if key != "awslogs-group" && key != "awslogs-stream-prefix" && key != "awslogs-region" {
					// Convert awslogs-* format to annotation format
					annotationKey := strings.ReplaceAll(key, "awslogs-", "")
					pod.Annotations[annotationPrefix+"-"+annotationKey] = value
				}
			}
		}
	}

	// Add global CloudWatch logs enabled annotation
	pod.Annotations["kecs.dev/cloudwatch-logs-enabled"] = "true"

	// Create log streams for each container
	for _, def := range containerDefs {
		if def.LogConfiguration != nil && def.LogConfiguration.LogDriver != nil &&
			*def.LogConfiguration.LogDriver == "awslogs" && def.Name != nil {
			options := def.LogConfiguration.Options
			if options == nil {
				options = make(map[string]string)
			}

			logGroupName := options["awslogs-group"]
			if logGroupName == "" {
				logGroupName = c.cloudWatchIntegration.GetLogGroupForTask(taskDef.ARN)
			}

			// Use stream prefix to construct stream name
			streamPrefix := options["awslogs-stream-prefix"]
			if streamPrefix == "" {
				streamPrefix = *def.Name
			}
			// Construct stream name: prefix/pod-name
			logStreamName := fmt.Sprintf("%s/%s", streamPrefix, pod.Name)

			// Create log group and stream
			if err := c.cloudWatchIntegration.CreateLogGroup(logGroupName); err != nil {
				logging.Warn("Failed to create log group", "logGroup", logGroupName, "error", err)
			}
			if err := c.cloudWatchIntegration.CreateLogStream(logGroupName, logStreamName); err != nil {
				logging.Warn("Failed to create log stream", "logGroup", logGroupName, "logStream", logStreamName, "error", err)
			}
		}
	}
}

// ApplyCloudWatchLogsConfiguration is a public wrapper for testing
func (c *TaskConverter) ApplyCloudWatchLogsConfiguration(pod *corev1.Pod, containerDefs []types.ContainerDefinition, taskDef *storage.TaskDefinition) {
	c.applyCloudWatchLogsConfiguration(pod, containerDefs, taskDef)
}

// applyTags adds tags as labels to the pod
func (c *TaskConverter) applyTags(pod *corev1.Pod, tags []types.Tag) {
	for _, tag := range tags {
		if tag.Key != nil && tag.Value != nil {
			// Kubernetes labels have restrictions, so we need to sanitize
			key := c.sanitizeLabelKey(*tag.Key)
			value := c.sanitizeLabelValue(*tag.Value)

			// Skip if sanitization results in empty key
			if key != "" {
				pod.Labels[key] = value
			}
		}
	}
}

// sanitizeLabelKey converts a tag key to a valid Kubernetes label key
func (c *TaskConverter) sanitizeLabelKey(key string) string {
	// Kubernetes label keys must:
	// - Be 63 characters or less
	// - Begin and end with alphanumeric
	// - Contain only alphanumeric, '-', '_', '.'
	// - Have optional prefix (up to 253 chars) separated by '/'

	// For simplicity, prefix with "tag." to avoid conflicts
	key = "tag." + key

	// Replace invalid characters
	key = regexp.MustCompile(`[^a-zA-Z0-9\-_\./]`).ReplaceAllString(key, "-")

	// Ensure it starts and ends with alphanumeric
	key = regexp.MustCompile(`^[^a-zA-Z0-9]+`).ReplaceAllString(key, "")
	key = regexp.MustCompile(`[^a-zA-Z0-9]+$`).ReplaceAllString(key, "")

	// Truncate if necessary
	if len(key) > 63 {
		key = key[:63]
		// Ensure it still ends with alphanumeric after truncation
		key = regexp.MustCompile(`[^a-zA-Z0-9]+$`).ReplaceAllString(key, "")
	}

	return key
}

// sanitizeLabelValue converts a tag value to a valid Kubernetes label value
func (c *TaskConverter) sanitizeLabelValue(value string) string {
	// Kubernetes label values must:
	// - Be 63 characters or less
	// - Be empty or begin and end with alphanumeric
	// - Contain only alphanumeric, '-', '_', '.'

	if value == "" {
		return ""
	}

	// Replace invalid characters
	value = regexp.MustCompile(`[^a-zA-Z0-9\-_\.]`).ReplaceAllString(value, "-")

	// Ensure it starts and ends with alphanumeric
	value = regexp.MustCompile(`^[^a-zA-Z0-9]+`).ReplaceAllString(value, "")
	value = regexp.MustCompile(`[^a-zA-Z0-9]+$`).ReplaceAllString(value, "")

	// Truncate if necessary
	if len(value) > 63 {
		value = value[:63]
		// Ensure it still ends with alphanumeric after truncation
		value = regexp.MustCompile(`[^a-zA-Z0-9]+$`).ReplaceAllString(value, "")
	}

	return value
}

// convertECSAttributeToK8sLabel maps ECS attributes to Kubernetes labels
func (c *TaskConverter) convertECSAttributeToK8sLabel(attribute string) string {
	// Common ECS attributes mapping
	mappings := map[string]string{
		"ecs.instance-type":     "node.kubernetes.io/instance-type",
		"ecs.availability-zone": "topology.kubernetes.io/zone",
		"ecs.os-type":           "kubernetes.io/os",
		"ecs.cpu-architecture":  "kubernetes.io/arch",
		"ecs.ami-id":            "kecs.dev/ami-id",
		"ecs.instance-id":       "kecs.dev/instance-id",
		"ecs.subnet-id":         "kecs.dev/subnet-id",
		"ecs.vpc-id":            "kecs.dev/vpc-id",
	}

	if k8sLabel, ok := mappings[attribute]; ok {
		return k8sLabel
	}

	// For custom attributes, prefix with kecs.dev/
	// Replace both dots and colons with hyphens for valid K8s label names
	sanitized := strings.ReplaceAll(attribute, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	return "kecs.dev/" + sanitized
}

// convertSecrets converts ECS secrets to Kubernetes environment variables
func (c *TaskConverter) convertSecrets(secrets []types.Secret) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0, len(secrets))

	for _, secret := range secrets {
		if secret.Name == nil || secret.ValueFrom == nil {
			continue
		}

		// Parse the secret ARN
		secretInfo, err := c.parseSecretArn(*secret.ValueFrom)
		if err != nil {
			// If we can't parse it, skip it
			// In production, you might want to handle this differently
			continue
		}

		envVar := corev1.EnvVar{
			Name: *secret.Name,
		}

		switch secretInfo.Source {
		case "secretsmanager":
			// Reference the synced Kubernetes secret in kecs-system namespace
			// Note: Cross-namespace secret reference requires proper RBAC setup
			envVar.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: c.getK8sSecretName("secretsmanager", secretInfo.SecretName),
					},
					Key: secretInfo.Key,
				},
			}

		case "ssm":
			// All SSM parameters are now stored as Secrets for consistency
			envVar.ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: c.getK8sSecretName("ssm", secretInfo.SecretName),
					},
					Key: "value",
				},
			}
		}

		envVars = append(envVars, envVar)
	}

	return envVars
}

// parseSecretArn parses an AWS secret ARN and extracts relevant information
func (c *TaskConverter) parseSecretArn(arn string) (*SecretInfo, error) {
	// ARN formats:
	// Secrets Manager: arn:aws:secretsmanager:region:account-id:secret:name-6RandomChars:key::
	// SSM: arn:aws:ssm:region:account-id:parameter/name

	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return nil, fmt.Errorf("invalid ARN format: %s", arn)
	}

	service := parts[2]
	info := &SecretInfo{}

	switch service {
	case "secretsmanager":
		info.Source = "secretsmanager"
		// Extract secret name and key from remaining parts
		// Format: arn:aws:secretsmanager:region:account-id:secret:name-6RandomChars:key::
		if len(parts) >= 7 {
			info.SecretName = parts[6]
			// Check if a JSON key is specified at index 7
			if len(parts) > 7 && parts[7] != "" && parts[7] != "*" {
				info.Key = parts[7]
			} else {
				// No JSON key specified, the entire secret value will be used
				// When synced by Secrets Manager integration, JSON secrets will have all keys available
				info.Key = "value"
			}
		} else {
			return nil, fmt.Errorf("invalid Secrets Manager ARN format: %s", arn)
		}

	case "ssm":
		info.Source = "ssm"
		// Extract parameter name from ARN
		// Format: arn:aws:ssm:region:account-id:parameter/path/to/param
		// The parameter path starts after "parameter/"
		resourcePart := parts[5]
		if strings.HasPrefix(resourcePart, "parameter/") {
			info.SecretName = strings.TrimPrefix(resourcePart, "parameter/")
		} else if strings.HasPrefix(resourcePart, "parameter") && len(parts) > 6 {
			// Sometimes the path might be in the next part
			info.SecretName = parts[6]
		} else {
			info.SecretName = resourcePart
		}
		info.Key = "value"

	default:
		return nil, fmt.Errorf("unsupported secret service: %s", service)
	}

	return info, nil
}

// getNamespacedSecretName returns the namespace-aware secret name for LocalStack
func (c *TaskConverter) getNamespacedSecretName(cluster *storage.Cluster, secretName string) string {
	// Format: <namespace>/<secret-name>
	// The namespace already contains cluster and region information
	namespace := c.getNamespace(cluster)
	return fmt.Sprintf("%s/%s", namespace, secretName)
}

// sanitizeSecretName converts a secret name to be Kubernetes-compatible
func (c *TaskConverter) sanitizeSecretName(name string) string {
	// Determine the prefix based on the source
	// The integration modules will handle the actual prefixing,
	// but we need to ensure consistency here

	// Remove the random suffix from Secrets Manager secret names
	// Format: my-secret-AbCdEf -> my-secret
	if idx := strings.LastIndex(name, "-"); idx > 0 && len(name)-idx == 7 {
		// Check if last part looks like a random suffix (6 chars)
		suffix := name[idx+1:]
		if len(suffix) == 6 && regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(suffix) {
			name = name[:idx]
		}
	}

	// Handle path separators for hierarchical secrets
	name = strings.ReplaceAll(name, "/", "-")

	// Similar to volume names, but for secrets
	name = strings.ToLower(name)
	name = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")

	// Return the sanitized name without prefix
	// The actual prefix (sm- or ssm-) will be added by the integration modules
	return name
}

// extractRoleNameFromARN extracts the role name from an IAM role ARN
func (c *TaskConverter) extractRoleNameFromARN(arn string) string {
	if arn == "" {
		return ""
	}

	// ARN format: arn:aws:iam::account-id:role/role-name
	parts := strings.Split(arn, ":")
	if len(parts) >= 6 && parts[2] == "iam" {
		// Get the last part after "role/"
		resourcePart := parts[5]
		if strings.HasPrefix(resourcePart, "role/") {
			return strings.TrimPrefix(resourcePart, "role/")
		}
	}

	// If it's not a valid ARN, assume it's already a role name
	if !strings.HasPrefix(arn, "arn:") {
		return arn
	}

	return ""
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
		if totalRequestedCPU > 0 && taskCPUMillis > 0 {
			proportionalCPU := (containerCPU * taskCPUMillis) / totalRequestedCPU
			if proportionalCPU > 0 {
				cpuQuantity := resource.NewMilliQuantity(proportionalCPU, resource.DecimalSI)
				container.Resources.Requests[corev1.ResourceCPU] = *cpuQuantity
				container.Resources.Limits[corev1.ResourceCPU] = *cpuQuantity
			}
		}

		if totalRequestedMemory > 0 && taskMemoryMi > 0 {
			proportionalMemory := (containerMemory * taskMemoryMi) / totalRequestedMemory
			if proportionalMemory > 0 {
				memQuantity := resource.MustParse(fmt.Sprintf("%dMi", proportionalMemory))
				container.Resources.Requests[corev1.ResourceMemory] = memQuantity
				container.Resources.Limits[corev1.ResourceMemory] = memQuantity
			}
		}
	}
}

// createArtifactInitContainers creates init containers for downloading artifacts
func (c *TaskConverter) createArtifactInitContainers(containerDefs []types.ContainerDefinition) ([]corev1.Container, []corev1.Volume) {
	var initContainers []corev1.Container
	var volumes []corev1.Volume

	for _, def := range containerDefs {
		if def.Artifacts == nil || len(def.Artifacts) == 0 {
			continue
		}

		// Create volume for this container's artifacts
		volumeName := fmt.Sprintf("artifacts-%s", *def.Name)
		volume := corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		volumes = append(volumes, volume)

		// Create init container for downloading artifacts
		initContainer := corev1.Container{
			Name:    fmt.Sprintf("artifact-downloader-%s", *def.Name),
			Image:   "amazon/aws-cli:latest", // Use AWS CLI image for S3 support
			Command: []string{"/bin/sh", "-c"},
			Args:    []string{c.generateArtifactDownloadScript(def.Artifacts)},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumeName,
					MountPath: "/artifacts",
				},
			},
			// Add environment variables for S3 endpoint if LocalStack is configured
			Env: c.getArtifactEnvironment(),
		}

		initContainers = append(initContainers, initContainer)
	}

	return initContainers, volumes
}

// generateArtifactDownloadScript generates a shell script to download artifacts
func (c *TaskConverter) generateArtifactDownloadScript(artifacts []types.Artifact) string {
	var commands []string

	for _, artifact := range artifacts {
		if artifact.ArtifactUrl == nil || artifact.TargetPath == nil {
			continue
		}

		url := *artifact.ArtifactUrl
		targetPath := filepath.Join("/artifacts", *artifact.TargetPath)

		// Create directory for target
		commands = append(commands, fmt.Sprintf("mkdir -p $(dirname %s)", targetPath))

		// Download based on URL type
		if strings.HasPrefix(url, "s3://") {
			// Use AWS CLI to download from S3
			commands = append(commands, fmt.Sprintf("aws s3 cp %s %s", url, targetPath))
		} else if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
			// Use curl for HTTP/HTTPS (available in aws-cli image)
			commands = append(commands, fmt.Sprintf("curl -s -L -o %s %s", targetPath, url))
		}

		// Set permissions if specified
		if artifact.Permissions != nil {
			commands = append(commands, fmt.Sprintf("chmod %s %s", *artifact.Permissions, targetPath))
		}
	}

	// Join all commands with && to ensure they all succeed
	return strings.Join(commands, " && ")
}

// getArtifactEnvironment returns environment variables for artifact downloading
func (c *TaskConverter) getArtifactEnvironment() []corev1.EnvVar {
	var env []corev1.EnvVar

	// Basic AWS credentials
	env = append(env, corev1.EnvVar{
		Name:  "AWS_ACCESS_KEY_ID",
		Value: "test",
	})
	env = append(env, corev1.EnvVar{
		Name:  "AWS_SECRET_ACCESS_KEY",
		Value: "test",
	})
	env = append(env, corev1.EnvVar{
		Name:  "AWS_DEFAULT_REGION",
		Value: c.region,
	})

	// Add LocalStack S3 endpoint if available
	// In proxy mode, use http://localstack-proxy.default.svc.cluster.local:4566
	// Otherwise, use direct LocalStack endpoint
	if c.artifactManager != nil {
		// Add S3 endpoint for LocalStack
		env = append(env, corev1.EnvVar{
			Name:  "AWS_ENDPOINT_URL_S3",
			Value: "http://localstack-proxy.default.svc.cluster.local:4566",
		})
		// For aws-cli v1 compatibility
		env = append(env, corev1.EnvVar{
			Name:  "S3_ENDPOINT_URL",
			Value: "http://localstack-proxy.default.svc.cluster.local:4566",
		})
	}

	return env
}

// getK8sSecretName returns the Kubernetes secret name for a given source and secret name
func (c *TaskConverter) getK8sSecretName(source, secretName string) string {
	switch source {
	case "secretsmanager":
		// Remove the random suffix that Secrets Manager adds (e.g., -AbCdEf)
		re := regexp.MustCompile(`-[A-Za-z0-9]{6}$`)
		cleanName := re.ReplaceAllString(secretName, "")
		cleanName = strings.ToLower(cleanName)
		cleanName = strings.ReplaceAll(cleanName, "/", "-")
		cleanName = strings.Trim(cleanName, "-")
		return "sm-" + cleanName
	case "ssm":
		cleanName := strings.Trim(secretName, "/")
		cleanName = strings.ReplaceAll(cleanName, "/", "-")
		cleanName = strings.ToLower(cleanName)
		return "ssm-" + cleanName
	default:
		return "unknown-" + strings.ToLower(secretName)
	}
}

// getK8sConfigMapName returns the Kubernetes ConfigMap name for a given SSM parameter
// DEPRECATED: All SSM parameters are now stored as Secrets for consistency
func (c *TaskConverter) getK8sConfigMapName(parameterName string) string {
	cleanName := strings.Trim(parameterName, "/")
	cleanName = strings.ReplaceAll(cleanName, "/", "-")
	cleanName = strings.ToLower(cleanName)
	return "ssm-cm-" + cleanName
}

// isSSMParameterSensitive determines if an SSM parameter should be treated as sensitive
// DEPRECATED: All SSM parameters are now stored as Secrets for consistency
func (c *TaskConverter) isSSMParameterSensitive(parameterName string) bool {
	// All SSM parameters are now treated as sensitive and stored as Secrets
	return true
}

// getPlaceholderSecretValue returns placeholder values for secrets
// NOTE: This is now deprecated in favor of actual Kubernetes secret references
// Kept for backward compatibility and testing
func (c *TaskConverter) getPlaceholderSecretValue(source, secretName, key string) string {
	// Generate deterministic placeholder values based on the secret name and key
	// This ensures consistency across deployments while being obviously fake

	switch source {
	case "secretsmanager":
		// Generate different placeholder values for different secret types
		if strings.Contains(strings.ToLower(secretName), "db") || strings.Contains(strings.ToLower(secretName), "database") {
			// Check if key or secretName contains password/pass
			if strings.Contains(strings.ToLower(key), "password") || strings.Contains(strings.ToLower(key), "pass") ||
				strings.Contains(strings.ToLower(secretName), "password") || strings.Contains(strings.ToLower(secretName), "pass") {
				return "placeholder-db-password-from-secrets-manager"
			}
			return "placeholder-db-connection-from-secrets-manager"
		}
		if strings.Contains(strings.ToLower(secretName), "jwt") {
			return "placeholder-jwt-secret-from-secrets-manager"
		}
		if strings.Contains(strings.ToLower(secretName), "encrypt") {
			return "placeholder-encryption-key-from-secrets-manager"
		}
		return fmt.Sprintf("placeholder-secret-from-secrets-manager-%s-%s", secretName, key)

	case "ssm":
		// Generate placeholder values for SSM parameters
		if strings.Contains(strings.ToLower(secretName), "database") {
			return "postgresql://placeholder:placeholder@localhost:5432/placeholder"
		}
		if strings.Contains(strings.ToLower(secretName), "api_key") {
			return "placeholder-api-key-from-ssm"
		}
		if strings.Contains(strings.ToLower(secretName), "feature") {
			return `{"new_ui": true, "beta_features": true, "maintenance_mode": false}`
		}
		return fmt.Sprintf("placeholder-parameter-from-ssm-%s", secretName)

	default:
		return fmt.Sprintf("placeholder-unknown-secret-%s", secretName)
	}
}
