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
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	ID        string          `json:"id,omitempty"`
	Timestamp time.Time       `json:"timestamp,omitempty"`
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan WebSocketMessage
	mu         sync.RWMutex
}

// WebSocketClient represents a WebSocket client connection
type WebSocketClient struct {
	hub    *WebSocketHub
	conn   *websocket.Conn
	send   chan WebSocketMessage
	id     string
	ctx    context.Context
	cancel context.CancelFunc
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking
		return true
	},
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan WebSocketMessage),
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
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(hub *WebSocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		ctx, cancel := context.WithCancel(r.Context())
		client := &WebSocketClient{
			hub:    hub,
			conn:   conn,
			send:   make(chan WebSocketMessage, 256),
			id:     fmt.Sprintf("%d", time.Now().UnixNano()),
			ctx:    ctx,
			cancel: cancel,
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
			// TODO: Implement subscription logic
			log.Printf("Client %s subscribed to %s:%s", c.id, payload.ResourceType, payload.ResourceID)
		}

	case "unsubscribe":
		// Handle unsubscription requests
		var payload struct {
			ResourceType string `json:"resourceType"`
			ResourceID   string `json:"resourceId"`
		}
		if err := json.Unmarshal(message.Payload, &payload); err == nil {
			// TODO: Implement unsubscription logic
			log.Printf("Client %s unsubscribed from %s:%s", c.id, payload.ResourceType, payload.ResourceID)
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
		Type:      "task_update",
		Payload:   payload,
		Timestamp: time.Now(),
	}

	h.broadcast <- message
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