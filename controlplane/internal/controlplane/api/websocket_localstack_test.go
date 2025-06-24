package api_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

var _ = Describe("WebSocket LocalStack Integration", func() {
	var (
		hub       *api.WebSocketHub
		processor *api.LocalStackEventProcessor
		ctx       context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
		hub = api.NewWebSocketHub()
		processor = api.NewLocalStackEventProcessor(hub)
	})

	Describe("LocalStackEventProcessor", func() {
		It("should process ECS task state change events", func() {
			event := &localstack.Event{
				Service:   "ecs",
				EventType: localstack.EventTypeTaskStateChange,
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      time.Now(),
				Detail: map[string]interface{}{
					"clusterArn":    "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					"taskArn":       "arn:aws:ecs:us-east-1:123456789012:task/test/abc123",
					"lastStatus":    "RUNNING",
					"desiredStatus": "RUNNING",
				},
			}

			err := processor.ProcessEvent(ctx, event)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process ECS service action events", func() {
			event := &localstack.Event{
				Service:   "ecs",
				EventType: localstack.EventTypeServiceAction,
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      time.Now(),
				Detail: map[string]interface{}{
					"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
					"serviceArn": "arn:aws:ecs:us-east-1:123456789012:service/test/web-service",
				},
			}

			err := processor.ProcessEvent(ctx, event)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should process ECS cluster state change events", func() {
			event := &localstack.Event{
				Service:   "ecs",
				EventType: localstack.EventTypeClusterStateChange,
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      time.Now(),
				Detail: map[string]interface{}{
					"clusterArn": "arn:aws:ecs:us-east-1:123456789012:cluster/test",
				},
			}

			err := processor.ProcessEvent(ctx, event)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should skip non-ECS events", func() {
			event := &localstack.Event{
				Service:   "s3",
				EventType: "S3 Bucket Created",
				Region:    "us-east-1",
				Account:   "123456789012",
				Time:      time.Now(),
				Detail: map[string]interface{}{
					"bucketName": "test-bucket",
				},
			}

			err := processor.ProcessEvent(ctx, event)
			Expect(err).NotTo(HaveOccurred()) // Should not error, but should skip
		})

		It("should return correct event filter", func() {
			filter := processor.GetFilter()
			Expect(filter.Services).To(ContainElement("ecs"))
			Expect(filter.EventTypes).To(ContainElement(localstack.EventTypeTaskStateChange))
			Expect(filter.EventTypes).To(ContainElement(localstack.EventTypeServiceAction))
			Expect(filter.EventTypes).To(ContainElement(localstack.EventTypeClusterStateChange))
		})
	})

	Describe("LocalStackEventIntegration", func() {
		var (
			mockManager *mockLocalStackManagerForWS
			integration *api.LocalStackEventIntegration
		)

		BeforeEach(func() {
			mockManager = &mockLocalStackManagerForWS{healthy: true}
			integration = api.NewLocalStackEventIntegration(
				mockManager,
				hub,
				api.DefaultLocalStackEventConfig(),
			)
		})

		It("should start and stop successfully", func() {
			err := integration.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(integration.IsRunning()).To(BeTrue())

			err = integration.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(integration.IsRunning()).To(BeFalse())
		})

		It("should not start if disabled in config", func() {
			disabledConfig := &api.LocalStackEventConfig{
				Enabled: false,
			}

			disabledIntegration := api.NewLocalStackEventIntegration(
				mockManager,
				hub,
				disabledConfig,
			)

			err := disabledIntegration.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(disabledIntegration.IsRunning()).To(BeFalse())
		})

		It("should use default config when nil is passed", func() {
			nilConfigIntegration := api.NewLocalStackEventIntegration(
				mockManager,
				hub,
				nil, // Pass nil config
			)

			err := nilConfigIntegration.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(nilConfigIntegration.IsRunning()).To(BeTrue())

			err = nilConfigIntegration.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("DefaultLocalStackEventConfig", func() {
		It("should return valid default configuration", func() {
			config := api.DefaultLocalStackEventConfig()
			Expect(config).NotTo(BeNil())
			Expect(config.Enabled).To(BeTrue())
			Expect(config.PollInterval).To(Equal("5s"))
			Expect(config.EventTypes).To(ContainElement(localstack.EventTypeTaskStateChange))
			Expect(config.FilterServices).To(ContainElement("ecs"))
		})
	})
})

// Mock LocalStack manager for WebSocket tests
type mockLocalStackManagerForWS struct {
	healthy bool
}

func (m *mockLocalStackManagerForWS) IsHealthy() bool {
	return m.healthy
}

// Implement other Manager interface methods as no-ops for testing
func (m *mockLocalStackManagerForWS) Start(ctx context.Context) error   { return nil }
func (m *mockLocalStackManagerForWS) Stop(ctx context.Context) error    { return nil }
func (m *mockLocalStackManagerForWS) Restart(ctx context.Context) error { return nil }
func (m *mockLocalStackManagerForWS) GetStatus() (*localstack.Status, error) {
	return &localstack.Status{Healthy: m.healthy}, nil
}
func (m *mockLocalStackManagerForWS) UpdateServices(services []string) error { return nil }
func (m *mockLocalStackManagerForWS) GetEnabledServices() ([]string, error) {
	return []string{"ecs"}, nil
}
func (m *mockLocalStackManagerForWS) GetEndpoint() (string, error) {
	return "http://localhost:4566", nil
}
func (m *mockLocalStackManagerForWS) GetServiceEndpoint(service string) (string, error) {
	return "http://localhost:4566", nil
}
func (m *mockLocalStackManagerForWS) WaitForReady(ctx context.Context, timeout time.Duration) error {
	return nil
}
func (m *mockLocalStackManagerForWS) IsRunning() bool                         { return m.healthy }
func (m *mockLocalStackManagerForWS) CheckServiceHealth(service string) error { return nil }
func (m *mockLocalStackManagerForWS) GetConfig() *localstack.Config {
	return &localstack.Config{
		Enabled:  true,
		Services: []string{"ecs"},
		Version:  "latest",
	}
}
