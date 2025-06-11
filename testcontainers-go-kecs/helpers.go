package kecs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// WaitForCluster waits for a cluster to reach the desired status
func WaitForCluster(ctx context.Context, client *ecs.Client, clusterName string, desiredStatus string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cluster %s to reach status %s", clusterName, desiredStatus)
		case <-ticker.C:
			output, err := client.DescribeClusters(ctx, &ecs.DescribeClustersInput{
				Clusters: []string{clusterName},
			})
			if err != nil {
				return fmt.Errorf("failed to describe cluster: %w", err)
			}

			if len(output.Clusters) > 0 && aws.ToString(output.Clusters[0].Status) == desiredStatus {
				return nil
			}
		}
	}
}

// WaitForService waits for a service to reach the desired status
func WaitForService(ctx context.Context, client *ecs.Client, clusterName, serviceName string, desiredStatus string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for service %s to reach status %s", serviceName, desiredStatus)
		case <-ticker.C:
			output, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
				Cluster:  aws.String(clusterName),
				Services: []string{serviceName},
			})
			if err != nil {
				return fmt.Errorf("failed to describe service: %w", err)
			}

			if len(output.Services) > 0 && aws.ToString(output.Services[0].Status) == desiredStatus {
				return nil
			}
		}
	}
}

// WaitForTask waits for a task to reach the desired status
func WaitForTask(ctx context.Context, client *ecs.Client, clusterName, taskArn string, desiredStatus string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for task %s to reach status %s", taskArn, desiredStatus)
		case <-ticker.C:
			output, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
				Cluster: aws.String(clusterName),
				Tasks:   []string{taskArn},
			})
			if err != nil {
				return fmt.Errorf("failed to describe task: %w", err)
			}

			if len(output.Tasks) > 0 && string(output.Tasks[0].LastStatus) == desiredStatus {
				return nil
			}
		}
	}
}

// CreateTestTaskDefinition creates a simple test task definition
func CreateTestTaskDefinition(ctx context.Context, client *ecs.Client, family string) (*types.TaskDefinition, error) {
	input := &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String(family),
		NetworkMode: types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{
			types.CompatibilityFargate,
		},
		Cpu:    aws.String("256"),
		Memory: aws.String("512"),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:  aws.String("test-container"),
				Image: aws.String("busybox:latest"),
				Command: []string{
					"sh",
					"-c",
					"echo 'Test container running' && sleep 3600",
				},
				Essential: aws.Bool(true),
				Memory:    aws.Int32(512),
			},
		},
	}

	output, err := client.RegisterTaskDefinition(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to register task definition: %w", err)
	}

	return output.TaskDefinition, nil
}

// CreateTestService creates a simple test service
func CreateTestService(ctx context.Context, client *ecs.Client, clusterName, serviceName, taskDefinition string) (*types.Service, error) {
	input := &ecs.CreateServiceInput{
		Cluster:        aws.String(clusterName),
		ServiceName:    aws.String(serviceName),
		TaskDefinition: aws.String(taskDefinition),
		DesiredCount:   aws.Int32(1),
		LaunchType:     types.LaunchTypeFargate,
		NetworkConfiguration: &types.NetworkConfiguration{
			AwsvpcConfiguration: &types.AwsVpcConfiguration{
				Subnets: []string{"subnet-12345"},
				SecurityGroups: []string{"sg-12345"},
				AssignPublicIp: types.AssignPublicIpEnabled,
			},
		},
	}

	output, err := client.CreateService(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	return output.Service, nil
}

// CleanupCluster deletes a cluster and all its resources
func CleanupCluster(ctx context.Context, client *ecs.Client, clusterName string) error {
	// List and stop all tasks
	listTasksOutput, err := client.ListTasks(ctx, &ecs.ListTasksInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	for _, taskArn := range listTasksOutput.TaskArns {
		_, _ = client.StopTask(ctx, &ecs.StopTaskInput{
			Cluster: aws.String(clusterName),
			Task:    aws.String(taskArn),
			Reason:  aws.String("Cleanup"),
		})
	}

	// List and delete all services
	listServicesOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, serviceArn := range listServicesOutput.ServiceArns {
		// Update service to 0 desired count
		_, _ = client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(serviceArn),
			DesiredCount: aws.Int32(0),
		})

		// Delete service
		_, _ = client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: aws.String(clusterName),
			Service: aws.String(serviceArn),
		})
	}

	// Delete cluster
	_, err = client.DeleteCluster(ctx, &ecs.DeleteClusterInput{
		Cluster: aws.String(clusterName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	return nil
}