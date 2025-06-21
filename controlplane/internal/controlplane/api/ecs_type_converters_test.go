package api

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/stretchr/testify/assert"
)

func TestNetworkConfigurationConverters(t *testing.T) {
	t.Run("ConvertToGeneratedNetworkConfiguration", func(t *testing.T) {
		// Test nil input
		assert.Nil(t, ConvertToGeneratedNetworkConfiguration(nil))

		// Test full configuration
		awsSDKConfig := &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        []string{"subnet-12345", "subnet-67890"},
				SecurityGroups: []string{"sg-abcdef", "sg-123456"},
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		}

		result := ConvertToGeneratedNetworkConfiguration(awsSDKConfig)
		assert.NotNil(t, result)
		assert.NotNil(t, result.AwsvpcConfiguration)
		assert.Equal(t, []string{"subnet-12345", "subnet-67890"}, result.AwsvpcConfiguration.Subnets)
		assert.Equal(t, []string{"sg-abcdef", "sg-123456"}, result.AwsvpcConfiguration.SecurityGroups)
		assert.Equal(t, generated.AssignPublicIpEnabled, *result.AwsvpcConfiguration.AssignPublicIp)

		// Test configuration without AssignPublicIp
		awsSDKConfigNoIP := &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        []string{"subnet-12345"},
				SecurityGroups: []string{"sg-abcdef"},
			},
		}

		resultNoIP := ConvertToGeneratedNetworkConfiguration(awsSDKConfigNoIP)
		assert.NotNil(t, resultNoIP)
		assert.NotNil(t, resultNoIP.AwsvpcConfiguration)
		assert.Equal(t, []string{"subnet-12345"}, resultNoIP.AwsvpcConfiguration.Subnets)
		assert.Equal(t, []string{"sg-abcdef"}, resultNoIP.AwsvpcConfiguration.SecurityGroups)
		assert.Nil(t, resultNoIP.AwsvpcConfiguration.AssignPublicIp)
	})

	t.Run("ConvertFromGeneratedNetworkConfiguration", func(t *testing.T) {
		// Test nil input
		assert.Nil(t, ConvertFromGeneratedNetworkConfiguration(nil))

		// Test full configuration
		assignPublicIp := generated.AssignPublicIpEnabled
		generatedConfig := &generated.NetworkConfiguration{
			AwsvpcConfiguration: &generated.AwsVpcConfiguration{
				Subnets:        []string{"subnet-12345", "subnet-67890"},
				SecurityGroups: []string{"sg-abcdef", "sg-123456"},
				AssignPublicIp: &assignPublicIp,
			},
		}

		result := ConvertFromGeneratedNetworkConfiguration(generatedConfig)
		assert.NotNil(t, result)
		assert.NotNil(t, result.AwsvpcConfiguration)
		assert.Equal(t, []string{"subnet-12345", "subnet-67890"}, result.AwsvpcConfiguration.Subnets)
		assert.Equal(t, []string{"sg-abcdef", "sg-123456"}, result.AwsvpcConfiguration.SecurityGroups)
		assert.Equal(t, types.AssignPublicIpEnabled, result.AwsvpcConfiguration.AssignPublicIp)

		// Test configuration without AssignPublicIp
		generatedConfigNoIP := &generated.NetworkConfiguration{
			AwsvpcConfiguration: &generated.AwsVpcConfiguration{
				Subnets:        []string{"subnet-12345"},
				SecurityGroups: []string{"sg-abcdef"},
			},
		}

		resultNoIP := ConvertFromGeneratedNetworkConfiguration(generatedConfigNoIP)
		assert.NotNil(t, resultNoIP)
		assert.NotNil(t, resultNoIP.AwsvpcConfiguration)
		assert.Equal(t, []string{"subnet-12345"}, resultNoIP.AwsvpcConfiguration.Subnets)
		assert.Equal(t, []string{"sg-abcdef"}, resultNoIP.AwsvpcConfiguration.SecurityGroups)
		assert.Equal(t, types.AssignPublicIp(""), resultNoIP.AwsvpcConfiguration.AssignPublicIp)
	})

	t.Run("RoundTrip", func(t *testing.T) {
		// Test round-trip conversion maintains data integrity
		original := &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets:        []string{"subnet-12345", "subnet-67890"},
				SecurityGroups: []string{"sg-abcdef", "sg-123456"},
				AssignPublicIp: types.AssignPublicIpDisabled,
			},
		}

		// Convert to generated and back
		generated := ConvertToGeneratedNetworkConfiguration(original)
		roundTrip := ConvertFromGeneratedNetworkConfiguration(generated)

		assert.NotNil(t, roundTrip)
		assert.NotNil(t, roundTrip.AwsvpcConfiguration)
		assert.Equal(t, original.AwsvpcConfiguration.Subnets, roundTrip.AwsvpcConfiguration.Subnets)
		assert.Equal(t, original.AwsvpcConfiguration.SecurityGroups, roundTrip.AwsvpcConfiguration.SecurityGroups)
		assert.Equal(t, original.AwsvpcConfiguration.AssignPublicIp, roundTrip.AwsvpcConfiguration.AssignPublicIp)
	})
}

