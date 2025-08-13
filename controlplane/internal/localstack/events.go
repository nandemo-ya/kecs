package localstack

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// Event represents a LocalStack event
type Event struct {
	Service   string                 `json:"service"`
	EventType string                 `json:"eventType"`
	Region    string                 `json:"region"`
	Account   string                 `json:"account"`
	Time      time.Time              `json:"time"`
	Detail    map[string]interface{} `json:"detail"`
}

// ECSEvent represents an ECS-specific event
type ECSEvent struct {
	Event
	ClusterArn string `json:"clusterArn,omitempty"`
	TaskArn    string `json:"taskArn,omitempty"`
	ServiceArn string `json:"serviceArn,omitempty"`
	Status     string `json:"status,omitempty"`
}

// EventType constants for ECS events
const (
	EventTypeTaskStateChange    = "ECS Task State Change"
	EventTypeServiceAction      = "ECS Service Action"
	EventTypeClusterStateChange = "ECS Cluster State Change"
	EventTypeContainerInstance  = "ECS Container Instance State Change"
)

// EventFilter defines filters for LocalStack events
type EventFilter struct {
	Services   []string `json:"services,omitempty"`   // Filter by AWS service (e.g., "ecs")
	EventTypes []string `json:"eventTypes,omitempty"` // Filter by event type
	Resources  []string `json:"resources,omitempty"`  // Filter by resource ARN patterns
	Regions    []string `json:"regions,omitempty"`    // Filter by region
}

// EventProcessor processes LocalStack events
type EventProcessor interface {
	// ProcessEvent processes a single LocalStack event
	ProcessEvent(ctx context.Context, event *Event) error

	// GetFilter returns the event filter for this processor
	GetFilter() EventFilter
}

// EventMonitor monitors LocalStack events
type EventMonitor interface {
	// Start starts monitoring LocalStack events
	Start(ctx context.Context) error

	// Stop stops monitoring LocalStack events
	Stop(ctx context.Context) error

	// Subscribe subscribes an event processor to events
	Subscribe(processor EventProcessor)

	// Unsubscribe unsubscribes an event processor
	Unsubscribe(processor EventProcessor)
}

// eventMonitor implements the EventMonitor interface
type eventMonitor struct {
	manager    Manager
	processors []EventProcessor
	stopCh     chan struct{}
	isRunning  bool
}

// NewEventMonitor creates a new LocalStack event monitor
func NewEventMonitor(manager Manager) EventMonitor {
	return &eventMonitor{
		manager:    manager,
		processors: make([]EventProcessor, 0),
		stopCh:     make(chan struct{}),
	}
}

// Start starts monitoring LocalStack events
func (em *eventMonitor) Start(ctx context.Context) error {
	if em.isRunning {
		return fmt.Errorf("event monitor is already running")
	}

	logging.Info("Starting LocalStack event monitoring")
	em.isRunning = true

	// Start event polling in a goroutine
	go em.pollEvents(ctx)

	return nil
}

// Stop stops monitoring LocalStack events
func (em *eventMonitor) Stop(ctx context.Context) error {
	if !em.isRunning {
		return nil
	}

	logging.Info("Stopping LocalStack event monitoring")
	close(em.stopCh)
	em.isRunning = false

	return nil
}

// Subscribe subscribes an event processor to events
func (em *eventMonitor) Subscribe(processor EventProcessor) {
	em.processors = append(em.processors, processor)
	logging.Debug("Subscribed event processor", "filter", processor.GetFilter())
}

// Unsubscribe unsubscribes an event processor
func (em *eventMonitor) Unsubscribe(processor EventProcessor) {
	for i, p := range em.processors {
		if p == processor {
			em.processors = append(em.processors[:i], em.processors[i+1:]...)
			logging.Debug("Unsubscribed event processor")
			break
		}
	}
}

