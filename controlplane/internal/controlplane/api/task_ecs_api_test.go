package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/converters"
	"github.com/nandemo-ya/kecs/controlplane/internal/storage"
)

// MockTaskStore for task operations
type MockTaskStore struct {
	storage *MockStorage
}

func (m *MockTaskStore) Create(ctx context.Context, task *storage.Task) error {
	if m.storage.tasks == nil {
		m.storage.tasks = make(map[string]*storage.Task)
	}
	m.storage.tasks[task.ID] = task
	return nil
}

func (m *MockTaskStore) Get(ctx context.Context, cluster, taskID string) (*storage.Task, error) {
	// Try by ID first
	if task, exists := m.storage.tasks[taskID]; exists {
		return task, nil
	}
	// Try by ARN
	for _, task := range m.storage.tasks {
		if task.ARN == taskID {
			return task, nil
		}
	}
	return nil, errors.New("task not found")
}

func (m *MockTaskStore) List(ctx context.Context, cluster string, filters storage.TaskFilters) ([]*storage.Task, error) {
	var tasks []*storage.Task
	for _, task := range m.storage.tasks {
		if task.ClusterARN != cluster {
			continue
		}
		
		// Apply filters
		if filters.ServiceName != "" && task.StartedBy != fmt.Sprintf("ecs-svc/%s", filters.ServiceName) {
			continue
		}
		if filters.Family != "" && !hasTaskFamily(task.TaskDefinitionARN, filters.Family) {
			continue
		}
		if filters.LaunchType != "" && task.LaunchType != filters.LaunchType {
			continue
		}
		if filters.DesiredStatus != "" && task.DesiredStatus != filters.DesiredStatus {
			continue
		}
		if filters.StartedBy != "" && task.StartedBy != filters.StartedBy {
			continue
		}
		
		tasks = append(tasks, task)
	}
	
	// Apply limit
	if filters.MaxResults > 0 && len(tasks) > filters.MaxResults {
		tasks = tasks[:filters.MaxResults]
	}
	
	return tasks, nil
}

func (m *MockTaskStore) Update(ctx context.Context, task *storage.Task) error {
	if _, exists := m.storage.tasks[task.ID]; !exists {
		return errors.New("task not found")
	}
	m.storage.tasks[task.ID] = task
	return nil
}

func (m *MockTaskStore) Delete(ctx context.Context, cluster, taskID string) error {
	delete(m.storage.tasks, taskID)
	return nil
}

func (m *MockTaskStore) GetByARNs(ctx context.Context, arns []string) ([]*storage.Task, error) {
	var tasks []*storage.Task
	for _, arn := range arns {
		for _, task := range m.storage.tasks {
			if task.ARN == arn {
				tasks = append(tasks, task)
				break
			}
		}
	}
	return tasks, nil
}

func hasTaskFamily(taskDefArn, family string) bool {
	// Check if task definition ARN contains the family name
	if taskDefArn == "" || family == "" {
		return false
	}
	// Check various formats
	return taskDefArn == family || // exact match
		fmt.Sprintf("task-definition/%s:", family) == taskDefArn || // partial ARN
		strings.Contains(taskDefArn, fmt.Sprintf(":%s:", family)) || // contains family with colons
		strings.Contains(taskDefArn, fmt.Sprintf("/%s:", family)) // ARN format
}

// MockTaskManager for Kubernetes operations
type MockTaskManager struct {
	tasks map[string]*storage.Task
}

