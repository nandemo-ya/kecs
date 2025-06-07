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

func TestWebSocketAuthentication(t *testing.T) {
	tests := []struct {
		name          string
		config        *WebSocketConfig
		authHeader    string
		expectConnect bool
	}{
		{
			name: "auth disabled - connection allowed",
			config: &WebSocketConfig{
				AuthEnabled: false,
			},
			authHeader:    "",
			expectConnect: true,
		},
		{
			name: "auth enabled - valid token",
			config: &WebSocketConfig{
				AuthEnabled: true,
				AuthFunc: func(r *http.Request) (*AuthInfo, bool) {
					auth := r.Header.Get("Authorization")
					if auth == "Bearer valid-token" {
						return &AuthInfo{
							UserID:   "user-123",
							Username: "testuser",
							Roles:    []string{"user"},
						}, true
					}
					return nil, false
				},
			},
			authHeader:    "Bearer valid-token",
			expectConnect: true,
		},
		{
			name: "auth enabled - invalid token",
			config: &WebSocketConfig{
				AuthEnabled: true,
				AuthFunc: func(r *http.Request) (*AuthInfo, bool) {
					auth := r.Header.Get("Authorization")
					if auth == "Bearer valid-token" {
						return &AuthInfo{
							UserID:   "user-123",
							Username: "testuser",
							Roles:    []string{"user"},
						}, true
					}
					return nil, false
				},
			},
			authHeader:    "Bearer invalid-token",
			expectConnect: false,
		},
		{
			name: "auth enabled - no token",
			config: &WebSocketConfig{
				AuthEnabled: true,
			},
			authHeader:    "",
			expectConnect: false,
		},
		{
			name: "auth enabled - default auth func",
			config: &WebSocketConfig{
				AuthEnabled: true,
				// Use default auth func
			},
			authHeader:    "Bearer test-token-12345678",
			expectConnect: true,
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
			if tt.authHeader != "" {
				header.Set("Authorization", tt.authHeader)
			}
			
			// Attempt connection
			conn, resp, err := dialer.Dial(wsURL, header)
			
			if tt.expectConnect {
				require.NoError(t, err)
				require.NotNil(t, conn)
				defer conn.Close()
				
				// Connection was successful - authentication passed
				// Give time for the client to be registered
				time.Sleep(50 * time.Millisecond)
			} else {
				// Connection should be rejected
				assert.Error(t, err)
				if resp != nil {
					assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
				}
			}
		})
	}
}

