package api

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// CreateCapacityProviderV2 implements the CreateCapacityProvider operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) CreateCapacityProviderV2(ctx context.Context, req *ecs.CreateCapacityProviderInput) (*ecs.CreateCapacityProviderOutput, error) {
	// TODO: Implement actual capacity provider creation logic
	// For now, return a mock response
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name

	return &ecs.CreateCapacityProviderOutput{
		CapacityProvider: &types.CapacityProvider{
			CapacityProviderArn:      aws.String(arn),
			Name:                     req.Name,
			Status:                   types.CapacityProviderStatusActive,
			AutoScalingGroupProvider: req.AutoScalingGroupProvider,
			Tags:                     req.Tags,
		},
	}, nil
}

// DeleteCapacityProviderV2 implements the DeleteCapacityProvider operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DeleteCapacityProviderV2(ctx context.Context, req *ecs.DeleteCapacityProviderInput) (*ecs.DeleteCapacityProviderOutput, error) {
	// TODO: Implement actual capacity provider deletion logic
	// For now, return a mock response
	name := ""
	if req.CapacityProvider != nil {
		name = *req.CapacityProvider
	}
	arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name

	return &ecs.DeleteCapacityProviderOutput{
		CapacityProvider: &types.CapacityProvider{
			CapacityProviderArn: aws.String(arn),
			Name:                req.CapacityProvider,
			Status:              types.CapacityProviderStatusInactive,
		},
	}, nil
}

// DescribeCapacityProvidersV2 implements the DescribeCapacityProviders operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) DescribeCapacityProvidersV2(ctx context.Context, req *ecs.DescribeCapacityProvidersInput) (*ecs.DescribeCapacityProvidersOutput, error) {
	// TODO: Implement actual capacity provider description logic
	// For now, return a mock response
	var capacityProviders []types.CapacityProvider

	if len(req.CapacityProviders) > 0 {
		for _, name := range req.CapacityProviders {
			arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name
			capacityProviders = append(capacityProviders, types.CapacityProvider{
				CapacityProviderArn: aws.String(arn),
				Name:                aws.String(name),
				Status:              types.CapacityProviderStatusActive,
			})
		}
	} else {
		// Return default capacity providers if none specified
		capacityProviders = append(capacityProviders, types.CapacityProvider{
			CapacityProviderArn: aws.String("arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/FARGATE"),
			Name:                aws.String("FARGATE"),
			Status:              types.CapacityProviderStatusActive,
		})
		capacityProviders = append(capacityProviders, types.CapacityProvider{
			CapacityProviderArn: aws.String("arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/FARGATE_SPOT"),
			Name:                aws.String("FARGATE_SPOT"),
			Status:              types.CapacityProviderStatusActive,
		})
	}

	return &ecs.DescribeCapacityProvidersOutput{
		CapacityProviders: capacityProviders,
	}, nil
}

// UpdateCapacityProviderV2 implements the UpdateCapacityProvider operation using AWS SDK v2 types
func (api *DefaultECSAPIV2) UpdateCapacityProviderV2(ctx context.Context, req *ecs.UpdateCapacityProviderInput) (*ecs.UpdateCapacityProviderOutput, error) {
	// TODO: Implement actual capacity provider update logic
	// For now, return a mock response
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	arn := "arn:aws:ecs:" + api.region + ":" + api.accountID + ":capacity-provider/" + name

	// Convert AutoScalingGroupProviderUpdate to AutoScalingGroupProvider
	var autoScalingGroupProvider *types.AutoScalingGroupProvider
	if req.AutoScalingGroupProvider != nil {
		autoScalingGroupProvider = &types.AutoScalingGroupProvider{
			ManagedScaling:               req.AutoScalingGroupProvider.ManagedScaling,
			ManagedTerminationProtection: req.AutoScalingGroupProvider.ManagedTerminationProtection,
		}
	}

	return &ecs.UpdateCapacityProviderOutput{
		CapacityProvider: &types.CapacityProvider{
			CapacityProviderArn:      aws.String(arn),
			Name:                     req.Name,
			Status:                   types.CapacityProviderStatusActive,
			AutoScalingGroupProvider: autoScalingGroupProvider,
			UpdateStatus:             types.CapacityProviderUpdateStatusUpdateComplete,
		},
	}, nil
}