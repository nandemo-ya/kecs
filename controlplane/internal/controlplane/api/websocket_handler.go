package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	clients           map[*WebSocketClient]bool
	register          chan *WebSocketClient
	unregister        chan *WebSocketClient
	broadcast         chan WebSocketMessage
	mu                sync.RWMutex
	config            *WebSocketConfig
	rateLimiter       *RateLimiter
	connectionLimiter *ConnectionLimiter
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
	authInfo      *AuthInfo                  // Authentication information
	request       *http.Request              // Original HTTP request for IP tracking
	lastActivity  time.Time                  // Last activity timestamp
	mu            sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub with default configuration
func NewWebSocketHub() *WebSocketHub {
	return NewWebSocketHubWithConfig(DefaultWebSocketConfig())
}

// NewWebSocketHubWithConfig creates a new WebSocket hub with custom configuration
func NewWebSocketHubWithConfig(config *WebSocketConfig) *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan WebSocketMessage),
		config:     config,
	}

	// Initialize rate limiter if configured
	if config.RateLimitConfig != nil {
		hub.rateLimiter = NewRateLimiter(config.RateLimitConfig)
	}

	// Initialize connection limiter if configured
	if config.ConnectionLimitConfig != nil {
		hub.connectionLimiter = NewConnectionLimiter(config.ConnectionLimitConfig)
	}

	return hub
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run(ctx context.Context) {
	// Start connection cleanup if configured
	if h.config.ConnectionLimitConfig != nil && h.config.ConnectionLimitConfig.ConnectionTimeout > 0 {
		go h.cleanupInactiveConnections(ctx)
	}

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

				// Clean up rate limiter
				if h.rateLimiter != nil {
					h.rateLimiter.Remove(client.id)
				}

				// Remove from connection limiter
				if h.connectionLimiter != nil && client.request != nil {
					h.connectionLimiter.RemoveConnection(client.request, client.authInfo)
				}

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

		// Authenticate if enabled
		var authInfo *AuthInfo
		if hub.config.AuthEnabled {
			authenticated := false
			if hub.config.AuthFunc != nil {
				authInfo, authenticated = hub.config.AuthFunc(r)
			} else {
				// Default authentication using Authorization header
				authInfo, authenticated = defaultAuthFunc(r)
			}

			if !authenticated {
				log.Printf("WebSocket authentication failed from origin %s", origin)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			log.Printf("WebSocket authenticated user: %s", authInfo.Username)
		}

		// Check connection limits
		if hub.connectionLimiter != nil && !hub.connectionLimiter.CanConnect(r, authInfo) {
			username := "anonymous"
			if authInfo != nil {
				username = authInfo.Username
			}
			log.Printf("WebSocket connection limit exceeded for user %s from %s",
				username, getClientIP(r))
			http.Error(w, "Connection limit exceeded", http.StatusTooManyRequests)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error from origin %s: %v", origin, err)
			return
		}

		// Add to connection limiter after successful upgrade
		if hub.connectionLimiter != nil && r != nil {
			hub.connectionLimiter.AddConnection(r, authInfo)
		}

		ctx, cancel := context.WithCancel(context.Background())
		client := &WebSocketClient{
			hub:           hub,
			conn:          conn,
			send:          make(chan WebSocketMessage, 256),
			id:            fmt.Sprintf("%d", time.Now().UnixNano()),
			ctx:           ctx,
			cancel:        cancel,
			subscriptions: make(map[string]map[string]bool),
			filters:       []EventFilter{},
			authInfo:      authInfo,
			request:       r,
			lastActivity:  time.Now(),
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
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection in readPump for client %s: %v", c.id, err)
		}
	}()

	// Set initial read deadline
	readTimeout := c.hub.config.PongTimeout
	if readTimeout == 0 {
		readTimeout = 60 * time.Second
	}
	c.conn.SetReadDeadline(time.Now().Add(readTimeout))

	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		c.updateLastActivity()
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

		// Check rate limit
		if c.hub.rateLimiter != nil && !c.hub.rateLimiter.Allow(c.id, c.authInfo) {
			// Send rate limit error
			c.send <- WebSocketMessage{
				Type:      "error",
				Timestamp: time.Now(),
				Payload:   []byte(`{"error":"rate_limit_exceeded","message":"Too many requests"}`),
			}
			continue
		}

		// Update last activity
		c.updateLastActivity()

		// Handle different message types
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
	// Use configured ping interval
	pingInterval := c.hub.config.PingInterval
	if pingInterval == 0 {
		pingInterval = 54 * time.Second
	}
	ticker := time.NewTicker(pingInterval)

	// Use configured write timeout
	writeTimeout := c.hub.config.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 10 * time.Second
	}

	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection in writePump for client %s: %v", c.id, err)
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
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
			// Check authorization for subscription
			resource := fmt.Sprintf("%s/%s", payload.ResourceType, payload.ResourceID)
			if !c.IsAuthorized("subscribe", resource) {
				// Send error response
				c.send <- WebSocketMessage{
					Type:      "error",
					ID:        message.ID,
					Timestamp: time.Now(),
					Payload:   []byte(`{"error":"unauthorized","message":"Not authorized to subscribe to this resource"}`),
				}
				return
			}

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
			// Check authorization for unsubscription
			resource := fmt.Sprintf("%s/%s", payload.ResourceType, payload.ResourceID)
			if !c.IsAuthorized("unsubscribe", resource) {
				// Send error response
				c.send <- WebSocketMessage{
					Type:      "error",
					ID:        message.ID,
					Timestamp: time.Now(),
					Payload:   []byte(`{"error":"unauthorized","message":"Not authorized to unsubscribe from this resource"}`),
				}
				return
			}

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
			// Check authorization for setting filters
			if !c.IsAuthorized("setFilters", "websocket") {
				// Send error response
				c.send <- WebSocketMessage{
					Type:      "error",
					ID:        message.ID,
					Timestamp: time.Now(),
					Payload:   []byte(`{"error":"unauthorized","message":"Not authorized to set filters"}`),
				}
				return
			}

			c.SetFilters(filters)
			// Send confirmation
			c.send <- WebSocketMessage{
				Type:      "filtersSet",
				ID:        message.ID,
				Timestamp: time.Now(),
				Payload:   message.Payload,
			}
		}

	case "authenticate":
		// Handle authentication/token refresh
		var payload struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err == nil {
			// Validate the token and update auth info
			if c.hub.config.AuthEnabled && c.hub.config.AuthFunc != nil {
				// Create a mock request with the token
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					log.Printf("Failed to create mock request for WebSocket authentication: %v", err)
					// Send error response
					c.send <- WebSocketMessage{
						Type:      "error",
						ID:        message.ID,
						Timestamp: time.Now(),
						Payload:   []byte(`{"error":"Authentication failed due to internal error"}`),
					}
					return
				}
				req.Header.Set("Authorization", "Bearer "+payload.Token)

				if authInfo, authenticated := c.hub.config.AuthFunc(req); authenticated {
					c.mu.Lock()
					c.authInfo = authInfo
					c.mu.Unlock()

					// Send success response
					c.send <- WebSocketMessage{
						Type:      "authenticated",
						ID:        message.ID,
						Timestamp: time.Now(),
						Payload:   []byte(fmt.Sprintf(`{"username":"%s","roles":%s}`, authInfo.Username, toJSON(authInfo.Roles))),
					}
				} else {
					// Send error response
					c.send <- WebSocketMessage{
						Type:      "error",
						ID:        message.ID,
						Timestamp: time.Now(),
						Payload:   []byte(`{"error":"authentication_failed","message":"Invalid token"}`),
					}
				}
			} else {
				// Auth not enabled, send success
				c.send <- WebSocketMessage{
					Type:      "authenticated",
					ID:        message.ID,
					Timestamp: time.Now(),
					Payload:   []byte(`{"message":"Authentication not required"}`),
				}
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

	if c.subscriptions == nil {
		c.subscriptions = make(map[string]map[string]bool)
	}
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

// defaultAuthFunc provides basic authentication using Authorization header
func defaultAuthFunc(r *http.Request) (*AuthInfo, bool) {
	// Check for Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, false
	}

	// Extract token (Bearer token format)
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return nil, false
	}

	token := strings.TrimPrefix(authHeader, bearerPrefix)

	// In a real implementation, validate the token
	// For now, we'll create a simple auth info based on the token
	// TODO: Implement proper token validation (JWT, API key, etc.)

	// Mock implementation - DO NOT USE IN PRODUCTION
	if token == "" {
		return nil, false
	}

	return &AuthInfo{
		UserID:   "user-" + token[:8], // Use first 8 chars as user ID
		Username: "user-" + token[:8],
		Roles:    []string{"user"},
		Permissions: []string{
			"websocket:connect",
			"task:read",
			"service:read",
		},
		Metadata: map[string]interface{}{
			"token": token,
		},
	}, true
}

