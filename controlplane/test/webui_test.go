package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nandemo-ya/kecs/controlplane/internal/controlplane/api"
)

func TestWebUIRouting(t *testing.T) {
	// Create a test file system
	fs := os.DirFS("testdata/webui")
	handler := api.NewWebUIHandler(fs)

	// Test cases for various UI paths
	testCases := []struct {
		path           string
		expectedStatus int
		description    string
	}{
		{"/", http.StatusOK, "Root path"},
		{"/dashboard", http.StatusOK, "Dashboard path (SPA routing)"},
		{"/tasks", http.StatusOK, "Tasks path (SPA routing)"},
		{"/services/my-service", http.StatusOK, "Nested service path (SPA routing)"},
		{"/static/css/main.css", http.StatusOK, "Static asset path"},
		{"/static/js/bundle.js", http.StatusOK, "JavaScript bundle path"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()
			
			handler.ServeHTTP(w, req)
			
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d for path %s, got %d", 
					tc.expectedStatus, tc.path, w.Code)
			}
		})
	}
}

func TestWebUIWithPrefix(t *testing.T) {
	// Create a test file system
	fs := os.DirFS("testdata/webui")
	handler := api.NewWebUIHandler(fs)
	
	// Wrap with StripPrefix as done in server.go
	wrappedHandler := http.StripPrefix("/ui", handler)

	// Test cases with /ui prefix
	testCases := []struct {
		path           string
		expectedStatus int
		description    string
	}{
		{"/ui/", http.StatusOK, "UI root path"},
		{"/ui/dashboard", http.StatusOK, "UI dashboard path"},
		{"/ui/tasks/task-123", http.StatusOK, "UI nested task path"},
		{"/ui/static/css/main.css", http.StatusOK, "UI static asset"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()
			
			wrappedHandler.ServeHTTP(w, req)
			
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d for path %s, got %d", 
					tc.expectedStatus, tc.path, w.Code)
			}
		})
	}
}