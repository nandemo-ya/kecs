package api

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
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


// HTTP Handlers for ECS Task Definition operations

// handleECSRegisterTaskDefinition handles the RegisterTaskDefinition operation
func (s *Server) handleECSRegisterTaskDefinition(w http.ResponseWriter, body []byte) {
	// Parse body as a generic map to handle generated type limitations
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			errorResponse := map[string]interface{}{
				"__type": "InvalidParameterException",
				"message": "Invalid request format",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}

	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
		return
	}

	ctx := context.Background()

	// Validate required fields
	if family, ok := requestData["family"].(string); !ok || family == "" {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request: Missing required field 'family'",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	if containerDefs, ok := requestData["containerDefinitions"].([]interface{}); !ok || len(containerDefs) == 0 {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "Invalid request: Missing required field 'containerDefinitions'",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Validate container definitions
	containerDefs := requestData["containerDefinitions"].([]interface{})
	for _, containerDefInterface := range containerDefs {
		containerDef, ok := containerDefInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		// Validate memory if specified
		if memoryInterface, ok := containerDef["memory"]; ok {
			var memory int
			switch v := memoryInterface.(type) {
			case float64:
				memory = int(v)
			case int:
				memory = v
			default:
				errorResponse := map[string]interface{}{
					"__type": "InvalidParameterException",
					"message": "Invalid memory value",
				}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errorResponse)
				return
			}
			
			if memory <= 0 {
				errorResponse := map[string]interface{}{
					"__type": "InvalidParameterException",
					"message": "Container memory must be greater than 0",
				}
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errorResponse)
				return
			}
		}
	}

	// Manually call storage service with converted data
	storageTaskDef, err := s.convertMapToStorageTaskDefinition(requestData)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	registered, err := s.storage.TaskDefinitionStore().Register(ctx, storageTaskDef)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Convert back to API format
	apiTaskDef, err := s.convertFromStorageTaskDefinitionToMap(registered)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	responseMap := map[string]interface{}{
		"taskDefinition": apiTaskDef,
	}
	
	if tags, ok := requestData["tags"]; ok && tags != nil {
		responseMap["tags"] = tags
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}


