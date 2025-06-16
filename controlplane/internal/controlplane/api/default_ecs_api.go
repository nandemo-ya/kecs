package api

import (
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/iam"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPI provides the default implementation of ECS API operations
type DefaultECSAPI struct {
	storage        storage.Storage
	kindManager    *kubernetes.KindManager
	region         string
	accountID      string
	iamIntegration iam.Integration
}

// NewDefaultECSAPI creates a new default ECS API implementation with storage and kubernetes manager
func NewDefaultECSAPI(storage storage.Storage, kindManager *kubernetes.KindManager) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		storage:     storage,
		kindManager: kindManager,
		region:      "ap-northeast-1", // Default region
		accountID:   "123456789012",   // Default account ID
	}
}

// SetIAMIntegration sets the IAM integration for the ECS API
func (api *DefaultECSAPI) SetIAMIntegration(iamIntegration iam.Integration) {
	api.iamIntegration = iamIntegration
}

// NewDefaultECSAPIWithConfig creates a new default ECS API implementation with custom region and accountID
func NewDefaultECSAPIWithConfig(storage storage.Storage, kindManager *kubernetes.KindManager, region, accountID string) generated.ECSAPIInterface {
	return &DefaultECSAPI{
		storage:     storage,
		kindManager: kindManager,
		region:      region,
		accountID:   accountID,
	}
}
