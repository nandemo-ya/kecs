package examples_test

import (
	"context"
	"fmt"
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

func TestAdvancedScenarios(t *testing.T) {
	ctx := context.Background()

	// Start KECS with custom configuration
	container, err := kecs.StartContainer(ctx,
		kecs.WithTestMode(),
		kecs.WithRegion("us-east-1"),
		kecs.WithWaitTimeout(2*time.Minute),
		kecs.WithEnv(map[string]string{
			"LOG_LEVEL": "info",
		}),
	)
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Create main cluster
	clusterName := "advanced-test-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
		Tags: []types.Tag{
			{Key: aws.String("Environment"), Value: aws.String("test")},
			{Key: aws.String("Team"), Value: aws.String("platform")},
		},
		Settings: []types.ClusterSetting{
			{
				Name:  types.ClusterSettingNameContainerInsights,
				Value: aws.String("enabled"),
			},
		},
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	t.Run("MicroservicesWithServiceDiscovery", func(t *testing.T) {
		// Register API service task definition
		apiTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			Family:      aws.String("api-service"),
			NetworkMode: types.NetworkModeAwsvpc,
			RequiresCompatibilities: []types.Compatibility{
				types.CompatibilityFargate,
			},
			Cpu:    aws.String("256"),
			Memory: aws.String("512"),
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:      aws.String("api"),
					Image:     aws.String("node:14-alpine"),
					Memory:    aws.Int32(512),
					Essential: aws.Bool(true),
					Command: []string{
						"sh", "-c",
						"echo 'API server running on port 3000' && node -e 'require(\"http\").createServer((req,res)=>res.end(\"API v1\")).listen(3000)'",
					},
					PortMappings: []types.PortMapping{
						{
							ContainerPort: aws.Int32(3000),
							Protocol:      types.TransportProtocolTcp,
						},
					},
					Environment: []types.KeyValuePair{
						{Name: aws.String("SERVICE_NAME"), Value: aws.String("api")},
						{Name: aws.String("VERSION"), Value: aws.String("1.0.0")},
					},
				},
			},
		})
		require.NoError(t, err)

		// Register worker service task definition
		workerTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			Family:      aws.String("worker-service"),
			NetworkMode: types.NetworkModeAwsvpc,
			RequiresCompatibilities: []types.Compatibility{
				types.CompatibilityFargate,
			},
			Cpu:    aws.String("256"),
			Memory: aws.String("512"),
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:      aws.String("worker"),
					Image:     aws.String("busybox:latest"),
					Memory:    aws.Int32(512),
					Essential: aws.Bool(true),
					Command: []string{
						"sh", "-c",
						"while true; do echo 'Processing jobs...'; sleep 10; done",
					},
					Environment: []types.KeyValuePair{
						{Name: aws.String("SERVICE_NAME"), Value: aws.String("worker")},
						{Name: aws.String("API_ENDPOINT"), Value: aws.String("http://api.local:3000")},
					},
				},
			},
		})
		require.NoError(t, err)

		// Create API service
		apiService, err := client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String("api-service"),
			TaskDefinition: apiTaskDef.TaskDefinition.TaskDefinitionArn,
			DesiredCount:   aws.Int32(2),
			LaunchType:     types.LaunchTypeFargate,
			NetworkConfiguration: &types.NetworkConfiguration{
				AwsvpcConfiguration: &types.AwsVpcConfiguration{
					Subnets:        []string{"subnet-12345"},
					SecurityGroups: []string{"sg-12345"},
					AssignPublicIp: types.AssignPublicIpEnabled,
				},
			},
			ServiceRegistries: []types.ServiceRegistry{
				{
					RegistryArn: aws.String("arn:aws:servicediscovery:us-east-1:123456789012:service/srv-123"),
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "api-service", aws.ToString(apiService.Service.ServiceName))

		// Create worker service
		workerService, err := client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String("worker-service"),
			TaskDefinition: workerTaskDef.TaskDefinition.TaskDefinitionArn,
			DesiredCount:   aws.Int32(3),
			LaunchType:     types.LaunchTypeFargate,
			NetworkConfiguration: &types.NetworkConfiguration{
				AwsvpcConfiguration: &types.AwsVpcConfiguration{
					Subnets:        []string{"subnet-12345"},
					SecurityGroups: []string{"sg-12345"},
					AssignPublicIp: types.AssignPublicIpEnabled,
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "worker-service", aws.ToString(workerService.Service.ServiceName))

		// List all services
		listOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
			Cluster: aws.String(clusterName),
		})
		require.NoError(t, err)
		assert.Len(t, listOutput.ServiceArns, 2)

		// Clean up services
		for _, serviceName := range []string{"api-service", "worker-service"} {
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
		}
	})

	t.Run("BatchJobProcessing", func(t *testing.T) {
		// Register batch job task definition
		batchTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
			Family:      aws.String("batch-processor"),
			NetworkMode: types.NetworkModeBridge,
			ContainerDefinitions: []types.ContainerDefinition{
				{
					Name:      aws.String("processor"),
					Image:     aws.String("busybox:latest"),
					Memory:    aws.Int32(256),
					Essential: aws.Bool(true),
					Command: []string{
						"sh", "-c",
						"echo 'Starting batch job'; for i in $(seq 1 5); do echo \"Processing item $i\"; sleep 2; done; echo 'Batch job completed'",
					},
					LogConfiguration: &types.LogConfiguration{
						LogDriver: types.LogDriverAwslogs,
						Options: map[string]string{
							"awslogs-group":         "/ecs/batch-processor",
							"awslogs-region":        "us-east-1",
							"awslogs-stream-prefix": "batch",
						},
					},
				},
			},
		})
		require.NoError(t, err)

		// Run multiple batch jobs
		var taskArns []string
		for i := 0; i < 3; i++ {
			runOutput, err := client.RunTask(ctx, &ecs.RunTaskInput{
				Cluster:        aws.String(clusterName),
				TaskDefinition: batchTaskDef.TaskDefinition.TaskDefinitionArn,
				Count:          aws.Int32(1),
				Overrides: &types.TaskOverride{
					ContainerOverrides: []types.ContainerOverride{
						{
							Name: aws.String("processor"),
							Environment: []types.KeyValuePair{
								{Name: aws.String("JOB_ID"), Value: aws.String(fmt.Sprintf("job-%d", i+1))},
								{Name: aws.String("BATCH_SIZE"), Value: aws.String("100")},
							},
						},
					},
				},
			})
			require.NoError(t, err)
			require.Len(t, runOutput.Tasks, 1)
			taskArns = append(taskArns, aws.ToString(runOutput.Tasks[0].TaskArn))
		}

		// Monitor batch jobs
		completedJobs := 0
		timeout := time.After(30 * time.Second)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for completedJobs < len(taskArns) {
			select {
			case <-timeout:
				t.Fatal("Timeout waiting for batch jobs to complete")
			case <-ticker.C:
				describeOutput, err := client.DescribeTasks(ctx, &ecs.DescribeTasksInput{
					Cluster: aws.String(clusterName),
					Tasks:   taskArns,
				})
				require.NoError(t, err)

				completedJobs = 0
				for _, task := range describeOutput.Tasks {
					if string(task.LastStatus) == "STOPPED" {
						completedJobs++
						t.Logf("Task %s completed with exit code: %d",
							aws.ToString(task.TaskArn),
							aws.ToInt32(task.Containers[0].ExitCode))
					}
				}
			}
		}

		assert.Equal(t, len(taskArns), completedJobs)
	})
}

