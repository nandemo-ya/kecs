package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocketAuthentication", func() {
	var (
		hub    *WebSocketHub
		server *Server
		ts     *httptest.Server
		wsURL  string
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
		if ts != nil {
			ts.Close()
		}
	})

	Context("when testing authentication", func() {
		DescribeTable("authentication scenarios",
			func(config *WebSocketConfig, authHeader string, expectConnect bool) {
				// Create hub with config
				hub = NewWebSocketHubWithConfig(config)
				
				// Start hub in background
				go hub.Run(ctx)
				
				// Give hub time to start
				time.Sleep(10 * time.Millisecond)
				
				// Create server
				server = &Server{
					webSocketHub: hub,
				}
				
				// Create test server
				handler := server.HandleWebSocket(hub)
				ts = httptest.NewServer(handler)
				
				// Convert http to ws
				wsURL = "ws" + strings.TrimPrefix(ts.URL, "http")
				
				// Create WebSocket connection request
				dialer := websocket.DefaultDialer
				header := http.Header{}
				if authHeader != "" {
					header.Set("Authorization", authHeader)
				}
				
				// Attempt connection
				conn, resp, err := dialer.Dial(wsURL, header)
				
				if expectConnect {
					Expect(err).NotTo(HaveOccurred())
					Expect(conn).NotTo(BeNil())
					defer conn.Close()
					
					// Connection was successful - authentication passed
					// Give time for the client to be registered
					time.Sleep(50 * time.Millisecond)
				} else {
					// Connection should be rejected
					Expect(err).To(HaveOccurred())
					if resp != nil {
						Expect(resp.StatusCode).To(Equal(http.StatusUnauthorized))
					}
				}
			},
			Entry("auth disabled - connection allowed",
				&WebSocketConfig{
					AuthEnabled: false,
				},
				"",
				true,
			),
			Entry("auth enabled - valid token",
				&WebSocketConfig{
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
				"Bearer valid-token",
				true,
			),
			Entry("auth enabled - invalid token",
				&WebSocketConfig{
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
				"Bearer invalid-token",
				false,
			),
			Entry("auth enabled - no token",
				&WebSocketConfig{
					AuthEnabled: true,
				},
				"",
				false,
			),
			Entry("auth enabled - default auth func",
				&WebSocketConfig{
					AuthEnabled: true,
					// Use default auth func
				},
				"Bearer test-token-12345678",
				true,
			),
		)
	})

	Context("when testing authorization", func() {
		BeforeEach(func() {
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
			
			hub = NewWebSocketHubWithConfig(config)
			go hub.Run(ctx)
			
			// Give hub time to start
			time.Sleep(50 * time.Millisecond)
			
			// Create server
			server = &Server{
				webSocketHub: hub,
			}
			
			// Create test server
			handler := server.HandleWebSocket(hub)
			ts = httptest.NewServer(handler)
			wsURL = "ws" + strings.TrimPrefix(ts.URL, "http")
		})

		It("should allow admin user to do everything", func() {
			header := http.Header{}
			header.Set("Authorization", "Bearer admin-token")
			
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			Expect(err).NotTo(HaveOccurred())
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
			Expect(err).NotTo(HaveOccurred())
			
			// Read response
			var response WebSocketMessage
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			err = conn.ReadJSON(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Type).To(Equal("subscribed"))
		})

		It("should enforce permissions for regular user", func() {
			header := http.Header{}
			header.Set("Authorization", "Bearer user-token")
			
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			Expect(err).NotTo(HaveOccurred())
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
			Expect(err).NotTo(HaveOccurred())
			
			// Should receive confirmation
			var response WebSocketMessage
			err = conn.ReadJSON(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Type).To(Equal("subscribed"))
			
			// Test subscribing to a cluster (not allowed)
			subscribeMsg = WebSocketMessage{
				Type:    "subscribe",
				ID:      "msg-3",
				Payload: []byte(`{"resourceType":"cluster","resourceId":"cluster-123"}`),
			}
			err = conn.WriteJSON(subscribeMsg)
			Expect(err).NotTo(HaveOccurred())
			
			// Should receive error
			err = conn.ReadJSON(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Type).To(Equal("error"))
			
			var errorPayload map[string]string
			err = json.Unmarshal(response.Payload, &errorPayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorPayload["error"]).To(Equal("unauthorized"))
			
			// Test setting filters (not allowed)
			setFiltersMsg := WebSocketMessage{
				Type:    "setFilters",
				ID:      "msg-4",
				Payload: []byte(`[{"eventTypes":["task_update"]}]`),
			}
			err = conn.WriteJSON(setFiltersMsg)
			Expect(err).NotTo(HaveOccurred())
			
			// Should receive error
			err = conn.ReadJSON(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Type).To(Equal("error"))
		})
	})

	Context("when testing token refresh", func() {
		var validTokens map[string]bool

		BeforeEach(func() {
			// Track token validity
			validTokens = map[string]bool{
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
			
			hub = NewWebSocketHubWithConfig(config)
			go hub.Run(ctx)
			
			// Give hub time to start
			time.Sleep(50 * time.Millisecond)
			
			// Create server
			server = &Server{
				webSocketHub: hub,
			}
			
			// Create test server
			handler := server.HandleWebSocket(hub)
			ts = httptest.NewServer(handler)
			wsURL = "ws" + strings.TrimPrefix(ts.URL, "http")
		})

		It("should handle token refresh", func() {
			// Connect with initial token
			header := http.Header{}
			header.Set("Authorization", "Bearer initial-token")
			
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
			Expect(err).NotTo(HaveOccurred())
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
			Expect(err).NotTo(HaveOccurred())
			
			// Should receive success response
			var response WebSocketMessage
			err = conn.ReadJSON(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Type).To(Equal("authenticated"))
			
			var authPayload map[string]interface{}
			err = json.Unmarshal(response.Payload, &authPayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(authPayload["username"]).To(Equal("testuser"))
			
			// Try to authenticate with invalid token
			authMsg = WebSocketMessage{
				Type:    "authenticate",
				ID:      "auth-2",
				Payload: []byte(`{"token":"invalid-token"}`),
			}
			err = conn.WriteJSON(authMsg)
			Expect(err).NotTo(HaveOccurred())
			
			// Should receive error response
			err = conn.ReadJSON(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Type).To(Equal("error"))
			
			var errorPayload map[string]string
			err = json.Unmarshal(response.Payload, &errorPayload)
			Expect(err).NotTo(HaveOccurred())
			Expect(errorPayload["error"]).To(Equal("authentication_failed"))
		})
	})

	Context("when testing default auth function", func() {
		DescribeTable("authentication scenarios",
			func(authHeader string, expectAuth bool, expectUser string) {
				req := httptest.NewRequest("GET", "/", nil)
				if authHeader != "" {
					req.Header.Set("Authorization", authHeader)
				}
				
				authInfo, authenticated := defaultAuthFunc(req)
				
				Expect(authenticated).To(Equal(expectAuth))
				
				if expectAuth {
					Expect(authInfo).NotTo(BeNil())
					Expect(authInfo.Username).To(Equal(expectUser))
					Expect(authInfo.Roles).To(ContainElement("user"))
					Expect(authInfo.Permissions).To(ContainElements("websocket:connect", "task:read", "service:read"))
				} else {
					Expect(authInfo).To(BeNil())
				}
			},
			Entry("valid bearer token", "Bearer test-token-12345678", true, "user-test-tok"),
			Entry("no authorization header", "", false, ""),
			Entry("invalid format", "Basic dXNlcjpwYXNz", false, ""),
			Entry("empty bearer token", "Bearer ", false, ""),
			Entry("bearer without space", "Bearertoken", false, ""),
		)
	})

	Context("when testing client authorization", func() {
		var hub *WebSocketHub

		BeforeEach(func() {
			hub = &WebSocketHub{
				config: &WebSocketConfig{
					AuthEnabled: true,
				},
			}
		})

		DescribeTable("authorization scenarios",
			func(authInfo *AuthInfo, operation string, resource string, expected bool) {
				client := &WebSocketClient{
					hub:      hub,
					authInfo: authInfo,
				}
				
				// Test with auth enabled
				result := client.IsAuthorized(operation, resource)
				Expect(result).To(Equal(expected))
				
				// Test with auth disabled
				hub.config.AuthEnabled = false
				result = client.IsAuthorized(operation, resource)
				Expect(result).To(BeTrue()) // Always true when auth is disabled
				
				hub.config.AuthEnabled = true
			},
			Entry("admin can do everything",
				&AuthInfo{Roles: []string{"admin"}},
				"delete",
				"task/task-123",
				true,
			),
			Entry("wildcard permission",
				&AuthInfo{Permissions: []string{"*:*"}},
				"delete",
				"task/task-123",
				true,
			),
			Entry("exact permission match",
				&AuthInfo{Permissions: []string{"task:delete"}},
				"delete",
				"task/task-123",
				true,
			),
			Entry("resource wildcard permission",
				&AuthInfo{Permissions: []string{"*:read"}},
				"read",
				"service/service-123",
				true,
			),
			Entry("operation wildcard permission",
				&AuthInfo{Permissions: []string{"task:*"}},
				"update",
				"task/task-123",
				true,
			),
			Entry("no matching permission",
				&AuthInfo{Permissions: []string{"task:read", "service:read"}},
				"delete",
				"task/task-123",
				false,
			),
			Entry("no auth info",
				nil,
				"delete",
				"task/task-123",
				false, // When auth is enabled but no auth info, deny
			),
		)
	})

	Context("when testing getResourceType function", func() {
		DescribeTable("resource type extraction",
			func(resource, expected string) {
				result := getResourceType(resource)
				Expect(result).To(Equal(expected))
			},
			Entry("task resource", "task/task-123", "task"),
			Entry("service with sub-path", "service/my-service/endpoint", "service"),
			Entry("cluster resource", "cluster", "cluster"),
			Entry("empty resource", "", ""),
			Entry("just slash", "/", ""),
		)
	})
})