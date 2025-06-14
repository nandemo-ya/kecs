package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSocketConfig holds WebSocket configuration
type WebSocketConfig struct {
	// AllowedOrigins is a list of allowed origins for WebSocket connections
	// If empty, all origins are allowed (not recommended for production)
	AllowedOrigins []string

	// AllowCredentials indicates whether credentials are allowed
	AllowCredentials bool

	// CheckOriginFunc is a custom function to check origin
	// If nil, the default origin check based on AllowedOrigins is used
	CheckOriginFunc func(r *http.Request) bool

	// AuthEnabled enables authentication for WebSocket connections
	AuthEnabled bool

	// AuthFunc is a custom function to authenticate WebSocket connections
	// Returns user info and whether authentication succeeded
	AuthFunc func(r *http.Request) (*AuthInfo, bool)

	// AuthorizeFunc is a custom function to authorize operations
	// Returns whether the user is authorized for the operation
	AuthorizeFunc func(authInfo *AuthInfo, operation string, resource string) bool

	// RateLimitConfig holds rate limiting configuration
	RateLimitConfig *RateLimitConfig

	// ConnectionLimitConfig holds connection limit configuration
	ConnectionLimitConfig *ConnectionLimitConfig

	// ReadTimeout is the maximum time allowed to read a message
	ReadTimeout time.Duration

	// WriteTimeout is the maximum time allowed to write a message
	WriteTimeout time.Duration

	// PingInterval is the interval for sending ping messages
	PingInterval time.Duration

	// PongTimeout is the maximum time to wait for a pong response
	PongTimeout time.Duration
}

// AuthInfo represents authenticated user information
type AuthInfo struct {
	UserID      string
	Username    string
	Roles       []string
	Permissions []string
	Metadata    map[string]interface{}
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// MessagesPerMinute is the maximum number of messages per minute per connection
	MessagesPerMinute int

	// BurstSize is the maximum burst size for rate limiting
	BurstSize int

	// GlobalMessagesPerMinute is the global rate limit across all connections
	GlobalMessagesPerMinute int

	// GlobalBurstSize is the global burst size
	GlobalBurstSize int

	// BypassRoles are roles that bypass rate limiting
	BypassRoles []string
}

// ConnectionLimitConfig holds connection limit configuration
type ConnectionLimitConfig struct {
	// MaxConnectionsPerUser is the maximum number of connections per user
	MaxConnectionsPerUser int

	// MaxConnectionsPerIP is the maximum number of connections per IP address
	MaxConnectionsPerIP int

	// MaxTotalConnections is the maximum total number of connections
	MaxTotalConnections int

	// ConnectionTimeout is the maximum idle time for a connection
	ConnectionTimeout time.Duration

	// BypassRoles are roles that bypass connection limits
	BypassRoles []string
}

// DefaultWebSocketConfig returns a default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		AllowedOrigins:   []string{},
		AllowCredentials: true,
		ReadTimeout:      60 * time.Second,
		WriteTimeout:     10 * time.Second,
		PingInterval:     54 * time.Second,
		PongTimeout:      60 * time.Second,
		RateLimitConfig: &RateLimitConfig{
			MessagesPerMinute:       60,  // 1 message per second
			BurstSize:               10,  // Allow burst of 10 messages
			GlobalMessagesPerMinute: 600, // 10 messages per second globally
			GlobalBurstSize:         100,
			BypassRoles:             []string{"admin"},
		},
		ConnectionLimitConfig: &ConnectionLimitConfig{
			MaxConnectionsPerUser: 5,
			MaxConnectionsPerIP:   10,
			MaxTotalConnections:   1000,
			ConnectionTimeout:     30 * time.Minute,
			BypassRoles:           []string{"admin"},
		},
	}
}

// CheckOrigin checks if the origin is allowed
func (c *WebSocketConfig) CheckOrigin(r *http.Request) bool {
	// If custom function is provided, use it
	if c.CheckOriginFunc != nil {
		return c.CheckOriginFunc(r)
	}

	// If no allowed origins specified, allow all (development mode)
	if len(c.AllowedOrigins) == 0 {
		return true
	}

	// Get the origin header
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No origin header, check if it's a same-origin request
		return c.isSameOrigin(r)
	}

	// Parse the origin URL
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}

	// Check against allowed origins
	for _, allowed := range c.AllowedOrigins {
		if c.matchOrigin(originURL, allowed) {
			return true
		}
	}

	return false
}

// matchOrigin checks if an origin matches an allowed pattern
func (c *WebSocketConfig) matchOrigin(originURL *url.URL, allowed string) bool {
	// Handle wildcard
	if allowed == "*" {
		return true
	}

	// Handle exact match
	if originURL.String() == allowed {
		return true
	}

	// Handle scheme-less match (e.g., "localhost:3000")
	if !strings.Contains(allowed, "://") {
		if originURL.Host == allowed {
			return true
		}
	}

	// Handle wildcard subdomains (e.g., "*.example.com")
	if strings.HasPrefix(allowed, "*.") {
		domain := strings.TrimPrefix(allowed, "*.")
		// Check if origin is a subdomain (not the domain itself)
		if strings.HasSuffix(originURL.Host, "."+domain) {
			return true
		}
	}

	// Parse allowed URL for more complex matching
	allowedURL, err := url.Parse(allowed)
	if err != nil {
		return false
	}

	// Check scheme, host, and port
	return originURL.Scheme == allowedURL.Scheme &&
		originURL.Host == allowedURL.Host
}

// isSameOrigin checks if the request is from the same origin
func (c *WebSocketConfig) isSameOrigin(r *http.Request) bool {
	// Get the host from the request
	host := r.Host

	// Get the referer
	referer := r.Header.Get("Referer")
	if referer == "" {
		// No referer, could be a direct connection
		return true
	}

	refererURL, err := url.Parse(referer)
	if err != nil {
		return false
	}

	// Compare hosts
	return refererURL.Host == host
}

// BuildWebSocketUpgrader creates a WebSocket upgrader with the configured origin check
func (c *WebSocketConfig) BuildWebSocketUpgrader() *Upgrader {
	return &Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		CheckOrigin:       c.CheckOrigin,
		EnableCompression: true,
	}
}

// Upgrader wraps gorilla/websocket.Upgrader with additional features
type Upgrader struct {
	ReadBufferSize    int
	WriteBufferSize   int
	CheckOrigin       func(r *http.Request) bool
	EnableCompression bool
}
