package api

import (
	"net/http"
	"net/url"
	"strings"
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
}

// AuthInfo represents authenticated user information
type AuthInfo struct {
	UserID      string
	Username    string
	Roles       []string
	Permissions []string
	Metadata    map[string]interface{}
}

// DefaultWebSocketConfig returns a default WebSocket configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		AllowedOrigins:   []string{},
		AllowCredentials: true,
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
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     c.CheckOrigin,
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