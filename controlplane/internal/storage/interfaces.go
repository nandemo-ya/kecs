package storage

import (
	"context"
	"time"
)

// Storage defines the interface for all storage operations
type Storage interface {
	// Initialize the storage backend
	Initialize(ctx context.Context) error

	// Close the storage connection
	Close() error

	// Cluster operations
	ClusterStore() ClusterStore

	// Task Definition operations
	TaskDefinitionStore() TaskDefinitionStore

	// Service operations
	ServiceStore() ServiceStore

	// Task operations
	TaskStore() TaskStore

	// Account setting operations
	AccountSettingStore() AccountSettingStore

	// TaskSet operations
	TaskSetStore() TaskSetStore

	// Container Instance operations
	ContainerInstanceStore() ContainerInstanceStore

	// Attribute operations
	AttributeStore() AttributeStore

	// ELBv2 operations
	ELBv2Store() ELBv2Store

	// Task log operations
	TaskLogStore() TaskLogStore

	// Transaction support
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error
}

// ClusterStore defines cluster-specific storage operations
type ClusterStore interface {
	// Create a new cluster
	Create(ctx context.Context, cluster *Cluster) error

	// Get a cluster by name
	Get(ctx context.Context, name string) (*Cluster, error)

	// List all clusters
	List(ctx context.Context) ([]*Cluster, error)

	// ListWithPagination retrieves clusters with pagination support
	ListWithPagination(ctx context.Context, limit int, nextToken string) ([]*Cluster, string, error)

	// Update a cluster
	Update(ctx context.Context, cluster *Cluster) error

	// Delete a cluster
	Delete(ctx context.Context, name string) error
}

