package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebSocketHub_Run(t *testing.T) {
	hub := NewWebSocketHub()
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Run hub in background
	go hub.Run(ctx)
	
	// Give it time to start
	time.Sleep(10 * time.Millisecond)
	
	// Test client registration
	client := &WebSocketClient{
		hub:           hub,
		send:          make(chan WebSocketMessage, 1),
		id:            "test-client",
		subscriptions: make(map[string]map[string]bool),
		filters:       []EventFilter{},
	}
	
	hub.register <- client
	time.Sleep(10 * time.Millisecond)
	
	// Check client is registered
	hub.mu.RLock()
	assert.True(t, hub.clients[client])
	hub.mu.RUnlock()
	
	// Test broadcast
	testMsg := WebSocketMessage{
		Type:      "test",
		Timestamp: time.Now(),
	}
	hub.broadcast <- testMsg
	
	// Client should receive the message
	select {
	case msg := <-client.send:
		assert.Equal(t, "test", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client did not receive broadcast message")
	}
	
	// Test client unregistration
	hub.unregister <- client
	time.Sleep(10 * time.Millisecond)
	
	// Check client is unregistered
	hub.mu.RLock()
	assert.False(t, hub.clients[client])
	hub.mu.RUnlock()
}

func TestWebSocketOriginCheck(t *testing.T) {
	tests := []struct {
		name           string
		config         *WebSocketConfig
		origin         string
		expectUpgrade  bool
	}{
		{
			name: "allowed origin",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			origin:        "http://localhost:3000",
			expectUpgrade: true,
		},
		{
			name: "denied origin",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			origin:        "http://malicious.com",
			expectUpgrade: false,
		},
		{
			name: "no origin header with allowed list",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			origin:        "",
			expectUpgrade: true, // Same origin request
		},
		{
			name:          "empty allowed list allows all",
			config:        &WebSocketConfig{},
			origin:        "http://any-origin.com",
			expectUpgrade: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create hub with config
			hub := NewWebSocketHubWithConfig(tt.config)
			
			// Start hub in background
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go hub.Run(ctx)
			
			// Give hub time to start
			time.Sleep(10 * time.Millisecond)
			
			// Create server
			server := &Server{
				webSocketHub: hub,
			}
			
			// Create test server
			handler := server.HandleWebSocket(hub)
			ts := httptest.NewServer(handler)
			defer ts.Close()
			
			// Convert http to ws
			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
			
			// Create WebSocket connection request
			dialer := websocket.DefaultDialer
			header := http.Header{}
			if tt.origin != "" {
				header.Set("Origin", tt.origin)
			}
			
			// Attempt connection
			conn, resp, err := dialer.Dial(wsURL, header)
			
			if tt.expectUpgrade {
				require.NoError(t, err)
				require.NotNil(t, conn)
				defer conn.Close()
				
				// Connection was successful - origin check passed
				// Give time for the client to be registered
				time.Sleep(50 * time.Millisecond)
			} else {
				// Connection should be rejected
				assert.Error(t, err)
				if resp != nil {
					assert.NotEqual(t, http.StatusSwitchingProtocols, resp.StatusCode)
				}
			}
		})
	}
}