// pollEvents polls LocalStack for events
func (em *eventMonitor) pollEvents(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-em.stopCh:
			return
		case <-ticker.C:
			em.fetchAndProcessEvents(ctx)
		}
	}
}

// fetchAndProcessEvents fetches events from LocalStack and processes them
func (em *eventMonitor) fetchAndProcessEvents(ctx context.Context) {
	// Check if LocalStack is healthy
	if !em.manager.IsHealthy() {
		logging.Debug("LocalStack is not healthy, skipping event fetch")
		return
	}

	// In a real implementation, this would call LocalStack's event API
	// For now, we'll simulate some ECS events for demonstration
	em.simulateECSEvents(ctx)
}

// simulateECSEvents simulates ECS events for demonstration purposes
// In production, this would be replaced with actual LocalStack event API calls
func (em *eventMonitor) simulateECSEvents(ctx context.Context) {
	// This is a placeholder implementation
	// In production, you would:
	// 1. Call LocalStack's CloudWatch Events API
	// 2. Query for ECS-related events
	// 3. Parse the events and convert them to our Event struct

	logging.Debug("Simulating LocalStack events (placeholder)")

	// Example: Simulate a task state change event
	event := &Event{
		Service:   "ecs",
		EventType: EventTypeTaskStateChange,
		Region:    "us-east-1",
		Account:   "123456789012",
		Time:      time.Now(),
		Detail: map[string]interface{}{
			"clusterArn":    "arn:aws:ecs:us-east-1:123456789012:cluster/default",
			"taskArn":       "arn:aws:ecs:us-east-1:123456789012:task/default/abc123",
			"lastStatus":    "RUNNING",
			"desiredStatus": "RUNNING",
		},
	}

	// Process the event with all subscribed processors
	for _, processor := range em.processors {
		if em.shouldProcessEvent(event, processor.GetFilter()) {
			if err := processor.ProcessEvent(ctx, event); err != nil {
				logging.Error("Failed to process event with processor", "error", err)
			}
		}
	}
}

// shouldProcessEvent checks if an event should be processed based on the filter
func (em *eventMonitor) shouldProcessEvent(event *Event, filter EventFilter) bool {
	// Check service filter
	if len(filter.Services) > 0 {
		found := false
		for _, service := range filter.Services {
			if event.Service == service {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check event type filter
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if event.EventType == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check region filter
	if len(filter.Regions) > 0 {
		found := false
		for _, region := range filter.Regions {
			if event.Region == region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// ConvertToECSEvent converts a generic event to an ECS-specific event
func ConvertToECSEvent(event *Event) (*ECSEvent, error) {
	if event.Service != "ecs" {
		return nil, fmt.Errorf("event is not an ECS event: %s", event.Service)
	}

	ecsEvent := &ECSEvent{
		Event: *event,
	}

	// Extract ECS-specific fields from detail
	if detail := event.Detail; detail != nil {
		if clusterArn, ok := detail["clusterArn"].(string); ok {
			ecsEvent.ClusterArn = clusterArn
		}
		if taskArn, ok := detail["taskArn"].(string); ok {
			ecsEvent.TaskArn = taskArn
		}
		if serviceArn, ok := detail["serviceArn"].(string); ok {
			ecsEvent.ServiceArn = serviceArn
		}
		if status, ok := detail["lastStatus"].(string); ok {
			ecsEvent.Status = status
		}
	}

	return ecsEvent, nil
}

// MarshalJSON implements json.Marshaler for Event
func (e *Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Time string `json:"time"`
		*Alias
	}{
		Time:  e.Time.Format(time.RFC3339),
		Alias: (*Alias)(e),
	})
}

// UnmarshalJSON implements json.Unmarshaler for Event
func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	aux := &struct {
		Time string `json:"time"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	var err error
	e.Time, err = time.Parse(time.RFC3339, aux.Time)
	if err != nil {
		return fmt.Errorf("failed to parse time: %w", err)
	}

	return nil
}
