package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("WebSocketRateLimit", func() {
	Context("when testing rate limiter", func() {
		var (
			config  *RateLimitConfig
			limiter *RateLimiter
		)

		BeforeEach(func() {
			config = &RateLimitConfig{
				MessagesPerMinute:       60, // 1 per second
				BurstSize:               5,
				GlobalMessagesPerMinute: 120, // 2 per second globally
				GlobalBurstSize:         10,
				BypassRoles:             []string{"admin"},
			}
			limiter = NewRateLimiter(config)
		})

		It("should enforce per-connection rate limiting", func() {
			clientID := "test-client"
			authInfo := &AuthInfo{UserID: "user1", Roles: []string{"user"}}

			// Should allow burst
			for i := 0; i < 5; i++ {
				Expect(limiter.Allow(clientID, authInfo)).To(BeTrue())
			}

			// Should be rate limited after burst
			Expect(limiter.Allow(clientID, authInfo)).To(BeFalse())
		})

		It("should bypass rate limits for admin role", func() {
			clientID := "admin-client"
			authInfo := &AuthInfo{UserID: "admin1", Roles: []string{"admin"}}

			// Admin should bypass rate limits
			for i := 0; i < 20; i++ {
				Expect(limiter.Allow(clientID, authInfo)).To(BeTrue())
			}
		})

		It("should reset rate limit after cleanup", func() {
			// Create a new limiter for this test to avoid global rate limit issues
			cleanupConfig := &RateLimitConfig{
				MessagesPerMinute:       60,  // 1 per second
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
			Expect(cleanupLimiter.Allow(clientID, authInfo)).To(BeFalse())

			// Remove the limiter
			cleanupLimiter.Remove(clientID)

			// Should get a fresh limiter with full burst after removal
			for i := 0; i < 5; i++ {
				Expect(cleanupLimiter.Allow(clientID, authInfo)).To(BeTrue())
			}
		})
	})

	Context("when testing connection limiter", func() {
		var (
			config  *ConnectionLimitConfig
			limiter *ConnectionLimiter
		)

		BeforeEach(func() {
			config = &ConnectionLimitConfig{
				MaxConnectionsPerUser: 2,
				MaxConnectionsPerIP:   3,
				MaxTotalConnections:   5,
				BypassRoles:           []string{"admin"},
			}
			limiter = NewConnectionLimiter(config)
		})

		It("should enforce per-user connection limit", func() {
			req1 := httptest.NewRequest("GET", "/ws", nil)
			req1.RemoteAddr = "192.168.1.1:1234"
			
			req2 := httptest.NewRequest("GET", "/ws", nil)
			req2.RemoteAddr = "192.168.1.2:1234"
			
			req3 := httptest.NewRequest("GET", "/ws", nil)
			req3.RemoteAddr = "192.168.1.3:1234"

			authInfo := &AuthInfo{UserID: "user1", Roles: []string{"user"}}

			// First two connections should be allowed
			Expect(limiter.CanConnect(req1, authInfo)).To(BeTrue())
			limiter.AddConnection(req1, authInfo)

			Expect(limiter.CanConnect(req2, authInfo)).To(BeTrue())
			limiter.AddConnection(req2, authInfo)

			// Third connection should be denied
			Expect(limiter.CanConnect(req3, authInfo)).To(BeFalse())

			// Remove one connection
			limiter.RemoveConnection(req1, authInfo)

			// Now third connection should be allowed
			Expect(limiter.CanConnect(req3, authInfo)).To(BeTrue())
		})

		It("should enforce per-IP connection limit", func() {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.RemoteAddr = "192.168.1.1:1234"

			// Different users from same IP
			for i := 0; i < 3; i++ {
				authInfo := &AuthInfo{UserID: string(rune('a' + i)), Roles: []string{"user"}}
				Expect(limiter.CanConnect(req, authInfo)).To(BeTrue())
				limiter.AddConnection(req, authInfo)
			}

			// Fourth connection from same IP should be denied
			authInfo := &AuthInfo{UserID: "user4", Roles: []string{"user"}}
			Expect(limiter.CanConnect(req, authInfo)).To(BeFalse())
		})

		It("should bypass connection limits for admin role", func() {
			req := httptest.NewRequest("GET", "/ws", nil)
			req.RemoteAddr = "192.168.1.1:1234"

			// Fill up the limits with regular users
			for i := 0; i < 5; i++ {
				authInfo := &AuthInfo{UserID: string(rune('a' + i)), Roles: []string{"user"}}
				limiter.AddConnection(req, authInfo)
			}

			// Admin should still be able to connect
			adminAuth := &AuthInfo{UserID: "admin1", Roles: []string{"admin"}}
			Expect(limiter.CanConnect(req, adminAuth)).To(BeTrue())
		})

		It("should track connection statistics", func() {
			req1 := httptest.NewRequest("GET", "/ws", nil)
			req1.RemoteAddr = "192.168.1.1:1234"
			
			req2 := httptest.NewRequest("GET", "/ws", nil)
			req2.RemoteAddr = "192.168.1.2:1234"

			authInfo1 := &AuthInfo{UserID: "user1"}
			authInfo2 := &AuthInfo{UserID: "user2"}

			limiter.AddConnection(req1, authInfo1)
			limiter.AddConnection(req2, authInfo2)

			stats := limiter.GetConnectionStats()
			Expect(stats["total_connections"]).To(Equal(2))
			Expect(stats["users_connected"]).To(Equal(2))
			Expect(stats["unique_ips"]).To(Equal(2))
		})
	})

	Context("when testing WebSocket with rate limiting", func() {
		var (
			hub    *WebSocketHub
			server *Server
			ts     *httptest.Server
			ctx    context.Context
			cancel context.CancelFunc
		)

		BeforeEach(func() {
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

			hub = NewWebSocketHubWithConfig(config)
			ctx, cancel = context.WithCancel(context.Background())
			go hub.Run(ctx)

			// Create server
			server = &Server{
				webSocketHub: hub,
			}

			// Create test server
			handler := server.HandleWebSocket(hub)
			ts = httptest.NewServer(handler)
		})

		AfterEach(func() {
			cancel()
			ts.Close()
		})

		It("should enforce rate limits on WebSocket messages", func() {
			// Connect
			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
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
				Expect(err).NotTo(HaveOccurred())
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
					Expect(err).NotTo(HaveOccurred())
					if errorPayload["error"] == "rate_limit_exceeded" {
						errorReceived = true
					}
				}
			}

			// Should receive 2 pongs (burst limit) and 1 error
			Expect(responsesReceived).To(Equal(2))
			Expect(errorReceived).To(BeTrue())
		})
	})

	Context("when testing WebSocket connection limits", func() {
		var (
			hub         *WebSocketHub
			server      *Server
			ts          *httptest.Server
			ctx         context.Context
			cancel      context.CancelFunc
			connections []*websocket.Conn
		)

		BeforeEach(func() {
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

			hub = NewWebSocketHubWithConfig(config)
			ctx, cancel = context.WithCancel(context.Background())
			go hub.Run(ctx)

			// Create server
			server = &Server{
				webSocketHub: hub,
			}

			// Create test server
			handler := server.HandleWebSocket(hub)
			ts = httptest.NewServer(handler)
			
			connections = make([]*websocket.Conn, 0)
		})

		AfterEach(func() {
			for _, conn := range connections {
				conn.Close()
			}
			cancel()
			ts.Close()
		})

		It("should enforce per-user connection limit", func() {
			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

			// Create headers for user1
			header1 := http.Header{}
			header1.Set("X-User-ID", "user1")

			// First two connections should succeed
			conn1, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
			Expect(err).NotTo(HaveOccurred())
			connections = append(connections, conn1)

			conn2, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
			Expect(err).NotTo(HaveOccurred())
			connections = append(connections, conn2)

			// Third connection should fail
			_, resp, err := websocket.DefaultDialer.Dial(wsURL, header1)
			Expect(err).To(HaveOccurred())
			if resp != nil {
				Expect(resp.StatusCode).To(Equal(http.StatusTooManyRequests))
			}
		})

		It("should enforce total connection limit", func() {
			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

			// First add connections for user1
			header1 := http.Header{}
			header1.Set("X-User-ID", "user1")

			conn1, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
			Expect(err).NotTo(HaveOccurred())
			connections = append(connections, conn1)

			conn2, _, err := websocket.DefaultDialer.Dial(wsURL, header1)
			Expect(err).NotTo(HaveOccurred())
			connections = append(connections, conn2)

			// Create connection for user2 (total would be 3 with the 2 from user1)
			header2 := http.Header{}
			header2.Set("X-User-ID", "user2")

			conn3, _, err := websocket.DefaultDialer.Dial(wsURL, header2)
			Expect(err).NotTo(HaveOccurred())
			connections = append(connections, conn3)

			// Fourth total connection should fail (we have 3 connections already)
			header3 := http.Header{}
			header3.Set("X-User-ID", "user3")
			
			_, resp, err := websocket.DefaultDialer.Dial(wsURL, header3)
			Expect(err).To(HaveOccurred())
			if resp != nil {
				Expect(resp.StatusCode).To(Equal(http.StatusTooManyRequests))
			}
		})
	})

	Context("when testing inactive connection cleanup", func() {
		It("should track inactive connections properly", func() {
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
			Expect(client.IsActive()).To(BeFalse(), "Client should be inactive after timeout")
			
			// Update activity
			client.updateLastActivity()
			Expect(client.IsActive()).To(BeTrue(), "Client should be active after update")
		})
	})

	Context("when testing getClientIP function", func() {
		DescribeTable("IP extraction scenarios",
			func(headers map[string]string, remoteAddr, expectedIP string) {
				req := httptest.NewRequest("GET", "/ws", nil)
				req.RemoteAddr = remoteAddr
				
				for k, v := range headers {
					req.Header.Set(k, v)
				}
				
				ip := getClientIP(req)
				Expect(ip).To(Equal(expectedIP))
			},
			Entry("X-Forwarded-For single IP",
				map[string]string{
					"X-Forwarded-For": "192.168.1.1",
				},
				"10.0.0.1:1234",
				"192.168.1.1",
			),
			Entry("X-Forwarded-For multiple IPs",
				map[string]string{
					"X-Forwarded-For": "192.168.1.1, 10.0.0.2, 10.0.0.3",
				},
				"10.0.0.1:1234",
				"192.168.1.1",
			),
			Entry("X-Real-IP",
				map[string]string{
					"X-Real-IP": "192.168.1.2",
				},
				"10.0.0.1:1234",
				"192.168.1.2",
			),
			Entry("RemoteAddr with port",
				map[string]string{},
				"192.168.1.3:1234",
				"192.168.1.3",
			),
			Entry("RemoteAddr without port",
				map[string]string{},
				"192.168.1.4",
				"192.168.1.4",
			),
		)
	})
})