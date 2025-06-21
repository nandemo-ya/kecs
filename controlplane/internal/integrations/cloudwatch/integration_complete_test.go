package cloudwatch_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	cloudwatchlogsapi "github.com/nandemo-ya/kecs/controlplane/internal/cloudwatchlogs/generated"
	"github.com/nandemo-ya/kecs/controlplane/internal/integrations/cloudwatch"
)

var _ = Describe("CloudWatch Integration Complete", func() {
	var (
		integration cloudwatch.Integration
		mockClient  *mockCloudWatchLogsTestClient
	)

	BeforeEach(func() {
		mockClient = &mockCloudWatchLogsTestClient{
			createLogGroupCalls:  []*cloudwatchlogsapi.CreateLogGroupRequest{},
			createLogStreamCalls: []*cloudwatchlogsapi.CreateLogStreamRequest{},
		}
		
		integration = cloudwatch.NewIntegrationWithClient(
			nil, // kubeClient not needed for these tests
			nil, // localstack manager not needed
			&cloudwatch.Config{
				LogGroupPrefix: "/ecs/",
				RetentionDays:  7,
				KubeNamespace:  "default",
			},
			mockClient,
		)
	})

	Describe("Task Logging Configuration", func() {
		It("should configure CloudWatch logging for a task", func() {
			taskArn := "arn:aws:ecs:us-east-1:123456789012:task/default/task-123"
			containerName := "my-app"
			logDriver := "awslogs"
			options := map[string]string{
				"awslogs-group":         "/ecs/my-app",
				"awslogs-region":        "us-east-1",
				"awslogs-stream-prefix": "my-app",
			}

			logConfig, err := integration.ConfigureContainerLogging(taskArn, containerName, logDriver, options)
			Expect(err).NotTo(HaveOccurred())
			Expect(logConfig).NotTo(BeNil())
			
			// Verify configuration
			Expect(logConfig.LogGroupName).To(Equal("/ecs/my-app"))
			Expect(logConfig.LogStreamName).To(Equal("my-app"))
			Expect(logConfig.LogDriver).To(Equal("awslogs"))
			
			// Verify FluentBit config was generated
			Expect(logConfig.FluentBitConfig).To(ContainSubstring("[OUTPUT]"))
			Expect(logConfig.FluentBitConfig).To(ContainSubstring("cloudwatch_logs"))
			Expect(logConfig.FluentBitConfig).To(ContainSubstring("/ecs/my-app"))
		})

		It("should create log group if not specified", func() {
			taskArn := "arn:aws:ecs:us-east-1:123456789012:task/default/task-456"
			containerName := "nginx"
			logDriver := "awslogs"
			options := map[string]string{} // No group specified

			logConfig, err := integration.ConfigureContainerLogging(taskArn, containerName, logDriver, options)
			Expect(err).NotTo(HaveOccurred())
			Expect(logConfig).NotTo(BeNil())
			
			// Should use default group
			Expect(logConfig.LogGroupName).To(Equal("/ecs/kecs-tasks"))
			
			// Verify log group creation was called
			Expect(mockClient.createLogGroupCalls).To(HaveLen(1))
			Expect(mockClient.createLogGroupCalls[0].LogGroupName).To(Equal("/ecs/kecs-tasks"))
		})

		It("should reject non-awslogs drivers", func() {
			taskArn := "arn:aws:ecs:us-east-1:123456789012:task/default/task-789"
			containerName := "app"
			logDriver := "json-file"
			options := map[string]string{}

			logConfig, err := integration.ConfigureContainerLogging(taskArn, containerName, logDriver, options)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported log driver"))
			Expect(logConfig).To(BeNil())
		})
	})

	Describe("Log Stream Management", func() {
		It("should create unique log streams for containers", func() {
			taskArn := "arn:aws:ecs:us-east-1:123456789012:task/default/task-abc123"
			
			// Get stream names for different containers
			stream1 := integration.GetLogStreamForContainer(taskArn, "web")
			stream2 := integration.GetLogStreamForContainer(taskArn, "app")
			
			Expect(stream1).To(Equal("web/task-abc123"))
			Expect(stream2).To(Equal("app/task-abc123"))
			Expect(stream1).NotTo(Equal(stream2))
		})

		It("should handle log group creation with retention", func() {
			groupName := "test-group"
			
			err := integration.CreateLogGroup(groupName)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify calls
			Expect(mockClient.createLogGroupCalls).To(HaveLen(1))
			Expect(mockClient.createLogGroupCalls[0].LogGroupName).To(Equal("/ecs/test-group"))
			
			// Verify retention policy was set
			Expect(mockClient.putRetentionPolicyCalls).To(HaveLen(1))
			Expect(mockClient.putRetentionPolicyCalls[0].LogGroupName).To(Equal("/ecs/test-group"))
			Expect(mockClient.putRetentionPolicyCalls[0].RetentionInDays).To(Equal(int32(7)))
		})
	})

	Describe("FluentBit Configuration", func() {
		It("should generate valid FluentBit configuration", func() {
			taskArn := "arn:aws:ecs:us-east-1:123456789012:task/default/task-xyz"
			containerName := "app"
			options := map[string]string{
				"awslogs-group":  "/ecs/my-service",
				"awslogs-region": "us-west-2",
			}

			logConfig, err := integration.ConfigureContainerLogging(taskArn, containerName, "awslogs", options)
			Expect(err).NotTo(HaveOccurred())
			
			// Verify FluentBit config contains required sections
			config := logConfig.FluentBitConfig
			Expect(config).To(ContainSubstring("[SERVICE]"))
			Expect(config).To(ContainSubstring("[INPUT]"))
			Expect(config).To(ContainSubstring("[FILTER]"))
			Expect(config).To(ContainSubstring("[OUTPUT]"))
			
			// Verify specific settings
			Expect(config).To(ContainSubstring("region              us-west-2"))
			Expect(config).To(ContainSubstring("log_group_name      /ecs/my-service"))
			Expect(config).To(ContainSubstring("endpoint            http://localstack.aws-services.svc.cluster.local:4566"))
		})
	})
})

