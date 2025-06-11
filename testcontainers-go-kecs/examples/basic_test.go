package examples_test

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

func TestBasicECSOperations(t *testing.T) {
	ctx := context.Background()

	// Start KECS container in test mode
	container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Test cluster operations
	t.Run("CreateAndDescribeCluster", func(t *testing.T) {
		clusterName := "test-cluster"

		// Create cluster
		createOutput, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: aws.String(clusterName),
		})
		require.NoError(t, err)
		assert.Equal(t, clusterName, aws.ToString(createOutput.Cluster.ClusterName))
		assert.Equal(t, "ACTIVE", aws.ToString(createOutput.Cluster.Status))

		// Describe cluster
		describeOutput, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
			Clusters: []string{clusterName},
		})
		require.NoError(t, err)
		require.Len(t, describeOutput.Clusters, 1)
		assert.Equal(t, clusterName, aws.ToString(describeOutput.Clusters[0].ClusterName))
	})

	// Test task definition operations
	t.Run("RegisterTaskDefinition", func(t *testing.T) {
		// Register task definition
		registerOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			Family: aws.String("test-task"),
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:      aws.String("nginx"),
					Image:     aws.String("nginx:latest"),
					Memory:    aws.Int32(512),
					Essential: aws.Bool(true),
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "test-task", aws.ToString(registerOutput.TaskDefinition.Family))
		assert.Equal(t, int32(1), aws.ToInt32(registerOutput.TaskDefinition.Revision))
	})

	// Test service operations
	t.Run("CreateAndListServices", func(t *testing.T) {
		clusterName := "service-test-cluster"
		
		// Create cluster first
		_, err := client.CreateCluster(ctx, &ecs.CreateClusterInput{
			ClusterName: aws.String(clusterName),
		})
		require.NoError(t, err)

		// Create task definition
		taskDef, err := kecs.CreateTestTaskDefinition(ctx, client, "service-task")
		require.NoError(t, err)

		// Create service
		serviceName := "test-service"
		createServiceOutput, err := client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String(serviceName),
			TaskDefinition: taskDef.TaskDefinitionArn,
			DesiredCount:   aws.Int32(2),
		})
		require.NoError(t, err)
		assert.Equal(t, serviceName, aws.ToString(createServiceOutput.Service.ServiceName))

		// List services
		listOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
			Cluster: aws.String(clusterName),
		})
		require.NoError(t, err)
		assert.Len(t, listOutput.ServiceArns, 1)

		// Clean up
		err = kecs.CleanupCluster(ctx, client, clusterName)
		assert.NoError(t, err)
	})
}

func TestClusterLifecycle(t *testing.T) {
	ctx := context.Background()

	// Start KECS with custom configuration
	container, err := kecs.StartContainer(ctx,
		kecs.WithTestMode(),
		kecs.WithRegion("us-west-2"),
		kecs.WithEnv(map[string]string{
			"LOG_LEVEL": "debug",
		}),
	)
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Verify configuration
	assert.Equal(t, "us-west-2", container.Region())

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	clusterName := "lifecycle-test"

	// Create cluster
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
		Tags: []types.Tag{
			{
				Key:   aws.String("Environment"),
				Value: aws.String("test"),
			},
		},
	})
	require.NoError(t, err)

	// Wait for cluster to be active
	err = kecs.WaitForCluster(ctx, client, clusterName, "ACTIVE", 10*time.Second)
	require.NoError(t, err)

	// List clusters
	listOutput, err := client.ListClusters(ctx, &ecs.ListClustersInput{})
	require.NoError(t, err)
	assert.Contains(t, listOutput.ClusterArns, clusterName)

	// Delete cluster
	_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
		Cluster: aws.String(clusterName),
	})
	require.NoError(t, err)

	// Verify cluster is deleted
	describeOutput, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
		Clusters: []string{clusterName},
	})
	require.NoError(t, err)
	if len(describeOutput.Clusters) > 0 {
		assert.Equal(t, "INACTIVE", aws.ToString(describeOutput.Clusters[0].Status))
	}
}