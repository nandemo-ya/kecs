package api

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestProxyHandler_shouldRouteToELBv2(t *testing.T) {
	tests := []struct {
		name     string
		request  *http.Request
		expected bool
	}{
		{
			name: "ELBv2 request with X-Amz-Target header",
			request: &http.Request{
				Method: "POST",
				URL:    &url.URL{Path: "/"},
				Header: http.Header{
					"X-Amz-Target": []string{"AWSie_backend_200507.DescribeLoadBalancers"},
				},
			},
			expected: true,
		},
		{
			name: "ELBv2 request with ElasticLoadBalancing in header",
			request: &http.Request{
				Method: "POST",
				URL:    &url.URL{Path: "/"},
				Header: http.Header{
					"X-Amz-Target": []string{"ElasticLoadBalancing_v2.CreateLoadBalancer"},
				},
			},
			expected: true,
		},
		{
			name: "ELBv2 request with path containing elasticloadbalancing",
			request: &http.Request{
				Method: "POST",
				URL:    &url.URL{Path: "/elasticloadbalancing/v2"},
			},
			expected: true,
		},
		{
			name: "ELBv2 request with Action in form data",
			request: func() *http.Request {
				body := "Action=DescribeLoadBalancers&Version=2015-12-01"
				req := &http.Request{
					Method: "POST",
					URL:    &url.URL{Path: "/"},
					Body:   io.NopCloser(strings.NewReader(body)),
					Header: http.Header{
						"Content-Type": []string{"application/x-www-form-urlencoded"},
					},
				}
				return req
			}(),
			expected: true,
		},
		{
			name: "ELBv2 request with CreateTargetGroup Action",
			request: func() *http.Request {
				body := "Action=CreateTargetGroup&Name=my-targets&Protocol=HTTP&Port=80&VpcId=vpc-12345678"
				req := &http.Request{
					Method: "POST",
					URL:    &url.URL{Path: "/"},
					Body:   io.NopCloser(strings.NewReader(body)),
					Header: http.Header{
						"Content-Type": []string{"application/x-www-form-urlencoded"},
					},
				}
				return req
			}(),
			expected: true,
		},
		{
			name: "Non-ELBv2 request with ECS Action",
			request: func() *http.Request {
				body := "Action=ListClusters&Version=2014-11-13"
				req := &http.Request{
					Method: "POST",
					URL:    &url.URL{Path: "/"},
					Body:   io.NopCloser(strings.NewReader(body)),
					Header: http.Header{
						"Content-Type": []string{"application/x-www-form-urlencoded"},
					},
				}
				return req
			}(),
			expected: false,
		},
		{
			name: "GET request should not check body",
			request: &http.Request{
				Method: "GET",
				URL:    &url.URL{Path: "/"},
			},
			expected: false,
		},
	}

	// Create a test proxy handler
	handler := &ProxyHandler{
		localStackURL: &url.URL{Host: "localhost:4566"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.shouldRouteToELBv2(tt.request)
			if result != tt.expected {
				t.Errorf("shouldRouteToELBv2() = %v, want %v", result, tt.expected)
			}

			// Verify that body can still be read after check
			if tt.request.Body != nil {
				bodyBytes, err := io.ReadAll(tt.request.Body)
				if err != nil {
					t.Errorf("Failed to read body after check: %v", err)
				}
				if len(bodyBytes) == 0 && tt.request.Method == "POST" {
					t.Error("Body was consumed and not restored")
				}
			}
		})
	}
}

func TestProxyHandler_isELBv2Request(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{
			name:     "DescribeLoadBalancers action",
			body:     "Action=DescribeLoadBalancers&Version=2015-12-01",
			expected: true,
		},
		{
			name:     "CreateLoadBalancer action",
			body:     "Action=CreateLoadBalancer&Name=my-lb&Subnets.member.1=subnet-12345",
			expected: true,
		},
		{
			name:     "CreateTargetGroup action",
			body:     "Action=CreateTargetGroup&Name=my-targets&Protocol=HTTP",
			expected: true,
		},
		{
			name:     "RegisterTargets action",
			body:     "Action=RegisterTargets&TargetGroupArn=arn:aws:elasticloadbalancing",
			expected: true,
		},
		{
			name:     "Non-ELBv2 action",
			body:     "Action=ListClusters&Version=2014-11-13",
			expected: false,
		},
		{
			name:     "No action parameter",
			body:     "SomeOtherParam=value&Version=2015-12-01",
			expected: false,
		},
		{
			name:     "Action in middle of body",
			body:     "Version=2015-12-01&Action=DescribeTargetHealth&TargetGroupArn=arn",
			expected: true,
		},
	}

	handler := &ProxyHandler{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Body: io.NopCloser(strings.NewReader(tt.body)),
			}
			result := handler.isELBv2Request(req)
			if result != tt.expected {
				t.Errorf("isELBv2Request() = %v, want %v", result, tt.expected)
			}

			// Verify body was restored
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Errorf("Failed to read body after check: %v", err)
			}
			if string(bodyBytes) != tt.body {
				t.Errorf("Body was not properly restored. Got %s, want %s", string(bodyBytes), tt.body)
			}
		})
	}
}

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