// IsAuthorized checks if the client is authorized for an operation
func (c *WebSocketClient) IsAuthorized(operation string, resource string) bool {
	// If auth is not enabled, allow all operations
	if c.hub.config.AuthEnabled == false {
		return true
	}

	if c.authInfo == nil {
		return false
	}

	// Use custom authorize function if provided
	if c.hub.config.AuthorizeFunc != nil {
		return c.hub.config.AuthorizeFunc(c.authInfo, operation, resource)
	}

	// Default authorization logic
	return c.defaultAuthorize(operation, resource)
}

// defaultAuthorize provides basic authorization logic
func (c *WebSocketClient) defaultAuthorize(operation string, resource string) bool {
	// Check permissions
	requiredPermission := fmt.Sprintf("%s:%s", getResourceType(resource), operation)

	for _, perm := range c.authInfo.Permissions {
		if perm == requiredPermission || perm == "*:*" {
			return true
		}

		// Check wildcard permissions
		parts := strings.Split(perm, ":")
		if len(parts) == 2 {
			if (parts[0] == "*" || parts[0] == getResourceType(resource)) &&
				(parts[1] == "*" || parts[1] == operation) {
				return true
			}
		}
	}

	// Check role-based permissions
	for _, role := range c.authInfo.Roles {
		if role == "admin" {
			return true // Admins can do everything
		}
	}

	return false
}