// Cluster represents an ECS cluster in storage
type Cluster struct {
	// Unique identifier
	ID string `json:"id"`

	// Cluster ARN
	ARN string `json:"arn"`

	// Cluster name
	Name string `json:"name"`

	// Cluster status (ACTIVE, INACTIVE, etc.)
	Status string `json:"status"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Configuration as JSON
	Configuration string `json:"configuration,omitempty"`

	// Settings as JSON
	Settings string `json:"settings,omitempty"`

	// Tags as JSON
	Tags string `json:"tags,omitempty"`

	// K8s cluster name (kecs-<cluster-name>)
	K8sClusterName string `json:"k8sClusterName,omitempty"`

	// Statistics
	RegisteredContainerInstancesCount int `json:"registeredContainerInstancesCount"`
	RunningTasksCount                 int `json:"runningTasksCount"`
	PendingTasksCount                 int `json:"pendingTasksCount"`
	ActiveServicesCount               int `json:"activeServicesCount"`

	// Capacity providers as JSON
	CapacityProviders string `json:"capacityProviders,omitempty"`

	// Default capacity provider strategy as JSON
	DefaultCapacityProviderStrategy string `json:"defaultCapacityProviderStrategy,omitempty"`

	// LocalStack deployment state
	LocalStackState string `json:"localStackState,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TaskDefinitionStore defines task definition-specific storage operations
type TaskDefinitionStore interface {
	// Register a new task definition (creates a new revision)
	Register(ctx context.Context, taskDef *TaskDefinition) (*TaskDefinition, error)

	// Get a specific task definition revision
	Get(ctx context.Context, family string, revision int) (*TaskDefinition, error)

	// Get the latest revision of a task definition family
	GetLatest(ctx context.Context, family string) (*TaskDefinition, error)

	// List task definition families with pagination
	ListFamilies(ctx context.Context, familyPrefix string, status string, limit int, nextToken string) ([]*TaskDefinitionFamily, string, error)

	// List revisions of a specific task definition family
	ListRevisions(ctx context.Context, family string, status string, limit int, nextToken string) ([]*TaskDefinitionRevision, string, error)

	// Deregister a task definition revision
	Deregister(ctx context.Context, family string, revision int) error

	// Get task definition by ARN
	GetByARN(ctx context.Context, arn string) (*TaskDefinition, error)
}

// TaskDefinition represents a task definition with its full configuration
type TaskDefinition struct {
	// Unique identifier
	ID string `json:"id"`

	// Task definition ARN
	ARN string `json:"arn"`

	// Task definition family
	Family string `json:"family"`

	// Task definition revision
	Revision int `json:"revision"`

	// Task role ARN
	TaskRoleARN string `json:"taskRoleArn,omitempty"`

	// Execution role ARN
	ExecutionRoleARN string `json:"executionRoleArn,omitempty"`

	// Network mode (bridge, host, awsvpc, none)
	NetworkMode string `json:"networkMode"`

	// Container definitions as JSON
	ContainerDefinitions string `json:"containerDefinitions"`

	// Volumes as JSON
	Volumes string `json:"volumes,omitempty"`

	// Placement constraints as JSON
	PlacementConstraints string `json:"placementConstraints,omitempty"`

	// Required compatibility (EC2, FARGATE, etc.)
	RequiresCompatibilities string `json:"requiresCompatibilities,omitempty"`

	// CPU value (in CPU units or vCPU)
	CPU string `json:"cpu,omitempty"`

	// Memory value (in MiB)
	Memory string `json:"memory,omitempty"`

	// Tags as JSON
	Tags string `json:"tags,omitempty"`

	// PID mode
	PidMode string `json:"pidMode,omitempty"`

	// IPC mode
	IpcMode string `json:"ipcMode,omitempty"`

	// Proxy configuration as JSON
	ProxyConfiguration string `json:"proxyConfiguration,omitempty"`

	// Inference accelerators as JSON
	InferenceAccelerators string `json:"inferenceAccelerators,omitempty"`

	// Runtime platform as JSON
	RuntimePlatform string `json:"runtimePlatform,omitempty"`

	// Status (ACTIVE, INACTIVE)
	Status string `json:"status"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Timestamps
	RegisteredAt   time.Time  `json:"registeredAt"`
	DeregisteredAt *time.Time `json:"deregisteredAt,omitempty"`
}

// TaskDefinitionFamily represents a task definition family summary
type TaskDefinitionFamily struct {
	Family          string `json:"family"`
	LatestRevision  int    `json:"latestRevision"`
	ActiveRevisions int    `json:"activeRevisions"`
}

// TaskDefinitionRevision represents a task definition revision summary
type TaskDefinitionRevision struct {
	ARN          string    `json:"arn"`
	Family       string    `json:"family"`
	Revision     int       `json:"revision"`
	Status       string    `json:"status"`
	RegisteredAt time.Time `json:"registeredAt"`
}

// ServiceStore defines service-specific storage operations
type ServiceStore interface {
	// Create a new service
	Create(ctx context.Context, service *Service) error

	// Get a service by cluster and service name
	Get(ctx context.Context, cluster, serviceName string) (*Service, error)

	// List services with filtering
	List(ctx context.Context, cluster string, serviceName string, launchType string, limit int, nextToken string) ([]*Service, string, error)

	// Update a service
	Update(ctx context.Context, service *Service) error

	// Delete a service
	Delete(ctx context.Context, cluster, serviceName string) error

	// Get service by ARN
	GetByARN(ctx context.Context, arn string) (*Service, error)

	// DeleteMarkedForDeletion deletes services marked for deletion before the specified time
	DeleteMarkedForDeletion(ctx context.Context, clusterARN string, before time.Time) (int, error)
}

// Service represents an ECS service in storage
type Service struct {
	// Unique identifier
	ID string `json:"id"`

	// Service ARN
	ARN string `json:"arn"`

	// Service name
	ServiceName string `json:"serviceName"`

	// Cluster ARN
	ClusterARN string `json:"clusterArn"`

	// Task definition ARN
	TaskDefinitionARN string `json:"taskDefinitionArn"`

	// Desired count
	DesiredCount int `json:"desiredCount"`

	// Running count
	RunningCount int `json:"runningCount"`

	// Pending count
	PendingCount int `json:"pendingCount"`

	// Launch type (EC2, FARGATE, EXTERNAL)
	LaunchType string `json:"launchType"`

	// Platform version
	PlatformVersion string `json:"platformVersion,omitempty"`

	// Status
	Status string `json:"status"`

	// Role ARN
	RoleARN string `json:"roleArn,omitempty"`

	// Load balancers as JSON
	LoadBalancers string `json:"loadBalancers,omitempty"`

	// Service registries as JSON
	ServiceRegistries string `json:"serviceRegistries,omitempty"`

	// Network configuration as JSON
	NetworkConfiguration string `json:"networkConfiguration,omitempty"`

	// Deployment configuration as JSON
	DeploymentConfiguration string `json:"deploymentConfiguration,omitempty"`

	// Deployment controller as JSON (type: ECS|CODE_DEPLOY|EXTERNAL)
	DeploymentController string `json:"deploymentController,omitempty"`

	// Placement constraints as JSON
	PlacementConstraints string `json:"placementConstraints,omitempty"`

	// Placement strategy as JSON
	PlacementStrategy string `json:"placementStrategy,omitempty"`

	// Capacity provider strategy as JSON
	CapacityProviderStrategy string `json:"capacityProviderStrategy,omitempty"`

	// Tags as JSON
	Tags string `json:"tags,omitempty"`

	// Scheduling strategy (REPLICA, DAEMON)
	SchedulingStrategy string `json:"schedulingStrategy"`

	// Service connect configuration as JSON
	ServiceConnectConfiguration string `json:"serviceConnectConfiguration,omitempty"`

	// Enable ECS managed tags
	EnableECSManagedTags bool `json:"enableECSManagedTags"`

	// Propagate tags (TASK_DEFINITION, SERVICE, NONE)
	PropagateTags string `json:"propagateTags,omitempty"`

	// Enable execute command
	EnableExecuteCommand bool `json:"enableExecuteCommand"`

	// Health check grace period
	HealthCheckGracePeriodSeconds int `json:"healthCheckGracePeriodSeconds,omitempty"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Kubernetes Deployment information (for tracking)
	DeploymentName string `json:"deploymentName,omitempty"`
	Namespace      string `json:"namespace,omitempty"`

	// NodePort information (for port-forward support)
	NodePorts      string `json:"nodePorts,omitempty"`      // JSON array of allocated NodePort numbers
	HasNodePort    bool   `json:"hasNodePort"`              // Whether service uses NodePort
	AssignPublicIp bool   `json:"assignPublicIp,omitempty"` // Whether assignPublicIp was enabled

	// Service registry metadata for service discovery
	ServiceRegistryMetadata map[string]string `json:"serviceRegistryMetadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TaskStore defines task-specific storage operations
type TaskStore interface {
	// Create a new task
	Create(ctx context.Context, task *Task) error

	// Get a task by cluster and task ID/ARN
	Get(ctx context.Context, cluster, taskID string) (*Task, error)

	// List tasks with filtering
	List(ctx context.Context, cluster string, filters TaskFilters) ([]*Task, error)

	// Update a task (status, etc.)
	Update(ctx context.Context, task *Task) error

	// Delete a task
	Delete(ctx context.Context, cluster, taskID string) error

	// Get tasks by ARNs
	GetByARNs(ctx context.Context, arns []string) ([]*Task, error)

	// CreateOrUpdate creates a new task or updates if it already exists
	CreateOrUpdate(ctx context.Context, task *Task) error

	// DeleteOlderThan deletes tasks older than the specified time with the given status
	DeleteOlderThan(ctx context.Context, clusterARN string, before time.Time, status string) (int, error)
}

// TaskFilters defines filters for listing tasks
type TaskFilters struct {
	// Filter by service name
	ServiceName string

	// Filter by task definition family
	Family string

	// Filter by container instance
	ContainerInstance string

	// Filter by launch type
	LaunchType string

	// Filter by status
	DesiredStatus string

	// Filter by started by
	StartedBy string

	// Maximum results
	MaxResults int

	// Next token for pagination
	NextToken string
}

// Task represents an ECS task in storage
type Task struct {
	// Unique identifier
	ID string `json:"id"`

	// Task ARN
	ARN string `json:"arn"`

	// Cluster ARN
	ClusterARN string `json:"clusterArn"`

	// Task definition ARN
	TaskDefinitionARN string `json:"taskDefinitionArn"`

	// Container instance ARN (for EC2 launch type)
	ContainerInstanceARN string `json:"containerInstanceArn,omitempty"`

	// Overrides as JSON
	Overrides string `json:"overrides,omitempty"`

	// Last status
	LastStatus string `json:"lastStatus"`

	// Desired status
	DesiredStatus string `json:"desiredStatus"`

	// CPU
	CPU string `json:"cpu,omitempty"`

	// Memory
	Memory string `json:"memory,omitempty"`

	// Containers as JSON (status information)
	Containers string `json:"containers"`

	// Started by (service name, user, etc.)
	StartedBy string `json:"startedBy,omitempty"`

	// Version
	Version int64 `json:"version"`

	// Stop code
	StopCode string `json:"stopCode,omitempty"`

	// Stop reason
	StoppedReason string `json:"stoppedReason,omitempty"`

	// Stopping at
	StoppingAt *time.Time `json:"stoppingAt,omitempty"`

	// Stopped at
	StoppedAt *time.Time `json:"stoppedAt,omitempty"`

	// Connectivity
	Connectivity string `json:"connectivity,omitempty"`

	// Connectivity at
	ConnectivityAt *time.Time `json:"connectivityAt,omitempty"`

	// Pull started at
	PullStartedAt *time.Time `json:"pullStartedAt,omitempty"`

	// Pull stopped at
	PullStoppedAt *time.Time `json:"pullStoppedAt,omitempty"`

	// Execution stopped at
	ExecutionStoppedAt *time.Time `json:"executionStoppedAt,omitempty"`

	// Created at
	CreatedAt time.Time `json:"createdAt"`

	// Started at
	StartedAt *time.Time `json:"startedAt,omitempty"`

	// Launch type
	LaunchType string `json:"launchType"`

	// Platform version
	PlatformVersion string `json:"platformVersion,omitempty"`

	// Platform family
	PlatformFamily string `json:"platformFamily,omitempty"`

	// Group
	Group string `json:"group,omitempty"`

	// Attachments as JSON
	Attachments string `json:"attachments,omitempty"`

	// Health status
	HealthStatus string `json:"healthStatus,omitempty"`

	// Tags as JSON
	Tags string `json:"tags,omitempty"`

	// Attributes as JSON
	Attributes string `json:"attributes,omitempty"`

	// Enable execute command
	EnableExecuteCommand bool `json:"enableExecuteCommand"`

	// Capacity provider name
	CapacityProviderName string `json:"capacityProviderName,omitempty"`

	// Ephemeral storage as JSON
	EphemeralStorage string `json:"ephemeralStorage,omitempty"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Kubernetes Pod name
	PodName string `json:"podName,omitempty"`

	// Kubernetes namespace
	Namespace string `json:"namespace,omitempty"`

	// Service registries (JSON)
	ServiceRegistries string `json:"serviceRegistries,omitempty"`
}

// AccountSettingStore defines account setting-specific storage operations
type AccountSettingStore interface {
	// Create or update an account setting
	Upsert(ctx context.Context, setting *AccountSetting) error

	// Get an account setting by principal ARN and name
	Get(ctx context.Context, principalARN, name string) (*AccountSetting, error)

	// Get default account setting by name
	GetDefault(ctx context.Context, name string) (*AccountSetting, error)

	// List account settings with filtering
	List(ctx context.Context, filters AccountSettingFilters) ([]*AccountSetting, string, error)

	// Delete an account setting
	Delete(ctx context.Context, principalARN, name string) error

	// Set default account setting
	SetDefault(ctx context.Context, name, value string) error
}

// AccountSetting represents an account setting in storage
type AccountSetting struct {
	// Unique identifier
	ID string `json:"id"`

	// Setting name
	Name string `json:"name"`

	// Setting value
	Value string `json:"value"`

	// Principal ARN (user/role ARN or "default" for default settings)
	PrincipalARN string `json:"principalArn"`

	// Is this a default setting
	IsDefault bool `json:"isDefault"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AccountSettingFilters defines filters for listing account settings
type AccountSettingFilters struct {
	// Filter by setting name
	Name string

	// Filter by value
	Value string

	// Filter by principal ARN
	PrincipalARN string

	// Include effective settings (defaults + overrides)
	EffectiveSettings bool

	// Maximum results
	MaxResults int

	// Next token for pagination
	NextToken string
}

// TaskSetStore defines task set-specific storage operations
type TaskSetStore interface {
	// Create a new task set
	Create(ctx context.Context, taskSet *TaskSet) error

	// Get a task set by service ARN and task set ID
	Get(ctx context.Context, serviceARN, taskSetID string) (*TaskSet, error)

	// List task sets for a service
	List(ctx context.Context, serviceARN string, taskSetIDs []string) ([]*TaskSet, error)

	// Update a task set
	Update(ctx context.Context, taskSet *TaskSet) error

	// Delete a task set
	Delete(ctx context.Context, serviceARN, taskSetID string) error

	// Get task set by ARN
	GetByARN(ctx context.Context, arn string) (*TaskSet, error)

	// Update primary task set
	UpdatePrimary(ctx context.Context, serviceARN, taskSetID string) error

	// DeleteOrphaned deletes task sets that no longer have an associated service
	DeleteOrphaned(ctx context.Context, clusterARN string) (int, error)
}

// TaskSet represents an ECS task set in storage
type TaskSet struct {
	// Unique identifier
	ID string `json:"id"`

	// Task set ARN
	ARN string `json:"arn"`

	// Service ARN
	ServiceARN string `json:"serviceArn"`

	// Cluster ARN
	ClusterARN string `json:"clusterArn"`

	// External ID
	ExternalID string `json:"externalId,omitempty"`

	// Task definition ARN
	TaskDefinition string `json:"taskDefinition"`

	// Launch type
	LaunchType string `json:"launchType,omitempty"`

	// Platform version
	PlatformVersion string `json:"platformVersion,omitempty"`

	// Platform family
	PlatformFamily string `json:"platformFamily,omitempty"`

	// Network configuration as JSON
	NetworkConfiguration string `json:"networkConfiguration,omitempty"`

	// Load balancers as JSON
	LoadBalancers string `json:"loadBalancers,omitempty"`

	// Service registries as JSON
	ServiceRegistries string `json:"serviceRegistries,omitempty"`

	// Capacity provider strategy as JSON
	CapacityProviderStrategy string `json:"capacityProviderStrategy,omitempty"`

	// Scale as JSON
	Scale string `json:"scale,omitempty"`

	// Computed desired count
	ComputedDesiredCount int32 `json:"computedDesiredCount"`

	// Pending count
	PendingCount int32 `json:"pendingCount"`

	// Running count
	RunningCount int32 `json:"runningCount"`

	// Status
	Status string `json:"status"`

	// Stability status
	StabilityStatus string `json:"stabilityStatus"`

	// Stability status at
	StabilityStatusAt *time.Time `json:"stabilityStatusAt,omitempty"`

	// Started by
	StartedBy string `json:"startedBy,omitempty"`

	// Tags as JSON
	Tags string `json:"tags,omitempty"`

	// Fargate ephemeral storage as JSON
	FargateEphemeralStorage string `json:"fargateEphemeralStorage,omitempty"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ContainerInstanceStore defines container instance-specific storage operations
type ContainerInstanceStore interface {
	// Register a new container instance
	Register(ctx context.Context, instance *ContainerInstance) error

	// Get a container instance by ARN
	Get(ctx context.Context, arn string) (*ContainerInstance, error)

	// List container instances with filtering and pagination
	ListWithPagination(ctx context.Context, cluster string, filters ContainerInstanceFilters, limit int, nextToken string) ([]*ContainerInstance, string, error)

	// Update a container instance
	Update(ctx context.Context, instance *ContainerInstance) error

	// Deregister a container instance
	Deregister(ctx context.Context, arn string) error

	// Get container instances by ARNs
	GetByARNs(ctx context.Context, arns []string) ([]*ContainerInstance, error)

	// DeleteStale deletes container instances that have been inactive before the specified time
	DeleteStale(ctx context.Context, clusterARN string, before time.Time) (int, error)
}

// ContainerInstance represents an ECS container instance in storage
type ContainerInstance struct {
	// Unique identifier
	ID string `json:"id"`

	// Container instance ARN
	ARN string `json:"arn"`

	// Cluster ARN
	ClusterARN string `json:"clusterArn"`

	// EC2 instance ID
	EC2InstanceID string `json:"ec2InstanceId"`

	// Status (ACTIVE, DRAINING, REGISTERING, DEREGISTERING, REGISTRATION_FAILED)
	Status string `json:"status"`

	// Status reason
	StatusReason string `json:"statusReason,omitempty"`

	// Agent connected
	AgentConnected bool `json:"agentConnected"`

	// Agent update status
	AgentUpdateStatus string `json:"agentUpdateStatus,omitempty"`

	// Running tasks count
	RunningTasksCount int32 `json:"runningTasksCount"`

	// Pending tasks count
	PendingTasksCount int32 `json:"pendingTasksCount"`

	// Version
	Version int64 `json:"version"`

	// Version info as JSON
	VersionInfo string `json:"versionInfo,omitempty"`

	// Registered resources as JSON
	RegisteredResources string `json:"registeredResources,omitempty"`

	// Remaining resources as JSON
	RemainingResources string `json:"remainingResources,omitempty"`

	// Attributes as JSON
	Attributes string `json:"attributes,omitempty"`

	// Attachments as JSON
	Attachments string `json:"attachments,omitempty"`

	// Tags as JSON
	Tags string `json:"tags,omitempty"`

	// Capacity provider name
	CapacityProviderName string `json:"capacityProviderName,omitempty"`

	// Health status
	HealthStatus string `json:"healthStatus,omitempty"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Timestamps
	RegisteredAt   time.Time  `json:"registeredAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	DeregisteredAt *time.Time `json:"deregisteredAt,omitempty"`
}

// ContainerInstanceFilters defines filters for listing container instances
type ContainerInstanceFilters struct {
	// Filter by status
	Status string

	// Filter by instance
	Filter string
}

// AttributeStore defines attribute-specific storage operations
type AttributeStore interface {
	// Put attributes
	Put(ctx context.Context, attributes []*Attribute) error

	// Delete attributes
	Delete(ctx context.Context, cluster string, attributes []*Attribute) error

	// List attributes with pagination
	ListWithPagination(ctx context.Context, targetType, cluster string, limit int, nextToken string) ([]*Attribute, string, error)
}

// Attribute represents an ECS attribute in storage
type Attribute struct {
	// Unique identifier
	ID string `json:"id"`

	// Attribute name
	Name string `json:"name"`

	// Attribute value
	Value string `json:"value,omitempty"`

	// Target type (container-instance)
	TargetType string `json:"targetType"`

	// Target ID (container instance ARN)
	TargetID string `json:"targetId"`

	// Cluster name
	Cluster string `json:"cluster"`

	// Region
	Region string `json:"region"`

	// Account ID
	AccountID string `json:"accountId"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ELBv2Store defines ELBv2-specific storage operations
type ELBv2Store interface {
	// Load Balancer operations
	CreateLoadBalancer(ctx context.Context, lb *ELBv2LoadBalancer) error
	GetLoadBalancer(ctx context.Context, arn string) (*ELBv2LoadBalancer, error)
	GetLoadBalancerByName(ctx context.Context, name string) (*ELBv2LoadBalancer, error)
	ListLoadBalancers(ctx context.Context, region string) ([]*ELBv2LoadBalancer, error)
	UpdateLoadBalancer(ctx context.Context, lb *ELBv2LoadBalancer) error
	DeleteLoadBalancer(ctx context.Context, arn string) error

	// Target Group operations
	CreateTargetGroup(ctx context.Context, tg *ELBv2TargetGroup) error
	GetTargetGroup(ctx context.Context, arn string) (*ELBv2TargetGroup, error)
	GetTargetGroupByName(ctx context.Context, name string) (*ELBv2TargetGroup, error)
	ListTargetGroups(ctx context.Context, region string) ([]*ELBv2TargetGroup, error)
	UpdateTargetGroup(ctx context.Context, tg *ELBv2TargetGroup) error
	DeleteTargetGroup(ctx context.Context, arn string) error

	// Listener operations
	CreateListener(ctx context.Context, listener *ELBv2Listener) error
	GetListener(ctx context.Context, arn string) (*ELBv2Listener, error)
	ListListeners(ctx context.Context, loadBalancerArn string) ([]*ELBv2Listener, error)
	UpdateListener(ctx context.Context, listener *ELBv2Listener) error
	DeleteListener(ctx context.Context, arn string) error

	// Target operations
	RegisterTargets(ctx context.Context, targetGroupArn string, targets []*ELBv2Target) error
	DeregisterTargets(ctx context.Context, targetGroupArn string, targetIDs []string) error
	ListTargets(ctx context.Context, targetGroupArn string) ([]*ELBv2Target, error)
	UpdateTargetHealth(ctx context.Context, targetGroupArn, targetID string, health *ELBv2TargetHealth) error

	// Rule operations
	CreateRule(ctx context.Context, rule *ELBv2Rule) error
	GetRule(ctx context.Context, ruleArn string) (*ELBv2Rule, error)
	ListRules(ctx context.Context, listenerArn string) ([]*ELBv2Rule, error)
	UpdateRule(ctx context.Context, rule *ELBv2Rule) error
	DeleteRule(ctx context.Context, ruleArn string) error
}

// ELBv2LoadBalancer represents a stored load balancer
type ELBv2LoadBalancer struct {
	ARN                   string            `json:"arn"`
	Name                  string            `json:"name"`
	DNSName               string            `json:"dnsName"`
	CanonicalHostedZoneID string            `json:"canonicalHostedZoneId"`
	State                 string            `json:"state"`
	Type                  string            `json:"type"`
	Scheme                string            `json:"scheme"`
	VpcID                 string            `json:"vpcId"`
	Subnets               []string          `json:"subnets"`
	AvailabilityZones     []string          `json:"availabilityZones"`
	SecurityGroups        []string          `json:"securityGroups"`
	IpAddressType         string            `json:"ipAddressType"`
	Tags                  map[string]string `json:"tags"`
	Region                string            `json:"region"`
	AccountID             string            `json:"accountId"`
	CreatedAt             time.Time         `json:"createdAt"`
	UpdatedAt             time.Time         `json:"updatedAt"`
}

// ELBv2TargetGroup represents a stored target group
type ELBv2TargetGroup struct {
	ARN                        string            `json:"arn"`
	Name                       string            `json:"name"`
	Protocol                   string            `json:"protocol"`
	Port                       int32             `json:"port"`
	VpcID                      string            `json:"vpcId"`
	TargetType                 string            `json:"targetType"`
	HealthCheckEnabled         bool              `json:"healthCheckEnabled"`
	HealthCheckProtocol        string            `json:"healthCheckProtocol"`
	HealthCheckPort            string            `json:"healthCheckPort"`
	HealthCheckPath            string            `json:"healthCheckPath"`
	HealthCheckIntervalSeconds int32             `json:"healthCheckIntervalSeconds"`
	HealthCheckTimeoutSeconds  int32             `json:"healthCheckTimeoutSeconds"`
	HealthyThresholdCount      int32             `json:"healthyThresholdCount"`
	UnhealthyThresholdCount    int32             `json:"unhealthyThresholdCount"`
	Matcher                    string            `json:"matcher"`
	LoadBalancerArns           []string          `json:"loadBalancerArns"`
	Tags                       map[string]string `json:"tags"`
	Region                     string            `json:"region"`
	AccountID                  string            `json:"accountId"`
	CreatedAt                  time.Time         `json:"createdAt"`
	UpdatedAt                  time.Time         `json:"updatedAt"`
}

// ELBv2Listener represents a stored listener
type ELBv2Listener struct {
	ARN             string            `json:"arn"`
	LoadBalancerArn string            `json:"loadBalancerArn"`
	Port            int32             `json:"port"`
	Protocol        string            `json:"protocol"`
	DefaultActions  string            `json:"defaultActions"` // JSON encoded actions
	SslPolicy       string            `json:"sslPolicy"`
	Certificates    string            `json:"certificates"` // JSON encoded certificates
	AlpnPolicy      []string          `json:"alpnPolicy"`
	Tags            map[string]string `json:"tags"`
	Region          string            `json:"region"`
	AccountID       string            `json:"accountId"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`
}

// ELBv2Target represents a registered target
type ELBv2Target struct {
	TargetGroupArn    string    `json:"targetGroupArn"`
	ID                string    `json:"id"`
	Port              int32     `json:"port"`
	AvailabilityZone  string    `json:"availabilityZone"`
	HealthState       string    `json:"healthState"`
	HealthReason      string    `json:"healthReason"`
	HealthDescription string    `json:"healthDescription"`
	RegisteredAt      time.Time `json:"registeredAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// ELBv2TargetHealth represents target health information
type ELBv2TargetHealth struct {
	State       string `json:"state"`
	Reason      string `json:"reason"`
	Description string `json:"description"`
}

// ELBv2Rule represents a stored listener rule
type ELBv2Rule struct {
	ARN         string            `json:"arn"`
	ListenerArn string            `json:"listenerArn"`
	Priority    int32             `json:"priority"`
	Conditions  string            `json:"conditions"` // JSON encoded conditions
	Actions     string            `json:"actions"`    // JSON encoded actions
	IsDefault   bool              `json:"isDefault"`
	Tags        map[string]string `json:"tags"`
	Region      string            `json:"region"`
	AccountID   string            `json:"accountId"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}
