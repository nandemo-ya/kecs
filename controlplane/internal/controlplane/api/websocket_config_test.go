package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebSocketConfig_CheckOrigin(t *testing.T) {
	tests := []struct {
		name           string
		config         *WebSocketConfig
		origin         string
		referer        string
		host           string
		expectedResult bool
	}{
		{
			name: "empty allowed origins allows all",
			config: &WebSocketConfig{
				AllowedOrigins: []string{},
			},
			origin:         "http://example.com",
			expectedResult: true,
		},
		{
			name: "wildcard allows all origins",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"*"},
			},
			origin:         "http://malicious.com",
			expectedResult: true,
		},
		{
			name: "exact match allowed",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://localhost:3000", "https://app.example.com"},
			},
			origin:         "http://localhost:3000",
			expectedResult: true,
		},
		{
			name: "exact match denied",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
			},
			origin:         "http://localhost:3001",
			expectedResult: false,
		},
		{
			name: "scheme mismatch denied",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"https://example.com"},
			},
			origin:         "http://example.com",
			expectedResult: false,
		},
		{
			name: "wildcard subdomain match",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"*.example.com"},
			},
			origin:         "https://app.example.com",
			expectedResult: true,
		},
		{
			name: "wildcard subdomain with nested subdomain",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"*.example.com"},
			},
			origin:         "https://api.app.example.com",
			expectedResult: true,
		},
		{
			name: "wildcard subdomain root domain denied",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"*.example.com"},
			},
			origin:         "https://example.com",
			expectedResult: false,
		},
		{
			name: "scheme-less match",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"localhost:3000"},
			},
			origin:         "http://localhost:3000",
			expectedResult: true,
		},
		{
			name: "same origin with no origin header",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://example.com"},
			},
			origin:         "",
			referer:        "http://localhost:8080/page",
			host:           "localhost:8080",
			expectedResult: true,
		},
		{
			name: "different origin with no origin header",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://example.com"},
			},
			origin:         "",
			referer:        "http://example.com/page",
			host:           "localhost:8080",
			expectedResult: false,
		},
		{
			name: "custom check function overrides",
			config: &WebSocketConfig{
				AllowedOrigins: []string{"http://denied.com"},
				CheckOriginFunc: func(r *http.Request) bool {
					// Custom function that always allows
					return true
				},
			},
			origin:         "http://custom.com",
			expectedResult: true,
		},
		{
			name: "multiple allowed origins",
			config: &WebSocketConfig{
				AllowedOrigins: []string{
					"http://localhost:3000",
					"http://localhost:8080",
					"https://production.app.com",
					"*.staging.app.com",
				},
			},
			origin:         "https://api.staging.app.com",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest("GET", "/ws", nil)
			
			// Set headers
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			if tt.host != "" {
				req.Host = tt.host
			}

			// Check origin
			result := tt.config.CheckOrigin(req)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestWebSocketConfig_matchOrigin(t *testing.T) {
	config := &WebSocketConfig{}

	tests := []struct {
		name     string
		origin   string
		allowed  string
		expected bool
	}{
		{
			name:     "wildcard matches anything",
			origin:   "http://example.com",
			allowed:  "*",
			expected: true,
		},
		{
			name:     "exact match",
			origin:   "http://example.com",
			allowed:  "http://example.com",
			expected: true,
		},
		{
			name:     "port mismatch",
			origin:   "http://example.com:8080",
			allowed:  "http://example.com",
			expected: false,
		},
		{
			name:     "subdomain wildcard match",
			origin:   "https://app.example.com",
			allowed:  "*.example.com",
			expected: true,
		},
		{
			name:     "subdomain wildcard no match on root",
			origin:   "https://example.com",
			allowed:  "*.example.com",
			expected: false,
		},
		{
			name:     "scheme-less match with http",
			origin:   "http://localhost:3000",
			allowed:  "localhost:3000",
			expected: true,
		},
		{
			name:     "scheme-less match with https",
			origin:   "https://localhost:3000",
			allowed:  "localhost:3000",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originURL, err := parseURL(tt.origin)
			assert.NoError(t, err)

			result := config.matchOrigin(originURL, tt.allowed)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultWebSocketConfig(t *testing.T) {
	config := DefaultWebSocketConfig()
	
	assert.NotNil(t, config)
	assert.Empty(t, config.AllowedOrigins)
	assert.True(t, config.AllowCredentials)
	assert.Nil(t, config.CheckOriginFunc)
	
	// Default config should allow all origins
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	assert.True(t, config.CheckOrigin(req))
}

// Helper function to parse URL for tests
func parseURL(urlStr string) (*url.URL, error) {
	return url.Parse(urlStr)
}