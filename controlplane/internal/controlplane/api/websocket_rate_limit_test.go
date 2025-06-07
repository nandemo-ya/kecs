package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	config := &RateLimitConfig{
		MessagesPerMinute:       60, // 1 per second
		BurstSize:               5,
		GlobalMessagesPerMinute: 120, // 2 per second globally
		GlobalBurstSize:         10,
		BypassRoles:             []string{"admin"},
	}

	limiter := NewRateLimiter(config)

	t.Run("per-connection rate limiting", func(t *testing.T) {
		clientID := "test-client"
		authInfo := &AuthInfo{UserID: "user1", Roles: []string{"user"}}

		// Should allow burst
		for i := 0; i < 5; i++ {
			assert.True(t, limiter.Allow(clientID, authInfo))
		}

		// Should be rate limited after burst
		assert.False(t, limiter.Allow(clientID, authInfo))
	})

	t.Run("bypass role", func(t *testing.T) {
		clientID := "admin-client"
		authInfo := &AuthInfo{UserID: "admin1", Roles: []string{"admin"}}

		// Admin should bypass rate limits
		for i := 0; i < 20; i++ {
			assert.True(t, limiter.Allow(clientID, authInfo))
		}
	})

	t.Run("cleanup", func(t *testing.T) {
		// Create a new limiter for this test to avoid global rate limit issues
		cleanupConfig := &RateLimitConfig{
			MessagesPerMinute:       60, // 1 per second
			BurstSize:               5,
			GlobalMessagesPerMinute: 600, // 10 per second globally
			GlobalBurstSize:         50,  // Large global burst
			BypassRoles:             []string{"admin"},
		}
		cleanupLimiter := NewRateLimiter(cleanupConfig)
		
		clientID := "cleanup-client"
		authInfo := &AuthInfo{UserID: "user2", Roles: []string{"user"}}

		// Use up the burst
		for i := 0; i < 5; i++ {
			cleanupLimiter.Allow(clientID, authInfo)
		}

		// Should be rate limited after burst
		assert.False(t, cleanupLimiter.Allow(clientID, authInfo))

		// Remove the limiter
		cleanupLimiter.Remove(clientID)

		// Should get a fresh limiter with full burst after removal
		for i := 0; i < 5; i++ {
			assert.True(t, cleanupLimiter.Allow(clientID, authInfo))
		}
	})
}

func TestConnectionLimiter(t *testing.T) {
	config := &ConnectionLimitConfig{
		MaxConnectionsPerUser: 2,
		MaxConnectionsPerIP:   3,
		MaxTotalConnections:   5,
		BypassRoles:           []string{"admin"},
	}

	limiter := NewConnectionLimiter(config)

	t.Run("per-user limit", func(t *testing.T) {
		req1 := httptest.NewRequest("GET", "/ws", nil)
		req1.RemoteAddr = "192.168.1.1:1234"
		
		req2 := httptest.NewRequest("GET", "/ws", nil)
		req2.RemoteAddr = "192.168.1.2:1234"
		
		req3 := httptest.NewRequest("GET", "/ws", nil)
		req3.RemoteAddr = "192.168.1.3:1234"

		authInfo := &AuthInfo{UserID: "user1", Roles: []string{"user"}}

		// First two connections should be allowed
		assert.True(t, limiter.CanConnect(req1, authInfo))
		limiter.AddConnection(req1, authInfo)

		assert.True(t, limiter.CanConnect(req2, authInfo))
		limiter.AddConnection(req2, authInfo)

		// Third connection should be denied
		assert.False(t, limiter.CanConnect(req3, authInfo))

		// Remove one connection
		limiter.RemoveConnection(req1, authInfo)

		// Now third connection should be allowed
		assert.True(t, limiter.CanConnect(req3, authInfo))
	})

	t.Run("per-IP limit", func(t *testing.T) {
		// Reset limiter
		limiter = NewConnectionLimiter(config)

		req := httptest.NewRequest("GET", "/ws", nil)
		req.RemoteAddr = "192.168.1.1:1234"

		// Different users from same IP
		for i := 0; i < 3; i++ {
			authInfo := &AuthInfo{UserID: string(rune('a' + i)), Roles: []string{"user"}}
			assert.True(t, limiter.CanConnect(req, authInfo))
			limiter.AddConnection(req, authInfo)
		}

		// Fourth connection from same IP should be denied
		authInfo := &AuthInfo{UserID: "user4", Roles: []string{"user"}}
		assert.False(t, limiter.CanConnect(req, authInfo))
	})

	t.Run("bypass role", func(t *testing.T) {
		// Reset limiter
		limiter = NewConnectionLimiter(config)

		req := httptest.NewRequest("GET", "/ws", nil)
		req.RemoteAddr = "192.168.1.1:1234"

		// Fill up the limits with regular users
		for i := 0; i < 5; i++ {
			authInfo := &AuthInfo{UserID: string(rune('a' + i)), Roles: []string{"user"}}
			limiter.AddConnection(req, authInfo)
		}

		// Admin should still be able to connect
		adminAuth := &AuthInfo{UserID: "admin1", Roles: []string{"admin"}}
		assert.True(t, limiter.CanConnect(req, adminAuth))
	})

	t.Run("connection stats", func(t *testing.T) {
		// Reset limiter
		limiter = NewConnectionLimiter(config)

		req1 := httptest.NewRequest("GET", "/ws", nil)
		req1.RemoteAddr = "192.168.1.1:1234"
		
		req2 := httptest.NewRequest("GET", "/ws", nil)
		req2.RemoteAddr = "192.168.1.2:1234"

		authInfo1 := &AuthInfo{UserID: "user1"}
		authInfo2 := &AuthInfo{UserID: "user2"}

		limiter.AddConnection(req1, authInfo1)
		limiter.AddConnection(req2, authInfo2)

		stats := limiter.GetConnectionStats()
		assert.Equal(t, 2, stats["total_connections"])
		assert.Equal(t, 2, stats["users_connected"])
		assert.Equal(t, 2, stats["unique_ips"])
	})
}

