package api

import (
	"context"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated/ptr"
)

// CreateCapacityProvider implements the CreateCapacityProvider operation
func (api *DefaultECSAPI) CreateCapacityProvider(ctx context.Context, req *generated.CreateCapacityProviderRequest) (*generated.CreateCapacityProviderResponse, error) {
	// TODO: Implement actual capacity provider creation logic
	// For now, return a mock response
	name := req.Name
	arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name

	resp := &generated.CreateCapacityProviderResponse{
		CapacityProvider: &generated.CapacityProvider{
			CapacityProviderArn:      ptr.String(arn),
			Name:                     ptr.String(req.Name),
			Status:                   (*generated.CapacityProviderStatus)(ptr.String("ACTIVE")),
			AutoScalingGroupProvider: &req.AutoScalingGroupProvider,
			Tags:                     req.Tags,
		},
	}

	return resp, nil
}

// DeleteCapacityProvider implements the DeleteCapacityProvider operation
func (api *DefaultECSAPI) DeleteCapacityProvider(ctx context.Context, req *generated.DeleteCapacityProviderRequest) (*generated.DeleteCapacityProviderResponse, error) {
	// TODO: Implement actual capacity provider deletion logic
	// For now, return a mock response
	name := req.CapacityProvider
	arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name

	resp := &generated.DeleteCapacityProviderResponse{
		CapacityProvider: &generated.CapacityProvider{
			CapacityProviderArn: ptr.String(arn),
			Name:                ptr.String(req.CapacityProvider),
			Status:              (*generated.CapacityProviderStatus)(ptr.String("INACTIVE")),
		},
	}

	return resp, nil
}

// DescribeCapacityProviders implements the DescribeCapacityProviders operation
func (api *DefaultECSAPI) DescribeCapacityProviders(ctx context.Context, req *generated.DescribeCapacityProvidersRequest) (*generated.DescribeCapacityProvidersResponse, error) {
	// TODO: Implement actual capacity provider description logic
	// For now, return a mock response
	capacityProviders := []generated.CapacityProvider{}

	if len(req.CapacityProviders) > 0 {
		for _, name := range req.CapacityProviders {
			arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name
			capacityProviders = append(capacityProviders, generated.CapacityProvider{
				CapacityProviderArn: ptr.String(arn),
				Name:                ptr.String(name),
				Status:              (*generated.CapacityProviderStatus)(ptr.String("ACTIVE")),
			})
		}
	} else {
		// Return default capacity providers if none specified
		capacityProviders = append(capacityProviders, generated.CapacityProvider{
			CapacityProviderArn: ptr.String("arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/FARGATE"),
			Name:                ptr.String("FARGATE"),
			Status:              (*generated.CapacityProviderStatus)(ptr.String("ACTIVE")),
		})
		capacityProviders = append(capacityProviders, generated.CapacityProvider{
			CapacityProviderArn: ptr.String("arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/FARGATE_SPOT"),
			Name:                ptr.String("FARGATE_SPOT"),
			Status:              (*generated.CapacityProviderStatus)(ptr.String("ACTIVE")),
		})
	}

	resp := &generated.DescribeCapacityProvidersResponse{
		CapacityProviders: capacityProviders,
	}

	return resp, nil
}

// UpdateCapacityProvider implements the UpdateCapacityProvider operation
func (api *DefaultECSAPI) UpdateCapacityProvider(ctx context.Context, req *generated.UpdateCapacityProviderRequest) (*generated.UpdateCapacityProviderResponse, error) {
	// TODO: Implement actual capacity provider update logic
	// For now, return a mock response
	name := req.Name
	arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name

	// Convert AutoScalingGroupProviderUpdate to AutoScalingGroupProvider
	autoScalingGroupProvider := &generated.AutoScalingGroupProvider{
		ManagedScaling:               req.AutoScalingGroupProvider.ManagedScaling,
		ManagedTerminationProtection: req.AutoScalingGroupProvider.ManagedTerminationProtection,
	}

	resp := &generated.UpdateCapacityProviderResponse{
		CapacityProvider: &generated.CapacityProvider{
			CapacityProviderArn:      ptr.String(arn),
			Name:                     ptr.String(req.Name),
			Status:                   (*generated.CapacityProviderStatus)(ptr.String("ACTIVE")),
			AutoScalingGroupProvider: autoScalingGroupProvider,
			UpdateStatus:             (*generated.CapacityProviderUpdateStatus)(ptr.String("UPDATE_COMPLETE")),
		},
	}

	return resp, nil
}
