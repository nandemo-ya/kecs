package api

import (
	"encoding/json"
	"net/http"
)

// TaskDefinition represents an ECS task definition
type TaskDefinition struct {
	TaskDefinitionArn            string                 `json:"taskDefinitionArn,omitempty"`
	ContainerDefinitions         []ContainerDefinition  `json:"containerDefinitions"`
	Family                       string                 `json:"family"`
	TaskRoleArn                  string                 `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn             string                 `json:"executionRoleArn,omitempty"`
	NetworkMode                  string                 `json:"networkMode,omitempty"`
	Revision                     int                    `json:"revision,omitempty"`
	Volumes                      []Volume               `json:"volumes,omitempty"`
	Status                       string                 `json:"status,omitempty"`
	RequiresAttributes           []Attribute            `json:"requiresAttributes,omitempty"`
	PlacementConstraints         []TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	Compatibilities              []string               `json:"compatibilities,omitempty"`
	RequiresCompatibilities      []string               `json:"requiresCompatibilities,omitempty"`
	Cpu                          string                 `json:"cpu,omitempty"`
	Memory                       string                 `json:"memory,omitempty"`
	InferenceAccelerators        []InferenceAccelerator `json:"inferenceAccelerators,omitempty"`
	PidMode                      string                 `json:"pidMode,omitempty"`
	IpcMode                      string                 `json:"ipcMode,omitempty"`
	ProxyConfiguration           *ProxyConfiguration    `json:"proxyConfiguration,omitempty"`
	RegisteredAt                 string                 `json:"registeredAt,omitempty"`
	DeregisteredAt               string                 `json:"deregisteredAt,omitempty"`
	RegisteredBy                 string                 `json:"registeredBy,omitempty"`
	EphemeralStorage             *EphemeralStorage      `json:"ephemeralStorage,omitempty"`
	RuntimePlatform              *RuntimePlatform       `json:"runtimePlatform,omitempty"`
}

// ContainerDefinition represents a container definition in a task definition
type ContainerDefinition struct {
	Name                   string                `json:"name"`
	Image                  string                `json:"image"`
	Cpu                    int                   `json:"cpu,omitempty"`
	Memory                 int                   `json:"memory,omitempty"`
	MemoryReservation      int                   `json:"memoryReservation,omitempty"`
	Links                  []string              `json:"links,omitempty"`
	PortMappings           []PortMapping         `json:"portMappings,omitempty"`
	Essential              bool                  `json:"essential,omitempty"`
	EntryPoint             []string              `json:"entryPoint,omitempty"`
	Command                []string              `json:"command,omitempty"`
	Environment            []KeyValuePair        `json:"environment,omitempty"`
	EnvironmentFiles       []EnvironmentFile     `json:"environmentFiles,omitempty"`
	MountPoints            []MountPoint          `json:"mountPoints,omitempty"`
	VolumesFrom            []VolumeFrom          `json:"volumesFrom,omitempty"`
	LinuxParameters        *LinuxParameters      `json:"linuxParameters,omitempty"`
	Secrets                []Secret              `json:"secrets,omitempty"`
	DependsOn              []ContainerDependency `json:"dependsOn,omitempty"`
	StartTimeout           int                   `json:"startTimeout,omitempty"`
	StopTimeout            int                   `json:"stopTimeout,omitempty"`
	Hostname               string                `json:"hostname,omitempty"`
	User                   string                `json:"user,omitempty"`
	WorkingDirectory       string                `json:"workingDirectory,omitempty"`
	DisableNetworking      bool                  `json:"disableNetworking,omitempty"`
	Privileged             bool                  `json:"privileged,omitempty"`
	ReadonlyRootFilesystem bool                  `json:"readonlyRootFilesystem,omitempty"`
	DnsServers             []string              `json:"dnsServers,omitempty"`
	DnsSearchDomains       []string              `json:"dnsSearchDomains,omitempty"`
	ExtraHosts             []HostEntry           `json:"extraHosts,omitempty"`
	DockerSecurityOptions  []string              `json:"dockerSecurityOptions,omitempty"`
	Interactive            bool                  `json:"interactive,omitempty"`
	PseudoTerminal         bool                  `json:"pseudoTerminal,omitempty"`
	DockerLabels           map[string]string     `json:"dockerLabels,omitempty"`
	Ulimits                []Ulimit              `json:"ulimits,omitempty"`
	LogConfiguration       *LogConfiguration     `json:"logConfiguration,omitempty"`
	HealthCheck            *HealthCheck          `json:"healthCheck,omitempty"`
	SystemControls         []SystemControl       `json:"systemControls,omitempty"`
	ResourceRequirements   []ResourceRequirement `json:"resourceRequirements,omitempty"`
	FirelensConfiguration  *FirelensConfiguration `json:"firelensConfiguration,omitempty"`
}

// PortMapping represents a port mapping for a container
type PortMapping struct {
	ContainerPort int    `json:"containerPort"`
	HostPort      int    `json:"hostPort,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

// EnvironmentFile represents an environment file for a container
type EnvironmentFile struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

// MountPoint represents a mount point for a container
type MountPoint struct {
	SourceVolume  string `json:"sourceVolume"`
	ContainerPath string `json:"containerPath"`
	ReadOnly      bool   `json:"readOnly,omitempty"`
}

// VolumeFrom represents a volume from another container
type VolumeFrom struct {
	SourceContainer string `json:"sourceContainer"`
	ReadOnly        bool   `json:"readOnly,omitempty"`
}

// LinuxParameters represents Linux-specific options for a container
type LinuxParameters struct {
	Capabilities      *KernelCapabilities `json:"capabilities,omitempty"`
	Devices           []Device            `json:"devices,omitempty"`
	InitProcessEnabled bool               `json:"initProcessEnabled,omitempty"`
	SharedMemorySize  int                 `json:"sharedMemorySize,omitempty"`
	Tmpfs             []Tmpfs             `json:"tmpfs,omitempty"`
	MaxSwap           int                 `json:"maxSwap,omitempty"`
	Swappiness        int                 `json:"swappiness,omitempty"`
}

// KernelCapabilities represents kernel capabilities for a container
type KernelCapabilities struct {
	Add  []string `json:"add,omitempty"`
	Drop []string `json:"drop,omitempty"`
}

// Device represents a device mapping for a container
type Device struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath,omitempty"`
	Permissions   []string `json:"permissions,omitempty"`
}

