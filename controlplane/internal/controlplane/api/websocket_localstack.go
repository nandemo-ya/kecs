package api

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// LocalStackEventProcessor processes LocalStack events and broadcasts them via WebSocket
type LocalStackEventProcessor struct {
	hub    *WebSocketHub
	filter localstack.EventFilter
}

// NewLocalStackEventProcessor creates a new LocalStack event processor for WebSocket broadcasting
func NewLocalStackEventProcessor(hub *WebSocketHub) *LocalStackEventProcessor {
	return &LocalStackEventProcessor{
		hub: hub,
		filter: localstack.EventFilter{
			Services: []string{"ecs"}, // Only process ECS events
			EventTypes: []string{
				localstack.EventTypeTaskStateChange,
				localstack.EventTypeServiceAction,
				localstack.EventTypeClusterStateChange,
				localstack.EventTypeContainerInstance,
			},
		},
	}
}

// ProcessEvent processes a LocalStack event and broadcasts it via WebSocket
func (p *LocalStackEventProcessor) ProcessEvent(ctx context.Context, event *localstack.Event) error {
	klog.V(3).Infof("Processing LocalStack event: %s - %s", event.Service, event.EventType)

	// Convert to ECS event if applicable
	ecsEvent, err := localstack.ConvertToECSEvent(event)
	if err != nil {
		klog.V(4).Infof("Event is not an ECS event, skipping: %v", err)
		return nil
	}

	// Create WebSocket message based on event type
	wsMessage, err := p.createWebSocketMessage(ecsEvent)
	if err != nil {
		klog.Errorf("Failed to create WebSocket message from LocalStack event: %v", err)
		return err
	}

	// Broadcast to WebSocket clients
	p.hub.BroadcastWithFiltering(*wsMessage)
	klog.V(2).Infof("Broadcasted LocalStack event via WebSocket: %s", wsMessage.Type)

	return nil
}

// GetFilter returns the event filter for this processor
func (p *LocalStackEventProcessor) GetFilter() localstack.EventFilter {
	return p.filter
}

// createWebSocketMessage creates a WebSocket message from an ECS event
func (p *LocalStackEventProcessor) createWebSocketMessage(ecsEvent *localstack.ECSEvent) (*WebSocketMessage, error) {
	var messageType string
	var resourceType string
	var resourceID string

	// Determine message type and resource info based on event type
	switch ecsEvent.EventType {
	case localstack.EventTypeTaskStateChange:
		messageType = "task_status_changed"
		resourceType = "task"
		resourceID = extractResourceID(ecsEvent.TaskArn)
	case localstack.EventTypeServiceAction:
		messageType = "service_updated"
		resourceType = "service"
		resourceID = extractResourceID(ecsEvent.ServiceArn)
	case localstack.EventTypeClusterStateChange:
		messageType = "cluster_updated"
		resourceType = "cluster"
		resourceID = extractResourceID(ecsEvent.ClusterArn)
	case localstack.EventTypeContainerInstance:
		messageType = "container_instance_updated"
		resourceType = "container_instance"
		resourceID = "unknown" // Container instance ARN not available in this context
	default:
		messageType = "localstack_event"
		resourceType = "unknown"
		resourceID = "unknown"
	}

	// Create the payload
	payload := map[string]interface{}{
		"source":    "localstack",
		"eventType": ecsEvent.EventType,
		"service":   ecsEvent.Service,
		"region":    ecsEvent.Region,
		"account":   ecsEvent.Account,
		"timestamp": ecsEvent.Time.Format(time.RFC3339),
		"detail":    ecsEvent.Detail,
	}

	// Add ECS-specific fields
	if ecsEvent.ClusterArn != "" {
		payload["clusterArn"] = ecsEvent.ClusterArn
	}
	if ecsEvent.TaskArn != "" {
		payload["taskArn"] = ecsEvent.TaskArn
	}
	if ecsEvent.ServiceArn != "" {
		payload["serviceArn"] = ecsEvent.ServiceArn
	}
	if ecsEvent.Status != "" {
		payload["status"] = ecsEvent.Status
	}

	// Marshal payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Create WebSocket message
	wsMessage := &WebSocketMessage{
		Type:         messageType,
		Payload:      json.RawMessage(payloadJSON),
		Timestamp:    time.Now(),
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}

	return wsMessage, nil
}

