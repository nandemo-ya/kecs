package localstack_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// Mock event processor for testing
type mockEventProcessor struct {
	processedEvents []localstack.Event
	filter          localstack.EventFilter
}

func (m *mockEventProcessor) ProcessEvent(ctx context.Context, event *localstack.Event) error {
	m.processedEvents = append(m.processedEvents, *event)
	return nil
}

func (m *mockEventProcessor) GetFilter() localstack.EventFilter {
	return m.filter
}

// Mock LocalStack manager for testing
type mockLocalStackManager struct {
	healthy bool
}

func (m *mockLocalStackManager) IsHealthy() bool {
	return m.healthy
}

// Implement other Manager interface methods as no-ops for testing
func (m *mockLocalStackManager) Start(ctx context.Context) error { return nil }
func (m *mockLocalStackManager) Stop(ctx context.Context) error { return nil }
func (m *mockLocalStackManager) Restart(ctx context.Context) error { return nil }
func (m *mockLocalStackManager) GetStatus() (*localstack.Status, error) { 
	return &localstack.Status{Healthy: m.healthy}, nil 
}
func (m *mockLocalStackManager) UpdateServices(services []string) error { return nil }
func (m *mockLocalStackManager) GetEnabledServices() ([]string, error) { return []string{"ecs"}, nil }
func (m *mockLocalStackManager) GetEndpoint() (string, error) { return "http://localhost:4566", nil }
func (m *mockLocalStackManager) GetServiceEndpoint(service string) (string, error) { return "http://localhost:4566", nil }
func (m *mockLocalStackManager) WaitForReady(ctx context.Context, timeout time.Duration) error { return nil }
func (m *mockLocalStackManager) IsRunning() bool { return m.healthy }
func (m *mockLocalStackManager) CheckServiceHealth(service string) error { return nil }
func (m *mockLocalStackManager) GetConfig() *localstack.Config { 
	return &localstack.Config{
		Enabled: true,
		Services: []string{"ecs"},
		Version: "latest",
	}
}

var _ = Describe("LocalStack Events", func() {
	var (
		eventMonitor   localstack.EventMonitor
		mockManager    *mockLocalStackManager
		mockProcessor  *mockEventProcessor
		ctx           context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockManager = &mockLocalStackManager{healthy: true}
		eventMonitor = localstack.NewEventMonitor(mockManager)
		
		mockProcessor = &mockEventProcessor{
			processedEvents: make([]localstack.Event, 0),
			filter: localstack.EventFilter{
				Services: []string{"ecs"},
				EventTypes: []string{localstack.EventTypeTaskStateChange},
			},
		}
	})

	Describe("EventMonitor", func() {
		It("should start and stop successfully", func() {
			err := eventMonitor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = eventMonitor.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should subscribe and unsubscribe processors", func() {
			// Subscribe processor
			eventMonitor.Subscribe(mockProcessor)

			// Start monitoring
			err := eventMonitor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Wait a bit for potential events
			time.Sleep(100 * time.Millisecond)

			// Unsubscribe processor
			eventMonitor.Unsubscribe(mockProcessor)

			// Stop monitoring
			err = eventMonitor.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not start if already running", func() {
			err := eventMonitor.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Try to start again
			err = eventMonitor.Start(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already running"))

			// Cleanup
			eventMonitor.Stop(ctx)
		})
	})

	Describe("Event Conversion", func() {
		It("should convert generic event to ECS event", func() {
			event := &localstack.Event{
				Service:   "ecs",
				EventType: localstack.EventTypeTaskStateChange,
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      time.Now(),
				Detail: map[string]interface{}{
					"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					"taskArn":    "arn:aws:ecs:us-east-1:123456789012:task/test/abc123",
					"lastStatus": "RUNNING",
				},
			}

			ecsEvent, err := localstack.ConvertToECSEvent(event)
			Expect(err).NotTo(HaveOccurred())
			Expect(ecsEvent).NotTo(BeNil())
			Expect(ecsEvent.ClusterArn).To(Equal("arn:aws:ecs:us-east-1:123456789012:cluster/test"))
			Expect(ecsEvent.TaskArn).To(Equal("arn:aws:ecs:us-east-1:123456789012:task/test/abc123"))
			Expect(ecsEvent.Status).To(Equal("RUNNING"))
		})

		It("should fail to convert non-ECS event", func() {
			event := &localstack.Event{
				Service:   "s3",
				EventType: "S3 Bucket Created",
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      time.Now(),
			}

			ecsEvent, err := localstack.ConvertToECSEvent(event)
			Expect(err).To(HaveOccurred())
			Expect(ecsEvent).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("not an ECS event"))
		})
	})

	Describe("Event Marshaling", func() {
		It("should marshal and unmarshal events correctly", func() {
			// Use UTC time to avoid timezone issues
			testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
			originalEvent := &localstack.Event{
				Service:   "ecs",
				EventType: localstack.EventTypeTaskStateChange,
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      testTime,
				Detail: map[string]interface{}{
					"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					"taskArn":    "arn:aws:ecs:us-east-1:123456789012:task/test/abc123",
					"lastStatus": "RUNNING",
				},
			}

			// Marshal to JSON
			jsonData, err := originalEvent.MarshalJSON()
			Expect(err).NotTo(HaveOccurred())


			// Unmarshal from JSON
			var unmarshaledEvent localstack.Event
			err = unmarshaledEvent.UnmarshalJSON(jsonData)
			Expect(err).NotTo(HaveOccurred())

			// Compare fields
			Expect(unmarshaledEvent.Service).To(Equal(originalEvent.Service))
			Expect(unmarshaledEvent.EventType).To(Equal(originalEvent.EventType))
			Expect(unmarshaledEvent.Region).To(Equal(originalEvent.Region))
			Expect(unmarshaledEvent.Account).To(Equal(originalEvent.Account))
			// Compare times with second precision
			Expect(unmarshaledEvent.Time.Unix()).To(Equal(originalEvent.Time.Unix()))
		})
	})
})