// handleECSDescribeTaskDefinition handles the DescribeTaskDefinition operation
func (s *Server) handleECSDescribeTaskDefinition(w http.ResponseWriter, body []byte) {
	
	// Parse body as a generic map
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			errorResponse := map[string]interface{}{
				"__type": "InvalidParameterException",
				"message": "Invalid request format",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}


	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}

	ctx := context.Background()

	// Extract task definition ARN or family:revision from request
	taskDefArn := ""
	if td, ok := requestData["taskDefinition"].(string); ok {
		taskDefArn = td
	}

	if taskDefArn == "" {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "taskDefinition parameter is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Parse the ARN to extract family and revision
	family, revision, err := s.parseTaskDefinitionIdentifier(taskDefArn)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	var taskDef *storage.TaskDefinition
	if revision > 0 {
		// Get specific revision
		taskDef, err = s.storage.TaskDefinitionStore().Get(ctx, family, revision)
	} else {
		// Get latest revision
		taskDef, err = s.storage.TaskDefinitionStore().GetLatest(ctx, family)
	}

	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "ClientException",
			"message": "TaskDefinition not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Convert back to API format
	apiTaskDef, err := s.convertFromStorageTaskDefinitionToMap(taskDef)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	responseMap := map[string]interface{}{
		"taskDefinition": apiTaskDef,
	}

	// Add tags if requested
	if include, ok := requestData["include"].([]interface{}); ok {
		for _, item := range include {
			if includeStr, ok := item.(string); ok && includeStr == "TAGS" {
				if taskDef.Tags != "" {
					var tags interface{}
					if err := json.Unmarshal([]byte(taskDef.Tags), &tags); err == nil {
						responseMap["tags"] = tags
					}
				}
				break
			}
		}
	}


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

// handleECSDeregisterTaskDefinition handles the DeregisterTaskDefinition operation
func (s *Server) handleECSDeregisterTaskDefinition(w http.ResponseWriter, body []byte) {
	
	// Parse body as a generic map
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			errorResponse := map[string]interface{}{
				"__type": "InvalidParameterException",
				"message": "Invalid request format",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}


	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
		return
	}

	ctx := context.Background()

	// Extract task definition ARN or family:revision from request
	taskDefArn := ""
	if td, ok := requestData["taskDefinition"].(string); ok {
		taskDefArn = td
	}

	if taskDefArn == "" {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "taskDefinition parameter is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Parse the ARN to extract family and revision
	family, revision, err := s.parseTaskDefinitionIdentifier(taskDefArn)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Get the task definition before deregistering it
	var taskDef *storage.TaskDefinition
	if revision > 0 {
		taskDef, err = s.storage.TaskDefinitionStore().Get(ctx, family, revision)
	} else {
		taskDef, err = s.storage.TaskDefinitionStore().GetLatest(ctx, family)
	}

	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "ClientException",
			"message": "TaskDefinition not found",
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Only call deregister, don't create new objects
	err = s.storage.TaskDefinitionStore().Deregister(ctx, taskDef.Family, taskDef.Revision)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Convert to API format using the original task definition and update status manually
	apiTaskDef, err := s.convertFromStorageTaskDefinitionToMap(taskDef)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Manually update the status and deregistration time
	apiTaskDef["status"] = "INACTIVE"
	apiTaskDef["deregisteredAt"] = time.Now().Format(time.RFC3339)

	responseMap := map[string]interface{}{
		"taskDefinition": apiTaskDef,
	}


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

// handleECSDeleteTaskDefinitions handles the DeleteTaskDefinitions operation
func (s *Server) handleECSDeleteTaskDefinitions(w http.ResponseWriter, body []byte) {
	
	// Parse body as a generic map
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			errorResponse := map[string]interface{}{
				"__type": "InvalidParameterException",
				"message": "Invalid request format",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}


	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"taskDefinitions": []interface{}{},
			"failures":        []interface{}{},
		})
		return
	}

	ctx := context.Background()

	// Extract task definition ARNs from request
	taskDefinitions := []string{}
	if td, ok := requestData["taskDefinitions"].([]interface{}); ok {
		for _, item := range td {
			if tdStr, ok := item.(string); ok {
				taskDefinitions = append(taskDefinitions, tdStr)
			}
		}
	}

	if len(taskDefinitions) == 0 {
		errorResponse := map[string]interface{}{
			"__type": "InvalidParameterException",
			"message": "taskDefinitions parameter is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	var deletedTaskDefs []interface{}
	var failures []interface{}

	for _, taskDefArn := range taskDefinitions {
		// Parse the ARN to extract family and revision
		family, revision, err := s.parseTaskDefinitionIdentifier(taskDefArn)
		if err != nil {
			failures = append(failures, map[string]interface{}{
				"arn":    taskDefArn,
				"reason": "INVALID_ARN",
				"detail": err.Error(),
			})
			continue
		}

		// Get the task definition before deleting
		var taskDef *storage.TaskDefinition
		if revision > 0 {
			taskDef, err = s.storage.TaskDefinitionStore().Get(ctx, family, revision)
		} else {
			taskDef, err = s.storage.TaskDefinitionStore().GetLatest(ctx, family)
		}

		if err != nil {
			failures = append(failures, map[string]interface{}{
				"arn":    taskDefArn,
				"reason": "TASK_DEFINITION_NOT_FOUND",
				"detail": err.Error(),
			})
			continue
		}

		// Convert to API format before deletion
		apiTaskDef, err := s.convertFromStorageTaskDefinitionToMap(taskDef)
		if err != nil {
			failures = append(failures, map[string]interface{}{
				"arn":    taskDefArn,
				"reason": "INTERNAL_ERROR",
				"detail": err.Error(),
			})
			continue
		}

		// Note: This is a soft delete operation - we mark as INACTIVE
		// In a real implementation, you might want to implement actual deletion
		err = s.storage.TaskDefinitionStore().Deregister(ctx, taskDef.Family, taskDef.Revision)
		if err != nil {
			failures = append(failures, map[string]interface{}{
				"arn":    taskDefArn,
				"reason": "DELETION_FAILED",
				"detail": err.Error(),
			})
			continue
		}

		// Mark as deleted status and add deregistered timestamp
		apiTaskDef["status"] = "DELETE_IN_PROGRESS"
		apiTaskDef["deregisteredAt"] = time.Now().Format(time.RFC3339)
		deletedTaskDefs = append(deletedTaskDefs, apiTaskDef)
	}

	responseMap := map[string]interface{}{
		"taskDefinitions": deletedTaskDefs,
		"failures":        failures,
	}


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

// handleECSListTaskDefinitions handles the ListTaskDefinitions operation
func (s *Server) handleECSListTaskDefinitions(w http.ResponseWriter, body []byte) {
	
	// Parse body as a generic map
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			errorResponse := map[string]interface{}{
				"__type": "InvalidParameterException",
				"message": "Invalid request format",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}


	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"taskDefinitionArns": []string{},
		})
		return
	}

	ctx := context.Background()

	// Extract parameters from request
	familyPrefix := ""
	if fp, ok := requestData["familyPrefix"].(string); ok {
		familyPrefix = fp
	}

	status := ""
	if st, ok := requestData["status"].(string); ok {
		status = st
	}

	limit := 0
	if mr, ok := requestData["maxResults"].(float64); ok {
		limit = int(mr)
	}

	nextToken := ""
	if nt, ok := requestData["nextToken"].(string); ok {
		nextToken = nt
	}

	families, newNextToken, err := s.storage.TaskDefinitionStore().ListFamilies(ctx, familyPrefix, status, limit, nextToken)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	var arns []string
	for _, family := range families {
		revisions, _, err := s.storage.TaskDefinitionStore().ListRevisions(ctx, family.Family, status, 0, "")
		if err != nil {
			continue
		}
		for _, rev := range revisions {
			arns = append(arns, rev.ARN)
		}
	}

	responseMap := map[string]interface{}{
		"taskDefinitionArns": arns,
	}
	
	if newNextToken != "" {
		responseMap["nextToken"] = newNextToken
	}


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

