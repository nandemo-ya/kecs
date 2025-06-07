package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type         string          `json:"type"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	ID           string          `json:"id,omitempty"`
	Timestamp    time.Time       `json:"timestamp,omitempty"`
	ResourceType string          `json:"resourceType,omitempty"`
	ResourceID   string          `json:"resourceId,omitempty"`
}

// EventFilter represents a filter for WebSocket events
type EventFilter struct {
	EventTypes    []string          `json:"eventTypes,omitempty"`    // Empty means all event types
	ResourceTypes []string          `json:"resourceTypes,omitempty"` // Empty means all resource types
	ResourceIDs   []string          `json:"resourceIds,omitempty"`   // Empty means all resource IDs
	Metadata      map[string]string `json:"metadata,omitempty"`      // Additional filter criteria
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan WebSocketMessage
	mu         sync.RWMutex
	config     *WebSocketConfig
}

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	hub           *WebSocketHub
	conn          *websocket.Conn
	send          chan WebSocketMessage
	id            string
	ctx           context.Context
	cancel        context.CancelFunc
	subscriptions map[string]map[string]bool // resourceType -> resourceID -> subscribed
	filters       []EventFilter              // Event filters for this client
	mu            sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub with default configuration
func NewWebSocketHub() *WebSocketHub {
	return NewWebSocketHubWithConfig(DefaultWebSocketConfig())
}

// NewWebSocketHubWithConfig creates a new WebSocket hub with custom configuration
func NewWebSocketHubWithConfig(config *WebSocketConfig) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan WebSocketMessage),
		config:     config,
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("WebSocket client registered: %s", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.mu.Unlock()
				log.Printf("WebSocket client unregistered: %s", client.id)
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.BroadcastWithFiltering(message)
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(hub *WebSocketHub) http.HandlerFunc {
	// Create upgrader with the hub's configuration
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     hub.config.CheckOrigin,
	}
	
	return func(w http.ResponseWriter, r *http.Request) {
		// Log origin for debugging
		origin := r.Header.Get("Origin")
		if origin != "" {
			log.Printf("WebSocket connection attempt from origin: %s", origin)
		}
		
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error from origin %s: %v", origin, err)
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		client := &WebSocketClient{
			hub:           hub,
			conn:          conn,
			send:          make(chan WebSocketMessage, 256),
			id:            fmt.Sprintf("%d", time.Now().UnixNano()),
			ctx:           ctx,
			cancel:        cancel,
			subscriptions: make(map[string]map[string]bool),
			filters:       []EventFilter{},
		}

		client.hub.register <- client

		// Start goroutines for reading and writing
		go client.writePump()
		go client.readPump()
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.cancel()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message WebSocketMessage
		err := c.conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *WebSocketClient) handleMessage(message WebSocketMessage) {
	// Add timestamp if not present
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}

	switch message.Type {
	case "ping":
		// Respond with pong
		c.send <- WebSocketMessage{
			Type:      "pong",
			ID:        message.ID,
			Timestamp: time.Now(),
		}

	case "subscribe":
		// Handle subscription requests
		var payload struct {
			ResourceType string `json:"resourceType"`
			ResourceID   string `json:"resourceId"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err == nil {
			c.Subscribe(payload.ResourceType, payload.ResourceID)
			// Send confirmation
			c.send <- WebSocketMessage{
				Type:      "subscribed",
				ID:        message.ID,
				Timestamp: time.Now(),
				Payload:   message.Payload,
			}
		}

	case "unsubscribe":
		// Handle unsubscription requests
		var payload struct {
			ResourceType string `json:"resourceType"`
			ResourceID   string `json:"resourceId"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err == nil {
			c.Unsubscribe(payload.ResourceType, payload.ResourceID)
			// Send confirmation
			c.send <- WebSocketMessage{
				Type:      "unsubscribed",
				ID:        message.ID,
				Timestamp: time.Now(),
				Payload:   message.Payload,
			}
		}

	case "setFilters":
		// Handle filter configuration
		var filters []EventFilter
		if err := json.Unmarshal(message.Payload, &filters); err == nil {
			c.SetFilters(filters)
			// Send confirmation
			c.send <- WebSocketMessage{
				Type:      "filtersSet",
				ID:        message.ID,
				Timestamp: time.Now(),
				Payload:   message.Payload,
			}
		}

	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}

// BroadcastTaskUpdate sends task update to all connected clients
func (h *WebSocketHub) BroadcastTaskUpdate(taskID string, update interface{}) {
	payload, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling task update: %v", err)
		return
	}

	message := WebSocketMessage{
		Type:         "task_update",
		Payload:      payload,
		Timestamp:    time.Now(),
		ResourceType: "task",
		ResourceID:   taskID,
	}

	h.BroadcastWithFiltering(message)
}

// BroadcastLogEntry sends log entry to all connected clients
func (h *WebSocketHub) BroadcastLogEntry(entry interface{}) {
	payload, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}

	message := WebSocketMessage{
		Type:      "log_entry",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	h.broadcast <- message
}

// BroadcastMetricUpdate sends metric update to all connected clients
func (h *WebSocketHub) BroadcastMetricUpdate(metrics interface{}) {
	payload, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("Error marshaling metric update: %v", err)
		return
	}

	message := WebSocketMessage{
		Type:      "metric_update",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	h.broadcast <- message
}

// Subscribe adds a subscription for the client
func (c *WebSocketClient) Subscribe(resourceType, resourceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.subscriptions[resourceType] == nil {
		c.subscriptions[resourceType] = make(map[string]bool)
	}
	c.subscriptions[resourceType][resourceID] = true
	
	log.Printf("Client %s subscribed to %s:%s", c.id, resourceType, resourceID)
}

// Unsubscribe removes a subscription for the client
func (c *WebSocketClient) Unsubscribe(resourceType, resourceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.subscriptions[resourceType] != nil {
		delete(c.subscriptions[resourceType], resourceID)
		if len(c.subscriptions[resourceType]) == 0 {
			delete(c.subscriptions, resourceType)
		}
	}
	
	log.Printf("Client %s unsubscribed from %s:%s", c.id, resourceType, resourceID)
}

// IsSubscribed checks if a client is subscribed to a resource
func (c *WebSocketClient) IsSubscribed(resourceType, resourceID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if resources, ok := c.subscriptions[resourceType]; ok {
		// Check specific resource or wildcard subscription
		return resources[resourceID] || resources["*"]
	}
	return false
}

// BroadcastToSubscribed sends a message to clients subscribed to a specific resource
func (h *WebSocketHub) BroadcastToSubscribed(resourceType, resourceID string, message WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	for client := range h.clients {
		if client.IsSubscribed(resourceType, resourceID) {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, skip
				log.Printf("Client %s send channel full, skipping message", client.id)
			}
		}
	}
}

// BroadcastTaskUpdateToSubscribed sends task update to subscribed clients only
func (h *WebSocketHub) BroadcastTaskUpdateToSubscribed(taskID string, update interface{}) {
	payload, err := json.Marshal(update)
	if err != nil {
		log.Printf("Error marshaling task update: %v", err)
		return
	}

	message := WebSocketMessage{
		Type:      "task_update",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	h.BroadcastToSubscribed("task", taskID, message)
}

// SetFilters sets the event filters for the client
func (c *WebSocketClient) SetFilters(filters []EventFilter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.filters = filters
	log.Printf("Client %s updated filters: %d filters set", c.id, len(filters))
}

// MatchesFilter checks if a message matches the client's filters
func (c *WebSocketClient) MatchesFilter(message WebSocketMessage) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// If no filters are set, accept all messages
	if len(c.filters) == 0 {
		return true
	}
	
	// Check each filter - if any filter matches, accept the message
	for _, filter := range c.filters {
		if c.matchesSingleFilter(message, filter) {
			return true
		}
	}
	
	return false
}

// matchesSingleFilter checks if a message matches a single filter
func (c *WebSocketClient) matchesSingleFilter(message WebSocketMessage, filter EventFilter) bool {
	// Check event type
	if len(filter.EventTypes) > 0 {
		found := false
		for _, eventType := range filter.EventTypes {
			if message.Type == eventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check resource type
	if len(filter.ResourceTypes) > 0 {
		found := false
		for _, resourceType := range filter.ResourceTypes {
			if message.ResourceType == resourceType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check resource ID
	if len(filter.ResourceIDs) > 0 {
		found := false
		for _, resourceID := range filter.ResourceIDs {
			if message.ResourceID == resourceID || resourceID == "*" {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check metadata if present
	if len(filter.Metadata) > 0 {
		// For now, we'll need to parse the payload to check metadata
		// This is a simplified implementation
		// In a real implementation, you might want to add metadata fields to WebSocketMessage
		return true // Simplified for now
	}
	
	return true
}

// BroadcastWithFiltering sends a message to all clients that match the filter criteria
func (h *WebSocketHub) BroadcastWithFiltering(message WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	for client := range h.clients {
		// Check if the message passes the filter
		if client.MatchesFilter(message) {
			select {
			case client.send <- message:
			default:
				// Client's send channel is full, skip
				log.Printf("Client %s send channel full, skipping message", client.id)
			}
		}
	}
}