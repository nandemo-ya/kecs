package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/logging"
)

// ProxyHandler handles routing requests to appropriate backends
type ProxyHandler struct {
	localStackURL   *url.URL
	localStackProxy *httputil.ReverseProxy
	ecsHandler      http.Handler
	elbv2Handler    http.Handler
	sdHandler       http.Handler // Service Discovery handler
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(localStackURL string, ecsHandler, elbv2Handler, sdHandler http.Handler) (*ProxyHandler, error) {
	parsedURL, err := url.Parse(localStackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LocalStack URL: %w", err)
	}

	// Create reverse proxy for LocalStack
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// Customize the director to preserve headers
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve original host header if needed
		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}
	}

	// Add error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logging.Error("Proxy error", "path", r.URL.Path, "error", err)
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}

	// Configure transport for better performance
	proxy.Transport = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  false,
	}

	return &ProxyHandler{
		localStackURL:   parsedURL,
		localStackProxy: proxy,
		ecsHandler:      ecsHandler,
		elbv2Handler:    elbv2Handler,
		sdHandler:       sdHandler,
	}, nil
}

// ServeHTTP implements http.Handler interface
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log incoming request
	logging.Debug("Incoming request",
		"method", r.Method,
		"path", r.URL.Path,
		"x-amz-target", r.Header.Get("X-Amz-Target"),
		"content-type", r.Header.Get("Content-Type"),
	)

	// Route based on X-Amz-Target header or path
	if h.shouldRouteToECS(r) {
		logging.Debug("Routing to ECS handler", "path", r.URL.Path)
		h.ecsHandler.ServeHTTP(w, r)
		return
	}

	if h.shouldRouteToELBv2(r) {
		logging.Debug("Routing to ELBv2 handler", "path", r.URL.Path)
		h.elbv2Handler.ServeHTTP(w, r)
		return
	}

	if h.shouldRouteToServiceDiscovery(r) {
		logging.Debug("Routing to Service Discovery handler", "path", r.URL.Path)
		h.sdHandler.ServeHTTP(w, r)
		return
	}

	// Default: proxy to LocalStack
	logging.Debug("Proxying to LocalStack", "path", r.URL.Path)
	h.localStackProxy.ServeHTTP(w, r)
}

// shouldRouteToECS determines if the request should be handled by ECS API
func (h *ProxyHandler) shouldRouteToECS(r *http.Request) bool {
	// Check X-Amz-Target header for ECS operations
	target := r.Header.Get("X-Amz-Target")
	if strings.HasPrefix(target, "AmazonEC2ContainerServiceV20141113.") {
		return true
	}

	// Check path-based routing for ECS
	path := r.URL.Path
	if strings.HasPrefix(path, "/v1/") {
		return true
	}

	// Check for ECS-specific paths
	if path == "/" && r.Method == "POST" {
		// Check if it's an ECS API call by examining the body
		// This requires reading and restoring the body
		if h.isECSRequest(r) {
			return true
		}
	}

	return false
}

// shouldRouteToELBv2 determines if the request should be handled by ELBv2 API
func (h *ProxyHandler) shouldRouteToELBv2(r *http.Request) bool {
	// Check X-Amz-Target header for ELBv2 operations
	target := r.Header.Get("X-Amz-Target")
	if strings.HasPrefix(target, "AWSie_backend_200507.") ||
		strings.Contains(target, "ElasticLoadBalancing") {
		return true
	}

	// Check for ELBv2-specific paths
	path := r.URL.Path
	if strings.Contains(path, "elasticloadbalancing") {
		return true
	}

	// Check for ELBv2 Action in form data (used by AWS CLI)
	if r.Method == "POST" && r.URL.Path == "/" {
		if h.isELBv2Request(r) {
			return true
		}
	}

	return false
}

// shouldRouteToServiceDiscovery determines if the request should be handled by Service Discovery API
func (h *ProxyHandler) shouldRouteToServiceDiscovery(r *http.Request) bool {
	// Check X-Amz-Target header for Service Discovery operations
	target := r.Header.Get("X-Amz-Target")
	if strings.HasPrefix(target, "Route53AutoNaming_v20170314.") ||
		strings.Contains(target, "ServiceDiscovery") {
		return true
	}

	// Check for Service Discovery-specific paths
	path := r.URL.Path
	if strings.Contains(path, "servicediscovery") {
		return true
	}

	return false
}

// isECSRequest examines the request body to determine if it's an ECS request
func (h *ProxyHandler) isECSRequest(r *http.Request) bool {
	// Read body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logging.Error("Failed to read request body", "error", err)
		return false
	}

	// Restore body for subsequent handlers
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Check for ECS-specific content
	bodyStr := string(bodyBytes)

	// Check for common ECS action parameters
	ecsActions := []string{
		"CreateCluster",
		"DeleteCluster",
		"ListClusters",
		"DescribeClusters",
		"RegisterTaskDefinition",
		"DeregisterTaskDefinition",
		"ListTaskDefinitions",
		"DescribeTaskDefinition",
		"CreateService",
		"UpdateService",
		"DeleteService",
		"ListServices",
		"DescribeServices",
		"RunTask",
		"StopTask",
		"ListTasks",
		"DescribeTasks",
	}

	for _, action := range ecsActions {
		if strings.Contains(bodyStr, action) {
			return true
		}
	}

	return false
}

// isELBv2Request examines the request body to determine if it's an ELBv2 request
func (h *ProxyHandler) isELBv2Request(r *http.Request) bool {
	// Read body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logging.Error("Failed to read request body", "error", err)
		return false
	}

	// Restore body for subsequent handlers
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Check for ELBv2-specific content
	bodyStr := string(bodyBytes)

	// Check for common ELBv2 action parameters
	elbv2Actions := []string{
		"CreateLoadBalancer",
		"DeleteLoadBalancer",
		"DescribeLoadBalancers",
		"ModifyLoadBalancerAttributes",
		"CreateTargetGroup",
		"DeleteTargetGroup",
		"DescribeTargetGroups",
		"ModifyTargetGroup",
		"RegisterTargets",
		"DeregisterTargets",
		"DescribeTargetHealth",
		"CreateListener",
		"DeleteListener",
		"DescribeListeners",
		"ModifyListener",
		"CreateRule",
		"DeleteRule",
		"DescribeRules",
		"ModifyRule",
		"SetRulePriorities",
		"DescribeLoadBalancerAttributes",
		"DescribeTargetGroupAttributes",
		"ModifyTargetGroupAttributes",
		"DescribeAccountLimits",
		"DescribeListenerCertificates",
		"AddListenerCertificates",
		"RemoveListenerCertificates",
		"DescribeTags",
		"AddTags",
		"RemoveTags",
	}

	// Check if body contains Action=<ELBv2Action>
	for _, action := range elbv2Actions {
		if strings.Contains(bodyStr, "Action="+action) {
			return true
		}
	}

	return false
}

// HealthCheck performs health check on LocalStack connection
func (h *ProxyHandler) HealthCheck(ctx context.Context) error {
	// Create a health check request
	req, err := http.NewRequestWithContext(ctx, "GET", h.localStackURL.String()+"/_localstack/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Send request with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
