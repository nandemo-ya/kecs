package api

import (
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/kubernetes"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPIWithLB extends DefaultECSAPI with load balancer integration
type DefaultECSAPIWithLB struct {
	*DefaultECSAPI
	elbv2Integration elbv2.Integration
}

// NewDefaultECSAPIWithLB creates a new ECS API implementation with ELBv2 integration
func NewDefaultECSAPIWithLB(
	storage storage.Storage,
	kindManager *kubernetes.KindManager,
	region, accountID string,
	elbv2Integration elbv2.Integration,
) generated.ECSAPIInterface {
	return &DefaultECSAPIWithLB{
		DefaultECSAPI: &DefaultECSAPI{
			storage:     storage,
			kindManager: kindManager,
			region:      region,
			accountID:   accountID,
		},
		elbv2Integration: elbv2Integration,
	}
}

// GetELBv2Integration returns the ELBv2 integration instance
func (api *DefaultECSAPIWithLB) GetELBv2Integration() elbv2.Integration {
	return api.elbv2Integration
}