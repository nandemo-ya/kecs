package api

import (
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
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
	storage                   storage.Storage
	clusterManager            kubernetes.ClusterManager
	region                    string
	accountID                 string
	iamIntegration            iam.Integration
	cloudWatchIntegration     cloudwatch.Integration
	ssmIntegration            ssm.Integration
	secretsManagerIntegration secretsmanager.Integration
	s3Integration             s3.Integration
	serviceDiscoveryManager   servicediscovery.Manager
	localStackManager         localstack.Manager
	localStackConfig          *localstack.Config
}

// NewDefaultECSAPI creates a new default ECS API implementation with storage
// Deprecated: Use NewDefaultECSAPIWithClusterManager instead
func NewDefaultECSAPI(storage storage.Storage) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		storage:   storage,
		region:    "us-east-1",    // Default region
		accountID: "123456789012", // Default account ID
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

// SetServiceDiscoveryManager sets the service discovery manager for the ECS API
func (api *DefaultECSAPI) SetServiceDiscoveryManager(serviceDiscoveryManager servicediscovery.Manager) {
	api.serviceDiscoveryManager = serviceDiscoveryManager
}

// SetLocalStackManager sets the LocalStack manager for the ECS API
func (api *DefaultECSAPI) SetLocalStackManager(localStackManager localstack.Manager) {
	api.localStackManager = localStackManager
}

// SetLocalStackConfig sets the LocalStack configuration for the ECS API
func (api *DefaultECSAPI) SetLocalStackConfig(config *localstack.Config) {
	api.localStackConfig = config
}

// NewDefaultECSAPIWithConfig creates a new default ECS API implementation with custom region and accountID
// Deprecated: Use NewDefaultECSAPIWithClusterManager instead
func NewDefaultECSAPIWithConfig(storage storage.Storage, region, accountID string) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		storage:   storage,
		region:    region,
		accountID: accountID,
	}
}

// NewDefaultECSAPIWithClusterManager creates a new default ECS API implementation with ClusterManager
func NewDefaultECSAPIWithClusterManager(storage storage.Storage, clusterManager kubernetes.ClusterManager, region, accountID string) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		storage:        storage,
		clusterManager: clusterManager,
		region:         region,
		accountID:      accountID,
	}
}

// getClusterManager returns the cluster manager
func (api *DefaultECSAPI) getClusterManager() kubernetes.ClusterManager {
	return api.clusterManager
}
