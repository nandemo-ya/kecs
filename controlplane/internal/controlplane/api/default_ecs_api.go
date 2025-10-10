package api

import (
	"github.com/nandemo-ya/kecs/controlplane/internal/config"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/s3"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/secretsmanager"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/ssm"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"github.com/nandemo-ya/kecs/controlplane/internal/servicediscovery"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPI provides the default implementation of ECS API operations
type DefaultECSAPI struct {
	config                    *config.Config
	storage                   storage.Storage
	serviceManager            *kubernetes.ServiceManager
	taskManagerInstance       *kubernetes.TaskManager
	taskSetManager            *kubernetes.TaskSetManager
	region                    string
	accountID                 string
	iamIntegration            iam.Integration
	cloudWatchIntegration     cloudwatch.Integration
	ssmIntegration            ssm.Integration
	secretsManagerIntegration secretsmanager.Integration
	s3Integration             s3.Integration
	elbv2Integration          elbv2.Integration
	serviceDiscoveryManager   servicediscovery.Manager
	localStackManager         localstack.Manager
	localStackConfig          *localstack.Config
	localStackUpdateCallback  func(localstack.Manager) // Callback when LocalStack manager is updated
}

// NewDefaultECSAPI creates a new default ECS API implementation with storage
func NewDefaultECSAPI(cfg *config.Config, storage storage.Storage) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		config:    cfg,
		storage:   storage,
		region:    "us-east-1",    // Default region
		accountID: "000000000000", // Default account ID (LocalStack standard)
	}
}

// SetIAMIntegration sets the IAM integration for the ECS API
func (api *DefaultECSAPI) SetIAMIntegration(iamIntegration iam.Integration) {
	api.iamIntegration = iamIntegration
}

// SetCloudWatchIntegration sets the CloudWatch integration for the ECS API
func (api *DefaultECSAPI) SetCloudWatchIntegration(cloudWatchIntegration cloudwatch.Integration) {
	api.cloudWatchIntegration = cloudWatchIntegration
}

// SetSSMIntegration sets the SSM integration for the ECS API
func (api *DefaultECSAPI) SetSSMIntegration(ssmIntegration ssm.Integration) {
	api.ssmIntegration = ssmIntegration
}

// SetSecretsManagerIntegration sets the Secrets Manager integration for the ECS API
func (api *DefaultECSAPI) SetSecretsManagerIntegration(secretsManagerIntegration secretsmanager.Integration) {
	api.secretsManagerIntegration = secretsManagerIntegration
}

// SetS3Integration sets the S3 integration for the ECS API
func (api *DefaultECSAPI) SetS3Integration(s3Integration s3.Integration) {
	api.s3Integration = s3Integration
}

// SetELBv2Integration sets the ELBv2 integration for the ECS API
func (api *DefaultECSAPI) SetELBv2Integration(elbv2Integration elbv2.Integration) {
	api.elbv2Integration = elbv2Integration
}

// SetServiceDiscoveryManager sets the service discovery manager for the ECS API
func (api *DefaultECSAPI) SetServiceDiscoveryManager(serviceDiscoveryManager servicediscovery.Manager) {
	api.serviceDiscoveryManager = serviceDiscoveryManager
}

// SetServiceManager sets the service manager for the ECS API
func (api *DefaultECSAPI) SetServiceManager(serviceManager *kubernetes.ServiceManager) {
	api.serviceManager = serviceManager
}

// SetTaskSetManager sets the TaskSet manager for the ECS API
func (api *DefaultECSAPI) SetTaskSetManager(taskSetManager *kubernetes.TaskSetManager) {
	api.taskSetManager = taskSetManager
}

// SetLocalStackManager sets the LocalStack manager for the ECS API
func (api *DefaultECSAPI) SetLocalStackManager(localStackManager localstack.Manager) {
	api.localStackManager = localStackManager
}

// SetLocalStackConfig sets the LocalStack configuration for the ECS API
func (api *DefaultECSAPI) SetLocalStackConfig(config *localstack.Config) {
	api.localStackConfig = config
}

// SetLocalStackUpdateCallback sets the callback function to be called when LocalStack manager is updated
func (api *DefaultECSAPI) SetLocalStackUpdateCallback(callback func(localstack.Manager)) {
	api.localStackUpdateCallback = callback
}

// NewDefaultECSAPIWithConfig creates a new default ECS API implementation with custom region and accountID
// Deprecated: Use NewDefaultECSAPIWithClusterManager instead
func NewDefaultECSAPIWithConfig(cfg *config.Config, storage storage.Storage, region, accountID string) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		config:    cfg,
		storage:   storage,
		region:    region,
		accountID: accountID,
	}
}

// taskManager returns the task manager, creating it if necessary
func (api *DefaultECSAPI) taskManager() (*kubernetes.TaskManager, error) {
	if api.taskManagerInstance == nil {
		tm, err := kubernetes.NewTaskManagerWithServiceDiscovery(api.storage, api.serviceDiscoveryManager)
		if err != nil {
			return nil, err
		}
		api.taskManagerInstance = tm
	}

	// Ensure the kubernetes client is initialized
	if err := api.taskManagerInstance.InitializeClient(); err != nil {
		return nil, err
	}

	return api.taskManagerInstance, nil
}