func TestWebSocketAuthorization(t *testing.T) {
	// Create hub with auth enabled
	config := &WebSocketConfig{
		AuthEnabled: true,
		AuthFunc: func(r *http.Request) (*AuthInfo, bool) {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				if token == "admin-token" {
					return &AuthInfo{
						UserID:   "admin-123",
						Username: "admin",
						Roles:    []string{"admin"},
						Permissions: []string{"*:*"},
					}, true
				} else if token == "user-token" {
					return &AuthInfo{
						UserID:   "user-456",
						Username: "user",
						Roles:    []string{"user"},
						Permissions: []string{
							"task:subscribe",
							"task:unsubscribe",
							"service:subscribe",
						},
					}, true
				}
			}
			return nil, false
		},
		AuthorizeFunc: func(authInfo *AuthInfo, operation string, resource string) bool {
			// Check permissions
			requiredPermission := fmt.Sprintf("%s:%s", getResourceType(resource), operation)
			
			for _, perm := range authInfo.Permissions {
				if perm == requiredPermission || perm == "*:*" {
					return true
				}
			}
			
			return false
		},
	}
	
	hub := NewWebSocketHubWithConfig(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	
	// Give hub time to start
	time.Sleep(50 * time.Millisecond)
	
	// Create server
	server := &Server{
		webSocketHub: hub,
	}
	
	// Create test server
	handler := server.HandleWebSocket(hub)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Test admin user - should be able to do everything
	t.Run("admin user authorization", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
		header := http.Header{}
		header.Set("Authorization", "Bearer admin-token")
		
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		require.NoError(t, err)
		defer conn.Close()
		
		// Give time for connection to be established
		time.Sleep(50 * time.Millisecond)
		
		// Test subscribing to a task
		subscribeMsg := WebSocketMessage{
			Type:    "subscribe",
			ID:      "msg-1",
			Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
		}
		err = conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)
		
		// Read response
		var response WebSocketMessage
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		err = conn.ReadJSON(&response)
		require.NoError(t, err)
		assert.Equal(t, "subscribed", response.Type)
	})
	
	// Test regular user - limited permissions
	t.Run("regular user authorization", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
		header := http.Header{}
		header.Set("Authorization", "Bearer user-token")
		
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
		require.NoError(t, err)
		defer conn.Close()
		
		// Give time for connection to be established
		time.Sleep(50 * time.Millisecond)
		
		// Set read deadline to avoid hanging
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		
		// Test subscribing to a task (allowed)
		subscribeMsg := WebSocketMessage{
			Type:    "subscribe",
			ID:      "msg-2",
			Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
		}
		err = conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)
		
		// Should receive confirmation
		var response WebSocketMessage
		err = conn.ReadJSON(&response)
		require.NoError(t, err)
		assert.Equal(t, "subscribed", response.Type)
		
		// Test subscribing to a cluster (not allowed)
		subscribeMsg = WebSocketMessage{
			Type:    "subscribe",
			ID:      "msg-3",
			Payload: []byte(`{"resourceType":"cluster","resourceId":"cluster-123"}`),
		}
		err = conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)
		
		// Should receive error
		err = conn.ReadJSON(&response)
		require.NoError(t, err)
		assert.Equal(t, "error", response.Type)
		
		var errorPayload map[string]string
		err = json.Unmarshal(response.Payload, &errorPayload)
		require.NoError(t, err)
		assert.Equal(t, "unauthorized", errorPayload["error"])
		
		// Test setting filters (not allowed)
		setFiltersMsg := WebSocketMessage{
			Type:    "setFilters",
			ID:      "msg-4",
			Payload: []byte(`[{"eventTypes":["task_update"]}]`),
		}
		err = conn.WriteJSON(setFiltersMsg)
		require.NoError(t, err)
		
		// Should receive error
		err = conn.ReadJSON(&response)
		require.NoError(t, err)
		assert.Equal(t, "error", response.Type)
	})
}

func TestWebSocketTokenRefresh(t *testing.T) {
	// Track token validity
	validTokens := map[string]bool{
		"initial-token": true,
		"refresh-token": true,
	}
	
	config := &WebSocketConfig{
		AuthEnabled: true,
		AuthFunc: func(r *http.Request) (*AuthInfo, bool) {
			auth := r.Header.Get("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				if valid, ok := validTokens[token]; ok && valid {
					return &AuthInfo{
						UserID:   "user-123",
						Username: "testuser",
						Roles:    []string{"user"},
						Permissions: []string{"task:subscribe"},
					}, true
				}
			}
			return nil, false
		},
	}
	
	hub := NewWebSocketHubWithConfig(config)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go hub.Run(ctx)
	
	// Give hub time to start
	time.Sleep(50 * time.Millisecond)
	
	// Create server
	server := &Server{
		webSocketHub: hub,
	}
	
	// Create test server
	handler := server.HandleWebSocket(hub)
	ts := httptest.NewServer(handler)
	defer ts.Close()
	
	// Connect with initial token
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	header := http.Header{}
	header.Set("Authorization", "Bearer initial-token")
	
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	require.NoError(t, err)
	defer conn.Close()
	
	// Invalidate initial token
	validTokens["initial-token"] = false
	
	// Try to authenticate with new token
	authMsg := WebSocketMessage{
		Type:    "authenticate",
		ID:      "auth-1",
		Payload: []byte(`{"token":"refresh-token"}`),
	}
	err = conn.WriteJSON(authMsg)
	require.NoError(t, err)
	
	// Should receive success response
	var response WebSocketMessage
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "authenticated", response.Type)
	
	var authPayload map[string]interface{}
	err = json.Unmarshal(response.Payload, &authPayload)
	require.NoError(t, err)
	assert.Equal(t, "testuser", authPayload["username"])
	
	// Try to authenticate with invalid token
	authMsg = WebSocketMessage{
		Type:    "authenticate",
		ID:      "auth-2",
		Payload: []byte(`{"token":"invalid-token"}`),
	}
	err = conn.WriteJSON(authMsg)
	require.NoError(t, err)
	
	// Should receive error response
	err = conn.ReadJSON(&response)
	require.NoError(t, err)
	assert.Equal(t, "error", response.Type)
	
	var errorPayload map[string]string
	err = json.Unmarshal(response.Payload, &errorPayload)
	require.NoError(t, err)
	assert.Equal(t, "authentication_failed", errorPayload["error"])
}