func TestWebSocketClient_handleMessage(t *testing.T) {
	client := &WebSocketClient{
		send:          make(chan WebSocketMessage, 10),
		id:            "test-client",
		subscriptions: make(map[string]map[string]bool),
		filters:       []EventFilter{},
	}
	
	tests := []struct {
		name     string
		message  WebSocketMessage
		validate func(t *testing.T, client *WebSocketClient)
	}{
		{
			name: "ping message",
			message: WebSocketMessage{
				Type: "ping",
				ID:   "123",
			},
			validate: func(t *testing.T, client *WebSocketClient) {
				select {
				case msg := <-client.send:
					assert.Equal(t, "pong", msg.Type)
					assert.Equal(t, "123", msg.ID)
				case <-time.After(100 * time.Millisecond):
					t.Fatal("No pong response received")
				}
			},
		},
		{
			name: "subscribe message",
			message: WebSocketMessage{
				Type:    "subscribe",
				Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
			},
			validate: func(t *testing.T, client *WebSocketClient) {
				// For now, just ensure no panic
				// In real implementation, would check subscription state
			},
		},
		{
			name: "unsubscribe message",
			message: WebSocketMessage{
				Type:    "unsubscribe",
				Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
			},
			validate: func(t *testing.T, client *WebSocketClient) {
				// For now, just ensure no panic
				// In real implementation, would check subscription state
			},
		},
		{
			name: "unknown message type",
			message: WebSocketMessage{
				Type: "unknown",
			},
			validate: func(t *testing.T, client *WebSocketClient) {
				// Should not crash
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client.handleMessage(tt.message)
			if tt.validate != nil {
				tt.validate(t, client)
			}
		})
	}
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	hub := NewWebSocketHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go hub.Run(ctx)
	
	// Create and register multiple clients
	clients := make([]*WebSocketClient, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &WebSocketClient{
			hub:           hub,
			send:          make(chan WebSocketMessage, 10),
			id:            fmt.Sprintf("client-%d", i),
			subscriptions: make(map[string]map[string]bool),
			filters:       []EventFilter{},
		}
		hub.register <- clients[i]
	}
	
	// Give time for registration
	time.Sleep(10 * time.Millisecond)
	
	// Test BroadcastTaskUpdate
	hub.BroadcastTaskUpdate("task-123", map[string]interface{}{
		"status": "RUNNING",
		"cpu":    "50%",
	})
	
	// All clients should receive the update
	for i, client := range clients {
		select {
		case msg := <-client.send:
			assert.Equal(t, "task_update", msg.Type)
			assert.NotNil(t, msg.Payload)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Client %d did not receive task update", i)
		}
	}
	
	// Test BroadcastLogEntry
	hub.BroadcastLogEntry(map[string]interface{}{
		"message":   "Test log",
		"level":     "info",
		"timestamp": time.Now(),
	})
	
	// All clients should receive the log
	for i, client := range clients {
		select {
		case msg := <-client.send:
			assert.Equal(t, "log_entry", msg.Type)
			assert.NotNil(t, msg.Payload)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Client %d did not receive log entry", i)
		}
	}
	
	// Test BroadcastMetricUpdate
	hub.BroadcastMetricUpdate(map[string]interface{}{
		"cpu_usage":    75.5,
		"memory_usage": 1024,
	})
	
	// All clients should receive the metrics
	for i, client := range clients {
		select {
		case msg := <-client.send:
			assert.Equal(t, "metric_update", msg.Type)
			assert.NotNil(t, msg.Payload)
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Client %d did not receive metric update", i)
		}
	}
}

func TestWebSocketEventFiltering(t *testing.T) {
	client := &WebSocketClient{
		id:      "test-client",
		filters: []EventFilter{},
	}

	// Test with no filters - should accept all messages
	msg := WebSocketMessage{
		Type:         "task_update",
		ResourceType: "task",
		ResourceID:   "task-123",
	}
	assert.True(t, client.MatchesFilter(msg))

	// Test with event type filter
	client.SetFilters([]EventFilter{
		{EventTypes: []string{"task_update", "service_update"}},
	})
	assert.True(t, client.MatchesFilter(msg))
	
	msg.Type = "log_entry"
	assert.False(t, client.MatchesFilter(msg))

	// Test with resource type filter
	client.SetFilters([]EventFilter{
		{ResourceTypes: []string{"task", "service"}},
	})
	msg.Type = "task_update"
	assert.True(t, client.MatchesFilter(msg))
	
	msg.ResourceType = "cluster"
	assert.False(t, client.MatchesFilter(msg))

	// Test with resource ID filter
	client.SetFilters([]EventFilter{
		{ResourceIDs: []string{"task-123", "task-456"}},
	})
	msg.ResourceType = "task"
	msg.ResourceID = "task-123"
	assert.True(t, client.MatchesFilter(msg))
	
	msg.ResourceID = "task-789"
	assert.False(t, client.MatchesFilter(msg))

	// Test with wildcard resource ID
	client.SetFilters([]EventFilter{
		{ResourceIDs: []string{"*"}},
	})
	assert.True(t, client.MatchesFilter(msg))

	// Test with combined filters (AND logic within a filter)
	client.SetFilters([]EventFilter{
		{
			EventTypes:    []string{"task_update"},
			ResourceTypes: []string{"task"},
			ResourceIDs:   []string{"task-123"},
		},
	})
	msg.Type = "task_update"
	msg.ResourceType = "task"
	msg.ResourceID = "task-123"
	assert.True(t, client.MatchesFilter(msg))
	
	msg.ResourceID = "task-456"
	assert.False(t, client.MatchesFilter(msg))

	// Test with multiple filters (OR logic between filters)
	client.SetFilters([]EventFilter{
		{EventTypes: []string{"task_update"}},
		{ResourceTypes: []string{"service"}},
	})
	msg.Type = "task_update"
	msg.ResourceType = "cluster"
	assert.True(t, client.MatchesFilter(msg)) // Matches first filter
	
	msg.Type = "cluster_update"
	msg.ResourceType = "service"
	assert.True(t, client.MatchesFilter(msg)) // Matches second filter
	
	msg.Type = "cluster_update"
	msg.ResourceType = "cluster"
	assert.False(t, client.MatchesFilter(msg)) // Matches neither filter
}