// getResourceType extracts resource type from resource string
func getResourceType(resource string) string {
	// Simple implementation - extract first part before slash
	parts := strings.Split(resource, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return resource
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

// toJSON is a helper function to convert data to JSON string
func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// updateLastActivity updates the client's last activity timestamp
func (c *WebSocketClient) updateLastActivity() {
	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()
}

// GetLastActivity returns the client's last activity timestamp
func (c *WebSocketClient) GetLastActivity() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActivity
}

// IsActive checks if the client is still active based on connection timeout
func (c *WebSocketClient) IsActive() bool {
	if c.hub.config.ConnectionLimitConfig == nil {
		return true // No connection limit configured
	}

	timeout := c.hub.config.ConnectionLimitConfig.ConnectionTimeout
	if timeout == 0 {
		return true // No timeout configured
	}

	return time.Since(c.GetLastActivity()) < timeout
}

// ConnectionStats returns statistics about current connections
func (h *WebSocketHub) ConnectionStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := map[string]interface{}{
		"total_clients": len(h.clients),
	}

	if h.connectionLimiter != nil {
		connectionStats := h.connectionLimiter.GetConnectionStats()
		for k, v := range connectionStats {
			stats[k] = v
		}
	}

	return stats
}

// cleanupInactiveConnections periodically checks for and removes inactive connections
func (h *WebSocketHub) cleanupInactiveConnections(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			inactiveClients := make([]*WebSocketClient, 0)

			for client := range h.clients {
				if !client.IsActive() {
					inactiveClients = append(inactiveClients, client)
				}
			}
			h.mu.RUnlock()

			// Close inactive connections
			for _, client := range inactiveClients {
				log.Printf("Closing inactive connection: %s", client.id)
				if err := client.conn.Close(); err != nil {
					log.Printf("Error closing inactive WebSocket connection %s: %v", client.id, err)
				}
			}
		}
	}
}