// Conversion helper functions

// generateUniqueID generates a unique ID for task definitions
func generateUniqueID() string {
	// Use UUID-like approach for better uniqueness
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp with microseconds if random fails
		return fmt.Sprintf("td-%d-%d", time.Now().UnixNano(), time.Now().Nanosecond())
	}
	// Format as UUID-like string
	return fmt.Sprintf("td-%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// convertMapToStorageTaskDefinition converts a map to storage.TaskDefinition
func (s *Server) convertMapToStorageTaskDefinition(requestData map[string]interface{}) (*storage.TaskDefinition, error) {
	// Get family
	family := ""
	if f, ok := requestData["family"].(string); ok {
		family = f
	}

	// Marshal container definitions
	containerDefs, err := json.Marshal(requestData["containerDefinitions"])
	if err != nil {
		return nil, fmt.Errorf("failed to marshal container definitions: %w", err)
	}

	// Optional fields
	var volumesJSON, tagsJSON, requiresCompatibilitiesJSON, placementConstraintsJSON string
	var inferenceAcceleratorsJSON, proxyConfigJSON, runtimePlatformJSON string

	if volumes := requestData["volumes"]; volumes != nil {
		v, _ := json.Marshal(volumes)
		volumesJSON = string(v)
	}

	if tags := requestData["tags"]; tags != nil {
		t, _ := json.Marshal(tags)
		tagsJSON = string(t)
	}

	if reqCompat := requestData["requiresCompatibilities"]; reqCompat != nil {
		rc, _ := json.Marshal(reqCompat)
		requiresCompatibilitiesJSON = string(rc)
	}

	if placementConstraints := requestData["placementConstraints"]; placementConstraints != nil {
		pc, _ := json.Marshal(placementConstraints)
		placementConstraintsJSON = string(pc)
	}

	if inferenceAccelerators := requestData["inferenceAccelerators"]; inferenceAccelerators != nil {
		ia, _ := json.Marshal(inferenceAccelerators)
		inferenceAcceleratorsJSON = string(ia)
	}

	if proxyConfig := requestData["proxyConfiguration"]; proxyConfig != nil {
		p, _ := json.Marshal(proxyConfig)
		proxyConfigJSON = string(p)
	}

	if runtimePlatform := requestData["runtimePlatform"]; runtimePlatform != nil {
		rp, _ := json.Marshal(runtimePlatform)
		runtimePlatformJSON = string(rp)
	}

	// Get string values
	taskRoleArn, _ := requestData["taskRoleArn"].(string)
	executionRoleArn, _ := requestData["executionRoleArn"].(string)
	networkMode, _ := requestData["networkMode"].(string)
	cpu, _ := requestData["cpu"].(string)
	memory, _ := requestData["memory"].(string)
	pidMode, _ := requestData["pidMode"].(string)
	ipcMode, _ := requestData["ipcMode"].(string)

	if networkMode == "" {
		networkMode = "bridge"
	}

	return &storage.TaskDefinition{
		ID:                       generateUniqueID(),
		Family:                   family,
		TaskRoleARN:              taskRoleArn,
		ExecutionRoleARN:         executionRoleArn,
		NetworkMode:              networkMode,
		ContainerDefinitions:     string(containerDefs),
		Volumes:                  volumesJSON,
		PlacementConstraints:     placementConstraintsJSON,
		RequiresCompatibilities:  requiresCompatibilitiesJSON,
		CPU:                      cpu,
		Memory:                   memory,
		Tags:                     tagsJSON,
		PidMode:                  pidMode,
		IpcMode:                  ipcMode,
		ProxyConfiguration:       proxyConfigJSON,
		InferenceAccelerators:    inferenceAcceleratorsJSON,
		RuntimePlatform:          runtimePlatformJSON,
		Region:                   "us-east-1",
		AccountID:                "123456789012",
	}, nil
}

// convertFromStorageTaskDefinitionToMap converts storage.TaskDefinition to a map
func (s *Server) convertFromStorageTaskDefinitionToMap(stored *storage.TaskDefinition) (map[string]interface{}, error) {
	// Parse JSON fields back to interface{}
	var containerDefs interface{}
	if err := json.Unmarshal([]byte(stored.ContainerDefinitions), &containerDefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal container definitions: %w", err)
	}

	taskDef := map[string]interface{}{
		"taskDefinitionArn":      stored.ARN,
		"family":                 stored.Family,
		"revision":               stored.Revision,
		"status":                 stored.Status,
		"networkMode":            stored.NetworkMode,
		"containerDefinitions":   containerDefs,
		"registeredAt":           stored.RegisteredAt.Format(time.RFC3339),
		"registeredBy":           "kecs",
	}

	// Add optional fields if they exist
	if stored.TaskRoleARN != "" {
		taskDef["taskRoleArn"] = stored.TaskRoleARN
	}
	if stored.ExecutionRoleARN != "" {
		taskDef["executionRoleArn"] = stored.ExecutionRoleARN
	}
	if stored.CPU != "" {
		taskDef["cpu"] = stored.CPU
	}
	if stored.Memory != "" {
		taskDef["memory"] = stored.Memory
	}

	// Parse and add JSON fields
	if stored.Volumes != "" {
		var volumes interface{}
		if err := json.Unmarshal([]byte(stored.Volumes), &volumes); err == nil {
			taskDef["volumes"] = volumes
		}
	}

	if stored.RequiresCompatibilities != "" {
		var reqCompat interface{}
		if err := json.Unmarshal([]byte(stored.RequiresCompatibilities), &reqCompat); err == nil {
			taskDef["requiresCompatibilities"] = reqCompat
			taskDef["compatibilities"] = reqCompat
		}
	}

	if stored.PlacementConstraints != "" {
		var placementConstraints interface{}
		if err := json.Unmarshal([]byte(stored.PlacementConstraints), &placementConstraints); err == nil {
			taskDef["placementConstraints"] = placementConstraints
		}
	}

	if stored.PidMode != "" {
		taskDef["pidMode"] = stored.PidMode
	}
	if stored.IpcMode != "" {
		taskDef["ipcMode"] = stored.IpcMode
	}

	if stored.DeregisteredAt != nil {
		taskDef["deregisteredAt"] = stored.DeregisteredAt.Format(time.RFC3339)
	}

	return taskDef, nil
}

// ListTaskDefinitionFamiliesRequest represents the request to list task definition families
type ListTaskDefinitionFamiliesRequest struct {
	FamilyPrefix string `json:"familyPrefix,omitempty"`
	Status       string `json:"status,omitempty"`
	MaxResults   int    `json:"maxResults,omitempty"`
	NextToken    string `json:"nextToken,omitempty"`
}

// ListTaskDefinitionFamiliesResponse represents the response from listing task definition families
type ListTaskDefinitionFamiliesResponse struct {
	Families  []string `json:"families"`
	NextToken string   `json:"nextToken,omitempty"`
}

// handleECSListTaskDefinitionFamilies handles the ListTaskDefinitionFamilies operation
func (s *Server) handleECSListTaskDefinitionFamilies(w http.ResponseWriter, body []byte) {
	
	// Parse body as a generic map
	var requestData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &requestData); err != nil {
			errorResponse := map[string]interface{}{
				"__type": "InvalidParameterException",
				"message": "Invalid request format",
			}
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errorResponse)
			return
		}
	}


	if s.storage == nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"families": []string{},
		})
		return
	}

	ctx := context.Background()

	// Extract parameters from request
	familyPrefix := ""
	if fp, ok := requestData["familyPrefix"].(string); ok {
		familyPrefix = fp
	}

	status := ""
	if st, ok := requestData["status"].(string); ok {
		status = st
	}

	limit := 0
	if mr, ok := requestData["maxResults"].(float64); ok {
		limit = int(mr)
	}

	nextToken := ""
	if nt, ok := requestData["nextToken"].(string); ok {
		nextToken = nt
	}

	families, newNextToken, err := s.storage.TaskDefinitionStore().ListFamilies(ctx, familyPrefix, status, limit, nextToken)
	if err != nil {
		errorResponse := map[string]interface{}{
			"__type": "InternalServerErrorException",
			"message": err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// Extract family names
	familyNames := make([]string, 0, len(families))
	for _, family := range families {
		familyNames = append(familyNames, family.Family)
	}

	responseMap := map[string]interface{}{
		"families": familyNames,
	}
	
	if newNextToken != "" {
		responseMap["nextToken"] = newNextToken
	}


	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responseMap)
}

