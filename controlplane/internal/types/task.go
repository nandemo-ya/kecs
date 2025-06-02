package types

// Task represents an ECS task
type Task struct {
	TaskArn              string            `json:"taskArn,omitempty"`
	ClusterArn           string            `json:"clusterArn,omitempty"`
	TaskDefinitionArn    string            `json:"taskDefinitionArn,omitempty"`
	ContainerInstanceArn string            `json:"containerInstanceArn,omitempty"`
	OverrideId           string            `json:"overrideId,omitempty"`
	LastStatus           string            `json:"lastStatus,omitempty"`
	DesiredStatus        string            `json:"desiredStatus,omitempty"`
	Cpu                  string            `json:"cpu,omitempty"`
	Memory               string            `json:"memory,omitempty"`
	Containers           []Container       `json:"containers,omitempty"`
	StartedBy            string            `json:"startedBy,omitempty"`
	Version              int               `json:"version,omitempty"`
	StoppedReason        string            `json:"stoppedReason,omitempty"`
	Connectivity         string            `json:"connectivity,omitempty"`
	ConnectivityAt       string            `json:"connectivityAt,omitempty"`
	PullStartedAt        string            `json:"pullStartedAt,omitempty"`
	PullStoppedAt        string            `json:"pullStoppedAt,omitempty"`
	ExecutionStoppedAt   string            `json:"executionStoppedAt,omitempty"`
	CreatedAt            string            `json:"createdAt,omitempty"`
	StartedAt            string            `json:"startedAt,omitempty"`
	StoppingAt           string            `json:"stoppingAt,omitempty"`
	StoppedAt            string            `json:"stoppedAt,omitempty"`
	Group                string            `json:"group,omitempty"`
	LaunchType           string            `json:"launchType,omitempty"`
	CapacityProviderName string            `json:"capacityProviderName,omitempty"`
	PlatformVersion      string            `json:"platformVersion,omitempty"`
	PlatformFamily       string            `json:"platformFamily,omitempty"`
	Attachments          []Attachment      `json:"attachments,omitempty"`
	Attributes           []Attribute       `json:"attributes,omitempty"`
	Tags                 []Tag             `json:"tags,omitempty"`
	EnableExecuteCommand bool              `json:"enableExecuteCommand,omitempty"`
	Overrides            *TaskOverride     `json:"overrides,omitempty"`
	EphemeralStorage     *EphemeralStorage `json:"ephemeralStorage,omitempty"`
	HealthStatus         string            `json:"healthStatus,omitempty"`
}

// Container represents a container in a task
type Container struct {
	ContainerArn      string             `json:"containerArn,omitempty"`
	TaskArn           string             `json:"taskArn,omitempty"`
	Name              string             `json:"name,omitempty"`
	Image             string             `json:"image,omitempty"`
	ImageDigest       string             `json:"imageDigest,omitempty"`
	RuntimeId         string             `json:"runtimeId,omitempty"`
	LastStatus        string             `json:"lastStatus,omitempty"`
	ExitCode          int                `json:"exitCode,omitempty"`
	Reason            string             `json:"reason,omitempty"`
	NetworkBindings   []NetworkBinding   `json:"networkBindings,omitempty"`
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces,omitempty"`
	HealthStatus      string             `json:"healthStatus,omitempty"`
	Cpu               string             `json:"cpu,omitempty"`
	Memory            string             `json:"memory,omitempty"`
	MemoryReservation string             `json:"memoryReservation,omitempty"`
	GpuIds            []string           `json:"gpuIds,omitempty"`
}

// NetworkBinding represents a network binding for a container
type NetworkBinding struct {
	BindIP        string `json:"bindIP,omitempty"`
	ContainerPort int    `json:"containerPort,omitempty"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// NetworkInterface represents a network interface for a container
type NetworkInterface struct {
	AttachmentId       string `json:"attachmentId,omitempty"`
	PrivateIpv4Address string `json:"privateIpv4Address,omitempty"`
	Ipv6Address        string `json:"ipv6Address,omitempty"`
}

// TaskOverride represents overrides for a task
type TaskOverride struct {
	ContainerOverrides []ContainerOverride `json:"containerOverrides,omitempty"`
	TaskRoleArn        *string             `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn   *string             `json:"executionRoleArn,omitempty"`
	Cpu                *string             `json:"cpu,omitempty"`
	Memory             *string             `json:"memory,omitempty"`
	EphemeralStorage   *EphemeralStorage   `json:"ephemeralStorage,omitempty"`
}

// ContainerOverride represents an override for a container
type ContainerOverride struct {
	Name                 *string                `json:"name"`
	Command              []string               `json:"command,omitempty"`
	Environment          []KeyValuePair         `json:"environment,omitempty"`
	EnvironmentFiles     []EnvironmentFile      `json:"environmentFiles,omitempty"`
	Cpu                  *int                   `json:"cpu,omitempty"`
	Memory               *int                   `json:"memory,omitempty"`
	MemoryReservation    *int                   `json:"memoryReservation,omitempty"`
	ResourceRequirements []ResourceRequirement  `json:"resourceRequirements,omitempty"`
}

