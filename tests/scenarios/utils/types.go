package utils

// Cluster represents an ECS cluster
type Cluster struct {
	ClusterArn                        string                   `json:"clusterArn"`
	ClusterName                       string                   `json:"clusterName"`
	Status                            string                   `json:"status"`
	RegisteredContainerInstancesCount int                      `json:"registeredContainerInstancesCount"`
	RunningTasksCount                 int                      `json:"runningTasksCount"`
	PendingTasksCount                 int                      `json:"pendingTasksCount"`
	ActiveServicesCount               int                      `json:"activeServicesCount"`
	Settings                          []map[string]string      `json:"settings,omitempty"`
	Configuration                     map[string]interface{}   `json:"configuration,omitempty"`
	CapacityProviders                 []string                 `json:"capacityProviders,omitempty"`
	DefaultCapacityProviderStrategy   []map[string]interface{} `json:"defaultCapacityProviderStrategy,omitempty"`
	Tags                              []map[string]string      `json:"tags,omitempty"`
}

// TaskDefinition represents an ECS task definition
type TaskDefinition struct {
	TaskDefinitionArn string                   `json:"taskDefinitionArn"`
	Family            string                   `json:"family"`
	Revision          int                      `json:"revision"`
	Status            string                   `json:"status"`
	ContainerDefs     []ContainerDefinition    `json:"containerDefinitions,omitempty"`
	Volumes           []Volume                 `json:"volumes,omitempty"`
	Cpu               string                   `json:"cpu,omitempty"`
	Memory            string                   `json:"memory,omitempty"`
	NetworkMode       string                   `json:"networkMode,omitempty"`
	RequiresCompatibilities []string         `json:"requiresCompatibilities,omitempty"`
}

// ContainerDefinition represents a container definition
type ContainerDefinition struct {
	Name      string                    `json:"name"`
	Image     string                    `json:"image"`
	Cpu       int                       `json:"cpu,omitempty"`
	Memory    int                       `json:"memory,omitempty"`
	Essential bool                      `json:"essential"`
	Command   []string                  `json:"command,omitempty"`
	Environment []EnvironmentVariable   `json:"environment,omitempty"`
}

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Volume represents a volume definition
type Volume struct {
	Name string                 `json:"name"`
	Host *HostVolumeProperties `json:"host,omitempty"`
}

// HostVolumeProperties represents host volume properties
type HostVolumeProperties struct {
	SourcePath string `json:"sourcePath,omitempty"`
}

// Service represents an ECS service
type Service struct {
	ServiceArn     string       `json:"serviceArn"`
	ServiceName    string       `json:"serviceName"`
	ClusterArn     string       `json:"clusterArn"`
	TaskDefinition string       `json:"taskDefinition"`
	DesiredCount   int          `json:"desiredCount"`
	RunningCount   int          `json:"runningCount"`
	PendingCount   int          `json:"pendingCount"`
	Status         string       `json:"status"`
	Deployments    []Deployment `json:"deployments,omitempty"`
}

// Deployment represents a service deployment
type Deployment struct {
	Id             string `json:"id"`
	Status         string `json:"status"`
	TaskDefinition string `json:"taskDefinition"`
	DesiredCount   int    `json:"desiredCount"`
	RunningCount   int    `json:"runningCount"`
	PendingCount   int    `json:"pendingCount"`
}

// Task represents an ECS task
type Task struct {
	TaskArn            string `json:"taskArn"`
	ClusterArn         string `json:"clusterArn"`
	TaskDefinitionArn  string `json:"taskDefinitionArn"`
	DesiredStatus      string `json:"desiredStatus"`
	LastStatus         string `json:"lastStatus"`
	LaunchType         string `json:"launchType,omitempty"`
	StartedAt          string `json:"startedAt,omitempty"`
	StoppedAt          string `json:"stoppedAt,omitempty"`
	StoppingAt         string `json:"stoppingAt,omitempty"`
	StoppedReason      string `json:"stoppedReason,omitempty"`
}

// RunTaskResponse represents the response from RunTask
type RunTaskResponse struct {
	Tasks    []Task `json:"tasks"`
	Failures []struct {
		Arn    string `json:"arn"`
		Reason string `json:"reason"`
	} `json:"failures"`
}

// Attribute represents an ECS attribute
type Attribute struct {
	Name       string `json:"name"`
	Value      string `json:"value,omitempty"`
	TargetType string `json:"targetType,omitempty"`
	TargetId   string `json:"targetId,omitempty"`
}