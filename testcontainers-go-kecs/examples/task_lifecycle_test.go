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
	"github.com/testcontainers/testcontainers-go"
)

func TestTaskLifecycle(t *testing.T) {
	ctx := context.Background()

	// Start KECS container
	container, err := kecs.StartContainer(ctx, kecs.WithTestMode())
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Setup cluster and task definition
	clusterName := "task-test-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	// Register task definition
	taskDefOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("lifecycle-task"),
		NetworkMode: types.NetworkModeBridge,
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:  aws.String("app"),
				Image: aws.String("busybox:latest"),
				Command: []string{
					"sh",
					"-c",
					"echo 'Starting task' && sleep 30 && echo 'Task completed'",
				},
				Memory:    aws.Int32(128),
				Essential: aws.Bool(true),
			},
		},
	})
	require.NoError(t, err)

	t.Run("RunAndStopTask", func(t *testing.T) {
		// Run task
		runTaskOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:        aws.String(clusterName),
			TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
			Count:          aws.Int32(1),
		})
		require.NoError(t, err)
		require.Len(t, runTaskOutput.Tasks, 1)

		taskArn := aws.ToString(runTaskOutput.Tasks[0].TaskArn)
		
		// Wait for task to be running
		err = kecs.WaitForTask(ctx, client, clusterName, taskArn, "RUNNING", 30*time.Second)
		require.NoError(t, err)

		// Describe task
		describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: aws.String(clusterName),
			Tasks:   []string{taskArn},
		})
		require.NoError(t, err)
		require.Len(t, describeOutput.Tasks, 1)
		assert.Equal(t, "RUNNING", string(describeOutput.Tasks[0].LastStatus))

		// Stop task
		stopOutput, err := client.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: aws.String(clusterName),
			Task:    aws.String(taskArn),
			Reason:  aws.String("Test completed"),
		})
		require.NoError(t, err)
		assert.Equal(t, taskArn, aws.ToString(stopOutput.Task.TaskArn))

		// Wait for task to be stopped
		err = kecs.WaitForTask(ctx, client, clusterName, taskArn, "STOPPED", 30*time.Second)
		require.NoError(t, err)
	})

	t.Run("TaskStatusTransitions", func(t *testing.T) {
		// Run a task that will complete naturally
		taskDefOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			Family:      aws.String("quick-task"),
			NetworkMode: types.NetworkModeBridge,
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:  aws.String("quick"),
					Image: aws.String("busybox:latest"),
					Command: []string{
						"sh",
						"-c",
						"echo 'Quick task' && sleep 5",
					},
					Memory:    aws.Int32(128),
					Essential: aws.Bool(true),
				},
			},
		})
		require.NoError(t, err)

		// Run task
		runTaskOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
			Cluster:        aws.String(clusterName),
			TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
			Count:          aws.Int32(1),
		})
		require.NoError(t, err)
		require.Len(t, runTaskOutput.Tasks, 1)

		taskArn := aws.ToString(runTaskOutput.Tasks[0].TaskArn)

		// Track status transitions
		var statuses []string
		timeout := time.After(20 * time.Second)
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		lastStatus := ""
		for {
			select {
			case <-timeout:
				t.Fatal("Timeout waiting for task to complete")
			case <-ticker.C:
				describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
					Cluster: aws.String(clusterName),
					Tasks:   []string{taskArn},
				})
				require.NoError(t, err)

				if len(describeOutput.Tasks) > 0 {
					currentStatus := string(describeOutput.Tasks[0].LastStatus)
					if currentStatus != lastStatus {
						statuses = append(statuses, currentStatus)
						lastStatus = currentStatus
						t.Logf("Task status: %s", currentStatus)
					}

					if currentStatus == "STOPPED" {
						goto done
					}
				}
			}
		}
	done:

		// Verify we saw expected status transitions
		assert.Contains(t, statuses, "PENDING")
		assert.Contains(t, statuses, "RUNNING")
		assert.Contains(t, statuses, "STOPPED")
	})
}

func TestTaskWithMultipleContainers(t *testing.T) {
	ctx := context.Background()

	// Start KECS container with logging
	container, err := kecs.StartContainer(ctx,
		kecs.WithTestMode(),
		kecs.WithLogConsumer(testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
			t.Logf("KECS: %s", log.Content)
		})),
	)
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Setup
	clusterName := "multi-container-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	// Register task definition with multiple containers
	taskDefOutput, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("multi-container-task"),
		NetworkMode: types.NetworkModeBridge,
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:      aws.String("web"),
				Image:     aws.String("nginx:alpine"),
				Memory:    aws.Int32(256),
				Essential: aws.Bool(true),
				PortMappings: []types.PortMapping{
					{
						ContainerPort: aws.Int32(80),
						HostPort:      aws.Int32(8080),
						Protocol:      types.TransportProtocolTcp,
					},
				},
			},
			{
				Name:  aws.String("sidecar"),
				Image: aws.String("busybox:latest"),
				Command: []string{
					"sh",
					"-c",
					"while true; do echo 'Sidecar running'; sleep 10; done",
				},
				Memory:    aws.Int32(128),
				Essential: aws.Bool(false),
			},
		},
	})
	require.NoError(t, err)

	// Run task
	runTaskOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
		Cluster:        aws.String(clusterName),
		TaskDefinition: taskDefOutput.TaskDefinition.TaskDefinitionArn,
		Count:          aws.Int32(1),
	})
	require.NoError(t, err)
	require.Len(t, runTaskOutput.Tasks, 1)

	taskArn := aws.ToString(runTaskOutput.Tasks[0].TaskArn)

	// Wait for task to be running
	err = kecs.WaitForTask(ctx, client, clusterName, taskArn, "RUNNING", 30*time.Second)
	require.NoError(t, err)

	// Verify both containers are running
	describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(clusterName),
		Tasks:   []string{taskArn},
	})
	require.NoError(t, err)
	require.Len(t, describeOutput.Tasks, 1)

	task := describeOutput.Tasks[0]
	assert.Len(t, task.Containers, 2)

	// Verify container statuses
	for _, container := range task.Containers {
		t.Logf("Container %s status: %s", aws.ToString(container.Name), aws.ToString(container.LastStatus))
		assert.Equal(t, "RUNNING", aws.ToString(container.LastStatus))
	}

	// Clean up
	_, err = client.StopTask(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(clusterName),
		Task:    aws.String(taskArn),
		Reason:  aws.String("Test cleanup"),
	})
	require.NoError(t, err)
}