// RunTaskRequest represents the request to run a task
type RunTaskRequest struct {
	Cluster                  *string                   `json:"cluster,omitempty"`
	TaskDefinition           *string                   `json:"taskDefinition"`
	Count                    *int                      `json:"count,omitempty"`
	Group                    *string                   `json:"group,omitempty"`
	StartedBy                *string                   `json:"startedBy,omitempty"`
	LaunchType               *string                   `json:"launchType,omitempty"`
	CapacityProviderStrategy []CapacityStrategy        `json:"capacityProviderStrategy,omitempty"`
	PlacementConstraints     []PlacementConstraint     `json:"placementConstraints,omitempty"`
	PlacementStrategy        []PlacementStrategy       `json:"placementStrategy,omitempty"`
	PlatformVersion          *string                   `json:"platformVersion,omitempty"`
	EnableECSManagedTags     *bool                     `json:"enableECSManagedTags,omitempty"`
	PropagateTags            *string                   `json:"propagateTags,omitempty"`
	ReferenceId              *string                   `json:"referenceId,omitempty"`
	Tags                     []Tag                     `json:"tags,omitempty"`
	EnableExecuteCommand     *bool                     `json:"enableExecuteCommand,omitempty"`
	Overrides                *TaskOverride             `json:"overrides,omitempty"`
}

// PlacementStrategy represents a placement strategy for a task
type PlacementStrategy struct {
	Type  *string `json:"type"`
	Field *string `json:"field,omitempty"`
}

// PlacementConstraint represents a placement constraint
type PlacementConstraint struct {
	Type       *string `json:"type,omitempty"`
	Expression *string `json:"expression,omitempty"`
}

// RunTaskResponse represents the response from running a task
type RunTaskResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures,omitempty"`
}

// StartTaskRequest represents the request to start a task
type StartTaskRequest struct {
	Cluster              *string       `json:"cluster,omitempty"`
	TaskDefinition       *string       `json:"taskDefinition"`
	ContainerInstances   []string      `json:"containerInstances"`
	Group                *string       `json:"group,omitempty"`
	StartedBy            *string       `json:"startedBy,omitempty"`
	EnableECSManagedTags *bool         `json:"enableECSManagedTags,omitempty"`
	PropagateTags        *string       `json:"propagateTags,omitempty"`
	ReferenceId          *string       `json:"referenceId,omitempty"`
	Tags                 []Tag         `json:"tags,omitempty"`
	EnableExecuteCommand *bool         `json:"enableExecuteCommand,omitempty"`
	Overrides            *TaskOverride `json:"overrides,omitempty"`
}

// StartTaskResponse represents the response from starting a task
type StartTaskResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures,omitempty"`
}

// StopTaskRequest represents the request to stop a task
type StopTaskRequest struct {
	Cluster *string `json:"cluster,omitempty"`
	Task    *string `json:"task"`
	Reason  *string `json:"reason,omitempty"`
}

// StopTaskResponse represents the response from stopping a task
type StopTaskResponse struct {
	Task *Task `json:"task"`
}

// DescribeTasksRequest represents the request to describe tasks
type DescribeTasksRequest struct {
	Cluster *string  `json:"cluster,omitempty"`
	Tasks   []string `json:"tasks"`
	Include []string `json:"include,omitempty"`
}

// DescribeTasksResponse represents the response from describing tasks
type DescribeTasksResponse struct {
	Tasks    []Task    `json:"tasks"`
	Failures []Failure `json:"failures,omitempty"`
}

// ListTasksRequest represents the request to list tasks
type ListTasksRequest struct {
	Cluster           *string `json:"cluster,omitempty"`
	ContainerInstance *string `json:"containerInstance,omitempty"`
	Family            *string `json:"family,omitempty"`
	StartedBy         *string `json:"startedBy,omitempty"`
	ServiceName       *string `json:"serviceName,omitempty"`
	DesiredStatus     *string `json:"desiredStatus,omitempty"`
	LaunchType        *string `json:"launchType,omitempty"`
	MaxResults        *int    `json:"maxResults,omitempty"`
	NextToken         *string `json:"nextToken,omitempty"`
}

// ListTasksResponse represents the response from listing tasks
type ListTasksResponse struct {
	TaskArns  []string `json:"taskArns"`
	NextToken *string  `json:"nextToken,omitempty"`
}

// GetTaskProtectionRequest represents the request to get task protection
type GetTaskProtectionRequest struct {
	Cluster *string  `json:"cluster,omitempty"`
	Tasks   []string `json:"tasks"`
}

// ProtectionResult represents a protection result for a task
type ProtectionResult struct {
	TaskArn               *string `json:"taskArn"`
	ProtectionEnabled     *bool   `json:"protectionEnabled"`
	ExpirationDate        *string `json:"expirationDate,omitempty"`
	ProtectedFromScaleIn  *bool   `json:"protectedFromScaleIn,omitempty"`
	ProtectedFromScaleOut *bool   `json:"protectedFromScaleOut,omitempty"`
}

// GetTaskProtectionResponse represents the response from getting task protection
type GetTaskProtectionResponse struct {
	ProtectedTasks []ProtectionResult `json:"protectedTasks"`
	Failures       []Failure          `json:"failures,omitempty"`
}