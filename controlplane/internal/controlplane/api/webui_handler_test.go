package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestWebUIHandler_ServeHTTP(t *testing.T) {
	// Create a test file system
	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><body>Test</body></html>`),
		},
		"static/css/main.css": &fstest.MapFile{
			Data: []byte(`body { font-family: Arial; }`),
		},
		"static/js/app.js": &fstest.MapFile{
			Data: []byte(`console.log("test");`),
		},
	}

	handler := NewWebUIHandler(fs)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedType   string
		description    string
	}{
		{
			name:           "Root path",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			description:    "Should serve index.html for root",
		},
		{
			name:           "Static CSS file",
			path:           "/static/css/main.css",
			expectedStatus: http.StatusOK,
			expectedType:   "text/css",
			description:    "Should serve CSS with correct content type",
		},
		{
			name:           "Static JS file",
			path:           "/static/js/app.js",
			expectedStatus: http.StatusOK,
			expectedType:   "application/javascript",
			description:    "Should serve JS with correct content type",
		},
		{
			name:           "Non-existent path (SPA routing)",
			path:           "/dashboard",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			description:    "Should serve index.html for client-side routing",
		},
		{
			name:           "Nested non-existent path",
			path:           "/tasks/task-123",
			expectedStatus: http.StatusOK,
			expectedType:   "text/html",
			description:    "Should serve index.html for nested client routes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.expectedType) {
				t.Errorf("Expected content type to contain %s, got %s", tt.expectedType, contentType)
			}
		})
	}
}

func TestWebUIHandler_WithStripPrefix(t *testing.T) {
	// Create a test file system
	fs := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte(`<!DOCTYPE html><html><body>Test</body></html>`),
		},
	}

	handler := NewWebUIHandler(fs)
	// Simulate the server setup with StripPrefix
	wrappedHandler := http.StripPrefix("/ui", handler)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		description    string
	}{
		{
			name:           "UI root with trailing slash",
			path:           "/ui/",
			expectedStatus: http.StatusOK,
			description:    "Should serve index.html for /ui/",
		},
		{
			name:           "UI nested path",
			path:           "/ui/tasks/123",
			expectedStatus: http.StatusOK,
			description:    "Should serve index.html for nested UI paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
