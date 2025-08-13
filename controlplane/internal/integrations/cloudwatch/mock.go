package cloudwatch

// MockIntegration is a mock implementation of the Integration interface for testing
type MockIntegration struct {
	CreateLogGroupCalled  bool
	CreateLogStreamCalled bool
	LogGroupCreated       string
	LogStreamCreated      string
	CreateLogGroupError   error
	CreateLogStreamError  error
}

// CreateLogGroup mock implementation
func (m *MockIntegration) CreateLogGroup(groupName string) error {
	m.CreateLogGroupCalled = true
	m.LogGroupCreated = groupName
	return m.CreateLogGroupError
}

// CreateLogStream mock implementation
func (m *MockIntegration) CreateLogStream(groupName, streamName string) error {
	m.CreateLogStreamCalled = true
	m.LogStreamCreated = streamName
	return m.CreateLogStreamError
}

// DeleteLogGroup mock implementation
func (m *MockIntegration) DeleteLogGroup(groupName string) error {
	return nil
}

// GetLogGroupForTask mock implementation
func (m *MockIntegration) GetLogGroupForTask(taskArn string) string {
	return "/ecs/default"
}

// GetLogStreamForContainer mock implementation
func (m *MockIntegration) GetLogStreamForContainer(taskArn, containerName string) string {
	return containerName + "-stream"
}

// ConfigureContainerLogging mock implementation
func (m *MockIntegration) ConfigureContainerLogging(taskArn string, containerName string, logDriver string, options map[string]string) (*LogConfiguration, error) {
	return &LogConfiguration{
		LogDriver: logDriver,
		Options:   options,
	}, nil
}
