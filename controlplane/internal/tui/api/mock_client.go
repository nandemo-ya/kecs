// Copyright 2025 The KECS Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MockClient implements the Client interface with mock data
type MockClient struct {
	instances []Instance
	clusters  map[string][]Cluster
	services  map[string][]Service
	tasks     map[string][]Task
}

// NewMockClient creates a new mock API client
func NewMockClient() *MockClient {
	client := &MockClient{
		clusters: make(map[string][]Cluster),
		services: make(map[string][]Service),
		tasks:    make(map[string][]Task),
	}
	
	// Initialize with mock data
	client.initMockData()
	return client
}

func (c *MockClient) initMockData() {
	// Mock instances
	c.instances = []Instance{
		{
			Name:      "dev",
			Status:    "running",
			Clusters:  2,
			Services:  5,
			Tasks:     12,
			APIPort:   8080,
			AdminPort: 8081,
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			Name:      "staging",
			Status:    "running",
			Clusters:  1,
			Services:  3,
			Tasks:     8,
			APIPort:   8090,
			AdminPort: 8091,
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
	}

	// Mock clusters for dev instance
	c.clusters["dev"] = []Cluster{
		{
			ClusterArn:                "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			ClusterName:               "default",
			Status:                    "ACTIVE",
			RegisteredContainerInstancesCount: 3,
			RunningTasksCount:         8,
			PendingTasksCount:         0,
			ActiveServicesCount:       3,
		},
		{
			ClusterArn:                "arn:aws:ecs:us-east-1:123456789012:cluster/production",
			ClusterName:               "production",
			Status:                    "ACTIVE",
			RegisteredContainerInstancesCount: 2,
			RunningTasksCount:         4,
			PendingTasksCount:         0,
			ActiveServicesCount:       2,
		},
	}

	// Mock services for dev/default cluster
	c.services["dev/default"] = []Service{
		{
			ServiceArn:     "arn:aws:ecs:us-east-1:123456789012:service/default/web-service",
			ServiceName:    "web-service",
			ClusterArn:     "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			Status:         "ACTIVE",
			DesiredCount:   3,
			RunningCount:   3,
			PendingCount:   0,
			TaskDefinition: "web-app:1",
			CreatedAt:      time.Now().Add(-12 * time.Hour),
		},
		{
			ServiceArn:     "arn:aws:ecs:us-east-1:123456789012:service/default/api-service",
			ServiceName:    "api-service",
			ClusterArn:     "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			Status:         "ACTIVE",
			DesiredCount:   2,
			RunningCount:   2,
			PendingCount:   0,
			TaskDefinition: "api:2",
			CreatedAt:      time.Now().Add(-6 * time.Hour),
		},
		{
			ServiceArn:     "arn:aws:ecs:us-east-1:123456789012:service/default/worker-service",
			ServiceName:    "worker-service",
			ClusterArn:     "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			Status:         "ACTIVE",
			DesiredCount:   3,
			RunningCount:   3,
			PendingCount:   0,
			TaskDefinition: "worker:1",
			CreatedAt:      time.Now().Add(-3 * time.Hour),
		},
	}

	// Mock tasks for dev/default/web-service
	c.tasks["dev/default/web-service"] = []Task{
		{
			TaskArn:           "arn:aws:ecs:us-east-1:123456789012:task/default/1234567890123456789",
			ClusterArn:        "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/web-app:1",
			ServiceName:       "web-service",
			LastStatus:        "RUNNING",
			DesiredStatus:     "RUNNING",
			HealthStatus:      "HEALTHY",
			Cpu:               "256",
			Memory:            "512",
			CreatedAt:         time.Now().Add(-2 * time.Hour),
			StartedAt:         timePtr(time.Now().Add(-119 * time.Minute)),
		},
		{
			TaskArn:           "arn:aws:ecs:us-east-1:123456789012:task/default/2234567890123456789",
			ClusterArn:        "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/web-app:1",
			ServiceName:       "web-service",
			LastStatus:        "RUNNING",
			DesiredStatus:     "RUNNING",
			HealthStatus:      "HEALTHY",
			Cpu:               "256",
			Memory:            "512",
			CreatedAt:         time.Now().Add(-1 * time.Hour),
			StartedAt:         timePtr(time.Now().Add(-59 * time.Minute)),
		},
		{
			TaskArn:           "arn:aws:ecs:us-east-1:123456789012:task/default/3234567890123456789",
			ClusterArn:        "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/web-app:1",
			ServiceName:       "web-service",
			LastStatus:        "PENDING",
			DesiredStatus:     "RUNNING",
			Cpu:               "256",
			Memory:            "512",
			CreatedAt:         time.Now().Add(-5 * time.Minute),
		},
	}
}

// Instance operations

func (c *MockClient) ListInstances(ctx context.Context) ([]Instance, error) {
	return c.instances, nil
}

func (c *MockClient) GetInstance(ctx context.Context, name string) (*Instance, error) {
	for _, inst := range c.instances {
		if inst.Name == name {
			return &inst, nil
		}
	}
	return nil, fmt.Errorf("instance not found: %s", name)
}

func (c *MockClient) CreateInstance(ctx context.Context, opts CreateInstanceOptions) (*Instance, error) {
	// Check for duplicate name
	for _, inst := range c.instances {
		if inst.Name == opts.Name {
			return nil, fmt.Errorf("instance with name '%s' already exists", opts.Name)
		}
	}
	
	// Check for port conflicts
	for _, inst := range c.instances {
		if inst.APIPort == opts.APIPort {
			return nil, fmt.Errorf("API port %d is already in use by instance '%s'", opts.APIPort, inst.Name)
		}
		if inst.AdminPort == opts.AdminPort {
			return nil, fmt.Errorf("Admin port %d is already in use by instance '%s'", opts.AdminPort, inst.Name)
		}
	}
	
	instance := Instance{
		Name:      opts.Name,
		Status:    "pending",
		Clusters:  0,
		Services:  0,
		Tasks:     0,
		APIPort:   opts.APIPort,
		AdminPort: opts.AdminPort,
		CreatedAt: time.Now(),
	}
	c.instances = append(c.instances, instance)
	
	// Simulate instance becoming ready after creation
	go func() {
		time.Sleep(2 * time.Second)
		for i := range c.instances {
			if c.instances[i].Name == opts.Name {
				c.instances[i].Status = "running"
				break
			}
		}
	}()
	
	return &instance, nil
}

func (c *MockClient) DeleteInstance(ctx context.Context, name string) error {
	for i, inst := range c.instances {
		if inst.Name == name {
			c.instances = append(c.instances[:i], c.instances[i+1:]...)
			delete(c.clusters, name)
			// Clean up related data
			for key := range c.services {
				// Key format is "instance/cluster"
				if strings.HasPrefix(key, name+"/") {
					delete(c.services, key)
				}
			}
			for key := range c.tasks {
				// Key format is "instance/cluster/service"
				if strings.HasPrefix(key, name+"/") {
					delete(c.tasks, key)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("instance not found: %s", name)
}

func (c *MockClient) GetInstanceLogs(ctx context.Context, name string, follow bool) (<-chan LogEntry, error) {
	ch := make(chan LogEntry)
	
	go func() {
		defer close(ch)
		
		// Send some mock log entries
		for i := 0; i < 10; i++ {
			select {
			case <-ctx.Done():
				return
			case ch <- LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   fmt.Sprintf("Mock log entry %d for instance %s", i, name),
			}:
			}
			
			if follow {
				time.Sleep(1 * time.Second)
			}
		}
	}()
	
	return ch, nil
}

// ECS Cluster operations

func (c *MockClient) ListClusters(ctx context.Context, instanceName string) ([]string, error) {
	clusters := c.clusters[instanceName]
	arns := make([]string, len(clusters))
	for i, cluster := range clusters {
		arns[i] = cluster.ClusterArn
	}
	return arns, nil
}

func (c *MockClient) DescribeClusters(ctx context.Context, instanceName string, clusterNames []string) ([]Cluster, error) {
	return c.clusters[instanceName], nil
}

func (c *MockClient) CreateCluster(ctx context.Context, instanceName, clusterName string) (*Cluster, error) {
	cluster := Cluster{
		ClusterArn:                fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", clusterName),
		ClusterName:               clusterName,
		Status:                    "ACTIVE",
		RegisteredContainerInstancesCount: 0,
		RunningTasksCount:         0,
		PendingTasksCount:         0,
		ActiveServicesCount:       0,
	}
	c.clusters[instanceName] = append(c.clusters[instanceName], cluster)
	return &cluster, nil
}

func (c *MockClient) DeleteCluster(ctx context.Context, instanceName, clusterName string) error {
	clusters := c.clusters[instanceName]
	for i, cluster := range clusters {
		if cluster.ClusterName == clusterName {
			c.clusters[instanceName] = append(clusters[:i], clusters[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("cluster not found: %s", clusterName)
}

// ECS Service operations

func (c *MockClient) ListServices(ctx context.Context, instanceName, clusterName string) ([]string, error) {
	key := fmt.Sprintf("%s/%s", instanceName, clusterName)
	services := c.services[key]
	arns := make([]string, len(services))
	for i, service := range services {
		arns[i] = service.ServiceArn
	}
	return arns, nil
}

func (c *MockClient) DescribeServices(ctx context.Context, instanceName, clusterName string, serviceNames []string) ([]Service, error) {
	key := fmt.Sprintf("%s/%s", instanceName, clusterName)
	return c.services[key], nil
}

func (c *MockClient) CreateService(ctx context.Context, instanceName, clusterName string, service Service) (*Service, error) {
	key := fmt.Sprintf("%s/%s", instanceName, clusterName)
	service.ServiceArn = fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:service/%s/%s", clusterName, service.ServiceName)
	service.Status = "ACTIVE"
	service.CreatedAt = time.Now()
	c.services[key] = append(c.services[key], service)
	return &service, nil
}

func (c *MockClient) UpdateService(ctx context.Context, instanceName, clusterName string, service Service) (*Service, error) {
	key := fmt.Sprintf("%s/%s", instanceName, clusterName)
	services := c.services[key]
	for i, s := range services {
		if s.ServiceName == service.ServiceName {
			services[i] = service
			return &service, nil
		}
	}
	return nil, fmt.Errorf("service not found: %s", service.ServiceName)
}

func (c *MockClient) DeleteService(ctx context.Context, instanceName, clusterName, serviceName string) error {
	key := fmt.Sprintf("%s/%s", instanceName, clusterName)
	services := c.services[key]
	for i, service := range services {
		if service.ServiceName == serviceName {
			c.services[key] = append(services[:i], services[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("service not found: %s", serviceName)
}

// ECS Task operations

func (c *MockClient) ListTasks(ctx context.Context, instanceName, clusterName string, serviceName string) ([]string, error) {
	key := fmt.Sprintf("%s/%s/%s", instanceName, clusterName, serviceName)
	tasks := c.tasks[key]
	arns := make([]string, len(tasks))
	for i, task := range tasks {
		arns[i] = task.TaskArn
	}
	return arns, nil
}

func (c *MockClient) DescribeTasks(ctx context.Context, instanceName, clusterName string, taskArns []string) ([]Task, error) {
	// Return all tasks for simplicity in mock
	var allTasks []Task
	for _, tasks := range c.tasks {
		allTasks = append(allTasks, tasks...)
	}
	return allTasks, nil
}

func (c *MockClient) RunTask(ctx context.Context, instanceName, clusterName string, taskDef string) (*Task, error) {
	task := Task{
		TaskArn:           fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task/%s/%d", clusterName, time.Now().Unix()),
		ClusterArn:        fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:cluster/%s", clusterName),
		TaskDefinitionArn: fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task-definition/%s", taskDef),
		LastStatus:        "PENDING",
		DesiredStatus:     "RUNNING",
		CreatedAt:         time.Now(),
	}
	return &task, nil
}

func (c *MockClient) StopTask(ctx context.Context, instanceName, clusterName, taskArn string) error {
	// Mock implementation
	return nil
}

// Task Definition operations

func (c *MockClient) ListTaskDefinitions(ctx context.Context, instanceName string) ([]string, error) {
	return []string{
		"arn:aws:ecs:us-east-1:123456789012:task-definition/web-app:1",
		"arn:aws:ecs:us-east-1:123456789012:task-definition/api:2",
		"arn:aws:ecs:us-east-1:123456789012:task-definition/worker:1",
	}, nil
}

func (c *MockClient) RegisterTaskDefinition(ctx context.Context, instanceName string, taskDef interface{}) (string, error) {
	return fmt.Sprintf("arn:aws:ecs:us-east-1:123456789012:task-definition/new-task:%d", time.Now().Unix()), nil
}

func (c *MockClient) DeregisterTaskDefinition(ctx context.Context, instanceName string, taskDefArn string) error {
	return nil
}

// Health check

func (c *MockClient) HealthCheck(ctx context.Context, instanceName string) error {
	for _, inst := range c.instances {
		if inst.Name == instanceName {
			if inst.Status == "running" {
				return nil
			}
			return fmt.Errorf("instance %s is not running: status=%s", instanceName, inst.Status)
		}
	}
	return fmt.Errorf("instance not found: %s", instanceName)
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}