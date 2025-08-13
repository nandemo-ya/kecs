package cloudwatch_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/fake"

	cloudwatchlogsapi "github.com/nandemo-ya/kecs/controlplane/internal/cloudwatchlogs/generated"
	kecsCloudWatch "github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// mockCloudWatchLogsClient is a mock implementation of CloudWatchLogsClient
type mockCloudWatchLogsClient struct {
	logGroups  map[string]bool
	logStreams map[string]map[string]bool // groupName -> streamName -> exists
}

func newMockCloudWatchLogsClient() *mockCloudWatchLogsClient {
	return &mockCloudWatchLogsClient{
		logGroups:  make(map[string]bool),
		logStreams: make(map[string]map[string]bool),
	}
}

func (m *mockCloudWatchLogsClient) CreateLogGroup(ctx context.Context, params *cloudwatchlogsapi.CreateLogGroupRequest) (*cloudwatchlogsapi.Unit, error) {
	groupName := params.LogGroupName
	if m.logGroups[groupName] {
		return nil, fmt.Errorf("ResourceAlreadyExistsException: log group already exists")
	}
	m.logGroups[groupName] = true
	m.logStreams[groupName] = make(map[string]bool)
	return &cloudwatchlogsapi.Unit{}, nil
}

func (m *mockCloudWatchLogsClient) DeleteLogGroup(ctx context.Context, params *cloudwatchlogsapi.DeleteLogGroupRequest) (*cloudwatchlogsapi.Unit, error) {
	groupName := params.LogGroupName
	if !m.logGroups[groupName] {
		return nil, fmt.Errorf("ResourceNotFoundException: log group not found")
	}
	delete(m.logGroups, groupName)
	delete(m.logStreams, groupName)
	return &cloudwatchlogsapi.Unit{}, nil
}

func (m *mockCloudWatchLogsClient) CreateLogStream(ctx context.Context, params *cloudwatchlogsapi.CreateLogStreamRequest) (*cloudwatchlogsapi.Unit, error) {
	groupName := params.LogGroupName
	streamName := params.LogStreamName

	if !m.logGroups[groupName] {
		return nil, fmt.Errorf("ResourceNotFoundException: log group not found")
	}

	if m.logStreams[groupName][streamName] {
		return nil, fmt.Errorf("ResourceAlreadyExistsException: log stream already exists")
	}

	m.logStreams[groupName][streamName] = true
	return &cloudwatchlogsapi.Unit{}, nil
}

func (m *mockCloudWatchLogsClient) PutRetentionPolicy(ctx context.Context, params *cloudwatchlogsapi.PutRetentionPolicyRequest) (*cloudwatchlogsapi.Unit, error) {
	groupName := params.LogGroupName
	if !m.logGroups[groupName] {
		return nil, fmt.Errorf("ResourceNotFoundException: log group not found")
	}
	return &cloudwatchlogsapi.Unit{}, nil
}

// mockLocalStackManager is a mock implementation of localstack.Manager
type mockLocalStackManager struct{}

func (m *mockLocalStackManager) Start(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) Stop(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) Restart(ctx context.Context) error {
	return nil
}

func (m *mockLocalStackManager) IsRunning() bool {
	return true
}

func (m *mockLocalStackManager) IsHealthy() bool {
	return true
}

func (m *mockLocalStackManager) GetEndpoint() (string, error) {
	return "http://localhost:4566", nil
}

func (m *mockLocalStackManager) GetStatus() (*localstack.Status, error) {
	return &localstack.Status{
		Running: true,
		Healthy: true,
	}, nil
}

func (m *mockLocalStackManager) EnableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) DisableService(service string) error {
	return nil
}

func (m *mockLocalStackManager) GetEnabledServices() ([]string, error) {
	return []string{"logs", "s3"}, nil
}

func (m *mockLocalStackManager) GetConfig() *localstack.Config {
	return &localstack.Config{
		Enabled: true,
	}
}

func (m *mockLocalStackManager) GetContainer() *localstack.LocalStackContainer {
	return nil
}

func (m *mockLocalStackManager) UpdateServices(services []string) error {
	return nil
}

func (m *mockLocalStackManager) GetServiceEndpoint(service string) (string, error) {
	return "http://localhost:4566", nil
}

func (m *mockLocalStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return nil
}

func (m *mockLocalStackManager) CheckServiceHealth(service string) error {
	return nil
}