func TestComplexServiceDependencies(t *testing.T) {
	ctx := context.Background()

	// Start KECS with detailed logging
	logConsumer := testcontainers.LogConsumerFunc(func(log testcontainers.Log) {
		t.Logf("[KECS] %s", log.Content)
	})

	container, err := kecs.StartContainer(ctx,
		kecs.WithTestMode(),
		kecs.WithLogConsumer(logConsumer),
	)
	require.NoError(t, err)
	defer container.Cleanup(ctx)

	// Create ECS client
	client, err := container.NewECSClient(ctx)
	require.NoError(t, err)

	// Create cluster
	clusterName := "complex-deps-cluster"
	_, err = client.CreateCluster(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(clusterName),
	})
	require.NoError(t, err)
	defer kecs.CleanupCluster(ctx, client, clusterName)

	// Create a complex application stack: Database -> API -> Frontend
	
	// 1. Database service
	dbTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("database"),
		NetworkMode: types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{
			types.CompatibilityFargate,
		},
		Cpu:    aws.String("512"),
		Memory: aws.String("1024"),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:      aws.String("postgres"),
				Image:     aws.String("postgres:13-alpine"),
				Memory:    aws.Int32(1024),
				Essential: aws.Bool(true),
				PortMappings: []types.PortMapping{
					{
						ContainerPort: aws.Int32(5432),
						Protocol:      types.TransportProtocolTcp,
					},
				},
				Environment: []types.KeyValuePair{
					{Name: aws.String("POSTGRES_DB"), Value: aws.String("testdb")},
					{Name: aws.String("POSTGRES_USER"), Value: aws.String("testuser")},
					{Name: aws.String("POSTGRES_PASSWORD"), Value: aws.String("testpass")},
				},
				HealthCheck: &types.HealthCheck{
					Command: []string{
						"CMD-SHELL",
						"pg_isready -U testuser",
					},
					Interval:    aws.Int32(30),
					Timeout:     aws.Int32(5),
					Retries:     aws.Int32(3),
					StartPeriod: aws.Int32(60),
				},
			},
		},
	})
	require.NoError(t, err)

	// 2. API service with dependency on database
	apiTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("api"),
		NetworkMode: types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{
			types.CompatibilityFargate,
		},
		Cpu:    aws.String("256"),
		Memory: aws.String("512"),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:      aws.String("api"),
				Image:     aws.String("node:14-alpine"),
				Memory:    aws.Int32(512),
				Essential: aws.Bool(true),
				Command: []string{
					"sh", "-c",
					"echo 'Waiting for database...' && sleep 10 && echo 'API server starting' && node -e 'require(\"http\").createServer((req,res)=>res.end(JSON.stringify({status:\"ok\",db:\"connected\"}))).listen(8080)'",
				},
				PortMappings: []types.PortMapping{
					{
						ContainerPort: aws.Int32(8080),
						Protocol:      types.TransportProtocolTcp,
					},
				},
				Environment: []types.KeyValuePair{
					{Name: aws.String("DATABASE_HOST"), Value: aws.String("database.local")},
					{Name: aws.String("DATABASE_PORT"), Value: aws.String("5432")},
					{Name: aws.String("DATABASE_NAME"), Value: aws.String("testdb")},
				},
				DependsOn: []types.ContainerDependency{
					{
						ContainerName: aws.String("postgres"),
						Condition:     types.ContainerConditionHealthy,
					},
				},
			},
		},
	})
	require.NoError(t, err)

	// 3. Frontend service with dependency on API
	frontendTaskDef, err := client.RegisterTaskDefinition(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:      aws.String("frontend"),
		NetworkMode: types.NetworkModeAwsvpc,
		RequiresCompatibilities: []types.Compatibility{
			types.CompatibilityFargate,
		},
		Cpu:    aws.String("256"),
		Memory: aws.String("512"),
		ContainerDefinitions: []types.ContainerDefinition{
			{
				Name:      aws.String("nginx"),
				Image:     aws.String("nginx:alpine"),
				Memory:    aws.Int32(512),
				Essential: aws.Bool(true),
				PortMappings: []types.PortMapping{
					{
						ContainerPort: aws.Int32(80),
						Protocol:      types.TransportProtocolTcp,
					},
				},
				Environment: []types.KeyValuePair{
					{Name: aws.String("API_URL"), Value: aws.String("http://api.local:8080")},
				},
			},
		},
	})
	require.NoError(t, err)

	// Create services in dependency order
	services := []struct {
		name           string
		taskDefinition string
		desiredCount   int32
	}{
		{"database-service", dbTaskDef.TaskDefinition.TaskDefinitionArn, 1},
		{"api-service", apiTaskDef.TaskDefinition.TaskDefinitionArn, 2},
		{"frontend-service", frontendTaskDef.TaskDefinition.TaskDefinitionArn, 3},
	}

	for _, svc := range services {
		_, err = client.CreateService(ctx, &ecs.CreateServiceInput{
			Cluster:        aws.String(clusterName),
			ServiceName:    aws.String(svc.name),
			TaskDefinition: aws.String(svc.taskDefinition),
			DesiredCount:   aws.Int32(svc.desiredCount),
			LaunchType:     types.LaunchTypeFargate,
			NetworkConfiguration: &types.NetworkConfiguration{
				AwsvpcConfiguration: &types.AwsVpcConfiguration{
					Subnets:        []string{"subnet-12345"},
					SecurityGroups: []string{"sg-12345"},
					AssignPublicIp: types.AssignPublicIpEnabled,
				},
			},
		})
		require.NoError(t, err)

		// Wait for service to stabilize
		err = kecs.WaitForService(ctx, client, clusterName, svc.name, "ACTIVE", 30*time.Second)
		require.NoError(t, err)
	}

	// Verify all services are running
	listOutput, err := client.ListServices(ctx, &ecs.ListServicesInput{
		Cluster: aws.String(clusterName),
	})
	require.NoError(t, err)
	assert.Len(t, listOutput.ServiceArns, 3)

	// Get service details
	describeOutput, err := client.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(clusterName),
		Services: []string{"database-service", "api-service", "frontend-service"},
	})
	require.NoError(t, err)
	require.Len(t, describeOutput.Services, 3)

	for _, service := range describeOutput.Services {
		t.Logf("Service %s: desired=%d, running=%d, pending=%d",
			aws.ToString(service.ServiceName),
			aws.ToInt32(service.DesiredCount),
			aws.ToInt32(service.RunningCount),
			aws.ToInt32(service.PendingCount))
		assert.Equal(t, "ACTIVE", aws.ToString(service.Status))
	}

	// Clean up in reverse order
	for i := len(services) - 1; i >= 0; i-- {
		_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
			Cluster:      aws.String(clusterName),
			Service:      aws.String(services[i].name),
			DesiredCount: aws.Int32(0),
		})
		require.NoError(t, err)

		_, err = client.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: aws.String(clusterName),
			Service: aws.String(services[i].name),
		})
		require.NoError(t, err)
	}
}