func TestLoadBalancerConverters(t *testing.T) {
	t.Run("ConvertToGeneratedLoadBalancers", func(t *testing.T) {
		// Test nil/empty input
		assert.Nil(t, ConvertToGeneratedLoadBalancers(nil))
		assert.Nil(t, ConvertToGeneratedLoadBalancers([]types.LoadBalancer{}))

		// Test full configuration
		containerName := "web-server"
		containerPort := int32(80)
		loadBalancerName := "my-load-balancer"
		targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-target-group/1234567890123456"

		awsSDKLBs := []types.LoadBalancer{
			{
				ContainerName:    &containerName,
				ContainerPort:    &containerPort,
				LoadBalancerName: &loadBalancerName,
				TargetGroupArn:   &targetGroupArn,
			},
			{
				ContainerName:  &containerName,
				ContainerPort:  &containerPort,
				TargetGroupArn: &targetGroupArn,
				// LoadBalancerName omitted for ALB/NLB
			},
		}

		result := ConvertToGeneratedLoadBalancers(awsSDKLBs)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		
		// Check first load balancer (with all fields)
		assert.Equal(t, &containerName, result[0].ContainerName)
		assert.Equal(t, &containerPort, result[0].ContainerPort)
		assert.Equal(t, &loadBalancerName, result[0].LoadBalancerName)
		assert.Equal(t, &targetGroupArn, result[0].TargetGroupArn)
		
		// Check second load balancer (without LoadBalancerName)
		assert.Equal(t, &containerName, result[1].ContainerName)
		assert.Equal(t, &containerPort, result[1].ContainerPort)
		assert.Nil(t, result[1].LoadBalancerName)
		assert.Equal(t, &targetGroupArn, result[1].TargetGroupArn)
	})

	t.Run("ConvertFromGeneratedLoadBalancers", func(t *testing.T) {
		// Test nil/empty input
		assert.Nil(t, ConvertFromGeneratedLoadBalancers(nil))
		assert.Nil(t, ConvertFromGeneratedLoadBalancers([]generated.LoadBalancer{}))

		// Test full configuration
		containerName := "web-server"
		containerPort := int32(80)
		loadBalancerName := "my-load-balancer"
		targetGroupArn := "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/my-target-group/1234567890123456"

		generatedLBs := []generated.LoadBalancer{
			{
				ContainerName:    &containerName,
				ContainerPort:    &containerPort,
				LoadBalancerName: &loadBalancerName,
				TargetGroupArn:   &targetGroupArn,
			},
			{
				ContainerName:  &containerName,
				ContainerPort:  &containerPort,
				TargetGroupArn: &targetGroupArn,
				// LoadBalancerName omitted
			},
		}

		result := ConvertFromGeneratedLoadBalancers(generatedLBs)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		
		// Check first load balancer (with all fields)
		assert.Equal(t, &containerName, result[0].ContainerName)
		assert.Equal(t, &containerPort, result[0].ContainerPort)
		assert.Equal(t, &loadBalancerName, result[0].LoadBalancerName)
		assert.Equal(t, &targetGroupArn, result[0].TargetGroupArn)
		
		// Check second load balancer (without LoadBalancerName)
		assert.Equal(t, &containerName, result[1].ContainerName)
		assert.Equal(t, &containerPort, result[1].ContainerPort)
		assert.Nil(t, result[1].LoadBalancerName)
		assert.Equal(t, &targetGroupArn, result[1].TargetGroupArn)
	})

	t.Run("RoundTrip", func(t *testing.T) {
		// Test round-trip conversion maintains data integrity
		containerName := "web-server"
		containerPort := int32(443)
		targetGroupArn := "arn:aws:elasticloadbalancing:us-west-2:123456789012:targetgroup/my-tg/abcdef123456"

		original := []types.LoadBalancer{
			{
				ContainerName:  &containerName,
				ContainerPort:  &containerPort,
				TargetGroupArn: &targetGroupArn,
			},
		}

		// Convert to generated and back
		generated := ConvertToGeneratedLoadBalancers(original)
		roundTrip := ConvertFromGeneratedLoadBalancers(generated)

		assert.NotNil(t, roundTrip)
		assert.Len(t, roundTrip, 1)
		assert.Equal(t, original[0].ContainerName, roundTrip[0].ContainerName)
		assert.Equal(t, original[0].ContainerPort, roundTrip[0].ContainerPort)
		assert.Equal(t, original[0].LoadBalancerName, roundTrip[0].LoadBalancerName)
		assert.Equal(t, original[0].TargetGroupArn, roundTrip[0].TargetGroupArn)
	})
}