func TestWebSocketClient_handleMessage_SetFilters(t *testing.T) {
	client := &WebSocketClient{
		send:    make(chan WebSocketMessage, 10),
		id:      "test-client",
		filters: []EventFilter{},
	}

	// Test setFilters message
	filters := []EventFilter{
		{
			EventTypes:    []string{"task_update", "service_update"},
			ResourceTypes: []string{"task", "service"},
		},
	}
	
	filtersJSON, _ := json.Marshal(filters)
	setFiltersMsg := WebSocketMessage{
		Type:    "setFilters",
		ID:      "msg-789",
		Payload: filtersJSON,
	}
	
	client.handleMessage(setFiltersMsg)
	
	// Check filters were set
	assert.Equal(t, 1, len(client.filters))
	assert.Equal(t, filters[0].EventTypes, client.filters[0].EventTypes)
	assert.Equal(t, filters[0].ResourceTypes, client.filters[0].ResourceTypes)
	
	// Check confirmation was sent
	select {
	case msg := <-client.send:
		assert.Equal(t, "filtersSet", msg.Type)
		assert.Equal(t, "msg-789", msg.ID)
		assert.Equal(t, setFiltersMsg.Payload, msg.Payload)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No confirmation message received")
	}
}

func TestWebSocketHub_BroadcastWithFiltering(t *testing.T) {
	hub := NewWebSocketHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create clients with different filters
	clientWithTaskFilter := &WebSocketClient{
		hub:           hub,
		send:          make(chan WebSocketMessage, 10),
		id:            "task-filter-client",
		subscriptions: make(map[string]map[string]bool),
		filters: []EventFilter{
			{EventTypes: []string{"task_update"}},
		},
	}
	
	clientWithServiceFilter := &WebSocketClient{
		hub:           hub,
		send:          make(chan WebSocketMessage, 10),
		id:            "service-filter-client",
		subscriptions: make(map[string]map[string]bool),
		filters: []EventFilter{
			{EventTypes: []string{"service_update"}},
		},
	}
	
	clientWithNoFilter := &WebSocketClient{
		hub:           hub,
		send:          make(chan WebSocketMessage, 10),
		id:            "no-filter-client",
		subscriptions: make(map[string]map[string]bool),
		filters:       []EventFilter{},
	}

	hub.register <- clientWithTaskFilter
	hub.register <- clientWithServiceFilter
	hub.register <- clientWithNoFilter
	time.Sleep(10 * time.Millisecond)

	// Broadcast a task update
	taskMessage := WebSocketMessage{
		Type:         "task_update",
		ResourceType: "task",
		ResourceID:   "task-123",
		Payload:      []byte(`{"status":"RUNNING"}`),
		Timestamp:    time.Now(),
	}
	hub.BroadcastWithFiltering(taskMessage)

	// Client with task filter should receive it
	select {
	case msg := <-clientWithTaskFilter.send:
		assert.Equal(t, "task_update", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client with task filter did not receive task update")
	}

	// Client with service filter should not receive it
	select {
	case <-clientWithServiceFilter.send:
		t.Fatal("Client with service filter should not receive task update")
	case <-time.After(50 * time.Millisecond):
		// Expected timeout
	}

	// Client with no filter should receive it
	select {
	case msg := <-clientWithNoFilter.send:
		assert.Equal(t, "task_update", msg.Type)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Client with no filter did not receive task update")
	}
}

func TestWebSocketSubscription(t *testing.T) {
	hub := NewWebSocketHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)

	// Create and register clients
	subscribedClient := &WebSocketClient{
		hub:           hub,
		send:          make(chan WebSocketMessage, 10),
		id:            "subscribed-client",
		subscriptions: make(map[string]map[string]bool),
		filters:       []EventFilter{},
	}
	
	unsubscribedClient := &WebSocketClient{
		hub:           hub,
		send:          make(chan WebSocketMessage, 10),
		id:            "unsubscribed-client",
		subscriptions: make(map[string]map[string]bool),
		filters:       []EventFilter{},
	}

	hub.register <- subscribedClient
	hub.register <- unsubscribedClient
	time.Sleep(10 * time.Millisecond)

	// Subscribe one client to task-123
	subscribedClient.Subscribe("task", "task-123")

	// Test targeted broadcast
	hub.BroadcastTaskUpdateToSubscribed("task-123", map[string]interface{}{
		"status": "RUNNING",
		"cpu":    "50%",
	})

	// Only subscribed client should receive the update
	select {
	case msg := <-subscribedClient.send:
		assert.Equal(t, "task_update", msg.Type)
		assert.NotNil(t, msg.Payload)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Subscribed client did not receive task update")
	}

	// Unsubscribed client should not receive anything
	select {
	case <-unsubscribedClient.send:
		t.Fatal("Unsubscribed client should not receive task update")
	case <-time.After(50 * time.Millisecond):
		// Expected timeout
	}
}

