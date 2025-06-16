package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
	"k8s.io/klog/v2"
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

	// Extract service name from the request
	service := h.extractServiceFromRequest(r)
	if service == "" {
		http.Error(w, "Unable to determine AWS service", http.StatusBadRequest)
		return
	}

	// Check if the service is enabled in LocalStack
	enabledServices, err := h.localStackManager.GetEnabledServices()
	if err != nil {
		http.Error(w, "Failed to get enabled services", http.StatusInternalServerError)
		return
	}

	serviceEnabled := false
	for _, s := range enabledServices {
		if s == service {
			serviceEnabled = true
			break
		}
	}

	if !serviceEnabled {
		http.Error(w, fmt.Sprintf("Service '%s' is not enabled in LocalStack", service), http.StatusNotFound)
		return
	}

	// Proxy the request to LocalStack
	h.reverseProxy.ServeHTTP(w, r)
}

// extractServiceFromRequest determines which AWS service is being called
func (h *AWSProxyHandler) extractServiceFromRequest(r *http.Request) string {
	// Check URL path patterns
	path := r.URL.Path
	
	// Common AWS service path patterns
	if strings.HasPrefix(path, "/api/v1/s3/") {
		return "s3"
	}
	if strings.HasPrefix(path, "/api/v1/iam/") {
		return "iam"
	}
	if strings.HasPrefix(path, "/api/v1/logs/") {
		return "logs"
	}
	if strings.HasPrefix(path, "/api/v1/ssm/") {
		return "ssm"
	}
	if strings.HasPrefix(path, "/api/v1/secretsmanager/") {
		return "secretsmanager"
	}
	if strings.HasPrefix(path, "/api/v1/elbv2/") {
		return "elbv2"
	}
	if strings.HasPrefix(path, "/api/v1/rds/") {
		return "rds"
	}
	if strings.HasPrefix(path, "/api/v1/dynamodb/") {
		return "dynamodb"
	}

	// Check for service in query parameters (some AWS APIs use this)
	if service := r.URL.Query().Get("Service"); service != "" {
		return strings.ToLower(service)
	}

	// Check for service in headers
	if target := r.Header.Get("X-Amz-Target"); target != "" {
		// X-Amz-Target format: "ServiceName_YYYYMMDD.OperationName"
		parts := strings.Split(target, "_")
		if len(parts) > 0 {
			return strings.ToLower(parts[0])
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

	return ""
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