func TestPlacementConstraintConverters(t *testing.T) {
	t.Run("ConvertToGeneratedPlacementConstraints", func(t *testing.T) {
		// Test nil/empty input
		assert.Nil(t, ConvertToGeneratedPlacementConstraints(nil))
		assert.Nil(t, ConvertToGeneratedPlacementConstraints([]types.PlacementConstraint{}))

		// Test full configuration
		expression := "attribute:ecs.instance-type =~ m5.*"
		
		awsSDKConstraints := []types.PlacementConstraint{
			{
				Expression: &expression,
				Type:       types.PlacementConstraintTypeMemberOf,
			},
			{
				Type: types.PlacementConstraintTypeDistinctInstance,
				// No expression for distinctInstance
			},
		}

		result := ConvertToGeneratedPlacementConstraints(awsSDKConstraints)
		assert.NotNil(t, result)
		assert.Len(t, result, 2)
		
		// Check first constraint (with expression)
		assert.Equal(t, &expression, result[0].Expression)
		assert.Equal(t, generated.PlacementConstraintTypeMemberOf, *result[0].Type)
		
		// Check second constraint (without expression)
		assert.Nil(t, result[1].Expression)
		assert.Equal(t, generated.PlacementConstraintTypeDistinctInstance, *result[1].Type)
	})

	t.Run("ConvertFromGeneratedPlacementConstraints", func(t *testing.T) {
		// Test nil/empty input
		assert.Nil(t, ConvertFromGeneratedPlacementConstraints(nil))
		assert.Nil(t, ConvertFromGeneratedPlacementConstraints([]generated.PlacementConstraint{}))

		// Test full configuration
		expression := "attribute:ecs.availability-zone != us-east-1a"
		constraintType := generated.PlacementConstraintTypeMemberOf
		
		generatedConstraints := []generated.PlacementConstraint{
			{
				Expression: &expression,
				Type:       &constraintType,
			},
		}

		result := ConvertFromGeneratedPlacementConstraints(generatedConstraints)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		
		assert.Equal(t, &expression, result[0].Expression)
		assert.Equal(t, types.PlacementConstraintTypeMemberOf, result[0].Type)
	})

	t.Run("RoundTrip", func(t *testing.T) {
		expression := "task:group == database"
		original := []types.PlacementConstraint{
			{
				Expression: &expression,
				Type:       types.PlacementConstraintTypeMemberOf,
			},
		}

		// Convert to generated and back
		generated := ConvertToGeneratedPlacementConstraints(original)
		roundTrip := ConvertFromGeneratedPlacementConstraints(generated)

		assert.NotNil(t, roundTrip)
		assert.Len(t, roundTrip, 1)
		assert.Equal(t, original[0].Expression, roundTrip[0].Expression)
		assert.Equal(t, original[0].Type, roundTrip[0].Type)
	})
}

