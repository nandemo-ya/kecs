package api

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for WebSocket connections
type RateLimiter struct {
	config          *RateLimitConfig
	perConnLimiters map[string]*rate.Limiter
	globalLimiter   *rate.Limiter
	mu              sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		return nil
	}

	return &RateLimiter{
		config:          config,
		perConnLimiters: make(map[string]*rate.Limiter),
		globalLimiter: rate.NewLimiter(
			rate.Limit(float64(config.GlobalMessagesPerMinute)/60.0),
			config.GlobalBurstSize,
		),
	}
}

// Allow checks if a message is allowed based on rate limiting
func (r *RateLimiter) Allow(clientID string, authInfo *AuthInfo) bool {
	if r == nil {
		return true
	}

	// Check if user has bypass role
	if authInfo != nil && r.hassBypassRole(authInfo.Roles) {
		return true
	}

	// Check global rate limit first
	if !r.globalLimiter.Allow() {
		return false
	}

	// Check per-connection rate limit
	r.mu.Lock()
	limiter, exists := r.perConnLimiters[clientID]
	if !exists {
		limiter = rate.NewLimiter(
			rate.Limit(float64(r.config.MessagesPerMinute)/60.0),
			r.config.BurstSize,
		)
		r.perConnLimiters[clientID] = limiter
	}
	r.mu.Unlock()

	return limiter.Allow()
}

// Remove removes a client's rate limiter
func (r *RateLimiter) Remove(clientID string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	delete(r.perConnLimiters, clientID)
	r.mu.Unlock()
}

// hassBypassRole checks if any of the user's roles are in the bypass list
func (r *RateLimiter) hassBypassRole(userRoles []string) bool {
	for _, userRole := range userRoles {
		for _, bypassRole := range r.config.BypassRoles {
			if userRole == bypassRole {
				return true
			}
		}
	}
	return false
}

// ConnectionLimiter manages connection limits
type ConnectionLimiter struct {
	config             *ConnectionLimitConfig
	connectionsPerUser map[string]int
	connectionsPerIP   map[string]int
	totalConnections   int
	mu                 sync.RWMutex
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(config *ConnectionLimitConfig) *ConnectionLimiter {
	if config == nil {
		return nil
	}

	return &ConnectionLimiter{
		config:             config,
		connectionsPerUser: make(map[string]int),
		connectionsPerIP:   make(map[string]int),
	}
}

// CanConnect checks if a new connection is allowed
func (c *ConnectionLimiter) CanConnect(r *http.Request, authInfo *AuthInfo) bool {
	if c == nil {
		return true
	}

	// Check if user has bypass role
	if authInfo != nil && c.hassBypassRole(authInfo.Roles) {
		return true
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check total connections
	if c.config.MaxTotalConnections > 0 && c.totalConnections >= c.config.MaxTotalConnections {
		return false
	}

	// Check per-user limit
	if authInfo != nil && c.config.MaxConnectionsPerUser > 0 {
		userConnections := c.connectionsPerUser[authInfo.UserID]
		if userConnections >= c.config.MaxConnectionsPerUser {
			return false
		}
	}

	// Check per-IP limit
	clientIP := getClientIP(r)
	if c.config.MaxConnectionsPerIP > 0 && clientIP != "" {
		ipConnections := c.connectionsPerIP[clientIP]
		if ipConnections >= c.config.MaxConnectionsPerIP {
			return false
		}
	}

	return true
}

// AddConnection registers a new connection
func (c *ConnectionLimiter) AddConnection(r *http.Request, authInfo *AuthInfo) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalConnections++

	if authInfo != nil {
		c.connectionsPerUser[authInfo.UserID]++
	}

	clientIP := getClientIP(r)
	if clientIP != "" {
		c.connectionsPerIP[clientIP]++
	}
}

// RemoveConnection removes a connection
func (c *ConnectionLimiter) RemoveConnection(r *http.Request, authInfo *AuthInfo) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.totalConnections > 0 {
		c.totalConnections--
	}

	if authInfo != nil {
		if count := c.connectionsPerUser[authInfo.UserID]; count > 0 {
			c.connectionsPerUser[authInfo.UserID]--
			if c.connectionsPerUser[authInfo.UserID] == 0 {
				delete(c.connectionsPerUser, authInfo.UserID)
			}
		}
	}

	clientIP := getClientIP(r)
	if clientIP != "" {
		if count := c.connectionsPerIP[clientIP]; count > 0 {
			c.connectionsPerIP[clientIP]--
			if c.connectionsPerIP[clientIP] == 0 {
				delete(c.connectionsPerIP, clientIP)
			}
		}
	}
}

// GetConnectionStats returns current connection statistics
func (c *ConnectionLimiter) GetConnectionStats() map[string]interface{} {
	if c == nil {
		return nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"total_connections":  c.totalConnections,
		"users_connected":    len(c.connectionsPerUser),
		"unique_ips":         len(c.connectionsPerIP),
		"max_total":          c.config.MaxTotalConnections,
		"max_per_user":       c.config.MaxConnectionsPerUser,
		"max_per_ip":         c.config.MaxConnectionsPerIP,
	}
}

// hassBypassRole checks if any of the user's roles are in the bypass list
func (c *ConnectionLimiter) hassBypassRole(userRoles []string) bool {
	for _, userRole := range userRoles {
		for _, bypassRole := range c.config.BypassRoles {
			if userRole == bypassRole {
				return true
			}
		}
	}
	return false
}

// CleanupStaleConnections removes stale connection entries
func (c *ConnectionLimiter) CleanupStaleConnections(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			// In a real implementation, we would track connection timestamps
			// and remove entries that haven't been active
			c.mu.Unlock()
		}
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied connections)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the chain
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to remote address
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}