// mockCloudWatchLogsTestClient for testing
type mockCloudWatchLogsTestClient struct {
	createLogGroupCalls      []*cloudwatchlogsapi.CreateLogGroupRequest
	createLogStreamCalls     []*cloudwatchlogsapi.CreateLogStreamRequest
	putRetentionPolicyCalls  []*cloudwatchlogsapi.PutRetentionPolicyRequest
}

func (m *mockCloudWatchLogsTestClient) CreateLogGroup(ctx context.Context, params *cloudwatchlogsapi.CreateLogGroupRequest) (*cloudwatchlogsapi.Unit, error) {
	m.createLogGroupCalls = append(m.createLogGroupCalls, params)
	return &cloudwatchlogsapi.Unit{}, nil
}

func (m *mockCloudWatchLogsTestClient) DeleteLogGroup(ctx context.Context, params *cloudwatchlogsapi.DeleteLogGroupRequest) (*cloudwatchlogsapi.Unit, error) {
	return &cloudwatchlogsapi.Unit{}, nil
}

func (m *mockCloudWatchLogsTestClient) CreateLogStream(ctx context.Context, params *cloudwatchlogsapi.CreateLogStreamRequest) (*cloudwatchlogsapi.Unit, error) {
	m.createLogStreamCalls = append(m.createLogStreamCalls, params)
	return &cloudwatchlogsapi.Unit{}, nil
}

func (m *mockCloudWatchLogsTestClient) PutRetentionPolicy(ctx context.Context, params *cloudwatchlogsapi.PutRetentionPolicyRequest) (*cloudwatchlogsapi.Unit, error) {
	m.putRetentionPolicyCalls = append(m.putRetentionPolicyCalls, params)
	return &cloudwatchlogsapi.Unit{}, nil
}