func TestPlacementStrategyConverters(t *testing.T) {
	t.Run("ConvertToGeneratedPlacementStrategy", func(t *testing.T) {
		// Test nil/empty input
		assert.Nil(t, ConvertToGeneratedPlacementStrategy(nil))
		assert.Nil(t, ConvertToGeneratedPlacementStrategy([]types.PlacementStrategy{}))

		// Test full configuration
		field := "attribute:ecs.availability-zone"
		
		awsSDKStrategies := []types.PlacementStrategy{
			{
				Type:  types.PlacementStrategyTypeSpread,
				Field: &field,
			},
			{
				Type: types.PlacementStrategyTypeRandom,
				// No field for random
			},
			{
				Type:  types.PlacementStrategyTypeBinpack,
				Field: aws.String("memory"),
			},
		}

		result := ConvertToGeneratedPlacementStrategy(awsSDKStrategies)
		assert.NotNil(t, result)
		assert.Len(t, result, 3)
		
		// Check spread strategy
		assert.Equal(t, generated.PlacementStrategyTypeSpread, *result[0].Type)
		assert.Equal(t, &field, result[0].Field)
		
		// Check random strategy
		assert.Equal(t, generated.PlacementStrategyTypeRandom, *result[1].Type)
		assert.Nil(t, result[1].Field)
		
		// Check binpack strategy
		assert.Equal(t, generated.PlacementStrategyTypeBinpack, *result[2].Type)
		assert.Equal(t, "memory", *result[2].Field)
	})

	t.Run("ConvertFromGeneratedPlacementStrategy", func(t *testing.T) {
		// Test nil/empty input
		assert.Nil(t, ConvertFromGeneratedPlacementStrategy(nil))
		assert.Nil(t, ConvertFromGeneratedPlacementStrategy([]generated.PlacementStrategy{}))

		// Test full configuration
		field := "cpu"
		strategyType := generated.PlacementStrategyTypeBinpack
		
		generatedStrategies := []generated.PlacementStrategy{
			{
				Type:  &strategyType,
				Field: &field,
			},
		}

		result := ConvertFromGeneratedPlacementStrategy(generatedStrategies)
		assert.NotNil(t, result)
		assert.Len(t, result, 1)
		
		assert.Equal(t, types.PlacementStrategyTypeBinpack, result[0].Type)
		assert.Equal(t, &field, result[0].Field)
	})

	t.Run("RoundTrip", func(t *testing.T) {
		field := "attribute:ecs.instance-type"
		original := []types.PlacementStrategy{
			{
				Type:  types.PlacementStrategyTypeSpread,
				Field: &field,
			},
		}

		// Convert to generated and back
		generated := ConvertToGeneratedPlacementStrategy(original)
		roundTrip := ConvertFromGeneratedPlacementStrategy(generated)

		assert.NotNil(t, roundTrip)
		assert.Len(t, roundTrip, 1)
		assert.Equal(t, original[0].Type, roundTrip[0].Type)
		assert.Equal(t, original[0].Field, roundTrip[0].Field)
	})
}