// Tmpfs represents a tmpfs mount for a container
type Tmpfs struct {
	ContainerPath string   `json:"containerPath"`
	Size          int      `json:"size"`
	MountOptions  []string `json:"mountOptions,omitempty"`
}

// Secret represents a secret for a container
type Secret struct {
	Name      string `json:"name"`
	ValueFrom string `json:"valueFrom"`
}

// ContainerDependency represents a dependency between containers
type ContainerDependency struct {
	ContainerName string `json:"containerName"`
	Condition     string `json:"condition"`
}

// HostEntry represents a host entry for a container
type HostEntry struct {
	Hostname  string `json:"hostname"`
	IpAddress string `json:"ipAddress"`
}

// Ulimit represents a ulimit for a container
type Ulimit struct {
	Name      string `json:"name"`
	SoftLimit int    `json:"softLimit"`
	HardLimit int    `json:"hardLimit"`
}

// LogConfiguration represents a log configuration for a container
type LogConfiguration struct {
	LogDriver string            `json:"logDriver"`
	Options   map[string]string `json:"options,omitempty"`
	SecretOptions []Secret      `json:"secretOptions,omitempty"`
}

// HealthCheck represents a health check for a container
type HealthCheck struct {
	Command             []string `json:"command"`
	Interval            int      `json:"interval,omitempty"`
	Timeout             int      `json:"timeout,omitempty"`
	Retries             int      `json:"retries,omitempty"`
	StartPeriod         int      `json:"startPeriod,omitempty"`
}

