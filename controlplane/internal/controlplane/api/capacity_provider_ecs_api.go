package api

import (
	"context"
	"fmt"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
)

// CreateCapacityProvider implements the CreateCapacityProvider operation
func (api *DefaultECSAPI) CreateCapacityProvider(ctx context.Context, req *generated.CreateCapacityProviderRequest) (*generated.CreateCapacityProviderResponse, error) {
	// TODO: Implement CreateCapacityProvider
	return nil, fmt.Errorf("CreateCapacityProvider not implemented")
}

// DeleteCapacityProvider implements the DeleteCapacityProvider operation
func (api *DefaultECSAPI) DeleteCapacityProvider(ctx context.Context, req *generated.DeleteCapacityProviderRequest) (*generated.DeleteCapacityProviderResponse, error) {
	// TODO: Implement DeleteCapacityProvider
	return nil, fmt.Errorf("DeleteCapacityProvider not implemented")
}

// DescribeCapacityProviders implements the DescribeCapacityProviders operation
func (api *DefaultECSAPI) DescribeCapacityProviders(ctx context.Context, req *generated.DescribeCapacityProvidersRequest) (*generated.DescribeCapacityProvidersResponse, error) {
	// TODO: Implement DescribeCapacityProviders
	return nil, fmt.Errorf("DescribeCapacityProviders not implemented")
}

// UpdateCapacityProvider implements the UpdateCapacityProvider operation
func (api *DefaultECSAPI) UpdateCapacityProvider(ctx context.Context, req *generated.UpdateCapacityProviderRequest) (*generated.UpdateCapacityProviderResponse, error) {
	// TODO: Implement UpdateCapacityProvider
	return nil, fmt.Errorf("UpdateCapacityProvider not implemented")
}