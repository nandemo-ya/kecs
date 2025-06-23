package api

import (
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/elbv2"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// DefaultECSAPIWithLB extends DefaultECSAPI with load balancer integration
type DefaultECSAPIWithLB struct {
	*DefaultECSAPI
	elbv2Integration elbv2.Integration
}

// NewDefaultECSAPIWithLB creates a new ECS API implementation with ELBv2 integration
// Deprecated: Use cluster manager version instead
func NewDefaultECSAPIWithLB(
	storage storage.Storage,
	region, accountID string,
	elbv2Integration elbv2.Integration,
) generated.ECSAPIInterface {
	return &DefaultECSAPIWithLB{
		DefaultECSAPI: &DefaultECSAPI{
			storage:   storage,
			region:    region,
			accountID: accountID,
		},
		elbv2Integration: elbv2Integration,
	}
}

// GetELBv2Integration returns the ELBv2 integration instance
func (api *DefaultECSAPIWithLB) GetELBv2Integration() elbv2.Integration {
	return api.elbv2Integration
}