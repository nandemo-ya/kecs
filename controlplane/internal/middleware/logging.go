package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// LoggingConfig configures the logging middleware
type LoggingConfig struct {
	// SkipPaths are paths that should not be logged (e.g., health checks)
	SkipPaths []string
	// LogLevel determines the log level for different operations
	LogLevel LogLevelFunc
	// LogBody determines whether to log request/response bodies
	LogBody bool
	// MaxBodySize is the maximum body size to log (to avoid huge logs)
	MaxBodySize int64
}

// LogLevelFunc determines the log level based on the request
type LogLevelFunc func(method, path string, status int, duration time.Duration) string

// DefaultLogLevel provides sensible defaults for log levels
func DefaultLogLevel(method, path string, status int, duration time.Duration) string {
	// Health checks and metrics should be debug
	if strings.Contains(path, "/health") || strings.Contains(path, "/metrics") {
		return "debug"
	}

	// Frequently called read operations should be debug
	if method == "POST" {
		// Check for frequent ECS API calls that should be debug
		if strings.Contains(path, "/ListTasks") ||
			strings.Contains(path, "/DescribeTasks") ||
			strings.Contains(path, "/DescribeTaskDefinition") ||
			strings.Contains(path, "/ListServices") ||
			strings.Contains(path, "/DescribeServices") {
			return "debug"
		}
	}

	// Errors should always be logged at error level
	if status >= 500 {
		return "error"
	}
	if status >= 400 {
		return "warn"
	}

	// Slow requests should be warned about
	if duration > 1*time.Second {
		return "warn"
	}

	// Everything else is info
	return "info"
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
	body   *bytes.Buffer
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.body != nil {
		rw.body.Write(b)
	}
	return rw.ResponseWriter.Write(b)
}

// LoggingMiddleware creates a middleware that logs HTTP requests
func LoggingMiddleware(config *LoggingConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = &LoggingConfig{
			LogLevel:    DefaultLogLevel,
			LogBody:     false,
			MaxBodySize: 1024, // 1KB default
		}
	}

	if config.LogLevel == nil {
		config.LogLevel = DefaultLogLevel
	}

	skipPathMap := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPathMap[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if we should skip logging for this path
			if skipPathMap[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Prepare logging fields
			fields := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			}

			// Extract action from request body for ECS API calls
			var action string
			if r.Method == "POST" && strings.Contains(r.URL.Path, "/v1/") {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body = io.NopCloser(bytes.NewReader(body))

					// Try to extract X-Amz-Target header first (standard AWS API)
					if target := r.Header.Get("X-Amz-Target"); target != "" {
						parts := strings.Split(target, ".")
						if len(parts) > 0 {
							action = parts[len(parts)-1]
						}
					}

					// If no header, try to extract from URL path
					if action == "" {
						pathParts := strings.Split(r.URL.Path, "/")
						for _, part := range pathParts {
							if strings.HasPrefix(part, "Describe") ||
								strings.HasPrefix(part, "List") ||
								strings.HasPrefix(part, "Create") ||
								strings.HasPrefix(part, "Delete") ||
								strings.HasPrefix(part, "Update") ||
								strings.HasPrefix(part, "Register") ||
								strings.HasPrefix(part, "Deregister") ||
								strings.HasPrefix(part, "Run") ||
								strings.HasPrefix(part, "Stop") ||
								strings.HasPrefix(part, "Put") {
								action = part
								break
							}
						}
					}

					if action != "" {
						fields = append(fields, "action", action)
					}

					// Log request body if configured and it's small enough
					if config.LogBody && int64(len(body)) <= config.MaxBodySize {
						var jsonBody map[string]interface{}
						if err := json.Unmarshal(body, &jsonBody); err == nil {
							// Remove sensitive fields
							delete(jsonBody, "password")
							delete(jsonBody, "token")
							delete(jsonBody, "secret")
							fields = append(fields, "request_body", jsonBody)
						}
					}
				}
			}

			// Wrap response writer to capture status
			wrapped := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}
			if config.LogBody {
				wrapped.body = &bytes.Buffer{}
			}

			// Call the next handler
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Add response fields
			fields = append(fields,
				"status", wrapped.status,
				"duration_ms", duration.Milliseconds(),
			)

			// Log response body if configured
			if config.LogBody && wrapped.body != nil && int64(wrapped.body.Len()) <= config.MaxBodySize {
				var jsonBody map[string]interface{}
				if err := json.Unmarshal(wrapped.body.Bytes(), &jsonBody); err == nil {
					fields = append(fields, "response_body", jsonBody)
				}
			}

			// Determine log level and log the request
			level := config.LogLevel(r.Method, r.URL.Path, wrapped.status, duration)

			// Build the message
			message := "HTTP Request"
			if action != "" {
				message = action
			}

			switch level {
			case "debug":
				logging.Debug(message, fields...)
			case "info":
				logging.Info(message, fields...)
			case "warn":
				logging.Warn(message, fields...)
			case "error":
				logging.Error(message, fields...)
			default:
				logging.Info(message, fields...)
			}
		})
	}
}

// APILoggingMiddleware returns a pre-configured middleware for ECS API
func APILoggingMiddleware() func(http.Handler) http.Handler {
	return LoggingMiddleware(&LoggingConfig{
		SkipPaths: []string{"/health"},
		LogLevel:  DefaultLogLevel,
		LogBody:   false, // Don't log bodies by default
	})
}

// AdminLoggingMiddleware returns a pre-configured middleware for Admin API
func AdminLoggingMiddleware() func(http.Handler) http.Handler {
	return LoggingMiddleware(&LoggingConfig{
		SkipPaths: []string{"/health", "/metrics"},
		LogLevel: func(method, path string, status int, duration time.Duration) string {
			// Admin API is less frequently called, so we can log more at info level
			if status >= 500 {
				return "error"
			}
			if status >= 400 {
				return "warn"
			}
			if duration > 2*time.Second {
				return "warn"
			}
			// Most admin operations should be logged at info
			return "info"
		},
		LogBody: false,
	})
}
