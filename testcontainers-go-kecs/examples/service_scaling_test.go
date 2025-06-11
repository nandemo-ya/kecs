package examples_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/nandemo-ya/kecs/testcontainers-go-kecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceScaling(t *testing.T) {
	ctx := context.Background()

	// Start KECS container
	container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Setup
	clusterName := "scaling-test-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	// Register task definition
	taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "scaling-task")
	require.NoError(t, err)

	t.Run("ScaleUpAndDown", func(t *testing.T) {
		serviceName := "scaling-service"

		// Create service with 1 task
		createOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String(serviceName),
			TaskDefinition: taskDef.TaskDefinitionArn,
			DesiredCount:   aws.Int32(1),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(1), aws.ToInt32(createOutput.Service.DesiredCount))

		// Wait for service to be active
		err = kecs.WaitForService(ctx, client, clusterName, serviceName, "ACTIVE", 30*time.Second)
		require.NoError(t, err)

		// Scale up to 3 tasks
		updateOutput, err := client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(serviceName),
			DesiredCount: aws.Int32(3),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(3), aws.ToInt32(updateOutput.Service.DesiredCount))

		// Wait a bit for scaling to take effect
		time.Sleep(2 * time.Second)

		// Verify running count
		describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterName),
			Services: []string{serviceName},
		})
		require.NoError(t, err)
		require.Len(t, describeOutput.Services, 1)
		assert.Equal(t, int32(3), aws.ToInt32(describeOutput.Services[0].DesiredCount))

		// Scale down to 0
		updateOutput, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(serviceName),
			DesiredCount: aws.Int32(0),
		})
		require.NoError(t, err)
		assert.Equal(t, int32(0), aws.ToInt32(updateOutput.Service.DesiredCount))

		// Delete service
		_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: aws.String(clusterName),
			Service: aws.String(serviceName),
		})
		require.NoError(t, err)
	})

	t.Run("ServiceWithPlacementStrategy", func(t *testing.T) {
		serviceName := "placement-service"

		// Create service with placement strategies
		createOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String(serviceName),
			TaskDefinition: taskDef.TaskDefinitionArn,
			DesiredCount:   aws.Int32(2),
			PlacementStrategy: []types.PlacementStrategy{
				{
					Type:  types.PlacementStrategyTypeSpread,
					Field: aws.String("attribute:ecs.availability-zone"),
				},
			},
			PlacementConstraints: []types.PlacementConstraint{
				{
					Type:       types.PlacementConstraintTypeDistinctInstance,
					Expression: aws.String(""),
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(2), aws.ToInt32(createOutput.Service.DesiredCount))
		assert.Len(t, createOutput.Service.PlacementStrategy, 1)
		assert.Len(t, createOutput.Service.PlacementConstraints, 1)

		// Clean up
		_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(serviceName),
			DesiredCount: aws.Int32(0),
		})
		require.NoError(t, err)

		_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: aws.String(clusterName),
			Service: aws.String(serviceName),
		})
		require.NoError(t, err)
	})
}

func TestServiceDeploymentConfiguration(t *testing.T) {
	ctx := context.Background()

	// Start KECS container
	container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Setup
	clusterName := "deployment-test-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	// Register initial task definition
	v1Output, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("deployment-task"),
		NetworkMode: types.NetworkModeBridge,
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:      aws.String("app"),
				Image:     aws.String("nginx:1.19"),
				Memory:    aws.Int32(256),
				Essential: aws.Bool(true),
			},
		},
	})
	require.NoError(t, err)

	t.Run("RollingUpdateDeployment", func(t *testing.T) {
		serviceName := "rolling-update-service"

		// Create service with custom deployment configuration
		createOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String(serviceName),
			TaskDefinition: v1Output.TaskDefinition.TaskDefinitionArn,
			DesiredCount:   aws.Int32(3),
			DeploymentConfiguration: &types.DeploymentConfiguration{
				MaximumPercent:        aws.Int32(200),
				MinimumHealthyPercent: aws.Int32(50),
			},
		})
		require.NoError(t, err)
		assert.Equal(t, int32(200), aws.ToInt32(createOutput.Service.DeploymentConfiguration.MaximumPercent))
		assert.Equal(t, int32(50), aws.ToInt32(createOutput.Service.DeploymentConfiguration.MinimumHealthyPercent))

		// Wait for initial deployment
		err = kecs.WaitForService(ctx, client, clusterName, serviceName, "ACTIVE", 30*time.Second)
		require.NoError(t, err)

		// Register new version of task definition
		v2Output, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			Family:      aws.String("deployment-task"),
			NetworkMode: types.NetworkModeBridge,
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:      aws.String("app"),
					Image:     aws.String("nginx:1.20"), // Updated version
					Memory:    aws.Int32(256),
					Essential: aws.Bool(true),
				},
			},
		})
		require.NoError(t, err)

		// Update service to use new task definition
		updateOutput, err := client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:        aws.String(clusterName),
			Service:        aws.String(serviceName),
			TaskDefinition: v2Output.TaskDefinition.TaskDefinitionArn,
		})
		require.NoError(t, err)
		assert.Contains(t, aws.ToString(updateOutput.Service.TaskDefinition), "deployment-task:2")

		// Check deployments
		describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  aws.String(clusterName),
			Services: []string{serviceName},
		})
		require.NoError(t, err)
		require.Len(t, describeOutput.Services, 1)

		service := describeOutput.Services[0]
		t.Logf("Service has %d deployments", len(service.Deployments))
		for _, deployment := range service.Deployments {
			t.Logf("Deployment: %s, Status: %s, Running: %d, Pending: %d, Desired: %d",
				aws.ToString(deployment.Id),
				aws.ToString(deployment.Status),
				aws.ToInt32(deployment.RunningCount),
				aws.ToInt32(deployment.PendingCount),
				aws.ToInt32(deployment.DesiredCount),
			)
		}

		// Clean up
		_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(serviceName),
			DesiredCount: aws.Int32(0),
		})
		require.NoError(t, err)

		_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: aws.String(clusterName),
			Service: aws.String(serviceName),
		})
		require.NoError(t, err)
	})
}