// extractResourceID extracts the resource ID from an ARN
func extractResourceID(arn string) string {
	if arn == "" {
		return "unknown"
	}

	// ARN format: arn:aws:service:region:account:resource-type/resource-name
	// or: arn:aws:service:region:account:resource-type:resource-name
	parts := strings.Split(arn, ":")
	if len(parts) < 6 {
		return "unknown"
	}

	// Get the resource part (last part)
	resourcePart := parts[len(parts)-1]

	// Handle resource-type/resource-name format
	if strings.Contains(resourcePart, "/") {
		resourceParts := strings.Split(resourcePart, "/")
		if len(resourceParts) > 1 {
			return resourceParts[len(resourceParts)-1]
		}
	}

	return resourcePart
}

// LocalStackEventConfig holds configuration for LocalStack event integration
type LocalStackEventConfig struct {
	Enabled        bool     `json:"enabled"`
	PollInterval   string   `json:"pollInterval"`   // e.g., "5s"
	EventTypes     []string `json:"eventTypes"`     // Event types to monitor
	FilterServices []string `json:"filterServices"` // Services to monitor (e.g., ["ecs"])
}

// DefaultLocalStackEventConfig returns default configuration for LocalStack events
func DefaultLocalStackEventConfig() *LocalStackEventConfig {
	return &LocalStackEventConfig{
		Enabled:      true,
		PollInterval: "5s",
		EventTypes: []string{
			localstack.EventTypeTaskStateChange,
			localstack.EventTypeServiceAction,
			localstack.EventTypeClusterStateChange,
		},
		FilterServices: []string{"ecs"},
	}
}

// LocalStackEventIntegration manages LocalStack event monitoring and WebSocket integration
type LocalStackEventIntegration struct {
	monitor   localstack.EventMonitor
	processor *LocalStackEventProcessor
	config    *LocalStackEventConfig
	isRunning bool
}

// NewLocalStackEventIntegration creates a new LocalStack event integration
func NewLocalStackEventIntegration(
	localStackManager localstack.Manager,
	hub *WebSocketHub,
	config *LocalStackEventConfig,
) *LocalStackEventIntegration {
	if config == nil {
		config = DefaultLocalStackEventConfig()
	}

	monitor := localstack.NewEventMonitor(localStackManager)
	processor := NewLocalStackEventProcessor(hub)

	return &LocalStackEventIntegration{
		monitor:   monitor,
		processor: processor,
		config:    config,
	}
}

// Start starts the LocalStack event integration
func (lei *LocalStackEventIntegration) Start(ctx context.Context) error {
	if !lei.config.Enabled {
		klog.Info("LocalStack event integration is disabled")
		return nil
	}

	if lei.isRunning {
		return nil
	}

	klog.Info("Starting LocalStack event integration for WebSocket broadcasting")

	// Subscribe the processor to the monitor
	lei.monitor.Subscribe(lei.processor)

	// Start the event monitor
	if err := lei.monitor.Start(ctx); err != nil {
		return err
	}

	lei.isRunning = true
	klog.Info("LocalStack event integration started successfully")
	return nil
}

// Stop stops the LocalStack event integration
func (lei *LocalStackEventIntegration) Stop(ctx context.Context) error {
	if !lei.isRunning {
		return nil
	}

	klog.Info("Stopping LocalStack event integration")

	// Unsubscribe the processor
	lei.monitor.Unsubscribe(lei.processor)

	// Stop the event monitor
	if err := lei.monitor.Stop(ctx); err != nil {
		return err
	}

	lei.isRunning = false
	klog.Info("LocalStack event integration stopped")
	return nil
}

// IsRunning returns whether the integration is currently running
func (lei *LocalStackEventIntegration) IsRunning() bool {
	return lei.isRunning
}