func (m *MockTaskManager) CreateTask(ctx context.Context, pod *corev1.Pod, task *storage.Task, secrets map[string]*converters.SecretInfo) error {
	if m.tasks == nil {
		m.tasks = make(map[string]*storage.Task)
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskManager) StopTask(ctx context.Context, cluster, taskID, reason string) error {
	if task, exists := m.tasks[taskID]; exists {
		now := time.Now()
		task.DesiredStatus = "STOPPED"
		task.StoppedReason = reason
		task.StoppingAt = &now
	}
	return nil
}

var _ = Describe("Task ECS API", func() {
	var (
		server *Server
		ctx    context.Context
	)

	BeforeEach(func() {
		mockStorage := NewMockStorage()
		mockStorage.tasks = make(map[string]*storage.Task)
		
		server = &Server{
			storage:     mockStorage,
			kindManager: nil,
			ecsAPI:      NewDefaultECSAPI(mockStorage, nil),
		}
		ctx = context.Background()
		
		// Pre-populate with test data
		// Add a default cluster
		cluster := &storage.Cluster{
			Name:       "default",
			ARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
			Status:     "ACTIVE",
			Region:     "ap-northeast-1",
			AccountID:  "123456789012",
		}
		mockStorage.clusters["default"] = cluster
		
		// Add a test task definition
		taskDef := &storage.TaskDefinition{
			ID:       "test-taskdef-1",
			ARN:      "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
			Family:   "nginx",
			Revision: 1,
			Status:   "ACTIVE",
			ContainerDefinitions: `[{"name":"nginx","image":"nginx:latest","memory":512}]`,
			Region:    "ap-northeast-1",
			AccountID: "123456789012",
		}
		mockStorage.taskDefinitions["nginx:1"] = taskDef
	})

	Describe("RunTask", func() {
		Context("when running a task", func() {
			It("should create a new task successfully", func() {
				taskDef := "nginx:1"
				req := &generated.RunTaskRequest{
					TaskDefinition: &taskDef,
				}
				
				resp, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(1))
				Expect(resp.Failures).To(BeEmpty())
				
				task := resp.Tasks[0]
				Expect(*task.TaskDefinitionArn).To(ContainSubstring("nginx:1"))
				Expect(*task.LastStatus).To(Equal("PROVISIONING"))
				Expect(*task.DesiredStatus).To(Equal("RUNNING"))
			})
			
			It("should create multiple tasks with count", func() {
				taskDef := "nginx"
				count := int32(3)
				req := &generated.RunTaskRequest{
					TaskDefinition: &taskDef,
					Count:          &count,
				}
				
				resp, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(3))
				Expect(resp.Failures).To(BeEmpty())
			})
			
			It("should fail when task definition not found", func() {
				taskDef := "non-existent:1"
				req := &generated.RunTaskRequest{
					TaskDefinition: &taskDef,
				}
				
				_, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("task definition not found"))
			})
			
			It("should fail without task definition", func() {
				req := &generated.RunTaskRequest{}
				
				_, err := server.ecsAPI.RunTask(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("taskDefinition is required"))
			})
		})
	})

	Describe("StopTask", func() {
		Context("when stopping a task", func() {
			BeforeEach(func() {
				// Add a running task
				task := &storage.Task{
					ID:                "task-123",
					ARN:               "arn:aws:ecs:ap-northeast-1:123456789012:task/default/task-123",
					ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
					TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
					LastStatus:        "RUNNING",
					DesiredStatus:     "RUNNING",
					LaunchType:        "EC2",
					Version:           1,
					CreatedAt:         time.Now(),
					Region:            "ap-northeast-1",
					AccountID:         "123456789012",
				}
				server.storage.(*MockStorage).tasks["task-123"] = task
			})
			
			It("should stop a running task", func() {
				taskID := "task-123"
				reason := "User requested stop"
				req := &generated.StopTaskRequest{
					Task:   &taskID,
					Reason: &reason,
				}
				
				resp, err := server.ecsAPI.StopTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Task).NotTo(BeNil())
				Expect(*resp.Task.TaskArn).To(ContainSubstring("task-123"))
			})
			
			It("should be idempotent when task already stopped", func() {
				// Set task as already stopped
				task := server.storage.(*MockStorage).tasks["task-123"]
				task.DesiredStatus = "STOPPED"
				
				taskID := "task-123"
				req := &generated.StopTaskRequest{
					Task: &taskID,
				}
				
				resp, err := server.ecsAPI.StopTask(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Task).NotTo(BeNil())
			})
			
			It("should fail when task not found", func() {
				taskID := "non-existent"
				req := &generated.StopTaskRequest{
					Task: &taskID,
				}
				
				_, err := server.ecsAPI.StopTask(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("task not found"))
			})
		})
	})

	Describe("DescribeTasks", func() {
		Context("when describing tasks", func() {
			BeforeEach(func() {
				// Add test tasks
				for i := 1; i <= 3; i++ {
					task := &storage.Task{
						ID:                fmt.Sprintf("task-%d", i),
						ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task/default/task-%d", i),
						ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
						TaskDefinitionARN: "arn:aws:ecs:ap-northeast-1:123456789012:task-definition/nginx:1",
						LastStatus:        "RUNNING",
						DesiredStatus:     "RUNNING",
						LaunchType:        "EC2",
						Version:           1,
						CreatedAt:         time.Now(),
						Region:            "ap-northeast-1",
						AccountID:         "123456789012",
						Tags:              `[{"key":"env","value":"test"}]`,
					}
					server.storage.(*MockStorage).tasks[task.ID] = task
				}
			})
			
			It("should describe multiple tasks", func() {
				req := &generated.DescribeTasksRequest{
					Tasks: []string{"task-1", "task-2"},
				}
				
				resp, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(2))
				Expect(resp.Failures).To(BeEmpty())
			})
			
			It("should include tags when requested", func() {
				req := &generated.DescribeTasksRequest{
					Tasks:   []string{"task-1"},
					Include: []generated.TaskField{generated.TaskFieldTags},
				}
				
				resp, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(1))
				Expect(resp.Tasks[0].Tags).To(HaveLen(1))
				Expect(string(*resp.Tasks[0].Tags[0].Key)).To(Equal("env"))
			})
			
			It("should report failures for non-existent tasks", func() {
				req := &generated.DescribeTasksRequest{
					Tasks: []string{"task-1", "non-existent"},
				}
				
				resp, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.Tasks).To(HaveLen(1))
				Expect(resp.Failures).To(HaveLen(1))
				Expect(*resp.Failures[0].Reason).To(Equal("MISSING"))
			})
			
			It("should fail without tasks", func() {
				req := &generated.DescribeTasksRequest{}
				
				_, err := server.ecsAPI.DescribeTasks(ctx, req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("tasks is required"))
			})
		})
	})

	Describe("ListTasks", func() {
		Context("when listing tasks", func() {
			BeforeEach(func() {
				// Add various test tasks
				tasks := []struct {
					id            string
					family        string
					launchType    string
					serviceName   string
					desiredStatus string
				}{
					{"task-web-1", "web", "FARGATE", "web-service", "RUNNING"},
					{"task-web-2", "web", "FARGATE", "web-service", "RUNNING"},
					{"task-api-1", "api", "EC2", "api-service", "RUNNING"},
					{"task-api-2", "api", "EC2", "api-service", "STOPPED"},
					{"task-batch-1", "batch", "EC2", "", "RUNNING"},
				}
				
				for _, t := range tasks {
					task := &storage.Task{
						ID:                t.id,
						ARN:               fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task/default/%s", t.id),
						ClusterARN:        "arn:aws:ecs:ap-northeast-1:123456789012:cluster/default",
						TaskDefinitionARN: fmt.Sprintf("arn:aws:ecs:ap-northeast-1:123456789012:task-definition/%s:1", t.family),
						LastStatus:        t.desiredStatus,
						DesiredStatus:     t.desiredStatus,
						LaunchType:        t.launchType,
						Version:           1,
						CreatedAt:         time.Now(),
						Region:            "ap-northeast-1",
						AccountID:         "123456789012",
					}
					if t.serviceName != "" {
						task.StartedBy = fmt.Sprintf("ecs-svc/%s", t.serviceName)
					}
					server.storage.(*MockStorage).tasks[t.id] = task
				}
			})
			
			It("should list all tasks", func() {
				req := &generated.ListTasksRequest{}
				
				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(5))
			})
			
			It("should filter by service name", func() {
				serviceName := "web-service"
				req := &generated.ListTasksRequest{
					ServiceName: &serviceName,
				}
				
				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
				for _, arn := range resp.TaskArns {
					Expect(arn).To(ContainSubstring("task-web"))
				}
			})
			
			It("should filter by family", func() {
				family := "api"
				req := &generated.ListTasksRequest{
					Family: &family,
				}
				
				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
				for _, arn := range resp.TaskArns {
					Expect(arn).To(ContainSubstring("task-api"))
				}
			})
			
			It("should filter by launch type", func() {
				launchType := generated.LaunchTypeFargate
				req := &generated.ListTasksRequest{
					LaunchType: &launchType,
				}
				
				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
			})
			
			It("should filter by desired status", func() {
				status := generated.DesiredStatusRunning
				req := &generated.ListTasksRequest{
					DesiredStatus: &status,
				}
				
				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(4))
			})
			
			It("should respect max results", func() {
				maxResults := int32(2)
				req := &generated.ListTasksRequest{
					MaxResults: &maxResults,
				}
				
				resp, err := server.ecsAPI.ListTasks(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				
				Expect(resp).NotTo(BeNil())
				Expect(resp.TaskArns).To(HaveLen(2))
				Expect(resp.NextToken).NotTo(BeNil())
			})
		})
	})
})