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

	// Initialize the reverse proxy with the endpoint from LocalStack manager
	if localStackManager != nil {
		// Get the configuration from LocalStack manager
		config := localStackManager.GetConfig()
		if config != nil && config.UseTraefik && config.ProxyEndpoint != "" {
			// Use the proxy endpoint from configuration
			endpoint := config.ProxyEndpoint
			klog.Infof("Using Traefik endpoint from LocalStack config: %s", endpoint)
			
			if err := handler.updateProxyTarget(endpoint); err != nil {
				klog.Warningf("Failed to initialize proxy target: %v", err)
			}
		} else {
			// Fallback to getting endpoint from manager
			endpoint, err := localStackManager.GetEndpoint()
			if err != nil {
				klog.Warningf("Failed to get LocalStack endpoint: %v", err)
				// Use default as last resort
				endpoint = "http://localhost:4566"
			}
			klog.Infof("Using LocalStack endpoint: %s", endpoint)
			
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
	// Check if LocalStack manager exists
	if h.localStackManager == nil {
		http.Error(w, "LocalStack is not available", http.StatusServiceUnavailable)
		return
	}
	
	// TODO: Fix health check to use Traefik endpoint
	// For now, we'll assume LocalStack is healthy if the manager exists
	// This is a temporary workaround until the health check is fixed
	// Remove the IsHealthy() check temporarily to match aws_proxy_middleware.go

	// Update proxy target if needed
	if h.reverseProxy == nil {
		// Get the configuration from LocalStack manager
		config := h.localStackManager.GetConfig()
		var endpoint string
		
		if config != nil && config.UseTraefik && config.ProxyEndpoint != "" {
			// Use the proxy endpoint from configuration
			endpoint = config.ProxyEndpoint
			klog.Infof("Initializing proxy with Traefik endpoint from config: %s", endpoint)
		} else {
			// Fallback to getting endpoint from manager
			var err error
			endpoint, err = h.localStackManager.GetEndpoint()
			if err != nil {
				klog.Warningf("Failed to get LocalStack endpoint: %v", err)
				// Use default as last resort
				endpoint = "http://localhost:4566"
			}
			klog.Infof("Initializing proxy with LocalStack endpoint: %s", endpoint)
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

