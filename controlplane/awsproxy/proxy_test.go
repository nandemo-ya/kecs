package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAWSProxy(t *testing.T) {
	tests := []struct {
		name               string
		localStackEndpoint string
		debug              bool
		wantErr            bool
	}{
		{
			name:               "valid endpoint",
			localStackEndpoint: "http://localhost:4566",
			debug:              false,
			wantErr:            false,
		},
		{
			name:               "valid endpoint with debug",
			localStackEndpoint: "http://localhost:4566",
			debug:              true,
			wantErr:            false,
		},
		{
			name:               "invalid endpoint",
			localStackEndpoint: "not-a-url",
			debug:              false,
			wantErr:            true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxy, err := NewAWSProxy(tt.localStackEndpoint, tt.debug)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAWSProxy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && proxy == nil {
				t.Error("NewAWSProxy() returned nil proxy without error")
			}
		})
	}
}

func TestAWSProxy_HandleHealth(t *testing.T) {
	// Create a test server that mimics LocalStack health endpoint
	localStackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/_localstack/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"running"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer localStackServer.Close()

	proxy, err := NewAWSProxy(localStackServer.URL, false)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Test health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify response contains expected fields
	body := w.Body.String()
	if !contains(body, `"status":"healthy"`) {
		t.Errorf("Expected healthy status in response, got: %s", body)
	}
	if !contains(body, `"localstack_endpoint"`) {
		t.Errorf("Expected localstack_endpoint in response, got: %s", body)
	}
}

func TestAWSProxy_HandleHealthUnhealthy(t *testing.T) {
	// Create proxy with unreachable endpoint
	proxy, err := NewAWSProxy("http://localhost:99999", false)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Override timeout for faster test
	proxy.httpClient.Timeout = 1 * time.Second

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, resp.StatusCode)
	}

	body := w.Body.String()
	if !contains(body, `"status":"unhealthy"`) {
		t.Errorf("Expected unhealthy status in response, got: %s", body)
	}
}

func TestAWSProxy_ExtractAWSService(t *testing.T) {
	proxy, _ := NewAWSProxy("http://localhost:4566", false)

	tests := []struct {
		name            string
		setupRequest    func() *http.Request
		expectedService string
	}{
		{
			name: "AWS Signature V4 - S3",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/bucket/key", nil)
				req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20230101/us-east-1/s3/aws4_request")
				return req
			},
			expectedService: "s3",
		},
		{
			name: "AWS Signature V4 - IAM",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/", nil)
				req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20230101/us-east-1/iam/aws4_request")
				return req
			},
			expectedService: "iam",
		},
		{
			name: "Host header - S3",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/bucket/key", nil)
				req.Host = "s3.localhost.localstack.cloud:4566"
				return req
			},
			expectedService: "s3",
		},
		{
			name: "X-Amz-Target - DynamoDB",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/", nil)
				req.Header.Set("X-Amz-Target", "DynamoDB_20120810.ListTables")
				return req
			},
			expectedService: "dynamodb_20120810",
		},
		{
			name: "User-Agent - S3Manager",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("User-Agent", "aws-sdk-go/1.44.122 (go1.19.2; linux; amd64) S3Manager")
				return req
			},
			expectedService: "s3manager",
		},
		{
			name: "No service hints",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			expectedService: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			service := proxy.extractAWSService(req)
			if service != tt.expectedService {
				t.Errorf("extractAWSService() = %v, want %v", service, tt.expectedService)
			}
		})
	}
}

func TestAWSProxy_ProxyRequest(t *testing.T) {
	// Create a test backend server
	backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back some request info
		w.Header().Set("X-Echo-Method", r.Method)
		w.Header().Set("X-Echo-Path", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Backend response for %s %s", r.Method, r.URL.Path)
	}))
	defer backendServer.Close()

	proxy, err := NewAWSProxy(backendServer.URL, false)
	if err != nil {
		t.Fatalf("Failed to create proxy: %v", err)
	}

	// Test proxying a request
	req := httptest.NewRequest("PUT", "/test-bucket/test-key", nil)
	req.Header.Set("Content-Type", "application/octet-stream")
	w := httptest.NewRecorder()

	proxy.ServeHTTP(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify headers were passed through
	if resp.Header.Get("X-Echo-Method") != "PUT" {
		t.Errorf("Expected method PUT, got %s", resp.Header.Get("X-Echo-Method"))
	}
	if resp.Header.Get("X-Echo-Path") != "/test-bucket/test-key" {
		t.Errorf("Expected path /test-bucket/test-key, got %s", resp.Header.Get("X-Echo-Path"))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}