// SystemControl represents a system control for a container
type SystemControl struct {
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

// ResourceRequirement represents a resource requirement for a container
type ResourceRequirement struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

// FirelensConfiguration represents a firelens configuration for a container
type FirelensConfiguration struct {
	Type    string `json:"type"`
	Options map[string]string `json:"options,omitempty"`
}

// Volume represents a volume in a task definition
type Volume struct {
	Name          string        `json:"name"`
	Host          *HostVolumeProperties `json:"host,omitempty"`
	DockerVolumeConfiguration *DockerVolumeConfiguration `json:"dockerVolumeConfiguration,omitempty"`
	EfsVolumeConfiguration    *EFSVolumeConfiguration    `json:"efsVolumeConfiguration,omitempty"`
	FsxWindowsFileServerVolumeConfiguration *FSxWindowsFileServerVolumeConfiguration `json:"fsxWindowsFileServerVolumeConfiguration,omitempty"`
}

// HostVolumeProperties represents host volume properties
type HostVolumeProperties struct {
	SourcePath string `json:"sourcePath,omitempty"`
}

// DockerVolumeConfiguration represents a docker volume configuration
type DockerVolumeConfiguration struct {
	Scope         string            `json:"scope,omitempty"`
	Autoprovision bool              `json:"autoprovision,omitempty"`
	Driver        string            `json:"driver,omitempty"`
	DriverOpts    map[string]string `json:"driverOpts,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// EFSVolumeConfiguration represents an EFS volume configuration
type EFSVolumeConfiguration struct {
	FileSystemId          string `json:"fileSystemId"`
	RootDirectory         string `json:"rootDirectory,omitempty"`
	TransitEncryption     string `json:"transitEncryption,omitempty"`
	TransitEncryptionPort int    `json:"transitEncryptionPort,omitempty"`
	AuthorizationConfig   *EFSAuthorizationConfig `json:"authorizationConfig,omitempty"`
}

// EFSAuthorizationConfig represents an EFS authorization configuration
type EFSAuthorizationConfig struct {
	AccessPointId string `json:"accessPointId,omitempty"`
	Iam           string `json:"iam,omitempty"`
}

// FSxWindowsFileServerVolumeConfiguration represents an FSx Windows File Server volume configuration
type FSxWindowsFileServerVolumeConfiguration struct {
	FileSystemId  string `json:"fileSystemId"`
	RootDirectory string `json:"rootDirectory"`
	AuthorizationConfig *FSxWindowsFileServerAuthorizationConfig `json:"authorizationConfig,omitempty"`
}

// FSxWindowsFileServerAuthorizationConfig represents an FSx Windows File Server authorization configuration
type FSxWindowsFileServerAuthorizationConfig struct {
	CredentialsParameter string `json:"credentialsParameter"`
	Domain              string `json:"domain"`
}

// Attribute represents an attribute in a task definition
type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// TaskPlacementConstraint represents a placement constraint for a task
type TaskPlacementConstraint struct {
	Type       string `json:"type"`
	Expression string `json:"expression,omitempty"`
}

// InferenceAccelerator represents an inference accelerator for a task
type InferenceAccelerator struct {
	DeviceName    string `json:"deviceName"`
	DeviceType    string `json:"deviceType"`
}

// ProxyConfiguration represents a proxy configuration for a task
type ProxyConfiguration struct {
	Type          string         `json:"type,omitempty"`
	ContainerName string         `json:"containerName"`
	Properties    []KeyValuePair `json:"properties,omitempty"`
}

// EphemeralStorage represents ephemeral storage for a task
type EphemeralStorage struct {
	SizeInGiB int `json:"sizeInGiB"`
}

// RuntimePlatform represents a runtime platform for a task
type RuntimePlatform struct {
	CpuArchitecture       string `json:"cpuArchitecture,omitempty"`
	OperatingSystemFamily string `json:"operatingSystemFamily,omitempty"`
}

// RegisterTaskDefinitionRequest represents the request to register a task definition
type RegisterTaskDefinitionRequest struct {
	ContainerDefinitions    []ContainerDefinition  `json:"containerDefinitions"`
	Family                  string                 `json:"family"`
	TaskRoleArn             string                 `json:"taskRoleArn,omitempty"`
	ExecutionRoleArn        string                 `json:"executionRoleArn,omitempty"`
	NetworkMode             string                 `json:"networkMode,omitempty"`
	Volumes                 []Volume               `json:"volumes,omitempty"`
	PlacementConstraints    []TaskPlacementConstraint `json:"placementConstraints,omitempty"`
	RequiresCompatibilities []string               `json:"requiresCompatibilities,omitempty"`
	Cpu                     string                 `json:"cpu,omitempty"`
	Memory                  string                 `json:"memory,omitempty"`
	Tags                    []Tag                  `json:"tags,omitempty"`
	PidMode                 string                 `json:"pidMode,omitempty"`
	IpcMode                 string                 `json:"ipcMode,omitempty"`
	ProxyConfiguration      *ProxyConfiguration    `json:"proxyConfiguration,omitempty"`
	InferenceAccelerators   []InferenceAccelerator `json:"inferenceAccelerators,omitempty"`
	EphemeralStorage        *EphemeralStorage      `json:"ephemeralStorage,omitempty"`
	RuntimePlatform         *RuntimePlatform       `json:"runtimePlatform,omitempty"`
}

// RegisterTaskDefinitionResponse represents the response from registering a task definition
type RegisterTaskDefinitionResponse struct {
	TaskDefinition TaskDefinition `json:"taskDefinition"`
	Tags          []Tag          `json:"tags,omitempty"`
}

// DeregisterTaskDefinitionRequest represents the request to deregister a task definition
type DeregisterTaskDefinitionRequest struct {
	TaskDefinition string `json:"taskDefinition"`
}

// DeregisterTaskDefinitionResponse represents the response from deregistering a task definition
type DeregisterTaskDefinitionResponse struct {
	TaskDefinition TaskDefinition `json:"taskDefinition"`
}

// DescribeTaskDefinitionRequest represents the request to describe a task definition
type DescribeTaskDefinitionRequest struct {
	TaskDefinition string   `json:"taskDefinition"`
	Include        []string `json:"include,omitempty"`
}

// DescribeTaskDefinitionResponse represents the response from describing a task definition
type DescribeTaskDefinitionResponse struct {
	TaskDefinition TaskDefinition `json:"taskDefinition"`
	Tags           []Tag          `json:"tags,omitempty"`
}

// ListTaskDefinitionsRequest represents the request to list task definitions
type ListTaskDefinitionsRequest struct {
	FamilyPrefix string `json:"familyPrefix,omitempty"`
	Status       string `json:"status,omitempty"`
	Sort         string `json:"sort,omitempty"`
	MaxResults   int    `json:"maxResults,omitempty"`
	NextToken    string `json:"nextToken,omitempty"`
}

// ListTaskDefinitionsResponse represents the response from listing task definitions
type ListTaskDefinitionsResponse struct {
	TaskDefinitionArns []string `json:"taskDefinitionArns"`
	NextToken          string   `json:"nextToken,omitempty"`
}

// DeleteTaskDefinitionsRequest represents the request to delete task definitions
type DeleteTaskDefinitionsRequest struct {
	TaskDefinitions []string `json:"taskDefinitions"`
}

// DeleteTaskDefinitionsResponse represents the response from deleting task definitions
type DeleteTaskDefinitionsResponse struct {
	TaskDefinitions []TaskDefinition `json:"taskDefinitions"`
	Failures        []Failure        `json:"failures,omitempty"`
}

// registerTaskDefinitionEndpoints registers all task definition-related API endpoints
func (s *Server) registerTaskDefinitionEndpoints(mux *http.ServeMux) {
	mux.HandleFunc("/v1/registertaskdefinition", s.handleRegisterTaskDefinition)
	mux.HandleFunc("/v1/deregistertaskdefinition", s.handleDeregisterTaskDefinition)
	mux.HandleFunc("/v1/describetaskdefinition", s.handleDescribeTaskDefinition)
	mux.HandleFunc("/v1/listtaskdefinitions", s.handleListTaskDefinitions)
	mux.HandleFunc("/v1/deletetaskdefinitions", s.handleDeleteTaskDefinitions)
}

// handleRegisterTaskDefinition handles the RegisterTaskDefinition API endpoint
func (s *Server) handleRegisterTaskDefinition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterTaskDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task definition registration logic

	// For now, return a mock response
	resp := RegisterTaskDefinitionResponse{
		TaskDefinition: TaskDefinition{
			TaskDefinitionArn:    "arn:aws:ecs:region:account:task-definition/" + req.Family + ":1",
			Family:               req.Family,
			ContainerDefinitions: req.ContainerDefinitions,
			Revision:             1,
			Status:               "ACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeregisterTaskDefinition handles the DeregisterTaskDefinition API endpoint
func (s *Server) handleDeregisterTaskDefinition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeregisterTaskDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task definition deregistration logic

	// For now, return a mock response
	resp := DeregisterTaskDefinitionResponse{
		TaskDefinition: TaskDefinition{
			TaskDefinitionArn: req.TaskDefinition,
			Status:            "INACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDescribeTaskDefinition handles the DescribeTaskDefinition API endpoint
func (s *Server) handleDescribeTaskDefinition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DescribeTaskDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task definition description logic

	// For now, return a mock response
	resp := DescribeTaskDefinitionResponse{
		TaskDefinition: TaskDefinition{
			TaskDefinitionArn: req.TaskDefinition,
			Family:            "sample-family",
			Revision:          1,
			Status:            "ACTIVE",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleListTaskDefinitions handles the ListTaskDefinitions API endpoint
func (s *Server) handleListTaskDefinitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ListTaskDefinitionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task definition listing logic

	// For now, return a mock response
	resp := ListTaskDefinitionsResponse{
		TaskDefinitionArns: []string{"arn:aws:ecs:region:account:task-definition/sample-family:1"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDeleteTaskDefinitions handles the DeleteTaskDefinitions API endpoint
func (s *Server) handleDeleteTaskDefinitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteTaskDefinitionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual task definition deletion logic

	// For now, return a mock response
	resp := DeleteTaskDefinitionsResponse{
		TaskDefinitions: []TaskDefinition{
			{
				TaskDefinitionArn: req.TaskDefinitions[0],
				Status:            "DELETE_IN_PROGRESS",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
