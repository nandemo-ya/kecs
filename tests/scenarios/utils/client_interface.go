package utils

// ClientMode defines the mode of ECS client to use
type ClientMode string

const (
	// CurlMode uses curl commands for API calls
	CurlMode ClientMode = "curl"
	// AWSCLIMode uses AWS CLI for API calls
	AWSCLIMode ClientMode = "awscli"
)

// ECSClientInterface defines the interface for ECS operations
type ECSClientInterface interface {
	CreateCluster(name string) error
	DescribeCluster(name string) (*Cluster, error)
	ListClusters() ([]string, error)
	DeleteCluster(name string) error
	RegisterTaskDefinition(family string, definition string) (*TaskDefinition, error)
	DescribeTaskDefinition(taskDefArn string) (*TaskDefinition, error)
	ListTaskDefinitions() ([]string, error)
	DeregisterTaskDefinition(taskDefArn string) error
	CreateService(clusterName, serviceName, taskDef string, desiredCount int) error
	DescribeService(clusterName, serviceName string) (*Service, error)
	ListServices(clusterName string) ([]string, error)
	UpdateService(clusterName, serviceName string, desiredCount *int, taskDef string) error
	DeleteService(clusterName, serviceName string) error
	RunTask(clusterName, taskDefArn string, count int) (*RunTaskResponse, error)
	DescribeTasks(clusterName string, taskArns []string) ([]Task, error)
	ListTasks(clusterName string, serviceName string) ([]string, error)
	StopTask(clusterName, taskArn, reason string) error
	TagResource(resourceArn string, tags map[string]string) error
	UntagResource(resourceArn string, tagKeys []string) error
	ListTagsForResource(resourceArn string) (map[string]string, error)
	PutAttributes(clusterName string, attributes []Attribute) error
	ListAttributes(clusterName, targetType string) ([]Attribute, error)
	DeleteAttributes(clusterName string, attributes []Attribute) error
	GetLocalStackStatus(clusterName string) (string, error)
}

