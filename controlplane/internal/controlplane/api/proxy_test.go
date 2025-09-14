package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestProxyHandler_ContentTypeRouting is removed as the routing logic has been simplified
// The new routing is based solely on Content-Type header, tested in TestProxyHandler_RouteIntegration

// TestProxyHandler_isELBv2Request is removed as body inspection is no longer used
// The new routing is based solely on Content-Type header to avoid body consumption issues

func TestProxyHandler_RouteIntegration(t *testing.T) {
	// Create test handlers
	ecsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ECS"))
	})

	elbv2Handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ELBv2"))
	})

	sdHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SD"))
	})

	// Create proxy handler
	proxyHandler, err := NewProxyHandler("http://localhost:4566", ecsHandler, elbv2Handler, sdHandler)
	if err != nil {
		t.Fatalf("Failed to create proxy handler: %v", err)
	}

	tests := []struct {
		name         string
		request      *http.Request
		expectedBody string
	}{
		{
			name: "ELBv2 form data request should route to ELBv2",
			request: func() *http.Request {
				body := "Action=DescribeLoadBalancers&Version=2015-12-01"
				req := httptest.NewRequest("POST", "/", bytes.NewReader([]byte(body)))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			}(),
			expectedBody: "ELBv2",
		},
		{
			name: "ECS X-Amz-Target request should route to ECS",
			request: func() *http.Request {
				req := httptest.NewRequest("POST", "/", nil)
				req.Header.Set("X-Amz-Target", "AmazonEC2ContainerServiceV20141113.ListClusters")
				return req
			}(),
			expectedBody: "ECS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			proxyHandler.ServeHTTP(recorder, tt.request)

			body := recorder.Body.String()
			if body != tt.expectedBody {
				t.Errorf("Expected body %s, got %s", tt.expectedBody, body)
			}
		})
	}
}
