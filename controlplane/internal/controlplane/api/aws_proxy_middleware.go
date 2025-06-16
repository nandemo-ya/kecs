package api

import (
	"net/http"
	"strings"

	"github.com/nandemo-ya/kecs/controlplane/internal/localstack"
)

// AWSProxyRouter routes AWS API calls to appropriate handlers
type AWSProxyRouter struct {
	localStackManager localstack.Manager
	awsProxyHandler   *AWSProxyHandler
}

// NewAWSProxyRouter creates a new AWS proxy router
func NewAWSProxyRouter(localStackManager localstack.Manager) (*AWSProxyRouter, error) {
	awsProxyHandler, err := NewAWSProxyHandler(localStackManager)
	if err != nil {
		return nil, err
	}

	return &AWSProxyRouter{
		localStackManager: localStackManager,
		awsProxyHandler:   awsProxyHandler,
	}, nil
}

// RegisterRoutes registers AWS proxy routes on the provided mux
func (apr *AWSProxyRouter) RegisterRoutes(mux *http.ServeMux) {
	// Register AWS service endpoints
	awsServices := []string{
		"s3",
		"iam",
		"logs",
		"ssm",
		"secretsmanager",
		"elbv2",
		"rds",
		"dynamodb",
	}

	for _, service := range awsServices {
		pattern := "/api/v1/" + service + "/"
		mux.Handle(pattern, apr.awsProxyHandler)
	}
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
		// But exclude ECS calls
		if !isECSRequest(r) {
			return true
		}
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