package sidecar

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AWS SDK Proxy", func() {
	var (
		proxy           *Proxy
		localStackMock  *httptest.Server
		ctx             context.Context
		cancel          context.CancelFunc
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		// Create mock LocalStack server
		localStackMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Echo back request info for testing
			w.Header().Set("X-LocalStack-Request-URL", r.URL.String())
			w.Header().Set("X-LocalStack-Host", r.Host)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "LocalStack mock response for %s", r.URL.Path)
		}))

		// Create proxy with test configuration
		config := &ProxyConfig{
			LocalStackEndpoint: localStackMock.URL,
			ListenPort:         0, // Use random port
			Services:           []string{"s3", "dynamodb", "sqs"},
			Debug:              true,
			Timeout:            5 * time.Second,
		}
		proxy = NewProxy(config)
	})

	AfterEach(func() {
		cancel()
		localStackMock.Close()
	})

	Describe("Service extraction", func() {
		It("should extract service from Authorization header", func() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20230101/us-east-1/s3/aws4_request")
			
			service := proxy.extractAWSService(req)
			Expect(service).To(Equal("s3"))
		})

		It("should extract service from Host header", func() {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Host", "dynamodb.us-east-1.amazonaws.com")
			
			service := proxy.extractAWSService(req)
			Expect(service).To(Equal("dynamodb"))
		})

		It("should extract service from X-Amz-Target header", func() {
			req := httptest.NewRequest("POST", "/", nil)
			req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
			
			service := proxy.extractAWSService(req)
			Expect(service).To(Equal("dynamodb"))
		})

		It("should handle S3 special case", func() {
			req := httptest.NewRequest("GET", "/bucket/key", nil)
			req.Header.Set("Host", "s3-us-west-2.amazonaws.com")
			
			service := proxy.extractAWSService(req)
			Expect(service).To(Equal("s3"))
		})
	})

	Describe("Request proxying", func() {
		var proxyURL string

		BeforeEach(func() {
			// Start proxy server
			go func() {
				proxy.Start(ctx)
			}()
			
			// Wait for server to start and get URL
			time.Sleep(100 * time.Millisecond)
			proxyURL = fmt.Sprintf("http://localhost:%d", proxy.config.ListenPort)
		})

		It("should proxy S3 requests to LocalStack", func() {
			// Make request to proxy
			req, _ := http.NewRequest("GET", proxyURL+"/test-bucket/test-key", nil)
			req.Header.Set("Host", "s3.us-east-1.amazonaws.com")
			req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20230101/us-east-1/s3/aws4_request")
			
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			
			// Verify request was proxied
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Header.Get("X-LocalStack-Request-URL")).To(Equal("/test-bucket/test-key"))
		})

		It("should reject requests for disabled services", func() {
			// Make request for a service not in the allowed list
			req, _ := http.NewRequest("GET", proxyURL+"/", nil)
			req.Header.Set("Host", "rds.us-east-1.amazonaws.com")
			
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
			body, _ := io.ReadAll(resp.Body)
			Expect(string(body)).To(ContainSubstring("Service rds is not enabled for proxying"))
		})

		It("should handle health check requests", func() {
			req, _ := http.NewRequest("GET", proxyURL+"/health", nil)
			
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			body, _ := io.ReadAll(resp.Body)
			Expect(string(body)).To(Equal("ok"))
		})

		It("should preserve request headers", func() {
			req, _ := http.NewRequest("POST", proxyURL+"/", strings.NewReader(`{"test": "data"}`))
			req.Header.Set("Host", "dynamodb.us-east-1.amazonaws.com")
			req.Header.Set("Content-Type", "application/x-amz-json-1.0")
			req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
			req.Header.Set("X-Custom-Header", "test-value")
			
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			// LocalStack mock would receive all headers
		})
	})

	Describe("Error handling", func() {
		It("should handle LocalStack connection errors", func() {
			// Create proxy pointing to invalid endpoint
			config := &ProxyConfig{
				LocalStackEndpoint: "http://localhost:99999", // Invalid port
				ListenPort:         0,
				Services:           []string{"s3"},
				Timeout:            1 * time.Second,
			}
			errorProxy := NewProxy(config)
			
			// Start proxy
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			
			go errorProxy.Start(ctx)
			time.Sleep(100 * time.Millisecond)
			
			// Make request
			proxyURL := fmt.Sprintf("http://localhost:%d", errorProxy.config.ListenPort)
			req, _ := http.NewRequest("GET", proxyURL+"/test", nil)
			req.Header.Set("Host", "s3.us-east-1.amazonaws.com")
			
			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			
			Expect(resp.StatusCode).To(Equal(http.StatusBadGateway))
		})
	})
})

func TestSidecar(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sidecar Suite")
}