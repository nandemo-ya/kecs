package api

import (
	"net/http"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// AWSProxyRouter routes AWS API calls to appropriate handlers
type AWSProxyRouter struct {
	LocalStackManager localstack.Manager
	AWSProxyHandler   *AWSProxyHandler
}

// NewAWSProxyRouter creates a new AWS proxy router
func NewAWSProxyRouter(localStackManager localstack.Manager) (*AWSProxyRouter, error) {
	awsProxyHandler, err := NewAWSProxyHandler(localStackManager)
	if err != nil {
		return nil, err
	}

	return &AWSProxyRouter{
		LocalStackManager: localStackManager,
		AWSProxyHandler:   awsProxyHandler,
	}, nil
}

// RegisterRoutes registers AWS proxy routes on the provided mux
func (apr *AWSProxyRouter) RegisterRoutes(mux *http.ServeMux) {
	// Register a catch-all handler for AWS API calls
	// This will handle all non-ECS AWS service calls
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Check if this should be proxied to LocalStack
		if ShouldProxyToLocalStack(r, apr.LocalStackManager) {
			apr.AWSProxyHandler.ServeHTTP(w, r)
		} else {
			// Not an AWS API call or LocalStack is not available
			http.NotFound(w, r)
		}
	})
}

// ShouldProxyToLocalStack determines if a request should be proxied to LocalStack
func ShouldProxyToLocalStack(r *http.Request, localStackManager localstack.Manager) bool {
	// Check if LocalStack is enabled and healthy
	if localStackManager == nil || !localStackManager.IsHealthy() {
		return false
	}

	// Check if this is an AWS API call (not ECS)
	if !isAWSAPIRequest(r) || isECSRequest(r) {
		return false
	}

	return true
}

// isAWSAPIRequest checks if the request is for an AWS API
func isAWSAPIRequest(r *http.Request) bool {
	// Check for AWS SDK headers first (most reliable)
	if r.Header.Get("X-Amz-Target") != "" ||
		strings.Contains(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256") ||
		r.Header.Get("X-Amz-Date") != "" ||
		r.Header.Get("X-Amz-Security-Token") != "" {
		return true
	}

	// Check for AWS-style endpoints
	host := r.Host
	if host == "" {
		host = r.URL.Host
	}

	// Check for common AWS patterns
	if strings.Contains(host, ".amazonaws.com") ||
		strings.Contains(host, "aws.amazon.com") ||
		host == "169.254.169.254" { // EC2 metadata service
		return true
	}

	// Check URL path for AWS service patterns
	path := r.URL.Path
	// AWS services often use /v1/, /v2/, or service-specific patterns
	if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/v2/") {
		// Could be an AWS API call
		return true
	}

	return false
}

// isECSRequest checks if the request is for the ECS API
func isECSRequest(r *http.Request) bool {
	// ECS API calls go through the main KECS API
	if strings.HasPrefix(r.URL.Path, "/v1/") {
		return true
	}

	// Check X-Amz-Target header for ECS service
	target := r.Header.Get("X-Amz-Target")
	if strings.HasPrefix(target, "AmazonEC2ContainerServiceV") {
		return true
	}

	// Check Authorization header for ECS service
	auth := r.Header.Get("Authorization")
	if strings.Contains(auth, "/ecs/") {
		return true
	}

	return false
}
