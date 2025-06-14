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

var _ = Describe("WebSocketHandler", func() {
	var (
		hub    *WebSocketHub
		ctx    context.Context
		cancel context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
	})

	Context("when running the WebSocket hub", func() {
		It("should handle client registration and unregistration", func() {
			hub = NewWebSocketHub()
			
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
			Expect(hub.clients[client]).To(BeTrue())
			hub.mu.RUnlock()
			
			// Test broadcast
			testMsg := WebSocketMessage{
				Type:      "test",
				Timestamp: time.Now(),
			}
			hub.broadcast <- testMsg
			
			// Client should receive the message
			Eventually(func() WebSocketMessage {
				select {
				case msg := <-client.send:
					return msg
				default:
					return WebSocketMessage{}
				}
			}, 100*time.Millisecond).Should(Equal(testMsg))
			
			// Test client unregistration
			hub.unregister <- client
			time.Sleep(10 * time.Millisecond)
			
			// Check client is unregistered
			hub.mu.RLock()
			Expect(hub.clients[client]).To(BeFalse())
			hub.mu.RUnlock()
		})
	})

	Context("when checking WebSocket origin", func() {
		DescribeTable("origin validation scenarios",
			func(config *WebSocketConfig, origin string, expectUpgrade bool) {
				// Create hub with config
				hub = NewWebSocketHubWithConfig(config)
				
				// Start hub in background
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
				if origin != "" {
					header.Set("Origin", origin)
				}
				
				// Attempt connection
				conn, resp, err := dialer.Dial(wsURL, header)
				
				if expectUpgrade {
					Expect(err).NotTo(HaveOccurred())
					Expect(conn).NotTo(BeNil())
					defer conn.Close()
					
					// Connection was successful - origin check passed
					// Give time for the client to be registered
					time.Sleep(50 * time.Millisecond)
				} else {
					// Connection should be rejected
					Expect(err).To(HaveOccurred())
					if resp != nil {
						Expect(resp.StatusCode).NotTo(Equal(http.StatusSwitchingProtocols))
					}
				}
			},
			Entry("allowed origin",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://localhost:3000"},
				},
				"http://localhost:3000",
				true,
			),
			Entry("denied origin",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://localhost:3000"},
				},
				"http://malicious.com",
				false,
			),
			Entry("no origin header with allowed list",
				&WebSocketConfig{
					AllowedOrigins: []string{"http://localhost:3000"},
				},
				"",
				true, // Same origin request
			),
			Entry("empty allowed list allows all",
				&WebSocketConfig{},
				"http://any-origin.com",
				true,
			),
		)
	})

	Context("when handling client messages", func() {
		var client *WebSocketClient

		BeforeEach(func() {
			// Create a hub with auth disabled for testing
			hub = &WebSocketHub{
				config: &WebSocketConfig{
					AuthEnabled: false,
				},
			}
			
			client = &WebSocketClient{
				hub:           hub,
				send:          make(chan WebSocketMessage, 10),
				id:            "test-client",
				subscriptions: make(map[string]map[string]bool),
				filters:       []EventFilter{},
			}
		})

		It("should handle ping messages", func() {
			message := WebSocketMessage{
				Type: "ping",
				ID:   "123",
			}
			
			client.handleMessage(message)
			
			var response WebSocketMessage
			Eventually(func() WebSocketMessage {
				select {
				case msg := <-client.send:
					response = msg
					return msg
				default:
					return WebSocketMessage{}
				}
			}, 100*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}))
			
			Expect(response.Type).To(Equal("pong"))
			Expect(response.ID).To(Equal("123"))
		})

		It("should handle subscribe messages", func() {
			message := WebSocketMessage{
				Type:    "subscribe",
				Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
			}
			
			client.handleMessage(message)
			// For now, just ensure no panic
			// In real implementation, would check subscription state
		})

		It("should handle unsubscribe messages", func() {
			message := WebSocketMessage{
				Type:    "unsubscribe",
				Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
			}
			
			client.handleMessage(message)
			// For now, just ensure no panic
			// In real implementation, would check subscription state
		})

		It("should handle unknown message types gracefully", func() {
			message := WebSocketMessage{
				Type: "unknown",
			}
			
			// Should not crash
			client.handleMessage(message)
		})
	})

	Context("when broadcasting messages", func() {
		var clients []*WebSocketClient

		BeforeEach(func() {
			hub = NewWebSocketHub()
			go hub.Run(ctx)
			
			// Create and register multiple clients
			clients = make([]*WebSocketClient, 3)
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
		})

		It("should broadcast task updates to all clients", func() {
			hub.BroadcastTaskUpdate("task-123", map[string]interface{}{
				"status": "RUNNING",
				"cpu":    "50%",
			})
			
			// All clients should receive the update
			for i, client := range clients {
				var msg WebSocketMessage
				Eventually(func() WebSocketMessage {
					select {
					case m := <-client.send:
						msg = m
						return m
					default:
						return WebSocketMessage{}
					}
				}, 100*time.Millisecond, 10*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}), "Client %d did not receive task update", i)
				
				Expect(msg.Type).To(Equal("task_update"))
				Expect(msg.Payload).NotTo(BeNil())
			}
		})

		It("should broadcast log entries to all clients", func() {
			hub.BroadcastLogEntry(map[string]interface{}{
				"message":   "Test log",
				"level":     "info",
				"timestamp": time.Now(),
			})
			
			// All clients should receive the log
			for i, client := range clients {
				var msg WebSocketMessage
				Eventually(func() WebSocketMessage {
					select {
					case m := <-client.send:
						msg = m
						return m
					default:
						return WebSocketMessage{}
					}
				}, 100*time.Millisecond, 10*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}), "Client %d did not receive log entry", i)
				
				Expect(msg.Type).To(Equal("log_entry"))
				Expect(msg.Payload).NotTo(BeNil())
			}
		})

		It("should broadcast metric updates to all clients", func() {
			hub.BroadcastMetricUpdate(map[string]interface{}{
				"cpu_usage":    75.5,
				"memory_usage": 1024,
			})
			
			// All clients should receive the metrics
			for i, client := range clients {
				var msg WebSocketMessage
				Eventually(func() WebSocketMessage {
					select {
					case m := <-client.send:
						msg = m
						return m
					default:
						return WebSocketMessage{}
					}
				}, 100*time.Millisecond, 10*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}), "Client %d did not receive metric update", i)
				
				Expect(msg.Type).To(Equal("metric_update"))
				Expect(msg.Payload).NotTo(BeNil())
			}
		})
	})

	Context("when testing event filtering", func() {
		var client *WebSocketClient

		BeforeEach(func() {
			// Create a hub with auth disabled for testing
			hub = &WebSocketHub{
				config: &WebSocketConfig{
					AuthEnabled: false,
				},
			}
			
			client = &WebSocketClient{
				hub:     hub,
				id:      "test-client",
				filters: []EventFilter{},
			}
		})

		It("should accept all messages with no filters", func() {
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "task",
				ResourceID:   "task-123",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue())
		})

		It("should filter by event type", func() {
			client.SetFilters([]EventFilter{
				{EventTypes: []string{"task_update", "service_update"}},
			})
			
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "task",
				ResourceID:   "task-123",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue())
			
			msg.Type = "log_entry"
			Expect(client.MatchesFilter(msg)).To(BeFalse())
		})

		It("should filter by resource type", func() {
			client.SetFilters([]EventFilter{
				{ResourceTypes: []string{"task", "service"}},
			})
			
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "task",
				ResourceID:   "task-123",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue())
			
			msg.ResourceType = "cluster"
			Expect(client.MatchesFilter(msg)).To(BeFalse())
		})

		It("should filter by resource ID", func() {
			client.SetFilters([]EventFilter{
				{ResourceIDs: []string{"task-123", "task-456"}},
			})
			
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "task",
				ResourceID:   "task-123",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue())
			
			msg.ResourceID = "task-789"
			Expect(client.MatchesFilter(msg)).To(BeFalse())
		})

		It("should support wildcard resource ID", func() {
			client.SetFilters([]EventFilter{
				{ResourceIDs: []string{"*"}},
			})
			
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "task",
				ResourceID:   "any-task-id",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue())
		})

		It("should apply AND logic within a filter", func() {
			client.SetFilters([]EventFilter{
				{
					EventTypes:    []string{"task_update"},
					ResourceTypes: []string{"task"},
					ResourceIDs:   []string{"task-123"},
				},
			})
			
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "task",
				ResourceID:   "task-123",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue())
			
			msg.ResourceID = "task-456"
			Expect(client.MatchesFilter(msg)).To(BeFalse())
		})

		It("should apply OR logic between filters", func() {
			client.SetFilters([]EventFilter{
				{EventTypes: []string{"task_update"}},
				{ResourceTypes: []string{"service"}},
			})
			
			msg := WebSocketMessage{
				Type:         "task_update",
				ResourceType: "cluster",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue()) // Matches first filter
			
			msg = WebSocketMessage{
				Type:         "cluster_update",
				ResourceType: "service",
			}
			Expect(client.MatchesFilter(msg)).To(BeTrue()) // Matches second filter
			
			msg = WebSocketMessage{
				Type:         "cluster_update",
				ResourceType: "cluster",
			}
			Expect(client.MatchesFilter(msg)).To(BeFalse()) // Matches neither filter
		})
	})

	Context("when handling setFilters message", func() {
		var client *WebSocketClient

		BeforeEach(func() {
			// Create a hub with auth disabled for testing
			hub = &WebSocketHub{
				config: &WebSocketConfig{
					AuthEnabled: false,
				},
			}
			
			client = &WebSocketClient{
				hub:     hub,
				send:    make(chan WebSocketMessage, 10),
				id:      "test-client",
				filters: []EventFilter{},
			}
		})

		It("should set filters and send confirmation", func() {
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
			Expect(client.filters).To(HaveLen(1))
			Expect(client.filters[0].EventTypes).To(Equal(filters[0].EventTypes))
			Expect(client.filters[0].ResourceTypes).To(Equal(filters[0].ResourceTypes))
			
			// Check confirmation was sent
			var msg WebSocketMessage
			Eventually(func() WebSocketMessage {
				select {
				case m := <-client.send:
					msg = m
					return m
				default:
					return WebSocketMessage{}
				}
			}, 100*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}))
			
			Expect(msg.Type).To(Equal("filtersSet"))
			Expect(msg.ID).To(Equal("msg-789"))
			Expect(msg.Payload).To(Equal(setFiltersMsg.Payload))
		})
	})

	Context("when broadcasting with filtering", func() {
		var (
			clientWithTaskFilter    *WebSocketClient
			clientWithServiceFilter *WebSocketClient
			clientWithNoFilter      *WebSocketClient
		)

		BeforeEach(func() {
			hub = NewWebSocketHub()
			go hub.Run(ctx)

			// Create clients with different filters
			clientWithTaskFilter = &WebSocketClient{
				hub:           hub,
				send:          make(chan WebSocketMessage, 10),
				id:            "task-filter-client",
				subscriptions: make(map[string]map[string]bool),
				filters: []EventFilter{
					{EventTypes: []string{"task_update"}},
				},
			}
			
			clientWithServiceFilter = &WebSocketClient{
				hub:           hub,
				send:          make(chan WebSocketMessage, 10),
				id:            "service-filter-client",
				subscriptions: make(map[string]map[string]bool),
				filters: []EventFilter{
					{EventTypes: []string{"service_update"}},
				},
			}
			
			clientWithNoFilter = &WebSocketClient{
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
		})

		It("should respect client filters when broadcasting", func() {
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
			Eventually(func() bool {
				select {
				case msg := <-clientWithTaskFilter.send:
					return msg.Type == "task_update"
				default:
					return false
				}
			}, 100*time.Millisecond, 10*time.Millisecond).Should(BeTrue())

			// Client with service filter should not receive it
			Consistently(func() bool {
				select {
				case <-clientWithServiceFilter.send:
					return true
				default:
					return false
				}
			}, 50*time.Millisecond).Should(BeFalse())

			// Client with no filter should receive it
			Eventually(func() bool {
				select {
				case msg := <-clientWithNoFilter.send:
					return msg.Type == "task_update"
				default:
					return false
				}
			}, 100*time.Millisecond, 10*time.Millisecond).Should(BeTrue())
		})
	})

	Context("when testing subscriptions", func() {
		var (
			subscribedClient   *WebSocketClient
			unsubscribedClient *WebSocketClient
		)

		BeforeEach(func() {
			hub = NewWebSocketHub()
			go hub.Run(ctx)

			// Create and register clients
			subscribedClient = &WebSocketClient{
				hub:           hub,
				send:          make(chan WebSocketMessage, 10),
				id:            "subscribed-client",
				subscriptions: make(map[string]map[string]bool),
				filters:       []EventFilter{},
			}
			
			unsubscribedClient = &WebSocketClient{
				hub:           hub,
				send:          make(chan WebSocketMessage, 10),
				id:            "unsubscribed-client",
				subscriptions: make(map[string]map[string]bool),
				filters:       []EventFilter{},
			}

			hub.register <- subscribedClient
			hub.register <- unsubscribedClient
			time.Sleep(10 * time.Millisecond)
		})

		It("should broadcast only to subscribed clients", func() {
			// Subscribe one client to task-123
			subscribedClient.Subscribe("task", "task-123")

			// Test targeted broadcast
			hub.BroadcastTaskUpdateToSubscribed("task-123", map[string]interface{}{
				"status": "RUNNING",
				"cpu":    "50%",
			})

			// Only subscribed client should receive the update
			Eventually(func() bool {
				select {
				case msg := <-subscribedClient.send:
					return msg.Type == "task_update" && msg.Payload != nil
				default:
					return false
				}
			}, 100*time.Millisecond, 10*time.Millisecond).Should(BeTrue())

			// Unsubscribed client should not receive anything
			Consistently(func() bool {
				select {
				case <-unsubscribedClient.send:
					return true
				default:
					return false
				}
			}, 50*time.Millisecond).Should(BeFalse())
		})
	})

	Context("when managing client subscriptions", func() {
		var client *WebSocketClient

		BeforeEach(func() {
			// Create a hub with auth disabled for testing
			hub = &WebSocketHub{
				config: &WebSocketConfig{
					AuthEnabled: false,
				},
			}
			
			client = &WebSocketClient{
				hub:           hub,
				id:            "test-client",
				subscriptions: make(map[string]map[string]bool),
			}
		})

		It("should handle regular subscriptions", func() {
			client.Subscribe("task", "task-123")
			Expect(client.IsSubscribed("task", "task-123")).To(BeTrue())
			Expect(client.IsSubscribed("task", "task-456")).To(BeFalse())
			Expect(client.IsSubscribed("service", "service-123")).To(BeFalse())
		})

		It("should handle wildcard subscriptions", func() {
			client.Subscribe("service", "*")
			Expect(client.IsSubscribed("service", "service-123")).To(BeTrue())
			Expect(client.IsSubscribed("service", "any-service")).To(BeTrue())
		})

		It("should handle unsubscription", func() {
			client.Subscribe("task", "task-123")
			Expect(client.IsSubscribed("task", "task-123")).To(BeTrue())
			
			client.Unsubscribe("task", "task-123")
			Expect(client.IsSubscribed("task", "task-123")).To(BeFalse())
		})

		It("should handle multiple subscriptions", func() {
			client.Subscribe("cluster", "cluster-1")
			client.Subscribe("cluster", "cluster-2")
			
			client.Unsubscribe("cluster", "cluster-1")
			Expect(client.IsSubscribed("cluster", "cluster-1")).To(BeFalse())
			Expect(client.IsSubscribed("cluster", "cluster-2")).To(BeTrue())
			
			client.Unsubscribe("cluster", "cluster-2")
			Expect(client.IsSubscribed("cluster", "cluster-2")).To(BeFalse())
		})
	})

	Context("when handling subscribe/unsubscribe messages", func() {
		var client *WebSocketClient

		BeforeEach(func() {
			// Create a hub with auth disabled for testing
			hub = &WebSocketHub{
				config: &WebSocketConfig{
					AuthEnabled: false,
				},
			}
			
			client = &WebSocketClient{
				hub:           hub,
				send:          make(chan WebSocketMessage, 10),
				id:            "test-client",
				subscriptions: make(map[string]map[string]bool),
				filters:       []EventFilter{},
			}
		})

		It("should handle subscribe message", func() {
			subscribeMsg := WebSocketMessage{
				Type:    "subscribe",
				ID:      "msg-123",
				Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
			}
			
			client.handleMessage(subscribeMsg)
			
			// Check subscription was added
			Expect(client.IsSubscribed("task", "task-123")).To(BeTrue())
			
			// Check confirmation was sent
			var msg WebSocketMessage
			Eventually(func() WebSocketMessage {
				select {
				case m := <-client.send:
					msg = m
					return m
				default:
					return WebSocketMessage{}
				}
			}, 100*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}))
			
			Expect(msg.Type).To(Equal("subscribed"))
			Expect(msg.ID).To(Equal("msg-123"))
			Expect(msg.Payload).To(Equal(subscribeMsg.Payload))
		})

		It("should handle unsubscribe message", func() {
			// First subscribe
			client.Subscribe("task", "task-123")
			
			unsubscribeMsg := WebSocketMessage{
				Type:    "unsubscribe",
				ID:      "msg-456",
				Payload: []byte(`{"resourceType":"task","resourceId":"task-123"}`),
			}
			
			client.handleMessage(unsubscribeMsg)
			
			// Check subscription was removed
			Expect(client.IsSubscribed("task", "task-123")).To(BeFalse())
			
			// Check confirmation was sent
			var msg WebSocketMessage
			Eventually(func() WebSocketMessage {
				select {
				case m := <-client.send:
					msg = m
					return m
				default:
					return WebSocketMessage{}
				}
			}, 100*time.Millisecond).ShouldNot(Equal(WebSocketMessage{}))
			
			Expect(msg.Type).To(Equal("unsubscribed"))
			Expect(msg.ID).To(Equal("msg-456"))
			Expect(msg.Payload).To(Equal(unsubscribeMsg.Payload))
		})
	})
})