func TestDefaultAuthFunc(t *testing.T) {
	tests := []struct {
		name      string
		authHeader string
		expectAuth bool
		expectUser string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer test-token-12345678",
			expectAuth: true,
			expectUser: "user-test-tok",
		},
		{
			name:       "no authorization header",
			authHeader: "",
			expectAuth: false,
		},
		{
			name:       "invalid format",
			authHeader: "Basic dXNlcjpwYXNz",
			expectAuth: false,
		},
		{
			name:       "empty bearer token",
			authHeader: "Bearer ",
			expectAuth: false,
		},
		{
			name:       "bearer without space",
			authHeader: "Bearertoken",
			expectAuth: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			authInfo, authenticated := defaultAuthFunc(req)
			
			assert.Equal(t, tt.expectAuth, authenticated)
			
			if tt.expectAuth {
				assert.NotNil(t, authInfo)
				assert.Equal(t, tt.expectUser, authInfo.Username)
				assert.Contains(t, authInfo.Roles, "user")
				assert.Contains(t, authInfo.Permissions, "websocket:connect")
				assert.Contains(t, authInfo.Permissions, "task:read")
				assert.Contains(t, authInfo.Permissions, "service:read")
			} else {
				assert.Nil(t, authInfo)
			}
		})
	}
}

func TestClientAuthorization(t *testing.T) {
	hub := &WebSocketHub{
		config: &WebSocketConfig{
			AuthEnabled: true,
		},
	}
	
	tests := []struct {
		name      string
		authInfo  *AuthInfo
		operation string
		resource  string
		expected  bool
	}{
		{
			name: "admin can do everything",
			authInfo: &AuthInfo{
				Roles: []string{"admin"},
			},
			operation: "delete",
			resource:  "task/task-123",
			expected:  true,
		},
		{
			name: "wildcard permission",
			authInfo: &AuthInfo{
				Permissions: []string{"*:*"},
			},
			operation: "delete",
			resource:  "task/task-123",
			expected:  true,
		},
		{
			name: "exact permission match",
			authInfo: &AuthInfo{
				Permissions: []string{"task:delete"},
			},
			operation: "delete",
			resource:  "task/task-123",
			expected:  true,
		},
		{
			name: "resource wildcard permission",
			authInfo: &AuthInfo{
				Permissions: []string{"*:read"},
			},
			operation: "read",
			resource:  "service/service-123",
			expected:  true,
		},
		{
			name: "operation wildcard permission",
			authInfo: &AuthInfo{
				Permissions: []string{"task:*"},
			},
			operation: "update",
			resource:  "task/task-123",
			expected:  true,
		},
		{
			name: "no matching permission",
			authInfo: &AuthInfo{
				Permissions: []string{"task:read", "service:read"},
			},
			operation: "delete",
			resource:  "task/task-123",
			expected:  false,
		},
		{
			name:      "no auth info",
			authInfo:  nil,
			operation: "delete",
			resource:  "task/task-123",
			expected:  false, // When auth is enabled but no auth info, deny
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &WebSocketClient{
				hub:      hub,
				authInfo: tt.authInfo,
			}
			
			// Test with auth enabled
			result := client.IsAuthorized(tt.operation, tt.resource)
			assert.Equal(t, tt.expected, result)
			
			// Test with auth disabled
			hub.config.AuthEnabled = false
			result = client.IsAuthorized(tt.operation, tt.resource)
			assert.True(t, result) // Always true when auth is disabled
			
			hub.config.AuthEnabled = true
		})
	}
}

func TestGetResourceType(t *testing.T) {
	tests := []struct {
		resource string
		expected string
	}{
		{"task/task-123", "task"},
		{"service/my-service/endpoint", "service"},
		{"cluster", "cluster"},
		{"", ""},
		{"/", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			result := getResourceType(tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}