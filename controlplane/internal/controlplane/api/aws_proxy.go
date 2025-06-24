package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"k8s.io/klog/v2"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// AWSProxyHandler handles proxying AWS API calls to LocalStack
type AWSProxyHandler struct {
	localStackManager localstack.Manager
	reverseProxy      *httputil.ReverseProxy
	localStackURL     *url.URL
}

// NewAWSProxyHandler creates a new AWS proxy handler
func NewAWSProxyHandler(localStackManager localstack.Manager) (*AWSProxyHandler, error) {
	handler := &AWSProxyHandler{
		localStackManager: localStackManager,
	}

	// Initialize the reverse proxy when LocalStack is ready
	if localStackManager != nil && localStackManager.IsHealthy() {
		endpoint, err := localStackManager.GetEndpoint()
		if err == nil {
			if err := handler.updateProxyTarget(endpoint); err != nil {
				klog.Warningf("Failed to initialize proxy target: %v", err)
			}
		}
	}

	return handler, nil
}

// updateProxyTarget updates the reverse proxy target
func (h *AWSProxyHandler) updateProxyTarget(endpoint string) error {
	targetURL, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid LocalStack endpoint: %w", err)
	}

	h.localStackURL = targetURL
	h.reverseProxy = httputil.NewSingleHostReverseProxy(targetURL)

	// Customize the reverse proxy director
	originalDirector := h.reverseProxy.Director
	h.reverseProxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Preserve the original host header for AWS SDK compatibility
		req.Host = targetURL.Host

		// Add LocalStack specific headers
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Header.Set("X-LocalStack-Edge", "1")

		// Log the proxied request
		klog.V(4).Infof("Proxying AWS request: %s %s to %s", req.Method, req.URL.Path, targetURL.Host)
	}

	// Custom error handler
	h.reverseProxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		klog.Errorf("Proxy error: %v", err)
		http.Error(w, "Failed to proxy request to LocalStack", http.StatusBadGateway)
	}

	return nil
}

// ServeHTTP handles incoming AWS API requests
func (h *AWSProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if LocalStack is healthy
	if h.localStackManager == nil || !h.localStackManager.IsHealthy() {
		http.Error(w, "LocalStack is not available", http.StatusServiceUnavailable)
		return
	}

	// Update proxy target if needed
	if h.reverseProxy == nil {
		endpoint, err := h.localStackManager.GetEndpoint()
		if err != nil {
			http.Error(w, "Failed to get LocalStack endpoint", http.StatusInternalServerError)
			return
		}
		if err := h.updateProxyTarget(endpoint); err != nil {
			http.Error(w, "Failed to initialize proxy", http.StatusInternalServerError)
			return
		}
	}

	// Extract service name from the request (for logging/debugging)
	service := h.extractServiceFromRequest(r)
	klog.V(3).Infof("Proxying request for AWS service: %s", service)

	// Note: We don't check if the service is enabled here anymore.
	// LocalStack will handle unknown or disabled services appropriately.
	// This allows for more flexibility and reduces maintenance.

	// Proxy the request to LocalStack
	h.reverseProxy.ServeHTTP(w, r)
}

// extractServiceFromRequest determines which AWS service is being called
func (h *AWSProxyHandler) extractServiceFromRequest(r *http.Request) string {
	// Check for service in headers (most reliable)
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		// X-Amz-Target format: "ServiceName_YYYYMMDD.OperationName" or "ServiceName.OperationName"
		parts := strings.SplitN(target, ".", 2)
		if len(parts) > 0 {
			servicePart := parts[0]
			// Remove date suffix if present
			if idx := strings.Index(servicePart, "_"); idx > 0 {
				servicePart = servicePart[:idx]
			}
			// Common service name mappings
			switch strings.ToLower(servicePart) {
			case "amazondynamodbv20120810", "dynamodb":
				return "dynamodb"
			case "amazons3":
				return "s3"
			case "awsie":
				return "iam"
			case "logs":
				return "logs"
			case "awsssm":
				return "ssm"
			case "secretsmanager":
				return "secretsmanager"
			default:
				return strings.ToLower(servicePart)
			}
		}
	}

	// Check Authorization header for service hint
	if auth := r.Header.Get("Authorization"); auth != "" {
		// AWS4-HMAC-SHA256 Credential=.../YYYYMMDD/region/service/aws4_request
		if strings.Contains(auth, "aws4_request") {
			parts := strings.Split(auth, "/")
			if len(parts) >= 5 {
				return strings.ToLower(parts[len(parts)-2])
			}
		}
	}

	// Check host for service hint
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Extract service from AWS endpoint pattern: service.region.amazonaws.com
	if strings.Contains(host, ".amazonaws.com") {
		parts := strings.Split(host, ".")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Check for service in query parameters (some AWS APIs use this)
	if service := r.URL.Query().Get("Service"); service != "" {
		return strings.ToLower(service)
	}

	// Default to unknown - let LocalStack handle it
	return "unknown"
}

// HealthCheck returns the health status of the AWS proxy
func (h *AWSProxyHandler) HealthCheck() (bool, error) {
	if h.localStackManager == nil {
		return false, fmt.Errorf("LocalStack manager not initialized")
	}

	return h.localStackManager.IsHealthy(), nil
}

// isAWSAPICall checks if the request is for an AWS API
func isAWSAPICall(r *http.Request) bool {
	path := r.URL.Path

	// Check for AWS API path patterns
	awsAPIPrefixes := []string{
		"/api/v1/s3/",
		"/api/v1/iam/",
		"/api/v1/logs/",
		"/api/v1/ssm/",
		"/api/v1/secretsmanager/",
		"/api/v1/elbv2/",
		"/api/v1/rds/",
		"/api/v1/dynamodb/",
	}

	for _, prefix := range awsAPIPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	// Check for AWS SDK headers
	if r.Header.Get("X-Amz-Target") != "" ||
		strings.Contains(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256") {
		return true
	}

	return false
}

// isECSAPICall checks if the request is for the ECS API
func isECSAPICall(r *http.Request) bool {
	// ECS API calls go through the main KECS API
	return strings.HasPrefix(r.URL.Path, "/v1/") ||
		r.Header.Get("X-Amz-Target") == "AmazonEC2ContainerServiceV20141113"
}