func TestWebSocketClient_Subscribe(t *testing.T) {
	client := &WebSocketClient{
		id:            "test-client",
		subscriptions: make(map[string]map[string]bool),
	}

	// Test subscription
	client.Subscribe("task", "task-123")
	assert.True(t, client.IsSubscribed("task", "task-123"))
	assert.False(t, client.IsSubscribed("task", "task-456"))
	assert.False(t, client.IsSubscribed("service", "service-123"))

	// Test wildcard subscription
	client.Subscribe("service", "*")
	assert.True(t, client.IsSubscribed("service", "service-123"))
	assert.True(t, client.IsSubscribed("service", "any-service"))

	// Test unsubscription
	client.Unsubscribe("task", "task-123")
	assert.False(t, client.IsSubscribed("task", "task-123"))

	// Test unsubscribing from all resources of a type
	client.Subscribe("cluster", "cluster-1")
	client.Subscribe("cluster", "cluster-2")
	client.Unsubscribe("cluster", "cluster-1")
	assert.False(t, client.IsSubscribed("cluster", "cluster-1"))
	assert.True(t, client.IsSubscribed("cluster", "cluster-2"))
	
	client.Unsubscribe("cluster", "cluster-2")
	assert.False(t, client.IsSubscribed("cluster", "cluster-2"))
}

func TestWebSocketClient_handleMessage_Subscribe(t *testing.T) {
	client := &WebSocketClient{
		send:          make(chan WebSocketMessage, 10),
		id:            "test-client",
		subscriptions: make(map[string]map[string]bool),
		filters:       []EventFilter{},
	}

	// Test subscribe message
	subscribeMsg := WebSocketMessage{
		Type:    "subscribe",
		ID:      "msg-123",
		Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
	}
	
	client.handleMessage(subscribeMsg)
	
	// Check subscription was added
	assert.True(t, client.IsSubscribed("task", "task-123"))
	
	// Check confirmation was sent
	select {
	case msg := <-client.send:
		assert.Equal(t, "subscribed", msg.Type)
		assert.Equal(t, "msg-123", msg.ID)
		assert.Equal(t, subscribeMsg.Payload, msg.Payload)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No confirmation message received")
	}

	// Test unsubscribe message
	unsubscribeMsg := WebSocketMessage{
		Type:    "unsubscribe",
		ID:      "msg-456",
		Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
	}
	
	client.handleMessage(unsubscribeMsg)
	
	// Check subscription was removed
	assert.False(t, client.IsSubscribed("task", "task-123"))
	
	// Check confirmation was sent
	select {
	case msg := <-client.send:
		assert.Equal(t, "unsubscribed", msg.Type)
		assert.Equal(t, "msg-456", msg.ID)
		assert.Equal(t, unsubscribeMsg.Payload, msg.Payload)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("No confirmation message received")
	}
}