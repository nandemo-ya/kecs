package kecs_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/nandemo-ya/kecs/testcontainers-go-kecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartContainer(t *testing.T) {
	ctx := context.Background()

	t.Run("DefaultConfiguration", func(t *testing.T) {
		container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
		require.NoError(t, err)
		defer container.Cleanup(ctx)

		// Verify endpoints
		assert.NotEmpty(t, container.Endpoint())
		assert.NotEmpty(t, container.AdminEndpoint())
		assert.Equal(t, kecs.DefaultRegion, container.Region())

		// Verify ECS client can be created
		client, err := container.NewECSClient(ctx)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("CustomConfiguration", func(t *testing.T) {
		container, err := kecs.StartContainer(ctx,
			kecs.WithTestMode(),
			kecs.WithRegion("eu-west-1"),
			kecs.WithAPIPort("9090"),
			kecs.WithAdminPort("9091"),
			kecs.WithEnv(map[string]string{
				"CUSTOM_VAR": "test",
			}),
		)
		require.NoError(t, err)
		defer container.Cleanup(ctx)

		assert.Equal(t, "eu-west-1", container.Region())
	})

	t.Run("ECSOperations", func(t *testing.T) {
		container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
		require.NoError(t, err)
		defer container.Cleanup(ctx)

		client, err := container.NewECSClient(ctx)
		require.NoError(t, err)

		// Create cluster
		clusterName := "test-cluster"
		createOutput, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: aws.String(clusterName),
		})
		require.NoError(t, err)
		assert.Equal(t, clusterName, aws.ToString(createOutput.Cluster.ClusterName))

		// List clusters
		listOutput, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
		require.NoError(t, err)
		assert.Contains(t, listOutput.ClusterArns, clusterName)

		// Delete cluster
		_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
			Cluster: aws.String(clusterName),
		})
		require.NoError(t, err)
	})
}

func TestHelperFunctions(t *testing.T) {
	ctx := context.Background()

	// Start container
	container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Create cluster
	clusterName := "helper-test-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	t.Run("CreateTestTaskDefinition", func(t *testing.T) {
		taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "test-family")
		require.NoError(t, err)
		assert.Equal(t, "test-family", aws.ToString(taskDef.Family))
		assert.Equal(t, int32(1), aws.ToInt32(taskDef.Revision))
		assert.Len(t, taskDef.ContainerDefinitions, 1)
	})

	t.Run("CreateTestService", func(t *testing.T) {
		// Create task definition first
		taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "service-test-family")
		require.NoError(t, err)

		// Create service
		service, err := kecs.CreateTestService(ctx, client, clusterName, "test-service", aws.ToString(taskDef.TaskDefinitionArn))
		require.NoError(t, err)
		assert.Equal(t, "test-service", aws.ToString(service.ServiceName))
		assert.Equal(t, int32(1), aws.ToInt32(service.DesiredCount))

		// Clean up service
		_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String("test-service"),
			DesiredCount: aws.Int32(0),
		})
		require.NoError(t, err)

		_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: aws.String(clusterName),
			Service: aws.String("test-service"),
		})
		require.NoError(t, err)
	})

	t.Run("WaitForCluster", func(t *testing.T) {
		// Cluster should already be active
		err := kecs.WaitForCluster(ctx, client, clusterName, "ACTIVE", 5*time.Second)
		assert.NoError(t, err)

		// Test timeout
		err = kecs.WaitForCluster(ctx, client, "non-existent-cluster", "ACTIVE", 1*time.Second)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}