var _ = Describe("CloudWatch Integration", func() {
	var (
		integration       kecsCloudWatch.Integration
		kubeClient        *fake.Clientset
		localstackManager localstack.Manager
		config            *kecsCloudWatch.Config
		logsClient        *mockCloudWatchLogsClient
	)

	BeforeEach(func() {
		kubeClient = fake.NewSimpleClientset()
		localstackManager = &mockLocalStackManager{}
		config = &kecsCloudWatch.Config{
			LocalStackEndpoint: "http://localhost:4566",
			LogGroupPrefix:     "/ecs/",
			RetentionDays:      7,
			KubeNamespace:      "default",
		}

		logsClient = newMockCloudWatchLogsClient()

		// Use the test constructor with mocked client
		integration = kecsCloudWatch.NewIntegrationWithClient(
			kubeClient,
			localstackManager,
			config,
			logsClient,
		)
	})

	Describe("CreateLogGroup", func() {
		It("should create a log group with prefix", func() {
			err := integration.CreateLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())
			Expect(logsClient.logGroups["/ecs/my-app"]).To(BeTrue())
		})

		It("should not error if log group already exists", func() {
			// Create first time
			err := integration.CreateLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())

			// Create again
			err = integration.CreateLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should preserve existing prefix", func() {
			err := integration.CreateLogGroup("/ecs/already-prefixed")
			Expect(err).NotTo(HaveOccurred())
			Expect(logsClient.logGroups["/ecs/already-prefixed"]).To(BeTrue())
			Expect(logsClient.logGroups["/ecs//ecs/already-prefixed"]).To(BeFalse())
		})
	})

	Describe("CreateLogStream", func() {
		It("should create a log stream in existing group", func() {
			// Create log group first
			err := integration.CreateLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())

			// Create log stream
			err = integration.CreateLogStream("my-app", "container-1/task-123")
			Expect(err).NotTo(HaveOccurred())
			Expect(logsClient.logStreams["/ecs/my-app"]["container-1/task-123"]).To(BeTrue())
		})

		It("should not error if log stream already exists", func() {
			// Create log group
			err := integration.CreateLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())

			// Create stream first time
			err = integration.CreateLogStream("my-app", "container-1/task-123")
			Expect(err).NotTo(HaveOccurred())

			// Create again
			err = integration.CreateLogStream("my-app", "container-1/task-123")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("DeleteLogGroup", func() {
		It("should delete an existing log group", func() {
			// Create log group
			err := integration.CreateLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())

			// Delete it
			err = integration.DeleteLogGroup("my-app")
			Expect(err).NotTo(HaveOccurred())
			Expect(logsClient.logGroups["/ecs/my-app"]).To(BeFalse())
		})

		It("should not error if log group doesn't exist", func() {
			err := integration.DeleteLogGroup("non-existent")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetLogGroupForTask", func() {
		It("should return log group name for task", func() {
			taskArn := "arn:aws:ecs:us-east-1:000000000000:task/default/1234567890"
			groupName := integration.GetLogGroupForTask(taskArn)
			Expect(groupName).To(Equal("/ecs/kecs-tasks"))
		})
	})

	Describe("GetLogStreamForContainer", func() {
		It("should return log stream name for container", func() {
			taskArn := "arn:aws:ecs:us-east-1:000000000000:task/default/1234567890"
			streamName := integration.GetLogStreamForContainer(taskArn, "my-container")
			Expect(streamName).To(Equal("my-container/1234567890"))
		})
	})

	Describe("ConfigureContainerLogging", func() {
		It("should configure awslogs driver", func() {
			taskArn := "arn:aws:ecs:us-east-1:000000000000:task/default/1234567890"
			options := map[string]string{
				"awslogs-group":         "/ecs/my-app",
				"awslogs-region":        "us-east-1",
				"awslogs-stream-prefix": "my-container",
			}

			logConfig, err := integration.ConfigureContainerLogging(taskArn, "my-container", "awslogs", options)
			Expect(err).NotTo(HaveOccurred())
			Expect(logConfig).NotTo(BeNil())
			Expect(logConfig.LogGroupName).To(Equal("/ecs/my-app"))
			Expect(logConfig.LogStreamName).To(Equal("my-container"))
			Expect(logConfig.LogDriver).To(Equal("awslogs"))
		})

		It("should use default log group if not specified", func() {
			taskArn := "arn:aws:ecs:us-east-1:000000000000:task/default/1234567890"
			options := map[string]string{}

			logConfig, err := integration.ConfigureContainerLogging(taskArn, "my-container", "awslogs", options)
			Expect(err).NotTo(HaveOccurred())
			Expect(logConfig.LogGroupName).To(Equal("/ecs/kecs-tasks"))
			Expect(logConfig.LogStreamName).To(Equal("my-container"))
		})

		It("should error on unsupported log driver", func() {
			taskArn := "arn:aws:ecs:us-east-1:000000000000:task/default/1234567890"
			_, err := integration.ConfigureContainerLogging(taskArn, "my-container", "json-file", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported log driver"))
		})
	})
})
