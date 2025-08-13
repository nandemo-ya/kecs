package api

import (
	"net/http"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		// In production, you should validate the origin against a whitelist
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Amz-Target, X-Amz-Date, X-Amz-Security-Token, X-Amz-User-Agent")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityHeadersMiddleware adds security headers to responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// For API endpoints, set strict CSP
		if !strings.HasPrefix(r.URL.Path, "/ui") && !strings.HasPrefix(r.URL.Path, "/ws") {
			w.Header().Set("Content-Security-Policy", "default-src 'none'")
		}

		next.ServeHTTP(w, r)
	})
}

// APIProxyMiddleware handles proxying API requests from the Web UI
func APIProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the request is from the Web UI and targets an API endpoint
		if strings.HasPrefix(r.URL.Path, "/api/") {
			// Strip the /api prefix and forward to the ECS API handler
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple request logging
		// In production, you might want to use a more sophisticated logging solution
		next.ServeHTTP(w, r)
	})
}

// LocalStackProxyMiddleware intercepts AWS API calls and routes them to LocalStack
func LocalStackProxyMiddleware(next http.Handler, server *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Debug log
		target := r.Header.Get("X-Amz-Target")
		if target != "" {
			logging.Debug("[LocalStackProxyMiddleware] Request",
				"method", r.Method, "path", r.URL.Path, "target", target, "hasAuth", r.Header.Get("Authorization") != "")
		}

		// Dynamically check if awsProxyRouter is available
		if server.awsProxyRouter != nil && server.awsProxyRouter.LocalStackManager != nil &&
			ShouldProxyToLocalStack(r, server.awsProxyRouter.LocalStackManager) {
			logging.Debug("[LocalStackProxyMiddleware] Proxying to LocalStack", "method", r.Method, "path", r.URL.Path)
			server.awsProxyRouter.AWSProxyHandler.ServeHTTP(w, r)
			return
		}

		// Not an AWS API call or LocalStack is not available
		next.ServeHTTP(w, r)
	})
}
