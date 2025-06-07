package api

import (
	"context"
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
		hub:  hub,
		send: make(chan WebSocketMessage, 1),
		id:   "test-client",
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
				
				// Test ping-pong
				msg := WebSocketMessage{
					Type: "ping",
					ID:   "test-123",
				}
				err = conn.WriteJSON(msg)
				assert.NoError(t, err)
				
				// Read pong response
				var response WebSocketMessage
				err = conn.ReadJSON(&response)
				assert.NoError(t, err)
				assert.Equal(t, "pong", response.Type)
				assert.Equal(t, "test-123", response.ID)
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
		send: make(chan WebSocketMessage, 10),
		id:   "test-client",
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
			hub:  hub,
			send: make(chan WebSocketMessage, 10),
			id:   fmt.Sprintf("client-%d", i),
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