package types

// TaskDefinition represents an ECS task definition
type TaskDefinition struct {
	TaskDefinitionArn       *string                   `json:"taskDefinitionArn,omitempty"`
	ContainerDefinitions    []ContainerDefinition     `json:"containerDefinitions"`
	Family                  *string                   `json:"family"`
	TaskRoleArn             *string                   `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn        *string                   `json:"executionRoleArn,omitempty"`
	NetworkMode             *string                   `json:"networkMode,omitempty"`
	Revision                *int                      `json:"revision,omitempty"`
	Volumes                 []Volume                  `json:"volumes,omitempty"`
	Status                  *string                   `json:"status,omitempty"`
	RequiresAttributes      []Attribute               `json:"requiresAttributes,omitempty"`
	PlacementConstraints    []TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	Compatibilities         []string                  `json:"compatibilities,omitempty"`
	RequiresCompatibilities []string                  `json:"requiresCompatibilities,omitempty"`
	Cpu                     *string                   `json:"cpu,omitempty"`
	Memory                  *string                   `json:"memory,omitempty"`
	InferenceAccelerators   []InferenceAccelerator    `json:"inferenceAccelerators,omitempty"`
	PidMode                 *string                   `json:"pidMode,omitempty"`
	IpcMode                 *string                   `json:"ipcMode,omitempty"`
	ProxyConfiguration      *ProxyConfiguration       `json:"proxyConfiguration,omitempty"`
	RegisteredAt            *string                   `json:"registeredAt,omitempty"`
	DeregisteredAt          *string                   `json:"deregisteredAt,omitempty"`
	RegisteredBy            *string                   `json:"registeredBy,omitempty"`
	EphemeralStorage        *EphemeralStorage         `json:"ephemeralStorage,omitempty"`
	RuntimePlatform         *RuntimePlatform          `json:"runtimePlatform,omitempty"`
}

// ContainerDefinition represents a container definition in a task definition
type ContainerDefinition struct {
	Name                   *string                `json:"name"`
	Image                  *string                `json:"image"`
	Cpu                    *int                   `json:"cpu,omitempty"`
	Memory                 *int                   `json:"memory,omitempty"`
	MemoryReservation      *int                   `json:"memoryReservation,omitempty"`
	Links                  []string               `json:"links,omitempty"`
	PortMappings           []PortMapping          `json:"portMappings,omitempty"`
	Essential              *bool                  `json:"essential,omitempty"`
	EntryPoint             []string               `json:"entryPoint,omitempty"`
	Command                []string               `json:"command,omitempty"`
	Environment            []KeyValuePair         `json:"environment,omitempty"`
	EnvironmentFiles       []EnvironmentFile      `json:"environmentFiles,omitempty"`
	MountPoints            []MountPoint           `json:"mountPoints,omitempty"`
	VolumesFrom            []VolumeFrom           `json:"volumesFrom,omitempty"`
	LinuxParameters        *LinuxParameters       `json:"linuxParameters,omitempty"`
	Secrets                []Secret               `json:"secrets,omitempty"`
	DependsOn              []ContainerDependency  `json:"dependsOn,omitempty"`
	StartTimeout           *int                   `json:"startTimeout,omitempty"`
	StopTimeout            *int                   `json:"stopTimeout,omitempty"`
	Hostname               *string                `json:"hostname,omitempty"`
	User                   *string                `json:"user,omitempty"`
	WorkingDirectory       *string                `json:"workingDirectory,omitempty"`
	DisableNetworking      *bool                  `json:"disableNetworking,omitempty"`
	Privileged             *bool                  `json:"privileged,omitempty"`
	ReadonlyRootFilesystem *bool                  `json:"readonlyRootFilesystem,omitempty"`
	DnsServers             []string               `json:"dnsServers,omitempty"`
	DnsSearchDomains       []string               `json:"dnsSearchDomains,omitempty"`
	ExtraHosts             []HostEntry            `json:"extraHosts,omitempty"`
	DockerSecurityOptions  []string               `json:"dockerSecurityOptions,omitempty"`
	Interactive            *bool                  `json:"interactive,omitempty"`
	PseudoTerminal         *bool                  `json:"pseudoTerminal,omitempty"`
	DockerLabels           map[string]string      `json:"dockerLabels,omitempty"`
	Ulimits                []Ulimit               `json:"ulimits,omitempty"`
	LogConfiguration       *LogConfiguration      `json:"logConfiguration,omitempty"`
	HealthCheck            *HealthCheck           `json:"healthCheck,omitempty"`
	SystemControls         []SystemControl        `json:"systemControls,omitempty"`
	ResourceRequirements   []ResourceRequirement  `json:"resourceRequirements,omitempty"`
	FirelensConfiguration  *FirelensConfiguration `json:"firelensConfiguration,omitempty"`
}

// PortMapping represents a port mapping for a container
type PortMapping struct {
	ContainerPort *int    `json:"containerPort"`
	HostPort      *int    `json:"hostPort,omitempty"`
	Protocol      *string `json:"protocol,omitempty"`
	Name          *string `json:"name,omitempty"`
}

// KeyValuePair represents a key-value pair for environment variables
type KeyValuePair struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

// EnvironmentFile represents an environment file for a container
type EnvironmentFile struct {
	Value *string `json:"value"`
	Type  *string `json:"type"`
}

// MountPoint represents a mount point for a container
type MountPoint struct {
	SourceVolume  *string `json:"sourceVolume"`
	ContainerPath *string `json:"containerPath"`
	ReadOnly      *bool   `json:"readOnly,omitempty"`
}

// Volume represents a volume in a task definition
type Volume struct {
	Name                        *string                      `json:"name,omitempty"`
	Host                        *HostVolumeProperties        `json:"host,omitempty"`
	DockerVolumeConfiguration   *DockerVolumeConfiguration   `json:"dockerVolumeConfiguration,omitempty"`
	EfsVolumeConfiguration      *EFSVolumeConfiguration      `json:"efsVolumeConfiguration,omitempty"`
	FsxWindowsFileServerVolumeConfiguration *FSxWindowsFileServerVolumeConfiguration `json:"fsxWindowsFileServerVolumeConfiguration,omitempty"`
}

// HostVolumeProperties represents host volume properties
type HostVolumeProperties struct {
	SourcePath *string `json:"sourcePath,omitempty"`
}

// DockerVolumeConfiguration represents Docker volume configuration
type DockerVolumeConfiguration struct {
	Scope         *string           `json:"scope,omitempty"`
	Autoprovision *bool             `json:"autoprovision,omitempty"`
	Driver        *string           `json:"driver,omitempty"`
	DriverOpts    map[string]string `json:"driverOpts,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// EFSVolumeConfiguration represents EFS volume configuration
type EFSVolumeConfiguration struct {
	FileSystemId          *string                       `json:"fileSystemId"`
	RootDirectory         *string                       `json:"rootDirectory,omitempty"`
	TransitEncryption     *string                       `json:"transitEncryption,omitempty"`
	TransitEncryptionPort *int                          `json:"transitEncryptionPort,omitempty"`
	AuthorizationConfig   *EFSAuthorizationConfig       `json:"authorizationConfig,omitempty"`
}

// EFSAuthorizationConfig represents EFS authorization configuration
type EFSAuthorizationConfig struct {
	AccessPointId *string `json:"accessPointId,omitempty"`
	Iam           *string `json:"iam,omitempty"`
}

// FSxWindowsFileServerVolumeConfiguration represents FSx Windows File Server volume configuration
type FSxWindowsFileServerVolumeConfiguration struct {
	FileSystemId        *string                                      `json:"fileSystemId"`
	RootDirectory       *string                                      `json:"rootDirectory"`
	AuthorizationConfig *FSxWindowsFileServerAuthorizationConfig    `json:"authorizationConfig"`
}

// FSxWindowsFileServerAuthorizationConfig represents FSx Windows File Server authorization configuration
type FSxWindowsFileServerAuthorizationConfig struct {
	CredentialsParameter *string `json:"credentialsParameter"`
	Domain               *string `json:"domain"`
}

// VolumeFrom represents a volume from another container
type VolumeFrom struct {
	SourceContainer *string `json:"sourceContainer"`
	ReadOnly        *bool   `json:"readOnly,omitempty"`
}

// LinuxParameters represents Linux-specific parameters
type LinuxParameters struct {
	Capabilities       *KernelCapabilities    `json:"capabilities,omitempty"`
	Devices            []Device               `json:"devices,omitempty"`
	InitProcessEnabled *bool                  `json:"initProcessEnabled,omitempty"`
	SharedMemorySize   *int                   `json:"sharedMemorySize,omitempty"`
	Tmpfs              []Tmpfs                `json:"tmpfs,omitempty"`
	MaxSwap            *int                   `json:"maxSwap,omitempty"`
	Swappiness         *int                   `json:"swappiness,omitempty"`
}

// KernelCapabilities represents kernel capabilities
type KernelCapabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// Device represents a device mapping
type Device struct {
	HostPath      *string  `json:"hostPath"`
	ContainerPath *string  `json:"containerPath,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
}

// Tmpfs represents a tmpfs mount
type Tmpfs struct {
	ContainerPath *string  `json:"containerPath"`
	Size          *int     `json:"size"`
	MountOptions  []string `json:"mountOptions,omitempty"`
}

// Secret represents a secret
type Secret struct {
	Name      *string `json:"name"`
	ValueFrom *string `json:"valueFrom"`
}

// ContainerDependency represents a container dependency
type ContainerDependency struct {
	ContainerName *string `json:"containerName"`
	Condition     *string `json:"condition"`
}

// HostEntry represents a host entry
type HostEntry struct {
	Hostname  *string `json:"hostname"`
	IpAddress *string `json:"ipAddress"`
}

// Ulimit represents a ulimit setting
type Ulimit struct {
	Name      *string `json:"name"`
	SoftLimit *int    `json:"softLimit"`
	HardLimit *int    `json:"hardLimit"`
}

// LogConfiguration represents log configuration
type LogConfiguration struct {
	LogDriver *string                   `json:"logDriver"`
	Options   map[string]string         `json:"options,omitempty"`
	SecretOptions []Secret              `json:"secretOptions,omitempty"`
}

// HealthCheck represents a health check
type HealthCheck struct {
	Command     []string `json:"command"`
	Interval    *int     `json:"interval,omitempty"`
	Timeout     *int     `json:"timeout,omitempty"`
	Retries     *int     `json:"retries,omitempty"`
	StartPeriod *int     `json:"startPeriod,omitempty"`
}

// SystemControl represents a system control
type SystemControl struct {
	Namespace *string `json:"namespace"`
	Value     *string `json:"value"`
}

// ResourceRequirement represents a resource requirement
type ResourceRequirement struct {
	Type  *string `json:"type"`
	Value *string `json:"value"`
}

// FirelensConfiguration represents Firelens configuration
type FirelensConfiguration struct {
	Type    *string           `json:"type"`
	Options map[string]string `json:"options,omitempty"`
}

// TaskPlacementConstraint represents a task placement constraint
type TaskPlacementConstraint struct {
	Type       *string `json:"type,omitempty"`
	Expression *string `json:"expression,omitempty"`
}

// InferenceAccelerator represents an inference accelerator
type InferenceAccelerator struct {
	DeviceName *string `json:"deviceName"`
	DeviceType *string `json:"deviceType"`
}

// ProxyConfiguration represents proxy configuration
type ProxyConfiguration struct {
	Type           *string        `json:"type,omitempty"`
	ContainerName  *string        `json:"containerName"`
	Properties     []KeyValuePair `json:"properties,omitempty"`
}

// EphemeralStorage represents ephemeral storage configuration
type EphemeralStorage struct {
	SizeInGiB *int `json:"sizeInGiB"`
}

// RuntimePlatform represents runtime platform configuration
type RuntimePlatform struct {
	CpuArchitecture       *string `json:"cpuArchitecture,omitempty"`
	OperatingSystemFamily *string `json:"operatingSystemFamily,omitempty"`
}