func TestWebSocketRateLimiting(t *testing.T) {
	// Create hub with rate limiting
	config := &WebSocketConfig{
		AuthEnabled: false,
		RateLimitConfig: &RateLimitConfig{
			MessagesPerMinute:       12, // 1 message per 5 seconds
			BurstSize:               2,  // Allow 2 messages burst
			GlobalMessagesPerMinute: 60, // Allow 1 per second globally
			GlobalBurstSize:         10, // Allow burst of 10 globally
		},
	}

	hub := NewWebSocketHubWithConfig(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create server
	server := &Server{
		webSocketHub: hub,
	}

	// Create test server
	handler := server.HandleWebSocket(hub)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// Connect
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Allow connection to establish
	time.Sleep(50 * time.Millisecond)

	// Send burst of messages
	for i := 0; i < 3; i++ {
		msg := WebSocketMessage{
			Type: "ping",
			ID:   string(rune('1' + i)),
		}
		err := conn.WriteJSON(msg)
		require.NoError(t, err)
	}

	// Read responses
	responsesReceived := 0
	errorReceived := false

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	for i := 0; i < 3; i++ {
		var response WebSocketMessage
		err := conn.ReadJSON(&response)
		if err != nil {
			break
		}

		if response.Type == "pong" {
			responsesReceived++
		} else if response.Type == "error" {
			var errorPayload map[string]string
			err := json.Unmarshal(response.Payload, &errorPayload)
			require.NoError(t, err)
			if errorPayload["error"] == "rate_limit_exceeded" {
				errorReceived = true
			}
		}
	}

	// Should receive 2 pongs (burst limit) and 1 error
	assert.Equal(t, 2, responsesReceived)
	assert.True(t, errorReceived)
}

func TestWebSocketConnectionLimits(t *testing.T) {
	// Create hub with connection limits
	config := &WebSocketConfig{
		AuthEnabled: true,
		AuthFunc: func(r *http.Request) (*AuthInfo, bool) {
			token := r.Header.Get("X-User-ID")
			if token == "" {
				return nil, false
			}
			return &AuthInfo{
				UserID:   token,
				Username: token,
				Roles:    []string{"user"},
			}, true
		},
		ConnectionLimitConfig: &ConnectionLimitConfig{
			MaxConnectionsPerUser: 2,
			MaxTotalConnections:   3,
		},
	}

	hub := NewWebSocketHubWithConfig(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)

	// Create server
	server := &Server{
		webSocketHub: hub,
	}

	// Create test server
	handler := server.HandleWebSocket(hub)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Keep connections alive across tests
	var connections []*websocket.Conn
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()

	t.Run("per-user connection limit", func(t *testing.T) {
		// Create headers for user1
		header1 := http.Header{}
		header1.Set("X-User-ID", "user1")

		// First two connections should succeed
		conn1, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
		require.NoError(t, err)
		connections = append(connections, conn1)

		conn2, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
		require.NoError(t, err)
		connections = append(connections, conn2)

		// Third connection should fail
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header1)
		assert.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		}
	})

	t.Run("total connection limit", func(t *testing.T) {
		// Create connection for user2 (total would be 3 with the 2 from user1)
		header2 := http.Header{}
		header2.Set("X-User-ID", "user2")

		conn3, _, err := websocket.DefaultDialer.Dial(wsURL, header2)
		require.NoError(t, err)
		connections = append(connections, conn3)

		// Fourth total connection should fail (we have 3 connections already)
		header3 := http.Header{}
		header3.Set("X-User-ID", "user3")
		
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, header3)
		assert.Error(t, err)
		if resp != nil {
			assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
		}
	})
}

func TestWebSocketInactiveConnectionCleanup(t *testing.T) {
	// Test that inactive connections are properly tracked
	config := &WebSocketConfig{
		AuthEnabled: false,
		ConnectionLimitConfig: &ConnectionLimitConfig{
			ConnectionTimeout: 100 * time.Millisecond,
		},
	}

	hub := NewWebSocketHubWithConfig(config)
	
	// Create a mock client
	client := &WebSocketClient{
		hub:          hub,
		lastActivity: time.Now().Add(-200 * time.Millisecond), // 200ms ago
	}
	
	// Test IsActive method
	assert.False(t, client.IsActive(), "Client should be inactive after timeout")
	
	// Update activity
	client.updateLastActivity()
	assert.True(t, client.IsActive(), "Client should be active after update")
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name          string
		headers       map[string]string
		remoteAddr    string
		expectedIP    string
	}{
		{
			name: "X-Forwarded-For single IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			remoteAddr: "10.0.0.1:1234",
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.2, 10.0.0.3",
			},
			remoteAddr: "10.0.0.1:1234",
			expectedIP: "192.168.1.1",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.2",
			},
			remoteAddr: "10.0.0.1:1234",
			expectedIP: "192.168.1.2",
		},
		{
			name:       "RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.3:1234",
			expectedIP: "192.168.1.3",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.4",
			expectedIP: "192.168.1.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			
			ip := getClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}