// parseTaskDefinitionIdentifier parses a task definition identifier (ARN, family, or family:revision)
// and returns the family name and revision (0 for latest)
func (s *Server) parseTaskDefinitionIdentifier(identifier string) (family string, revision int, err error) {
	// Handle different formats:
	// 1. ARN: arn:aws:ecs:region:account:task-definition/family:revision
	// 2. family:revision
	// 3. family (returns revision 0 for latest)
	
	if strings.HasPrefix(identifier, "arn:aws:ecs:") {
		// Parse ARN format
		parts := strings.Split(identifier, "/")
		if len(parts) < 2 {
			return "", 0, fmt.Errorf("invalid task definition ARN format")
		}
		familyRevision := parts[len(parts)-1]
		return s.parseFamilyRevision(familyRevision)
	}
	
	// Parse family or family:revision format
	return s.parseFamilyRevision(identifier)
}

// parseFamilyRevision parses "family:revision" or "family" format
func (s *Server) parseFamilyRevision(familyRevision string) (family string, revision int, err error) {
	parts := strings.Split(familyRevision, ":")
	if len(parts) == 1 {
		// Just family name, return revision 0 for latest
		return parts[0], 0, nil
	} else if len(parts) == 2 {
		// family:revision format
		family = parts[0]
		var rev int
		n, err := fmt.Sscanf(parts[1], "%d", &rev)
		if err != nil || n != 1 || rev <= 0 {
			return "", 0, fmt.Errorf("invalid revision number: %s", parts[1])
		}
		return family, rev, nil
	}
	
	return "", 0, fmt.Errorf("invalid task definition identifier format: %s", familyRevision)
}

